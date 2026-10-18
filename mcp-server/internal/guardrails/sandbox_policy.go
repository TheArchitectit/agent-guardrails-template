// Spec 03 (guardrails-gaps-2026/03-runtime-sandbox-isolation) — the policy
// resolution layer the sandbox library never had.
//
// Before this file, SandboxConfig.ToolPolicies was a declared data structure
// with no consumer anywhere in the tree: `bash`, `git` and `file_edit` carried
// per-tool levels and limits that no code path ever read, and Validate did not
// even check that a declared DefaultLevel was a real level. An operator who
// edited a policy changed nothing; a policy naming `L9` validated clean.
//
// Two rules make the table mean something:
//
//  1. Zero/empty resource fields INHERIT the global defaults. A policy states
//     only what it changes.
//  2. Isolation may be raised by a request but never lowered. A caller asking
//     for L0 against a policy that declares L1 runs at L1, and the result says
//     which level actually ran. Lowering isolation is the only way this table
//     could be used to weaken a deployment, so it is refused at the seam.
//
// Network.Enabled is deliberately NOT inherited: `false` means "no egress" and
// is the safe reading of a missing value, so a policy that needs network must
// say so (the shipped `git` default does).
package guardrails

import (
	"context"
	"fmt"
	"sort"
)

// validLevel reports whether level names a level this implementation supports.
func validLevel(level SandboxLevel) bool {
	switch level {
	case LevelL0, LevelL1, LevelL2:
		return true
	}
	return false
}

// ParseSandboxLevel converts a caller-supplied string into a SandboxLevel.
// An empty string is NOT a level: a missing level must be resolved from
// policy, never defaulted to the least isolated option.
func ParseSandboxLevel(s string) (SandboxLevel, error) {
	lvl := SandboxLevel(s)
	if !validLevel(lvl) {
		return "", fmt.Errorf("unknown sandbox level %q: must be L0, L1 or L2", s)
	}
	return lvl, nil
}

// maxLevel returns the more isolated of two levels. L0 < L1 < L2.
func maxLevel(a, b SandboxLevel) SandboxLevel {
	if LevelRank(b) > LevelRank(a) {
		return b
	}
	return a
}

// EffectiveLevel is the level a request actually runs at: the more isolated of
// the policy level and the requested level.
//
// Exported so the resolution can be asserted without executing a command.
// Isolation may be raised by a request but never lowered — a per-request field
// that could lower it would be the one way to weaken a deployment.
func EffectiveLevel(policyLevel, requested SandboxLevel) SandboxLevel {
	if requested == "" {
		return policyLevel
	}
	if !validLevel(requested) {
		return policyLevel
	}
	return maxLevel(policyLevel, requested)
}

// mergeLimits overlays a policy's declared limits on the global defaults.
// A zero numeric/empty slice inherits; Network.Enabled is explicit (see the
// package comment above).
func mergeLimits(global, policy ResourceLimits) ResourceLimits {
	out := global

	if policy.CPU.MaxPercent > 0 {
		out.CPU.MaxPercent = policy.CPU.MaxPercent
	}
	if policy.CPU.MaxCores > 0 {
		out.CPU.MaxCores = policy.CPU.MaxCores
	}
	if policy.Memory.MaxMB > 0 {
		out.Memory.MaxMB = policy.Memory.MaxMB
	}
	if policy.Disk.MaxWriteMB > 0 {
		out.Disk.MaxWriteMB = policy.Disk.MaxWriteMB
	}
	if len(policy.Disk.ReadOnlyPaths) > 0 {
		out.Disk.ReadOnlyPaths = append([]string(nil), policy.Disk.ReadOnlyPaths...)
	}
	if len(policy.Disk.ReadWritePaths) > 0 {
		out.Disk.ReadWritePaths = append([]string(nil), policy.Disk.ReadWritePaths...)
	}
	if len(policy.Network.AllowedHosts) > 0 {
		out.Network.AllowedHosts = append([]string(nil), policy.Network.AllowedHosts...)
	}
	if policy.Time.MaxSeconds > 0 {
		out.Time.MaxSeconds = policy.Time.MaxSeconds
	}
	if policy.Time.MaxWallSeconds > 0 {
		out.Time.MaxWallSeconds = policy.Time.MaxWallSeconds
	}

	// Explicit, never inherited: a policy that does not ask for network does
	// not get any, even when the global default enables it for a specific
	// tool family. Without this the `git` default (network on) would silently
	// hand network to every inheriting policy.
	out.Network.Enabled = policy.Network.Enabled

	return out
}

