package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
)

// syncBuffer is a concurrency-safe log sink (the logger may write from a
// different goroutine than the reader).
type syncBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestRedactRequestURI pins the redaction rule: the path survives, the query is
// withheld. A name-based allowlist would fail open on the first unlisted
// parameter, so the whole query is replaced.
func TestRedactRequestURI(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no query is unchanged", "/api/rules", "/api/rules"},
		{"empty is empty", "", ""},
		{"query is withheld", "/api/search?q=hi&token=SECRET_CANARY", "/api/search?<redacted>"},
		{"api key param is withheld", "/api/rules?api_key=SECRET_CANARY", "/api/rules?<redacted>"},
		{"bare question mark is withheld", "/api/rules?", "/api/rules?<redacted>"},
		{"path is preserved verbatim", "/a/b/c?x=1", "/a/b/c?<redacted>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RedactRequestURI(tc.in); got != tc.want {
				t.Fatalf("RedactRequestURI(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRequestLoggerWithholdsQueryString drives the REAL request logger through
// the real echo chain with a secret in the query string and asserts the log
// line names the path but never the secret. Spec 17 R17-08.
func TestRequestLoggerWithholdsQueryString(t *testing.T) {
	const canary = "SECRET_CANARY_QUERY_b7f3a9c1"

	buf := &syncBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	e := echo.New()
	e.Use(RequestLoggerWithConfig(RequestLoggerConfig{
		Skipper:       func(c echo.Context) bool { return false },
		LogLevel:      slog.LevelInfo,
		LogLevelError: slog.LevelError,
	}))
	e.GET("/api/search", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/search?token="+canary+"&q=hi", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	out := buf.String()
	// Positive control: the logger really ran and recorded the request.
	if !strings.Contains(out, "/api/search") {
		t.Fatalf("logger did not record the request path; log was %q", out)
	}
	if !strings.Contains(out, "HTTP Request") {
		t.Fatalf("logger did not emit a request record; log was %q", out)
	}
	if strings.Contains(out, canary) {
		t.Fatalf("query-string secret leaked into the request log: %s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Fatalf("expected the withheld-query marker in the log: %s", out)
	}
}
