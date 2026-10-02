package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
