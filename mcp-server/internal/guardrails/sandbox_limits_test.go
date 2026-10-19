package guardrails

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// findFlagValue returns the value of flag in args, whether it was rendered as
// "--flag value" or "--flag=value", and whether it exists at all.
func findFlagValue(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
		if strings.HasPrefix(a, flag+"=") {
			return strings.TrimPrefix(a, flag+"="), true
		}
	}
	return "", false
}

func hasFlag(args []string, flag string) bool {
	_, ok := findFlagValue(args, flag)
	return ok
}

// TestSwapIsDisabledUnlessAllowed proves `swap_allowed` binds.
//
// The shipped default declares `swap_allowed: false` and, before this pass, no
// execution path read the field: the container got `--memory` and no
// `--memory-swap`, so the ceiling was advisory — the process swapped and kept
// running past the bound the config named. This is a config field with no
// consumer, the same class as a gate whose scan matches nothing.
func TestSwapIsDisabledUnlessAllowed(t *testing.T) {
	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Memory.MaxMB = 512

	t.Run("SwapAllowedFalsePinsMemorySwap", func(t *testing.T) {
		limits.Memory.SwapAllowed = false
		args := containerResourceArgs(limits)
		v, ok := findFlagValue(args, "--memory-swap")
		if !ok {
			t.Fatalf("swap_allowed=false emitted no --memory-swap; the memory ceiling is advisory: %v", args)
		}
		if v != "512m" {
			t.Errorf("--memory-swap = %q, want 512m (equal to --memory disables swap)", v)
		}
	})

	t.Run("SwapAllowedTrueDoesNotPinMemorySwap", func(t *testing.T) {
		limits.Memory.SwapAllowed = true
		args := containerResourceArgs(limits)
		if hasFlag(args, "--memory-swap") {
			t.Errorf("swap_allowed=true still emitted --memory-swap: %v", args)
		}
	})
}

// TestUnlimitedMemoryStillDisablesSwap proves the default branch is safe.
//
// A limit set with no memory bound must not silently acquire swap: the absent
// field means "no declared ceiling", and the fallback ceiling still refuses to
// swap past it.
func TestUnlimitedMemoryStillDisablesSwap(t *testing.T) {
	args := containerResourceArgs(ResourceLimits{})
	v, ok := findFlagValue(args, "--memory-swap")
	if !ok {
		t.Fatalf("no memory bound emitted no --memory-swap: %v", args)
	}
	if v != "256m" {
		t.Errorf("--memory-swap = %q, want 256m to match the fallback --memory", v)
	}
	if mem, ok := findFlagValue(args, "--memory"); !ok || mem != "256m" {
		t.Errorf("fallback --memory = %q (present=%v), want 256m", mem, ok)
	}
}

// TestMaxWriteMBSizesTheWritableSurface proves `max_write_mb` binds.
//
// The field was declared at 500 MB in the shipped config and read by nothing;
// the writable tmpfs was a hardcoded 64 MB, so the number an operator reviewed
// had no effect on what a command could write.
func TestMaxWriteMBSizesTheWritableSurface(t *testing.T) {
	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Disk.MaxWriteMB = 200
	args := containerHardeningArgs(limits)

	var tmpSize string
	for i, a := range args {
		if a == "--tmpfs" && i+1 < len(args) && strings.HasPrefix(args[i+1], "/tmp:") {
			tmpSize = args[i+1]
		}
	}
	if tmpSize == "" {
		t.Fatalf("no /tmp tmpfs emitted: %v", args)
	}
	if !strings.Contains(tmpSize, "size=200m") {
		t.Errorf("/tmp tmpfs = %q, want size=200m from max_write_mb", tmpSize)
	}
}

// TestMaxWriteMBFallbackIsBounded proves the absent bound stays bounded.
//
// A zero max_write_mb must not produce a zero-size tmpfs (every write fails for
// a reason the policy never stated) and must not produce an unbounded one.
func TestMaxWriteMBFallbackIsBounded(t *testing.T) {
	if got := tmpfsSizeMB(ResourceLimits{}, 64); got != 64 {
		t.Errorf("tmpfsSizeMB(zero) = %d, want the documented fallback 64", got)
	}
	if got := tmpfsSizeMB(ResourceLimits{Disk: struct {
		MaxWriteMB     int      `yaml:"max_write_mb"`
		ReadOnlyPaths  []string `yaml:"read_only_paths"`
		ReadWritePaths []string `yaml:"read_write_paths"`
	}{MaxWriteMB: 500}}, 64); got != 500 {
		t.Errorf("tmpfsSizeMB(500) = %d, want 500", got)
	}
}

