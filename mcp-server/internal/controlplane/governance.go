package controlplane

// Governance: staged adoption, promotion, atomic rollback and the root-change
// audit record (Spec 08 R08-4.4.2 .. R08-4.4.4, acceptance criterion 8).
//
// The rest of this package answers "what is composed and what did it prove".
// This file answers a question a report cannot: *was the system allowed to be
// in this mode at all*, and can it be put back. Three rules are structural:
//
//  1. A deployment does not become blocking by configuration alone
//     (R08-4.4.2). Promotion requires a reviewed change reference, an explicit
//     profile version, and a passing acceptance bundle whose manifest and
//     policy digests are the ones the process is actually running. A bundle
//     from another composition is evidence about a different system.
//  2. A change to a governed root — the policy digest, a component's trust
//     tier, its mandatory/optional class, its version or digest, or its
//     dependencies — may not be adopted without an authorised human review, and
//     every adoption writes a record naming what changed (R08-4.4.3). The
//     review is checked against the *computed difference*, never against a
//     field the caller supplies to describe it: a caller that says "no review
//     needed" is not asked to describe the change it wants to make.
//  3. Rollback restores the prior manifest and policy digest as one value
//     (R08-4.4.4). The prior digest is recomputed after the swap, so a
//     manifest mutated in place — the failure a shared pointer invites — is
//     caught rather than reported as a successful restore. A rollback that
//     cannot be completed leaves the instance NOT READY: serving on a
//     candidate the operator has already decided to abandon is the one
//     outcome a rollback exists to prevent.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Approval is the authorised human review R08-4.4.3 requires for a change to a
// governed root. Every field is required: a review reference without an
// accountable person is an audit row nobody can follow up, and a name without
// a reference is a name attached to nothing.
type Approval struct {
	ReviewRef  string
	ApprovedBy string
	ApprovedAt time.Time
}

func (a Approval) validate() error {
	if strings.TrimSpace(a.ReviewRef) == "" {
		return errors.New("a change to a governed root requires a review reference")
	}
	if strings.TrimSpace(a.ApprovedBy) == "" {
		return errors.New("a change to a governed root requires an accountable approver")
	}
	if a.ApprovedAt.IsZero() {
		return errors.New("a change to a governed root requires an approval time")
	}
	return nil
}

// Record kinds written to the audit trail. They are stable strings because a
// record is read by a human reconstructing what happened and by a script
// diffing two states.
const (
	RecordAdopt    = "adopt"
	RecordPromote  = "promote"
	RecordRollback = "rollback"
)

// Record is one entry in the governance audit trail: enough to reconstruct the
// change without storing policy text or credential material.
type Record struct {
	Kind                string    `json:"kind"`
	At                  time.Time `json:"at"`
	ManifestDigest      string    `json:"manifest_digest"`
	PriorManifestDigest string    `json:"prior_manifest_digest,omitempty"`
	PolicyDigest        string    `json:"policy_digest,omitempty"`
	ProfileVersion      string    `json:"profile_version,omitempty"`
	ReviewRef           string    `json:"review_ref,omitempty"`
	ApprovedBy          string    `json:"approved_by,omitempty"`
	Changes             []string  `json:"changes,omitempty"`
	Reason              string    `json:"reason,omitempty"`
}

// State is the persisted governance posture. It is a file because a promotion
// that does not survive a restart is a promotion for one process, and the
// whole point of a blocking mode is that it holds until it is reviewed away.
type State struct {
	// ObserveOnly is true until a reviewed promotion succeeds. The zero value
	// is observe-only, so a missing or truncated state file cannot make a
	// deployment blocking.
	ObserveOnly bool `json:"observe_only"`
	// ManifestDigest binds this state to ONE composition. A state file that
	// names a different manifest does not apply: the deployment is read as
	// observe-only and says why, rather than inheriting a promotion granted to
	// a composition it is not running.
	ManifestDigest string `json:"manifest_digest"`
	PolicyDigest   string `json:"policy_digest"`
	// ProfileVersion is the version the promotion was granted against.
	ProfileVersion string `json:"profile_version,omitempty"`
	// RollbackFailed latches. Once a rollback could not be completed the
	// instance stays not-ready even if a later attempt succeeds, until an
	// operator clears it deliberately; a transient "it worked the second
	// time" must not erase the fact that the system ran in an unknown state.
	RollbackFailed   bool     `json:"rollback_failed"`
	RollbackFailures []string `json:"rollback_failures,omitempty"`
	Records          []Record `json:"records,omitempty"`
}

