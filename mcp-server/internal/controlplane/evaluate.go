package controlplane

// Evaluation: the integration-state model, the evidence bundle and the
// derived verdict (Spec 08 R08-4.2 / R08-4.3).
//
// The central rule: state is observed, never declared. A manifest says what
// SHOULD be composed; this file asks the running process what actually IS, and
// reports the difference. Three consequences are deliberate:
//
//   - `installed` is not `reachable`. A component whose library is linked but
//     whose entry point no request can reach reports `installed` only, and the
//     system verdict does not credit its protection (criterion 2).
//   - `reachable` is not `exercised`. Entry point wiring proves a request
//     COULD reach the component; only an observed invocation proves it did.
//     A deployment that has never called a control reports `reachable`, never
//     `exercised`, so "configured" cannot pass for "proven".
//   - a mandatory check that is `skipped`, `error`, `inconclusive` or
//     `not_run` prevents a `pass` result. The verdict is derived from the
//     profile requirement and the per-check outcomes; nothing can assign
//     `pass` directly (R08-4.3.4).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// IntegrationState is the measured composition state of one component
// (R08-4.2.1). The order is meaningful: each state is strictly stronger than
// the one before it, and a report never claims a stronger state than it
// observed.
type IntegrationState string

const (
	// StateDeclared means only the manifest names it.
	StateDeclared IntegrationState = "declared"
	// StateInstalled means the code is present in this build.
	StateInstalled IntegrationState = "installed"
	// StateInitialized means it was constructed and configured at startup.
	StateInitialized IntegrationState = "initialized"
	// StateReachable means a request/entry point can invoke it.
	StateReachable IntegrationState = "reachable"
	// StateExercised means it was actually invoked at least once.
	StateExercised IntegrationState = "exercised"
	// StateFailed means measurement found it broken; it outranks nothing and
	// always blocks a mandatory pass.
	StateFailed IntegrationState = "failed"
)

// rank orders the states so a probe result can be compared. `failed` ranks
// above `exercised` only so it is never masked by a weaker state; the verdict
// logic tests `failed` explicitly rather than relying on rank.
func (s IntegrationState) rank() int {
	switch s {
	case StateDeclared:
		return 1
	case StateInstalled:
		return 2
	case StateInitialized:
		return 3
	case StateReachable:
		return 4
	case StateExercised:
		return 5
	case StateFailed:
		return 6
	}
	return 0
}

// Valid reports whether the state is one this package understands.
func (s IntegrationState) Valid() bool { return s.rank() > 0 }

// Observation is what the running process reports about one component. It is
// supplied by the wiring layer, never read from the manifest.
type Observation struct {
	// ComponentID names the component being observed.
	ComponentID string
	// State is the strongest state the observer can justify.
	State IntegrationState
	// Detail is a bounded, non-secret explanation. It is recorded in the
	// evidence bundle and must never contain a credential or raw user content.
	Detail string
	// PolicyDigest is the policy identity the component is running under, when
	// the component consumes policy (R08-4.2.2).
	PolicyDigest string
}

// Observer measures the live process. A nil observer is not "everything is
// fine": the evaluator then reports every component `declared` and the overall
// verdict `incomplete`.
type Observer interface {
	Observe(component Component) Observation
}

// ObserverFunc adapts a function to Observer.
type ObserverFunc func(component Component) Observation

// Observe implements Observer.
func (f ObserverFunc) Observe(c Component) Observation { return f(c) }

// CheckOutcome is the terminal result of one evidence check (R08-4.3.2).
type CheckOutcome string

const (
	OutcomePass         CheckOutcome = "pass"
	OutcomeBlock        CheckOutcome = "block"
	OutcomeInconclusive CheckOutcome = "inconclusive"
	OutcomeError        CheckOutcome = "error"
	OutcomeSkipped      CheckOutcome = "skipped"
)

// Valid reports whether the outcome is a known terminal result. An unknown
// outcome is refused rather than coerced: coercing it would pick a verdict on
// the caller's behalf.
func (o CheckOutcome) Valid() bool {
	switch o {
	case OutcomePass, OutcomeBlock, OutcomeInconclusive, OutcomeError, OutcomeSkipped:
		return true
	}
	return false
}

