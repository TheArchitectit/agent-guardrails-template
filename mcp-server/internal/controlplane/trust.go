package controlplane

// Spec 08 R08-4.5 — trust labels and policy-input provenance.
//
// R08-4.5.1 requires that "policy inputs, component output, user content,
// external content, and evidence metadata MUST have distinct trust labels".
// Before this file the repository had a trust TIER per component (used to
// order dependencies) and nothing else: a piece of text flowing into an
// evaluation carried no label at all, so a consumer could not tell a policy
// input from a user prompt, and an evidence bundle could not say which class
// of input produced a verdict.
//
// The distinction matters because the classes are not interchangeable. A
// policy input is authority: it may change what a decision means. User content
// is a subject: it may be judged and must never be obeyed. Conflating them is
// how a document that says "ignore all previous instructions" becomes policy.
//
// Three rules give the labels weight:
//
//   - a label is set by the EVALUATOR from the role the caller declared, not
//     read out of the content itself, so content cannot label itself "policy"
//     and gain authority by assertion;
//   - an unrecognized label fails closed with an explicit reason rather than
//     defaulting to the least dangerous class (R08-4.5.5);
//   - the label travels with the input into the bundle, so evidence says which
//     class of input produced each verdict.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// TrustLabel is the evaluated class of a piece of input (R08-4.5.1).
//
// The five values are the five the requirement names. They are deliberately
// not ordered: "which is more trusted" is a question about a COMPONENT (see
// TrustTier), and a ranking over input classes would invite a caller to
// compare a policy input to user content and infer authority from the
// comparison.
type TrustLabel string

const (
	// LabelPolicyInput is authority: a resolved policy pack, a rule set, a
	// locked configuration whose digest is recorded. It may decide.
	LabelPolicyInput TrustLabel = "policy_input"
	// LabelComponentOutput is the product of a component. It is data a
	// decision may read, never an instruction the evaluator obeys
	// (R08-4.5.3).
	LabelComponentOutput TrustLabel = "component_output"
	// LabelUserContent is the subject of an evaluation. It is judged and
	// never privileged, whatever it says about itself.
	LabelUserContent TrustLabel = "user_content"
	// LabelExternalContent came from outside the trust boundary — a fetched
	// document, a webhook body, a third-party API response.
	LabelExternalContent TrustLabel = "external_content"
	// LabelEvidenceMetadata describes evidence rather than acting as input:
	// timestamps, ids, digests, retention notes.
	LabelEvidenceMetadata TrustLabel = "evidence_metadata"
)

// Valid reports whether the label is one this evaluator understands.
func (l TrustLabel) Valid() bool {
	switch l {
	case LabelPolicyInput, LabelComponentOutput, LabelUserContent,
		LabelExternalContent, LabelEvidenceMetadata:
		return true
	}
	return false
}

// Authoritative reports whether this class may change what a decision means.
// Only a policy input may. Everything else is judged or recorded.
func (l TrustLabel) Authoritative() bool { return l == LabelPolicyInput }

// AllTrustLabels lists every legal label in a fixed order, so an error message
// and a documentation test can enumerate them without drifting.
func AllTrustLabels() []TrustLabel {
	return []TrustLabel{
		LabelPolicyInput, LabelComponentOutput, LabelUserContent,
		LabelExternalContent, LabelEvidenceMetadata,
	}
}

