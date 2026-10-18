package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// Spec 07 R07-4.4 — the request path.
//
// The loader is proven in the guardrails package. What matters here is that a
// RUNNING server exposes the active set and can attribute an enforcement
// decision to it. A loader with no consumer was precisely the state this work
// removes, so a test that only exercised the loader would not be evidence.

func policyPackDir(t *testing.T) string {
	t.Helper()
	for _, c := range []string{
		filepath.Join("..", "..", "policy-packs"),
		filepath.Join("..", "..", "..", "policy-packs"),
	} {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, err := filepath.Abs(c)
			if err != nil {
				t.Fatalf("abs: %v", err)
			}
			return abs
		}
	}
	t.Fatalf("policy-packs not found from %s", mustGetwdMCP(t))
	return ""
}

func mustGetwdMCP(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return wd
}

// TestPolicyPackRulesNameRegisteredTools is the cross-check the guardrails
// package cannot perform: every rule in the checked-in pack must name a tool
// this server actually registers. A rule naming a nonexistent tool can never
// fire, so it would read as enforcement and do nothing.
func TestPolicyPackRulesNameRegisteredTools(t *testing.T) {
	s := &MCPServer{}
	set, err := guardrails.LoadPolicySet(policyPackDir(t), s.RegisteredToolNames())
	if err != nil {
		t.Fatalf("checked-in policy set does not resolve against the live tool registry: %v", err)
	}
	if len(set.Rules()) == 0 {
		t.Fatal("resolved no rules; the assertion below would be vacuous")
	}
	registered := s.RegisteredToolNames()
	if len(registered) == 0 {
		t.Fatal("server registers no tools; the assertion would be vacuous")
	}
	for _, rule := range set.Rules() {
		if !registered[rule.Tool] {
			t.Fatalf("rule %s names tool %q, which this server does not register", rule.ID, rule.Tool)
		}
	}
}

// TestPolicyPackStatus_NotConfiguredIsAnAbsence proves the diagnostic never
// reports an empty-but-active policy set. "No pack configured" and "a pack is
// enforcing these rules" must not look alike.
func TestPolicyPackStatus_NotConfiguredIsAnAbsence(t *testing.T) {
	s := &MCPServer{}
	status := s.policyPackStatusSnapshot(nil)
	if status.Status != "not_configured" {
		t.Fatalf("status = %q, want not_configured", status.Status)
	}
	if status.EffectiveDigest != "" {
		t.Fatalf("a not-configured server reported a digest %q", status.EffectiveDigest)
	}
	if len(status.Rules) != 0 {
		t.Fatalf("a not-configured server reported %d rules", len(status.Rules))
	}

	if _, ruleID, ok := s.policyAttributionFields("guardrail_validate_scope"); ok {
		t.Fatalf("a not-configured server attributed a tool to rule %q", ruleID)
	}
}

// TestPolicyPackStatus_ActiveReportsDigestAndAttribution drives the real
// manager over the real pack directory and asserts the diagnostic an operator
// reads: the effective digest, the packs, the rules, and — per tool — the rule
// that governs it (R07-4.4.1 / R07-4.4.3).
func TestPolicyPackStatus_ActiveReportsDigestAndAttribution(t *testing.T) {
	s := &MCPServer{}
	m := guardrails.NewPolicyPackManager()
	if err := m.ActivateFrom(policyPackDir(t), s.RegisteredToolNames()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	s.SetPolicyPacks(m)

	status := s.policyPackStatusSnapshot(nil)
	if status.Status != "active" {
		t.Fatalf("status = %q, want active", status.Status)
	}
	if status.EffectiveDigest == "" || !strings.HasPrefix(status.EffectiveDigest, "sha256:") {
		t.Fatalf("effective digest is not a sha256 identity: %q", status.EffectiveDigest)
	}
	if len(status.Packs) == 0 {
		t.Fatal("active policy set reports no packs")
	}
	if len(status.Rules) == 0 {
		t.Fatal("active policy set reports no rules")
	}

	entry, ok := status.Attribution["guardrail_validate_scope"]
	if !ok {
		t.Fatalf("no attribution for guardrail_validate_scope; got %v", status.Attribution)
	}
	if entry.RuleID != "L2" {
		t.Fatalf("attribution rule = %q, want L2", entry.RuleID)
	}
	if entry.Pack != "core/four-laws" {
		t.Fatalf("attribution pack = %q, want core/four-laws", entry.Pack)
	}
	if entry.PackDigest != status.EffectiveDigest {
		t.Fatalf("attribution digest %q is not the active digest %q", entry.PackDigest, status.EffectiveDigest)
	}
}

// TestPolicyPackResourceIsRegisteredAndReadable proves the resource exists on
// the real registry and its handler returns the status document, so the digest
// is reachable through the MCP surface rather than only through a Go accessor.
func TestPolicyPackResourceIsRegisteredAndReadable(t *testing.T) {
	s := &MCPServer{}

	var found bool
	for _, spec := range s.resourceSpecs() {
		if spec.uri == "guardrail://policy-packs" {
			found = true
		}
	}
	if !found {
		t.Fatal("guardrail://policy-packs is not registered")
	}

	m := guardrails.NewPolicyPackManager()
	if err := m.ActivateFrom(policyPackDir(t), s.RegisteredToolNames()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	s.SetPolicyPacks(m)

	result, err := s.readPolicyPacksResource(context.Background(), "guardrail://policy-packs")
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("resource returned %d contents", len(result.Contents))
	}
	content, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("resource content is %T, want TextResourceContents", result.Contents[0])
	}
	status := decodeStatus(t, content.Text)
	if status.Status != "active" {
		t.Fatalf("resource status = %q, want active", status.Status)
	}
	if status.EffectiveDigest == "" {
		t.Fatal("resource did not report the effective digest")
	}
}

// TestPolicyPackStatus_FailedCandidateIsReported proves a configured but
// unresolvable directory reports failure with its reason, rather than reading
// as "no policy configured" or as an active empty set.
func TestPolicyPackStatus_FailedCandidateIsReported(t *testing.T) {
	s := &MCPServer{}
	m := guardrails.NewPolicyPackManager()
	s.SetPolicyPacks(m)

	status := s.policyPackStatusSnapshot(nil)
	if status.Status != "failed" {
		t.Fatalf("status = %q, want failed for a configured manager with no active set", status.Status)
	}
	if status.Error == "" {
		t.Fatal("a failed status must carry the reason")
	}
}

// --- helpers --------------------------------------------------------------

func decodeStatus(t *testing.T, body string) policyPackStatus {
	t.Helper()
	var status policyPackStatus
	if err := json.Unmarshal([]byte(body), &status); err != nil {
		t.Fatalf("status document is not valid JSON: %v", err)
	}
	return status
}