// unit is a manifest and the digest it was adopted under. The two are one
// value on purpose: a rollback that restores a manifest but keeps the
// candidate's digest is the partial rollback R08-4.4.4 refuses.
type unit struct {
	manifest *Manifest
	digest   string
}

// Governance holds the active composition and the posture around it. It is
// safe for concurrent use: the resource and the readiness probe read it while
// an operator may promote or roll back.
type Governance struct {
	mu sync.Mutex

	active *unit
	prior  *unit
	state  State

	// path is the state file, empty for an in-memory governance (tests, and a
	// deployment that asks for no persistence).
	path string
	// clock is injectable for deterministic records.
	clock func() time.Time
}

// NewGovernance creates governance over a resolved manifest. The initial
// posture is observe-only: adopting a composition is not a licence to block on
// it, and the process must be promoted by a reviewed change.
func NewGovernance(m *Manifest) (*Governance, error) {
	if m == nil {
		return nil, errors.New("governance requires a resolved manifest")
	}
	if err := m.Resolve(); err != nil {
		return nil, err
	}
	clone := cloneManifest(m)
	return &Governance{
		active: &unit{manifest: clone, digest: clone.Digest()},
		state: State{
			ObserveOnly:    true,
			ManifestDigest: clone.Digest(),
			PolicyDigest:   clone.PolicyDigest,
		},
		clock: time.Now,
	}, nil
}

// OpenGovernance loads governance over a manifest and a state file, creating
// the file when absent. An absent file means observe-only (never blocking). A
// malformed file is an ERROR: the caller must not silently fall back to a
// posture nobody can read, because "unreadable" and "observe-only" are
// different answers and only one of them is safe to guess.
func OpenGovernance(manifestPath, statePath string) (*Governance, error) {
	m, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	return OpenGovernanceWithManifest(m, statePath)
}

// OpenGovernanceWithManifest opens governance over an already-resolved
// manifest and a state file. It exists because a caller that resolved the
// manifest already (the server does) must not resolve it twice and risk two
// different answers about the same file.
func OpenGovernanceWithManifest(m *Manifest, statePath string) (*Governance, error) {
	g, err := NewGovernance(m)
	if err != nil {
		return nil, err
	}
	g.path = statePath
	st, err := readState(statePath)
	if err != nil {
		return nil, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if st != nil {
		g.state = *st
		if g.state.ManifestDigest != g.active.digest {
			g.state.ObserveOnly = true
			g.state.ProfileVersion = ""
			g.state.Records = append(g.state.Records, Record{
				Kind:                RecordAdopt,
				At:                  g.clock().UTC(),
				ManifestDigest:      g.active.digest,
				PriorManifestDigest: g.state.ManifestDigest,
				PolicyDigest:        g.active.manifest.PolicyDigest,
				Reason:              "state file names a different composition; promotion does not transfer",
			})
			g.state.ManifestDigest = g.active.digest
			g.state.PolicyDigest = g.active.manifest.PolicyDigest
		}
	}
	return g, nil
}

// Active returns a copy of the active manifest. A copy, because a caller that
// mutates the manifest it was handed would change the composition without an
// adoption, an approval or a record.
func (g *Governance) Active() *Manifest {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.active == nil {
		return nil
	}
	return cloneManifest(g.active.manifest)
}

// Mode reports the current posture: "observe_only" or "blocking".
func (g *Governance) Mode() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state.ObserveOnly {
		return "observe_only"
	}
	return "blocking"
}

// ProfileVersion reports the reviewed profile revision a promotion was granted
// against. It is empty in observe-only mode: an observe-only deployment has no
// promoted revision to name.
func (g *Governance) ProfileVersion() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state.ObserveOnly {
		return ""
	}
	return g.state.ProfileVersion
}

// RollbackBlocked reports why this instance may not be considered ready. It is
// the readiness contribution of R08-4.4.4: a rollback that could not restore
// the prior unit leaves the process in an unknown composition.
func (g *Governance) RollbackBlocked() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.state.RollbackFailed {
		return nil
	}
	if len(g.state.RollbackFailures) == 0 {
		return []string{"a rollback did not complete and the composition is unverified"}
	}
	out := make([]string, len(g.state.RollbackFailures))
	copy(out, g.state.RollbackFailures)
	return out
}