// BlocksPass reports whether this outcome prevents an overall `pass`. Every
// non-pass outcome does; the method exists so call sites state that intent
// instead of negating a comparison.
func (o CheckOutcome) BlocksPass() bool { return o != OutcomePass }

// CheckRecord is one entry in the evidence bundle (R08-4.3.2).
type CheckRecord struct {
	ComponentID   string       `json:"component_id"`
	ComponentVer  string       `json:"component_version"`
	CheckID       string       `json:"check_id"`
	InputDigest   string       `json:"input_digest"`
	Outcome       CheckOutcome `json:"outcome"`
	ReasonCode    string       `json:"reason_code"`
	Detail        string       `json:"detail,omitempty"`
	Mandatory     bool         `json:"mandatory"`
	ComponentStat string       `json:"component_state"`
}

// Verdict is the derived overall result.
type Verdict string

const (
	VerdictPass       Verdict = "pass"
	VerdictBlock      Verdict = "block"
	VerdictIncomplete Verdict = "incomplete"
	VerdictFailed     Verdict = "failed"
)

// Bundle is the immutable evaluation record (R08-4.3.1).
type Bundle struct {
	// EvaluationID identifies this run. It is derived from content, not
	// random, so a replay of identical inputs is recognizable.
	EvaluationID     string    `json:"evaluation_id"`
	SubjectDigest    string    `json:"subject_digest"`
	ManifestDigest   string    `json:"manifest_digest"`
	PolicyDigest     string    `json:"policy_digest"`
	Profile          string    `json:"profile"`
	EvaluatorVersion string    `json:"evaluator_version"`
	StartedAt        time.Time `json:"started_at"`
	EndedAt          time.Time `json:"ended_at"`
	Verdict          Verdict   `json:"verdict"`
	// ObserveOnly marks a report produced in observe-only mode (R08-4.4.1).
	// An observe-only bundle can never carry a blocking verdict.
	ObserveOnly bool             `json:"observe_only"`
	Checks      []CheckRecord    `json:"checks"`
	Components  []ComponentState `json:"components"`
	// Unmet names what prevented a pass, in a fixed order.
	Unmet []string `json:"unmet,omitempty"`
	// Digest is the digest of this bundle's content, computed after the fact.
	Digest string `json:"digest"`
}

// ComponentState is the measured state of one component in the report.
type ComponentState struct {
	ID        string           `json:"id"`
	Version   string           `json:"version"`
	Mandatory bool             `json:"mandatory"`
	Declared  bool             `json:"declared"`
	State     IntegrationState `json:"state"`
	Detail    string           `json:"detail,omitempty"`
	// Active is true only when the component is exercised AND, if mandatory,
	// not failed. It is the only field a status page may use to claim a
	// protection is active (R08-4.2.5).
	Active bool `json:"active"`
}

// EvaluatorVersion identifies this implementation in the bundle. It is
// published so a silent behavioural change is visible in old evidence.
const EvaluatorVersion = "controlplane/1"

// Check is one evaluable unit supplied by the wiring layer. A check whose
// component is not declared in the manifest is refused: evidence for an
// undeclared component is evidence about something the profile does not cover.
type Check struct {
	ComponentID string
	CheckID     string
	InputDigest string
	Mandatory   bool
	// Run performs the check. A nil Run is a wiring mistake and is reported
	// `error`, never `pass`.
	Run func() (CheckOutcome, string, string)
}

// EvaluateRequest is the input to Evaluate.
type EvaluateRequest struct {
	Manifest      *Manifest
	Observer      Observer
	Checks        []Check
	SubjectDigest string
	// ObserveOnly produces a non-blocking report that is labelled as such
	// (R08-4.4.1). It does not weaken the checks; it changes the label.
	ObserveOnly bool
	// Clock is injectable for deterministic tests.
	Clock func() time.Time
}

