package guardrails

// Spec 06 — runtime compliance probes.
//
// The defect this file exists to remove is stated in the spec itself:
//
//	"Do not quote a score from this code."
//
// CheckRequirement returned true purely because the hand-written
// compliance_status string in compliance_requirements.json equalled "full",
// with the in-code comment "In a real implementation, this would check if the
// features are actually enabled." A score produced that way restates the
// database; it does not measure the system. Presenting it to an auditor would
// be inaccurate — the single most consequential gap in the six specs.
//
// The rule enforced here is the vacuity rule used by the repo's other gates: a
// requirement is satisfied only when the controls it NAMES are observed live.
// A control that cannot be observed is "unknown", and unknown never counts as
// covered. An unimplemented probe must not be able to pass by default.

import (
	"context"
	"sort"
	"strings"
)

// FeatureStatus is the measured state of one declared guardrail feature.
type FeatureStatus string

const (
	// FeatureActive means the feature was observed present and evaluating.
	FeatureActive FeatureStatus = "active"
	// FeaturePartial means it is present but cannot exercise its full declared
	// contract (e.g. a detector with no semantic backend attached).
	FeaturePartial FeatureStatus = "partial"
	// FeatureAbsent means the declared control does not exist in this build.
	FeatureAbsent FeatureStatus = "absent"
	// FeatureUnknown means nothing measured it. It must never count as covered:
	// "we did not look" and "we looked and it was fine" are different answers,
	// and only one of them is evidence.
	FeatureUnknown FeatureStatus = "unknown"
)

// FeatureObservation is the result of probing one declared feature name.
type FeatureObservation struct {
	Feature string        `json:"feature"`
	Status  FeatureStatus `json:"status"`
	Detail  string        `json:"detail"`
}

// ProbeFunc inspects live state. It must look at the running system, never at
// the requirement database: a probe that reads the same JSON it is meant to
// check reproduces the defect this file removes.
type ProbeFunc func(context.Context) (FeatureStatus, string)

// ProbeSet maps a declared feature name to the probe that measures it.
type ProbeSet map[string]ProbeFunc