// Records returns a copy of the audit trail, oldest first.
func (g *Governance) Records() []Record {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Record, len(g.state.Records))
	copy(out, g.state.Records)
	return out
}

// Adopt makes a candidate composition active. A change to a governed root
// requires an Approval; every adoption writes a record naming what changed.
//
// The candidate is resolved before anything is swapped, so an invalid
// composition cannot become active and then be reported as a resolution
// failure — the failure belongs to the adoption, not to the running system.
func (g *Governance) Adopt(candidate *Manifest, ap Approval) ([]string, error) {
	if candidate == nil {
		return nil, errors.New("adoption requires a candidate manifest")
	}
	if err := candidate.Resolve(); err != nil {
		return nil, err
	}
	clone := cloneManifest(candidate)

	g.mu.Lock()
	defer g.mu.Unlock()

	changes := RequiresReview(g.active.manifest, clone)
	if len(changes) > 0 {
		if err := ap.validate(); err != nil {
			sort.Strings(changes)
			return changes, fmt.Errorf("%w (this change touches: %s)", err, strings.Join(changes, "; "))
		}
	}

	prior := g.active
	next := &unit{manifest: clone, digest: clone.Digest()}

	// One assignment: the manifest and the digest cannot disagree between the
	// two lines below.
	g.prior = prior
	g.active = next
	g.state.ManifestDigest = next.digest
	g.state.PolicyDigest = next.manifest.PolicyDigest
	// A new composition has not been promoted, whatever the old one had.
	g.state.ObserveOnly = true
	g.state.ProfileVersion = ""
	g.state.RollbackFailed = false
	g.state.RollbackFailures = nil

	sort.Strings(changes)
	g.state.Records = append(g.state.Records, Record{
		Kind:                RecordAdopt,
		At:                  g.clock().UTC(),
		ManifestDigest:      next.digest,
		PriorManifestDigest: prior.digest,
		PolicyDigest:        next.manifest.PolicyDigest,
		ReviewRef:           ap.ReviewRef,
		ApprovedBy:          ap.ApprovedBy,
		Changes:             changes,
	})
	if len(changes) > 0 {
		// A governed-root change must not survive in memory only.
		if err := g.persistLocked(); err != nil {
			return changes, err
		}
	}
	return changes, nil
}

// PromotionRequest is the evidence R08-4.4.2 requires before a profile may
// block.
type PromotionRequest struct {
	// ProfileVersion names the profile revision being promoted. It must be the
	// version of the active manifest: a promotion naming a profile the process
	// is not running promotes something else.
	ProfileVersion string
	// ReviewRef and ApprovedBy identify the reviewed change.
	ReviewRef  string
	ApprovedBy string
	// Acceptance is the passing acceptance suite. Observe-only evidence cannot
	// promote: it is a report about what would happen under enforcement.
	Acceptance *Bundle
}

