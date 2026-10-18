package guardrails

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Spec 07 R07-4.4 — runtime activation and auditability of the resolved policy
// set.
//
// The lock-record half of Spec 07 (R07-4.1 identity/determinism, R07-4.3.2
// schema-as-data, R07-5.8 CI verification) is enforced by
// `scripts/check_policy_pack_lock.py`. That gate proves the *record* is honest.
// It does not make the running server obey the pack, and before this file there
// was no runtime consumer at all: the engine resolved policy from its own
// configured rule set, the effective digest was never exposed, and no
// enforcement decision was attributable to a pack rule.
//
// This file is the consumer. It is deliberately narrow:
//
//   - Acquisition is explicit. A pack directory is only ever the one the
//     operator configured (R07-4.3.3); nothing here discovers a path, follows a
//     URL, or reads a location named inside a policy document.
//   - Parsing is inert. Pack bytes are decoded as JSON data. No value in a pack
//     is ever executed, templated, or fetched, so text that looks like a shell
//     command or a prompt stays inert (R07-4.3.1).
//   - Resolution and enforcement are separate operations (R07-4.4.4). Loading
//     produces an immutable snapshot; nothing in this file enforces anything.
//     Attribution is a lookup the enforcement path performs, not a side effect
//     of resolution.
//   - Activation is atomic and candidate-first (R07-4.4.2). A candidate that
//     fails to resolve or validate leaves the previously active set in force —
//     it never degrades to "no policy" and never partially applies.

// PackRule is one inert rule record from a policy pack.
type PackRule struct {
	ID          string `json:"id"`
	Law         string `json:"law"`
	Enforcement string `json:"enforcement"`
	Tool        string `json:"tool"`
	OnUnknown   string `json:"on_unknown"`
	Evidence    string `json:"evidence,omitempty"`
}

// Pack is one parsed policy pack. Seed marks an intentionally empty pack, which
// R07-4.2.4 permits but forbids from being presented as domain coverage.
type Pack struct {
	Name    string `json:"pack"`
	Version string `json:"version"`
	// Tier is the declared precedence class: core, domain or overlay
	// (Spec 07 R07-4.2.1). It is DECLARED rather than inferred from the path so
	// a reviewer sees the precedence claim in the diff, and the declaration is
	// then checked against the path so a domain policy cannot be filed as core.
	Tier      string           `json:"tier"`
	Seed      bool             `json:"seed"`
	Rules     []PackRule       `json:"rules"`
	DependsOn []PackDependency `json:"depends_on,omitempty"`
}

// PackDependency is one declared inter-pack dependency (Spec 07 R07-4.1.4).
//
// A pack states which other pack it needs and which versions of that pack it
// accepts. Resolution fails when the dependency is absent, when its resolved
// version is outside Range, or when the declared graph is cyclic — and every
// error names the dependency path, because "resolution failed" without the path
// is not a message an operator can act on.
//
// The edge is DECLARED in the pack, never inferred from rule or tool overlap: an
// inferred edge would change meaning whenever a rule moved, so a policy set
// could acquire a dependency without anyone reviewing one.
type PackDependency struct {
	// Pack is the declared pack name (e.g. "core/four-laws"), never a file path:
	// the same identity the lock record and the diagnostic surface use.
	Pack string `json:"pack"`
	// Range constrains the dependency's declared version. The accepted forms are
	// documented on satisfiesVersionRange; an unrecognised form is refused rather
	// than read as "any", because a typo must not widen an accepted range.
	Range string `json:"range"`
}

// PackEdge is one resolved dependency edge. It is reported on the diagnostic
// surface so the graph an operator reads is the graph resolution actually
// walked, not a restatement of the declarations.
type PackEdge struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Range   string `json:"range"`
	Version string `json:"version"`
}

// PackRecord is the lock-record entry for one pack, mirroring the field set the
// CI verifier writes (R07-4.1.3).
type PackRecord struct {
	Path            string `json:"-"`
	Pack            string `json:"pack"`
	Version         string `json:"version"`
	Digest          string `json:"digest"`
	ResolverVersion string `json:"resolver_version"`
	OverlayDigest   string `json:"overlay_digest"`
	ResolvedAt      string `json:"resolved_at"`
}

