package mcp

// Spec 23 — live tools/list and resources/list sequence over the production
// Streamable HTTP transport. Closes the R20.3 (spec 10 §4.2.1) gap: the real
// transport was exercised for initialize and tools/call and resources/read but
// never for tool or resource *listing*.
//
// These tests reuse the same httptest + requireBearerWithRegistry wiring as
// streamable_http_authz_test.go and never call handlers directly.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

type listToolsResult struct {
	Tools []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		InputSchema struct {
			Type       string                     `json:"type"`
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"inputSchema"`
	} `json:"tools"`
}

type listResourcesResult struct {
	Resources []struct {
		URI  string `json:"uri"`
		Name string `json:"name"`
	} `json:"resources"`
}

// TestStreamableHTTPListingSequence drives initialize → tools/list →
// resources/list → tools/call (success) → tools/call (rejected) over the real
// Streamable HTTP endpoint, asserting discovery against the live registry
// rather than a hard-coded list.
func TestStreamableHTTPListingSequence(t *testing.T) {
	cfg := &config.Config{SchemaVersion: "1.0", MCPAPIKey: fullPathLegacyKey}
	s, ts := newFullPathServer(t, cfg, recordsForFullPath())

	t.Run("initialize advertises tool and resource capabilities", func(t *testing.T) {
		code, env := rpcPost(t, ts, "allow-secret", "initialize", map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]interface{}{"name": "listing-test", "version": "1"},
		})
		if code != http.StatusOK {
			t.Fatalf("initialize status = %d, want 200", code)
		}
		if len(env.Result) == 0 {
			t.Fatalf("no initialize result (error=%s)", string(env.Error))
		}
		var init struct {
			ProtocolVersion string                     `json:"protocolVersion"`
			Capabilities    map[string]json.RawMessage `json:"capabilities"`
		}
		if err := json.Unmarshal(env.Result, &init); err != nil {
			t.Fatalf("decode initialize result %q: %v", string(env.Result), err)
		}
		if init.ProtocolVersion == "" {
			t.Fatal("initialize returned no protocolVersion")
		}
		if _, ok := init.Capabilities["tools"]; !ok {
			t.Fatalf("initialize capabilities omit tools: %v", init.Capabilities)
		}
		if _, ok := init.Capabilities["resources"]; !ok {
			t.Fatalf("initialize capabilities omit resources: %v", init.Capabilities)
		}
	})

	t.Run("tools/list matches the live registry", func(t *testing.T) {
		code, env := rpcPost(t, ts, "allow-secret", "tools/list", map[string]interface{}{})
		if code != http.StatusOK {
			t.Fatalf("tools/list status = %d, want 200", code)
		}
		if len(env.Result) == 0 {
			t.Fatalf("no tools/list result (error=%s)", string(env.Error))
		}
		var got listToolsResult
		if err := json.Unmarshal(env.Result, &got); err != nil {
			t.Fatalf("decode tools/list %q: %v", string(env.Result), err)
		}
		want := s.toolList()
		if len(got.Tools) != len(want) {
			t.Fatalf("tools/list returned %d tools, live registry has %d", len(got.Tools), len(want))
		}
		live := map[string]bool{}
		for _, tool := range want {
			live[tool.Name] = true
		}
		for _, tool := range got.Tools {
			if !live[tool.Name] {
				t.Errorf("tools/list returned %q which is not in the live registry", tool.Name)
			}
			if tool.InputSchema.Type != "object" {
				t.Errorf("tool %q advertised input schema type %q, want object", tool.Name, tool.InputSchema.Type)
			}
		}
	})

	t.Run("resources/list returns the registered resources", func(t *testing.T) {
		code, env := rpcPost(t, ts, "allow-secret", "resources/list", map[string]interface{}{})
		if code != http.StatusOK {
			t.Fatalf("resources/list status = %d, want 200", code)
		}
		if len(env.Result) == 0 {
			t.Fatalf("no resources/list result (error=%s)", string(env.Error))
		}
		var got listResourcesResult
		if err := json.Unmarshal(env.Result, &got); err != nil {
			t.Fatalf("decode resources/list %q: %v", string(env.Result), err)
		}
		if len(got.Resources) == 0 {
			t.Fatal("resources/list returned no resources but the server registered fixed resources")
		}
		found := false
		for _, r := range got.Resources {
			if r.URI == "guardrail://config" {
				found = true
			}
		}
		if !found {
			t.Errorf("resources/list did not include guardrail://config: %v", got.Resources)
		}
	})

	t.Run("tools/call success then rejection over the same transport", func(t *testing.T) {
		initArgs := map[string]interface{}{
			"name":      "guardrail_init_session",
			"arguments": map[string]interface{}{"user_id": "user-1"},
		}
		before := len(s.sessions)
		code, env := rpcPost(t, ts, "allow-secret", "tools/call", initArgs)
		if code != http.StatusOK {
			t.Fatalf("authorized tools/call status = %d, want 200", code)
		}
		if tr := decodeToolResult(t, env); tr.IsError {
			t.Fatalf("authorized call returned error result: %s", tr.text())
		}
		if len(s.sessions) != before+1 {
			t.Fatalf("authorized call produced no side effect")
		}

		before = len(s.sessions)
		code, env = rpcPost(t, ts, "noscope-secret", "tools/call", initArgs)
		if code != http.StatusOK {
			t.Fatalf("rejected tools/call status = %d, want 200", code)
		}
		tr := decodeToolResult(t, env)
		if !tr.IsError || tr.text() != auth.FormatDenyError() {
			t.Fatalf("want stable permission denial, got isError=%v text=%q", tr.IsError, tr.text())
		}
		if len(s.sessions) != before {
			t.Fatalf("rejected call produced a side effect")
		}
	})

	t.Run("tools/list unauthenticated is rejected before dispatch", func(t *testing.T) {
		code, _ := rpcPost(t, ts, "", "tools/list", map[string]interface{}{})
		if code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated tools/list status = %d, want 401", code)
		}
	})
}
