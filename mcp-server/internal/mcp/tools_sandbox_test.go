package mcp

// Spec 03 — runtime sandbox isolation must be reachable AND must not be a
// host-execution path.
//
// The defect these tests exist to prevent is the one the spec's own status
// table recorded: "The library shipped; neither tool exists." A fully tested
// sandbox that nothing can call protects nothing. But wiring it naively would
// be worse than leaving it unwired, because a tool whose purpose is isolation
// and which can be asked to run at L0 is a remote command execution surface
// with a reassuring name.
//
// So these tests drive the REAL dispatched handlers and assert the properties
// that make the tool safe rather than merely present:
//
//   - L0 is refused outright, even when a policy declares it;
//   - a per-request override can narrow but never widen;
//   - an isolation request can raise the policy level, never lower it;
//   - an unknown tool type is refused rather than run at the global default;
//   - when isolation setup fails, the command does NOT run on the host.

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

func newSandboxServer(t *testing.T) *MCPServer {
	t.Helper()
	s := &MCPServer{}
	s.SetGuardrailsEngine(guardrails.NewEngine(nil, nil))
	return s
}

func callSandboxTool(t *testing.T, s *MCPServer, name string, args map[string]any) (*mcp.CallToolResult, map[string]any) {
	t.Helper()
	var res *mcp.CallToolResult
	var err error
	switch name {
	case "guardrail_sandbox_execute":
		res, err = s.handleSandboxExecute(context.Background(), args)
	case "guardrail_sandbox_config":
		res, err = s.handleSandboxConfig(context.Background(), args)
	default:
		t.Fatalf("unknown sandbox tool %q", name)
	}
	if err != nil {
		t.Fatalf("%s: transport error: %v", name, err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatalf("%s: empty result", name)
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("%s: unexpected content type %T", name, res.Content[0])
	}
	return res, decodeJSON(t, tc.Text)
}

func TestSandboxToolsAreRegisteredDispatchedAndClassified(t *testing.T) {
	s := newSandboxServer(t)

	advertised := map[string]bool{}
	for _, tool := range s.toolList() {
		advertised[tool.Name] = true
	}
	for _, name := range []string{"guardrail_sandbox_execute", "guardrail_sandbox_config"} {
		if !advertised[name] {
			t.Errorf("%s is not advertised by toolList()", name)
		}
	}

	// Dispatch reachability: a tool advertised but not routed is a tool that
	// answers "unknown tool" to every caller.
	for _, name := range []string{"guardrail_sandbox_execute", "guardrail_sandbox_config"} {
		res, err := s.dispatchToolCall(context.Background(), name, map[string]any{})
		if err != nil {
			t.Fatalf("%s: dispatch error: %v", name, err)
		}
		if res == nil {
			t.Fatalf("%s: dispatch returned no result", name)
		}
	}

	// Classification: an unclassified tool is denied before its handler runs,
	// so a perfect handler would still be uncallable.
	cases := map[string]auth.ActionKind{
		"guardrail_sandbox_execute": auth.ActionMutate,
		"guardrail_sandbox_config":  auth.ActionAdmin,
	}
	for name, want := range cases {
		kind, ok := auth.MCPToolAction(name)
		if !ok {
			t.Errorf("%s is unclassified — it would be denied for every caller", name)
			continue
		}
		if kind != want {
			t.Errorf("%s classified %q, want %q", name, kind, want)
		}
	}
	// Both must take the audit-before-effect path.
	for _, name := range []string{"guardrail_sandbox_execute", "guardrail_sandbox_config"} {
		if !auth.IsSensitiveMCPTool(name) {
			t.Errorf("%s is not treated as security-sensitive", name)
		}
	}
}

func TestSandboxExecuteRefusesL0EvenWhenPolicyDeclaresIt(t *testing.T) {
	s := newSandboxServer(t)

	// file_edit ships with an L0 policy, and the spec's own fallback story
	// says L0 is a valid level. A tool whose purpose is isolation must still
	// refuse it: L0 here would be arbitrary host execution behind a name that
	// implies the opposite.
	if err := s.guardrailsEngine.SetSandboxPolicy("host_path", guardrails.SandboxPolicy{
		DefaultLevel: guardrails.LevelL0,
	}); err != nil {
		t.Fatalf("setting an L0 policy should be valid config: %v", err)
	}

	marker := filepath.Join(t.TempDir(), "should-not-exist")
	res, payload := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":   "touch " + marker,
		"tool_type": "host_path",
	})
	if !res.IsError {
		t.Fatal("L0 execution was not refused")
	}
	if _, ok := payload["error"]; !ok {
		t.Fatalf("refusal carried no error: %v", payload)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("the command RAN: L0 refusal is not enforced")
	}
}