// TestMaxWallSecondsBindsIndependently proves the second time field is real.
//
// Only MaxSeconds was ever read. A config declaring max_seconds: 300 with
// max_wall_seconds: 600 therefore had a 300 s bound, silently — and one
// declaring max_seconds: 600 / max_wall_seconds: 60 had no 60 s bound at all.
func TestMaxWallSecondsBindsIndependently(t *testing.T) {
	base := DefaultSandboxConfig().GlobalDefaults

	t.Run("WallTighterThanProcessWins", func(t *testing.T) {
		l := base
		l.Time.MaxSeconds = 600
		l.Time.MaxWallSeconds = 30
		if got := effectiveDeadline(l); got != 30*time.Second {
			t.Errorf("effectiveDeadline = %v, want 30s (the tighter declared bound)", got)
		}
	})

	t.Run("ProcessTighterThanWallWins", func(t *testing.T) {
		l := base
		l.Time.MaxSeconds = 45
		l.Time.MaxWallSeconds = 600
		if got := effectiveDeadline(l); got != 45*time.Second {
			t.Errorf("effectiveDeadline = %v, want 45s", got)
		}
	})

	t.Run("ZeroBoundsUseTheDocumentedDefault", func(t *testing.T) {
		// This is the fail-open that shipped: ResourceLimits{} produced a
		// zero-duration context, so every command was killed before it began
		// and the caller received ErrSandboxViolation for work that never ran.
		if got := effectiveDeadline(ResourceLimits{}); got != DefaultTimeoutSeconds*time.Second {
			t.Errorf("effectiveDeadline(zero) = %v, want %ds", got, DefaultTimeoutSeconds)
		}
	})
}

// TestZeroLimitsExecuteInsteadOfFailingClosed proves the engine's default path.
//
// engine.Evaluate passed ResourceLimits{} for any tool with no declared policy.
// The zero deadline made the command expire instantly, detectViolations
// recorded "resource limit exceeded (OOM/timeout)", and Execute returned
// ErrSandboxViolation — a security verdict about a command that never ran.
func TestZeroLimitsExecuteInsteadOfFailingClosed(t *testing.T) {
	requireSh(t)
	mgr := NewSandboxManager(nil, slog.Default())

	res, err := mgr.Execute(context.Background(), "echo alive", LevelL0, ResourceLimits{})
	if err != nil {
		t.Fatalf("zero limits failed closed instead of running: %v", err)
	}
	if res.ExitCode != 0 || strings.TrimSpace(res.Stdout) != "alive" {
		t.Fatalf("zero limits: exit=%d stdout=%q", res.ExitCode, res.Stdout)
	}
	for _, v := range res.SandboxViolations {
		t.Errorf("zero limits reported a violation for a command that ran: %q", v)
	}
}

// TestDefaultLimitsIsNotTheZeroValue proves the engine fallback is populated.
func TestDefaultLimitsIsNotTheZeroValue(t *testing.T) {
	mgr := NewSandboxManager(nil, slog.Default())
	l := mgr.DefaultLimits()
	if l.Time.MaxSeconds == 0 && l.Time.MaxWallSeconds == 0 {
		t.Fatal("DefaultLimits returned zero time bounds; the engine fallback would kill every command")
	}
	if l.Process.MaxProcesses <= 0 {
		t.Errorf("DefaultLimits has no process bound (%d); fork bombs would be unprevented", l.Process.MaxProcesses)
	}
	if effectiveDeadline(l) < time.Second {
		t.Errorf("DefaultLimits deadline = %v, too small to run anything", effectiveDeadline(l))
	}
}

