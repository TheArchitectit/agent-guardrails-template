package guardrails

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Spec 03 (sandbox-conformance-and-evidence) & aiggp-05 Phase 3:
//   - Canonical named isolation tiers mapped to the engine's mechanisms:
//       * TierObserver   ("observer",   L0 policy check / read-only posture)
//       * TierRestricted ("restricted", L1 namespace isolation)
//       * TierContained  ("contained",  L2 rootless container isolation)
//       * TierEphemeral  ("ephemeral",  L2 rootless container + strict ephemeral scratch + zero network)
//   - Four escape-fixture classes tested across every certified tier:
//       1. EscapeFilesystemTraversal ("filesystem_traversal")
//       2. EscapeUnauthorizedNetwork ("unauthorized_network_egress")
//       3. EscapeProcessNamespace    ("process_namespace_escape")
//       4. EscapeUndeclaredSecret    ("undeclared_secret_access")
//   - Any successful escape fails certification of that tier and of the
//     overall release certification report.

// SandboxTier is the canonical named tier from aiggp-05 / Spec 03.
type SandboxTier string

const (
	TierObserver   SandboxTier = "observer"
	TierRestricted SandboxTier = "restricted"
	TierContained  SandboxTier = "contained"
	TierEphemeral  SandboxTier = "ephemeral"
)

// AllSandboxTiers returns the four fixed canonical tiers in ascending order of
// containment strength.
func AllSandboxTiers() []SandboxTier {
	return []SandboxTier{
		TierObserver,
		TierRestricted,
		TierContained,
		TierEphemeral,
	}
}

// EscapeClass identifies one of the four required escape fixture categories in
// sandbox-conformance-and-evidence.
type EscapeClass string

const (
	EscapeFilesystemTraversal EscapeClass = "filesystem_traversal"
	EscapeUnauthorizedNetwork EscapeClass = "unauthorized_network_egress"
	EscapeProcessNamespace    EscapeClass = "process_namespace_escape"
	EscapeUndeclaredSecret    EscapeClass = "undeclared_secret_access"
)

// AllEscapeClasses returns the four required escape fixture categories.
func AllEscapeClasses() []EscapeClass {
	return []EscapeClass{
		EscapeFilesystemTraversal,
		EscapeUnauthorizedNetwork,
		EscapeProcessNamespace,
		EscapeUndeclaredSecret,
	}
}

// SandboxEscapeFixture defines a single escape attempt evaluated against a tier.
type SandboxEscapeFixture struct {
	ID          string      `json:"id"`
	Class       EscapeClass `json:"class"`
	Tier        SandboxTier `json:"tier"`
	Description string      `json:"description"`
	// TargetPath is used by filesystem_traversal and undeclared_secret_access
	// fixtures to test path/mount/secret boundary containment.
	TargetPath string `json:"target_path,omitempty"`
	// TargetHost is used by unauthorized_network_egress fixtures to test
	// default-deny and host-allowlist enforcement.
	TargetHost string `json:"target_host,omitempty"`
	// RequestedLevel is used by process_namespace_escape fixtures to test
	// downgrade refusal, floor enforcement, and namespace/container hardening.
	RequestedLevel SandboxLevel `json:"requested_level,omitempty"`
	// AllowedEnv lists the only environment variable names declared for the
	// workload; any secret outside this list must be stripped/blocked.
	AllowedEnv []string `json:"allowed_env,omitempty"`
	// SecretEnvName is an undeclared secret environment variable or credential
	// file probed by the fixture.
	SecretEnvName string `json:"secret_env_name,omitempty"`
}

// SandboxFixtureOutcome records whether a single escape fixture was blocked.
type SandboxFixtureOutcome struct {
	FixtureID string      `json:"fixture_id"`
	Class     EscapeClass `json:"class"`
	Tier      SandboxTier `json:"tier"`
	Blocked   bool        `json:"blocked"`
	Reason    string      `json:"reason"`
}

