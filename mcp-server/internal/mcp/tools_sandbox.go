package mcp

// Spec 03 (guardrails-gaps-2026/03-runtime-sandbox-isolation) — the two tools
// the spec defines, wired to the sandbox library.
//
// Before this file the sandbox was a fully built, fully tested library with no
// entry point: no MCP tool, no production caller, no config loader. It could
// not protect anything, because nothing could reach it. The spec's own status
// table said exactly that ("The library shipped; neither tool exists").
//
// Two properties matter more than the plumbing:
//
//  1. The tool cannot be used as a host-execution path. guardrail_sandbox_execute
//     refuses an effective L0 outright, even when a policy declares it. A tool
//     whose purpose is isolation must not become the way to bypass it.
//  2. A request may tighten the policy but never loosen it. The per-request
//     overrides are all narrowing-only: a smaller memory cap, a shorter
//     timeout, a subset of the policy's writable paths. Anything that would
//     widen the sandbox is refused rather than clamped silently, because a
//     clamped value reads as "your override applied".

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// sandboxResultEnvelope is what a caller receives. It carries the actual
// isolation level that ran, never the requested one.
type sandboxResultEnvelope struct {
	ExitCode           int      `json:"exit_code"`
	Stdout             string   `json:"stdout"`
	Stderr             string   `json:"stderr"`
	SandboxViolations  []string `json:"sandbox_violations"`
	ActualIsolationLvl string   `json:"actual_isolation_level"`
	RequestedLevel     string   `json:"requested_level"`
	ToolType           string   `json:"tool_type"`
	ExecutionMs        int64    `json:"execution_ms"`
}

func (s *MCPServer) sandboxTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name: "guardrail_sandbox_execute",
			Description: "Run a command under the isolation level and resource limits declared for a tool type. " +
				"Isolation may be raised by the request but never lowered, and L0 is refused: this tool is not a host-execution path",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The command to execute inside the sandbox",
					},
					"tool_type": map[string]any{
						"type":        "string",
						"description": "Which declared sandbox policy applies (for example bash, git, file_edit). An unknown tool type is refused",
					},
					"sandbox_level": map[string]any{
						"type":        "string",
						"description": "Optional isolation level to raise to (L1 or L2). A lower level than the policy is ignored and the policy level is used",
						"enum":        []string{"L1", "L2"},
					},
					"resource_limits": map[string]any{
						"type":        "object",
						"description": "Optional NARROWING overrides. A value larger than the policy's is refused, not clamped",
						"properties": map[string]any{
							"max_mb": map[string]any{
								"type":        "integer",
								"description": "Lower memory cap in MB than the policy allows",
							},
							"max_seconds": map[string]any{
								"type":        "integer",
								"description": "Shorter execution timeout in seconds than the policy allows",
							},
							"max_cpu_percent": map[string]any{
								"type":        "number",
								"description": "Lower CPU percentage than the policy allows",
							},
						},
					},
					"allowed_paths": map[string]any{
						"type":        "array",
						"description": "Optional subset of the policy's writable paths. A path outside the policy's set is refused",
						"items":       map[string]any{"type": "string"},
					},
				},
				Required: []string{"command", "tool_type"},
			},
		},
		{
			Name: "guardrail_sandbox_config",
			Description: "Configure the sandbox policy for a tool type. The whole configuration is validated before it is applied, " +
				"so a refused policy leaves the previous one in force",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"tool_type": map[string]any{
						"type":        "string",
						"description": "Tool type whose policy is being set (for example bash, git, file_edit)",
					},
					"sandbox_level": map[string]any{
						"type":        "string",
						"description": "Default isolation level for this tool type",
						"enum":        []string{"L0", "L1", "L2"},
					},
					"resource_limits": map[string]any{
						"type":        "object",
						"description": "Resource limits for this tool type; omitted fields inherit the global defaults",
						"properties": map[string]any{
							"max_mb": map[string]any{
								"type":        "integer",
								"description": "Memory cap in MB",
							},
							"max_seconds": map[string]any{
								"type":        "integer",
								"description": "Execution timeout in seconds",
							},
							"max_cpu_percent": map[string]any{
								"type":        "number",
								"description": "CPU percentage (0-100)",
							},
						},
					},
				},
				Required: []string{"tool_type", "sandbox_level"},
			},
		},
	}
}

// sandboxUnavailable is the fail-closed answer when no engine is configured.
// Running the command anyway would be the one outcome this tool must not have.
func sandboxUnavailable(tool string) *mcp.CallToolResult {
	payload, _ := json.Marshal(map[string]any{
		"error": "sandbox is not configured",
		"tool":  tool,
	})
	return errorResult(string(payload))
}