// Promote moves the deployment from observe-only to blocking (R08-4.4.2).
//
// Every refusal here names the field or the reason, because "promotion failed"
// sends an operator to the logs while "the acceptance bundle describes manifest
// sha256:… but this process runs sha256:…" sends them to the actual mistake.
func (g *Governance) Promote(req PromotionRequest) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.active == nil {
		return errors.New("no composition is active; there is nothing to promote")
	}
	if !g.state.ObserveOnly {
		return errors.New("the deployment is already blocking")
	}
	if strings.TrimSpace(req.ReviewRef) == "" {
		return errors.New("promotion requires a review reference")
	}
	if strings.TrimSpace(req.ApprovedBy) == "" {
		return errors.New("promotion requires an accountable approver")
	}
	if req.ProfileVersion == "" {
		return errors.New("promotion requires an explicit profile version")
	}
	if req.ProfileVersion != g.active.manifest.Version {
		return fmt.Errorf("promotion names profile version %q but the active manifest is version %q",
			req.ProfileVersion, g.active.manifest.Version)
	}
	if req.Acceptance == nil {
		return errors.New("promotion requires a passing acceptance suite")
	}
	// One implementation of "may this bundle satisfy a blocking gate": the
	// same Enforce() a release gate uses, so the two cannot disagree.
	if err := req.Acceptance.Enforce(); err != nil {
		return fmt.Errorf("acceptance suite does not satisfy a blocking gate: %w", err)
	}
	if req.Acceptance.ManifestDigest != g.active.digest {
		return fmt.Errorf("acceptance bundle was produced under manifest %s but this process runs %s",
			req.Acceptance.ManifestDigest, g.active.digest)
	}
	if req.Acceptance.PolicyDigest != g.active.manifest.PolicyDigest {
		return fmt.Errorf("acceptance bundle was produced under policy %s but this process runs %s",
			req.Acceptance.PolicyDigest, g.active.manifest.PolicyDigest)
	}
	if req.Acceptance.Profile != g.active.manifest.Profile {
		return fmt.Errorf("acceptance bundle profile %q is not the active profile %q",
			req.Acceptance.Profile, g.active.manifest.Profile)
	}

	g.state.ObserveOnly = false
	g.state.ProfileVersion = req.ProfileVersion
	g.state.Records = append(g.state.Records, Record{
		Kind:           RecordPromote,
		At:             g.clock().UTC(),
		ManifestDigest: g.active.digest,
		PolicyDigest:   g.active.manifest.PolicyDigest,
		ProfileVersion: req.ProfileVersion,
		ReviewRef:      req.ReviewRef,
		ApprovedBy:     req.ApprovedBy,
		// The evidence this promotion rests on, by identity rather than by a
		// claim that it was green.
		Changes: []string{
			"evaluation=" + req.Acceptance.EvaluationID,
			"bundle_digest=" + req.Acceptance.Digest,
			"verdict=" + string(req.Acceptance.Verdict),
		},
	})
	return g.persistLocked()
}

// Rollback restores the prior composition (R08-4.4.4).
//
// The prior unit is verified after the swap rather than trusted: a manifest
// pointer that was mutated in place would otherwise restore "the same object"
// and report success while the composition is the mutated one. If verification
// fails — or there is no prior unit to restore — the failure latches and the
// instance reports not-ready until an operator clears it.
func (g *Governance) Rollback(reason string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	fail := func(format string, args ...any) error {
		msg := fmt.Sprintf(format, args...)
		g.state.RollbackFailed = true
		g.state.RollbackFailures = append(g.state.RollbackFailures, msg)
		_ = g.persistLocked()
		return errors.New(msg)
	}

	if g.prior == nil {
		return fail("no prior composition is recorded, so the rollback target does not exist")
	}
	if got := g.prior.manifest.Digest(); got != g.prior.digest {
		return fail("prior composition digest changed before restore: recorded %s, recomputed %s",
			g.prior.digest, got)
	}

	prior, candidate := g.prior, g.active
	g.active, g.prior = prior, candidate

	// Verify the restored unit, not the intent.
	if got := g.active.manifest.Digest(); got != g.active.digest {
		return fail("restored composition digest is %s but the recorded digest is %s",
			got, g.active.digest)
	}

	g.state.ManifestDigest = g.active.digest
	g.state.PolicyDigest = g.active.manifest.PolicyDigest
	// A restored composition is one somebody rolled back TO. It does not
	// inherit the promotion of the composition that was rolled back.
	g.state.ObserveOnly = true
	g.state.ProfileVersion = ""
	g.state.Records = append(g.state.Records, Record{
		Kind:                RecordRollback,
		At:                  g.clock().UTC(),
		ManifestDigest:      g.active.digest,
		PriorManifestDigest: candidate.digest,
		PolicyDigest:        g.active.manifest.PolicyDigest,
		Reason:              reason,
	})
	return g.persistLocked()
}

// ClearRollbackFailure removes the latched not-ready state. It is a deliberate
// operator action, never automatic: the latch exists so a transient success
// cannot erase that the system ran unverified.
func (g *Governance) ClearRollbackFailure(reviewRef string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if strings.TrimSpace(reviewRef) == "" {
		return errors.New("clearing a rollback failure requires a review reference")
	}
	if !g.state.RollbackFailed {
		return nil
	}
	g.state.RollbackFailed = false
	g.state.RollbackFailures = nil
	g.state.Records = append(g.state.Records, Record{
		Kind:           "clear_rollback_failure",
		At:             g.clock().UTC(),
		ManifestDigest: g.state.ManifestDigest,
		ReviewRef:      reviewRef,
	})
	return g.persistLocked()
}