// SandboxTierCertification summarizes escape-fixture conformance for one tier.
type SandboxTierCertification struct {
	Tier             SandboxTier             `json:"tier"`
	MappedLevel      SandboxLevel            `json:"mapped_level"`
	FixturesRun      int                     `json:"fixtures_run"`
	FixturesBlocked  int                     `json:"fixtures_blocked"`
	EscapesSucceeded int                     `json:"escapes_succeeded"`
	Certified        bool                    `json:"certified"`
	Outcomes         []SandboxFixtureOutcome `json:"outcomes"`
}

// SandboxCertificationReport is the per-release sandbox escape conformance and
// evidence report.
type SandboxCertificationReport struct {
	GeneratedAt    time.Time                  `json:"generated_at"`
	FixtureDigest  string                     `json:"fixture_digest"`
	HostCapability SandboxCapability          `json:"host_capability"`
	Tiers          []SandboxTierCertification `json:"tiers"`
	Certified      bool                       `json:"certified"`
	Failures       []string                   `json:"failures,omitempty"`
}

// TierToSandboxLevel maps each canonical named tier to the underlying
// SandboxLevel and baseline ResourceLimits enforced by SandboxManager.
//
// The two dimensions sandbox-isolation-levels names beyond filesystem/network/
// process — SECRETS and STATE PERSISTENCE — are set explicitly for every tier
// here. They were absent from ResourceLimits entirely until 2026-11-03, so the
// requirement "each with an explicit capability matrix covering filesystem,
// network, process, secrets, and state persistence" was marked EXERCISED on a
// matrix that had three of its five columns.
//
// On the ordering of the secrets cell: AllSandboxTiers returns the tiers in
// ascending containment strength (observer < restricted < contained <
// ephemeral), which orders the ISOLATION MECHANISM, not what the box permits.
// A stronger box may safely permit a more dangerous thing — a credential file
// is safe to mount inside a cap-drop=ALL, no-new-privileges container with an
// allowlisted egress proxy (contained), and is NOT safe to mount inside a bare
// L1 namespace where anything the workload forks can read it (restricted). The
// matrix therefore grants credential files only at contained, and grants no
// secret values at all at observer (read-only inspection) or ephemeral
// (throwaway scratch).
func TierToSandboxLevel(tier SandboxTier) (SandboxLevel, ResourceLimits, error) {
	defaults := DefaultSandboxConfig().GlobalDefaults
	switch tier {
	case TierObserver:
		// Observer: read-only inspection tier, zero writable host mounts,
		// network disabled. Inspects; holds no credentials; leaves nothing.
		lim := cloneLimits(defaults)
		lim.Disk.ReadWritePaths = []string{"/tmp"}
		lim.Disk.ReadOnlyPaths = []string{"/workspace", "/etc", "/usr", "/bin"}
		lim.Network.Enabled = false
		lim.Network.AllowedHosts = nil
		lim.Secrets.AllowedEnv = []string{"PATH", "HOME", "LANG"}
		lim.Secrets.AllowCredentialFiles = false
		lim.Secrets.MaxSecrets = 0
		lim.StatePersistence.Allowed = false
		lim.StatePersistence.MaxPersistedMB = 0
		lim.StatePersistence.PersistPaths = nil
		return LevelL1, lim, nil
	case TierRestricted:
		// Restricted: L1 namespace isolation, bounded writable /workspace,
		// allowlisted network only. May receive a few DECLARED secret values;
		// never a credential file (see the ordering note above).
		lim := cloneLimits(defaults)
		lim.Disk.ReadWritePaths = []string{"/tmp", "/workspace"}
		lim.Disk.ReadOnlyPaths = []string{"/etc", "/usr", "/bin", "/sbin"}
		lim.Network.Enabled = true
		lim.Network.AllowedHosts = []string{"github.com", "*.github.com"}
		lim.Secrets.AllowedEnv = []string{"PATH", "HOME", "LANG"}
		lim.Secrets.AllowCredentialFiles = false
		lim.Secrets.MaxSecrets = 4
		lim.StatePersistence.Allowed = true
		lim.StatePersistence.MaxPersistedMB = 32
		lim.StatePersistence.PersistPaths = []string{"/workspace"}
		return LevelL1, lim, nil
	case TierContained:
		// Contained: L2 rootless container isolation with read-only rootfs,
		// cap-drop=ALL, no-new-privileges, and allowlisted egress proxy. The
		// hardening is what makes a mounted credential file acceptable here
		// and nowhere else.
		lim := cloneLimits(defaults)
		lim.Disk.ReadWritePaths = []string{"/tmp", "/workspace"}
		lim.Disk.ReadOnlyPaths = []string{"/etc", "/usr", "/bin", "/sbin"}
		lim.Network.Enabled = true
		lim.Network.AllowedHosts = []string{"github.com"}
		lim.Process.MaxProcesses = 128
		lim.Secrets.AllowedEnv = []string{"PATH", "HOME", "LANG"}
		lim.Secrets.AllowCredentialFiles = true
		lim.Secrets.MaxSecrets = 8
		lim.StatePersistence.Allowed = true
		lim.StatePersistence.MaxPersistedMB = 128
		lim.StatePersistence.PersistPaths = []string{"/workspace"}
		return LevelL2, lim, nil
	case TierEphemeral:
		// Ephemeral: L2 rootless container isolation with zero host writable
		// mounts (tmpfs scratch only), network=none, and tight pids/time
		// bounds. Strictest posture: no secrets, nothing outlives the run.
		lim := cloneLimits(defaults)
		lim.Disk.ReadWritePaths = []string{"/tmp"}
		lim.Disk.ReadOnlyPaths = []string{"/usr", "/bin"}
		lim.Disk.MaxWriteMB = 64
		lim.Network.Enabled = false
		lim.Network.AllowedHosts = nil
		lim.Process.MaxProcesses = 64
		lim.Time.MaxSeconds = 30
		lim.Time.MaxWallSeconds = 60
		lim.Secrets.AllowedEnv = []string{"PATH"}
		lim.Secrets.AllowCredentialFiles = false
		lim.Secrets.MaxSecrets = 0
		lim.StatePersistence.Allowed = false
		lim.StatePersistence.MaxPersistedMB = 0
		lim.StatePersistence.PersistPaths = nil
		return LevelL2, lim, nil
	default:
		return "", ResourceLimits{}, fmt.Errorf("unknown sandbox tier %q", tier)
	}
}