// PolicyInput is one policy input actually used by an evaluation, with the
// source and digest R08-4.5.4 requires.
//
// The field set mirrors the policy lock record on purpose. `Source` is the pack
// path, `ID` the pack name, and `ResolverVersion`/`OverlayDigest` the remaining
// lock columns, so `effectivePolicyDigest` below recomputes the SAME digest
// `scripts/check_policy_pack_lock.py` writes. A provenance record that hashed
// its own convenient shape would be a second opinion about which policy is
// loaded, and two opinions is exactly how a runtime and a gate come to
// disagree.
type PolicyInput struct {
	// ID is the pack name as the lock records it (e.g. "core/four-laws").
	ID string `json:"id"`
	// Source is the pack path as the lock records it.
	Source string `json:"source"`
	// Version pins the input. A mutable ref cannot identify what was used.
	Version string `json:"version"`
	// Digest is the content digest, in the lock's canonical prefixed form.
	Digest string `json:"digest"`
	// ResolverVersion is the resolver that produced the record.
	ResolverVersion string `json:"resolver_version"`
	// OverlayDigest is the overlay the record was resolved with ("none" when
	// no overlay applied), as the lock writes it.
	OverlayDigest string `json:"overlay_digest"`
	// Label is assigned by the evaluator from the declared role (R08-4.5.1).
	Label TrustLabel `json:"label"`
}

// ProvenanceStatus answers "did this evaluation record its policy inputs?"
// It is kept separate from the verdict because "no inputs were declared" and
// "the inputs matched" are different claims, and an evidence bundle that
// blurred them would let an unverified run read as a verified one.
type ProvenanceStatus string

const (
	// ProvenanceRecorded means policy inputs were supplied and their
	// effective identity matched the manifest's declared policy digest.
	ProvenanceRecorded ProvenanceStatus = "recorded"
	// ProvenanceNotDeclared means the caller supplied no policy inputs. The
	// evaluation says so rather than claiming provenance it does not have.
	ProvenanceNotDeclared ProvenanceStatus = "not_declared"
	// ProvenanceMismatch means the supplied inputs do not hash to the
	// manifest's policy digest. The process is running a policy set it did
	// not declare, so its verdict is about a different system.
	ProvenanceMismatch ProvenanceStatus = "mismatch"
	// ProvenanceUnverifiable means inputs were supplied in a form whose
	// identity cannot be recomputed (a missing lock column). It is not a
	// pass and not a mismatch: nothing was proven either way.
	ProvenanceUnverifiable ProvenanceStatus = "unverifiable"
)

// policyInputFingerprints reduces a policy-input set to one comparable string
// per input id. It is used by replay comparison, where the question is "did
// this evaluation use the same inputs?" rather than "were they valid?" —
// validity was already enforced at evaluation time.
func policyInputFingerprints(inputs []PolicyInput) map[string]string {
	out := make(map[string]string, len(inputs))
	for _, in := range inputs {
		out[in.ID] = in.Source + "|" + in.Version + "|" + in.Digest + "|" + string(in.Label)
	}
	return out
}