// lockFile is the on-disk lock-record shape.
type lockFile struct {
	ResolverVersion string                `json:"resolver_version"`
	EffectiveDigest string                `json:"effective_digest"`
	Packs           map[string]PackRecord `json:"packs"`
}

// ResolvedPolicySet is an immutable snapshot of a successfully resolved policy
// set. Its zero value is not active: Active() is the only way to ask.
type ResolvedPolicySet struct {
	digest          string
	resolverVersion string
	packs           []PackRecord
	rules           []PackRule
	ruleToPack      map[string]string
	toolToRule      map[string]PackRule
	edges           []PackEdge
	// Spec 07 R07-4.2 — precedence, composition and the reviewed exceptions.
	// The three are reported separately because "one pack won" and "an operator
	// allowed a weakening" are different facts an audit has to tell apart.
	applications []RuleApplication
	conflicts    []RuleConflict
	exceptions   []Exception
	ruleSetID    string
}

// Active reports whether this snapshot came from a successful resolution.
// A nil or zero set is never active, so a caller that forgets to check gets a
// denial rather than an empty-but-passing policy set.
func (r *ResolvedPolicySet) Active() bool { return r != nil && r.digest != "" }

// EffectiveDigest is the resolved-set identity exposed at startup and through
// the diagnostic resource (R07-4.4.1).
func (r *ResolvedPolicySet) EffectiveDigest() string {
	if r == nil {
		return ""
	}
	return r.digest
}

// ResolverVersion identifies the resolver that produced the set.
func (r *ResolvedPolicySet) ResolverVersion() string {
	if r == nil {
		return ""
	}
	return r.resolverVersion
}

// Packs returns a copy of the resolved pack records.
func (r *ResolvedPolicySet) Packs() []PackRecord {
	if r == nil {
		return nil
	}
	out := make([]PackRecord, len(r.packs))
	copy(out, r.packs)
	return out
}

// Rules returns a copy of the resolved rules.
func (r *ResolvedPolicySet) Rules() []PackRule {
	if r == nil {
		return nil
	}
	out := make([]PackRule, len(r.rules))
	copy(out, r.rules)
	return out
}

// Applications reports how each rule id resolved across tiers (R07-4.2.1),
// including which declarations were overridden and whether an exception was
// needed. It is the precedence decision made observable: "core is mandatory"
// with no way to see an override is a claim, not a control.
func (r *ResolvedPolicySet) Applications() []RuleApplication {
	if r == nil {
		return nil
	}
	out := make([]RuleApplication, len(r.applications))
	copy(out, r.applications)
	return out
}

// Conflicts reports every per-tool rule collision the composition resolved and
// which declaration won (R07-4.2.2). An empty result means no tool was governed
// by more than one effective rule — an answer, not a skipped check.
func (r *ResolvedPolicySet) Conflicts() []RuleConflict {
	if r == nil {
		return nil
	}
	out := make([]RuleConflict, len(r.conflicts))
	copy(out, r.conflicts)
	return out
}

// Exceptions reports the reviewed, unexpired exceptions in force (R07-4.2.3).
func (r *ResolvedPolicySet) Exceptions() []Exception {
	if r == nil {
		return nil
	}
	out := make([]Exception, len(r.exceptions))
	copy(out, r.exceptions)
	return out
}

// RuleSetDigest is the identity of the effective rule set, which every
// exception binds to. It is separate from EffectiveDigest: one identifies the
// inputs (packs, versions, content), the other the resulting decisions.
func (r *ResolvedPolicySet) RuleSetDigest() string {
	if r == nil {
		return ""
	}
	return r.ruleSetID
}

// Dependencies returns the resolved dependency graph in declaration order.
// An empty result means the set is a single node with no declared edges — the
// truthful answer for a one-pack tree, and never a claim that edges were
// impossible.
func (r *ResolvedPolicySet) Dependencies() []PackEdge {
	if r == nil {
		return nil
	}
	out := make([]PackEdge, len(r.edges))
	copy(out, r.edges)
	return out
}