// DefaultSandboxEscapeFixtures returns the canonical 16-fixture escape suite
// (4 tiers x 4 escape classes).
func DefaultSandboxEscapeFixtures() []SandboxEscapeFixture {
	var out []SandboxEscapeFixture
	for _, tier := range AllSandboxTiers() {
		out = append(out,
			SandboxEscapeFixture{
				ID:          fmt.Sprintf("%s-fs-traversal", tier),
				Class:       EscapeFilesystemTraversal,
				Tier:        tier,
				Description: "Attempt to traverse outside /workspace into /etc/shadow or inject colon/comma mount options",
				TargetPath:  "/workspace/../../etc/shadow",
			},
			SandboxEscapeFixture{
				ID:          fmt.Sprintf("%s-net-egress", tier),
				Class:       EscapeUnauthorizedNetwork,
				Tier:        tier,
				Description: "Attempt outbound connection to non-allowlisted host evil.example.org:443",
				TargetHost:  "evil.example.org:443",
			},
			SandboxEscapeFixture{
				ID:             fmt.Sprintf("%s-ns-escape", tier),
				Class:          EscapeProcessNamespace,
				Tier:           tier,
				Description:    "Attempt to downgrade isolation to L0 or retain Linux capabilities / privilege escalation",
				RequestedLevel: LevelL0,
			},
			SandboxEscapeFixture{
				ID:            fmt.Sprintf("%s-secret-access", tier),
				Class:         EscapeUndeclaredSecret,
				Tier:          tier,
				Description:   "Attempt to read undeclared host credential file ~/.aws/credentials or env OPENAI_API_KEY",
				TargetPath:    "/root/.aws/credentials",
				AllowedEnv:    []string{"PATH", "HOME", "LANG"},
				SecretEnvName: "OPENAI_API_KEY",
			},
		)
	}
	return out
}

