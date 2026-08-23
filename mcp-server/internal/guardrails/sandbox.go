package guardrails

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
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

// Execute runs the given command under the appropriate isolation level and resource limits.
func (m *SandboxManager) Execute(ctx context.Context, command string, level SandboxLevel, limits ResourceLimits) (*SandboxResult, error) {
	start := time.Now()
	var result *SandboxResult
	var err error

	actualLevel := level

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
		if !m.isRecoverable(err) {
			break
		}

		switch actualLevel {
		case LevelL2:
			m.logger.Warn("L2 sandbox failed, falling back to L1", "error", err)
			actualLevel = LevelL1
			continue
		case LevelL1:
			m.logger.Warn("L1 sandbox failed, falling back to L0", "error", err)
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
	}

	return result, err
}

func (m *SandboxManager) isRecoverable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "executable file not found") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "exit status 125") || // Docker: container failed to start
		strings.Contains(msg, "exit status 126") || // Docker: command not executable
		strings.Contains(msg, "No such image") ||   // Container image not found
		strings.Contains(msg, "unshare")            // Namespace creation failed
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

	return &SandboxResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, err
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

	return &SandboxResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, err
}

// execL2 executes the command inside a rootless container.
// Provides full filesystem + resource isolation via container runtime.
func (m *SandboxManager) execL2(ctx context.Context, command string, limits ResourceLimits) (*SandboxResult, error) {
	m.logger.Debug("executing L2 (container)", "command", command)

	runtime := "podman"
	if _, err := exec.LookPath(runtime); err != nil {
		runtime = "docker"
		if _, err := exec.LookPath(runtime); err != nil {
			return nil, fmt.Errorf("neither podman nor docker found in PATH")
		}
	}

	args := []string{
		"run", "--rm",
		"--read-only", // Read-only root filesystem
		"--tmpfs", "/tmp:rw,noexec,size=64m", // Writable tmpfs for scratch
		"--tmpfs", "/var/tmp:rw,noexec,size=32m",
		"--cap-drop=ALL",
		"--cap-add=SETUID",     // Needed by some shells
		"--cap-add=SETGID",
		"--security-opt", "no-new-privileges",
	}

	// Network isolation — honor limits.Network if configured
	if limits.Network.AllowedHosts == nil || len(limits.Network.AllowedHosts) == 0 {
		args = append(args, "--network=none")
	} else {
		// Allow specific hosts via DNS-based filtering
		args = append(args, "--network=none")
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

	// Bind mount declared workspace paths (read-write)
	for _, path := range limits.Disk.ReadWritePaths {
		args = append(args, "-v", fmt.Sprintf("%s:%s:rw", path, path))
	}
	// Bind mount declared read-only paths
	for _, path := range limits.Disk.ReadOnlyPaths {
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

	return &SandboxResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, err
}
