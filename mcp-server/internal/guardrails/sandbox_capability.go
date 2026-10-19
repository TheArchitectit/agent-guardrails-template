package guardrails

// Spec 03 Sec. 2.2 — sandbox CAPABILITY, measured.
//
// This file exists because the previous answer was an inference. The readiness
// probe ran LookPath over unshare/podman/docker and called any hit "active",
// while the executor refused every policy-driven call on the same host:
//
//	unshare: write failed /proc/self/uid_map: Operation not permitted   (exit 1)
//	sandbox security violation detected: L1 setup failed and falling back
//	would run at L0, below the required floor L1
//
// Presence on PATH is not capability. A namespace is a kernel grant; a
// container runtime binary is not a reachable daemon. Active is the only status
// the compliance report counts as covered, so three requirements scored a
// control as covered on a host where it could not run once.
//
// The measurement lives in its own declared file, and `sandbox_capability.go`
// is listed in the `sandbox-isolation` component's `source_paths`. The probe
// now decides a published compliance verdict, so a drift here must move the
// component digest: while this logic sat in a shared probe file that no
// component declared, the report could have changed without the manifest
// noticing — the exact drift the digest gate exists to catch.

import (
	"context"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// sandboxProber is the injection seam for capability measurement.
type sandboxProber struct {
	lookPath func(string) (string, error)
	run      func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func defaultSandboxProber() sandboxProber {
	return sandboxProber{
		lookPath: exec.LookPath,
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
	}
}

// SandboxCapability is the MEASURED answer to "what can this host actually
// run", as opposed to "what is installed on PATH".
type SandboxCapability struct {
	// UnshareOK is true only when a namespace was actually created.
	UnshareOK bool
	// UnshareDetail always says what was measured, success or failure.
	UnshareDetail string
	// ContainerOK is true only when a runtime answered.
	ContainerOK bool
	// ContainerDetail always says what was measured.
	ContainerDetail string
	// Runtime names the container runtime that answered, empty otherwise.
	Runtime string
}

// Supports reports whether the host can execute at the given level.
//
// L0 is host execution: it is always "supported", and that is exactly why it
// is never sufficient on its own — there is no isolation to check.
func (c SandboxCapability) Supports(level SandboxLevel) bool {
	switch level {
	case LevelL0:
		return true
	case LevelL1:
		return c.UnshareOK
	case LevelL2:
		return c.ContainerOK
	}
	return false
}

// ReasonFor returns the measured evidence for a level, so a caller can name
// why it is unavailable instead of only that it is.
func (c SandboxCapability) ReasonFor(level SandboxLevel) string {
	switch level {
	case LevelL1:
		return c.UnshareDetail
	case LevelL2:
		return c.ContainerDetail
	case LevelL0:
		return "no isolation is applied at L0"
	}
	return "unknown isolation level"
}

// capabilityProbeTimeout bounds each measurement. A hung runtime must not be
// reported as available, and must not hang the report either.
const capabilityProbeTimeout = 5 * time.Second

// Capability measures (once) and returns what this host can actually run.
func (m *SandboxManager) Capability(ctx context.Context) SandboxCapability {
	m.capOnce.Do(func() {
		m.cap = m.measureCapability(ctx)
	})
	return m.cap
}

// measureCapability execs the mechanisms and records what happened.
func (m *SandboxManager) measureCapability(ctx context.Context) SandboxCapability {
	p := m.prober
	var c SandboxCapability

	// L1: the exact argv execL1 builds. Measuring a different shape would
	// prove a capability the execution path does not use.
	if _, err := p.lookPath("unshare"); err != nil {
		c.UnshareDetail = "unshare is not installed"
	} else {
		pctx, cancel := context.WithTimeout(ctx, capabilityProbeTimeout)
		out, err := p.run(pctx, "unshare",
			"--pid", "--mount", "--net", "--uts", "--user", "--map-root-user", "--fork", "true")
		cancel()
		if err == nil {
			c.UnshareOK = true
			c.UnshareDetail = "unshare created a namespace"
		} else {
			c.UnshareDetail = "unshare cannot create a namespace: " + firstLine(string(out))
		}
	}

	// L2: a runtime that answers is a runtime that can be asked to run.
	// Named limit: this proves the daemon is reachable, not that a container
	// starts — pulling/booting an image inside a report would cost seconds
	// and could fail on a registry outage unrelated to this host.
	for _, rt := range []string{"podman", "docker"} {
		if _, err := p.lookPath(rt); err != nil {
			continue
		}
		pctx, cancel := context.WithTimeout(ctx, capabilityProbeTimeout)
		_, err := p.run(pctx, rt, "info")
		cancel()
		if err == nil {
			c.ContainerOK = true
			c.Runtime = rt
			c.ContainerDetail = rt + " daemon answered"
			break
		}
		c.ContainerDetail = rt + " is installed but its daemon did not answer"
	}
	if c.ContainerDetail == "" {
		c.ContainerDetail = "no container runtime is installed"
	}

	return c
}

// SandboxCapability exposes the measured capability on the engine, so the
// readiness layer and the compliance report read ONE decision rather than two
// that can disagree about the same process.
func (e *Engine) SandboxCapability(ctx context.Context) (SandboxCapability, bool) {
	if e == nil || e.sandbox == nil {
		return SandboxCapability{}, false
	}
	return e.sandbox.Capability(ctx), true
}

// ProbeSandbox exposes the sandbox readiness verdict to layers that must not
// disagree with it.
//
// The readiness observer originally derived the component's state from
// `cfg.Enabled` alone and reported `reachable`, while the compliance probe
// reported Active off a PATH test and the executor refused every call. Three
// surfaces, three answers, one process. They now read this single decision.
func (e *Engine) ProbeSandbox(ctx context.Context) (FeatureStatus, string) {
	return e.probeSandbox(ctx)
}

// probeSandbox reports whether the sandbox can actually EXECUTE at the level
// its policies demand — not whether the usual binaries are installed.
//
// Status mapping, fail-closed:
//   - absent:  the manager is not enabled.
//   - partial: enabled, but the host cannot execute at the level the policies
//     require. Present-but-unusable is PARTIAL, never Active — it is not
//     absent because the configuration and the code exist, and it is not
//     Active because nothing can run under it.
//   - active:  the host can execute at the declared level.
func (e *Engine) probeSandbox(ctx context.Context) (FeatureStatus, string) {
	if e == nil || e.sandbox == nil || e.config == nil || !e.config.Sandbox.Enabled {
		return FeatureAbsent, "sandbox manager is not enabled"
	}

	capability := e.sandbox.Capability(ctx)

	// The level that matters is the one the shipped policies ask for. Reading
	// a single fixed level would let the probe pass while a policy asks for
	// more than the host can do.
	required, declaredBy := requiredSandboxLevel(e.config)
	if capability.Supports(required) {
		// The evidence list is built from the MEASUREMENT, never re-read from
		// PATH: re-checking LookPath here would reintroduce the presence-as-
		// capability inference this probe exists to remove.
		var have []string
		if capability.UnshareOK {
			have = append(have, "unshare")
		}
		if capability.ContainerOK && capability.Runtime != "" {
			have = append(have, capability.Runtime)
		}
		return FeatureActive, "sandbox can execute at " + string(required) +
			" (required by " + declaredBy + "): " + strings.Join(have, ", ")
	}

	// Enabled but unenforceable. Name the level, the policy that asked for it,
	// and the measured reason, so the operator is told what to fix rather than
	// handed a status word.
	return FeaturePartial, "sandbox enabled but cannot execute at the required level " +
		string(required) + " (required by " + declaredBy + "); measured: " +
		capability.ReasonFor(required)
}

// requiredSandboxLevel returns the highest isolation level any declared tool
// policy asks for, plus the policy that asked for it.
//
// Highest, not first: a deployment where one policy demands L1 and another L2
// is only as capable as the L2 requirement, and reporting the weakest would
// let the report pass on a host that cannot run the strongest declared policy.
func requiredSandboxLevel(cfg *EngineConfig) (SandboxLevel, string) {
	best := LevelL0
	who := "no tool policy (L0)"
	if cfg == nil {
		return best, who
	}
	names := make([]string, 0, len(cfg.Sandbox.ToolPolicies))
	for name := range cfg.Sandbox.ToolPolicies {
		names = append(names, name)
	}
	// Sorted so the reported policy is deterministic across map iteration.
	sort.Strings(names)
	for _, name := range names {
		level := cfg.Sandbox.ToolPolicies[name].DefaultLevel
		if !validLevel(level) {
			continue
		}
		if LevelRank(level) > LevelRank(best) {
			best = level
			who = "tool policy " + name
		}
	}
	return best, who
}
