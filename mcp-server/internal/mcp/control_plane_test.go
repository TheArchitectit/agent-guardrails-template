package mcp

// Spec 08 R08-4.2.1/R08-4.2.5 — the request path.
//
// The controlplane package proves the composition logic. What matters here is
// that a RUNNING server (a) records that a control was actually invoked, and
// (b) exposes the measured composition through the resource a client reads.
// A test that only exercised the evaluator would not distinguish a wired server
// from one that never calls it — precisely the defect this spec is about.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/controlplane"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

var errFixture = errors.New("manifest does not resolve (fixture)")

// repoRoot resolves the GIT root (not the module root) so the checked-in
// manifest, whose source_paths are repository-relative, can be read the same
// way the server reads it. From internal/mcp that is three levels up.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return root
}

// loadCheckedInManifest loads the real manifest from the repository. Using the
// committed file rather than a fixture means the test fails if the manifest and
// the tree drift, which is the point of checking it in.
func loadCheckedInManifest(t *testing.T) *controlplane.Manifest {
	t.Helper()
	path := filepath.Join(repoRoot(t), "control-plane.manifest.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("checked-in manifest is missing at %s: %v", path, err)
	}
	m, err := controlplane.LoadManifest(path)
	if err != nil {
		t.Fatalf("checked-in manifest does not resolve: %v", err)
	}
	return m
}

// The checked-in manifest must describe the real tree. This is the check that
// stops a manifest from being a typed constant: every declared digest is
// recomputed from the files it names.
func TestCheckedInManifestMatchesTheTree(t *testing.T) {
	m := loadCheckedInManifest(t)
	problems, err := controlplane.VerifySources(repoRoot(t), m)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(problems) > 0 {
		t.Fatalf("manifest digests do not match the tree:\n%s", strings.Join(problems, "\n"))
	}
}

// The manifest and the observation layer must not drift apart. A component the
// manifest declares but nothing observes sits at `declared` forever, which reads
// as an inert control.
func TestEveryManifestComponentIsObserved(t *testing.T) {
	m := loadCheckedInManifest(t)
	server := &MCPServer{exercise: &exerciseCounters{}}
	observed := map[string]bool{}
	for _, id := range server.observedComponentIDs() {
		observed[id] = true
	}
	for _, c := range m.Components {
		if !observed[c.ID] {
			t.Errorf("manifest declares %q but this server cannot observe it", c.ID)
		}
	}
}

// Dispatch is what records exercise. It must fire for a mapped tool and must
// NOT fire for an unmapped one — a counter that counts everything distinguishes
// nothing.
func TestDispatchRecordsExercise(t *testing.T) {
	server := &MCPServer{exercise: &exerciseCounters{}}

	if got := server.counters().count(componentInjectionScan); got != 0 {
		t.Fatalf("counter started at %d, want 0", got)
	}

	// Drive the real dispatch entry point. The handler will fail its own
	// preconditions (no engine), which is fine: the observation is recorded
	// before dispatch precisely so a routed call proves reachability even when
	// the invocation then errors.
	_, _ = server.dispatchToolCall(context.Background(), "guardrail_detect_injection", map[string]any{})

	if got := server.counters().count(componentInjectionScan); got != 1 {
		t.Fatalf("injection counter = %d after dispatch, want 1", got)
	}

	// An unmapped tool proves nothing about any component.
	before := server.counters().count(componentPolicyEngine)
	_, _ = server.dispatchToolCall(context.Background(), "guardrail_get_context", map[string]any{})
	if after := server.counters().count(componentPolicyEngine); after != before {
		t.Fatalf("unmapped tool changed a component counter: %d -> %d", before, after)
	}
}

// A component that has been constructed and wired but never invoked must report
// `reachable`, NOT `exercised`. This is the exact claim R08-4.2.1 requires and
// the exact overstatement R08-4.2.5 forbids.
func TestReachableIsNotExercised(t *testing.T) {
	engine := guardrails.NewEngine(nil, nil)
	server := &MCPServer{
		exercise:         &exerciseCounters{},
		guardrailsEngine: engine,
	}

	obs := observationSource{server: server}.Observe(controlplane.Component{ID: componentInjectionScan})
	if obs.State == controlplane.StateExercised {
		t.Fatalf("a never-invoked detector reported exercised")
	}
	if obs.State != controlplane.StateReachable {
		t.Fatalf("detector state = %s, want reachable (the pipeline works, nothing called it)", obs.State)
	}

	// Record one real invocation and re-observe.
	server.recordExercise("guardrail_detect_injection")
	obs = observationSource{server: server}.Observe(controlplane.Component{ID: componentInjectionScan})
	if obs.State != controlplane.StateExercised {
		t.Fatalf("after an invocation state = %s, want exercised", obs.State)
	}
}

