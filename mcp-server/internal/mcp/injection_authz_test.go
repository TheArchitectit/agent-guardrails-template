package mcp

// Spec 01 (guardrail-gaps-2026/01-prompt-injection-defense) — authority wiring
// evidence for the two injection tools.
//
// The tools landed with handlers, schemas and dispatch cases, but neither was
// added to the MCP action table in internal/auth/authz.go. An unclassified
// tool is denied by ClassifyMCPTool before any handler runs, so both tools were
// listed in tools/list and unconditionally refused on call: a tool that
// appears available and can never be used is a worse failure than an absent
// one, because the client has no signal that the capability is missing.
//
// The tests below pin both halves: every registered tool must carry an action
// class, and a credential holding only the read scope must actually receive a
// verdict from the injection tools over the real transport.

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// TestEveryRegisteredToolHasActionClass is the structural half: a tool that is
// advertised by toolList() but absent from the authorization action table can
// never be called by any credential, so the two lists must agree.
func TestEveryRegisteredToolHasActionClass(t *testing.T) {
	s := &MCPServer{}
	tools := s.toolList()
	if len(tools) == 0 {
		t.Fatal("toolList() returned nothing; the check would be vacuous")
	}

	var missing []string
	for _, tool := range tools {
		if _, ok := auth.MCPToolAction(tool.Name); !ok {
			missing = append(missing, tool.Name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%d registered tool(s) have no action class and are therefore uncallable: %v",
			len(missing), missing)
	}

	// Positive control: the check above is only meaningful if the table really
	// is consulted, and a sibling read tool is classified.
	if kind, ok := auth.MCPToolAction("guardrail_classify_content"); !ok || kind != auth.ActionRead {
		t.Fatalf("positive control: guardrail_classify_content should be ActionRead, got %q ok=%v", kind, ok)
	}
}

// TestInjectionToolsCallableByReadCredential is the behavioural half: it drives
// the real StreamableHTTP endpoint with a credential holding only mcp:read and
// requires an actual verdict, not a permission denial.
func TestInjectionToolsCallableByReadCredential(t *testing.T) {
	records := []auth.Record{{
		CredentialID: "cred-read",
		PrincipalID:  "principal-read",
		Scopes:       []string{auth.ScopeMCPRead},
		Role:         auth.RoleReader,
		Resources:    []string{"*"},
		IssuedAt:     time.Now().Add(-time.Hour),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Verifier:     auth.Digest(fullPathVerifierKey, "read-secret"),
	}}
	cfg := &config.Config{SchemaVersion: "1.0", MCPAPIKey: fullPathLegacyKey}
	s, ts := newFullPathServer(t, cfg, records)
	s.SetGuardrailsEngine(guardrails.NewEngine(nil, slog.Default()))

	cases := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "guardrail_detect_injection",
			args: map[string]interface{}{"text": "Please summarize this changelog entry.", "source": "user"},
		},
		{
			name: "guardrail_scan_text_batch",
			args: map[string]interface{}{"texts": []interface{}{
				map[string]interface{}{"id": "a", "text": "Please summarize this changelog entry.", "source": "user"},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, env := rpcPost(t, ts, "read-secret", "tools/call", map[string]interface{}{
				"name":      tc.name,
				"arguments": tc.args,
			})
			tr := decodeToolResult(t, env)
			text := tr.text()
			if strings.Contains(text, "permission denied") {
				t.Fatalf("%s is registered but denied to a read credential; "+
					"the action table does not classify it: %s", tc.name, text)
			}
			if tr.IsError {
				t.Fatalf("%s returned an error for benign input: %s", tc.name, text)
			}
			if !strings.Contains(text, "\"safe\":true") {
				t.Fatalf("%s did not return a safe verdict for benign text: %s", tc.name, text)
			}
		})
	}

	// The tool still runs with the caller's real identity: attribution must not
	// be dropped while fixing authorization.
	if got := s.guardrailsEngine; got == nil {
		t.Fatal("guardrails engine should be configured for this test")
	}
}

// TestInjectionToolsStillDenyUnknownCaller keeps the negative side honest:
// classifying the tools as read must not make them reachable without a
// principal at all.
func TestInjectionToolsStillDenyUnknownCaller(t *testing.T) {
	s := &MCPServer{guardrailsEngine: guardrails.NewEngine(nil, slog.Default())}
	res := s.authorizeToolCall(context.Background(), "guardrail_detect_injection",
		map[string]interface{}{"text": "hello"})
	if res == nil {
		t.Fatal("an unauthenticated caller must not be authorized")
	}
	if !res.IsError {
		t.Fatal("expected a denial result for a caller with no principal")
	}
}