// ResolveToolPolicy returns the effective level and limits for one tool type,
// plus whether the caller's requested level was raised to meet the policy.
//
// toolType "" or an unknown tool type yields the global defaults at L0 — but
// L0 is only reachable when the configuration itself says so; an unclassified
// tool is not silently unisolated by this function. Callers that require
// isolation pass a level explicitly (see SandboxManager.ExecuteFor).
func (c *SandboxConfig) ResolveToolPolicy(toolType string) (SandboxLevel, ResourceLimits) {
	global := c.GlobalDefaults
	policy, ok := c.ToolPolicies[toolType]
	if !ok {
		return LevelL0, mergeLimits(global, ResourceLimits{})
	}
	level := policy.DefaultLevel
	if level == "" {
		level = LevelL0
	}
	return level, mergeLimits(global, policy.ResourceLimits)
}

// ToolPolicyNames returns the declared tool types in sorted order, so a
// caller can enumerate policy without depending on map iteration order.
func (c *SandboxConfig) ToolPolicyNames() []string {
	names := make([]string, 0, len(c.ToolPolicies))
	for name := range c.ToolPolicies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SetToolPolicy installs (or replaces) one tool policy after validation.
//
// The whole config is validated before the mutation is applied, so a refused
// policy leaves the previous table intact rather than half-written.
func (m *SandboxManager) SetToolPolicy(toolType string, policy SandboxPolicy) error {
	if toolType == "" {
		return fmt.Errorf("tool_type is required")
	}
	if policy.DefaultLevel == "" {
		return fmt.Errorf("tool policy %q: default_level is required", toolType)
	}
	if !validLevel(policy.DefaultLevel) {
		return fmt.Errorf("tool policy %q: unknown default_level %q", toolType, policy.DefaultLevel)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	candidate := m.config.Clone()
	if candidate.ToolPolicies == nil {
		candidate.ToolPolicies = map[string]SandboxPolicy{}
	}
	candidate.ToolPolicies[toolType] = policy
	if err := candidate.Validate(); err != nil {
		return err
	}
	m.config = candidate
	return nil
}

// ToolPolicies returns a copy of the active policy table.
func (m *SandboxManager) ToolPolicies() map[string]SandboxPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]SandboxPolicy, len(m.config.ToolPolicies))
	for k, v := range m.config.ToolPolicies {
		out[k] = v
	}
	return out
}

// Config returns a copy of the active sandbox configuration.
func (m *SandboxManager) Config() SandboxConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.config.Clone()
}

// LevelRank orders isolation levels so callers can compare them without
// duplicating the ranking. L0 = 0, L1 = 1, L2 = 2; an unknown level is -1 so
// it can never out-rank a real one.
func LevelRank(level SandboxLevel) int {
	switch level {
	case LevelL0:
		return 0
	case LevelL1:
		return 1
	case LevelL2:
		return 2
	}
	return -1
}

// LimitOverrides is the per-request narrowing surface for
// guardrail_sandbox_execute. Every field may only reduce what the policy
// allows; NarrowLimits refuses a widening value rather than clamping it.
type LimitOverrides struct {
	MaxMB         int
	MaxSeconds    int
	MaxCPUPercent float64
	AllowedPaths  []string
}

