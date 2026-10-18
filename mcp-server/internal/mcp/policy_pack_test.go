package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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

// TestPolicyPackStatus_ReportsResolvedDependencyGraph proves the dependency
// edges resolution walked reach the MCP surface (R07-4.1.4 + R07-4.4.1).
//
// The checked-in tree is a single pack, so this drives a two-pack set with one
// declared edge. Asserting only that the field exists would pass for a struct
// field nothing populates — the "declared but unreachable" failure this program
// keeps finding.
func TestPolicyPackStatus_ReportsResolvedDependencyGraph(t *testing.T) {
	dir := t.TempDir()
	base := `{"pack":"core/base","version":"1.0.0","tier":"core","rules":[{"id":"B1","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`
	domain := `{"pack":"core/domain","version":"1.0.0","tier":"core","depends_on":[{"pack":"core/base","range":">=1.0.0"}],"rules":[{"id":"D1","enforcement":"advisory","tool":"guardrail_validate_commit","on_unknown":"stop"}]}`
	writePackFixture(t, dir, "core/base", base)
	writePackFixture(t, dir, "core/domain", domain)
	writeLockFixture(t, dir)

	s := &MCPServer{}
	m := guardrails.NewPolicyPackManager()
	if err := m.ActivateFrom(dir, s.RegisteredToolNames()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	s.SetPolicyPacks(m)

	result, err := s.readPolicyPacksResource(context.Background(), "guardrail://policy-packs")
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	content, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("resource content is %T", result.Contents[0])
	}
	status := decodeStatus(t, content.Text)
	if len(status.Dependencies) != 1 {
		t.Fatalf("resource reported %d dependency edges, want 1: %+v", len(status.Dependencies), status.Dependencies)
	}
	edge := status.Dependencies[0]
	if edge.From != "core/domain" || edge.To != "core/base" || edge.Range != ">=1.0.0" || edge.Version != "1.0.0" {
		t.Fatalf("edge = %+v, want core/domain -> core/base ^>=1.0.0 at 1.0.0", edge)
	}
}

// TestPolicyPackStatus_ReportsPrecedenceAndExceptions proves the precedence
// decision reaches the MCP surface (R07-4.2.1-R07-4.2.3).
//
// A resolver that decided precedence and a surface that did not report it would
// leave an operator unable to see that a core rule had been lowered, or that a
// reviewed exception was the reason. Asserting the fields exist would pass for
// struct fields nothing populates, so this drives the registered handler over a
// real two-tier set carrying a real exception and reads what a client receives.
func TestPolicyPackStatus_ReportsPrecedenceAndExceptions(t *testing.T) {
	dir := t.TempDir()
	base := `{"pack":"core/base","version":"1.0.0","tier":"core","rules":[{"id":"L1","law":"core","enforcement":"blocking","tool":"guardrail_validate_scope","on_unknown":"stop","evidence":"unit"}]}`
	domain := `{"pack":"domain/local","version":"1.0.0","tier":"domain","rules":[{"id":"L1","law":"core","enforcement":"advisory","tool":"guardrail_validate_scope","on_unknown":"stop"}]}`
	writePackFixture(t, dir, "core/base", base)
	writePackFixture(t, dir, "domain/local", domain)
	writeLockFixture(t, dir)

	digest, err := guardrails.PolicyRuleSetDigest(dir, nil)
	if err != nil {
		t.Fatalf("rule-set digest: %v", err)
	}
	ledger := `{"exceptions":[{"rule":"L1","pack":"domain/local","scope":"local-authoring","reason":"local flow needs a warning","approver":"roger","expires":"2027-01-01T00:00:00Z","resulting_rule_set_digest":"` + digest + `"}]}`
	if err := os.WriteFile(filepath.Join(dir, "EXCEPTIONS.json"), []byte(ledger), 0o600); err != nil {
		t.Fatalf("write ledger: %v", err)
	}

	s := &MCPServer{}
	m := guardrails.NewPolicyPackManager()
	if err := m.ActivateFrom(dir, s.RegisteredToolNames()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	s.SetPolicyPacks(m)

	result, err := s.readPolicyPacksResource(context.Background(), "guardrail://policy-packs")
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	content, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("resource content is %T", result.Contents[0])
	}
	status := decodeStatus(t, content.Text)

	if status.RuleSetDigest != digest {
		t.Fatalf("resource rule_set_digest = %q, want %q", status.RuleSetDigest, digest)
	}
	if len(status.Applications) != 1 {
		t.Fatalf("resource reported %d applications, want 1: %+v", len(status.Applications), status.Applications)
	}
	app := status.Applications[0]
	if !app.Weakened || app.Exception != "local-authoring" {
		t.Fatalf("application = %+v, want weakened=true and exception scope local-authoring", app)
	}
	if app.Base != "blocking" || app.Effective != "advisory" {
		t.Fatalf("application base/effective = %q/%q, want blocking/advisory", app.Base, app.Effective)
	}
	if len(status.Exceptions) != 1 || status.Exceptions[0].Approver != "roger" {
		t.Fatalf("resource exceptions = %+v, want the reviewed exception with its approver", status.Exceptions)
	}
	// The effective rule is the weakened one, so the tool attribution must name
	// the override rather than the core declaration that was lowered.
	if got := status.Attribution["guardrail_validate_scope"]; got.Pack != "domain/local" {
		t.Fatalf("attribution pack = %q, want domain/local", got.Pack)
	}
}

// writePackFixture writes one pack file under dir/<name>.pack.json.
func writePackFixture(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name+".pack.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write pack: %v", err)
	}
}

// writeLockFixture writes a LOCK.json in the canonical form the loader verifies:
// a JSON array of [path, pack, version, digest, resolver_version, overlay_digest]
// rows in path order, hashed with compact separators.
func writeLockFixture(t *testing.T, dir string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.pack.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	nested, err := filepath.Glob(filepath.Join(dir, "*", "*.pack.json"))
	if err != nil {
		t.Fatalf("glob nested: %v", err)
	}
	paths = append(paths, nested...)
	sort.Strings(paths)

	type row struct{ key, pack, version, digest string }
	rows := make([]row, 0, len(paths))
	for _, p := range paths {
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			t.Fatalf("rel: %v", err)
		}
		rel = filepath.ToSlash(rel)
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var parsed struct {
			Pack    string `json:"pack"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		sum := sha256.Sum256(raw)
		rows = append(rows, row{
			key: rel, pack: parsed.Pack, version: parsed.Version,
			digest: "sha256:" + hex.EncodeToString(sum[:]),
		})
	}

	canonical := make([][]string, 0, len(rows))
	packs := map[string]any{}
	for _, r := range rows {
		canonical = append(canonical, []string{r.key, r.pack, r.version, r.digest, "1", "none"})
		packs[r.key] = map[string]any{
			"pack": r.pack, "version": r.version, "digest": r.digest,
			"resolver_version": "1", "overlay_digest": "none",
			"resolved_at": "2026-10-18T00:00:00Z",
		}
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("marshal canonical: %v", err)
	}
	total := sha256.Sum256(encoded)
	lock := map[string]any{
		"resolver_version": "1",
		"effective_digest": "sha256:" + hex.EncodeToString(total[:]),
		"packs":            packs,
	}
	body, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatalf("marshal lock: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "LOCK.json"), body, 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
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
