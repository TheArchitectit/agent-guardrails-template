package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// These tests drive the real StreamableHTTP MCP endpoint through httptest —
// the same wiring Serve() uses (requireBearerWithRegistry wrapping the
// mcp-go StreamableHTTPServer) — and perform authenticated tools/call and
// resources/read over JSON-RPC. They prove authorization allow AND denial
// (missing scope / role / resource) plus argument privacy at the transport
// boundary, not by calling handleToolCall/readConfigResourceContents directly.
//
// Hermetic tool choice: guardrail_init_session is the closest tool that
// completes end-to-end without external dependencies (no database, validator,
// guardrails engine, or network). It is a mutate-class tool, so it exercises
// the scope × role × resource intersection with a real side effect (a session
// registered in the server's session map) that denial must not produce.
const (
	fullPathVerifierKey = "full-path-test-verifier-key-0123456789"
	fullPathLegacyKey   = "legacy-mcp-fullpath-key"
)

// fullPathServer builds a minimal MCPServer with the real tool/resource
// surface registered but no database, for hermetic transport tests.
func fullPathServer(cfg *config.Config) *MCPServer {
	s := &MCPServer{
		config:   cfg,
		sessions: make(map[string]*models.Session),
	}
	s.mcpServer = server.NewMCPServer("full-path-test", "1.0",
		server.WithResourceCapabilities(true, true))
	s.setupHandlers()
	return s
}

// newFullPathServer wires a full-path httptest MCP endpoint behind the real
// credential-registry middleware.
func newFullPathServer(t *testing.T, cfg *config.Config, records []auth.Record) (*MCPServer, *httptest.Server) {
	t.Helper()
	raw, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	registry, err := auth.LoadFromSources(string(raw), "", fullPathVerifierKey)
	if err != nil {
		t.Fatalf("LoadFromSources: %v", err)
	}
	s := fullPathServer(cfg)
	httpSrv := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", requireBearerWithRegistry(cfg.MCPAPIKey, registry, nil, httpSrv))
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return s, ts
}

type rpcEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// rpcPosts a JSON-RPC request to the endpoint and returns the HTTP status and
// decoded envelope (SSE or plain JSON, matching what the transport chooses).
func rpcPost(t *testing.T, ts *httptest.Server, token, method string, params map[string]interface{}) (int, rpcEnvelope) {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env rpcEnvelope
	if resp.StatusCode == http.StatusOK {
		payload := decodeBody(raw, resp.Header.Get("Content-Type"))
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &env); err != nil {
				t.Fatalf("decode rpc response from %q: %v", raw, err)
			}
		}
	}
	return resp.StatusCode, env
}

// decodeBody extracts the JSON-RPC payload from either a plain JSON body or
// the first SSE `data:` line of a text/event-stream response.
func decodeBody(raw []byte, contentType string) []byte {
	if strings.Contains(contentType, "text/event-stream") {
		sc := bufio.NewScanner(bytes.NewReader(raw))
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "data:") {
				return []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		return nil
	}
	return raw
}

type toolResultBody struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

func decodeToolResult(t *testing.T, env rpcEnvelope) toolResultBody {
	t.Helper()
	if len(env.Result) == 0 {
		t.Fatalf("no result in envelope (error=%s)", string(env.Error))
	}
	var tr toolResultBody
	if err := json.Unmarshal(env.Result, &tr); err != nil {
		t.Fatalf("decode tool result %q: %v", string(env.Result), err)
	}
	return tr
}

func (tr toolResultBody) text() string {
	if len(tr.Content) == 0 {
		return ""
	}
	return tr.Content[0].Text
}

