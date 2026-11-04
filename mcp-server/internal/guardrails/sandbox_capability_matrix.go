package guardrails

// sandbox-isolation-levels R1 ("named levels with capability matrices"):
//
//	The specification SHALL define four isolation levels (observer, restricted,
//	contained, ephemeral), each with an explicit capability matrix covering
//	filesystem, network, process, secrets, and state persistence.
//
// Until 2026-11-03 the "explicit capability matrix" did not exist as a thing a
// caller could read. The five dimensions were spread across ResourceLimits as
// three sections (Disk, Network, Process) plus Time and CPU/Memory that the
// requirement does not name — and the two dimensions it DOES name beyond the
// first three, SECRETS and STATE PERSISTENCE, were absent from the type
// entirely. The requirement-closure table nonetheless marked the requirement
// EXERCISED.
//
// This file is the matrix. It is DERIVED from the ResourceLimits a tier
// actually enforces (TierToSandboxLevel) rather than written alongside them: a
// matrix cell that disagreed with the enforced limits would be a security claim
// the runtime does not implement, which is the exact defect class the closure
// table was already committing.
//
// The scenario the requirement names is "the level name resolves to its
// canonical capability matrix and unknown level names are rejected". That is
// ResolveCapabilityMatrix, below.

import (
	"fmt"
	"strings"
)

// SandboxCapabilityMatrix is the explicit five-dimension capability cell set
// sandbox-isolation-levels requires for every named tier. The five dimension
// names are exactly the five the requirement enumerates.
type SandboxCapabilityMatrix struct {
	Tier             SandboxTier  `json:"tier"`
	Level            SandboxLevel `json:"level"`
	Filesystem       string       `json:"filesystem"`
	Network          string       `json:"network"`
	Process          string       `json:"process"`
	Secrets          string       `json:"secrets"`
	StatePersistence string       `json:"state_persistence"`
}

// CapabilityMatrixForTier returns the explicit capability matrix for one named
// tier, rendered from the limits the tier actually enforces.
//
// An unknown tier is an error, not a zero matrix: a caller that asks for a
// matrix and silently receives empty strings would treat "I do not know this
// level" as "this level permits nothing", which is a different claim.
func CapabilityMatrixForTier(tier SandboxTier) (SandboxCapabilityMatrix, error) {
	level, limits, err := TierToSandboxLevel(tier)
	if err != nil {
		return SandboxCapabilityMatrix{}, err
	}
	return SandboxCapabilityMatrix{
		Tier:             tier,
		Level:            level,
		Filesystem:       renderFilesystem(limits),
		Network:          renderNetwork(limits),
		Process:          renderProcess(limits),
		Secrets:          renderSecrets(limits),
		StatePersistence: renderStatePersistence(limits),
	}, nil
}

// ResolveCapabilityMatrix resolves a policy-bundle level NAME to its canonical
// capability matrix. This is the scenario sandbox-isolation-levels names:
// "the level name resolves to its canonical capability matrix and unknown
// level names are rejected."
//
// Rejection is the point. A bundle naming a level this implementation does not
// have must fail validation rather than resolve to a zero matrix that reads as
// "permits nothing" and certifies clean.
func ResolveCapabilityMatrix(levelName string) (SandboxCapabilityMatrix, error) {
	if strings.TrimSpace(levelName) == "" {
		return SandboxCapabilityMatrix{}, fmt.Errorf("isolation level name is required")
	}
	// Whitespace is refused rather than trimmed. Case is conventional for these
	// names, so `CONTAINED` resolves; a trailing space is an invisible
	// character that a bundle author did not mean to type, and silently
	// accepting it would make a typo look like a valid level name.
	if strings.TrimSpace(levelName) != levelName {
		return SandboxCapabilityMatrix{}, fmt.Errorf("isolation level name %q has leading or trailing whitespace", levelName)
	}
	name := levelName
	// Only the four canonical tier names resolve. The engine's internal
	// SandboxLevel (L0/L1/L2) deliberately does NOT: two of the four tiers map
	// to L2, so a bundle naming "L2" would resolve to whichever tier happened
	// to be iterated first, and the matrix would then disagree with the tier
	// the runtime actually ran. The requirement enumerates the four names; the
	// four names are the contract.
	for _, tier := range AllSandboxTiers() {
		if strings.EqualFold(string(tier), name) {
			return CapabilityMatrixForTier(tier)
		}
	}
	return SandboxCapabilityMatrix{}, fmt.Errorf("unknown isolation level %q: must be one of observer, restricted, contained, ephemeral", levelName)
}

