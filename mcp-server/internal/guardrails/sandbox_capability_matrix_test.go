package guardrails

import (
	"strings"
	"testing"
)

// These tests pin the EXPLICIT capability matrix sandbox-isolation-levels R1
// requires: four named levels, each with a matrix covering filesystem,
// network, process, secrets and state persistence. The two last dimensions
// were absent from ResourceLimits until 2026-11-03 while the requirement-closure
// table marked the requirement EXERCISED.

// TestCapabilityMatrix_CoversAllFiveDimensions is the requirement itself. A
// cell that renders empty would mean the dimension is still undeclared.
func TestCapabilityMatrix_CoversAllFiveDimensions(t *testing.T) {
	for _, tier := range AllSandboxTiers() {
		m, err := CapabilityMatrixForTier(tier)
		if err != nil {
			t.Fatalf("CapabilityMatrixForTier(%s): %v", tier, err)
		}
		if m.Tier != tier {
			t.Errorf("matrix tier = %q, want %q", m.Tier, tier)
		}
		if m.Level == "" {
			t.Errorf("tier %s: level cell is empty", tier)
		}
		for name, cell := range map[string]string{
			"filesystem":        m.Filesystem,
			"network":           m.Network,
			"process":           m.Process,
			"secrets":           m.Secrets,
			"state_persistence": m.StatePersistence,
		} {
			if strings.TrimSpace(cell) == "" {
				t.Errorf("tier %s: %s cell is empty — the dimension is undeclared", tier, name)
			}
		}
	}
}

// TestCapabilityMatrix_FourNamedLevels pins the four canonical names and their
// mapping, so a rename cannot silently drop one.
func TestCapabilityMatrix_FourNamedLevels(t *testing.T) {
	want := map[SandboxTier]SandboxLevel{
		TierObserver:   LevelL1,
		TierRestricted: LevelL1,
		TierContained:  LevelL2,
		TierEphemeral:  LevelL2,
	}
	if len(AllSandboxTiers()) != 4 {
		t.Fatalf("AllSandboxTiers() = %v, want exactly the four canonical tiers", AllSandboxTiers())
	}
	for tier, level := range want {
		m, err := CapabilityMatrixForTier(tier)
		if err != nil {
			t.Fatalf("CapabilityMatrixForTier(%s): %v", tier, err)
		}
		if m.Level != level {
			t.Errorf("tier %s maps to %s, want %s", tier, m.Level, level)
		}
	}
}

// TestCapabilityMatrix_SecretsAndStateCellsPinTheDeclaredPosture pins the exact
// secrets and state-persistence cells the spec names. These are the two
// dimensions that did not exist; if a future edit moves a cell, this test is
// where the change has to be argued.
func TestCapabilityMatrix_SecretsAndStateCellsPinTheDeclaredPosture(t *testing.T) {
	want := map[SandboxTier]struct{ secrets, state string }{
		// Observer inspects and holds no credentials; leaves nothing behind.
		TierObserver: {
			secrets: "env:PATH,HOME,LANG credential-files:deny injected:0",
			state:   "deny",
		},
		// Restricted may receive a few DECLARED secret values but never a
		// credential file; persists bounded state in /workspace.
		TierRestricted: {
			secrets: "env:PATH,HOME,LANG credential-files:deny injected:4",
			state:   "allow:/workspace max-mb:32",
		},
		// Contained is the ONLY tier that may read a mounted credential file:
		// the cap-drop=ALL / no-new-privileges hardening is what makes that safe.
		TierContained: {
			secrets: "env:PATH,HOME,LANG credential-files:allow injected:8",
			state:   "allow:/workspace max-mb:128",
		},
		// Ephemeral is the strictest: no secrets, nothing outlives the run.
		TierEphemeral: {
			secrets: "env:PATH credential-files:deny injected:0",
			state:   "deny",
		},
	}
	for tier, w := range want {
		m, err := CapabilityMatrixForTier(tier)
		if err != nil {
			t.Fatalf("CapabilityMatrixForTier(%s): %v", tier, err)
		}
		if m.Secrets != w.secrets {
			t.Errorf("tier %s secrets cell = %q, want %q", tier, m.Secrets, w.secrets)
		}
		if m.StatePersistence != w.state {
			t.Errorf("tier %s state_persistence cell = %q, want %q", tier, m.StatePersistence, w.state)
		}
	}
}