// Attribution reports the pack digest and rule ID that govern a tool, so an
// enforcement decision can name what produced it (R07-4.4.3). ok=false means
// the tool is not covered by the active policy set — which is an answer, not an
// error, and must not be reported as if a rule had matched.
func (r *ResolvedPolicySet) Attribution(tool string) (digest, ruleID string, ok bool) {
	if r == nil || r.toolToRule == nil {
		return "", "", false
	}
	rule, found := r.toolToRule[tool]
	if !found {
		return r.digest, "", false
	}
	return r.digest, rule.ID, true
}

// PackForRule names the pack that declared a rule, or "" when the rule id is
// not in the active set.
func (r *ResolvedPolicySet) PackForRule(ruleID string) string {
	if r == nil || r.ruleToPack == nil {
		return ""
	}
	return r.ruleToPack[ruleID]
}

// PolicyPackManager holds the active snapshot and applies candidate updates
// atomically. A nil manager reports "not configured" rather than "no policy".
type PolicyPackManager struct {
	mu     sync.RWMutex
	active *ResolvedPolicySet
	source string
}

// NewPolicyPackManager returns a manager with no active set. An operator
// configures a pack directory with ActivateFrom; until then the manager reports
// not-configured, which is the truthful state (R07-4.4.4: resolution does not
// imply that controls are active).
func NewPolicyPackManager() *PolicyPackManager {
	return &PolicyPackManager{}
}

// Source is the configured pack directory, or "" when none was configured.
func (m *PolicyPackManager) Source() string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.source
}

