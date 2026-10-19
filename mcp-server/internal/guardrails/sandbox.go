package guardrails

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	// ErrSandboxViolation is returned when a security boundary is breached.
	ErrSandboxViolation = errors.New("sandbox security violation detected")
	// ErrResourceExceeded is returned when a resource limit is exceeded.
	ErrResourceExceeded = errors.New("sandbox resource limit exceeded")
)

// SandboxResult contains the outcome of a sandboxed execution.
type SandboxResult struct {
	ExitCode           int           `json:"exit_code"`
	Stdout             string        `json:"stdout"`
	Stderr             string        `json:"stderr"`
	SandboxViolations  []string      `json:"sandbox_violations"`
	ActualIsolationLvl SandboxLevel  `json:"actual_isolation_level"`
	ExecutionTime      time.Duration `json:"execution_time"`
}

// SandboxManager handles the lifecycle of isolated executions.
//
// The config is guarded by mu because a policy can be changed after
// construction (guardrail_sandbox_config). Execute reads it under a read lock
// and SetToolPolicy replaces the whole validated config atomically, so a
// concurrent read can never observe a half-written policy table.
type SandboxManager struct {
	mu     sync.RWMutex
	config *SandboxConfig
	logger *slog.Logger
}

// NewSandboxManager creates a new manager with the provided configuration.
func NewSandboxManager(cfg *SandboxConfig, logger *slog.Logger) *SandboxManager {
	if cfg == nil {
		cfg = DefaultSandboxConfig()
	}
	return &SandboxManager{
		config: cfg.Clone(),
		logger: logger,
	}
}

// sandboxSetupErr wraps an infrastructure error that occurred BEFORE the
// command itself began executing under isolation. Only these errors may
// justify a fallback to a lower isolation level.
type sandboxSetupErr struct{ msg string }

func (e *sandboxSetupErr) Error() string { return e.msg }

// isSetupError reports whether err is a genuine infrastructure-setup failure
// (e.g. the container runtime or unshare binary is missing, or the container
// could not be started at all) rather than a rejection of the command that
// actually ran inside the sandbox. Container runtime exit codes 125/126 mean
// the container failed to start — treated as setup, NOT a command denial.
func isSetupError(err error) bool {
	if err == nil {
		return false
	}
	var s *sandboxSetupErr
	if errors.As(err, &s) {
		return true
	}
	msg := err.Error()
	if strings.Contains(msg, "executable file not found") ||
		strings.Contains(msg, "No such image") ||
		strings.Contains(msg, "unshare") ||
		strings.Contains(msg, "exit status 125") ||
		strings.Contains(msg, "exit status 126") {
		return true
	}
	return false
}

// nextFallbackLevel returns the level the fallback chain would try after
// failing at the given level, and whether a fallback exists at all.
//
// Pure and exported-by-test so the downgrade ladder can be asserted without
// executing anything: the security property (never below L1 for an isolating
// caller) must be provable on a runner that HAS user namespaces and on one
// that does not, and a test that shells out proves a different thing on each.
func nextFallbackLevel(from SandboxLevel) (SandboxLevel, bool) {
	switch from {
	case LevelL2:
		return LevelL1, true
	case LevelL1:
		return LevelL0, true
	}
	return from, false
}

// WouldRunBelowFloor reports whether a setup failure at `from` would make the
// chain execute below `floor`. True means the command would run on the host.
func WouldRunBelowFloor(from, floor SandboxLevel) bool {
	next, ok := nextFallbackLevel(from)
	if !ok {
		return false
	}
	return LevelRank(next) < LevelRank(floor)
}

