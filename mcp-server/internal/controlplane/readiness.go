package controlplane

import "strings"

// Readiness: the per-mandatory-component decision of Spec 08 R08-4.2.4.
//
// The requirement is narrow and easy to satisfy vacuously:
//
//	"Readiness MUST report per-mandatory-component state and return not-ready
//	 when a mandatory component is unavailable. Liveness and readiness MUST be
//	 separate signals."
//
// Reporting the state is the easy half. The half that matters is the mapping
// from a measured `IntegrationState` to "available", because the plausible
// mistake is to treat presence as availability: a component whose library is
// linked, or whose struct was constructed at startup, looks ready in every
// way a struct field can look ready — and a request still cannot reach it.
// That is the exact defect this program keeps finding, so the mapping below
// is deliberately unforgiving:
//
//   - `declared` / `installed` / `initialized` are NOT available. The
//     component exists in the process; nothing proves an entry point reaches
//     it. A mandatory control in that state makes the instance not ready,
//     because serving traffic would advertise a protection that cannot run.
//   - `reachable` is available but NOT exercised. A freshly started instance
//     has legitimately never called anything, so blocking on "exercised"
//     would make readiness permanently false and operators would learn to
//     ignore it. The state is reported as `degraded` instead: visible to a
//     human, and distinguishable from proven.
//   - `exercised` is available and proven.
//   - `failed` is not available and outranks everything else, so a component
//     that answered "broken" is never masked by a weaker observation.
//
// Only MANDATORY components decide the aggregate. An optional component that
// failed is a real fact and is reported, but it does not stop the instance:
// the manifest itself declared it non-essential, and readiness that blocks on
// an optional input is readiness nobody can satisfy.
//
// A manifest that declares no mandatory component cannot produce `ready`. The
// profile would be claiming a composition with nothing it requires, which is
// the same vacuity class as a gate that scans zero files and exits 0.

// ReadinessStatus is the derived control-plane readiness outcome.
type ReadinessStatus string

const (
	// ReadinessNotConfigured means no manifest was supplied, so no composition
	// is being claimed. It is not a failure and it is not a green signal.
	ReadinessNotConfigured ReadinessStatus = "not_configured"
	// ReadinessReady means every mandatory component is available and every
	// one of them has been exercised at least once.
	ReadinessReady ReadinessStatus = "ready"
	// ReadinessDegraded means every mandatory component is at least
	// reachable, but at least one has never been exercised. The instance may
	// serve; the composition is not yet proven.
	ReadinessDegraded ReadinessStatus = "degraded"
	// ReadinessNotReady means at least one mandatory component is unavailable
	// or a mandatory check did not pass.
	ReadinessNotReady ReadinessStatus = "not_ready"
)

// MandatoryReadiness is one mandatory component's contribution to readiness.
//
// `Available` and `Exercised` are separate fields on purpose. A single
// boolean would collapse "can reach it" and "it ran", which is the
// distinction the whole spec exists to preserve.
type MandatoryReadiness struct {
	ID        string           `json:"id"`
	State     IntegrationState `json:"state"`
	Available bool             `json:"available"`
	Exercised bool             `json:"exercised"`
	// Reason is a stable code, never free prose: a readiness body is scraped
	// by orchestrators and a changing sentence is a changing contract.
	Reason string `json:"reason"`
}

// Readiness is the derived control-plane readiness decision (R08-4.2.4).
type Readiness struct {
	Status ReadinessStatus `json:"status"`
	// Ready is the boolean an orchestrator gates traffic on. It is true only
	// for `ready` and `degraded`.
	Ready bool `json:"ready"`
	// Mandatory reports every mandatory component, in declaration order, so
	// the answer to "which mandatory component" is in the payload rather than
	// in a log line an operator has to correlate.
	Mandatory []MandatoryReadiness `json:"mandatory"`
	// Unmet names what prevented `ready`, in a fixed order.
	Unmet []string `json:"unmet,omitempty"`
}

// Readiness derives the control-plane readiness decision from a measured
// bundle. It reads what the evaluation observed; it never re-derives state, so
// the resource, the evidence bundle and readiness cannot disagree about the
// same process.
func (b *Bundle) Readiness() Readiness {
	out := Readiness{Status: ReadinessNotConfigured}
	if b == nil {
		return out
	}

	mandatoryIDs := map[string]bool{}
	for _, cs := range b.Components {
		if cs.Mandatory {
			mandatoryIDs[cs.ID] = true
		}
	}

	if len(mandatoryIDs) == 0 {
		// No mandatory component: the profile requires nothing, so readiness
		// would be a claim about an empty set. Fail closed with the reason in
		// the payload rather than returning a green status nothing supports.
		out.Status = ReadinessNotReady
		out.Unmet = []string{"manifest declares no mandatory component"}
		return out
	}

	allAvailable, allExercised := true, true
	for _, cs := range b.Components {
		if !cs.Mandatory {
			continue
		}
		mr := MandatoryReadiness{ID: cs.ID, State: cs.State}
		switch cs.State {
		case StateFailed:
			mr.Available = false
			mr.Exercised = false
			mr.Reason = "component_failed"
		case StateExercised:
			mr.Available = true
			mr.Exercised = true
			mr.Reason = "exercised"
		case StateReachable:
			mr.Available = true
			mr.Exercised = false
			mr.Reason = "reachable_not_exercised"
		default:
			// declared / installed / initialized: present but not reachable.
			// The default branch is the fail-closed one so a state added later
			// cannot silently count as available.
			mr.Available = false
			mr.Exercised = false
			mr.Reason = "not_reachable_" + string(cs.State)
		}
		if !mr.Available {
			allAvailable = false
			out.Unmet = append(out.Unmet, "mandatory component "+cs.ID+" is "+string(cs.State))
		}
		if !mr.Exercised {
			allExercised = false
		}
		out.Mandatory = append(out.Mandatory, mr)
	}

	// A mandatory check that did not pass is a broken control even when the
	// component behind it is reachable: the check ran and refused to prove it.
	for _, c := range b.Checks {
		if !c.Mandatory || !c.Outcome.BlocksPass() {
			continue
		}
		allAvailable = false
		out.Unmet = append(out.Unmet,
			"mandatory check "+c.CheckID+" on "+c.ComponentID+" was "+string(c.Outcome))
	}

	// Profile obligations (R08-4.1.5) gate readiness too. The evaluator derives
	// the verdict from them, so a readiness answer that ignored them could say
	// `ready` about the same process the bundle calls `incomplete`. Readiness
	// and evidence must not disagree about one instance.
	for _, r := range b.Requirements {
		if r.Met {
			continue
		}
		allAvailable = false
		out.Unmet = append(out.Unmet, "profile requirement "+r.ID+" not met: "+strings.Join(r.Unmet, "; "))
	}

	switch {
	case !allAvailable:
		out.Status = ReadinessNotReady
		out.Ready = false
	case !allExercised:
		out.Status = ReadinessDegraded
		out.Ready = true
	default:
		out.Status = ReadinessReady
		out.Ready = true
	}
	return out
}