// Active returns the current snapshot (nil when none has been activated).
func (m *PolicyPackManager) Active() *ResolvedPolicySet {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// ActivateFrom builds and validates a candidate set from dir, then swaps it in.
//
// A candidate that fails any check returns the error and leaves the previously
// active set exactly as it was (R07-4.4.2). There is no path where a failed
// update clears the active policy — that would turn a typo into "no policy".
func (m *PolicyPackManager) ActivateFrom(dir string, knownTools map[string]bool) error {
	if m == nil {
		return fmt.Errorf("policy pack manager is nil")
	}
	candidate, err := LoadPolicySet(dir, knownTools)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.active = candidate
	m.source = dir
	m.mu.Unlock()
	return nil
}

// LoadPolicySet resolves the packs under dir and verifies them against the
// lock record. It returns a snapshot only when every check passes.
//
// knownTools, when non-nil, is the set of tools the server actually registers.
// A rule naming a tool outside that set is rejected: such a rule can never
// fire, so shipping it would be a policy that reads as enforcement and does
// nothing. An empty map is treated the same as nil (unknown), not as "no tools
// exist" — the caller that has a registry passes a non-empty set and a
// discovered tool missing from it fails.
func LoadPolicySet(dir string, knownTools map[string]bool) (*ResolvedPolicySet, error) {
	return LoadPolicySetAt(dir, knownTools, time.Now())
}

// LoadPolicySetAt is LoadPolicySet with an injected clock. Exception expiry is
// a real decision that must be testable without waiting for a calendar, and a
// test that cannot place the clock past an expiry cannot prove the expiry is
// enforced.
func LoadPolicySetAt(dir string, knownTools map[string]bool, now time.Time) (*ResolvedPolicySet, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("policy pack directory is not configured")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("policy pack directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("policy pack directory %q is not a directory", dir)
	}

	paths, err := discoverPackFiles(dir)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		// Non-vacuity: "the lock matches all packs" is meaningless over an
		// empty set. The CI gate holds the same line.
		return nil, fmt.Errorf("no *.pack.json files under %q; refusing an empty policy set", dir)
	}

	lockPath := filepath.Join(dir, "LOCK.json")
	lock, err := readLockFile(lockPath)
	if err != nil {
		return nil, err
	}

	set := &ResolvedPolicySet{
		resolverVersion: lock.ResolverVersion,
		ruleToPack:      map[string]string{},
		toolToRule:      map[string]PackRule{},
	}
	rows := make([]PackRecord, 0, len(paths))
	packsByName := make(map[string]*Pack, len(paths))
	// Lock-record paths are repository-root relative ("policy-packs/core/x.pack.json")
	// while the loader walks a pack directory ("core/x.pack.json"). Match on the
	// path suffix at a separator boundary and carry the RECORDED path forward, so
	// the effective digest is computed over exactly the keys the CI gate uses.
	lockKeyFor := make(map[string]string, len(paths))
	matched := make(map[string]string, len(paths))
	for _, path := range paths {
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return nil, fmt.Errorf("policy pack path %q: %w", path, relErr)
		}
		rel = filepath.ToSlash(rel)
		key, matchErr := matchLockKey(lock.Packs, rel)
		if matchErr != nil {
			return nil, matchErr
		}
		lockKeyFor[rel] = key
		matched[key] = rel
	}

	for _, path := range paths {
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return nil, fmt.Errorf("policy pack path %q: %w", path, relErr)
		}
		rel = filepath.ToSlash(rel)

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("policy pack %q: %w", rel, readErr)
		}

		pack, parseErr := parsePack(rel, raw, knownTools)
		if parseErr != nil {
			return nil, parseErr
		}

		sum := sha256.Sum256(raw)
		digest := "sha256:" + hex.EncodeToString(sum[:])

		lockKey := lockKeyFor[rel]
		recorded, ok := lock.Packs[lockKey]
		if !ok {
			return nil, fmt.Errorf("policy pack %q has no lock record; re-lock before activating (R07-5.8)", rel)
		}
		if recorded.Digest != digest {
			return nil, fmt.Errorf("policy pack %q content digest is %s but the lock records %s; the pack changed without a re-lock", rel, digest, recorded.Digest)
		}
		if recorded.Pack != pack.Name || recorded.Version != pack.Version {
			return nil, fmt.Errorf("policy pack %q is %s@%s but the lock records %s@%s", rel, pack.Name, pack.Version, recorded.Pack, recorded.Version)
		}

		record := recorded
		record.Path = lockKey
		record.Digest = digest
		rows = append(rows, record)

		// Rule ids are NOT accumulated here. The same id may legally appear in
		// two tiers — that is what precedence exists to resolve — so the
		// effective set is decided by resolvePolicyPrecedence below, which
		// refuses an equal-tier duplicate and reports every override.
		if _, dup := packsByName[pack.Name]; dup {
			return nil, fmt.Errorf("two packs declare the same name %q; dependency edges would be ambiguous", pack.Name)
		}
		packsByName[pack.Name] = pack
	}

	// Stale lock entries are a mismatch in the other direction: a record with no
	// pack means the lock describes a policy set that is not on disk.
	for key := range lock.Packs {
		if _, ok := matched[key]; !ok {
			return nil, fmt.Errorf("lock records %q but no such pack exists on disk", key)
		}
	}

	edges, depErr := resolveDependencies(packsByName)
	if depErr != nil {
		return nil, depErr
	}
	set.edges = edges

	// Spec 07 R07-4.2 — precedence, composition and the exception ledger all
	// run here, after the lock and the dependency graph, because they are
	// decisions about a set that has already been proven to be the set on disk.
	exceptions, excErr := loadExceptions(dir)
	if excErr != nil {
		return nil, excErr
	}
	precedence, precErr := resolvePolicyPrecedence(packsByName, exceptions, now)
	if precErr != nil {
		return nil, precErr
	}
	if validErr := validateExceptions(precedence, now); validErr != nil {
		// The digest is named in the error because an exception has to bind to
		// it: an operator who cannot see the value has to run the failing load
		// again to learn what to write, and the value is already computed here.
		return nil, fmt.Errorf("%w\nrule-set digest for a new exception: %s", validErr, precedence.Digest)
	}
	set.applications = precedence.Applications
	set.rules = precedence.Rules
	set.ruleToPack = precedence.RuleToPack
	set.toolToRule = precedence.ToolToRule
	set.conflicts = precedence.Conflicts
	set.exceptions = precedence.Exceptions
	set.ruleSetID = precedence.Digest

	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
	set.packs = rows

	effective, err := effectiveDigest(rows)
	if err != nil {
		return nil, err
	}
	if lock.EffectiveDigest != effective {
		return nil, fmt.Errorf("lock effective_digest is %s but the resolved set hashes to %s", lock.EffectiveDigest, effective)
	}
	set.digest = effective

	return set, nil
}