// Execute runs the given command under the appropriate isolation level and
// resource limits.
//
// Fail-closed policy: isolation level is never DOWNGRADED on error. If a
// command is denied or fails while executing under L2/L1, Execute returns the
// error immediately instead of re-running the same command on a less
// isolated level (which would run it on the host with no isolation at all).
//
// The only permitted downgrade is for genuine infrastructure-setup errors —
// failures that occur before the command ever begins executing, such as a
// missing container runtime or unshare binary. Those are surfaced via
// sandboxSetupErr. A command that actually began running under isolation and
// was then denied (permission denied, exit 125/126 from a started container,
// etc.) is treated as a security rejection and fails closed.
func (m *SandboxManager) Execute(ctx context.Context, command string, level SandboxLevel, limits ResourceLimits) (*SandboxResult, error) {
	return m.execute(ctx, command, level, limits, LevelL0)
}

// ExecuteWithFloor is Execute with a minimum isolation level that the fallback
// chain may not go below.
//
// This is a security property, not a preference. The shipped fallback chain
// downgrades L2 -> L1 -> L0 on setup errors, and L0 is direct host execution.
// For the generic library that is arguably acceptable (a caller that asked for
// L1 and got L0 is told so via ActualIsolationLvl). For a tool whose entire
// purpose is isolation it is a fail-open: an environment without user
// namespaces silently runs the command on the host, and the caller reads a
// result rather than a refusal. Callers that must not reach the host pass a
// floor of L1; below it the chain STOPS and returns the setup error.
func (m *SandboxManager) ExecuteWithFloor(ctx context.Context, command string, level SandboxLevel, limits ResourceLimits, floor SandboxLevel) (*SandboxResult, error) {
	return m.execute(ctx, command, level, limits, floor)
}

func (m *SandboxManager) execute(ctx context.Context, command string, level SandboxLevel, limits ResourceLimits, floor SandboxLevel) (*SandboxResult, error) {
	start := time.Now()
	var result *SandboxResult
	var err error

	// A level the implementation does not have must not reach the switch below
	// and must not be interpreted as "no isolation".
	if !validLevel(level) {
		return nil, fmt.Errorf("unsupported sandbox level: %q", level)
	}

	// Reject malformed mount paths before any level is tried, so a missing
	// container runtime can never mask the error by falling back to a level
	// that ignores the mount.
	for _, p := range limits.Disk.ReadWritePaths {
		if verr := validateMountPath(p); verr != nil {
			return nil, verr
		}
	}

	actualLevel := level
	// fallbackWasFailClosed is true when the chain downgraded because a
	// command was denied/rejected while executing under isolation (as opposed
	// to a genuine setup-time infrastructure error). A fail-closed downgrade
	// is itself a security boundary breach; a setup fallback is not.
	fallbackWasFailClosed := false

	for {
		switch actualLevel {
		case LevelL2:
			result, err = m.execL2(ctx, command, limits)
		case LevelL1:
			result, err = m.execL1(ctx, command, limits)
		case LevelL0:
			result, err = m.execL0(ctx, command, limits)
		default:
			return nil, fmt.Errorf("unsupported sandbox level: %s", level)
		}

		if err == nil || !m.configSnapshot().FallbackEnabled {
			break
		}
		// Only genuine setup-time failures may justify a downgrade. A command
		// that was denied while executing under isolation fails closed.
		if !isSetupError(err) {
			m.logger.Warn("sandbox execution failed closed (no downgrade)",
				"requested_level", level, "actual_level", actualLevel, "error", err)
			fallbackWasFailClosed = true
			break
		}

		next, ok := nextFallbackLevel(actualLevel)
		if !ok {
			break
		}
		if LevelRank(next) < LevelRank(floor) {
			// The fallback would cross the floor. Refusing here is the whole
			// point: without it, an environment with no user namespaces runs
			// the command directly on the host and hands the caller a result.
			m.logger.Error("sandbox setup failed and the fallback would cross the isolation floor; refusing to execute",
				"requested_level", level, "failed_level", actualLevel, "floor", floor, "error", err)
			return nil, fmt.Errorf("%w: %s setup failed and falling back would run at %s, below the required floor %s: %v",
				ErrSandboxViolation, actualLevel, next, floor, err)
		}
		m.logger.Warn("sandbox setup failed, falling back to a lower isolation level",
			"from", actualLevel, "to", next, "error", err)
		actualLevel = next
		continue
	}

	if err != nil && result == nil {
		return nil, err
	}

	if result != nil {
		result.ActualIsolationLvl = actualLevel
		result.ExecutionTime = time.Since(start)
		// A command that ran at a lower isolation level than requested due to
		// a fail-closed denial (NOT a setup fallback) is a security breach.
		if fallbackWasFailClosed && actualLevel < level {
			result.SandboxViolations = append(result.SandboxViolations,
				fmt.Sprintf("requested isolation %s but executed at lower level %s due to execution denial", level, actualLevel))
		}
		if len(result.SandboxViolations) > 0 {
			return result, SandboxViolationsError(result.SandboxViolations)
		}
	}

	return result, err
}