// TestResolveCapabilityMatrix_ResolvesAndRejects is the scenario the
// requirement names: "the level name resolves to its canonical capability
// matrix and unknown level names are rejected."
func TestResolveCapabilityMatrix_ResolvesAndRejects(t *testing.T) {
	for _, name := range []string{"observer", "restricted", "contained", "ephemeral",
		"Observer", "CONTAINED"} {
		m, err := ResolveCapabilityMatrix(name)
		if err != nil {
			t.Errorf("ResolveCapabilityMatrix(%q): %v", name, err)
			continue
		}
		if m.Tier == "" || m.Level == "" {
			t.Errorf("ResolveCapabilityMatrix(%q) returned an empty matrix: %+v", name, m)
		}
	}

	// L0/L1/L2 are the engine's internal SandboxLevel, not the spec's four
	// named levels. Two tiers map to L2, so accepting it would resolve
	// ambiguously; it must be rejected alongside genuinely unknown names.
	for _, bad := range []string{"", "   ", "L9", "L0", "L1", "L2", "guest", "restricted ", "sandboxed", "none"} {
		if _, err := ResolveCapabilityMatrix(bad); err == nil {
			t.Errorf("ResolveCapabilityMatrix(%q) resolved; unknown level names must be rejected", bad)
		}
	}
}

// TestResolveCapabilityMatrix_RejectionNamesTheKnownLevels proves the error
// teaches the caller what IS accepted, rather than only refusing.
func TestResolveCapabilityMatrix_RejectionNamesTheKnownLevels(t *testing.T) {
	_, err := ResolveCapabilityMatrix("guest")
	if err == nil {
		t.Fatal("expected an error for an unknown level name")
	}
	for _, name := range []string{"observer", "restricted", "contained", "ephemeral"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("rejection %q does not name the known level %q", err, name)
		}
	}
}

// TestScrubEnvironment_DropsUndeclaredSecrets is the enforcement behind the
// secrets cell, and the fix for the escape the conformance report claimed was
// already blocked: exec built no cmd.Env, so the child inherited the parent's
// entire environment.
func TestScrubEnvironment_DropsUndeclaredSecrets(t *testing.T) {
	_, limits, err := TierToSandboxLevel(TierRestricted)
	if err != nil {
		t.Fatalf("TierToSandboxLevel: %v", err)
	}
	parent := []string{
		"PATH=/usr/bin",
		"HOME=/home/agent",
		"LANG=C",
		"OPENAI_API_KEY=fixture-not-a-real-credential",
		"AWS_SECRET_ACCESS_KEY=fixture-not-a-real-credential",
		"MY_TOKEN=fixture-not-a-real-credential",
	}
	got := ScrubEnvironment(parent, limits)

	want := map[string]bool{"PATH": true, "HOME": true, "LANG": true}
	seen := map[string]bool{}
	for _, kv := range got {
		name, _, _ := strings.Cut(kv, "=")
		seen[name] = true
		if !want[name] {
			t.Errorf("ScrubEnvironment leaked %q (value not printed)", name)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("ScrubEnvironment dropped declared name %q", name)
		}
	}
	if len(got) != len(want) {
		t.Errorf("ScrubEnvironment returned %d entries, want %d: names %v", len(got), len(want), got)
	}
}

// TestScrubEnvironment_EmptyAllowlistYieldsNoEnvironment proves the fail-closed
// reading: a cell that declares no env gives the workload none, rather than
// falling back to "inherit everything".
func TestScrubEnvironment_EmptyAllowlistYieldsNoEnvironment(t *testing.T) {
	limits := ResourceLimits{}
	limits.Secrets.AllowedEnv = nil
	parent := []string{"PATH=/usr/bin", "OPENAI_API_KEY=fixture-not-a-real-credential"}
	if got := ScrubEnvironment(parent, limits); len(got) != 0 {
		t.Fatalf("empty allowlist returned %v, want no environment at all", got)
	}
}