// PolicyRuleSetDigest resolves the packs under dir and returns the effective
// rule-set digest without requiring the exception ledger to authorise anything.
//
// This exists because granting an exception is necessarily a two-step act: the
// approver has to bind to the digest of the set they are approving, and that set
// does not yet resolve (that is the point of the exception). Computing the
// digest by hand would mean reimplementing precedence outside the resolver, and
// a hand-computed digest that disagrees by one byte produces an exception that
// can never be valid.
//
// It returns the digest only. Whether the resulting set is ACTIVATABLE is a
// separate question answered by LoadPolicySet, and this function does not
// answer it.
func PolicyRuleSetDigest(dir string, knownTools map[string]bool) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("policy pack directory is not configured")
	}
	paths, err := discoverPackFiles(dir)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("no *.pack.json files under %q; refusing an empty policy set", dir)
	}
	packsByName := map[string]*Pack{}
	for _, path := range paths {
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return "", fmt.Errorf("policy pack path %q: %w", path, relErr)
		}
		rel = filepath.ToSlash(rel)
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", fmt.Errorf("policy pack %q: %w", rel, readErr)
		}
		pack, parseErr := parsePack(rel, raw, knownTools)
		if parseErr != nil {
			return "", parseErr
		}
		if _, dup := packsByName[pack.Name]; dup {
			return "", fmt.Errorf("two packs declare the same name %q; dependency edges would be ambiguous", pack.Name)
		}
		packsByName[pack.Name] = pack
	}
	exceptions, excErr := loadExceptions(dir)
	if excErr != nil {
		return "", excErr
	}
	resolved, resolveErr := resolvePolicyPrecedence(packsByName, exceptions, time.Now())
	if resolveErr != nil {
		return "", resolveErr
	}
	return resolved.Digest, nil
}

// matchLockKey resolves the lock-record key for a pack path relative to the
// configured pack directory. A lock may record repository-root-relative paths
// (the CI gate's form, "policy-packs/core/x.pack.json") or directory-relative
// ones ("core/x.pack.json"); both are accepted, but the match must be unique so
// a permissive prefix can never associate a pack with somebody else's record.
func matchLockKey(packs map[string]PackRecord, rel string) (string, error) {
	if _, ok := packs[rel]; ok {
		return rel, nil
	}
	var found string
	for key := range packs {
		if strings.HasSuffix(key, "/"+rel) {
			if found != "" {
				return "", fmt.Errorf("policy pack %q matches more than one lock record (%q and %q)", rel, found, key)
			}
			found = key
		}
	}
	if found == "" {
		return "", fmt.Errorf("policy pack %q has no lock record; re-lock before activating (R07-5.8)", rel)
	}
	return found, nil
}

func hasPath(rows []PackRecord, key string) bool {
	for _, row := range rows {
		if row.Path == key {
			return true
		}
	}
	return false
}

