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
		strings.Contains(msg, "permission denied")
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
func (m *SandboxManager) execL1(ctx context.Context, command string, limits ResourceLimits) (*SandboxResult, error) {
	m.logger.Debug("executing L1 (namespaces)", "command", command)

	args := []string{
		"--pid",
		"--mount",
		"--net",
		"--uts",
		"--fork",
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
		"--network=none",
		fmt.Sprintf("--memory=%dm", limits.Memory.MaxMB),
		fmt.Sprintf("--cpus=%.1f", limits.CPU.MaxPercent/100.0),
		"--read-only",
	}

	for _, path := range limits.Disk.ReadWritePaths {
		args = append(args, "-v", fmt.Sprintf("%s:%s:rw", path, path))
	}

	args = append(args, "alpine:latest", "sh", "-c", command)

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