// Evaluate composes an evidence bundle. It never returns a `pass` verdict for a
// mandatory non-pass check, whatever the caller asked for.
func Evaluate(req EvaluateRequest) (*Bundle, error) {
	if req.Manifest == nil {
		return nil, errors.New("control-plane manifest is required")
	}
	if err := req.Manifest.Resolve(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.SubjectDigest) == "" {
		return nil, errors.New("subject digest is required")
	}
	clock := req.Clock
	if clock == nil {
		clock = time.Now
	}

	started := clock().UTC()
	b := &Bundle{
		SubjectDigest:    req.SubjectDigest,
		ManifestDigest:   req.Manifest.Digest(),
		PolicyDigest:     req.Manifest.PolicyDigest,
		Profile:          req.Manifest.Profile,
		EvaluatorVersion: EvaluatorVersion,
		StartedAt:        started,
		ObserveOnly:      req.ObserveOnly,
	}

	// 1. Measure component state from the live process.
	b.Components = make([]ComponentState, 0, len(req.Manifest.Components))
	for _, c := range req.Manifest.Components {
		cs := ComponentState{
			ID:        c.ID,
			Version:   c.Version,
			Mandatory: c.Mandatory,
			Declared:  true,
			State:     StateDeclared,
			Detail:    "declared in the manifest; nothing observed it",
		}
		if req.Observer != nil {
			obs := req.Observer.Observe(c)
			if obs.ComponentID != "" && obs.ComponentID != c.ID {
				// A mismatched observation is a wiring defect. Treating it as
				// evidence for this component would attribute one control's
				// health to another.
				cs.State = StateFailed
				cs.Detail = fmt.Sprintf("observer returned %q for %q", obs.ComponentID, c.ID)
			} else if !obs.State.Valid() {
				cs.State = StateFailed
				cs.Detail = fmt.Sprintf("observer returned an unknown state %q", obs.State)
			} else {
				cs.State = obs.State
				// R08-4.3.3: an observer supplies free-form prose, so it is
				// redacted here rather than trusted. The bundle is shipped.
				cs.Detail = RedactDetail(obs.Detail)
			}
		}
		// Active is the only claim a status page may make. An unexercised
		// component is never active, and neither is a failed one.
		cs.Active = cs.State == StateExercised
		b.Components = append(b.Components, cs)
	}

	// 2. Run the checks the wiring layer supplied.
	declared := make(map[string]Component, len(req.Manifest.Components))
	for _, c := range req.Manifest.Components {
		declared[c.ID] = c
	}
	for _, chk := range req.Checks {
		comp, ok := declared[chk.ComponentID]
		if !ok {
			return nil, fmt.Errorf("check %q names component %q, which the manifest does not declare",
				chk.CheckID, chk.ComponentID)
		}
		rec := CheckRecord{
			ComponentID:   comp.ID,
			ComponentVer:  comp.Version,
			CheckID:       chk.CheckID,
			InputDigest:   chk.InputDigest,
			Mandatory:     chk.Mandatory || comp.Mandatory,
			ComponentStat: string(stateOf(b.Components, comp.ID)),
		}
		switch {
		case chk.Run == nil:
			// A check with no implementation is an error, not a skip: a skip
			// reads as "not applicable", and this one was wired and is broken.
			rec.Outcome = OutcomeError
			rec.ReasonCode = "check_not_implemented"
			rec.Detail = "check was declared without an implementation"
		case strings.TrimSpace(chk.CheckID) == "":
			rec.Outcome = OutcomeError
			rec.ReasonCode = "check_id_missing"
			rec.Detail = "check declared no id"
		default:
			outcome, code, detail := chk.Run()
			if !outcome.Valid() {
				rec.Outcome = OutcomeError
				rec.ReasonCode = "unknown_outcome"
				rec.Detail = fmt.Sprintf("check returned an unrepresentable outcome %q", outcome)
			} else {
				rec.Outcome = outcome
				rec.ReasonCode = code
				rec.Detail = RedactDetail(detail)
			}
		}
		b.Checks = append(b.Checks, rec)
	}

	// Stable ordering so a replay of the same inputs produces the same digest.
	sort.SliceStable(b.Checks, func(i, j int) bool {
		if b.Checks[i].ComponentID != b.Checks[j].ComponentID {
			return b.Checks[i].ComponentID < b.Checks[j].ComponentID
		}
		return b.Checks[i].CheckID < b.Checks[j].CheckID
	})

	b.EndedAt = clock().UTC()
	b.Verdict, b.Unmet = deriveVerdict(req.Manifest, b)
	b.EvaluationID = b.computeID()
	b.Digest = b.computeDigest()
	return b, nil
}