// effectiveDigest recomputes the resolved-set identity exactly as
// `scripts/check_policy_pack_lock.py` does, so the runtime consumer and the CI
// gate can never disagree about what set they are looking at.
//
// The canonical form is a JSON array of
// [path, pack, version, digest, resolver_version, overlay_digest] rows in path
// order, with compact separators. resolved_at is metadata and is excluded, so
// re-resolution is reproducible (R07-4.1.2).
func effectiveDigest(rows []PackRecord) (string, error) {
	sorted := make([]PackRecord, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	canonical := make([][]string, 0, len(sorted))
	for _, row := range sorted {
		canonical = append(canonical, []string{
			row.Path, row.Pack, row.Version, row.Digest, row.ResolverVersion, row.OverlayDigest,
		})
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode canonical policy set: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func discoverPackFiles(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".pack.json") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan policy packs: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func readLockFile(path string) (*lockFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("policy pack lock %q: %w", path, err)
	}
	var lock lockFile
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&lock); err != nil {
		return nil, fmt.Errorf("policy pack lock %q is not a recognised lock record: %w", path, err)
	}
	if lock.ResolverVersion == "" {
		return nil, fmt.Errorf("policy pack lock %q has no resolver_version", path)
	}
	if lock.EffectiveDigest == "" {
		return nil, fmt.Errorf("policy pack lock %q has no effective_digest", path)
	}
	if len(lock.Packs) == 0 {
		return nil, fmt.Errorf("policy pack lock %q records no packs", path)
	}
	return &lock, nil
}

// packFields and ruleFields mirror the schema the CI gate enforces (R07-4.3.2).
// They are duplicated rather than shared because the gate is a Python script;
// the two are kept in step by the pack schema test.
var packFields = map[string]bool{"pack": true, "version": true, "tier": true, "rules": true, "seed": true, "depends_on": true}

var ruleFields = map[string]bool{
	"id": true, "law": true, "enforcement": true, "tool": true,
	"on_unknown": true, "evidence": true,
}

var enforcementClasses = map[string]bool{"advisory": true, "checked": true, "blocking": true}

// parsePack decodes one pack as inert data and validates it. Nothing here
// executes pack text: a pack that contains shell-looking or
// prompt-injection-looking strings is decoded as strings and goes no further
// (R07-4.3.1 / acceptance 5).
func parsePack(rel string, raw []byte, knownTools map[string]bool) (*Pack, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("policy pack %q is not valid JSON: %w", rel, err)
	}
	for key := range probe {
		if !packFields[key] && !strings.HasPrefix(key, "x-") {
			return nil, fmt.Errorf("policy pack %q has unknown critical field %q (R07-4.3.2)", rel, key)
		}
	}

	var pack Pack
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pack); err != nil {
		return nil, fmt.Errorf("policy pack %q: %w", rel, err)
	}

	if strings.TrimSpace(pack.Name) == "" {
		return nil, fmt.Errorf("policy pack %q has no 'pack' name", rel)
	}
	// The declared name must be exactly what the path implies, the same rule the
	// CI gate applies — otherwise a pack can be relabelled without a re-lock.
	expected := strings.TrimSuffix(rel, ".pack.json")
	if pack.Name != expected {
		return nil, fmt.Errorf("policy pack %q declares pack %q but its path implies %q", rel, pack.Name, expected)
	}
	if !semverPattern(pack.Version) {
		return nil, fmt.Errorf("policy pack %q version %q is not MAJOR.MINOR.PATCH", rel, pack.Version)
	}
	// R07-4.2.1 — the tier is declared, and the declaration is checked against
	// the name. A path prefix is a filing convention an author can get wrong
	// silently; requiring both to agree means a pack cannot be promoted or
	// demoted by moving a file, and a reviewer sees the precedence claim in the
	// diff rather than inferring it from a directory.
	if _, ok := tierRank[pack.Tier]; !ok {
		return nil, fmt.Errorf("policy pack %q declares tier %q; want one of core, domain, overlay (R07-4.2.1)", rel, pack.Tier)
	}
	prefix := strings.SplitN(expected, "/", 2)[0]
	if pack.Tier != prefix {
		return nil, fmt.Errorf("policy pack %q declares tier %q but its path implies %q (R07-4.2.1)", rel, pack.Tier, prefix)
	}
	if len(pack.Rules) == 0 && !pack.Seed {
		return nil, fmt.Errorf("policy pack %q has no rules and is not marked 'seed': true (R07-4.2.4)", rel)
	}

	for i, rule := range pack.Rules {
		at := fmt.Sprintf("policy pack %q rule %d", rel, i)
		if strings.TrimSpace(rule.ID) == "" {
			return nil, fmt.Errorf("%s has no id", at)
		}
		if !enforcementClasses[rule.Enforcement] {
			return nil, fmt.Errorf("%s (%s) has enforcement %q; want one of advisory, checked, blocking", at, rule.ID, rule.Enforcement)
		}
		if rule.Enforcement != "advisory" && strings.TrimSpace(rule.Evidence) == "" {
			// A checked/blocking claim with no evidence is "advisory dressed as
			// enforcement" — the exact failure the pack schema forbids.
			return nil, fmt.Errorf("%s (%s) claims enforcement %q with no 'evidence'", at, rule.ID, rule.Enforcement)
		}
		if rule.OnUnknown != "stop" {
			return nil, fmt.Errorf("%s (%s) has on_unknown %q; only 'stop' is permitted", at, rule.ID, rule.OnUnknown)
		}
		if strings.TrimSpace(rule.Tool) == "" {
			return nil, fmt.Errorf("%s (%s) names no tool", at, rule.ID)
		}
		if len(knownTools) > 0 && !knownTools[rule.Tool] {
			return nil, fmt.Errorf("%s (%s) names tool %q, which the server does not register", at, rule.ID, rule.Tool)
		}
	}

	// R07-4.1.4 — declarations are validated here so a malformed edge never
	// reaches graph resolution; existence, compatibility and cycles need the
	// whole set and are checked in resolveDependencies.
	seenDeps := map[string]bool{}
	for i, dep := range pack.DependsOn {
		at := fmt.Sprintf("policy pack %q depends_on[%d]", rel, i)
		if strings.TrimSpace(dep.Pack) == "" {
			return nil, fmt.Errorf("%s names no pack", at)
		}
		if dep.Pack == pack.Name {
			return nil, fmt.Errorf("%s declares a dependency on itself (%q)", at, dep.Pack)
		}
		if seenDeps[dep.Pack] {
			return nil, fmt.Errorf("%s declares dependency %q twice", at, dep.Pack)
		}
		seenDeps[dep.Pack] = true
		if _, _, err := parseVersionRange(dep.Range); err != nil {
			return nil, fmt.Errorf("%s on %q has an unusable range: %w", at, dep.Pack, err)
		}
	}

	return &pack, nil
}