// configSnapshot returns the active configuration under a read lock.
func (m *SandboxManager) configSnapshot() *SandboxConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// ExecuteFor resolves the effective isolation level and resource limits for a
// tool type, then executes under them.
//
// This is the seam the sandbox library never had. Before it, every caller had
// to pass a level and a limit set by hand, so the declared per-tool
// SandboxConfig.ToolPolicies were read by nothing and a request could choose
// L0 for a tool the deployment had classified L2.
//
// requestedLevel may be empty, in which case the tool policy decides. A
// non-empty request may RAISE isolation but never lower it: the effective
// level is the more isolated of the two, and the result records which level
// actually ran.
//
// An unknown tool type is refused rather than executed under the global
// defaults: "this tool type has no policy" is a configuration gap, and
// silently running it at the least isolated level is the fail-open reading.
func (m *SandboxManager) ExecuteFor(ctx context.Context, toolType, command string, requestedLevel SandboxLevel) (*SandboxResult, error) {
	cfg := m.configSnapshot()
	if cfg == nil {
		return nil, fmt.Errorf("sandbox is not configured")
	}
	if strings.TrimSpace(toolType) == "" {
		return nil, fmt.Errorf("tool_type is required")
	}
	if _, known := cfg.ToolPolicies[toolType]; !known {
		return nil, fmt.Errorf("no sandbox policy for tool type %q", toolType)
	}
	policyLevel, limits := cfg.ResolveToolPolicy(toolType)
	if policyLevel == "" {
		return nil, fmt.Errorf("sandbox policy for tool type %q declares no level", toolType)
	}

	if requestedLevel != "" && !validLevel(requestedLevel) {
		return nil, fmt.Errorf("unsupported sandbox level: %q", requestedLevel)
	}
	effective := maxLevel(policyLevel, requestedLevel)
	if effective != policyLevel {
		// Recorded, not a violation: the caller asked for MORE isolation than
		// the policy requires, which is permitted.
		m.logger.Debug("sandbox isolation raised above policy",
			"tool_type", toolType, "policy_level", policyLevel,
			"requested_level", requestedLevel, "effective_level", effective)
	}

	// Floor of L1, for the same reason as ExecuteForLimits: this entry point is
	// the policy-driven one, and a policy-driven call that silently reaches the
	// host is the fail-open the policy table exists to prevent.
	return m.ExecuteWithFloor(ctx, command, effective, limits, LevelL1)
}