// deriveVerdict computes the overall result from the profile requirement and
// the per-check outcomes (R08-4.3.4). No code path assigns `pass`.
//
// The three failure verdicts are deliberately distinct, because they call for
// different operator action:
//
//   - `block`: a control ran and refused. The system is working; the subject
//     was rejected.
//   - `failed`: something that should have run is broken. Act on the system.
//   - `incomplete`: nothing failed, but the evaluation did not finish being
//     proven (a skipped check, a component that was never exercised).
//
// Collapsing `failed` into `incomplete` would report a broken mandatory control
// as merely unmeasured, which reads as "run it again" instead of "fix it".
func deriveVerdict(m *Manifest, b *Bundle) (Verdict, []string) {
	var unmet []string
	var blocked, failed bool

	// A mandatory component that is not exercised is not proven. This is the
	// rule that makes "installed" insufficient: an unreachable control cannot
	// protect a request, whether or not its library is linked.
	for _, cs := range b.Components {
		if !cs.Mandatory {
			continue
		}
		switch cs.State {
		case StateFailed:
			failed = true
			unmet = append(unmet,
				fmt.Sprintf("mandatory component %q failed: %s", cs.ID, cs.Detail))
		case StateExercised:
			// proven
		default:
			unmet = append(unmet,
				fmt.Sprintf("mandatory component %q is %s, not exercised", cs.ID, cs.State))
		}
	}

	for _, c := range b.Checks {
		if !c.Mandatory || !c.Outcome.BlocksPass() {
			continue
		}
		unmet = append(unmet,
			fmt.Sprintf("mandatory check %q on %q was %s (%s)", c.CheckID, c.ComponentID, c.Outcome, c.ReasonCode))
		switch c.Outcome {
		case OutcomeBlock:
			blocked = true
		case OutcomeError:
			failed = true
		}
	}

	sort.Strings(unmet)

	switch {
	case len(unmet) == 0:
		return VerdictPass, nil
	case blocked:
		return VerdictBlock, unmet
	case failed:
		return VerdictFailed, unmet
	default:
		// Skipped / inconclusive / not-run: the evaluation did not complete,
		// which is a different answer from "failed" and must not read as one.
		return VerdictIncomplete, unmet
	}
}

func stateOf(states []ComponentState, id string) IntegrationState {
	for _, cs := range states {
		if cs.ID == id {
			return cs.State
		}
	}
	return StateDeclared
}

// Enforce reports whether this bundle may satisfy a blocking gate (R08-4.4.1).
//
// It exists as a method, not a caller-side condition, because "check the
// ObserveOnly flag and the verdict" is a rule each caller would have to
// remember. An observe-only bundle is a report about what WOULD happen; letting
// one gate a release would make the observe-only mode a compliance fiction.
// A non-pass verdict is refused for the same reason, with the unmet reasons in
// the error so the operator sees what to fix rather than only that it failed.
func (b *Bundle) Enforce() error {
	if b == nil {
		return errors.New("no evaluation bundle: an absent result is not a pass")
	}
	if b.ObserveOnly {
		return errors.New("evaluation was observe-only; it can never satisfy a blocking gate")
	}
	if b.Verdict != VerdictPass {
		if len(b.Unmet) == 0 {
			return fmt.Errorf("evaluation verdict is %s", b.Verdict)
		}
		return fmt.Errorf("evaluation verdict is %s: %s", b.Verdict, strings.Join(b.Unmet, "; "))
	}
	return nil
}

// canonicalJSON marshals a value with sorted map keys and no HTML escaping so
// two equal bundles always produce equal bytes.
func canonicalJSON(v any) []byte {
	buf := &strings.Builder{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return []byte("unencodable")
	}
	return []byte(strings.TrimRight(buf.String(), "\n"))
}

