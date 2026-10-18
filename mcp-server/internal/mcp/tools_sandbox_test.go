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

	// Positive control: a NARROWING override must get PAST the override layer.
	// Without this, the assertions above could pass against a tool that refused
	// every override for the wrong reason.
	//
	// It is asserted at the override layer, not at execution: on a host where
	// namespace creation is unavailable the isolation floor now correctly
	// refuses to run the command at all, so "did it execute?" is not a
	// statement about the override. What must hold everywhere is that the
	// narrowing value is accepted rather than refused.
	if _, err := guardrails.NarrowLimits(base, guardrails.LimitOverrides{MaxMB: base.Memory.MaxMB - 1}); err != nil {
		t.Fatalf("a narrowing override was refused by the limit layer: %v", err)
	}
	_, narrowed := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":         "true",
		"tool_type":       "bash",
		"resource_limits": map[string]any{"max_mb": base.Memory.MaxMB - 1},
	})
	if msg, _ := narrowed["error"].(string); strings.Contains(msg, "exceeds") {
		t.Fatalf("a narrowing override was refused by the handler: %v", narrowed)
	}
	// Whether it then ran or was floored is environment-dependent and is
	// asserted by TestSandboxExecuteFloorsAtL1 instead.
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

// TestSandboxNeverReportsLessThanThePolicyLevel is the load-bearing test, and
// it is written as an INVARIANT rather than as a proof about one machine.
//
// The first version of this test touched a marker file and asserted its absence,
// on the theory that host execution would create it. That premise was wrong and
// CI proved it: L1 uses a PID/mount/net/uts namespace but shares the filesystem,
// so the marker appears whether the command ran isolated or not. The test failed
// on a runner that HAS user namespaces — not because isolation broke, but
// because the assertion could not distinguish the two outcomes it claimed to.
//
// The property that actually matters is narrower and checkable everywhere: for a
// tool whose policy declares L1, a successful result must never report a level
// below L1. A result claiming L0 for such a tool means the command ran on the
// host, whatever the filesystem looks like.
func TestSandboxNeverReportsLessThanThePolicyLevel(t *testing.T) {
	s := newSandboxServer(t)

	res, payload := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":   "true",
		"tool_type": "bash",
	})

	level, _ := payload["actual_isolation_level"].(string)
	t.Logf("sandbox outcome: isError=%v actual_level=%q error=%v",
		res.IsError, level, payload["error"])

	if level != "" && guardrails.LevelRank(guardrails.SandboxLevel(level)) < guardrails.LevelRank(guardrails.LevelL1) {
		t.Fatalf("result reports isolation %q for a policy that declares L1 — the command ran on the host", level)
	}
	// If it failed, it must have failed with an error rather than silently
	// returning an empty shell of a result.
	if res.IsError {
		if msg, _ := payload["error"].(string); msg == "" {
			t.Fatalf("failed sandbox execution carried no error: %v", payload)
		}
	}
}

// TestSandboxFallbackFloorIsEnforced is the security property, tested as the
// PURE function that decides it, so it holds identically on a runner with user
// namespaces and one without.
//
// The shipped fallback chain degrades L2 -> L1 -> L0 on setup errors, and L0 is
// direct host execution. For guardrail_sandbox_execute that is a fail-open: an
// environment without user namespaces would run the command on the host and hand
// the caller a result. ExecuteWithFloor(L1) stops the chain instead.
func TestSandboxFallbackFloorIsEnforced(t *testing.T) {
	cases := []struct {
		from, floor guardrails.SandboxLevel
		want        bool
	}{
		// Falling from L1 to L0 crosses an L1 floor: refused.
		{guardrails.LevelL1, guardrails.LevelL1, true},
		// Falling from L2 to L1 does NOT cross an L1 floor: allowed.
		{guardrails.LevelL2, guardrails.LevelL1, false},
		// No floor: the historical behaviour is unchanged.
		{guardrails.LevelL1, guardrails.LevelL0, false},
		{guardrails.LevelL2, guardrails.LevelL0, false},
		// An L2 floor refuses both rungs.
		{guardrails.LevelL2, guardrails.LevelL2, true},
		// L0 has no further fallback.
		{guardrails.LevelL0, guardrails.LevelL1, false},
	}
	for _, tc := range cases {
		if got := guardrails.WouldRunBelowFloor(tc.from, tc.floor); got != tc.want {
			t.Errorf("WouldRunBelowFloor(%s, floor=%s) = %v, want %v", tc.from, tc.floor, got, tc.want)
		}
	}
}

// TestSandboxExecuteFloorsAtL1 proves the tool path actually passes the floor,
// by driving the real handler in this environment. Where namespaces are
// unavailable the call must fail rather than degrade; where they are available it
// must succeed at L1 or better. Both are acceptable; L0 is not.
func TestSandboxExecuteFloorsAtL1(t *testing.T) {
	s := newSandboxServer(t)
	res, payload := callSandboxTool(t, s, "guardrail_sandbox_execute", map[string]any{
		"command":   "true",
		"tool_type": "bash",
	})

	level, _ := payload["actual_isolation_level"].(string)
	if level == string(guardrails.LevelL0) {
		t.Fatalf("command executed at L0 through a tool that must floor at L1: %v", payload)
	}
	if res.IsError {
		// Acceptable only as a refusal: it must say so.
		if msg, _ := payload["error"].(string); msg == "" {
			t.Fatalf("failed execution had no error: %v", payload)
		}
		if level != "" {
			t.Fatalf("refusal reported an isolation level, implying it ran: %v", payload)
		}
	}
}

func TestSortedPolicyNamesIsDeterministic(t *testing.T) {
	s := newSandboxServer(t)
	policies := s.guardrailsEngine.SandboxPolicies()
	names := sortedPolicyNames(policies)
	if !sort.StringsAreSorted(names) {
		t.Fatalf("policy names not sorted: %v", names)
	}
}
