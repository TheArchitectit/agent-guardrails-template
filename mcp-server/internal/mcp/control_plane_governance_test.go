package mcp

// Spec 08 R08-4.4.2 .. R08-4.4.4 — the governance posture as a client sees it.
//
// The controlplane package proves promotion and rollback. What matters here is
// that the RUNNING server reports the posture and refuses to look ready when a
// rollback did not complete. A governance object that nothing reads would be
// the same defect this spec keeps finding: a mechanism present and unreachable.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/controlplane"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

func governedServer(t *testing.T) (*MCPServer, *controlplane.Manifest, *controlplane.Governance) {
	t.Helper()
	m := loadCheckedInManifest(t)
	engine := guardrails.NewEngine(nil, nil)
	server := &MCPServer{
		exercise:         &exerciseCounters{},
		guardrailsEngine: engine,
	}
	server.SetControlPlane(m, "control-plane.manifest.json")
	gov, err := controlplane.OpenGovernanceWithManifest(m, "")
	if err != nil {
		t.Fatalf("governance: %v", err)
	}
	server.SetGovernance(gov)
	return server, m, gov
}

func readStatus(t *testing.T, s *MCPServer) controlPlaneStatus {
	t.Helper()
	result, err := s.readControlPlaneResource(context.Background(), "guardrail://control-plane")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("resource content is %T", result.Contents[0])
	}
	var st controlPlaneStatus
	if err := json.Unmarshal([]byte(content.Text), &st); err != nil {
		t.Fatalf("status is not JSON: %v", err)
	}
	return st
}

// The resource must say observe_only, and a promoted deployment must say
// blocking — with the reviewed profile version. The distinction is the one
// R08-4.4.1 exists to make visible to a client.
func TestResourceReportsGovernanceMode(t *testing.T) {
	server, m, gov := governedServer(t)

	st := readStatus(t, server)
	if st.Mode != "observe_only" {
		t.Fatalf("mode = %q, want observe_only", st.Mode)
	}

	clock := func() time.Time { return time.Date(2026, 10, 18, 16, 0, 0, 0, time.UTC) }
	bundle, err := controlplane.Evaluate(controlplane.EvaluateRequest{
		Manifest: m,
		Observer: controlplane.ObserverFunc(func(c controlplane.Component) controlplane.Observation {
			return controlplane.Observation{ComponentID: c.ID, State: controlplane.StateExercised}
		}),
		SubjectDigest: m.Digest(),
		Clock:         clock,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if err := gov.Promote(controlplane.PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-100",
		ApprovedBy:     "roger",
		Acceptance:     bundle,
	}); err != nil {
		t.Fatalf("promote: %v", err)
	}

	st = readStatus(t, server)
	if st.Mode != "blocking" {
		t.Fatalf("mode = %q after promotion, want blocking", st.Mode)
	}
	if st.ProfileVersion != m.Version {
		t.Fatalf("profile version = %q, want %q", st.ProfileVersion, m.Version)
	}
	// The diagnostic read stays observe-only: a GET cannot gate a release.
	if !st.ObserveOnly {
		t.Fatal("a promoted deployment reported its diagnostic read as blocking evidence")
	}
}

// An adopted candidate must be what the resource evaluates. If the resource kept
// reading the startup copy, an adoption would change nothing anyone could see.
func TestResourceEvaluatesTheAdoptedComposition(t *testing.T) {
	server, _, gov := governedServer(t)

	candidate := server.activeManifest()
	if candidate == nil {
		t.Fatal("no active manifest")
	}
	candidate.Version = "2026.10.19"
	if _, err := gov.Adopt(candidate, controlplane.Approval{
		ReviewRef:  "CHG-101",
		ApprovedBy: "roger",
		ApprovedAt: time.Date(2026, 10, 18, 16, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("adopt: %v", err)
	}

	st := readStatus(t, server)
	if st.ManifestDigest != gov.Active().Digest() {
		t.Fatalf("resource reported manifest %s but governance runs %s", st.ManifestDigest, gov.Active().Digest())
	}
	if st.ManifestDigest == server.controlPlane.Digest() {
		t.Fatal("resource still reports the startup manifest after an adoption")
	}
	// A new composition is never blocking: the adoption did not promote it.
	if st.Mode != "observe_only" {
		t.Fatalf("an adopted composition reported mode %q", st.Mode)
	}
}

// R08-4.4.4: a rollback that could not complete must make the instance NOT
// READY even though every component measures fine. Serving on a composition
// nobody restored is the outcome a rollback exists to prevent.
func TestReadinessIsNotReadyAfterAFailedRollback(t *testing.T) {
	server, _, _ := governedServer(t)

	before := server.ControlPlaneReadiness(context.Background())
	if before == nil {
		t.Fatal("readiness reported nothing for a configured composition")
	}

	// No prior unit: the rollback target does not exist.
	if err := server.Governance().Rollback("candidate withdrawn"); err == nil {
		t.Fatal("rollback with no prior composition succeeded")
	}

	after := server.ControlPlaneReadiness(context.Background())
	if after.Ready {
		t.Fatal("readiness stayed true after a rollback that could not complete")
	}
	if after.Status != controlplane.ReadinessNotReady {
		t.Fatalf("status = %q, want not_ready", after.Status)
	}
	joined := strings.Join(after.Unmet, "; ")
	if !strings.Contains(joined, "no prior composition") {
		t.Fatalf("readiness does not name the rollback failure: %s", joined)
	}

	st := readStatus(t, server)
	if len(st.GovernanceBlocked) == 0 {
		t.Fatal("the resource does not report the latched rollback failure")
	}
}

// A server with no governance must be unaffected: an unopted deployment is not
// blocked by a mechanism it never configured.
func TestReadinessUnchangedWithoutGovernance(t *testing.T) {
	m := loadCheckedInManifest(t)
	server := &MCPServer{
		exercise:         &exerciseCounters{},
		guardrailsEngine: guardrails.NewEngine(nil, nil),
	}
	server.SetControlPlane(m, "control-plane.manifest.json")
	if blocked := server.governanceRollbackFailures(); len(blocked) != 0 {
		t.Fatalf("a server with no governance reported rollback failures: %v", blocked)
	}
}