// TestScrubEnvironment_NonVacuous pins that the scrubber can actually FAIL: a
// name that is not in the cell must be absent from the result. A scrubber that
// returned the parent unchanged would satisfy every assertion above except this
// class of check, so the drop is asserted directly.
func TestScrubEnvironment_NonVacuous(t *testing.T) {
	limits := ResourceLimits{}
	limits.Secrets.AllowedEnv = []string{"PATH"}
	got := ScrubEnvironment([]string{"PATH=/bin", "OPENAI_API_KEY=fixture-not-a-real-credential"}, limits)
	if len(got) != 1 {
		t.Fatalf("got %d entries, want exactly the one declared name: %v", len(got), got)
	}
	if !strings.HasPrefix(got[0], "PATH=") {
		t.Fatalf("got %q, want only PATH", got[0])
	}
}

// TestContainerEnvArgs_ForwardsDeclaredNamesExceptPATH pins the L2 form of the
// same enforcement. PATH is the interesting case: forwarding the host's PATH
// into a container replaces the image's, and every command then exits 127.
func TestContainerEnvArgs_ForwardsDeclaredNamesExceptPATH(t *testing.T) {
	_, limits, err := TierToSandboxLevel(TierContained)
	if err != nil {
		t.Fatalf("TierToSandboxLevel: %v", err)
	}
	args := containerEnvArgs(limits)

	var names []string
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) || args[i] != "--env" {
			t.Fatalf("containerEnvArgs returned malformed args: %v", args)
		}
		names = append(names, args[i+1])
	}
	joined := strings.Join(names, ",")
	if strings.Contains(strings.ToUpper(joined), "PATH") {
		t.Errorf("PATH was forwarded into the container (%v); the image owns PATH and the host's is unresolvable there", names)
	}
	for _, want := range []string{"HOME", "LANG"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("declared name %q was not forwarded: %v", want, names)
		}
	}
	// No value may appear in argv — name only.
	for _, a := range args {
		if a == "--env" {
			continue
		}
		if strings.Contains(a, "=") {
			t.Errorf("containerEnvArgs put a value in argv (%q); `ps` on the host would re-print it", a)
		}
	}
}

// TestContainerEnvArgs_EmptyCellForwardsNothing: the container's own clean env
// is the correct answer when the cell declares nothing.
func TestContainerEnvArgs_EmptyCellForwardsNothing(t *testing.T) {
	limits := ResourceLimits{}
	if got := containerEnvArgs(limits); len(got) != 0 {
		t.Fatalf("empty secrets cell produced %v, want no --env flags", got)
	}
}

// TestTierSecretsCell_IsWiredToCheckSecretContainment proves the matrix cell is
// what CheckSecretContainment enforces, rather than only the fixture's own
// list. Before this, a fixture could declare an allowlist and the tier could
// say nothing about secrets at all.
func TestTierSecretsCell_IsWiredToCheckSecretContainment(t *testing.T) {
	_, limits, err := TierToSandboxLevel(TierRestricted)
	if err != nil {
		t.Fatalf("TierToSandboxLevel: %v", err)
	}
	// A name the tier does not allow is blocked even when the workload declares
	// it: the tier is the authority.
	blocked, reason := CheckSecretContainment("OPENAI_API_KEY", "", []string{"OPENAI_API_KEY"}, limits)
	if !blocked {
		t.Fatalf("tier allowed a secret outside its own allowlist: %s", reason)
	}
	if !strings.Contains(reason, "tier allows only") {
		t.Errorf("reason %q does not say the tier's allowlist decided", reason)
	}

	// A name the tier allows but the workload never declared is also blocked.
	blocked, reason = CheckSecretContainment("HOME", "", []string{"PATH"}, limits)
	if !blocked {
		t.Fatalf("tier allowed an undeclared secret: %s", reason)
	}
	if !strings.Contains(reason, "never declared") {
		t.Errorf("reason %q does not say the workload never declared it", reason)
	}

	// Contained may read a credential file; Restricted may not.
	_, contained, err := TierToSandboxLevel(TierContained)
	if err != nil {
		t.Fatalf("TierToSandboxLevel(contained): %v", err)
	}
	// /workspace is inside both tiers' writable set, so the mount alone would
	// allow the read; the secrets cell is what decides.
	blocked, reason = CheckSecretContainment("", "/workspace/creds.json", nil, limits)
	if !blocked {
		t.Fatalf("restricted allowed a credential file read: %s", reason)
	}
	if !strings.Contains(reason, "allow_credential_files=false") {
		t.Errorf("reason %q does not name the secrets cell", reason)
	}
	blocked, reason = CheckSecretContainment("", "/workspace/creds.json", nil, contained)
	if blocked {
		t.Fatalf("contained refused a credential file read it is hardened enough to allow: %s", reason)
	}
}