// recordsForFullPath returns one credential per authorization outcome.
func recordsForFullPath() []auth.Record {
	return []auth.Record{
		{
			CredentialID: "cred-allow",
			PrincipalID:  "principal-allow",
			Scopes:       []string{auth.ScopeMCPMutate, auth.ScopeMCPRead, auth.ScopeMCPValidate},
			Role:         auth.RoleAdministrator,
			Resources:    []string{"*"},
			Verifier:     auth.Digest(fullPathVerifierKey, "allow-secret"),
		},
		{
			// Has mutate role + resource grant, but only a read scope.
			CredentialID: "cred-no-scope",
			PrincipalID:  "principal-no-scope",
			Scopes:       []string{auth.ScopeMCPRead},
			Role:         auth.RoleAdministrator,
			Resources:    []string{"*"},
			Verifier:     auth.Digest(fullPathVerifierKey, "noscope-secret"),
		},
		{
			// Has the mutate scope but a role that does not allow mutation.
			CredentialID: "cred-no-role",
			PrincipalID:  "principal-no-role",
			Scopes:       []string{auth.ScopeMCPMutate, auth.ScopeMCPRead},
			Role:         auth.RoleReader,
			Resources:    []string{"*"},
			Verifier:     auth.Digest(fullPathVerifierKey, "norole-secret"),
		},
		{
			// Admin capability but only granted project proj-a.
			CredentialID: "cred-scoped",
			PrincipalID:  "principal-scoped",
			Scopes:       []string{auth.ScopeAdminManage, auth.ScopeMCPRead, auth.ScopeMCPMutate},
			Role:         auth.RoleAdministrator,
			Resources:    []string{"proj-a"},
			Verifier:     auth.Digest(fullPathVerifierKey, "scoped-secret"),
		},
	}
}

// TestStreamableHTTPToolsCallAuthorizesAndDenies proves allow AND denial over
// the real StreamableHTTP tools/call path, with a zero-side-effect assertion
// on every denial.
func TestStreamableHTTPToolsCallAuthorizesAndDenies(t *testing.T) {
	cfg := &config.Config{SchemaVersion: "1.0", MCPAPIKey: fullPathLegacyKey}
	s, ts := newFullPathServer(t, cfg, recordsForFullPath())

	initArgs := map[string]interface{}{
		"name":      "guardrail_init_session",
		"arguments": map[string]interface{}{"user_id": "user-1", "environment": "test"},
	}

	t.Run("authorized mutate succeeds and has an effect", func(t *testing.T) {
		before := len(s.sessions)
		code, env := rpcPost(t, ts, "allow-secret", "tools/call", initArgs)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		tr := decodeToolResult(t, env)
		if tr.IsError {
			t.Fatalf("authorized call returned error result: %s", tr.text())
		}
		if len(s.sessions) != before+1 {
			t.Fatalf("session side effect missing: before=%d after=%d", before, len(s.sessions))
		}
	})

	t.Run("missing scope denies with zero side effect", func(t *testing.T) {
		before := len(s.sessions)
		code, env := rpcPost(t, ts, "noscope-secret", "tools/call", initArgs)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		tr := decodeToolResult(t, env)
		if !tr.IsError || tr.text() != auth.FormatDenyError() {
			t.Fatalf("want stable permission denial, got isError=%v text=%q", tr.IsError, tr.text())
		}
		if len(s.sessions) != before {
			t.Fatalf("denied call produced a side effect: %d -> %d", before, len(s.sessions))
		}
	})

	t.Run("wrong role denies with zero side effect", func(t *testing.T) {
		before := len(s.sessions)
		code, env := rpcPost(t, ts, "norole-secret", "tools/call", initArgs)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		tr := decodeToolResult(t, env)
		if !tr.IsError || tr.text() != auth.FormatDenyError() {
			t.Fatalf("want stable permission denial, got isError=%v text=%q", tr.IsError, tr.text())
		}
		if len(s.sessions) != before {
			t.Fatalf("denied call produced a side effect: %d -> %d", before, len(s.sessions))
		}
	})

	t.Run("out-of-grant resource denies", func(t *testing.T) {
		// Admin-class tool; the caller may act on proj-a only. The resource
		// target comes from the allowlisted project_id argument, never from
		// the caller — an identity claim in arguments is ignored.
		args := map[string]interface{}{
			"name": "guardrail_project_delete",
			"arguments": map[string]interface{}{
				"project_id": "proj-b",
				"principal_id": "principal-allow", // must be ignored
			},
		}
		code, env := rpcPost(t, ts, "scoped-secret", "tools/call", args)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		tr := decodeToolResult(t, env)
		if !tr.IsError || tr.text() != auth.FormatDenyError() {
			t.Fatalf("out-of-grant resource should deny, got isError=%v text=%q", tr.IsError, tr.text())
		}
	})

	t.Run("unauthenticated tools/call is rejected", func(t *testing.T) {
		code, _ := rpcPost(t, ts, "", "tools/call", initArgs)
		if code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401", code)
		}
	})
}

