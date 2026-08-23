// Package guardrails provides runtime isolation and resource limiting for MCP tool executions.
package guardrails

import "fmt"

// SandboxLevel defines the isolation depth for a process execution.
type SandboxLevel string

const (
	// LevelL0 provides no isolation. Execution is direct.
	LevelL0 SandboxLevel = "L0"
	// LevelL1 provides namespace-based isolation (PID, Mount, Network, UTS).
	LevelL1 SandboxLevel = "L1"
	// LevelL2 provides container-based isolation (e.g., rootless podman/docker).
	LevelL2 SandboxLevel = "L2"
)

// ResourceLimits defines the hard constraints for the sandboxed process.
type ResourceLimits struct {
	CPU struct {
		MaxPercent float64 `yaml:"max_percent"` // Percentage of CPU available (0-100)
		MaxCores   int     `yaml:"max_cores"`   // Absolute max cores
	} `yaml:"cpu"`

	Memory struct {
		MaxMB       int  `yaml:"max_mb"`       // Maximum RAM in Megabytes
		SwapAllowed bool `yaml:"swap_allowed"` // Whether swap is permitted
	} `yaml:"memory"`

	Disk struct {
		MaxWriteMB     int      `yaml:"max_write_mb"`     // Maximum disk writes in MB
		ReadOnlyPaths  []string `yaml:"read_only_paths"`  // Paths that MUST be read-only
		ReadWritePaths []string `yaml:"read_write_paths"` // Paths that MAY be written
	} `yaml:"disk"`

	Network struct {
		Enabled      bool     `yaml:"enabled"`       // Whether network access is permitted
		AllowedHosts []string `yaml:"allowed_hosts"` // Domain whitelist
	} `yaml:"network"`

	Time struct {
		MaxSeconds     int `yaml:"max_seconds"`      // Process execution timeout
		MaxWallSeconds int `yaml:"max_wall_seconds"` // Maximum wall-clock time
	} `yaml:"time"`
}

// SandboxPolicy defines the default sandbox behavior for a specific tool type.
type SandboxPolicy struct {
	DefaultLevel   SandboxLevel   `yaml:"default_level"`
	ResourceLimits ResourceLimits `yaml:"resource_limits"`
}

// SandboxConfig holds the overall sandbox configuration for the MCP server.
type SandboxConfig struct {
	Enabled         bool                     `yaml:"enabled"`
	Image           string                   `yaml:"image"`            // Container image for L2 (default: alpine:latest)
	GlobalDefaults  ResourceLimits           `yaml:"global_defaults"`
	ToolPolicies    map[string]SandboxPolicy `yaml:"tool_policies"`
	FallbackEnabled bool                     `yaml:"fallback_enabled"` // Fallback L2 -> L1 -> L0
	NetworkMode     string                   `yaml:"network_mode"`     // Pre-provisioned container network for L2 (e.g. "agentnet")
}

// DefaultSandboxConfig returns a secure base configuration based on Spec 03.
func DefaultSandboxConfig() *SandboxConfig {
	return &SandboxConfig{
		Enabled:         true,
		FallbackEnabled: true,
		GlobalDefaults: ResourceLimits{
			CPU: struct {
				MaxPercent float64 `yaml:"max_percent"`
				MaxCores   int     `yaml:"max_cores"`
			}{MaxPercent: 50.0, MaxCores: 2},
			Memory: struct {
				MaxMB       int  `yaml:"max_mb"`
				SwapAllowed bool `yaml:"swap_allowed"`
			}{MaxMB: 1024, SwapAllowed: false},
			Disk: struct {
				MaxWriteMB     int      `yaml:"max_write_mb"`
				ReadOnlyPaths  []string `yaml:"read_only_paths"`
				ReadWritePaths []string `yaml:"read_write_paths"`
			}{
				MaxWriteMB:     500,
				ReadOnlyPaths:  []string{"/etc", "/usr", "/bin", "/sbin"},
				ReadWritePaths: []string{"/tmp", "/workspace"},
			},
			Network: struct {
				Enabled      bool     `yaml:"enabled"`
				AllowedHosts []string `yaml:"allowed_hosts"`
			}{Enabled: false},
			Time: struct {
				MaxSeconds     int `yaml:"max_seconds"`
				MaxWallSeconds int `yaml:"max_wall_seconds"`
			}{MaxSeconds: 300, MaxWallSeconds: 600},
		},
		ToolPolicies: map[string]SandboxPolicy{
			"bash": {
				DefaultLevel: LevelL1,
				ResourceLimits: ResourceLimits{
					Time: struct {
						MaxSeconds     int `yaml:"max_seconds"`
						MaxWallSeconds int `yaml:"max_wall_seconds"`
					}{MaxSeconds: 120, MaxWallSeconds: 300},
				},
			},
			"git": {
				DefaultLevel: LevelL1,
				ResourceLimits: ResourceLimits{
					Network: struct {
						Enabled      bool     `yaml:"enabled"`
						AllowedHosts []string `yaml:"allowed_hosts"`
					}{Enabled: true, AllowedHosts: []string{"github.com", "gitlab.com"}},
					Time: struct {
						MaxSeconds     int `yaml:"max_seconds"`
						MaxWallSeconds int `yaml:"max_wall_seconds"`
					}{MaxSeconds: 60, MaxWallSeconds: 120},
				},
			},
			"file_edit": {
				DefaultLevel: LevelL0,
				ResourceLimits: ResourceLimits{
					Time: struct {
						MaxSeconds     int `yaml:"max_seconds"`
						MaxWallSeconds int `yaml:"max_wall_seconds"`
					}{MaxSeconds: 30, MaxWallSeconds: 60},
				},
			},
		},
	}
}

// Validate checks the config for errors.
func (c *SandboxConfig) Validate() error {
	if c.GlobalDefaults.Time.MaxSeconds < 0 {
		return fmt.Errorf("sandbox max_seconds must be >= 0")
	}
	if c.GlobalDefaults.Time.MaxWallSeconds < 0 {
		return fmt.Errorf("sandbox max_wall_seconds must be >= 0")
	}
	if c.GlobalDefaults.CPU.MaxPercent < 0 || c.GlobalDefaults.CPU.MaxPercent > 100 {
		return fmt.Errorf("sandbox max_cpu_percent must be 0-100")
	}
	if c.GlobalDefaults.Memory.MaxMB < 0 {
		return fmt.Errorf("sandbox max_memory_mb must be >= 0")
	}
	// Validate tool policies
	for name, policy := range c.ToolPolicies {
		if policy.ResourceLimits.Time.MaxSeconds < 0 {
			return fmt.Errorf("tool policy %q: max_seconds must be >= 0", name)
		}
		if policy.ResourceLimits.CPU.MaxPercent < 0 || policy.ResourceLimits.CPU.MaxPercent > 100 {
			return fmt.Errorf("tool policy %q: max_cpu_percent must be 0-100", name)
		}
		if policy.ResourceLimits.Memory.MaxMB < 0 {
			return fmt.Errorf("tool policy %q: max_memory_mb must be >= 0", name)
		}
	}
	return nil
}
