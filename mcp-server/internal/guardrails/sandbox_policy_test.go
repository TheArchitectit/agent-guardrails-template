package guardrails

import "testing"

func TestResolveToolPolicyInheritsGlobalDefaults(t *testing.T) {
	cfg := DefaultSandboxConfig()

	level, limits := cfg.ResolveToolPolicy("bash")
	if level != LevelL1 {
		t.Fatalf("bash policy level = %q, want L1", level)
	}
	// The bash policy declares only time limits; everything else must come
	// from the global defaults rather than being zeroed.
	if limits.Memory.MaxMB != cfg.GlobalDefaults.Memory.MaxMB {
		t.Errorf("memory inherited as %d, want %d", limits.Memory.MaxMB, cfg.GlobalDefaults.Memory.MaxMB)
	}
	if limits.CPU.MaxPercent != cfg.GlobalDefaults.CPU.MaxPercent {
		t.Errorf("cpu inherited as %v, want %v", limits.CPU.MaxPercent, cfg.GlobalDefaults.CPU.MaxPercent)
	}
	if len(limits.Disk.ReadWritePaths) == 0 {
		t.Error("disk paths not inherited from global defaults")
	}
	if limits.Time.MaxSeconds != 120 {
		t.Errorf("bash max_seconds = %d, want the policy's 120", limits.Time.MaxSeconds)
	}
	// Network is explicit: bash never declared it, so it gets none even
	// though the git policy (and nothing global) enables it.
	if limits.Network.Enabled {
		t.Error("bash policy inherited network access it never declared")
	}
}

func TestResolveToolPolicyNetworkIsNotInheritedFromSibling(t *testing.T) {
	cfg := DefaultSandboxConfig()
	_, gitLimits := cfg.ResolveToolPolicy("git")
	_, bashLimits := cfg.ResolveToolPolicy("bash")
	if !gitLimits.Network.Enabled {
		t.Fatal("git policy should declare network")
	}
	if bashLimits.Network.Enabled {
		t.Error("bash inherited git's network posture — policies must not leak into each other")
	}
}

func TestValidateRejectsUnknownPolicyLevel(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.ToolPolicies["bash"] = SandboxPolicy{DefaultLevel: "L9"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("a policy naming an unknown level validated clean")
	}
}

func TestValidateRejectsEmptyPolicyLevel(t *testing.T) {
	cfg := DefaultSandboxConfig()
	cfg.ToolPolicies["bash"] = SandboxPolicy{}
	if err := cfg.Validate(); err == nil {
		t.Fatal("a policy with no level validated clean")
	}
}

func TestSetToolPolicyAppliesAndRefusesInvalid(t *testing.T) {
	m := NewSandboxManager(DefaultSandboxConfig(), nil)

	err := m.SetToolPolicy("custom", SandboxPolicy{
		DefaultLevel: LevelL2,
		ResourceLimits: ResourceLimits{
			Memory: struct {
				MaxMB       int  `yaml:"max_mb"`
				SwapAllowed bool `yaml:"swap_allowed"`
			}{MaxMB: 64},
		},
	})
	if err != nil {
		t.Fatalf("valid policy refused: %v", err)
	}
	cfg := m.Config()
	level, limits := cfg.ResolveToolPolicy("custom")
	if level != LevelL2 || limits.Memory.MaxMB != 64 {
		t.Fatalf("policy not applied: level=%q mem=%d", level, limits.Memory.MaxMB)
	}

	// A refused mutation must leave the table exactly as it was.
	before := m.ToolPolicies()
	if err := m.SetToolPolicy("custom", SandboxPolicy{DefaultLevel: "L9"}); err == nil {
		t.Fatal("invalid policy accepted")
	}
	after := m.ToolPolicies()
	if len(after) != len(before) {
		t.Fatalf("refused mutation changed the table: %d -> %d entries", len(before), len(after))
	}
	if after["custom"].DefaultLevel != LevelL2 {
		t.Error("refused mutation overwrote the previous policy")
	}
}

func TestMaxLevelNeverLowers(t *testing.T) {
	cases := []struct{ a, b, want SandboxLevel }{
		{LevelL0, LevelL1, LevelL1},
		{LevelL1, LevelL0, LevelL1},
		{LevelL1, LevelL2, LevelL2},
		{LevelL2, LevelL0, LevelL2},
		{LevelL0, LevelL0, LevelL0},
	}
	for _, tc := range cases {
		if got := maxLevel(tc.a, tc.b); got != tc.want {
			t.Errorf("maxLevel(%s,%s) = %s, want %s", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestParseSandboxLevelRefusesEmptyAndUnknown(t *testing.T) {
	if _, err := ParseSandboxLevel(""); err == nil {
		t.Error("empty level accepted — a missing level must resolve from policy, not default to L0")
	}
	if _, err := ParseSandboxLevel("L3"); err == nil {
		t.Error("L3 accepted although this implementation has no L3")
	}
	if lvl, err := ParseSandboxLevel("L1"); err != nil || lvl != LevelL1 {
		t.Errorf("L1 rejected: %v", err)
	}
}