// Merge returns a set containing both, with b winning on conflict. It exists so
// the MCP layer can add the controls it alone can see (durable audit, metrics
// export) to the engine's own probes without either package reaching into the
// other's internals.
func (p ProbeSet) Merge(b ProbeSet) ProbeSet {
	out := make(ProbeSet, len(p)+len(b))
	for k, v := range p {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// Names returns the probed feature names, sorted, for a stable report.
func (p ProbeSet) Names() []string {
	names := make([]string, 0, len(p))
	for k := range p {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// RequirementAssessment is a requirement judged against live observation
// rather than against its own declaration.
type RequirementAssessment struct {
	RequirementID string               `json:"requirement_id"`
	Title         string               `json:"title"`
	Declared      string               `json:"declared_status"`
	Measured      FeatureStatus        `json:"measured_status"`
	Observed      []FeatureObservation `json:"observed_features"`
	Gaps          []string             `json:"gaps,omitempty"`
	// DeclarationAgrees is true only when the measured status is at least as
	// strong as the declaration. A false value is a finding in its own right:
	// the database claims a control the system does not have.
	DeclarationAgrees bool `json:"declaration_agrees"`
	// Covered is the single field a score may count. It is true only for
	// FeatureActive — never for partial, absent or unknown.
	Covered bool `json:"covered"`
}

// AssessRequirement measures one requirement's declared features.
//
// Failure modes are deliberate and each one fails closed:
//   - a requirement that declares no features is ABSENT, not compliant: there
//     is nothing to observe, so nothing can have been demonstrated;
//   - a feature with no registered probe is UNKNOWN, which is never covered;
//   - any ABSENT feature makes the whole requirement ABSENT (a named control
//     that does not exist is not offset by the ones that do);
//   - any UNKNOWN feature makes the requirement UNKNOWN (a claim of full
//     compliance cannot rest on a control nobody measured).
func AssessRequirement(ctx context.Context, reqID string, req Requirement, probes ProbeSet) RequirementAssessment {
	a := RequirementAssessment{
		RequirementID: reqID,
		Title:         req.Title,
		Declared:      req.ComplianceStatus,
		Measured:      FeatureUnknown,
		Covered:       false,
	}

	if len(req.GuardrailFeatures) == 0 {
		a.Measured = FeatureAbsent
		a.Gaps = append(a.Gaps,
			"requirement names no guardrail features, so no control can be observed")
		a.DeclarationAgrees = req.ComplianceStatus != "full"
		return a
	}

	worst := FeatureActive
	var absent, unknown, partial []string

	for _, feature := range req.GuardrailFeatures {
		probe, ok := probes[feature]
		if !ok {
			// The exact vacuity this file exists to prevent: an unprobed
			// feature used to inherit the declared "full" status.
			a.Observed = append(a.Observed, FeatureObservation{
				Feature: feature,
				Status:  FeatureUnknown,
				Detail:  "no probe registered for this feature",
			})
			unknown = append(unknown, feature)
			worst = FeatureUnknown
			continue
		}

		status, detail := probe(ctx)
		if status == "" {
			status = FeatureUnknown
			detail = "probe returned no status"
		}
		a.Observed = append(a.Observed, FeatureObservation{
			Feature: feature,
			Status:  status,
			Detail:  detail,
		})

		switch status {
		case FeatureActive:
			// no change
		case FeaturePartial:
			partial = append(partial, feature)
			if worst == FeatureActive {
				worst = FeaturePartial
			}
		case FeatureAbsent:
			absent = append(absent, feature)
			worst = FeatureAbsent
		default:
			unknown = append(unknown, feature)
			if worst != FeatureAbsent {
				worst = FeatureUnknown
			}
		}
	}

	if len(absent) > 0 {
		a.Gaps = append(a.Gaps, "absent controls: "+strings.Join(absent, ", "))
	}
	if len(unknown) > 0 {
		a.Gaps = append(a.Gaps, "unmeasured controls: "+strings.Join(unknown, ", "))
	}
	if len(partial) > 0 {
		a.Gaps = append(a.Gaps, "partially exercised controls: "+strings.Join(partial, ", "))
	}

	a.Measured = worst
	a.Covered = worst == FeatureActive
	a.DeclarationAgrees = declarationAtLeast(req.ComplianceStatus, worst)
	return a
}

// declarationFull is the rank a declaration carries when it claims a control
// works. It is named because the UNKNOWN branch below is defined against it.
const declarationFull = 3

// declarationAtLeast reports whether a declared status is no stronger than what
// was measured. Comparing ranks (rather than strings) is what makes an
// over-claim visible: declared "full" against measured "partial" is a finding.
//
// The UNKNOWN case is not blanket disagreement. Nothing observed the control,
// so nothing can contradict a declaration of DEFICIENCY — a requirement the
// database already marks "gap"/"partial" is not over-claiming when no probe
// could measure it. A declaration of POSITIVITY is the over-claim: "full"
// asserts that a control works and nothing supports that. Treating unknown as
// plain disagreement would have reported Art. 50 (declared "gap", no
// watermarking implemented) as an over-declaration the moment its last probe
// could not observe it — a false finding in the opposite direction, from the
// same table.
func declarationAtLeast(declared string, measured FeatureStatus) bool {
	if measured == FeatureUnknown {
		return declarationRank(declared) < declarationFull
	}
	return declarationRank(declared) <= statusRank(measured)
}

func declarationRank(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "full", "compliant":
		return declarationFull
	case "partial":
		return 2
	case "gap", "absent", "none":
		return 1
	}
	return 0 // unknown declaration ranks lowest, so it never over-claims
}

func statusRank(s FeatureStatus) int {
	switch s {
	case FeatureActive:
		return 3
	case FeaturePartial:
		return 2
	case FeatureAbsent:
		return 1
	}
	return 0
}
