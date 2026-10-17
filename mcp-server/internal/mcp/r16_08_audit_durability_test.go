package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
)

// Spec 16 R16-08 — decision/effect audit durability on the MCP transport.
//
// The audit row for R16-08 read `E(web) / NR(MCP)`: the web fail-closed
// durability path was exercised (`TestAuthzAdminMutationFailsClosedWithoutAudit`)
// but "MCP audit durability not exercised". R16-08 requires a security-sensitive
// mutation to fail closed when its required durable record cannot be written,
// on BOTH transports, and requires the record to carry the principal, the
// opaque credential ID, the action, the outcome and a stable reason code —
// never a raw key or hashAPIKey-as-identity.
//
// These tests drive the REAL StreamableHTTP endpoint (httptest + the same
// `requireBearerWithPolicy` wiring `Serve` uses) with a REAL admin credential
// and a REAL on-disk project fixture, so the assertion is about the effect
// (the project file survives a blocked delete) and not about a handler return
// value alone.

// r16FailAuditStore fails every durable insert.
type r16FailAuditStore struct{}

func (r16FailAuditStore) Insert(ctx context.Context, event *database.AuditEvent) error {
	return errors.New("audit store unavailable")
}

// r16CaptureAuditStore accepts inserts and records them.
type r16CaptureAuditStore struct {
	mu     sync.Mutex
	events []*database.AuditEvent
}

func (s *r16CaptureAuditStore) Insert(ctx context.Context, event *database.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Copy: the caller may reuse the pointer.
	cp := *event
	s.events = append(s.events, &cp)
	return nil
}

func (s *r16CaptureAuditStore) last() *database.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) == 0 {
		return nil
	}
	return s.events[len(s.events)-1]
}

// r16AuditRecords returns one administrator credential granted only the
// project used by the tests. A project_delete is an admin-class tool, so this
// credential is the minimum that authorizes the effect.
func r16AuditRecords(project string) []auth.Record {
	return []auth.Record{
		{
			CredentialID: "r16-cred-admin",
			PrincipalID:  "r16-principal-admin",
			Scopes:       []string{auth.ScopeAdminManage, auth.ScopeMCPRead, auth.ScopeMCPMutate},
			Role:         auth.RoleAdministrator,
			Resources:    []string{project},
			Verifier:     auth.Digest(fullPathVerifierKey, "r16-audit-secret"),
		},
	}
}

// r16ProjectDir changes into a scratch directory and creates the project
// fixture the delete tool operates on. The tool resolves `.teams/<name>.json`
// relative to CWD, so the fixture must live there.
func r16ProjectDir(t *testing.T, project string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".teams"), 0o755); err != nil {
		t.Fatalf("mkdir .teams: %v", err)
	}
	path := filepath.Join(dir, ".teams", project+".json")
	if err := os.WriteFile(path, []byte(`{"project":"`+project+`"}`), 0o600); err != nil {
		t.Fatalf("write project fixture: %v", err)
	}
	t.Chdir(dir)
	return path
}

// r16AuditServer wires the real full-path MCP endpoint with an explicit audit
// logger, so the durability boundary under test is the one the serving path
// uses.
func r16AuditServer(t *testing.T, logger *audit.Logger, project string) (*MCPServer, func(string, string, map[string]interface{}) (int, rpcEnvelope)) {
	t.Helper()
	cfg := &config.Config{SchemaVersion: "1.0", MCPAPIKey: fullPathLegacyKey}
	s, ts := newFullPathServer(t, cfg, r16AuditRecords(project))
	s.audit = logger
	post := func(token string, method string, params map[string]interface{}) (int, rpcEnvelope) {
		return rpcPost(t, ts, token, method, params)
	}
	return s, post
}

