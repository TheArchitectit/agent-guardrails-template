// Package guardrails provides runtime isolation and resource limiting for MCP tool executions.
package guardrails

import (
	"fmt"
	"strings"
)

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

	Process struct {
		MaxProcesses int `yaml:"max_processes"` // Maximum concurrent processes (0 = no bound)
	} `yaml:"process"`

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
	Image           string                   `yaml:"image"` // Container image for L2 (default: alpine:latest)
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
			Process: struct {
				MaxProcesses int `yaml:"max_processes"`
			}{MaxProcesses: 256},
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

// Clone returns a deep copy of the configuration.
//
// The maps and slices are copied because SetToolPolicy validates a candidate
// built from a clone: a shallow copy would let a refused policy mutate the
// live table through the shared backing map.
func (c *SandboxConfig) Clone() *SandboxConfig {
	if c == nil {
		return nil
	}
	out := *c
	out.GlobalDefaults = cloneLimits(c.GlobalDefaults)
	if c.ToolPolicies != nil {
		out.ToolPolicies = make(map[string]SandboxPolicy, len(c.ToolPolicies))
		for name, policy := range c.ToolPolicies {
			policy.ResourceLimits = cloneLimits(policy.ResourceLimits)
			out.ToolPolicies[name] = policy
		}
	}
	return &out
}

func cloneLimits(in ResourceLimits) ResourceLimits {
	out := in
	out.Disk.ReadOnlyPaths = append([]string(nil), in.Disk.ReadOnlyPaths...)
	out.Disk.ReadWritePaths = append([]string(nil), in.Disk.ReadWritePaths...)
	out.Network.AllowedHosts = append([]string(nil), in.Network.AllowedHosts...)
	return out
}

// validateImageRef refuses an image value that cannot be passed as a single
// argv element to the container runtime.
//
// The previous check was `strings.ContainsAny(image, " \t\n:;")`, which
// refused EVERY valid OCI reference — `alpine:latest`, `alpine:3.19`,
// `ghcr.io/org/img:v1` and `alpine@sha256:…` all failed — while leaving the
// shipped default (`""`) as the only value that validated. That was invisible
// for as long as no configuration file could set the field; the moment an
// operator-supplied file can name an image, a correct value would have stopped
// startup. The `:` in a registry port and in a tag is part of the reference,
// not a second argument.
//
// What is refused is what would actually change the command line: whitespace
// (splits one argv element in every runtime that reparses), a leading `-`
// (would be read as a runtime flag), and shell metacharacters that make an
// injected image string look like more than an image.
func validateImageRef(image string) error {
	if strings.TrimSpace(image) != image {
		return fmt.Errorf("sandbox image %q has leading or trailing whitespace", image)
	}
	if strings.ContainsAny(image, " \t\n\r") {
		return fmt.Errorf("sandbox image %q must not contain whitespace", image)
	}
	if strings.HasPrefix(image, "-") {
		return fmt.Errorf("sandbox image %q must not begin with '-' (it would be read as a runtime flag)", image)
	}
	if strings.ContainsAny(image, ";&|`$()<>") {
		return fmt.Errorf("sandbox image %q must not contain shell metacharacters", image)
	}
	return nil
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
	if c.GlobalDefaults.Process.MaxProcesses < 0 {
		return fmt.Errorf("sandbox max_processes must be >= 0")
	}
	if c.GlobalDefaults.Disk.MaxWriteMB < 0 {
		return fmt.Errorf("sandbox max_write_mb must be >= 0")
	}
	if c.Image != "" {
		if err := validateImageRef(c.Image); err != nil {
			return err
		}
	}
	// Validate tool policies
	for name, policy := range c.ToolPolicies {
		// A policy that names no level or an unknown one is a policy that
		// cannot be applied. Before this check existed, `L9` validated clean
		// and resolved to L0 at use time — a typo in a security control that
		// failed open instead of failing at load.
		if policy.DefaultLevel == "" {
			return fmt.Errorf("tool policy %q: default_level is required", name)
		}
		if !validLevel(policy.DefaultLevel) {
			return fmt.Errorf("tool policy %q: unknown default_level %q (must be L0, L1 or L2)", name, policy.DefaultLevel)
		}
		if policy.ResourceLimits.Time.MaxSeconds < 0 {
			return fmt.Errorf("tool policy %q: max_seconds must be >= 0", name)
		}
		if policy.ResourceLimits.CPU.MaxPercent < 0 || policy.ResourceLimits.CPU.MaxPercent > 100 {
			return fmt.Errorf("tool policy %q: max_cpu_percent must be 0-100", name)
		}
		if policy.ResourceLimits.Memory.MaxMB < 0 {
			return fmt.Errorf("tool policy %q: max_memory_mb must be >= 0", name)
		}
		if policy.ResourceLimits.Process.MaxProcesses < 0 {
			return fmt.Errorf("tool policy %q: max_processes must be >= 0", name)
		}
		if policy.ResourceLimits.Disk.MaxWriteMB < 0 {
			return fmt.Errorf("tool policy %q: max_write_mb must be >= 0", name)
		}
	}
	return nil
}
