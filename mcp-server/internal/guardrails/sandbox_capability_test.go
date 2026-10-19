package guardrails

// Spec 03 Sec. 2.2/3.3, Spec 06 probes — the sandbox readiness probe reported
// capability it never measured.
//
// Measured on this host before the fix: probeSandbox returned
//   FeatureActive, "isolation mechanisms present: unshare, podman, docker"
// while `unshare` could not create a namespace at all
//   unshare: write failed /proc/self/uid_map: Operation not permitted  (exit 1)
// and every policy-driven execution was refused, because the shipped tool
// policies are L1 and ExecuteFor floors at L1:
//   sandbox security violation detected: L1 setup failed and falling back
//   would run at L0, below the required floor L1
//
// Active is the ONLY status collectMeasuredReport counts as covered, so three
// requirements (eu_ai_act art_9, eu_ai_act art_15, nist_rmf manage) scored a
// control as COVERED on a host where it could not run once.
//
// The defect was the inference, not the hosts: LookPath answers "is a file
// installed", which is not "can a namespace be created". A namespace is a
// kernel grant. These tests pin the measurement, and each one fails on the
// previous implementation.

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"strings"
	"testing"
)

// fakeProber returns a prober whose answers are fixed by the caller.
func fakeProber(unshareExists bool, unshareErr error, runtime string, runtimeErr error) sandboxProber {
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "unshare" {
			if unshareErr != nil {
				return []byte("unshare: write failed /proc/self/uid_map: Operation not permitted\n"), unshareErr
			}
			return nil, nil
		}
		if runtime != "" && name == runtime {
			if runtimeErr != nil {
				return []byte("Cannot connect to the Docker daemon\n"), runtimeErr
			}
			return []byte("true\n"), nil
		}
		return nil, errors.New("not found")
	}
	lookPath := func(name string) (string, error) {
		if name == "unshare" && unshareExists {
			return "/usr/bin/unshare", nil
		}
		if name == runtime && runtime != "" {
			return "/usr/bin/" + runtime, nil
		}
		return "", exec.ErrNotFound
	}
	return sandboxProber{lookPath: lookPath, run: run}
}

func engineWithProber(t *testing.T, p sandboxProber) *Engine {
	t.Helper()
	cfg := DefaultEngineConfig()
	cfg.Sandbox.Enabled = true
	e := NewEngine(cfg, slog.New(slog.DiscardHandler))
	e.sandbox.prober = p
	return e
}

// The regression: a binary on PATH that cannot create a namespace is NOT
// active. This is the exact host measurement above.
func TestProbeIsPartialWhenUnshareCannotCreateNamespace(t *testing.T) {
	e := engineWithProber(t, fakeProber(true, errors.New("exit status 1"), "podman", nil))

	status, detail := e.probeSandbox(context.Background())

	if status == FeatureActive {
		t.Fatalf("probe reported ACTIVE for an unshare that cannot create a namespace; "+
			"presence on PATH is not capability. detail=%q", detail)
	}
	if status != FeaturePartial {
		t.Fatalf("status = %q, want partial (configured and present, but cannot execute)", status)
	}
	// The operator must be told what was measured, not just a status word.
	if !strings.Contains(detail, "cannot create a namespace") {
		t.Errorf("detail does not name the measured reason: %q", detail)
	}
	if !strings.Contains(detail, "L1") {
		t.Errorf("detail does not name the level that cannot run: %q", detail)
	}
}

// Negative control: the fix must not simply report partial forever. A host
// that genuinely creates a namespace is active.
func TestProbeIsActiveWhenNamespaceIsCreated(t *testing.T) {
	e := engineWithProber(t, fakeProber(true, nil, "", nil))

	status, detail := e.probeSandbox(context.Background())
	if status != FeatureActive {
		t.Fatalf("status = %q, want active when unshare created a namespace; detail=%q", status, detail)
	}
	if !strings.Contains(detail, "unshare") {
		t.Errorf("active detail does not name the mechanism actually measured: %q", detail)
	}
}

// A host with no mechanism at all stays partial, and says so without claiming
// a level it cannot run.
func TestProbeIsPartialWithNoIsolationMechanism(t *testing.T) {
	e := engineWithProber(t, fakeProber(false, nil, "", nil))

	status, detail := e.probeSandbox(context.Background())
	if status != FeaturePartial {
		t.Fatalf("status = %q, want partial with no mechanism installed", status)
	}
	if !strings.Contains(detail, "unshare is not installed") {
		t.Errorf("detail does not name the measured absence: %q", detail)
	}
}

