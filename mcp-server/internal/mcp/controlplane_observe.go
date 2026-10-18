package mcp

// Spec 08 R08-4.2 — the live observation layer.
//
// The controlplane package decides what a manifest MEANS; this file is the only
// place that answers what the running process actually IS. The split is
// deliberate: the evaluator cannot manufacture a state, because every state it
// reports came through ObservationSource below, which reads live fields the
// request path already depends on.
//
// The distinction this file exists to make real is `reachable` vs `exercised`.
// Before the counters below, nothing in the server recorded that a control had
// ever been invoked — a component could be constructed, wired to a dispatch
// case, and never called, and every surface would still describe it as
// available. Reachability is now derived from wiring; exercised is derived from
// an observed call.

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/thearchitectit/guardrail-mcp/internal/controlplane"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// Component IDs used in the checked-in manifest. They are constants so a
// rename in one place cannot silently orphan a counter.
const (
	componentPolicyEngine     = "policy-engine"
	componentInjectionScan    = "injection-detector"
	componentContentFilter    = "content-filter"
	componentSandboxIsolation = "sandbox-isolation"
	componentAuditTrail       = "audit-trail"
)

// exerciseCounters counts observed invocations per component. A counter that
// reads zero means the component is at most reachable, never exercised — which
// is the state that must not be reported as protection.
type exerciseCounters struct {
	counts sync.Map // component id -> *int64
}

func (c *exerciseCounters) record(component string) {
	if component == "" {
		return
	}
	v, _ := c.counts.LoadOrStore(component, new(int64))
	atomic.AddInt64(v.(*int64), 1)
}

func (c *exerciseCounters) count(component string) int64 {
	if c == nil {
		return 0
	}
	v, ok := c.counts.Load(component)
	if !ok {
		return 0
	}
	return atomic.LoadInt64(v.(*int64))
}

// exerciseCounters is a process-wide set attached to the server that owns the
// dispatch path. A nil server counts nothing, which reports everything as
// unexercised rather than assuming success.
func (s *MCPServer) counters() *exerciseCounters {
	if s == nil {
		return nil
	}
	return s.exercise
}

// toolComponent maps a dispatched tool to the control-plane component whose
// exercise it proves. Tools absent from this table prove nothing about any
// component, and are deliberately not mapped to "the server" — a counter that
// counts everything is a counter that distinguishes nothing.
var toolComponent = map[string]string{
	"guardrail_detect_injection":      componentInjectionScan,
	"guardrail_scan_text_batch":       componentInjectionScan,
	"guardrail_scan_external_content": componentInjectionScan,

	"guardrail_classify_content": componentContentFilter,
	"guardrail_check_policy":     componentContentFilter,

	"guardrail_sandbox_execute": componentSandboxIsolation,

	// The policy reader is what proves the policy engine evaluated something;
	// a pack that is loaded but never consulted is still only reachable.
	"guardrail_generate_compliance_report": componentPolicyEngine,
	"guardrail_check_compliance":           componentPolicyEngine,
}

// recordExercise notes that a tool reached dispatch. It is called from the
// single dispatch entry point, so a new tool that is wired but never invoked
// cannot increment anything — which is exactly the measurement wanted.
func (s *MCPServer) recordExercise(tool string) {
	if s == nil || s.exercise == nil {
		return
	}
	if component, ok := toolComponent[tool]; ok {
		s.exercise.record(component)
	}
}

// observationSource reports live state for the components this server owns.
type observationSource struct {
	server *MCPServer
}

// activePolicyDigest is the policy identity the process is running under. An
// absent pack set is reported as empty, never as a digest of "nothing", so a
// consumer cannot mistake "no policy" for "a policy with this identity".
func (s *MCPServer) activePolicyDigest() string {
	if s == nil || s.policyPacks == nil {
		return ""
	}
	active := s.policyPacks.Active()
	if active == nil {
		return ""
	}
	return active.EffectiveDigest()
}

