package guardrails

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// readSandboxSource returns the source of sandbox.go, for the wiring assertion
// below. execL2 cannot be executed without a container runtime, so the call
// site is checked structurally rather than by running a container.
func readSandboxSource(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the test's own path")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "sandbox.go"))
	if err != nil {
		t.Fatalf("read sandbox.go: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("sandbox.go read as empty; a source assertion over an empty file passes vacuously")
	}
	return string(data)
}

// Spec 03 (guardrails-gaps-2026/03-runtime-sandbox-isolation) — two declared
// controls that were carried through resolution and read by nothing.
//
// Both belong to the class this program keeps closing: a field that appears in
// the config, in the policy table, in the docs and in the resolved limits, with
// no consumer on the path that actually executes a command. The review trail
// says the control exists; the process applies something else.

// TestSandboxNetworkDisabledBindsEvenWithAnAllowlist proves `network.enabled`
// is the switch and `allowed_hosts` is only the allowlist.
//
// Sec. 3.2 documents `enabled: false   # no network by default` immediately
// above `allowed_hosts: []  # whitelist if network needed`. The natural reading
// of "disabled, with a whitelist prepared for when it is switched on" produced
// the opposite behaviour: the egress decision keyed off
// len(AllowedHosts) > 0 alone, so declaring a whitelist while network was
// disabled opened a provisioned bridge and a CONNECT proxy to exactly those
// hosts. The field that says "no" was ignored, and the field that says "these
// hosts, if permitted" was treated as permission.
func TestSandboxNetworkDisabledBindsEvenWithAnAllowlist(t *testing.T) {
	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Network.Enabled = false
	limits.Network.AllowedHosts = []string{"github.com", "gitlab.com"}

	if egressPermitted(limits) {
		t.Error("network.enabled=false with an allowlist still permitted egress; the declared control was ignored")
	}

	// And the container argument that carries that decision must be `none`.
	// Evaluated with a bridge configured as well, because that is the shape
	// that actually reached production: a process-wide NetworkMode plus a
	// per-tool policy that said network disabled.
	if got := containerNetworkPlan(limits, "agentnet"); got != "--network=none" {
		t.Errorf("network.enabled=false produced %q; want --network=none", got)
	}
}

// TestSandboxNetworkEnabledWithAllowlistOpensEgress is the negative control:
// the switch must actually switch something on, or the assertions above would
// pass for a function that is simply constant false.
func TestSandboxNetworkEnabledWithAllowlistOpensEgress(t *testing.T) {
	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Network.Enabled = true
	limits.Network.AllowedHosts = []string{"github.com"}

	if !egressPermitted(limits) {
		t.Error("network.enabled=true with an allowlist did not permit egress")
	}

	// No allowlist and network on is still no egress: there is no host the
	// execution is permitted to reach, so opening a bridge would be egress to
	// everything.
	noHosts := limits
	noHosts.Network.AllowedHosts = nil
	if egressPermitted(noHosts) {
		t.Error("network.enabled=true with no allowlist permitted egress to any host")
	}

	// An unset limits block must never permit egress (the zero value is the
	// least privileged reading).
	if egressPermitted(ResourceLimits{}) {
		t.Error("an unset limits block permitted egress")
	}
}

// TestSandboxNetworkModeIgnoredWhenNetworkDisabled proves the process-wide
// NetworkMode cannot hand a bridge to a policy that declared no network.
//
// NetworkMode is a single global field; a tool policy that says
// `network: {enabled: false}` used to receive --network=<bridge> anyway,
// because the mode was read straight from the config snapshot with no gate on
// the per-execution limits. One tool's declaration was overridden by another
// layer's field.
func TestSandboxNetworkModeIgnoredWhenNetworkDisabled(t *testing.T) {
	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Network.Enabled = false

	if got := containerNetworkPlan(limits, "agentnet"); got != "--network=none" {
		t.Errorf("a disabled policy still received the global NetworkMode: got %q, want --network=none", got)
	}

	enabled := limits
	enabled.Network.Enabled = true
	if got := containerNetworkPlan(enabled, "agentnet"); got != "--network=agentnet" {
		t.Errorf("an enabled policy did not receive the configured NetworkMode: got %q", got)
	}

	// With no bridge configured there is nothing to attach to, even when the
	// execution asks for network.
	if got := containerNetworkPlan(enabled, ""); got != "--network=none" {
		t.Errorf("no configured bridge produced %q, want --network=none", got)
	}
}

// TestSandboxNetworkEnabledReachesTheExecutionPath proves the decision is the
// one execL2 uses, not a helper nothing calls.
//
// A pure function that the production path does not consult is the exact
// vacuity this file exists to prevent, so this asserts the wiring by reading
// the source of truth the executor builds its argv from. It is deliberately a
// structural assertion: execL2 cannot run without a container runtime, and a
// test that shells out to podman proves a different thing on every machine.
func TestSandboxNetworkEnabledReachesTheExecutionPath(t *testing.T) {
	if !strings.Contains(readSandboxSource(t), "if egressPermitted(limits) {") {
		t.Error("execL2 does not consult egressPermitted; the network switch has no consumer on the execution path")
	}
	if !strings.Contains(readSandboxSource(t), "containerNetworkPlan(limits, netMode)") {
		t.Error("execL2 does not build its --network argument from containerNetworkPlan")
	}
}

// TestDisabledSandboxRefusesToExecute proves the manager is the single source
// of truth for its own posture.
//
// `sandbox.enabled` was read by exactly one place in the tree: the readiness
// probe. Readiness therefore reported "sandbox manager is not enabled" — the
// control is absent — while every execution still went through Execute. A
// deployment whose operator turned the sandbox off got a report saying it was
// off and commands that ran anyway. Refusing is the only reading that keeps
// the two honest: an unsandboxed execution is the thing `enabled: false`
// claims is not happening.
func TestDisabledSandboxRefusesToExecute(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.Enabled = false
	mgr := NewSandboxManager(cfg, slog.Default())

	_, err := mgr.Execute(context.Background(), "echo should-not-run", LevelL1, mgr.DefaultLimits())
	if err == nil {
		t.Fatal("a disabled sandbox executed the command anyway; the readiness probe and the execution path disagree")
	}
	if !errors.Is(err, ErrSandboxViolation) {
		t.Fatalf("a disabled sandbox must report a security refusal, got %v", err)
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("refusal does not name the cause: %v", err)
	}

	// The policy-driven entry point must refuse for the same reason.
	if _, err := mgr.ExecuteFor(context.Background(), "bash", "echo nope", ""); err == nil {
		t.Error("ExecuteFor executed under a disabled sandbox")
	}
	if _, err := mgr.ExecuteForLimits(context.Background(), "bash", "echo nope", "", LimitOverrides{}); err == nil {
		t.Error("ExecuteForLimits executed under a disabled sandbox")
	}
}

// TestEnabledSandboxStillExecutes is the negative control for the refusal
// above: the shipped configuration (enabled) must still run commands, so the
// fix cannot be "refuse everything".
func TestEnabledSandboxStillExecutes(t *testing.T) {
	mgr := NewSandboxManager(nil, slog.Default()) // shipped default: enabled
	res, err := mgr.Execute(context.Background(), "echo alive", LevelL0, mgr.DefaultLimits())
	if err != nil {
		t.Fatalf("an enabled sandbox refused to execute: %v", err)
	}
	if res == nil || !strings.Contains(res.Stdout, "alive") {
		t.Fatalf("enabled sandbox did not run the command: %+v", res)
	}
}