// TestMergeLimits_SecretsSwitchesAreNotInherited pins the documented rule: a
// security switch that is false in a policy is false, even when the global
// default turns it on. Inheriting a true would hand every policy a capability
// its own declaration never asked for.
func TestMergeLimits_SecretsSwitchesAreNotInherited(t *testing.T) {
	var global ResourceLimits
	global.Secrets.AllowedEnv = []string{"PATH"}
	global.Secrets.AllowCredentialFiles = true
	global.Secrets.MaxSecrets = 8
	global.StatePersistence.Allowed = true
	global.StatePersistence.MaxPersistedMB = 128
	global.StatePersistence.PersistPaths = []string{"/workspace"}

	// A policy that declares nothing about secrets/state.
	out := mergeLimits(global, ResourceLimits{})
	if out.Secrets.AllowCredentialFiles {
		t.Error("allow_credential_files was inherited as true from the global; absence must mean deny")
	}
	if out.Secrets.MaxSecrets != 0 {
		t.Errorf("max_secrets = %d, want 0 (absence means inject none)", out.Secrets.MaxSecrets)
	}
	if out.StatePersistence.Allowed {
		t.Error("state_persistence.allowed was inherited as true from the global; absence must mean deny")
	}
	// The list-valued fields DO inherit (a policy states only what it changes).
	if len(out.Secrets.AllowedEnv) != 1 || out.Secrets.AllowedEnv[0] != "PATH" {
		t.Errorf("allowed_env = %v, want the global list inherited", out.Secrets.AllowedEnv)
	}
	if len(out.StatePersistence.PersistPaths) != 1 {
		t.Errorf("persist_paths = %v, want the global list inherited", out.StatePersistence.PersistPaths)
	}

	// A policy that names them gets them.
	policy := ResourceLimits{}
	policy.Secrets.AllowedEnv = []string{"PATH", "HOME"}
	policy.Secrets.AllowCredentialFiles = true
	policy.Secrets.MaxSecrets = 2
	policy.StatePersistence.Allowed = true
	policy.StatePersistence.MaxPersistedMB = 16
	out = mergeLimits(global, policy)
	if !out.Secrets.AllowCredentialFiles || out.Secrets.MaxSecrets != 2 {
		t.Errorf("policy secrets not applied: %+v", out.Secrets)
	}
	if !out.StatePersistence.Allowed || out.StatePersistence.MaxPersistedMB != 16 {
		t.Errorf("policy state_persistence not applied: %+v", out.StatePersistence)
	}
}

// TestValidate_RejectsBadSecretsAndStateCells proves the new fields are
// actually validated, so a typo in a security control fails at load.
func TestValidate_RejectsBadSecretsAndStateCells(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*SandboxConfig)
		wants string
	}{
		{"negative max_secrets", func(c *SandboxConfig) { c.GlobalDefaults.Secrets.MaxSecrets = -1 }, "max_secrets"},
		{"negative max_persisted_mb", func(c *SandboxConfig) { c.GlobalDefaults.StatePersistence.MaxPersistedMB = -1 }, "max_persisted_mb"},
		{"empty allowed_env entry", func(c *SandboxConfig) { c.GlobalDefaults.Secrets.AllowedEnv = []string{"PATH", ""} }, "empty"},
		{"allowed_env entry with =", func(c *SandboxConfig) { c.GlobalDefaults.Secrets.AllowedEnv = []string{"PATH=BAD"} }, "must not contain"},
		{"persist path with colon", func(c *SandboxConfig) { c.GlobalDefaults.StatePersistence.PersistPaths = []string{"/w:rw"} }, "persist_paths"},
	}
	for _, tc := range cases {
		cfg := DefaultSandboxConfig()
		tc.mut(cfg)
		err := cfg.Validate()
		if err == nil {
			t.Errorf("%s: Validate accepted it", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.wants) {
			t.Errorf("%s: error %q does not mention %q", tc.name, err, tc.wants)
		}
	}
}