// RequiresReview computes the governed-root differences between two
// compositions (R08-4.4.3). It is computed, never described by the caller.
//
// Scope is the list R08-4.5.2 protects: the policy root, component trust, and
// the mandatory control list. Version, digest and dependency changes are
// included because each silently changes which code a mandatory control runs.
// Exception allowlists have no representation in this manifest; that is a
// named limit, not an omission the check could have covered.
func RequiresReview(prior, next *Manifest) []string {
	var changes []string
	add := func(format string, args ...any) {
		changes = append(changes, fmt.Sprintf(format, args...))
	}

	if prior == nil || next == nil {
		return []string{"a composition is missing"}
	}
	if prior.PolicyDigest != next.PolicyDigest {
		add("policy_digest: %s -> %s", prior.PolicyDigest, next.PolicyDigest)
	}
	if prior.Profile != next.Profile {
		add("profile: %s -> %s", prior.Profile, next.Profile)
	}
	if strings.Join(sortedCopy(prior.SupportedInterfaces), ",") !=
		strings.Join(sortedCopy(next.SupportedInterfaces), ",") {
		add("supported_interfaces: %s -> %s",
			strings.Join(sortedCopy(prior.SupportedInterfaces), ","),
			strings.Join(sortedCopy(next.SupportedInterfaces), ","))
	}

	byID := func(m *Manifest) map[string]Component {
		out := make(map[string]Component, len(m.Components))
		for _, c := range m.Components {
			out[c.ID] = c
		}
		return out
	}
	oldComps, newComps := byID(prior), byID(next)

	for _, id := range sortedKeys(oldComps) {
		old := oldComps[id]
		nw, ok := newComps[id]
		if !ok {
			add("component %q removed (mandatory=%t, trust=%s)", id, old.Mandatory, old.TrustTier)
			continue
		}
		if old.TrustTier != nw.TrustTier {
			add("component %q trust_tier: %s -> %s", id, old.TrustTier, nw.TrustTier)
		}
		if old.Mandatory != nw.Mandatory {
			add("component %q mandatory: %t -> %t", id, old.Mandatory, nw.Mandatory)
		}
		if old.Version != nw.Version {
			add("component %q version: %s -> %s", id, old.Version, nw.Version)
		}
		if old.Digest != nw.Digest {
			add("component %q digest changed", id)
		}
		if strings.Join(sortedCopy(old.DependsOn), ",") != strings.Join(sortedCopy(nw.DependsOn), ",") {
			add("component %q depends_on: [%s] -> [%s]", id,
				strings.Join(sortedCopy(old.DependsOn), ","),
				strings.Join(sortedCopy(nw.DependsOn), ","))
		}
	}
	for _, id := range sortedKeys(newComps) {
		if _, ok := oldComps[id]; !ok {
			nw := newComps[id]
			add("component %q added (mandatory=%t, trust=%s)", id, nw.Mandatory, nw.TrustTier)
		}
	}
	sort.Strings(changes)
	return changes
}

// persistLocked writes the state file. It is called with mu held.
func (g *Governance) persistLocked() error {
	if g.path == "" {
		return nil
	}
	return writeState(g.path, g.state)
}

// readState reads a state file. A missing file is not an error: it means no
// promotion was ever granted, which is observe-only.
func readState(path string) (*State, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read control-plane state: %w", err)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil, fmt.Errorf("control-plane state file %s is empty", path)
	}
	var st State
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	// An unknown field is refused: a state file carrying a key this build does
	// not understand was written by a build with a different contract, and
	// guessing which posture it meant is how a typo becomes a mode.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&st); err != nil {
		return nil, fmt.Errorf("decode control-plane state: %w", err)
	}
	return &st, nil
}

// writeState writes the state atomically at mode 0600. A partially written
// state file is a posture nobody can read, and the temp+rename pair makes that
// impossible rather than unlikely.
func writeState(path string, st State) error {
	payload, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode control-plane state: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create control-plane state directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".control-plane-state-*")
	if err != nil {
		return fmt.Errorf("create control-plane state temp file: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("set control-plane state mode: %w", err)
	}
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		return fmt.Errorf("write control-plane state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync control-plane state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close control-plane state: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace control-plane state: %w", err)
	}
	return nil
}

// cloneManifest deep-copies a manifest through its JSON form, so a caller can
// neither mutate the active composition nor share a slice with it.
func cloneManifest(m *Manifest) *Manifest {
	if m == nil {
		return nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var out Manifest
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return &out
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