// TestMCPAuditDurabilityFailsClosed proves the negative half over the real
// endpoint: when the required durable audit record cannot be written, an
// otherwise-authorized admin mutation is denied AND produces no effect.
func TestMCPAuditDurabilityFailsClosed(t *testing.T) {
	const project = "r16-audit-deny"
	projectPath := r16ProjectDir(t, project)

	logger := audit.NewLoggerWithStore(8, r16FailAuditStore{})
	defer logger.Stop()

	_, post := r16AuditServer(t, logger, project)

	call := map[string]interface{}{
		"name": "guardrail_project_delete",
		"arguments": map[string]interface{}{
			"project_name": project,
			"confirmed":    true,
		},
	}
	code, env := post("r16-audit-secret", "tools/call", call)
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}
	tr := decodeToolResult(t, env)
	if !tr.IsError {
		t.Fatalf("admin mutation must fail closed without durable audit; got isError=false text=%q", tr.text())
	}
	if !strings.Contains(tr.text(), "audit unavailable") {
		t.Fatalf("want a durable-audit-unavailable denial, got %q", tr.text())
	}
	// The effect must not have happened: the project file survives.
	if _, err := os.Stat(projectPath); err != nil {
		t.Fatalf("project file was removed despite the audit denial: %v", err)
	}
}

// TestMCPAuditDurabilityRecorded proves the positive half: with a durable
// store the effect proceeds AND an attributable record is written carrying the
// principal, the opaque credential ID, the action, the outcome, the reason
// code and the policy version — never a raw key or hashAPIKey-as-identity.
func TestMCPAuditDurabilityRecorded(t *testing.T) {
	const project = "r16-audit-allow"
	projectPath := r16ProjectDir(t, project)

	store := &r16CaptureAuditStore{}
	logger := audit.NewLoggerWithStore(8, store)
	defer logger.Stop()

	_, post := r16AuditServer(t, logger, project)

	call := map[string]interface{}{
		"name": "guardrail_project_delete",
		"arguments": map[string]interface{}{
			"project_name": project,
			"confirmed":    true,
		},
	}
	code, env := post("r16-audit-secret", "tools/call", call)
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}
	tr := decodeToolResult(t, env)
	if tr.IsError {
		t.Fatalf("authorized admin mutation with durable audit must succeed; got %q", tr.text())
	}
	// The effect happened.
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		t.Fatalf("project file still present after an authorized delete: %v", err)
	}

	rec := store.last()
	if rec == nil {
		t.Fatal("no durable audit record written for the authorized admin mutation")
	}
	if rec.EventType != string(audit.EventAuthzDecision) {
		t.Fatalf("event_type = %q, want %q", rec.EventType, audit.EventAuthzDecision)
	}
	if rec.Actor != "r16-principal-admin" {
		t.Fatalf("actor = %q, want the server-controlled principal", rec.Actor)
	}
	if rec.Action != "guardrail_project_delete" {
		t.Fatalf("action = %q, want guardrail_project_delete", rec.Action)
	}
	if rec.Status != audit.DecisionAllow {
		t.Fatalf("status = %q, want %q", rec.Status, audit.DecisionAllow)
	}

	raw, err := json.Marshal(rec.Details)
	if err != nil {
		t.Fatalf("marshal details: %v", err)
	}
	details := string(raw)
	// The record must be attributable and must never carry raw credential
	// material (the secret used to authenticate) or hashAPIKey-as-identity.
	if !strings.Contains(details, "r16-cred-admin") {
		t.Fatalf("record lacks the opaque credential ID: %s", details)
	}
	if strings.Contains(details, "r16-audit-secret") {
		t.Fatalf("record leaked the raw credential: %s", details)
	}
	if !strings.Contains(details, `"surface":"mcp"`) {
		t.Fatalf("record does not name the mcp surface: %s", details)
	}
	if !strings.Contains(details, `"policy_version"`) {
		t.Fatalf("record lacks the policy version: %s", details)
	}
	if !strings.Contains(details, `"reason"`) {
		t.Fatalf("record lacks a stable reason code: %s", details)
	}
}