// sortedInputIDs returns the union of two fingerprint maps' ids in sorted
// order, so a replay comparison reports differences deterministically instead
// of in Go's random map order.
func sortedInputIDs(a, b map[string]string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for id := range a {
		seen[id] = true
	}
	for id := range b {
		seen[id] = true
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Recommendation is a component-provided suggestion (R08-4.5.3).
//
// The requirement is one sentence and it is the whole design: "Component-provided
// recommendations MUST be data for a decision, not executable instructions."
// Two things make that true rather than aspirational:
//
//   - `Text` is never parsed, evaluated, templated or executed. It is recorded
//     and returned as a string. There is no code path from this field to an
//     action, so there is nothing for an injection to reach.
//   - A recommendation that ASKS for authority — a change to the policy
//     digest, a component's trust tier, the mandatory set, or the manifest —
//     is refused and recorded as refused. A lower-trust component suggesting
//     that it be trusted is precisely the move R08-4.5.2 forbids, and a
//     recommendation channel is the obvious place to attempt it.
type Recommendation struct {
	// ComponentID is the component that produced it. Required: an unattributed
	// recommendation cannot be weighed against its origin's trust tier, and a
	// channel open to anonymous suggestions is open to non-components.
	ComponentID string `json:"component_id"`
	// Text is the suggestion, recorded verbatim and never executed.
	Text string `json:"text"`
	// Target names a governed root the recommendation asks to change, if any.
	// Empty means it asks for nothing structural.
	Target string `json:"target,omitempty"`
}

// Governed roots a recommendation may not touch (R08-4.5.2).
const (
	TargetNone           = ""
	TargetPolicyDigest   = "policy_digest"
	TargetTrustTier      = "trust_tier"
	TargetMandatorySet   = "mandatory_set"
	TargetManifest       = "manifest"
	TargetTrustedKeySet  = "trusted_key_set"
	TargetPolicyLock     = "policy_lock"
	TargetExceptionAllow = "exception_allowlist"
)

// RecommendationRecord is the evaluated form: what was suggested, from whom,
// and what the evaluator did with it.
type RecommendationRecord struct {
	ComponentID string    `json:"component_id"`
	TrustTier   TrustTier `json:"trust_tier"`
	Text        string    `json:"text"`
	Target      string    `json:"target,omitempty"`
	// Applied is always false and exists so a reader does not have to infer
	// that a recorded recommendation changed nothing. No code path sets it.
	Applied bool `json:"applied"`
	// Refused is true when the recommendation asked for authority it cannot
	// have. The reason is in Refusal.
	Refused bool   `json:"refused"`
	Refusal string `json:"refusal,omitempty"`
}

// governedTargets is the closed set of roots a recommendation may not modify.
func governedTargets() map[string]bool {
	return map[string]bool{
		TargetPolicyDigest:   true,
		TargetTrustTier:      true,
		TargetMandatorySet:   true,
		TargetManifest:       true,
		TargetTrustedKeySet:  true,
		TargetPolicyLock:     true,
		TargetExceptionAllow: true,
	}
}

// validatePolicyInputs checks the supplied inputs and returns the normalized
// set plus every problem found.
//
// Every refusal here is a case where the alternative would be a quieter
// answer: an unknown label defaulted to "harmless", a missing digest read as
// "unchanged", a mutable version accepted as an identity.
func validatePolicyInputs(inputs []PolicyInput) ([]PolicyInput, []string) {
	if len(inputs) == 0 {
		return nil, nil
	}
	var problems []string
	seen := make(map[string]bool, len(inputs))
	out := make([]PolicyInput, 0, len(inputs))

	for i, in := range inputs {
		where := fmt.Sprintf("policy input %d", i)
		if strings.TrimSpace(in.ID) != "" {
			where = fmt.Sprintf("policy input %q", in.ID)
		}
		switch {
		case strings.TrimSpace(in.ID) == "":
			problems = append(problems, where+": no id")
			continue
		case seen[in.ID]:
			problems = append(problems, where+": declared twice")
			continue
		case strings.TrimSpace(in.Source) == "":
			problems = append(problems, where+": no source")
			continue
		case strings.TrimSpace(in.Version) == "":
			problems = append(problems, where+": no version, so the input cannot be identified")
			continue
		case !digestRe.MatchString(in.Digest):
			problems = append(problems, where+": digest is not a sha256 digest")
			continue
		}
		seen[in.ID] = true

		// A caller may declare a role, but only a role that exists. An
		// unrecognized label does not become user content: a typo would then
		// silently strip authority from a policy input, or grant it.
		label := in.Label
		if label == "" {
			label = LabelPolicyInput
		}
		if !label.Valid() {
			problems = append(problems, fmt.Sprintf("%s: unknown trust label %q", where, in.Label))
			continue
		}
		n := in
		n.Label = label
		out = append(out, n)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	return out, problems
}

// effectivePolicyDigest recomputes the identity of a policy input set in the
// canonical form `scripts/check_policy_pack_lock.py` and
// `internal/guardrails.effectiveDigest` both use.
//
// The canonical form is a JSON array of [path, pack, version, digest,
// resolver_version, overlay_digest] rows in path order with compact
// separators. `resolved_at` is metadata and excluded, so re-resolution is
// reproducible. Reproducing the shipped lock's effective digest from these
// rows is asserted by a test against the real `policy-packs/LOCK.json`, which
// is what makes this a check rather than a second opinion.
func effectivePolicyDigest(inputs []PolicyInput) (string, error) {
	if len(inputs) == 0 {
		return "", errors.New("no policy inputs")
	}
	sorted := make([]PolicyInput, len(inputs))
	copy(sorted, inputs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Source < sorted[j].Source })

	rows := make([][]string, 0, len(sorted))
	for _, in := range sorted {
		if !in.Label.Authoritative() {
			return "", fmt.Errorf("policy input %q carries label %q; only a policy input may define the policy identity",
				in.ID, in.Label)
		}
		if strings.TrimSpace(in.ResolverVersion) == "" || strings.TrimSpace(in.OverlayDigest) == "" {
			return "", fmt.Errorf("policy input %q is missing a lock column, so its identity cannot be recomputed", in.ID)
		}
		rows = append(rows, []string{
			in.Source, in.ID, in.Version, in.Digest, in.ResolverVersion, in.OverlayDigest,
		})
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return "", fmt.Errorf("encode canonical policy inputs: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// verifyProvenance compares the supplied inputs to the digest the manifest
// declares. Three outcomes are kept distinct, because collapsing them is how
// an unverified run starts reading as a verified one.
func verifyProvenance(declared string, inputs []PolicyInput) (ProvenanceStatus, string, string) {
	if len(inputs) == 0 {
		return ProvenanceNotDeclared, "", "this evaluation declared no policy input, so no policy provenance is claimed"
	}
	effective, err := effectivePolicyDigest(inputs)
	if err != nil {
		return ProvenanceUnverifiable, "", err.Error()
	}
	if effective != declared {
		return ProvenanceMismatch, effective,
			fmt.Sprintf("policy inputs hash to %s but the manifest declares %s; this evaluation ran under a policy set it did not declare", effective, declared)
	}
	return ProvenanceRecorded, effective, ""
}

// evaluateRecommendations records every suggestion and refuses any that asks
// for a governed root (R08-4.5.2, R08-4.5.3).
//
// A refusal is per-recommendation and does not abort the evaluation: a
// component overstepping is a finding about that component, and the rest of
// the evidence is still true. It does make the verdict non-pass — an
// evaluation in which a component tried to rewrite policy is not a clean
// evaluation, whatever the checks said.
func evaluateRecommendations(m *Manifest, recs []Recommendation) ([]RecommendationRecord, []string) {
	if len(recs) == 0 {
		return nil, nil
	}
	byID := make(map[string]Component, len(m.Components))
	for _, c := range m.Components {
		byID[c.ID] = c
	}

	var out []RecommendationRecord
	var problems []string
	for i, r := range recs {
		where := fmt.Sprintf("recommendation %d", i)
		comp, ok := byID[r.ComponentID]
		if !ok {
			problems = append(problems, where+": names a component the manifest does not declare")
			continue
		}
		if strings.TrimSpace(r.Text) == "" {
			problems = append(problems, where+": empty text")
			continue
		}

		rec := RecommendationRecord{
			ComponentID: comp.ID,
			TrustTier:   comp.TrustTier,
			// The text is recorded, not interpreted. This is the whole of
			// R08-4.5.3's "data, not executable instructions".
			Text:   RedactDetail(r.Text),
			Target: r.Target,
		}
		if r.Target != TargetNone {
			if !governedTargets()[r.Target] {
				// An unknown target is refused too: "some other root I did not
				// name" is not a structural change this evaluator can weigh.
				rec.Refused = true
				rec.Refusal = fmt.Sprintf("target %q is not a governed root this evaluator recognizes", r.Target)
			} else {
				rec.Refused = true
				rec.Refusal = fmt.Sprintf("a component may not modify %q; a governed root changes only through a reviewed adoption", r.Target)
			}
			problems = append(problems,
				fmt.Sprintf("component %q recommended a change to %s", comp.ID, r.Target))
		}
		out = append(out, rec)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ComponentID != out[j].ComponentID {
			return out[i].ComponentID < out[j].ComponentID
		}
		return out[i].Text < out[j].Text
	})
	return out, problems
}