// computeID derives the evaluation id from the identity of the run: subject,
// manifest, policy, profile and evaluator. Two runs over the same subject under
// the same manifest are the same evaluation, which is what makes a replay
// comparison meaningful.
func (b *Bundle) computeID() string {
	payload := fmt.Sprintf("%s|%s|%s|%s|%s",
		b.SubjectDigest, b.ManifestDigest, b.PolicyDigest, b.Profile, b.EvaluatorVersion)
	sum := sha256.Sum256([]byte(payload))
	return "eval:sha256:" + hex.EncodeToString(sum[:])
}

// computeDigest digests the bundle content, excluding the digest field itself.
func (b *Bundle) computeDigest() string {
	type alias Bundle
	copyB := *b
	copyB.Digest = ""
	sum := sha256.Sum256(canonicalJSON(alias(copyB)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ReplayComparison reports whether a stored bundle describes the same
// evaluation as a freshly computed one (R08-4.3.5 / criterion 6).
type ReplayComparison struct {
	Equivalent bool `json:"equivalent"`
	// Reasons names every difference, so a caller is told what changed rather
	// than only that something did.
	Reasons []string `json:"reasons,omitempty"`
}

// CompareReplay compares a stored bundle with a new one. It deliberately does
// not compare timestamps: a replay is not required to happen at the same
// instant, and claiming byte-identity across runs would be a stronger claim
// than the evidence supports.
func CompareReplay(stored, fresh *Bundle) ReplayComparison {
	cmp := ReplayComparison{Equivalent: true}
	add := func(format string, args ...any) {
		cmp.Equivalent = false
		cmp.Reasons = append(cmp.Reasons, fmt.Sprintf(format, args...))
	}
	if stored == nil || fresh == nil {
		return ReplayComparison{Equivalent: false, Reasons: []string{"a bundle is missing"}}
	}
	if stored.EvaluationID != fresh.EvaluationID {
		add("evaluation identity changed: %s -> %s", stored.EvaluationID, fresh.EvaluationID)
	}
	if stored.ManifestDigest != fresh.ManifestDigest {
		add("manifest digest changed: %s -> %s", stored.ManifestDigest, fresh.ManifestDigest)
	}
	if stored.PolicyDigest != fresh.PolicyDigest {
		add("policy digest changed: %s -> %s", stored.PolicyDigest, fresh.PolicyDigest)
	}
	if stored.SubjectDigest != fresh.SubjectDigest {
		add("subject digest changed: %s -> %s", stored.SubjectDigest, fresh.SubjectDigest)
	}
	if stored.EvaluatorVersion != fresh.EvaluatorVersion {
		add("evaluator version changed: %s -> %s", stored.EvaluatorVersion, fresh.EvaluatorVersion)
	}
	if len(stored.Checks) != len(fresh.Checks) {
		add("check count changed: %d -> %d", len(stored.Checks), len(fresh.Checks))
	} else {
		for i := range stored.Checks {
			s, f := stored.Checks[i], fresh.Checks[i]
			if s.ComponentID != f.ComponentID || s.CheckID != f.CheckID {
				add("check %d identity changed: %s/%s -> %s/%s", i, s.ComponentID, s.CheckID, f.ComponentID, f.CheckID)
				continue
			}
			if s.Outcome != f.Outcome {
				add("check %s/%s outcome changed: %s -> %s", s.ComponentID, s.CheckID, s.Outcome, f.Outcome)
			}
			if s.ComponentVer != f.ComponentVer {
				add("check %s/%s component version changed: %s -> %s",
					s.ComponentID, s.CheckID, s.ComponentVer, f.ComponentVer)
			}
		}
	}
	if cmp.Equivalent && stored.Digest != fresh.Digest {
		// Same identity and same outcomes but a different digest means a field
		// outside the compared set moved (a component state, a detail string).
		// Reporting it keeps a non-equivalent replay from being labelled
		// identical because only the compared fields happened to match.
		add("bundle digest changed: %s -> %s", stored.Digest, fresh.Digest)
	}
	if len(cmp.Reasons) > 0 {
		sort.Strings(cmp.Reasons)
	}
	return cmp
}