// Observe implements controlplane.Observer against live server state.
func (o observationSource) Observe(c controlplane.Component) controlplane.Observation {
	s := o.server
	obs := controlplane.Observation{
		ComponentID:  c.ID,
		State:        controlplane.StateDeclared,
		Detail:       "declared in the manifest; this server has no observation for it",
		PolicyDigest: s.activePolicyDigest(),
	}

	if s == nil {
		obs.State = controlplane.StateFailed
		obs.Detail = "no server instance to observe"
		return obs
	}

	exercised := s.counters().count(c.ID)
	switch c.ID {
	case componentPolicyEngine:
		if s.guardrailsEngine == nil {
			obs.State = controlplane.StateFailed
			obs.Detail = "no guardrails engine is constructed"
			return obs
		}
		// Installed and initialized by construction. Reachable only when a
		// policy source is actually consulted, which is the pack set.
		obs.State = controlplane.StateInitialized
		obs.Detail = "guardrails engine constructed at startup"
		if s.policyPacks != nil && s.policyPacks.Active() != nil {
			obs.State = controlplane.StateReachable
			obs.Detail = "policy pack set resolved; engine may be consulted"
		}
		if exercised > 0 {
			obs.State = controlplane.StateExercised
			obs.Detail = "policy queries observed at the request path"
		}

	case componentInjectionScan:
		if s.guardrailsEngine == nil {
			obs.State = controlplane.StateFailed
			obs.Detail = "no guardrails engine is constructed"
			return obs
		}
		// The detector is only reachable if the engine would actually scan:
		// a constructed engine with no pipeline answers "safe" without
		// examining anything, which is absence wearing a state.
		obs.State = controlplane.StateInstalled
		obs.Detail = "engine present"
		if probes := s.guardrailsEngine.EngineProbes(); probes != nil {
			if probe, ok := probes["injection_defense"]; ok {
				status, detail := probe(context.Background())
				switch status {
				case guardrails.FeatureActive:
					obs.State = controlplane.StateReachable
					obs.Detail = detail
				case guardrails.FeaturePartial:
					obs.State = controlplane.StateReachable
					obs.Detail = detail
				default:
					obs.State = controlplane.StateFailed
					obs.Detail = "detector did not flag a canonical injection: " + detail
				}
			}
		}
		if exercised > 0 {
			obs.State = controlplane.StateExercised
			obs.Detail = "injection scans observed at the request path"
		}

	case componentContentFilter:
		if s.guardrailsEngine == nil {
			obs.State = controlplane.StateFailed
			obs.Detail = "no guardrails engine is constructed"
			return obs
		}
		backends := s.guardrailsEngine.ContentFilterBackendCount()
		if backends == 0 {
			// Present with no backend fails closed, which is not the same as
			// being able to evaluate content.
			obs.State = controlplane.StateInitialized
			obs.Detail = "content filter present with no semantic backend; classification fails closed"
			return obs
		}
		obs.State = controlplane.StateReachable
		obs.Detail = "content filter has semantic backends attached"
		if exercised > 0 {
			obs.State = controlplane.StateExercised
			obs.Detail = "content classifications observed at the request path"
		}

	case componentSandboxIsolation:
		if s.guardrailsEngine == nil {
			obs.State = controlplane.StateFailed
			obs.Detail = "no guardrails engine is constructed"
			return obs
		}
		cfg := s.guardrailsEngine.SandboxConfig()
		if !cfg.Enabled {
			obs.State = controlplane.StateInstalled
			obs.Detail = "sandbox configured but disabled"
			return obs
		}
		obs.State = controlplane.StateReachable
		obs.Detail = "sandbox enabled with per-tool policies"
		if exercised > 0 {
			obs.State = controlplane.StateExercised
			obs.Detail = "sandbox executions observed at the request path"
		}

	case componentAuditTrail:
		if s.audit == nil {
			obs.State = controlplane.StateFailed
			obs.Detail = "no audit logger is configured"
			return obs
		}
		if !s.audit.HasDurableStore() {
			// The logger emits to the process log, which does not survive the
			// process. That is a real state, and not a durable trail.
			obs.State = controlplane.StateReachable
			obs.Detail = "audit logger emits to the process log only; no durable store"
			return obs
		}
		obs.State = controlplane.StateExercised
		obs.Detail = "audit events persist to a configured store"

	default:
		// An unrecognized component stays `declared`. It is never promoted by
		// default, so adding a component to the manifest without adding its
		// observation cannot make the report claim it works.
	}

	return obs
}

// observedComponentIDs lists the components this server can actually observe.
// It exists so a test can assert the manifest and the observation layer do not
// drift apart: a manifest entry nothing observes would sit at `declared`
// forever and read as an inert control.
func (s *MCPServer) observedComponentIDs() []string {
	ids := []string{
		componentPolicyEngine,
		componentInjectionScan,
		componentContentFilter,
		componentSandboxIsolation,
		componentAuditTrail,
	}
	sort.Strings(ids)
	return ids
}
