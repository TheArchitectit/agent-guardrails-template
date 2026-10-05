package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// canarySecret is a single distinctive canary planted in every secret-bearing
// config field and in tool arguments. Spec 19 R19-06: no secret value may
// appear in logs, audit records, error responses, or any serialized output.
const canarySecret = "CANARY_SECRET_LEAK_b7f3a9c1d2e4"

func canaryLeakConfig() *config.Config {
	return &config.Config{
		SchemaVersion:             "1.0",
		MCPPort:                   8080,
		LogLevel:                  "info",
		DBPassword:                canarySecret,
		MCPAPIKey:                 canarySecret,
		IDEAPIKey:                 canarySecret,
		JWTSecret:                 canarySecret,
		CredentialVerifierKey:     canarySecret,
		RedisPassword:             canarySecret,
		CredentialRegistryJSON:    `[{"credential_id":"` + canarySecret + `"}]`,
		CredentialRegistryFile:    "/run/secrets/" + canarySecret,
		CredentialVerifierKeyFile: "/run/secrets/" + canarySecret,
		TLSKeyPath:                "/etc/tls/" + canarySecret,
	}
}

// captureAuditStore captures every audit event the real tool path persists so
// the test can assert on the serialized form the operator would read.
type captureAuditStore struct {
	mu     sync.Mutex
	events []*database.AuditEvent
}

func (s *captureAuditStore) Insert(ctx context.Context, e *database.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func (s *captureAuditStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

func (s *captureAuditStore) serialized() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	lines := make([]string, 0, len(s.events))
	for _, e := range s.events {
		raw, _ := json.Marshal(e)
		lines = append(lines, string(raw))
	}
	return strings.Join(lines, "\n")
}

// waitForEvent blocks until the async audit path has persisted at least one
// record (or fails the test). Non-sensitive decisions are enqueued and written
// by the logger's background worker, so a bare read would be racy.
func (s *captureAuditStore) waitForEvent(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s.count() > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected the real tool path to persist an audit decision record")
}

// TestSecretLeakRedactionAcrossRealPaths drives three real paths that touch
// config/secrets — a resource read, an authorized tool call (which produces a
// real audit decision), and a denied error response — and asserts the canary
// never appears in logs, audit records, error responses, or serialized output.
func TestSecretLeakRedactionAcrossRealPaths(t *testing.T) {
	store := &captureAuditStore{}
	auditLogger := audit.NewLoggerWithStore(16, store)
	defer auditLogger.Stop()

	s := &MCPServer{
		config:   canaryLeakConfig(),
		audit:    auditLogger,
		sessions: make(map[string]*models.Session),
	}

	var configText, resultJSON, deniedJSON string

	logs := captureLogs(t, func() {
		// Real path 1: read guardrail://config, the resource that projects the
		// live configuration (and, if redaction regressed, its secrets).
		contents := s.readConfigResourceContents("guardrail://config")
		if len(contents) != 1 {
			t.Fatalf("expected 1 config resource content, got %d", len(contents))
		}
		tc, ok := contents[0].(mcp.TextResourceContents)
		if !ok {
			t.Fatalf("unexpected resource content type %T", contents[0])
		}
		configText = tc.Text

		// Real path 2: an authorized mutate tool call carrying the canary at
		// several argument nesting levels. The authorization boundary runs and
		// emits a real audit decision record.
		args := map[string]interface{}{
			"user_id":    "user-1",
			"password":   canarySecret,
			"api_key":    canarySecret,
			"token":      canarySecret,
			"project_id": "proj-1",
			"nested": map[string]interface{}{
				"credential": canarySecret,
				"deeper":     map[string]interface{}{"authorization": canarySecret},
			},
		}
		res, err := s.handleToolCall(authzTestContext(), "guardrail_init_session", args)
		if err != nil {
			t.Fatalf("handleToolCall init_session: %v", err)
		}
		if res == nil {
			t.Fatal("expected a tool result")
		}
		raw, _ := json.Marshal(res)
		resultJSON = string(raw)

		// Real path 3: a stable denied error response.
		denyRes, err := s.handleToolCall(authzTestContext(), "no_such_tool", map[string]interface{}{
			"password": canarySecret,
		})
		if err != nil {
			t.Fatalf("handleToolCall deny: %v", err)
		}
		if denyRes == nil || !denyRes.IsError {
			t.Fatal("unknown tool must yield a stable non-leaking denial error response")
		}
		draw, _ := json.Marshal(denyRes)
		deniedJSON = string(draw)
	})

	store.waitForEvent(t)

	// Positive controls: the paths produced real output and audit records, so
	// the redaction assertions below are not vacuous.
	if configText == "" || resultJSON == "" || deniedJSON == "" {
		t.Fatal("a real path produced no serialized output")
	}
	if !strings.Contains(configText, "schema_version") {
		t.Fatalf("config resource projection looks empty: %q", configText)
	}
	auditJSON := store.serialized()
	if auditJSON == "" {
		t.Fatal("no audit records captured")
	}

	surfaces := map[string]string{
		"logs":            logs,
		"config resource": configText,
		"tool result":     resultJSON,
		"denial response": deniedJSON,
		"audit records":   auditJSON,
	}
	for name, text := range surfaces {
		if strings.Contains(text, canarySecret) {
			t.Fatalf("canary secret leaked into %s: %s", name, text)
		}
	}
}
