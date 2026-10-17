package clientip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func req(t *testing.T, remoteAddr string, headers map[string]string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

// TestDefaultTrustsNoProxy covers Spec 12 Sec. 3.4.2: with TRUSTED_PROXIES unset
// the socket peer is the client IP and forwarded headers are ignored.
func TestDefaultTrustsNoProxy(t *testing.T) {
	s, err := NewStance(nil)
	if err != nil {
		t.Fatalf("NewStance(nil): %v", err)
	}
	if !s.Empty() {
		t.Fatal("empty stance must trust no proxy")
	}

	cases := []struct {
		name   string
		header string
		value  string
	}{
		{"x-forwarded-for ignored", "X-Forwarded-For", "1.2.3.4"},
		{"x-real-ip ignored", "X-Real-IP", "1.2.3.4"},
		{"forwarded ignored", "Forwarded", "for=1.2.3.4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := req(t, "10.0.0.9:5555", map[string]string{tc.header: tc.value})
			ip, anomaly := s.ClientIP(r)
			if ip != "10.0.0.9" {
				t.Fatalf("client ip = %q, want socket peer 10.0.0.9", ip)
			}
			if !anomaly {
				t.Fatal("forwarded header from untrusted peer must be reported as an anomaly")
			}
		})
	}
}

// TestTrustedPeerHonoursForwarded covers Spec 12 Sec. 3.4.1: forwarded headers are
// honoured only when the immediate peer is inside a configured CIDR.
func TestTrustedPeerHonoursForwarded(t *testing.T) {
	s, err := NewStance([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("NewStance: %v", err)
	}
	if s.Empty() {
		t.Fatal("stance must trust the configured CIDR")
	}

	t.Run("x-forwarded-for left-most wins", func(t *testing.T) {
		r := req(t, "10.0.0.5:5555", map[string]string{"X-Forwarded-For": "203.0.113.7, 10.0.0.1"})
		ip, anomaly := s.ClientIP(r)
		if ip != "203.0.113.7" {
			t.Fatalf("client ip = %q, want 203.0.113.7", ip)
		}
		if anomaly {
			t.Fatal("trusted peer must not raise an anomaly")
		}
	})

	t.Run("x-real-ip used when no xff", func(t *testing.T) {
		r := req(t, "10.0.0.5:5555", map[string]string{"X-Real-IP": "203.0.113.8"})
		if ip, _ := s.ClientIP(r); ip != "203.0.113.8" {
			t.Fatalf("client ip = %q, want 203.0.113.8", ip)
		}
	})

	t.Run("rfc7239 forwarded used last", func(t *testing.T) {
		r := req(t, "10.0.0.5:5555", map[string]string{"Forwarded": "for=203.0.113.9;proto=https"})
		if ip, _ := s.ClientIP(r); ip != "203.0.113.9" {
			t.Fatalf("client ip = %q, want 203.0.113.9", ip)
		}
	})

	t.Run("trusted peer with no forwarded header uses socket peer", func(t *testing.T) {
		r := req(t, "10.0.0.5:5555", nil)
		if ip, anomaly := s.ClientIP(r); ip != "10.0.0.5" || anomaly {
			t.Fatalf("client ip = %q anomaly=%v, want 10.0.0.5 false", ip, anomaly)
		}
	})

	t.Run("trusted peer with malformed forwarded falls back to socket peer", func(t *testing.T) {
		r := req(t, "10.0.0.5:5555", map[string]string{"X-Forwarded-For": "not-an-ip"})
		if ip, _ := s.ClientIP(r); ip != "10.0.0.5" {
			t.Fatalf("client ip = %q, want socket peer 10.0.0.5", ip)
		}
	})
}

// TestBareIPEntryTrusted covers bare-IP (non-CIDR) entries: a configured proxy
// address is trusted as a single host.
func TestBareIPEntryTrusted(t *testing.T) {
	s, err := NewStance([]string{"10.0.0.5"})
	if err != nil {
		t.Fatalf("NewStance: %v", err)
	}
	r := req(t, "10.0.0.5:5555", map[string]string{"X-Real-IP": "203.0.113.10"})
	if ip, _ := s.ClientIP(r); ip != "203.0.113.10" {
		t.Fatalf("client ip = %q, want 203.0.113.10", ip)
	}
	// A sibling address in the same /8 is NOT trusted when a bare host is set.
	r2 := req(t, "10.0.0.6:5555", map[string]string{"X-Real-IP": "203.0.113.10"})
	if ip, anomaly := s.ClientIP(r2); ip != "10.0.0.6" || !anomaly {
		t.Fatalf("client ip = %q anomaly=%v, want 10.0.0.6 true", ip, anomaly)
	}
}

// TestMalformedEntryFailsClosed covers the fail-closed parse: a bad CIDR is an
// error, not a silent narrowing to "trust nothing".
func TestMalformedEntryFailsClosed(t *testing.T) {
	for _, bad := range []string{"not-a-cidr", "999.1.1.1/24", "10.0.0.0/99"} {
		if _, err := NewStance([]string{bad}); err == nil {
			t.Fatalf("NewStance(%q) must fail closed", bad)
		}
	}
}

// TestIPv6AndZoneHandling covers bracketed IPv6, IPv6 with a zone, and
// IPv4-mapped peers.
func TestIPv6AndZoneHandling(t *testing.T) {
	s, err := NewStance([]string{"2001:db8::/32"})
	if err != nil {
		t.Fatalf("NewStance: %v", err)
	}
	r := req(t, "[2001:db8::1]:5555", map[string]string{"X-Forwarded-For": "[2001:db8:beef::2]"})
	if ip, _ := s.ClientIP(r); ip != "2001:db8:beef::2" {
		t.Fatalf("client ip = %q, want 2001:db8:beef::2", ip)
	}
	// Zone suffix on the peer must not defeat the CIDR match.
	r2 := req(t, "[2001:db8::1%eth0]:5555", nil)
	if ip, _ := s.ClientIP(r2); ip != "2001:db8::1" {
		t.Fatalf("client ip = %q, want 2001:db8::1", ip)
	}
}

// TestExtractorMatchesClientIP covers the Echo adapter: the extractor returns
// the same value the middleware would use.
func TestExtractorMatchesClientIP(t *testing.T) {
	s, _ := NewStance(nil)
	r := req(t, "192.0.2.5:1234", map[string]string{"X-Forwarded-For": "1.2.3.4"})
	if got := s.Extractor()(r); got != "192.0.2.5" {
		t.Fatalf("extractor = %q, want 192.0.2.5", got)
	}
}

// TestForwardedHeadersPresent covers the anomaly probe.
func TestForwardedHeadersPresent(t *testing.T) {
	if ForwardedHeadersPresent(req(t, "10.0.0.1:1", nil)) {
		t.Fatal("no forwarded header must read as absent")
	}
	if !ForwardedHeadersPresent(req(t, "10.0.0.1:1", map[string]string{"X-Forwarded-For": "1.2.3.4"})) {
		t.Fatal("x-forwarded-for must read as present")
	}
	if !ForwardedHeadersPresent(req(t, "10.0.0.1:1", map[string]string{"Forwarded": "for=1.2.3.4"})) {
		t.Fatal("forwarded must read as present")
	}
}