// AllCapabilityMatrices returns the matrix for every canonical tier, in the
// same ascending-containment order as AllSandboxTiers. Used by the certification
// report and by tests that pin the whole table at once.
func AllCapabilityMatrices() ([]SandboxCapabilityMatrix, error) {
	tiers := AllSandboxTiers()
	out := make([]SandboxCapabilityMatrix, 0, len(tiers))
	for _, t := range tiers {
		m, err := CapabilityMatrixForTier(t)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// --- dimension renderers ----------------------------------------------------
//
// Each renderer prints one cell as a single comparable string. They exist so a
// reviewer can diff two tiers' matrices at a glance and so a test can pin a
// cell without re-deriving the policy.

func renderFilesystem(l ResourceLimits) string {
	return fmt.Sprintf("rw:%s ro:%s max-write-mb:%d",
		joinOrNone(l.Disk.ReadWritePaths), joinOrNone(l.Disk.ReadOnlyPaths), l.Disk.MaxWriteMB)
}

func renderNetwork(l ResourceLimits) string {
	if !l.Network.Enabled {
		return "disabled"
	}
	return "allowlist:" + joinOrNone(l.Network.AllowedHosts)
}

func renderProcess(l ResourceLimits) string {
	if l.Process.MaxProcesses <= 0 {
		return "unbounded"
	}
	return fmt.Sprintf("max-processes:%d", l.Process.MaxProcesses)
}

func renderSecrets(l ResourceLimits) string {
	cred := "deny"
	if l.Secrets.AllowCredentialFiles {
		cred = "allow"
	}
	return fmt.Sprintf("env:%s credential-files:%s injected:%d",
		joinOrNone(l.Secrets.AllowedEnv), cred, l.Secrets.MaxSecrets)
}

func renderStatePersistence(l ResourceLimits) string {
	if !l.StatePersistence.Allowed {
		return "deny"
	}
	return fmt.Sprintf("allow:%s max-mb:%d",
		joinOrNone(l.StatePersistence.PersistPaths), l.StatePersistence.MaxPersistedMB)
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ",")
}

// --- environment scrubber ---------------------------------------------------

// ScrubEnvironment returns the environment a sandboxed workload may observe:
// only the names in limits.Secrets.AllowedEnv, taken from parent. Everything
// else the parent holds — every credential-shaped variable, every AWS_* and
// *_API_KEY — is dropped before exec.
//
// This is the enforcement behind the secrets cell. Before 2026-11-03 the
// conformance report asserted an undeclared secret was "stripped by sandbox
// environment scrubber" while execL0/L1/L2 built no cmd.Env at all, so the
// child inherited the parent's ENTIRE environment. The report claimed a control
// the runtime did not apply — a live undeclared-secret escape that the 16
// fixtures certified as blocked.
//
// An empty allowlist yields an empty environment. PATH is granted by naming it,
// never implicitly: a workload that needs PATH must declare it, and the shipped
// defaults do.
func ScrubEnvironment(parent []string, limits ResourceLimits) []string {
	if len(limits.Secrets.AllowedEnv) == 0 {
		return nil
	}
	allow := make(map[string]bool, len(limits.Secrets.AllowedEnv))
	for _, name := range limits.Secrets.AllowedEnv {
		allow[strings.TrimSpace(name)] = true
	}
	out := make([]string, 0, len(limits.Secrets.AllowedEnv))
	seen := make(map[string]bool, len(limits.Secrets.AllowedEnv))
	for _, kv := range parent {
		name, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(name)
		if !allow[key] || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, kv)
	}
	return out
}

// containerEnvArgs returns the `--env` flags that put the secrets cell into
// effect for an L2 execution: the names limits.Secrets.AllowedEnv declares,
// forwarded from the parent by NAME only, minus PATH.
//
// Name-only forwarding is deliberate. `--env NAME` asks the runtime to copy the
// parent's value; `--env NAME=value` would put that value in argv, where `ps`
// on the host re-prints it to anyone who can see the runtime's process. The
// secrets cell is about not leaking, so the value must stay out of argv.
//
// PATH is never forwarded. Inside a container the image owns PATH: the host's
// PATH names host directories that do not exist in the image, so forwarding it
// replaces `/usr/local/sbin:...:/bin` with something unresolvable and every
// command fails with exit 127 — which is how the shipped-defaults L2 test went
// red the first time this forwarded the whole cell. PATH is still in the matrix
// cell because the workload MAY observe it (execL0/execL1 keep the parent's);
// at L2 it is observed from the image, which is the only place it is correct.
// The rule is "the host may contribute these names, except PATH where the
// image is authoritative", not "the cell is forwarded verbatim".
//
// An empty allowlist yields no flags at all, which is the container's own
// default (clean env) — so the matrix cell and the runtime agree by
// construction rather than by hoping the image ships nothing sensitive.
func containerEnvArgs(limits ResourceLimits) []string {
	if len(limits.Secrets.AllowedEnv) == 0 {
		return nil
	}
	args := make([]string, 0, len(limits.Secrets.AllowedEnv)*2)
	for _, name := range limits.Secrets.AllowedEnv {
		n := strings.TrimSpace(name)
		if n == "" || strings.EqualFold(n, "PATH") {
			continue
		}
		args = append(args, "--env", n)
	}
	return args
}