// semverPattern reports whether v is MAJOR.MINOR.PATCH in digits.
func semverPattern(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// semverParts splits a validated MAJOR.MINOR.PATCH into comparable integers.
// semverPattern has already rejected anything else, so a parse failure here is
// impossible and is reported rather than ignored.
func semverParts(v string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("%q is not MAJOR.MINOR.PATCH", v)
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, fmt.Errorf("%q is not numeric in %q", part, v)
		}
		out[i] = n
	}
	return out, nil
}

func compareSemver(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return 0
}

// parseVersionRange validates a dependency range and returns its operator and
// bound version. Syntax is checked separately from comparison so a malformed
// range is refused at pack-parse time, before any target pack is known.
func parseVersionRange(rng string) (string, [3]int, error) {
	var zero [3]int
	trimmed := strings.TrimSpace(rng)
	if trimmed == "" {
		return "", zero, fmt.Errorf("no version range declared (R07-4.1.4); write an exact version or >=/^/~ before it")
	}
	op := "="
	spec := trimmed
	for _, candidate := range []string{">=", "^", "~"} {
		if strings.HasPrefix(trimmed, candidate) {
			op = candidate
			spec = strings.TrimSpace(strings.TrimPrefix(trimmed, candidate))
			break
		}
	}
	if !semverPattern(spec) {
		return "", zero, fmt.Errorf("range %q is not an exact MAJOR.MINOR.PATCH or one of >=/^/~ followed by one", rng)
	}
	want, err := semverParts(spec)
	if err != nil {
		return "", zero, err
	}
	return op, want, nil
}

