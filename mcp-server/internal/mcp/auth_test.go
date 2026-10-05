package mcp

import (
	"encoding/json"
	"errors"
	"github.com/mark3labs/mcp-go/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
)

// errRegistryBroken is a stand-in for "configured registry failed to load".
var errRegistryBroken = errors.New("credential registry configured but failed to load")

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
		// Registry-only cutover: an absent legacy key does not block registered
		// callers, and it authenticates nobody else.
		{"registered credential works without legacy key", "", "Bearer registered-secret", 200},
		{"absent legacy key authenticates nobody", "", "Bearer secret-key", 401},
		{"absent legacy key, unknown credential denied", "", "Bearer nope", 401},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			requireBearerWithRegistry(c.key, registry, nil, okHandler()).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("got %d want %d", rec.Code, c.want)
			}
		})
	}
}

// TestRequireBearerWithBrokenRegistryDeniesAll covers fail-closed behaviour:
// a registry that is configured but failed to load denies every caller,
// including the legacy key — there is no unrestricted fallback.
func TestRequireBearerWithBrokenRegistryDeniesAll(t *testing.T) {
	regErr := errRegistryBroken
	cases := []struct {
		name, key, header string
	}{
		{"legacy key denied", "secret-key", "Bearer secret-key"},
		{"any token denied", "secret-key", "Bearer anything"},
		{"empty key denied", "", "Bearer anything"},
		{"missing header denied", "secret-key", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			requireBearerWithRegistry(c.key, nil, regErr, okHandler()).ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("got %d want 401", rec.Code)
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
