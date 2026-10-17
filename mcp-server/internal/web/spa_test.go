package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
)

// newSPATestRoot builds a temporary web root with a shell and one real asset,
// plus a canary file OUTSIDE the root that a traversal would reach.
func newSPATestRoot(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "js"), 0o755); err != nil {
		t.Fatalf("mkdir js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("SPA_SHELL"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "js", "app.js"), []byte("REAL_ASSET"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	// A file beside the root, never inside it.
	if err := os.WriteFile(filepath.Join(filepath.Dir(root), "OUTSIDE_CANARY.txt"), []byte("TOP_SECRET_CANARY"), 0o644); err != nil {
		t.Fatalf("write canary: %v", err)
	}
	return root
}

// spaServer mirrors the shipped Web UI registration (registerRoutes): the SPA
// route is registered before the two Static mounts, so the harness exercises
// the real route precedence rather than the SPA route in isolation.
func spaServer(t *testing.T, root string) *httptest.Server {
	t.Helper()
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/web/*", newSPAHandler(root))
	e.Static("/web", root)
	e.Static("/", root)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv
}

// TestSPAHandlerRefusesPathTraversal is the Spec 19 R19-09 path-bypass control
// for the Web UI shell: an unauthenticated request must never reach a file
// outside the web root, in any encoding. Verified live before the fix:
// `GET /web/../OUTSIDE_CANARY.txt` returned the canary.
func TestSPAHandlerRefusesPathTraversal(t *testing.T) {
	root := newSPATestRoot(t)
	srv := spaServer(t, root)

	cases := []struct {
		name string
		path string
	}{
		{"parent segment", "/web/../OUTSIDE_CANARY.txt"},
		{"encoded separator", "/web/..%2fOUTSIDE_CANARY.txt"},
		{"encoded dots", "/web/%2e%2e/OUTSIDE_CANARY.txt"},
		{"double encoded", "/web/%252e%252e/OUTSIDE_CANARY.txt"},
		{"deep parent", "/web/../../OUTSIDE_CANARY.txt"},
		{"dot segment then file", "/web/./../OUTSIDE_CANARY.txt"},
		{"backslash-ish", "/web/..%5cOUTSIDE_CANARY.txt"},
		{"absolute-ish", "/web//etc/passwd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tc.path, err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if containsString(string(body), "TOP_SECRET_CANARY") {
				t.Fatalf("GET %s: traversal escaped the web root (body=%q)", tc.path, string(body))
			}
		})
	}
}

// TestSPAHandlerServesRealAssetsAndShell is the positive control: the
// confinement must not break the shell or the real assets (non-vacuous).
func TestSPAHandlerServesRealAssetsAndShell(t *testing.T) {
	root := newSPATestRoot(t)
	srv := spaServer(t, root)

	cases := []struct{ path, want string }{
		{"/web/index.html", "SPA_SHELL"},
		{"/web/js/app.js", "REAL_ASSET"},
		{"/web/dashboard", "SPA_SHELL"},  // client-side route -> shell
		{"/web/missing.js", "SPA_SHELL"}, // absent file -> shell fallback
		{"/web/", "SPA_SHELL"},           // directory -> shell
	}
	for _, tc := range cases {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !containsString(string(body), tc.want) {
			t.Fatalf("GET %s: got %d body=%q, want %q", tc.path, resp.StatusCode, string(body), tc.want)
		}
	}
}

func containsString(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