func TestSandboxExecuteRefusesUnknownToolType(t *testing.T) {
	s := newSandboxServer(t)
	res, payload := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":   "true",
		"tool_type": "does_not_exist",
	})
	if !res.IsError {
		t.Fatal("unknown tool type was accepted — it would run at the global default")
	}
	known, ok := payload["known_types"].([]any)
	if !ok || len(known) == 0 {
		t.Fatalf("refusal did not name the known types: %v", payload)
	}
}

func TestSandboxExecuteRefusesUnknownLevel(t *testing.T) {
	s := newSandboxServer(t)
	res, _ := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":       "true",
		"tool_type":     "bash",
		"sandbox_level": "L9",
	})
	if !res.IsError {
		t.Fatal("an unknown level was accepted")
	}
}

func TestSandboxExecuteRefusesWideningOverrides(t *testing.T) {
	s := newSandboxServer(t)
	cfg := s.guardrailsEngine.SandboxConfig()
	base := cfg.ResolveToolPolicyLimits("bash")

	// Each case must be refused BY THE OVERRIDE CHECK, before any execution.
	//
	// The first version of this test asserted only `res.IsError`, and a
	// mutation that deleted every widening refusal SURVIVED it: execution
	// fails in an environment without user namespaces, so IsError was true
	// for an unrelated reason and the assertion passed on a sandbox that had
	// accepted the widening. The test now requires the refuse-before-execute
	// shape — an error naming the field and NO execution evidence — which is
	// what "refused" actually means here.
	cases := []struct {
		name      string
		args      map[string]any
		wantInErr string
	}{
		{
			name:      "memory above policy",
			args:      map[string]any{"resource_limits": map[string]any{"max_mb": base.Memory.MaxMB + 1}},
			wantInErr: "max_mb",
		},
		{
			name:      "time above policy",
			args:      map[string]any{"resource_limits": map[string]any{"max_seconds": base.Time.MaxSeconds + 1}},
			wantInErr: "max_seconds",
		},
		{
			name:      "cpu above policy",
			args:      map[string]any{"resource_limits": map[string]any{"max_cpu_percent": base.CPU.MaxPercent + 1}},
			wantInErr: "max_cpu_percent",
		},
		{
			name:      "path outside the policy set",
			args:      map[string]any{"allowed_paths": []any{"/etc"}},
			wantInErr: "allowed path",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.args["command"] = "true"
			tc.args["tool_type"] = "bash"
			res, payload := callSandboxTool(t, s, "guardrail_sandbox_execute", tc.args)
			if !res.IsError {
				t.Fatalf("widening override accepted: %v", payload)
			}
			msg, _ := payload["error"].(string)
			if msg == "" {
				t.Fatalf("refusal carried no error: %v", payload)
			}
			if !strings.Contains(msg, tc.wantInErr) {
				t.Fatalf("refusal does not name %q (got %q) — it may not be an override refusal at all", tc.wantInErr, msg)
			}
			// No execution may have been attempted: a refused override must not
			// have reached the sandbox at all.
			if _, ran := payload["actual_isolation_level"]; ran {
				t.Fatalf("override was rejected but the command still executed: %v", payload)
			}
		})
	}

	// Positive control: a NARROWING override is legal and must reach
	// execution. Without this, the assertions above could pass against a tool
	// that refused every override for the wrong reason.
	_, narrowed := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":         "true",
		"tool_type":       "bash",
		"resource_limits": map[string]any{"max_mb": base.Memory.MaxMB - 1},
	})
	if msg, _ := narrowed["error"].(string); strings.Contains(msg, "max_mb") {
		t.Fatalf("a narrowing override was refused: %v", narrowed)
	}
	if _, reached := narrowed["actual_isolation_level"]; !reached {
		t.Fatalf("a narrowing override did not reach execution: %v", narrowed)
	}
}

func TestEffectiveLevelRaisesButNeverLowers(t *testing.T) {
	cases := []struct {
		policy, requested, want guardrails.SandboxLevel
	}{
		{guardrails.LevelL1, "", guardrails.LevelL1},
		{guardrails.LevelL1, guardrails.LevelL0, guardrails.LevelL1}, // never lowered
		{guardrails.LevelL1, guardrails.LevelL2, guardrails.LevelL2}, // raised
		{guardrails.LevelL2, guardrails.LevelL1, guardrails.LevelL2}, // never lowered
		{guardrails.LevelL2, "", guardrails.LevelL2},
	}
	for _, tc := range cases {
		if got := guardrails.EffectiveLevel(tc.policy, tc.requested); got != tc.want {
			t.Errorf("EffectiveLevel(%s, %q) = %s, want %s", tc.policy, tc.requested, got, tc.want)
		}
	}
}

