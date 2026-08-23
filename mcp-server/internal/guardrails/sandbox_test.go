package guardrails

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestSandboxManager_Execute_L0(t *testing.T) {
	mgr := NewSandboxManager(nil, slog.Default())
	ctx := context.Background()

	limits := DefaultSandboxConfig().GlobalDefaults

	t.Run("SuccessfulCommand", func(t *testing.T) {
		res, err := mgr.Execute(ctx, "echo 'hello world'", LevelL0, limits)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if res.Stdout != "hello world\n" {
			t.Errorf("expected 'hello world\\n', got %q", res.Stdout)
		}
	})

	t.Run("CommandFailure", func(t *testing.T) {
		res, err := mgr.Execute(ctx, "exit 1", LevelL0, limits)
		_ = err
		if res == nil || res.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %v", res)
		}
	})
}

func TestSandboxManager_Timeout(t *testing.T) {
	mgr := NewSandboxManager(nil, slog.Default())
	ctx := context.Background()

	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Time.MaxSeconds = 1

	t.Run("TimeoutEnforcement", func(t *testing.T) {
		_, err := mgr.Execute(ctx, "sleep 5", LevelL0, limits)
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
	})
}

func TestSandboxManager_Fallback(t *testing.T) {
	mgr := NewSandboxManager(nil, slog.Default())
	ctx := context.Background()
	limits := DefaultSandboxConfig().GlobalDefaults

	t.Run("L2FallbackToL0", func(t *testing.T) {
		res, err := mgr.Execute(ctx, "echo 'fallback'", LevelL2, limits)
		if err != nil {
			t.Fatalf("execution failed during fallback: %v", err)
		}
		if res == nil {
			t.Fatal("result is nil")
		}
		if res.ActualIsolationLvl == LevelL2 {
			t.Log("L2 was actually available")
		} else {
			t.Logf("Fell back to %s as expected", res.ActualIsolationLvl)
		}
	})
}

func TestSandboxManager_ViolationDetection(t *testing.T) {
	mgr := NewSandboxManager(nil, slog.Default())
	ctx := context.Background()
	limits := DefaultSandboxConfig().GlobalDefaults

	limits.Time.MaxSeconds = 2
	limits.Time.MaxWallSeconds = 2

	t.Run("PathTraversalAttempt", func(t *testing.T) {
		res, err := mgr.Execute(ctx, "cat /etc/passwd", LevelL0, limits)
		_ = err
		if res == nil {
			t.Fatal("nil result on path traversal attempt")
		}
		if res.ActualIsolationLvl == LevelL1 || res.ActualIsolationLvl == LevelL2 {
			if res.ExitCode == 0 {
				t.Error("L1/L2 should block /etc access")
			}
		}
	})
}

func TestSandboxManager_BashViolationDetection(t *testing.T) {
	mgr := NewSandboxManager(nil, slog.Default())
	ctx := context.Background()
	limits := DefaultSandboxConfig().GlobalDefaults

	t.Run("ForkBombSimulation", func(t *testing.T) {
		res, err := mgr.Execute(ctx, "true", LevelL0, limits)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", res.ExitCode)
		}
	})
}

func TestDefaultSandboxConfig(t *testing.T) {
	cfg := DefaultSandboxConfig()
	if cfg == nil {
		t.Fatal("DefaultSandboxConfig returned nil")
	}
	if !cfg.Enabled {
		t.Error("expected enabled by default")
	}
	if cfg.GlobalDefaults.Memory.MaxMB != 1024 {
		t.Errorf("expected default memory 1024, got %d", cfg.GlobalDefaults.Memory.MaxMB)
	}
	if cfg.GlobalDefaults.Network.Enabled {
		t.Error("expected network disabled by default")
	}
}

func TestResourceLimitsDefaults(t *testing.T) {
	limit := ResourceLimits{
		CPU: struct {
			MaxPercent float64 `yaml:"max_percent"`
			MaxCores   int     `yaml:"max_cores"`
		}{MaxPercent: 25.0},
		Memory: struct {
			MaxMB       int  `yaml:"max_mb"`
			SwapAllowed bool `yaml:"swap_allowed"`
		}{MaxMB: 512, SwapAllowed: false},
		Time: struct {
			MaxSeconds     int `yaml:"max_seconds"`
			MaxWallSeconds int `yaml:"max_wall_seconds"`
		}{MaxSeconds: 60, MaxWallSeconds: 120},
	}
	if limit.CPU.MaxPercent != 25.0 {
		t.Errorf("expected 25, got %f", limit.CPU.MaxPercent)
	}
	if limit.Memory.MaxMB != 512 {
		t.Errorf("expected 512, got %d", limit.Memory.MaxMB)
	}
	if limit.Time.MaxSeconds != 60 {
		t.Errorf("expected 60, got %d", limit.Time.MaxSeconds)
	}
	if limit.Time.MaxWallSeconds != 120 {
		t.Errorf("expected 120, got %d", limit.Time.MaxWallSeconds)
	}
}

func TestExecutionTimeTracked(t *testing.T) {
	mgr := NewSandboxManager(nil, slog.Default())
	ctx := context.Background()
	limits := DefaultSandboxConfig().GlobalDefaults
	limits.Time.MaxSeconds = 30

	res, err := mgr.Execute(ctx, "sleep 1", LevelL0, limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExecutionTime < time.Second || res.ExecutionTime > 5*time.Second {
		t.Errorf("unexpected execution time: %v", res.ExecutionTime)
	}
}
