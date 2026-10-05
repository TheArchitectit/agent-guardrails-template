package mcp

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// fakeArgSecret is planted at multiple nesting levels in tool arguments. It
// must never appear in log output (S-A0 / R16-05).
const fakeArgSecret = "FAKE_ARG_SECRET_MARKER_9f3a2b"

func captureLogs(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	prev := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(prev)
	fn()
	return buf.String()
}

func privacyServer() *MCPServer {
	return &MCPServer{
		config:   &config.Config{SchemaVersion: "1.0"},
		sessions: make(map[string]*models.Session),
	}
}

// authzTestContext returns a context carrying a server-controlled principal
// authorized for the tools under test. Identity is never taken from arguments.
func authzTestContext() context.Context {
	ctx := context.WithValue(context.Background(), ctxKeyRequestID, "req-123")
	ctx = context.WithValue(ctx, ctxKeyPrincipalID, "principal-alpha")
	ctx = context.WithValue(ctx, ctxKeyCredentialID, "cred-1")
	ctx = context.WithValue(ctx, ctxKeyCaller, auth.Caller{
		PrincipalID:  "principal-alpha",
		CredentialID: "cred-1",
		Scopes:       []string{auth.ScopeMCPRead, auth.ScopeMCPValidate, auth.ScopeMCPMutate},
		Role:         auth.RoleAdministrator,
		Resources:    []string{"*"},
	})
	return ctx
}

// TestArgumentPrivacyFakeSecretAbsentFromLogs seeds a fake secret at several
// argument nesting levels and asserts it is absent from captured log output
// for both success and error paths.
func TestArgumentPrivacyFakeSecretAbsentFromLogs(t *testing.T) {
	s := privacyServer()

	nestedArgs := map[string]interface{}{
		"user_id":    "user-1",
		"password":   fakeArgSecret,
		"api_key":    fakeArgSecret,
		"secret":     fakeArgSecret,
		"token":      fakeArgSecret,
		"credential": fakeArgSecret,
		"nested": map[string]interface{}{
			"password": fakeArgSecret,
			"deeper": map[string]interface{}{
				"authorization": fakeArgSecret,
			},
			"list": []interface{}{fakeArgSecret, map[string]interface{}{"key": fakeArgSecret}},
		},
		"reason": fakeArgSecret,
		"path":   "/tmp/" + fakeArgSecret,
	}

	// Success path (guardrail_init_session).
	out := captureLogs(t, func() {
		res, err := s.handleToolCall(authzTestContext(), "guardrail_init_session", nestedArgs)
		if err != nil {
			t.Fatalf("handleToolCall: %v", err)
		}
		if res == nil {
			t.Fatal("expected a result")
		}
	})
	if strings.Contains(out, fakeArgSecret) {
		t.Fatalf("fake secret leaked into success-path logs: %s", out)
	}

	// Error path (unknown tool) — stable permission-denied result, no leak.
	out = captureLogs(t, func() {
		res, err := s.handleToolCall(authzTestContext(), "no_such_tool", nestedArgs)
		if err != nil {
			t.Fatalf("handleToolCall: %v", err)
		}
		if res == nil || !res.IsError {
			t.Fatal("unknown tool should yield a stable permission-denied result")
		}
	})
	if strings.Contains(out, fakeArgSecret) {
		t.Fatalf("fake secret leaked into error-path logs: %s", out)
	}

	// Rejected-tool path (error result, no Go error).
	out = captureLogs(t, func() {
		res, err := s.handleToolCall(authzTestContext(), "guardrail_get_context", map[string]interface{}{
			"path": fakeArgSecret,
			"note": fakeArgSecret,
		})
		if err != nil {
			t.Fatalf("handleToolCall: %v", err)
		}
		if res == nil {
			t.Fatal("expected a result")
		}
	})
	if strings.Contains(out, fakeArgSecret) {
		t.Fatalf("fake secret leaked into rejected-path logs: %s", out)
	}
}

// TestArgumentPrivacyLogsApprovedFieldsOnly asserts the tool-call log carries
// the approved audit fields and does not dump raw argument maps.
func TestArgumentPrivacyLogsApprovedFieldsOnly(t *testing.T) {
	s := privacyServer()
	ctx := authzTestContext()

	out := captureLogs(t, func() {
		_, _ = s.handleToolCall(ctx, "guardrail_init_session", map[string]interface{}{
			"user_id":    "user-1",
			"project_id": "proj-42",
			"password":   fakeArgSecret,
		})
	})

	for _, want := range []string{
		"operation=guardrail_init_session",
		"request_id=req-123",
		"principal_id=principal-alpha",
		"credential_id=cred-1",
		"resource=proj-42",
		"outcome=success",
		"policy_version=authz-1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("log missing approved field %q; got: %s", want, out)
		}
	}
	if strings.Contains(out, fakeArgSecret) {
		t.Fatalf("raw argument content leaked: %s", out)
	}
	if strings.Contains(out, "args=") || strings.Contains(out, "password=") {
		t.Fatalf("raw argument map appears in log: %s", out)
	}
}

// TestSafeResourceIDRejectsSecrets ensures the resource identifier extractor
// never forwards free-form or secret-looking values.
func TestSafeResourceIDRejectsSecrets(t *testing.T) {
	cases := []struct {
		name string
		args map[string]interface{}
		want string
	}{
		{"allowlisted opaque id", map[string]interface{}{"project_id": "proj-42"}, "proj-42"},
		{"missing key", map[string]interface{}{"other": "x"}, ""},
		{"secret in allowlisted key with spaces", map[string]interface{}{"project_id": "has space " + fakeArgSecret}, ""},
		{"secret as free-form field", map[string]interface{}{"password": fakeArgSecret}, ""},
		{"nested secret ignored", map[string]interface{}{"nested": map[string]interface{}{"project_id": fakeArgSecret}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safeResourceID(tc.args); got != tc.want {
				t.Fatalf("safeResourceID = %q, want %q", got, tc.want)
			}
		})
	}
}