// handleSandboxExecute implements guardrail_sandbox_execute.
func (s *MCPServer) handleSandboxExecute(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	command, _ := args["command"].(string)
	toolType, _ := args["tool_type"].(string)
	requested, _ := args["sandbox_level"].(string)

	switch {
	case strings.TrimSpace(command) == "":
		return errorResult(`{"error":"command is required"}`), nil
	case strings.TrimSpace(toolType) == "":
		return errorResult(`{"error":"tool_type is required"}`), nil
	case s.guardrailsEngine == nil:
		return sandboxUnavailable("guardrail_sandbox_execute"), nil
	}

	requestedLevel := guardrails.SandboxLevel("")
	if requested != "" {
		lvl, err := guardrails.ParseSandboxLevel(requested)
		if err != nil {
			payload, _ := json.Marshal(map[string]any{"error": err.Error()})
			return errorResult(string(payload)), nil
		}
		requestedLevel = lvl
	}

	policies := s.guardrailsEngine.SandboxPolicies()
	policy, ok := policies[toolType]
	if !ok {
		payload, _ := json.Marshal(map[string]any{
			"error":         fmt.Sprintf("no sandbox policy for tool type %q", toolType),
			"known_types":   sortedPolicyNames(policies),
			"tool_type":     toolType,
			"sandbox_level": "",
		})
		return errorResult(string(payload)), nil
	}

	// Overrides are narrowing-only. Resolve the policy's effective limits the
	// same way the executor will, then compare against them.
	effectiveLimits := s.guardrailsEngine.SandboxConfig()
	baseLimits := effectiveLimits.ResolveToolPolicyLimits(toolType)
	overrides, refusal := limitOverridesFromArgs(args)
	if refusal != "" {
		payload, _ := json.Marshal(map[string]any{"error": refusal})
		return errorResult(string(payload)), nil
	}
	// Refuse a widening override BEFORE executing, so the caller learns its
	// override was rejected rather than being silently clamped.
	if _, err := guardrails.NarrowLimits(baseLimits, overrides); err != nil {
		payload, _ := json.Marshal(map[string]any{"error": err.Error()})
		return errorResult(string(payload)), nil
	}

	// Effective level: the policy level, raised (never lowered) by the request.
	policyLevel := policy.DefaultLevel
	if policyLevel == "" {
		payload, _ := json.Marshal(map[string]any{"error": fmt.Sprintf("sandbox policy %q declares no level", toolType)})
		return errorResult(string(payload)), nil
	}
	effective := guardrails.EffectiveLevel(policyLevel, requestedLevel)

	// A tool that exists to isolate must never become the way to reach the
	// host. L0 is refused even when a policy (or the global default) says so.
	if effective == guardrails.LevelL0 {
		payload, _ := json.Marshal(map[string]any{
			"error": "refusing to execute: effective isolation is L0 (no isolation). " +
				"guardrail_sandbox_execute is not a host-execution path; declare a policy of at least L1 for this tool type",
			"tool_type":    toolType,
			"policy_level": policyLevel,
			"requested":    string(requestedLevel),
			"actual_level": string(effective),
		})
		return errorResult(string(payload)), nil
	}

	ctx = guardrails.WithAuditAttribution(ctx, "guardrail_sandbox_execute", toolType)

	result, err := s.guardrailsEngine.ExecuteSandboxForLimits(ctx, toolType, command, requestedLevel, overrides)
	if result == nil {
		payload, _ := json.Marshal(map[string]any{
			"error":     sandboxErrText(err, "sandbox execution failed"),
			"tool_type": toolType,
		})
		return errorResult(string(payload)), nil
	}

	envelope := sandboxResultEnvelope{
		ExitCode:           result.ExitCode,
		Stdout:             result.Stdout,
		Stderr:             result.Stderr,
		SandboxViolations:  result.SandboxViolations,
		ActualIsolationLvl: string(result.ActualIsolationLvl),
		RequestedLevel:     string(effective),
		ToolType:           toolType,
		ExecutionMs:        result.ExecutionTime.Milliseconds(),
	}
	// A violation is a security verdict, not a transport error, but it must
	// not be readable as a pass by a caller that only inspects IsError.
	return jsonToolResult(envelope, len(result.SandboxViolations) > 0 || err != nil)
}