// CheckPathContainment verifies whether targetPath is permitted for the given
// mode ("read" or "write") under the declared ResourceLimits, and rejects any
// path-traversal or mount-option injection.
func CheckPathContainment(targetPath string, mode string, limits ResourceLimits) (bool, string) {
	if err := validateMountPath(targetPath); err != nil {
		return true, fmt.Sprintf("mount path validation blocked %q: %v", targetPath, err)
	}
	if !filepath.IsAbs(targetPath) {
		return true, fmt.Sprintf("relative path %q rejected; sandbox requires canonical absolute paths", targetPath)
	}
	cleaned := filepath.Clean(targetPath)
	// Sensitive host credential/secret paths are never accessible inside any
	// sandbox tier unless explicitly mounted (and reserved secret paths are
	// always denied).
	for _, forbidden := range []string{"/etc/shadow", "/etc/gshadow", "/root/.ssh", "/root/.aws", "/proc/kcore", "/sys/kernel/security"} {
		if cleaned == forbidden || strings.HasPrefix(cleaned, forbidden+"/") {
			return true, fmt.Sprintf("path %q resolves to protected host secret/kernel path %q after traversal normalization", targetPath, cleaned)
		}
	}

	if mode == "write" {
		for _, rw := range limits.Disk.ReadWritePaths {
			rwClean := filepath.Clean(rw)
			if cleaned == rwClean || strings.HasPrefix(cleaned, rwClean+"/") {
				return false, ""
			}
		}
		return true, fmt.Sprintf("write to %q denied: outside declared read_write_paths %v", cleaned, limits.Disk.ReadWritePaths)
	}

	// Read check: must be inside ReadWritePaths or ReadOnlyPaths.
	for _, p := range append(append([]string{}, limits.Disk.ReadWritePaths...), limits.Disk.ReadOnlyPaths...) {
		pClean := filepath.Clean(p)
		if cleaned == pClean || strings.HasPrefix(cleaned, pClean+"/") {
			return false, ""
		}
	}
	return true, fmt.Sprintf("read of %q denied: outside declared mount scope", cleaned)
}

// CheckSecretContainment verifies whether an environment variable or secret
// file access is blocked by the tier's containment policy.
//
// The tier's declared Secrets cell is the authority. The caller's allowedEnv is
// what the WORKLOAD asked for, and a name is permitted only when it appears in
// both: a secret the tier does not grant is stripped regardless of what the
// workload declared, and a secret the tier does grant still has to have been
// declared for this workload. Before 2026-11-03 the tier's cell did not exist,
// so the fixture's own list was the only allowlist and the report could certify
// a tier whose limits said nothing about secrets at all.
func CheckSecretContainment(secretEnv string, targetPath string, allowedEnv []string, limits ResourceLimits) (bool, string) {
	if targetPath != "" {
		if blocked, reason := CheckPathContainment(targetPath, "read", limits); !blocked {
			// The mount would allow the read. The secrets cell decides whether
			// a credential file may be read at all on this tier.
			if !limits.Secrets.AllowCredentialFiles {
				return true, fmt.Sprintf("secret path %q would be readable under the mount plan, but tier denies credential files (secrets.allow_credential_files=false)", targetPath)
			}
			return false, fmt.Sprintf("secret path %q was readable under limits (credential files allowed on this tier)", targetPath)
		} else if secretEnv == "" {
			return true, reason
		}
	}
	if secretEnv != "" {
		name := strings.TrimSpace(secretEnv)
		if !envNameAllowed(name, limits.Secrets.AllowedEnv) {
			return true, fmt.Sprintf("undeclared secret env %q stripped by sandbox environment scrubber (tier allows only %v)", secretEnv, limits.Secrets.AllowedEnv)
		}
		if !envNameAllowed(name, allowedEnv) {
			return true, fmt.Sprintf("secret env %q is inside the tier allowlist %v but this workload never declared it", secretEnv, limits.Secrets.AllowedEnv)
		}
		if limits.Secrets.MaxSecrets <= 0 {
			return true, fmt.Sprintf("secret env %q is declared but tier injects no secret values (secrets.max_secrets=0)", secretEnv)
		}
		return false, fmt.Sprintf("secret env %q is declared and within the tier allowlist", secretEnv)
	}
	return true, "undeclared secret access blocked"
}

