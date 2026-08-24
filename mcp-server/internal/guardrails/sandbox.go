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
type SandboxManager struct {
	config *SandboxConfig
	logger *slog.Logger
}

// NewSandboxManager creates a new manager with the provided configuration.
func NewSandboxManager(cfg *SandboxConfig, logger *slog.Logger) *SandboxManager {
	if cfg == nil {
		cfg = DefaultSandboxConfig()
	}
	return &SandboxManager{
		config: cfg,
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
	start := time.Now()
	var result *SandboxResult
	var err error

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

		if err == nil || !m.config.FallbackEnabled {
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

		switch actualLevel {
		case LevelL2:
			m.logger.Warn("L2 container runtime unavailable, falling back to L1", "error", err)
			actualLevel = LevelL1
			continue
		case LevelL1:
			m.logger.Warn("L1 namespace setup unavailable, falling back to L0", "error", err)
			actualLevel = LevelL0
			continue
		}
		break
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

// execL0 executes the command directly on the host.
func (m *SandboxManager) execL0(ctx context.Context, command string, limits ResourceLimits) (*SandboxResult, error) {
	m.logger.Debug("executing L0 (no isolation)", "command", command)

	ctx, cancel := context.WithTimeout(ctx, time.Duration(limits.Time.MaxSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
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
		"sh", "-c", command,
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(limits.Time.MaxSeconds)*time.Second)
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
	return result, err
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

	args := []string{
		"run", "--rm",
		"--read-only",                        // Read-only root filesystem
		"--tmpfs", "/tmp:rw,noexec,size=64m", // Writable tmpfs for scratch
		"--tmpfs", "/var/tmp:rw,noexec,size=32m",
		"--cap-drop=ALL",
		"--cap-add=SETUID", // Needed by some shells
		"--cap-add=SETGID",
		"--security-opt", "no-new-privileges",
	}

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
	netMode := m.config.NetworkMode
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
	if limits.Memory.MaxMB > 0 {
		args = append(args, fmt.Sprintf("--memory=%dm", limits.Memory.MaxMB))
	} else {
		args = append(args, "--memory=256m") // Safe default
	}
	if limits.CPU.MaxPercent > 0 {
		args = append(args, fmt.Sprintf("--cpus=%.2f", limits.CPU.MaxPercent/100.0))
	} else if limits.CPU.MaxCores > 0 {
		args = append(args, fmt.Sprintf("--cpus=%d", limits.CPU.MaxCores))
	} else {
		args = append(args, "--cpus=1.0") // Safe default
	}

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
	image := m.config.Image
	if image == "" {
		image = "alpine:latest"
	}
	args = append(args, image, "sh", "-c", command)

	ctx, cancel := context.WithTimeout(ctx, time.Duration(limits.Time.MaxSeconds)*time.Second)
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
//   - network access attempt outside AllowedHosts (best-effort)
//
// The "ran at a lower isolation level than requested" breach is recorded by
// Execute against the requested level, not here (exec funcs don't know it).
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