func TestSandboxConfigAppliesAndKeepsPreviousOnRefusal(t *testing.T) {
	s := newSandboxServer(t)

	res, payload := callSandboxTool(t, s, "guardrail_sandbox_config", map[string]any{
		"tool_type":     "custom_tool",
		"sandbox_level": "L2",
		"resource_limits": map[string]any{
			"max_mb":      128,
			"max_seconds": 30,
		},
	})
	if res.IsError {
		t.Fatalf("valid policy refused: %v", payload)
	}
	if configured, _ := payload["configured"].(bool); !configured {
		t.Fatalf("policy not reported configured: %v", payload)
	}

	applied := s.guardrailsEngine.SandboxPolicies()["custom_tool"]
	if applied.DefaultLevel != guardrails.LevelL2 {
		t.Fatalf("policy level = %q, want L2", applied.DefaultLevel)
	}

	// A refused policy must not disturb the accepted one.
	res, _ = callSandboxTool(t, s, "guardrail_sandbox_config", map[string]any{
		"tool_type":     "custom_tool",
		"sandbox_level": "L9",
	})
	if !res.IsError {
		t.Fatal("invalid level accepted")
	}
	if after := s.guardrailsEngine.SandboxPolicies()["custom_tool"]; after.DefaultLevel != guardrails.LevelL2 {
		t.Fatalf("refused config overwrote the applied policy: %s", after.DefaultLevel)
	}
}

func TestSandboxConfigRejectsEmptyToolType(t *testing.T) {
	s := newSandboxServer(t)
	res, _ := callSandboxTool(t, s, "guardrail_sandbox_config", map[string]any{
		"sandbox_level": "L1",
	})
	if !res.IsError {
		t.Fatal("empty tool_type accepted")
	}
}

func TestSandboxToolsFailClosedWithoutEngine(t *testing.T) {
	s := &MCPServer{} // no engine: the command must NOT run

	marker := filepath.Join(t.TempDir(), "should-not-exist")
	res, payload := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":   "touch " + marker,
		"tool_type": "bash",
	})
	if !res.IsError {
		t.Fatal("execute without an engine was not refused")
	}
	if _, ok := payload["error"]; !ok {
		t.Fatalf("refusal carried no error: %v", payload)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the command RAN with no engine configured")
	}

	res, _ = callSandboxTool(t, s, "guardrail_sandbox_config", map[string]any{
		"tool_type":     "bash",
		"sandbox_level": "L1",
	})
	if !res.IsError {
		t.Fatal("config without an engine was not refused")
	}
}

// TestSandboxIsolationFailureDoesNotExecuteOnHost is the load-bearing test.
//
// In an environment where namespace setup is unavailable (no user namespaces,
// no container runtime) the sandbox must fail closed, not fall back to running
// the command on the host. The marker file is the proof: if the isolation
// attempt degraded to host execution, the file exists.
//
// This runs the REAL L1 path through the REAL handler. Where the environment
// supports unshare the command runs isolated and the marker also does not
// exist — so the assertion holds either way, and the second half of the check
// distinguishes the two outcomes rather than assuming one.
func TestSandboxIsolationFailureDoesNotExecuteOnHost(t *testing.T) {
	s := newSandboxServer(t)

	dir := t.TempDir()
	marker := filepath.Join(dir, "executed")

	// The policy's writable set must contain the directory for the mount to be
	// legal at L1; L1 does not enforce mounts, so this is about intent.
	res, payload := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":   "touch " + marker,
		"tool_type": "bash",
	})

	_, statErr := os.Stat(marker)
	isolated := statErr != nil

	if !isolated {
		t.Fatal("the command executed on the host outside the declared isolation")
	}

	level, _ := payload["actual_isolation_level"].(string)
	if !res.IsError && level == string(guardrails.LevelL0) {
		t.Fatal("a successful result reported L0 for a tool whose policy declares L1")
	}
	t.Logf("sandbox outcome: isError=%v actual_level=%q violations=%v",
		res.IsError, level, payload["sandbox_violations"])
}

func TestSortedPolicyNamesIsDeterministic(t *testing.T) {
	s := newSandboxServer(t)
	policies := s.guardrailsEngine.SandboxPolicies()
	names := sortedPolicyNames(policies)
	if !sort.StringsAreSorted(names) {
		t.Fatalf("policy names not sorted: %v", names)
	}
}