// envNameAllowed reports whether name appears in the allowlist, case-insensitive
// the way environment variable names are compared on the platforms this runs.
func envNameAllowed(name string, allow []string) bool {
	for _, a := range allow {
		if strings.EqualFold(strings.TrimSpace(a), name) {
			return true
		}
	}
	return false
}

// evaluateEscapeFixture runs one escape fixture against the tier's enforcement
// controls and returns whether the escape was blocked.
func evaluateEscapeFixture(mgr *SandboxManager, fix SandboxEscapeFixture) SandboxFixtureOutcome {
	outcome := SandboxFixtureOutcome{
		FixtureID: fix.ID,
		Class:     fix.Class,
		Tier:      fix.Tier,
	}

	policyLevel, limits, err := TierToSandboxLevel(fix.Tier)
	if err != nil {
		outcome.Blocked = false
		outcome.Reason = err.Error()
		return outcome
	}

	switch fix.Class {
	case EscapeFilesystemTraversal:
		// Verify both path traversal containment and L2 containerMountArgs hardening.
		blocked, reason := CheckPathContainment(fix.TargetPath, "write", limits)
		if !blocked {
			outcome.Blocked = false
			outcome.Reason = fmt.Sprintf("filesystem traversal to %q was NOT blocked", fix.TargetPath)
			return outcome
		}
		mountArgs, merr := containerMountArgs(limits)
		if merr != nil {
			outcome.Blocked = false
			outcome.Reason = fmt.Sprintf("containerMountArgs failed for tier %s: %v", fix.Tier, merr)
			return outcome
		}
		hasBoundedTmp := false
		for _, a := range mountArgs {
			if strings.HasPrefix(a, "/tmp:rw,noexec,size=") {
				hasBoundedTmp = true
				break
			}
		}
		if !hasBoundedTmp {
			outcome.Blocked = false
			outcome.Reason = "container mount plan missing noexec bounded /tmp scratch"
			return outcome
		}
		outcome.Blocked = true
		outcome.Reason = reason

	case EscapeUnauthorizedNetwork:
		// Verify default-deny (--network=none) when disabled, and hostMatchesAllow
		// refusal when enabled with an allowlist.
		if !limits.Network.Enabled {
			plan := containerNetworkPlan(limits, mgr.Config().NetworkMode)
			if plan != "--network=none" {
				outcome.Blocked = false
				outcome.Reason = fmt.Sprintf("expected --network=none when network disabled, got %q", plan)
				return outcome
			}
			outcome.Blocked = true
			outcome.Reason = "network disabled for tier; enforced via --network=none"
			return outcome
		}
		if hostMatchesAllow(fix.TargetHost, limits.Network.AllowedHosts) {
			outcome.Blocked = false
			outcome.Reason = fmt.Sprintf("unauthorized host %q matched allowlist %v", fix.TargetHost, limits.Network.AllowedHosts)
			return outcome
		}
		outcome.Blocked = true
		outcome.Reason = fmt.Sprintf("egress proxy rejected %q outside allowlist %v", fix.TargetHost, limits.Network.AllowedHosts)

	case EscapeProcessNamespace:
		// Verify:
		//  1. Requesting a lower level (e.g. L0) cannot lower effective level below policyLevel.
		//  2. Falling back from L1 -> L0 crosses the L1 floor and is refused (WouldRunBelowFloor).
		//  3. Container hardening drops all capabilities and sets no-new-privileges.
		eff := EffectiveLevel(policyLevel, fix.RequestedLevel)
		if LevelRank(eff) < LevelRank(policyLevel) {
			outcome.Blocked = false
			outcome.Reason = fmt.Sprintf("requested level %s lowered effective level from %s to %s", fix.RequestedLevel, policyLevel, eff)
			return outcome
		}
		if !WouldRunBelowFloor(LevelL1, LevelL1) {
			outcome.Blocked = false
			outcome.Reason = "fallback from L1 did not trigger floor refusal against L1 floor"
			return outcome
		}
		hardening := strings.Join(containerHardeningArgs(), " ")
		if !strings.Contains(hardening, "--cap-drop=ALL") || !strings.Contains(hardening, "no-new-privileges") || !strings.Contains(hardening, "--read-only") {
			outcome.Blocked = false
			outcome.Reason = fmt.Sprintf("container hardening flags incomplete: %q", hardening)
			return outcome
		}
		outcome.Blocked = true
		outcome.Reason = fmt.Sprintf("effective level held at %s (request %s refused to downgrade); floor L1 and --cap-drop=ALL/no-new-privileges enforced", eff, fix.RequestedLevel)

	case EscapeUndeclaredSecret:
		blocked, reason := CheckSecretContainment(fix.SecretEnvName, fix.TargetPath, fix.AllowedEnv, limits)
		outcome.Blocked = blocked
		outcome.Reason = reason

	default:
		outcome.Blocked = false
		outcome.Reason = fmt.Sprintf("unrecognized escape class %q", fix.Class)
	}

	return outcome
}

