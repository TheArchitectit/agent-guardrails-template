// Package guardrails provides runtime isolation and resource limiting for MCP tool executions.
package guardrails

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
	GlobalDefaults  ResourceLimits           `yaml:"global_defaults"`
	ToolPolicies    map[string]SandboxPolicy `yaml:"tool_policies"`
	FallbackEnabled bool                     `yaml:"fallback_enabled"` // Fallback L2 -> L1 -> L0
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
				DefaultLevel:   LevelL1,
				ResourceLimits: ResourceLimits{},
			},
			"git": {
				DefaultLevel: LevelL1,
				ResourceLimits: ResourceLimits{
					Network: struct {
						Enabled      bool     `yaml:"enabled"`
						AllowedHosts []string `yaml:"allowed_hosts"`
					}{Enabled: true, AllowedHosts: []string{"github.com", "gitlab.com"}},
				},
			},
			"file_edit": {
				DefaultLevel:   LevelL0,
				ResourceLimits: ResourceLimits{},
			},
		},
	}
}