// TestForkBombClassIsDetectedOnlyWhenABoundWasSet proves the violation class
// binds to a control that is actually in force, and cannot fabricate a verdict.
func TestForkBombClassIsDetectedOnlyWhenABoundWasSet(t *testing.T) {
	cfg := DefaultSandboxConfig()
	mgr := NewSandboxManager(cfg, slog.Default())
	ctx := context.Background()

	withBound := cfg.GlobalDefaults
	withBound.Process.MaxProcesses = 64
	noBound := cfg.GlobalDefaults
	noBound.Process.MaxProcesses = 0

	t.Run("ForkFailureUnderAnEnforcedBoundIsAViolation", func(t *testing.T) {
		// L2 is the level where a process-count bound is real (container pid
		// limit); at L0/L1 the class must not fire.
		res := &SandboxResult{ExitCode: 1, Stderr: "sh: 3: fork: retry: Resource temporarily unavailable"}
		mgr.detectViolations(res, LevelL2, withBound, ctx, nil)
		if len(res.SandboxViolations) == 0 {
			t.Fatal("fork failure under an enforced process bound was not reported")
		}
		if !strings.Contains(strings.Join(res.SandboxViolations, ";"), "fork bomb") {
			t.Errorf("violation text does not name the fork-bomb class: %v", res.SandboxViolations)
		}
	})

	t.Run("ForkFailureWithoutABoundIsNotAViolation", func(t *testing.T) {
		res := &SandboxResult{ExitCode: 1, Stderr: "sh: 3: fork: retry: Resource temporarily unavailable"}
		mgr.detectViolations(res, LevelL2, noBound, ctx, nil)
		if len(res.SandboxViolations) != 0 {
			t.Errorf("violation reported with no configured process bound: %v", res.SandboxViolations)
		}
	})

	t.Run("ForkFailureAtL0IsNotAViolation", func(t *testing.T) {
		// No isolation level imposes a process count at L0. Reporting one would
		// describe a control that is not in force.
		res := &SandboxResult{ExitCode: 1, Stderr: "sh: 3: fork: retry: Resource temporarily unavailable"}
		mgr.detectViolations(res, LevelL0, withBound, ctx, nil)
		if len(res.SandboxViolations) != 0 {
			t.Errorf("process violation reported at L0 where no bound applies: %v", res.SandboxViolations)
		}
	})

	t.Run("UnrelatedPermissionErrorIsNotAForkBomb", func(t *testing.T) {
		// The privilege-escalation class is deliberately prevention at the
		// boundary (cap-drop + user namespace), not text matching: grepping
		// stderr for "Operation not permitted" would flag ordinary failures.
		res := &SandboxResult{ExitCode: 1, Stderr: "touch: cannot touch '/etc/x': Operation not permitted"}
		mgr.detectViolations(res, LevelL2, withBound, ctx, nil)
		if len(res.SandboxViolations) != 0 {
			t.Errorf("ordinary permission error reported as a violation: %v", res.SandboxViolations)
		}
	})
}

// TestCPUBoundReachesTheProcess proves a declared bound is applied by the
// process itself, not merely written into a config struct.
//
// It reads RLIMIT_CPU back from /proc rather than inferring it from a command's
// exit status: a spinning loop's exit code depends on how the shell gives up,
// so a test that greps for a failure proves a different thing on every machine.
// Reading the limit back proves exactly the claim — the value the policy
// declares is the value the process runs under.
func TestCPUBoundReachesTheProcess(t *testing.T) {
	requireSh(t)
	mgr := NewSandboxManager(nil, slog.Default())

	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Time.MaxSeconds = 42
	limits.Time.MaxWallSeconds = 42

	res, err := mgr.Execute(context.Background(), "cat /proc/self/limits", LevelL0, limits)
	if err != nil {
		t.Fatalf("limits read failed: %v", err)
	}
	soft, ok := parseLimitRow(res.Stdout, "Max cpu time")
	if !ok {
		t.Fatalf("no CPU-time row in /proc/self/limits:\n%s", res.Stdout)
	}
	if soft != "42" {
		t.Fatalf("RLIMIT_CPU soft limit = %q, want 42 from max_seconds", soft)
	}
}