// The probe must judge the STRONGEST declared policy, not the weakest. A
// deployment with one L2 policy is only as capable as L2.
func TestProbeJudgesTheStrongestDeclaredPolicy(t *testing.T) {
	e := engineWithProber(t, fakeProber(true, nil, "", nil)) // L1 only
	e.config.Sandbox.ToolPolicies["deploy"] = SandboxPolicy{DefaultLevel: LevelL2}

	status, detail := e.probeSandbox(context.Background())
	if status != FeaturePartial {
		t.Fatalf("status = %q, want partial: an L2 policy is declared and L2 cannot run; detail=%q",
			status, detail)
	}
	if !strings.Contains(detail, "L2") {
		t.Errorf("detail does not name the unmet L2 requirement: %q", detail)
	}
	if !strings.Contains(detail, "deploy") {
		t.Errorf("detail does not name the policy that requires it: %q", detail)
	}
}

// requiredSandboxLevel is deterministic across map iteration, so the reported
// policy cannot change run to run.
func TestRequiredSandboxLevelIsDeterministic(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.Sandbox.ToolPolicies["aaa"] = SandboxPolicy{DefaultLevel: LevelL1}
	cfg.Sandbox.ToolPolicies["zzz"] = SandboxPolicy{DefaultLevel: LevelL1}

	first, firstWho := requiredSandboxLevel(cfg)
	for i := 0; i < 50; i++ {
		level, who := requiredSandboxLevel(cfg)
		if level != first || who != firstWho {
			t.Fatalf("requiredSandboxLevel is not deterministic: %q/%q vs %q/%q",
				level, who, first, firstWho)
		}
	}
	if first != LevelL1 {
		t.Errorf("level = %q, want L1 (the shipped policies)", first)
	}
}

// The measurement execs a process, so it must happen once per manager — a
// compliance report must not fork a namespace per requirement.
func TestCapabilityIsMeasuredOnce(t *testing.T) {
	calls := 0
	base := fakeProber(true, nil, "", nil)
	counting := sandboxProber{
		lookPath: base.lookPath,
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			calls++
			return base.run(ctx, name, args...)
		},
	}
	e := engineWithProber(t, counting)

	for i := 0; i < 5; i++ {
		e.SandboxCapability(context.Background())
	}
	// unshare is the only mechanism installed in this fixture.
	if calls != 1 {
		t.Fatalf("capability measured %d times, want 1 (memoised per manager)", calls)
	}
}

// The report and the executor must not disagree about one process. This is
// the property the defect violated: probe Active, execution refused.
func TestProbeAgreesWithExecutorOnThisHost(t *testing.T) {
	e := NewEngine(DefaultEngineConfig(), slog.New(slog.DiscardHandler))
	ctx := context.Background()

	status, detail := e.probeSandbox(ctx)
	required, _ := requiredSandboxLevel(e.config)
	capability := e.sandbox.Capability(ctx)

	probeSaysRunnable := status == FeatureActive
	measuredRunnable := capability.Supports(required)

	if probeSaysRunnable != measuredRunnable {
		t.Fatalf("probe status %q disagrees with the measured capability (supports %s = %v); detail=%q",
			status, required, measuredRunnable, detail)
	}

	// If the probe claims the host can run the declared level, an execution at
	// that level must not fail for a capability reason. If it does not claim
	// it, a refusal is consistent and the probe must have said partial.
	if !probeSaysRunnable && status != FeaturePartial {
		t.Fatalf("host cannot run %s but probe reported %q, want partial", required, status)
	}
}

// Supports is the single level decision; pin it so a future level addition
// cannot default to "supported".
func TestCapabilitySupportsIsFailClosed(t *testing.T) {
	none := SandboxCapability{}
	if !none.Supports(LevelL0) {
		t.Error("L0 is host execution and is always supported")
	}
	if none.Supports(LevelL1) || none.Supports(LevelL2) {
		t.Error("an empty capability must support neither L1 nor L2")
	}
	if none.Supports(SandboxLevel("L9")) {
		t.Error("an unknown level must never be reported supported")
	}
	full := SandboxCapability{UnshareOK: true, ContainerOK: true}
	if !full.Supports(LevelL1) || !full.Supports(LevelL2) {
		t.Error("a fully measured host must support L1 and L2")
	}
}
