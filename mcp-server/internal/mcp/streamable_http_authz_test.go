package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/vision"
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
	return fullPathServerWithVision(cfg, nil)
}

func fullPathServerWithVision(cfg *config.Config, visionTools *VisionToolSet) *MCPServer {
	s := &MCPServer{
		config:      cfg,
		sessions:    make(map[string]*models.Session),
		visionTools: visionTools,
	}
	s.mcpServer = server.NewMCPServer("full-path-test", "1.0",
		server.WithResourceCapabilities(true, true))
	s.setupHandlers()
	return s
}

// newFullPathServer wires a full-path httptest MCP endpoint behind the real
// credential-registry middleware.
func newFullPathServer(t *testing.T, cfg *config.Config, records []auth.Record) (*MCPServer, *httptest.Server) {
	return newFullPathServerWithVision(t, cfg, records, nil)
}

func newFullPathServerWithVision(t *testing.T, cfg *config.Config, records []auth.Record, visionTools *VisionToolSet) (*MCPServer, *httptest.Server) {
	t.Helper()
	raw, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	registry, err := auth.LoadFromSources(string(raw), "", fullPathVerifierKey)
	if err != nil {
		t.Fatalf("LoadFromSources: %v", err)
	}
	s := fullPathServerWithVision(cfg, visionTools)
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
			IssuedAt:     time.Now().Add(-time.Hour),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			Verifier:     auth.Digest(fullPathVerifierKey, "allow-secret"),
		},
		{
			// Has mutate role + resource grant, but only a read scope.
			CredentialID: "cred-no-scope",
			PrincipalID:  "principal-no-scope",
			Scopes:       []string{auth.ScopeMCPRead},
			Role:         auth.RoleAdministrator,
			Resources:    []string{"*"},
			IssuedAt:     time.Now().Add(-time.Hour),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			Verifier:     auth.Digest(fullPathVerifierKey, "noscope-secret"),
		},
		{
			// Has the mutate scope but a role that does not allow mutation.
			CredentialID: "cred-no-role",
			PrincipalID:  "principal-no-role",
			Scopes:       []string{auth.ScopeMCPMutate, auth.ScopeMCPRead},
			Role:         auth.RoleReader,
			Resources:    []string{"*"},
			IssuedAt:     time.Now().Add(-time.Hour),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			Verifier:     auth.Digest(fullPathVerifierKey, "norole-secret"),
		},
		{
			// Admin capability but only granted project proj-a.
			CredentialID: "cred-scoped",
			PrincipalID:  "principal-scoped",
			Scopes:       []string{auth.ScopeAdminManage, auth.ScopeMCPRead, auth.ScopeMCPMutate},
			Role:         auth.RoleAdministrator,
			Resources:    []string{"proj-a"},
			IssuedAt:     time.Now().Add(-time.Hour),
			ExpiresAt:    time.Now().Add(24 * time.Hour),
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
				"project_id":   "proj-b",
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

type countingVisionClient struct{ calls int }

func (c *countingVisionClient) ReviewImage(_ context.Context, _ string, _ string, _ []vision.Finding) (*vision.ReviewResponse, error) {
	c.calls++
	return &vision.ReviewResponse{Confidence: 1, BackendUsed: "synthetic", ModelUsed: "fixture"}, nil
}

func (c *countingVisionClient) HealthCheck(context.Context) (vision.HealthStatus, error) {
	return vision.HealthStatus{Backend: "synthetic", Healthy: true}, nil
}

// TestOptionalVisionToolsUseCommonAuthorization proves the enabled Vision
// tools are registered on the real StreamableHTTP boundary and pass through
// common operation authorization before the vision handler can read or mutate
// report state.
func TestOptionalVisionToolsUseCommonAuthorization(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "synthetic.png")
	imageFile, err := os.Create(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(imageFile, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		_ = imageFile.Close()
		t.Fatal(err)
	}
	if err := imageFile.Close(); err != nil {
		t.Fatal(err)
	}

	storage, err := vision.NewStorage(filepath.Join(root, "vision.sqlite"))
	if err != nil {
		t.Fatalf("create synthetic Vision store: %v", err)
	}
	if err := storage.CreateReview(&vision.Review{
		ID: "review-42", ScreenshotPath: imagePath,
		Status: vision.StatusCompleted, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed synthetic Vision review: %v", err)
	}
	client := &countingVisionClient{}
	engine := vision.NewReviewEngine(client, storage, vision.ReviewEngineConfig{})
	visionTools := &VisionToolSet{enabled: true, engine: engine}
	cfg := &config.Config{SchemaVersion: "1.0", MCPAPIKey: fullPathLegacyKey}
	_, ts := newFullPathServerWithVision(t, cfg, recordsForFullPath(), visionTools)

	t.Run("authorized read reaches report handler", func(t *testing.T) {
		code, env := rpcPost(t, ts, "allow-secret", "tools/call", map[string]interface{}{
			"name": "vision_get_report", "arguments": map[string]interface{}{"review_id": "review-42"},
		})
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		result := decodeToolResult(t, env)
		if result.IsError || !strings.Contains(result.text(), "review-42") {
			t.Fatalf("authorized report read did not return the synthetic report: isError=%v text=%q", result.IsError, result.text())
		}
	})

	t.Run("denied mutation never invokes provider or writes report state", func(t *testing.T) {
		before, err := storage.ListIterations("review-42")
		if err != nil {
			t.Fatal(err)
		}
		code, env := rpcPost(t, ts, "noscope-secret", "tools/call", map[string]interface{}{
			"name": "vision_iterate_review", "arguments": map[string]interface{}{"review_id": "review-42"},
		})
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		result := decodeToolResult(t, env)
		if !result.IsError || result.text() != auth.FormatDenyError() {
			t.Fatalf("want stable permission denial, got isError=%v text=%q", result.IsError, result.text())
		}
		after, err := storage.ListIterations("review-42")
		if err != nil {
			t.Fatal(err)
		}
		if client.calls != 0 || len(after) != len(before) {
			t.Fatalf("denied vision operation reached provider or mutated storage: calls=%d iterations=%d -> %d", client.calls, len(before), len(after))
		}
	})

	t.Run("allowed mutation is audit-gated before dispatch", func(t *testing.T) {
		code, env := rpcPost(t, ts, "allow-secret", "tools/call", map[string]interface{}{
			"name": "vision_iterate_review", "arguments": map[string]interface{}{"review_id": "review-42"},
		})
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		result := decodeToolResult(t, env)
		if !result.IsError || result.text() != "authorization audit unavailable" {
			t.Fatalf("missing durable decision audit must block Vision mutation, got isError=%v text=%q", result.IsError, result.text())
		}
		iterations, err := storage.ListIterations("review-42")
		if err != nil {
			t.Fatal(err)
		}
		if client.calls != 0 || len(iterations) != 0 {
			t.Fatalf("audit failure reached provider or mutated storage: calls=%d iterations=%d", client.calls, len(iterations))
		}
	})
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