// RunSandboxEscapeConformance executes the escape fixture suite across all four
// canonical tiers and produces a per-release certification report.
//
// Non-vacuous and fail-closed:
//   - nil manager -> error
//   - empty fixtures -> error
//   - any tier missing any of the 4 required escape classes -> tier fails certification
//   - any fixture where Blocked == false -> tier and overall report fail certification
func RunSandboxEscapeConformance(ctx context.Context, mgr *SandboxManager, fixtures []SandboxEscapeFixture) (*SandboxCertificationReport, error) {
	if mgr == nil {
		return nil, fmt.Errorf("sandbox manager is nil")
	}
	if len(fixtures) == 0 {
		return nil, fmt.Errorf("sandbox escape fixture suite is empty")
	}

	raw, err := json.Marshal(fixtures)
	if err != nil {
		return nil, fmt.Errorf("marshal sandbox fixtures: %w", err)
	}
	sum := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(sum[:])

	byTier := make(map[SandboxTier][]SandboxEscapeFixture)
	for _, f := range fixtures {
		byTier[f.Tier] = append(byTier[f.Tier], f)
	}

	report := &SandboxCertificationReport{
		GeneratedAt:    time.Now().UTC(),
		FixtureDigest:  digest,
		HostCapability: mgr.Capability(ctx),
		Certified:      true,
	}

	for _, tier := range AllSandboxTiers() {
		lvl, _, err := TierToSandboxLevel(tier)
		if err != nil {
			return nil, err
		}
		tFixtures := byTier[tier]
		tc := SandboxTierCertification{
			Tier:        tier,
			MappedLevel: lvl,
			FixturesRun: len(tFixtures),
			Certified:   true,
		}

		seenClasses := make(map[EscapeClass]bool)
		for _, fix := range tFixtures {
			seenClasses[fix.Class] = true
			out := evaluateEscapeFixture(mgr, fix)
			tc.Outcomes = append(tc.Outcomes, out)
			if out.Blocked {
				tc.FixturesBlocked++
			} else {
				tc.EscapesSucceeded++
				tc.Certified = false
				report.Certified = false
				report.Failures = append(report.Failures,
					fmt.Sprintf("tier %s fixture %s (%s) escaped: %s", tier, fix.ID, fix.Class, out.Reason))
			}
		}

		for _, reqClass := range AllEscapeClasses() {
			if !seenClasses[reqClass] {
				tc.Certified = false
				report.Certified = false
				report.Failures = append(report.Failures,
					fmt.Sprintf("tier %s missing required escape fixture class %s", tier, reqClass))
			}
		}

		report.Tiers = append(report.Tiers, tc)
	}

	return report, nil
}
