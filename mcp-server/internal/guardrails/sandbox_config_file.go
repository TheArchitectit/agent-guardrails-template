package guardrails

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spec 03 (guardrails-gaps-2026/03-runtime-sandbox-isolation) Sec. 5.2 item
// "Sandbox configuration hot-reload" and the residual note in Sec. 8 item 3.
//
// Before this file the per-tool sandbox policy existed only as a Go literal:
// `DefaultSandboxConfig()` was the only source of a SandboxConfig anywhere in
// the tree, so an operator who wanted `bash` at L2 with a 256 MB ceiling had no
// file to write and no environment variable to set. The documented YAML in
// Sec. 3.2/3.3 described a configuration path nothing read. That is the same
// defect class as the Spec 01 config file (`INJECTION_CONFIG_FILE`) one spec
// over, and it is worse here because the artifact is a *security boundary*: an
// operator who writes `network: {enabled: false}` and believes it took effect
// has a review trail asserting a control the process never applied.
//
// Two rules shape this loader:
//
//  1. Unknown keys are an ERROR (KnownFields). A typo in `max_wall_seconds`
//     must not leave the bound at its zero value and load clean.
//  2. Absence is distinguishable from an explicit false. Every boolean in the
//     YAML mirror is a pointer, so an omitted switch inherits the shipped
//     default while an explicit `false` wins. Without this, a file that names
//     only `max_mb` would silently turn `fallback_enabled` off — or, worse,
//     turn `network.enabled` on.
func LoadSandboxConfigFromYAML(path string) (*SandboxConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var root sandboxConfigFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&root); err != nil {
		// An empty file is not a configuration. Reporting the decoder's bare
		// "EOF" would send an operator looking for a YAML syntax error in a
		// file that is simply empty, so name the actual problem.
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("config %s is empty; it declares no sandbox section", path)
		}
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if root.Sandbox == nil {
		return nil, fmt.Errorf("config %s declares no sandbox section", path)
	}
	if root.SandboxV2 != nil {
		return nil, fmt.Errorf("config %s declares sandbox twice (`sandbox:` and `sandbox_config:`); use one", path)
	}
	section := root.Sandbox

	// The spec documents `sandbox_policies:` (Sec. 3.3); the Go type is
	// `tool_policies`. Both are accepted, but not at once: two policy tables in
	// one file has no correct merge order, and picking one would make the
	// effective policy depend on which key the parser saw last.
	if section.ToolPolicies != nil && section.SandboxPolicies != nil {
		return nil, fmt.Errorf("sandbox section declares both `tool_policies` and `sandbox_policies`; use one")
	}
	policies := section.ToolPolicies
	if policies == nil {
		policies = section.SandboxPolicies
	}

	cfg := DefaultSandboxConfig()

	if section.Enabled != nil {
		cfg.Enabled = *section.Enabled
	}
	if section.FallbackEnabled != nil {
		cfg.FallbackEnabled = *section.FallbackEnabled
	}
	if section.Image != nil {
		cfg.Image = *section.Image
	}
	if section.NetworkMode != nil {
		cfg.NetworkMode = *section.NetworkMode
	}
	if section.GlobalDefaults != nil {
		limits, lerr := section.GlobalDefaults.toLimits(cfg.GlobalDefaults)
		if lerr != nil {
			return nil, fmt.Errorf("global_defaults: %w", lerr)
		}
		cfg.GlobalDefaults = limits
	}
	if policies != nil {
		// A declared policy table REPLACES the shipped one rather than
		// merging into it. Merging would mean a policy the operator deleted
		// kept running, which is the opposite of what deleting it says.
		out := make(map[string]SandboxPolicy, len(policies))
		for name, p := range policies {
			if strings.TrimSpace(name) == "" {
				return nil, fmt.Errorf("sandbox_policies declares an empty tool type")
			}
			if p == nil {
				return nil, fmt.Errorf("sandbox policy %q is empty", name)
			}
			level := SandboxLevel(p.DefaultLevel)
			// An absent level is refused rather than defaulted: defaulting to
			// L0 would grant the least isolation to the policy that said
			// nothing, and defaulting to L1 would invent a decision the
			// operator did not make.
			if level == "" {
				return nil, fmt.Errorf("sandbox policy %q declares no default_level", name)
			}
			if !validLevel(level) {
				return nil, fmt.Errorf("sandbox policy %q: unknown default_level %q (must be L0, L1 or L2)", name, p.DefaultLevel)
			}
			base := mergeLimits(cfg.GlobalDefaults, ResourceLimits{})
			base.Network.Enabled = false
			limits, lerr := p.ResourceLimits.toLimits(base)
			if lerr != nil {
				return nil, fmt.Errorf("sandbox policy %q: %w", name, lerr)
			}
			// The policy-level path sets are the documented spelling (Sec. 3.3).
			// Declaring the same set in both places is refused rather than
			// merged: which one wins would depend on parse order.
			nested := p.ResourceLimits != nil && p.ResourceLimits.Disk != nil
			if p.ReadOnlyPaths != nil {
				if nested && p.ResourceLimits.Disk.ReadOnlyPaths != nil {
					return nil, fmt.Errorf("sandbox policy %q declares read_only_paths both at the policy level and under resource_limits.disk; use one", name)
				}
				limits.Disk.ReadOnlyPaths = append([]string(nil), p.ReadOnlyPaths...)
			}
			if p.ReadWritePaths != nil {
				if nested && p.ResourceLimits.Disk.ReadWritePaths != nil {
					return nil, fmt.Errorf("sandbox policy %q declares read_write_paths both at the policy level and under resource_limits.disk; use one", name)
				}
				limits.Disk.ReadWritePaths = append([]string(nil), p.ReadWritePaths...)
			}
			out[name] = SandboxPolicy{DefaultLevel: level, ResourceLimits: limits}
		}
		cfg.ToolPolicies = out
	}

	// Mount paths are argv fragments for `-v host:container:mode`, so a path
	// carrying ':' or ',' builds a different mount than the one written. This
	// is checked at load rather than at first execution: a malformed path that
	// only fails when a command runs is a config error discovered in
	// production.
	if err := validateLimitPaths(cfg.GlobalDefaults); err != nil {
		return nil, fmt.Errorf("global_defaults: %w", err)
	}
	for _, name := range sortedPolicyNames(cfg.ToolPolicies) {
		if err := validateLimitPaths(cfg.ToolPolicies[name].ResourceLimits); err != nil {
			return nil, fmt.Errorf("sandbox policy %q: %w", name, err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sandbox config: %w", err)
	}
	return cfg, nil
}

func sortedPolicyNames(policies map[string]SandboxPolicy) []string {
	names := make([]string, 0, len(policies))
	for name := range policies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// validateLimitPaths refuses a bind-mount path that cannot be rendered.
func validateLimitPaths(l ResourceLimits) error {
	for _, p := range l.Disk.ReadOnlyPaths {
		if err := validateMountPath(p); err != nil {
			return fmt.Errorf("read_only_paths: %w", err)
		}
	}
	for _, p := range l.Disk.ReadWritePaths {
		if err := validateMountPath(p); err != nil {
			return fmt.Errorf("read_write_paths: %w", err)
		}
	}
	return nil
}

// --- YAML mirror types -------------------------------------------------------
//
// These exist so every boolean can distinguish "absent" from "explicitly
// false" and every numeric field can distinguish "absent" from "explicitly 0"
// (0 is legal: it means "no bound"). Decoding straight into ResourceLimits
// cannot make either distinction.

type sandboxConfigFile struct {
	Sandbox   *sandboxSectionYAML `yaml:"sandbox"`
	SandboxV2 *sandboxSectionYAML `yaml:"sandbox_config"`
}

type sandboxSectionYAML struct {
	Enabled         *bool                         `yaml:"enabled"`
	Image           *string                       `yaml:"image"`
	FallbackEnabled *bool                         `yaml:"fallback_enabled"`
	NetworkMode     *string                       `yaml:"network_mode"`
	GlobalDefaults  *sandboxLimitsYAML            `yaml:"global_defaults"`
	ToolPolicies    map[string]*sandboxPolicyYAML `yaml:"tool_policies"`
	SandboxPolicies map[string]*sandboxPolicyYAML `yaml:"sandbox_policies"`
}

type sandboxPolicyYAML struct {
	DefaultLevel   string             `yaml:"default_level"`
	ResourceLimits *sandboxLimitsYAML `yaml:"resource_limits"`

	// Spec 03 Sec. 3.3 puts the path sets at the POLICY level, beside
	// `resource_limits`, not inside its `disk:` block:
	//
	//	sandbox_policies:
	//	  bash:
	//	    resource_limits: { ... }
	//	    read_write_paths: ["/tmp"]
	//
	// Supporting only the nested spelling would have refused the documented
	// shape with an unknown-field error — the same "the spec's own example
	// does not load" defect as the Spec 01 config seam. Both spellings are
	// accepted; declaring one switch in both is refused (below), because two
	// spellings of one path set has no correct merge order.
	ReadOnlyPaths  []string `yaml:"read_only_paths"`
	ReadWritePaths []string `yaml:"read_write_paths"`
}

type sandboxLimitsYAML struct {
	CPU *struct {
		MaxPercent *float64 `yaml:"max_percent"`
		MaxCores   *int     `yaml:"max_cores"`
	} `yaml:"cpu"`
	Memory *struct {
		MaxMB       *int  `yaml:"max_mb"`
		SwapAllowed *bool `yaml:"swap_allowed"`
	} `yaml:"memory"`
	Disk *struct {
		MaxWriteMB     *int     `yaml:"max_write_mb"`
		ReadOnlyPaths  []string `yaml:"read_only_paths"`
		ReadWritePaths []string `yaml:"read_write_paths"`
	} `yaml:"disk"`
	Process *struct {
		MaxProcesses *int `yaml:"max_processes"`
	} `yaml:"process"`
	Network *struct {
		Enabled      *bool    `yaml:"enabled"`
		AllowedHosts []string `yaml:"allowed_hosts"`
	} `yaml:"network"`
	Time *struct {
		MaxSeconds     *int `yaml:"max_seconds"`
		MaxWallSeconds *int `yaml:"max_wall_seconds"`
	} `yaml:"time"`
}

// toLimits overlays the declared fields on base. A field that is absent keeps
// base's value; a field that is present — including an explicit zero or false
// — wins.
func (s *sandboxLimitsYAML) toLimits(base ResourceLimits) (ResourceLimits, error) {
	if s == nil {
		return base, nil
	}
	out := cloneLimits(base)

	if s.CPU != nil {
		if s.CPU.MaxPercent != nil {
			out.CPU.MaxPercent = *s.CPU.MaxPercent
		}
		if s.CPU.MaxCores != nil {
			out.CPU.MaxCores = *s.CPU.MaxCores
		}
	}
	if s.Memory != nil {
		if s.Memory.MaxMB != nil {
			out.Memory.MaxMB = *s.Memory.MaxMB
		}
		if s.Memory.SwapAllowed != nil {
			out.Memory.SwapAllowed = *s.Memory.SwapAllowed
		}
	}
	if s.Disk != nil {
		if s.Disk.MaxWriteMB != nil {
			out.Disk.MaxWriteMB = *s.Disk.MaxWriteMB
		}
		if s.Disk.ReadOnlyPaths != nil {
			out.Disk.ReadOnlyPaths = append([]string(nil), s.Disk.ReadOnlyPaths...)
		}
		if s.Disk.ReadWritePaths != nil {
			out.Disk.ReadWritePaths = append([]string(nil), s.Disk.ReadWritePaths...)
		}
	}
	if s.Process != nil && s.Process.MaxProcesses != nil {
		out.Process.MaxProcesses = *s.Process.MaxProcesses
	}
	if s.Network != nil {
		if s.Network.Enabled != nil {
			out.Network.Enabled = *s.Network.Enabled
		}
		if s.Network.AllowedHosts != nil {
			out.Network.AllowedHosts = append([]string(nil), s.Network.AllowedHosts...)
		}
	}
	if s.Time != nil {
		if s.Time.MaxSeconds != nil {
			out.Time.MaxSeconds = *s.Time.MaxSeconds
		}
		if s.Time.MaxWallSeconds != nil {
			out.Time.MaxWallSeconds = *s.Time.MaxWallSeconds
		}
	}
	return out, nil
}
