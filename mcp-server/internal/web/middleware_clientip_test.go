package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/thearchitectit/guardrail-mcp/internal/clientip"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// newServerWithProxies builds the minimal server surface needed to assert the
// client-IP trust boundary is installed on the Echo instance.
func newServerWithProxies(t *testing.T, proxies []string) *echo.Echo {
	t.Helper()
	cfg := &config.Config{TrustedProxies: proxies}
	stance, err := clientip.NewStance(cfg.TrustedProxies)
	if err != nil {
		t.Fatalf("NewStance: %v", err)
	}
	e := echo.New()
	e.IPExtractor = stance.Extractor()
	return e
}

// TestRateLimitKeyIgnoresForgedForwardedHeaders covers Spec 12 acceptance
// criterion 7: with TRUSTED_PROXIES unset, a request carrying a forged
// X-Forwarded-For must derive the client IP from the socket peer, so two
// requests with different forged headers and one real peer share a bucket.
func TestRateLimitKeyIgnoresForgedForwardedHeaders(t *testing.T) {
	e := newServerWithProxies(t, nil)

	ipOf := func(forged string) string {
		req := httptest.NewRequest(http.MethodGet, "/api/rules", nil)
		req.RemoteAddr = "192.0.2.44:51234"
		if forged != "" {
			req.Header.Set("X-Forwarded-For", forged)
		}
		c := e.NewContext(req, httptest.NewRecorder())
		return c.RealIP()
	}

	first := ipOf("1.2.3.4")
	second := ipOf("5.6.7.8")
	plain := ipOf("")

	if first != "192.0.2.44" || second != "192.0.2.44" || plain != "192.0.2.44" {
		t.Fatalf("rate-limit key must be the socket peer for every forged header: got %q, %q, %q", first, second, plain)
	}
	if first != second {
		t.Fatal("forged headers must not create separate rate-limit buckets")
	}
}

// TestTrustedProxyPeerUsesForwardedHeader covers the positive half of
// criterion 7: a peer inside TRUSTED_PROXIES may supply the client IP.
func TestTrustedProxyPeerUsesForwardedHeader(t *testing.T) {
	e := newServerWithProxies(t, []string{"10.0.0.0/8"})

	req := httptest.NewRequest(http.MethodGet, "/api/rules", nil)
	req.RemoteAddr = "10.0.0.5:51234"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	c := e.NewContext(req, httptest.NewRecorder())
	if got := c.RealIP(); got != "203.0.113.7" {
		t.Fatalf("trusted peer client ip = %q, want 203.0.113.7", got)
	}
}

// TestForwardedHeaderAnomalyMiddlewarePassesThrough covers Sec. 3.4.4: an
// untrusted peer's forwarded header is logged as an anomaly but the request is
// still served (not rejected).
func TestForwardedHeaderAnomalyMiddlewarePassesThrough(t *testing.T) {
	stance, _ := clientip.NewStance(nil)
	e := echo.New()
	handler := forwardedHeaderAnomalyMiddleware(stance)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/rules", nil)
	req.RemoteAddr = "192.0.2.44:51234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("untrusted forwarded header must not reject: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