// satisfiesVersionRange reports whether version meets rng (Spec 07 R07-4.1.4).
//
// Accepted forms, deliberately few: an exact MAJOR.MINOR.PATCH, ">=X", "^X"
// (same major, at least X) and "~X" (same major.minor, at least X). Anything
// else — including "", "*" and "latest" — is an ERROR, not "any version": a
// dependency whose constraint silently widens accepts a breaking change without
// a review, which is the failure this spec exists to prevent.
func satisfiesVersionRange(version, rng string) (bool, error) {
	op, want, err := parseVersionRange(rng)
	if err != nil {
		return false, err
	}
	if !semverPattern(version) {
		return false, fmt.Errorf("declared version %q is not MAJOR.MINOR.PATCH", version)
	}
	got, err := semverParts(version)
	if err != nil {
		return false, err
	}
	cmp := compareSemver(got, want)
	switch op {
	case "=":
		return cmp == 0, nil
	case ">=":
		return cmp >= 0, nil
	case "^":
		return got[0] == want[0] && cmp >= 0, nil
	case "~":
		return got[0] == want[0] && got[1] == want[1] && cmp >= 0, nil
	}
	return false, fmt.Errorf("unreachable range operator %q", op)
}

// resolveDependencies validates the declared inter-pack graph (R07-4.1.4) and
// returns its edges in deterministic order.
//
// Three failures, each naming the dependency PATH, because an operator cannot
// act on "resolution failed":
//   - a declared dependency is not in the resolved set;
//   - the resolved version is outside the declared range;
//   - the graph is cyclic (the cycle is printed as the path that closes it).
//
// A single-pack set with no declarations is a legal one-node graph, not a
// skipped check: the same walk runs and finds nothing to reject.
func resolveDependencies(packsByName map[string]*Pack) ([]PackEdge, error) {
	names := make([]string, 0, len(packsByName))
	for name := range packsByName {
		names = append(names, name)
	}
	sort.Strings(names)

	var edges []PackEdge
	for _, from := range names {
		pack := packsByName[from]
		for _, dep := range pack.DependsOn {
			target, ok := packsByName[dep.Pack]
			if !ok {
				return nil, fmt.Errorf("policy pack %q depends on %q, which is not in the resolved set (dependency path: %s -> %s)",
					from, dep.Pack, from, dep.Pack)
			}
			okVersion, err := satisfiesVersionRange(target.Version, dep.Range)
			if err != nil {
				return nil, fmt.Errorf("policy pack %q dependency %q: %w", from, dep.Pack, err)
			}
			if !okVersion {
				return nil, fmt.Errorf("policy pack %q requires %q %s but the resolved set has %s (dependency path: %s -> %s)",
					from, dep.Pack, dep.Range, target.Version, from, dep.Pack)
			}
			edges = append(edges, PackEdge{From: from, To: dep.Pack, Range: dep.Range, Version: target.Version})
		}
	}

	if err := detectDependencyCycles(packsByName, names); err != nil {
		return nil, err
	}
	return edges, nil
}

// detectDependencyCycles runs a depth-first walk with an explicit path stack so
// a cycle error can print the path that closes it rather than naming two nodes.
func detectDependencyCycles(packsByName map[string]*Pack, names []string) error {
	const (
		unvisited = 0
		onPath    = 1
		done      = 2
	)
	state := make(map[string]int, len(packsByName))
	var stack []string

	var walk func(name string) error
	walk = func(name string) error {
		switch state[name] {
		case done:
			return nil
		case onPath:
			// The cycle is the tail of the stack from name, closed by name.
			start := 0
			for i, node := range stack {
				if node == name {
					start = i
					break
				}
			}
			cycle := append(append([]string{}, stack[start:]...), name)
			return fmt.Errorf("policy pack dependency cycle: %s", strings.Join(cycle, " -> "))
		}
		state[name] = onPath
		stack = append(stack, name)

		pack := packsByName[name]
		if pack != nil {
			for _, dep := range pack.DependsOn {
				if _, ok := packsByName[dep.Pack]; !ok {
					continue // already reported as an absent dependency
				}
				if err := walk(dep.Pack); err != nil {
					return err
				}
			}
		}

		stack = stack[:len(stack)-1]
		state[name] = done
		return nil
	}

	for _, name := range names {
		if err := walk(name); err != nil {
			return err
		}
	}
	return nil
}