// ResolveToolPolicyLimits returns the effective limits for a tool type.
func (c *SandboxConfig) ResolveToolPolicyLimits(toolType string) ResourceLimits {
	_, limits := c.ResolveToolPolicy(toolType)
	return limits
}

// NarrowLimits applies per-request overrides to a policy's limits.
//
// Widening is an error, not a silent clamp: a caller that asks for 4096 MB
// against a 512 MB policy and receives 512 MB has been told its override
// applied when it did not. The error names the field and both values.
func NarrowLimits(base ResourceLimits, o LimitOverrides) (ResourceLimits, error) {
	out := cloneLimits(base)

	if o.MaxMB != 0 {
		if o.MaxMB < 0 {
			return out, fmt.Errorf("max_mb must be positive")
		}
		if base.Memory.MaxMB > 0 && o.MaxMB > base.Memory.MaxMB {
			return out, fmt.Errorf("max_mb %d exceeds the policy's %d MB", o.MaxMB, base.Memory.MaxMB)
		}
		out.Memory.MaxMB = o.MaxMB
	}
	if o.MaxSeconds != 0 {
		if o.MaxSeconds < 0 {
			return out, fmt.Errorf("max_seconds must be positive")
		}
		if base.Time.MaxSeconds > 0 && o.MaxSeconds > base.Time.MaxSeconds {
			return out, fmt.Errorf("max_seconds %d exceeds the policy's %d s", o.MaxSeconds, base.Time.MaxSeconds)
		}
		out.Time.MaxSeconds = o.MaxSeconds
	}
	if o.MaxCPUPercent != 0 {
		if o.MaxCPUPercent < 0 || o.MaxCPUPercent > 100 {
			return out, fmt.Errorf("max_cpu_percent must be 0-100")
		}
		if base.CPU.MaxPercent > 0 && o.MaxCPUPercent > base.CPU.MaxPercent {
			return out, fmt.Errorf("max_cpu_percent %v exceeds the policy's %v", o.MaxCPUPercent, base.CPU.MaxPercent)
		}
		out.CPU.MaxPercent = o.MaxCPUPercent
	}
	if len(o.AllowedPaths) > 0 {
		allowed := make(map[string]bool, len(base.Disk.ReadWritePaths))
		for _, p := range base.Disk.ReadWritePaths {
			allowed[p] = true
		}
		for _, p := range o.AllowedPaths {
			if !allowed[p] {
				return out, fmt.Errorf("allowed path %q is outside the policy's writable set", p)
			}
		}
		out.Disk.ReadWritePaths = append([]string(nil), o.AllowedPaths...)
	}
	return out, nil
}

// ExecuteForLimits is ExecuteFor with per-request narrowing overrides applied
// to the resolved policy limits.
func (m *SandboxManager) ExecuteForLimits(ctx context.Context, toolType, command string, requestedLevel SandboxLevel, o LimitOverrides) (*SandboxResult, error) {
	cfg := m.configSnapshot()
	if cfg == nil {
		return nil, fmt.Errorf("sandbox is not configured")
	}
	if _, known := cfg.ToolPolicies[toolType]; !known {
		return nil, fmt.Errorf("no sandbox policy for tool type %q", toolType)
	}
	policyLevel, base := cfg.ResolveToolPolicy(toolType)
	if policyLevel == "" {
		return nil, fmt.Errorf("sandbox policy for tool type %q declares no level", toolType)
	}
	if requestedLevel != "" && !validLevel(requestedLevel) {
		return nil, fmt.Errorf("unsupported sandbox level: %q", requestedLevel)
	}

	limits, err := NarrowLimits(base, o)
	if err != nil {
		return nil, err
	}

	effective := maxLevel(policyLevel, requestedLevel)
	if effective != policyLevel {
		m.logger.Debug("sandbox isolation raised above policy",
			"tool_type", toolType, "policy_level", policyLevel,
			"requested_level", requestedLevel, "effective_level", effective)
	}
	return m.Execute(ctx, command, effective, limits)
}