// handleSandboxConfig implements guardrail_sandbox_config.
func (s *MCPServer) handleSandboxConfig(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	toolType, _ := args["tool_type"].(string)
	levelStr, _ := args["sandbox_level"].(string)

	switch {
	case strings.TrimSpace(toolType) == "":
		return errorResult(`{"configured":false,"error":"tool_type is required"}`), nil
	case strings.TrimSpace(levelStr) == "":
		return errorResult(`{"configured":false,"error":"sandbox_level is required"}`), nil
	case s.guardrailsEngine == nil:
		return sandboxUnavailable("guardrail_sandbox_config"), nil
	}

	level, err := guardrails.ParseSandboxLevel(levelStr)
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"configured": false, "error": err.Error()})
		return errorResult(string(payload)), nil
	}

	limits, err := resourceLimitsFromArgs(args)
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"configured": false, "error": err.Error()})
		return errorResult(string(payload)), nil
	}
	ctx = guardrails.WithAuditAttribution(ctx, "guardrail_sandbox_config", toolType)

	if err := s.guardrailsEngine.SetSandboxPolicy(toolType, guardrails.SandboxPolicy{
		DefaultLevel:   level,
		ResourceLimits: limits,
	}); err != nil {
		payload, _ := json.Marshal(map[string]any{"configured": false, "error": err.Error()})
		return errorResult(string(payload)), nil
	}

	return jsonToolResult(map[string]any{
		"configured":    true,
		"tool_type":     toolType,
		"sandbox_level": string(level),
		"tool_policies": len(s.guardrailsEngine.SandboxPolicies()),
	}, false)
}

// sandboxErrText renders an execution error, or a fallback when there is none.
func sandboxErrText(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}

func sortedPolicyNames(policies map[string]guardrails.SandboxPolicy) []string {
	names := make([]string, 0, len(policies))
	for name := range policies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// limitOverridesFromArgs reads the optional narrowing overrides. Numeric
// fields are accepted as either JSON numbers or the float64s encoding/json
// produces for an untyped map.
func limitOverridesFromArgs(args map[string]any) (guardrails.LimitOverrides, string) {
	var o guardrails.LimitOverrides

	// allowed_paths lives at the top level of the tool input, NOT inside
	// resource_limits, so it must be read independently. The first version of
	// this function returned early when resource_limits was absent, which meant
	// a caller that supplied only allowed_paths had the field silently
	// ignored — a declared override with no effect, which is the same
	// fail-open class this whole pass exists to remove.
	if v, present := args["allowed_paths"]; present {
		list, ok := v.([]any)
		if !ok {
			return o, "allowed_paths must be an array of strings"
		}
		for _, item := range list {
			s, ok := item.(string)
			if !ok {
				return o, "allowed_paths must be an array of strings"
			}
			o.AllowedPaths = append(o.AllowedPaths, s)
		}
	}

	raw, ok := args["resource_limits"].(map[string]any)
	if !ok {
		if args["resource_limits"] == nil {
			return o, ""
		}
		return o, "resource_limits must be an object"
	}
	if v, present := raw["max_mb"]; present {
		n, ok := intArg(v)
		if !ok {
			return o, "resource_limits.max_mb must be an integer"
		}
		o.MaxMB = n
	}
	if v, present := raw["max_seconds"]; present {
		n, ok := intArg(v)
		if !ok {
			return o, "resource_limits.max_seconds must be an integer"
		}
		o.MaxSeconds = n
	}
	if v, present := raw["max_cpu_percent"]; present {
		f, ok := floatArg(v)
		if !ok {
			return o, "resource_limits.max_cpu_percent must be a number"
		}
		o.MaxCPUPercent = f
	}
	return o, ""
}

func intArg(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		if n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	}
	return 0, false
}

func floatArg(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

// resourceLimitsFromArgs reads the resource_limits object for
// guardrail_sandbox_config. Omitted fields stay zero, which means "inherit the
// global defaults" at resolution time.
func resourceLimitsFromArgs(args map[string]any) (guardrails.ResourceLimits, error) {
	var limits guardrails.ResourceLimits
	raw, ok := args["resource_limits"].(map[string]any)
	if !ok || raw == nil {
		return limits, nil
	}
	if v, present := raw["max_mb"]; present {
		n, ok := intArg(v)
		if !ok {
			return limits, fmt.Errorf("resource_limits.max_mb must be an integer")
		}
		limits.Memory.MaxMB = n
	}
	if v, present := raw["max_seconds"]; present {
		n, ok := intArg(v)
		if !ok {
			return limits, fmt.Errorf("resource_limits.max_seconds must be an integer")
		}
		limits.Time.MaxSeconds = n
		// A wall-clock bound shorter than the process bound would kill the
		// command early for a reason the caller did not ask for; keep them
		// equal unless a separate value is provided.
		limits.Time.MaxWallSeconds = n
	}
	if v, present := raw["max_cpu_percent"]; present {
		f, ok := floatArg(v)
		if !ok {
			return limits, fmt.Errorf("resource_limits.max_cpu_percent must be a number")
		}
		limits.CPU.MaxPercent = f
	}
	return limits, nil
}