// execL0 executes the command directly on the host.
func (m *SandboxManager) execL0(ctx context.Context, command string, limits ResourceLimits) (*SandboxResult, error) {
	m.logger.Debug("executing L0 (no isolation)", "command", command)

	ctx, cancel := context.WithTimeout(ctx, effectiveDeadline(limits))
	defer cancel()

	launcher, largs, lerr := limitLauncher(limits, command)
	if lerr != nil {
		return nil, lerr
	}
	cmd := exec.CommandContext(ctx, launcher, largs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	result := &SandboxResult{
		ExitCode:           exitCode,
		Stdout:             stdout.String(),
		Stderr:             stderr.String(),
		ActualIsolationLvl: LevelL0,
	}
	m.detectViolations(result, LevelL0, limits, ctx, err)
	return result, err
}

// execL1 executes the command using Linux namespaces via unshare(2).
// Provides process-level isolation: PID, mount, network, UTS, and user namespaces.
// NOTE: L1 provides process separation, not full VM-level isolation.
func (m *SandboxManager) execL1(ctx context.Context, command string, limits ResourceLimits) (*SandboxResult, error) {
	m.logger.Debug("executing L1 (namespaces)", "command", command)

	args := []string{
		"--pid",           // Isolate PID namespace
		"--mount",         // Isolate mount namespace
		"--net",           // Isolate network namespace
		"--uts",           // Isolate UTS namespace (hostname)
		"--user",          // Isolate user namespace (maps to unprivileged)
		"--map-root-user", // Map to root inside namespace
		"--fork",          // Fork before exec
	}
	launcher, largs, lerr := limitLauncher(limits, command)
	if lerr != nil {
		return nil, lerr
	}
	args = append(args, launcher)
	args = append(args, largs...)

	ctx, cancel := context.WithTimeout(ctx, effectiveDeadline(limits))
	defer cancel()

	cmd := exec.CommandContext(ctx, "unshare", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	result := &SandboxResult{
		ExitCode:           exitCode,
		Stdout:             stdout.String(),
		Stderr:             stderr.String(),
		ActualIsolationLvl: LevelL1,
	}
	m.detectViolations(result, LevelL1, limits, ctx, err)

	// unshare failing to CREATE the namespace is a setup failure, not a denial
	// of the command: the command never started. Without this, a host with
	// user namespaces disabled produced exit 1 with unshare's own message on
	// stderr, which isSetupError classified as "the command ran and was
	// denied" — so the caller got a security-violation verdict about a command
	// that had never executed, and the documented setup fallback never ran.
	if err != nil && unshareSetupFailed(result.Stderr) {
		return result, &sandboxSetupErr{msg: "unshare could not create the namespace: " + firstLine(result.Stderr)}
	}
	return result, err
}

// unshareSetupFailed reports whether stderr came from unshare itself rather
// than from the command it was asked to run. unshare prefixes its own errors
// with "unshare:", and the kernel's namespace-creation failures surface as
// uid_map/gid_map writes or an explicit "Operation not permitted" from it.
func unshareSetupFailed(stderr string) bool {
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "unshare:") {
			return true
		}
		if strings.Contains(line, "uid_map") || strings.Contains(line, "gid_map") {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// execL2 executes the command inside a rootless container.
// Provides full filesystem + resource isolation via container runtime.
func (m *SandboxManager) execL2(ctx context.Context, command string, limits ResourceLimits) (*SandboxResult, error) {
	m.logger.Debug("executing L2 (container)", "command", command)

	runtime := "podman"
	if _, err := exec.LookPath(runtime); err != nil {
		runtime = "docker"
		if _, err := exec.LookPath(runtime); err != nil {
			// Setup-time failure: the container runtime is simply not
			// installed. This is a sandboxSetupErr so a fallback to a lower
			// isolation level is permitted — the command never ran.
			return nil, &sandboxSetupErr{msg: "neither podman nor docker found in PATH"}
		}
	}

	args := containerHardeningArgs(limits)

	// Network isolation.
	//
	//   - AllowedHosts empty => --network=none (fully offline). Fail-closed.
	//   - AllowedHosts set AND no pre-configured NetworkMode =>
	//       provision/reuse the "guardrail-egress" bridge network and run a
	//       local CONNECT filtering proxy (sandbox_network.go). The container
	//       routes HTTPS through the proxy via HTTPS_PROXY/HTTP_PROXY env.
	//   - NetworkMode set => use it directly (operator-managed network); the
	//       proxy is still wired when AllowedHosts is set.
	//
	// If proxy startup or network provisioning fails, we degrade the FEATURE
	// (network egress), not the whole sandbox: fall back to --network=none and
	// record a warning. This is NOT a sandboxSetupErr — execution setup for a
	// requested feature failed, so we fail closed on the feature.
	netMode := m.configSnapshot().NetworkMode
	egressProxyAddr := ""
	if len(limits.Network.AllowedHosts) > 0 {
		if err := ValidateAllowedHosts(limits.Network.AllowedHosts); err != nil {
			// Overly broad allowlist (e.g. bare "*"): refuse to open egress.
			m.logger.Warn("invalid AllowedHosts; using network=none (fail-closed)",
				"error", err, "allowed_hosts", limits.Network.AllowedHosts)
			netMode = ""
		} else if proxy, addr, perr := startEgressProxy(limits.Network.AllowedHosts, m.logger); perr == nil {
			defer proxy.stopEgressProxy()
			egressProxyAddr = addr
			if netMode == "" {
				if nm, nerr := provisionEgressNetwork(runtime, m.logger); nerr == nil {
					netMode = nm
				} else {
					// Provisioning failed: refuse egress entirely.
					netMode = ""
					egressProxyAddr = ""
					proxy.stopEgressProxy()
				}
			}
		} else {
			m.logger.Warn("egress proxy startup failed; using network=none (fail-closed)",
				"error", perr, "allowed_hosts", limits.Network.AllowedHosts)
			netMode = ""
		}
	}
	if netMode != "" {
		args = append(args, "--network="+netMode)
	} else {
		args = append(args, "--network=none")
	}
	if egressProxyAddr != "" {
		proxyURL := egressProxyURL(egressProxyAddr)
		args = append(args, "--env", "HTTPS_PROXY="+proxyURL, "--env", "HTTP_PROXY="+proxyURL)
	}

	// Resource limits — guard against zero values
	//
	// Every declared limit binds here. `swap_allowed`, `max_write_mb` and the
	// process bound were declared in the shipped default config and consumed by
	// nothing, so a deployment read a config that promised three controls and
	// got none of them. A limit with no consumer is worse than an absent one:
	// it produces a review trail that says the bound exists.
	args = append(args, containerResourceArgs(limits)...)

	// Bind mount declared workspace paths (read-write). Paths containing ':'
	// or ',' corrupt the -v spec (':' separates host:container:mode and ','
	// separates mount options), so reject them instead of building a malformed
	// mount that could escape the intended path.
	for _, path := range limits.Disk.ReadWritePaths {
		if err := validateMountPath(path); err != nil {
			return nil, err
		}
		args = append(args, "-v", fmt.Sprintf("%s:%s:rw", path, path))
	}
	// Bind mount declared read-only paths
	for _, path := range limits.Disk.ReadOnlyPaths {
		if err := validateMountPath(path); err != nil {
			return nil, err
		}
		args = append(args, "-v", fmt.Sprintf("%s:%s:ro", path, path))
	}

	// Use configurable image, fallback to alpine
	image := m.configSnapshot().Image
	if image == "" {
		image = "alpine:latest"
	}
	// At L2 the runtime enforces the process bound itself (--pids-limit) and the
	// memory/CPU ceilings via cgroups, so no in-shell prefix is needed or wanted:
	// requiring prlimit inside the image would make an otherwise-hardened
	// container fail for a reason the container already handles.
	args = append(args, image, "sh", "-c", command)

	ctx, cancel := context.WithTimeout(ctx, effectiveDeadline(limits))
	defer cancel()

	cmd := exec.CommandContext(ctx, runtime, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	result := &SandboxResult{
		ExitCode:           exitCode,
		Stdout:             stdout.String(),
		Stderr:             stderr.String(),
		ActualIsolationLvl: LevelL2,
	}
	m.detectViolations(result, LevelL2, limits, ctx, err)
	return result, err
}

// detectViolations inspects an executed result and records any sandbox
// security boundary breaches in result.SandboxViolations. A non-empty
// SandboxViolations means ErrSandboxViolation is returned by the caller path.
//
// Detected cases:
//   - resource limit exceeded: exit code 137 (OOM/sigkill) or ctx timeout
//   - process limit reached (see processLimitSignatures): only reported when a
//     process bound was actually configured for this execution
//   - network access attempt outside AllowedHosts (best-effort)
//
// The "ran at a lower isolation level than requested" breach is recorded by
// Execute against the requested level, not here (exec funcs don't know it).
//
// Privilege escalation is deliberately NOT detected from output text. At L2 it
// is prevented (--cap-drop=ALL plus no-new-privileges); at L1 the user
// namespace maps root to an unprivileged host user. Grepping stderr for
// "Operation not permitted" would flag ordinary permission errors — the same
// false-positive generator that Law 2 became in gap-spec 04 — so the control
// here is prevention at the boundary, not text matching.
func (m *SandboxManager) detectViolations(result *SandboxResult, actualLevel SandboxLevel, limits ResourceLimits, ctx context.Context, execErr error) {
	// Resource limit exceeded: OOM kill / SIGKILL, or the execution context
	// deadline was hit (timeout). When the context deadline fires, exec kills
	// the process and returns "signal: killed"; we also check ctx.Err() so
	// the timeout is reliably detected regardless of how the error surfaces.
	if result.ExitCode == 137 || errors.Is(ctx.Err(), context.DeadlineExceeded) ||
		(errors.Is(execErr, context.DeadlineExceeded)) {
		result.SandboxViolations = append(result.SandboxViolations,
			"resource limit exceeded (OOM/timeout)")
	}
	// Process limit reached. Gated on a bound that is actually in force for
	// this execution, so the message can only describe a control that exists.
	if processLimitEnforcedFor(actualLevel, limits) && forkedPastLimit(result.Stderr+"\n"+result.Stdout) {
		result.SandboxViolations = append(result.SandboxViolations,
			"process limit reached (fork bomb attempt)")
	}
	// Network egress outside the declared allowlist (best-effort detection).
	if len(limits.Network.AllowedHosts) > 0 {
		for _, addr := range extractNetworkTargets(result.Stderr + result.Stdout) {
			if !hostAllowed(addr, limits.Network.AllowedHosts) {
				result.SandboxViolations = append(result.SandboxViolations,
					fmt.Sprintf("network access to non-allowed host: %s", addr))
			}
		}
	}
}

// processLimitSignatures are the messages a process-creation failure produces
// when RLIMIT_NPROC or a container pid limit is hit. Matched case-insensitively
// as whole substrings so a command that merely mentions the words cannot
// fabricate a violation about itself.
var processLimitSignatures = []string{
	"fork: retry:",
	"fork: resource temporarily unavailable",
	"resource temporarily unavailable: can't fork",
	"pthread_create: resource temporarily unavailable",
	"cannot allocate memory: fork",
	"can't fork",
	"unable to fork",
}

// forkedPastLimit reports whether the output carries a fork-failure signature.
func forkedPastLimit(text string) bool {
	lower := strings.ToLower(text)
	for _, sig := range processLimitSignatures {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// DefaultTimeoutSeconds is used when a limit set declares no usable time
// bound. It exists so that "no deadline configured" means "the documented
// default", never "expire immediately" and never "run forever".
const DefaultTimeoutSeconds = 300

// effectiveDeadline returns the wall-clock bound for one execution.
//
// Both declared time fields bind, and the tighter one wins: MaxSeconds is the
// in-sandbox CPU-time limit, MaxWallSeconds the supervisor bound. Using only
// MaxSeconds made `max_wall_seconds` a declared field with no consumer, which
// is the same class of defect as a gate that scans nothing. A limit set that
// declares neither falls back to the documented default rather than to a zero
// duration: `ResourceLimits{}` used to produce an already-expired context, so
// every command was killed before it started and the caller was handed
// ErrSandboxViolation — a security verdict about work that never ran.
func effectiveDeadline(limits ResourceLimits) time.Duration {
	secs := 0
	for _, v := range []int{limits.Time.MaxSeconds, limits.Time.MaxWallSeconds} {
		if v > 0 && (secs == 0 || v < secs) {
			secs = v
		}
	}
	if secs == 0 {
		secs = DefaultTimeoutSeconds
	}
	return time.Duration(secs) * time.Second
}

// processLimitEnforcedFor reports whether a process-count bound is actually in
// force for this execution.
//
// It is at L2 only, where the container runtime applies a cgroup pids limit
// scoped to the container. It is NOT at L0 (no isolation) or L1 (a namespace
// bounds visibility and identity, not how many tasks may exist).
//
// Detection is gated on this because reporting "fork bomb" for a control that
// was never applied is a false positive about a security boundary — the same
// class of defect as a gate that scans nothing and reports clean.
//
// RLIMIT_NPROC is deliberately not used as the mechanism. It is counted per
// REAL UID across the whole machine, not per process tree, so on a host whose
// account already runs more processes than the limit, every fork fails —
// including the ones the command needs in order to start. Measured here: a
// prlimit --nproc=128 wrapper made `sh -c` fail with "Cannot fork" because the
// account already had 837 processes. That is an outage shaped like a control.
func processLimitEnforcedFor(level SandboxLevel, limits ResourceLimits) bool {
	return level == LevelL2 && limits.Process.MaxProcesses > 0
}

// processBoundFlags returns the per-process bounds enforced through prlimit.
//
// RLIMIT_CPU is per-process and correct: it caps the CPU time one command may
// burn. Process COUNT is not here for the per-UID reason above; it is enforced
// by the container pid limit at L2.
func processBoundFlags(limits ResourceLimits) []string {
	var flags []string
	if limits.Time.MaxSeconds > 0 {
		flags = append(flags, fmt.Sprintf("--cpu=%d", limits.Time.MaxSeconds))
	}
	return flags
}

// limitLauncher returns the argv that runs command under the declared bounds.
//
// A shell builtin is NOT a viable mechanism. `sh` on Debian is dash, which
// rejects `ulimit -u` outright: a prefix of the form `ulimit -u 128 2>/dev/null;`
// wrote "Illegal option -u" to stderr, swallowed it with 2>/dev/null, and ran
// the command COMPLETELY UNBOUNDED. That is the exact defect class this pass
// removes — a configured control with no effect — and my own first attempt at
// the fix had it. The test that read `ulimit -u` back caught it.
//
// `prlimit(1)` sets RLIMIT_CPU on the process it execs. It needs no shell
// cooperation, so a shell script cannot sidestep it. When a bound is declared
// and no mechanism exists, this returns an error rather than running unbounded:
// silently dropping a declared security limit is the fail-open behaviour the
// rest of this file refuses.
func limitLauncher(limits ResourceLimits, command string) (string, []string, error) {
	flags := processBoundFlags(limits)
	if len(flags) == 0 {
		return "sh", []string{"-c", command}, nil
	}
	if _, err := exec.LookPath("prlimit"); err != nil {
		return "", nil, fmt.Errorf("sandbox declares a process/cpu bound but prlimit is not installed; refusing to run unbounded (declare max_processes and max_seconds as 0 to opt out explicitly)")
	}
	return "prlimit", append(flags, "--", "sh", "-c", command), nil
}

// containerHardeningArgs builds the fixed isolation flags for an L2 run.
//
// Pure and separated from execL2 so the declared posture can be asserted on a
// host with no container runtime: a test that shells out to podman proves a
// different thing on every machine, and on CI it proves nothing at all.
func containerHardeningArgs(limits ResourceLimits) []string {
	return []string{
		"run", "--rm",
		"--read-only", // Read-only root filesystem
		// Writable tmpfs for scratch, sized from the declared disk bound so the
		// writable surface is the number the policy names rather than a constant
		// the policy cannot influence. `/tmp` is where a shell actually writes;
		// a max_write_mb larger than the tmpfs only moves the failure, so the
		// tmpfs is the enforcement point for the write bound at L2.
		"--tmpfs", fmt.Sprintf("/tmp:rw,noexec,size=%dm", tmpfsSizeMB(limits, 64)),
		"--tmpfs", "/var/tmp:rw,noexec,size=32m",
		"--cap-drop=ALL",
		"--cap-add=SETUID", // Needed by some shells
		"--cap-add=SETGID",
		"--security-opt", "no-new-privileges",
	}
}

// containerResourceArgs builds the resource-limit flags for an L2 run.
//
// Every field it reads is a field the config declares. `swap_allowed: false`
// without `--memory-swap` is advisory: the container swaps and keeps running
// past the memory ceiling it was given. `max_processes` without
// `--pids-limit` leaves fork-bomb prevention to a detection heuristic that can
// only read output after the damage.
func containerResourceArgs(limits ResourceLimits) []string {
	var args []string
	if limits.Memory.MaxMB > 0 {
		args = append(args, fmt.Sprintf("--memory=%dm", limits.Memory.MaxMB))
		if !limits.Memory.SwapAllowed {
			args = append(args, fmt.Sprintf("--memory-swap=%dm", limits.Memory.MaxMB))
		}
	} else {
		args = append(args, "--memory=256m", "--memory-swap=256m") // Safe default, no swap
	}
	switch {
	case limits.CPU.MaxPercent > 0:
		args = append(args, fmt.Sprintf("--cpus=%.2f", limits.CPU.MaxPercent/100.0))
	case limits.CPU.MaxCores > 0:
		args = append(args, fmt.Sprintf("--cpus=%d", limits.CPU.MaxCores))
	default:
		args = append(args, "--cpus=1.0") // Safe default
	}
	if limits.Process.MaxProcesses > 0 {
		args = append(args, fmt.Sprintf("--pids-limit=%d", limits.Process.MaxProcesses))
	}
	return args
}

// tmpfsSizeMB returns the writable scratch size for an execution.
//
// A declared max_write_mb bounds how much the command may write; the tmpfs is
// where that write lands at L2. When the policy declares no disk bound the
// caller-supplied fallback is used, so "not configured" stays the documented
// default rather than zero (a zero-size tmpfs would make every write fail for
// a reason the policy never stated).
func tmpfsSizeMB(limits ResourceLimits, fallback int) int {
	if limits.Disk.MaxWriteMB > 0 {
		return limits.Disk.MaxWriteMB
	}
	return fallback
}

// validateMountPath rejects paths that would corrupt the container -v mount
// spec. The ':' character separates host:container:mode and ',' separates
// mount options, so either character in a path produces an ambiguous or
// exploitable mount.
func validateMountPath(path string) error {
	if strings.ContainsAny(path, ":,") {
		return fmt.Errorf("invalid bind-mount path %q: must not contain ':' or ','", path)
	}
	if path == "" {
		return fmt.Errorf("invalid bind-mount path: empty path")
	}
	return nil
}

// hostAllowed reports whether the given hostname matches the allowlist.
// Matching is suffix-based so subdomains of an allowed host are permitted.
func hostAllowed(host string, allowed []string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	for _, a := range allowed {
		a = strings.ToLower(a)
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

// extractNetworkTargets pulls candidate hostnames out of command/tool output
// (e.g. curl/wget/ssh target hosts) for best-effort network-violation checks.
func extractNetworkTargets(text string) []string {
	var hosts []string
	// URLs with explicit scheme.
	urlRe := regexp.MustCompile(`(?i)(?:https?://|ssh://|ftp://|//)([a-z0-9.-]+)`)
	// Tool-reported host resolution failures, e.g. curl's
	// "Could not resolve host: evil.example.com".
	resolveRe := regexp.MustCompile(`(?i)(?:could not resolve host|resolve host|to|connect to|connected to):\s*([a-z0-9.-]+)`)
	for _, mt := range urlRe.FindAllStringSubmatch(text, -1) {
		hosts = append(hosts, mt[1])
	}
	for _, mt := range resolveRe.FindAllStringSubmatch(text, -1) {
		hosts = append(hosts, mt[1])
	}
	return hosts
}

// ErrSandboxViolation is returned by Execute when a security boundary is
// breached (i.e. result.SandboxViolations is non-empty).
func SandboxViolationsError(violations []string) error {
	return fmt.Errorf("%w: %s", ErrSandboxViolation, strings.Join(violations, "; "))
}
