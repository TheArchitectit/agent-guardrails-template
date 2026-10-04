package mcp

import (
	"encoding/json"
	"github.com/mark3labs/mcp-go/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
)

func TestRequireBearer(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	cases := []struct {
		name, key, header string
		want              int
	}{
		{"valid", "secret-key", "Bearer secret-key", 200},
		{"case-insensitive scheme", "secret-key", "bearer secret-key", 200},
		{"missing header", "secret-key", "", 401},
		{"wrong key", "secret-key", "Bearer nope", 401},
		{"wrong scheme", "secret-key", "Basic secret-key", 401},
		{"empty configured key fails closed", "", "Bearer ", 401},
		{"empty configured key, no header", "", "", 401},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			requireBearer(c.key, ok).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("got %d want %d", rec.Code, c.want)
			}
		})
	}
}

func TestRequireBearerWithRegistry(t *testing.T) {
	const verifierKey = "mcp-test-verifier-key-0123456789"

	raw, err := json.Marshal([]auth.Record{{
		CredentialID: "cred-1",
		PrincipalID:  "principal-alpha",
		Verifier:     auth.Digest(verifierKey, "registered-secret"),
	}})
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	registry, err := auth.LoadFromSources(string(raw), "", verifierKey)
	if err != nil {
		t.Fatalf("LoadFromSources: %v", err)
	}

	cases := []struct {
		name, key, header string
		want              int
	}{
		{"registered credential accepted", "secret-key", "Bearer registered-secret", 200},
		{"legacy key keeps its surface", "secret-key", "Bearer secret-key", 200},
		{"unknown credential denied", "secret-key", "Bearer nope", 401},
		{"missing header denied", "secret-key", "", 401},
		{"empty configured key fails closed", "", "Bearer registered-secret", 401},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			requireBearerWithRegistry(c.key, registry, okHandler()).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("got %d want %d", rec.Code, c.want)
			}
		})
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
}

func TestMCPEndpointBehindBearer(t *testing.T) {
	inner := server.NewStreamableHTTPServer(server.NewMCPServer("t", "1"), server.WithEndpointPath("/mcp"), server.WithStateLess(true))
	mux := http.NewServeMux()
	mux.Handle("/mcp", requireBearer("secret-key", inner))
	ts := httptest.NewServer(mux)
	defer ts.Close()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
	do := func(auth string) int {
		req, _ := http.NewRequest("POST", ts.URL+"/mcp", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if got := do(""); got != 401 {
		t.Fatalf("unauthenticated: got %d want 401", got)
	}
	if got := do("Bearer secret-key"); got != 200 {
		t.Fatalf("authenticated initialize: got %d want 200", got)
	}
}