// TestStreamableHTTPResourceReadOmitsSecrets proves the guardrail://config
// resources/read path over the real endpoint returns only the non-secret
// projection, and that an unauthenticated read is rejected at the boundary.
func TestStreamableHTTPResourceReadOmitsSecrets(t *testing.T) {
	cfg := secretConfig()
	cfg.MCPAPIKey = fullPathLegacyKey
	s, ts := newFullPathServer(t, cfg, recordsForFullPath())
	_ = s

	read := map[string]interface{}{"uri": "guardrail://config"}

	code, env := rpcPost(t, ts, "allow-secret", "resources/read", read)
	if code != http.StatusOK {
		t.Fatalf("authenticated read status = %d, want 200", code)
	}
	if len(env.Result) == 0 {
		t.Fatalf("no result (error=%s)", string(env.Error))
	}
	var rr struct {
		Contents []struct {
			URI      string `json:"uri"`
			MIMEType string `json:"mimeType"`
			Text     string `json:"text"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(env.Result, &rr); err != nil {
		t.Fatalf("decode resource result %q: %v", string(env.Result), err)
	}
	if len(rr.Contents) != 1 {
		t.Fatalf("expected 1 contents entry, got %d", len(rr.Contents))
	}
	for _, marker := range secretMarkers {
		if strings.Contains(rr.Contents[0].Text, marker) {
			t.Fatalf("guardrail://config leaked secret marker %q over the endpoint", marker)
		}
	}
	var view map[string]interface{}
	if err := json.Unmarshal([]byte(rr.Contents[0].Text), &view); err != nil {
		t.Fatalf("config projection is not JSON: %v", err)
	}
	if view["schema_version"] != "1.0" {
		t.Fatalf("schema_version = %v, want 1.0", view["schema_version"])
	}

	code, _ = rpcPost(t, ts, "", "resources/read", read)
	if code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated read status = %d, want 401", code)
	}
}

// TestStreamableHTTPConfiguredBrokenRegistryDeniesAll proves the fail-closed
// contract (R16-03) on the real MCP HTTP endpoint: a registry that is
// configured but failed to load denies every caller — including the legacy
// key — with no unrestricted legacy fallback and no side effects.
func TestStreamableHTTPConfiguredBrokenRegistryDeniesAll(t *testing.T) {
	cfg := &config.Config{SchemaVersion: "1.0", MCPAPIKey: fullPathLegacyKey}
	s := fullPathServer(cfg)
	httpSrv := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", requireBearerWithRegistry(cfg.MCPAPIKey, nil, errRegistryBroken, httpSrv))
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	initArgs := map[string]interface{}{
		"name":      "guardrail_init_session",
		"arguments": map[string]interface{}{"user_id": "user-1"},
	}
	for _, token := range []string{fullPathLegacyKey, "registered-secret", ""} {
		code, _ := rpcPost(t, ts, token, "tools/call", initArgs)
		if code != http.StatusUnauthorized {
			t.Fatalf("broken registry, token %q: status = %d, want 401", token, code)
		}
	}
	if len(s.sessions) != 0 {
		t.Fatalf("broken registry produced side effects: %d sessions", len(s.sessions))
	}
}

// TestStreamableHTTPArgumentPrivacyAbsentFromLogs plants nested fake secrets in
// a tools/call over the real endpoint and asserts none appear in log output.
func TestStreamableHTTPArgumentPrivacyAbsentFromLogs(t *testing.T) {
	cfg := &config.Config{SchemaVersion: "1.0", MCPAPIKey: fullPathLegacyKey}
	_, ts := newFullPathServer(t, cfg, recordsForFullPath())

	args := map[string]interface{}{
		"name": "guardrail_init_session",
		"arguments": map[string]interface{}{
			"user_id":  "user-1",
			"password": fakeArgSecret,
			"nested": map[string]interface{}{
				"token":         fakeArgSecret,
				"authorization": fakeArgSecret,
			},
		},
	}

	out := captureLogs(t, func() {
		code, env := rpcPost(t, ts, "allow-secret", "tools/call", args)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if tr := decodeToolResult(t, env); tr.IsError {
			t.Fatalf("unexpected error result: %s", tr.text())
		}
	})
	if strings.Contains(out, fakeArgSecret) {
		t.Fatalf("fake argument secret leaked into full-path logs: %s", out)
	}
}
