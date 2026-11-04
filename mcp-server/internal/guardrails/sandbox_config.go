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

	// Secrets is the fourth capability-matrix dimension named by
	// sandbox-isolation-levels ("filesystem, network, process, secrets, and
	// state persistence"). Until 2026-11-03 the matrix had no secrets cell at
	// all: ResourceLimits carried filesystem (Disk), network and process, and
	// the closure table nonetheless marked the requirement EXERCISED.
	Secrets struct {
		// AllowedEnv is the COMPLETE list of environment variable names the
		// workload may observe. Everything else the parent holds — every
		// credential-shaped variable — is dropped before exec by
		// scrubEnvironment. An empty list means the workload gets no
		// environment at all; PATH is granted explicitly, never implicitly.
		AllowedEnv []string `yaml:"allowed_env"`
		// AllowCredentialFiles permits reads of mounted credential files.
		// false — the safe reading of an absent value, matching
		// Network.Enabled — denies them even when a mount would allow the
		// read.
		AllowCredentialFiles bool `yaml:"allow_credential_files"`
		// MaxSecrets bounds how many secret VALUES may be injected into the
		// workload (0 = none may be injected). Explicit, never inherited:
		// absence means "no secrets", which is the safe reading.
		MaxSecrets int `yaml:"max_secrets"`
	} `yaml:"secrets"`

	// StatePersistence is the fifth capability-matrix dimension named by
	// sandbox-isolation-levels. It answers a question the Disk paths cannot:
	// not "where may the workload write" but "may anything it writes outlive
	// the run".
	StatePersistence struct {
		// Allowed declares whether any state may outlive the workload run.
		// false — the safe reading of an absent value — means the run is
		// ephemeral: writes go to bounded scratch that dies with the process.
		Allowed bool `yaml:"allowed"`
		// MaxPersistedMB bounds the bytes of state that may outlive the run
		// (0 = none, even when Allowed is true). Only meaningful when Allowed
		// is true; follows the MaxWriteMB inheritance rule (>0 overrides).
		MaxPersistedMB int `yaml:"max_persisted_mb"`
		// PersistPaths names the only directories where state may outlive the
		// run. Empty means none, even when Allowed is true.
		PersistPaths []string `yaml:"persist_paths"`
	} `yaml:"state_persistence"`
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
			Secrets: struct {
				AllowedEnv           []string `yaml:"allowed_env"`
				AllowCredentialFiles bool     `yaml:"allow_credential_files"`
				MaxSecrets           int      `yaml:"max_secrets"`
			}{
				// A minimal NON-secret baseline. PATH is granted explicitly
				// (never implicitly) so a workload can find binaries; HOME and
				// LANG are the usual harmless pair the escape fixtures already
				// treat as declared. Every other name — AWS_*, *_API_KEY,
				// OPENAI_API_KEY, the lot — is stripped by scrubEnvironment.
				AllowedEnv:           []string{"PATH", "HOME", "LANG"},
				AllowCredentialFiles: false,
				MaxSecrets:           0,
			},
			StatePersistence: struct {
				Allowed        bool     `yaml:"allowed"`
				MaxPersistedMB int      `yaml:"max_persisted_mb"`
				PersistPaths   []string `yaml:"persist_paths"`
			}{
				// Absent-by-default posture: a run leaves nothing behind
				// unless a policy says otherwise. This is the reading that
				// matches Network.Enabled's rule (false is the safe reading of
				// a missing value).
				Allowed:        false,
				MaxPersistedMB: 0,
				PersistPaths:   nil,
			},
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
	out.Secrets.AllowedEnv = append([]string(nil), in.Secrets.AllowedEnv...)
	out.StatePersistence.PersistPaths = append([]string(nil), in.StatePersistence.PersistPaths...)
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
	if c.GlobalDefaults.Secrets.MaxSecrets < 0 {
		return fmt.Errorf("sandbox max_secrets must be >= 0")
	}
	if c.GlobalDefaults.StatePersistence.MaxPersistedMB < 0 {
		return fmt.Errorf("sandbox max_persisted_mb must be >= 0")
	}
	if err := validateSecretEnvNames(c.GlobalDefaults.Secrets.AllowedEnv); err != nil {
		return fmt.Errorf("sandbox allowed_env: %w", err)
	}
	for _, p := range c.GlobalDefaults.StatePersistence.PersistPaths {
		if verr := validateMountPath(p); verr != nil {
			return fmt.Errorf("sandbox persist_paths: %w", verr)
		}
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
		if policy.ResourceLimits.Secrets.MaxSecrets < 0 {
			return fmt.Errorf("tool policy %q: max_secrets must be >= 0", name)
		}
		if policy.ResourceLimits.StatePersistence.MaxPersistedMB < 0 {
			return fmt.Errorf("tool policy %q: max_persisted_mb must be >= 0", name)
		}
		if err := validateSecretEnvNames(policy.ResourceLimits.Secrets.AllowedEnv); err != nil {
			return fmt.Errorf("tool policy %q: allowed_env: %w", name, err)
		}
		for _, p := range policy.ResourceLimits.StatePersistence.PersistPaths {
			if verr := validateMountPath(p); verr != nil {
				return fmt.Errorf("tool policy %q: persist_paths: %w", name, verr)
			}
		}
	}
	return nil
}

// validateSecretEnvNames refuses an allowed_env entry that cannot name an
// environment variable. An empty or `=`-bearing entry would silently match
// nothing (or worse, everything the parser mis-splits), so the allowlist would
// look declared while denying or granting the wrong names.
func validateSecretEnvNames(names []string) error {
	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			return fmt.Errorf("entry must not be empty")
		}
		if strings.ContainsAny(n, "=\x00") {
			return fmt.Errorf("entry %q must not contain '=' or NUL", n)
		}
		if strings.TrimSpace(n) != n {
			return fmt.Errorf("entry %q has leading or trailing whitespace", n)
		}
	}
	return nil
}