// A server with no engine must report `failed`, not a weaker state: a component
// that cannot exist cannot protect anything.
func TestMissingEngineReportsFailed(t *testing.T) {
	server := &MCPServer{exercise: &exerciseCounters{}}
	for _, id := range []string{componentInjectionScan, componentContentFilter, componentSandboxIsolation, componentPolicyEngine} {
		obs := observationSource{server: server}.Observe(controlplane.Component{ID: id})
		if obs.State != controlplane.StateFailed {
			t.Errorf("%s state = %s, want failed with no engine", id, obs.State)
		}
	}
}

// The resource a client reads must report the MEASURED states and a non-pass
// verdict while mandatory controls have not been exercised.
func TestControlPlaneResourceReportsMeasuredState(t *testing.T) {
	m := loadCheckedInManifest(t)
	engine := guardrails.NewEngine(nil, nil)
	server := &MCPServer{
		exercise:         &exerciseCounters{},
		guardrailsEngine: engine,
	}
	server.SetControlPlane(m, "control-plane.manifest.json")

	result, err := server.readControlPlaneResource(context.Background(), "guardrail://control-plane")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("contents = %d, want 1", len(result.Contents))
	}
	content, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("resource content is %T, want TextResourceContents", result.Contents[0])
	}

	var status controlPlaneStatus
	if err := json.Unmarshal([]byte(content.Text), &status); err != nil {
		t.Fatalf("status is not valid JSON: %v\n%s", err, content.Text)
	}

	if status.Status != "evaluated" {
		t.Fatalf("status = %q, want evaluated", status.Status)
	}
	if status.Verdict == string(controlplane.VerdictPass) {
		t.Fatalf("verdict was pass while mandatory controls had never been exercised")
	}
	if len(status.Components) != len(m.Components) {
		t.Fatalf("reported %d components, manifest declares %d", len(status.Components), len(m.Components))
	}
	if !status.ObserveOnly {
		t.Fatalf("a diagnostic read reported itself as blocking")
	}

	// Every component must carry a state, and no unexercised one may be active.
	for _, cs := range status.Components {
		if cs.State == "" {
			t.Fatalf("component %q reported no state", cs.ID)
		}
		if cs.Active && cs.State != controlplane.StateExercised {
			t.Fatalf("component %q reports active while %s", cs.ID, cs.State)
		}
	}

	// The report must name what is missing rather than only a verdict.
	if len(status.Unmet) == 0 {
		t.Fatalf("non-pass verdict carried no unmet reasons")
	}
}

// An absent manifest reports not_configured, and a broken one reports failed.
// The two must be distinguishable: "nothing configured" is not "broken".
func TestControlPlaneResourceDistinguishesAbsenceFromFailure(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		server := &MCPServer{exercise: &exerciseCounters{}}
		status := server.controlPlaneStatusSnapshot(context.Background())
		if status.Status != "not_configured" {
			t.Fatalf("status = %q, want not_configured", status.Status)
		}
		if status.Error != "" {
			t.Fatalf("not_configured carried an error")
		}
	})

	t.Run("failed", func(t *testing.T) {
		server := &MCPServer{exercise: &exerciseCounters{}}
		server.SetControlPlaneError("bad.json", errFixture)
		status := server.controlPlaneStatusSnapshot(context.Background())
		if status.Status != "failed" {
			t.Fatalf("status = %q, want failed", status.Status)
		}
		if status.Error == "" {
			t.Fatalf("failed carried no error")
		}
	})
}

// The diagnostic must pass the same per-resource authorization every other
// resource passes rather than bypassing it by being new (R16-01/R16-04).
func TestControlPlaneResourceIsInThePermissionMatrix(t *testing.T) {
	action, known := classifyResource("guardrail://control-plane")
	if !known {
		t.Fatalf("guardrail://control-plane is not in the declared permission matrix")
	}
	if string(action.Kind) != "read" {
		t.Fatalf("control-plane resource kind = %s, want read", action.Kind)
	}

	found := false
	for _, row := range resourcePolicy {
		if row.URI == "guardrail://control-plane" {
			found = true
			if strings.TrimSpace(row.Rationale) == "" {
				t.Fatalf("matrix row for guardrail://control-plane declares no rationale")
			}
		}
	}
	if !found {
		t.Fatalf("no declared permission row for guardrail://control-plane")
	}
}

// Every registered resource must still have a matrix row after adding one. This
// is the non-vacuity check: without it the matrix could fall behind the registry
// and every new resource would be denied (or, worse, silently allowed).
func TestResourceMatrixCoversEveryRegisteredResource(t *testing.T) {
	server := &MCPServer{}
	specs := server.resourceSpecs()
	if len(specs) == 0 {
		t.Fatalf("no resources registered — refusing to pass a vacuous scan")
	}
	for _, spec := range specs {
		if _, known := classifyResource(spec.uri); !known {
			t.Errorf("registered resource %q has no permission row", spec.uri)
		}
	}
}