// TestNoDeclaredBoundLeavesTheHostLimitsAlone proves an undeclared CPU bound
// does not silently impose one.
func TestNoDeclaredBoundLeavesTheHostLimitsAlone(t *testing.T) {
	requireSh(t)
	mgr := NewSandboxManager(nil, slog.Default())

	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Time.MaxSeconds = 0
	limits.Time.MaxWallSeconds = 10

	res, err := mgr.Execute(context.Background(), "cat /proc/self/limits", LevelL0, limits)
	if err != nil {
		t.Fatalf("limits read failed: %v", err)
	}
	soft, ok := parseLimitRow(res.Stdout, "Max cpu time")
	if !ok {
		t.Fatalf("no CPU-time row in /proc/self/limits:\n%s", res.Stdout)
	}
	if soft == "0" {
		t.Fatal("no declared CPU bound produced RLIMIT_CPU = 0, which kills the command immediately")
	}
}

// parseLimitRow pulls the soft limit out of a /proc/self/limits row.
func parseLimitRow(text, name string) (string, bool) {
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, name) {
			continue
		}
		fields := strings.Fields(line)
		// Fields: <name words...> <soft> <hard> <units>
		if len(fields) >= 3 {
			return fields[len(fields)-3], true
		}
	}
	return "", false
}

// TestProcessBoundIsRenderedForContainer proves --pids-limit is emitted, which
// is the only process bound a container runtime itself enforces.
func TestProcessBoundIsRenderedForContainer(t *testing.T) {
	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Process.MaxProcesses = 128
	args := containerResourceArgs(limits)
	v, ok := findFlagValue(args, "--pids-limit")
	if !ok {
		t.Fatalf("max_processes=128 emitted no --pids-limit: %v", args)
	}
	if v != "128" {
		t.Errorf("--pids-limit = %q, want 128", v)
	}

	limits.Process.MaxProcesses = 0
	if hasFlag(containerResourceArgs(limits), "--pids-limit") {
		t.Error("an undeclared process bound emitted --pids-limit")
	}
}

// TestLimitLauncherUsesRealEnforcement proves the bound is applied by a
// mechanism a shell cannot ignore.
//
// `sh` on Debian is dash, which rejects `ulimit -u`; a shell prefix therefore
// wrote "Illegal option -u" to stderr, swallowed it, and ran the command
// unbounded. My own first attempt at this fix had exactly that defect, and the
// assertion below is what caught it.
func TestLimitLauncherUsesRealEnforcement(t *testing.T) {
	t.Run("NoBoundsRunsPlainShell", func(t *testing.T) {
		launcher, args, err := limitLauncher(ResourceLimits{}, "echo hi")
		if err != nil {
			t.Fatalf("no bounds refused: %v", err)
		}
		if launcher != "sh" {
			t.Errorf("launcher = %q, want sh when nothing is declared", launcher)
		}
		if len(args) != 2 || args[0] != "-c" {
			t.Errorf("args = %v, want [-c <command>]", args)
		}
	})

	t.Run("DeclaredCPUBoundGoesThroughPrlimit", func(t *testing.T) {
		l := DefaultSandboxConfig().GlobalDefaults
		launcher, args, err := limitLauncher(l, "echo hi")
		if err != nil {
			t.Fatalf("prlimit path failed: %v", err)
		}
		if launcher != "prlimit" {
			t.Fatalf("launcher = %q, want prlimit; a shell builtin silently no-ops under dash", launcher)
		}
		if !hasFlag(args, "--cpu") {
			t.Errorf("--cpu missing: %v", args)
		}
		// RLIMIT_NPROC is per-UID, so it must never be passed as a
		// per-execution bound: it would fail every fork on a busy host.
		if hasFlag(args, "--nproc") {
			t.Errorf("--nproc must not be used as a per-execution bound (it is per-UID): %v", args)
		}
	})
}

// TestDeclaredBoundCannotSilentlyDisappear proves a declared bound reaches
// argv and cannot be dropped on the way.
func TestDeclaredBoundCannotSilentlyDisappear(t *testing.T) {
	l := DefaultSandboxConfig().GlobalDefaults
	l.Time.MaxSeconds = 32
	launcher, args, err := limitLauncher(l, "true")
	if err != nil {
		t.Skipf("prlimit unavailable on this host: %v", err)
	}
	if launcher != "prlimit" || !hasFlag(args, "--cpu") {
		t.Fatalf("declared CPU bound did not reach argv: launcher=%q args=%v", launcher, args)
	}
}
