package guardrails

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHostMatchesAllow(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		allowed []string
		want    bool
	}{
		{"exact match", "example.com", []string{"example.com"}, true},
		{"exact mismatch", "evil.com", []string{"example.com"}, false},
		{"subdomain suffix", "api.example.com", []string{"example.com"}, true},
		{"deep subdomain", "a.b.example.com", []string{"example.com"}, true},
		{"wildcard allows self", "example.com", []string{"*.example.com"}, true},
		{"wildcard allows subdomain", "api.example.com", []string{"*.example.com"}, true},
		{"wildcard allows deep", "sub.api.example.com", []string{"*.example.com"}, true},
		{"wildcard rejects other", "evil.com", []string{"*.example.com"}, false},
		{"wildcard rejects parent suffix", "notexample.com", []string{"*.example.com"}, false},
		{"port stripped", "example.com:443", []string{"example.com"}, true},
		{"port stripped wildcard", "api.example.com:8443", []string{"*.example.com"}, true},
		{"case insensitive", "Example.COM", []string{"example.com"}, true},
		{"case insensitive wildcard", "API.Example.com", []string{"*.example.com"}, true},
		{"empty host", "", []string{"example.com"}, false},
		{"empty allowed", "example.com", nil, false},
		{"bare star rejected", "anything.com", []string{"*"}, false},
		{"bare star-dot rejected", "anything.com", []string{"*."}, false},
		{"multiple allowed first", "a.com", []string{"a.com", "b.com"}, true},
		{"multiple allowed second", "b.com", []string{"a.com", "b.com"}, true},
		{"multiple allowed none", "c.com", []string{"a.com", "b.com"}, false},
		{"whitespace trimmed", "example.com", []string{"  example.com "}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hostMatchesAllow(tt.host, tt.allowed)
			if got != tt.want {
				t.Errorf("hostMatchesAllow(%q, %v) = %v, want %v", tt.host, tt.allowed, got, tt.want)
			}
		})
	}
}

func TestValidateAllowedHosts(t *testing.T) {
	t.Run("valid hosts", func(t *testing.T) {
		if err := ValidateAllowedHosts([]string{"example.com", "*.example.com"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("empty valid", func(t *testing.T) {
		if err := ValidateAllowedHosts(nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("bare star rejected", func(t *testing.T) {
		if err := ValidateAllowedHosts([]string{"*"}); err == nil {
			t.Error("expected error for bare *")
		}
	})
	t.Run("star-dot rejected", func(t *testing.T) {
		if err := ValidateAllowedHosts([]string{"*."}); err == nil {
			t.Error("expected error for *.")
		}
	})
	t.Run("url rejected", func(t *testing.T) {
		if err := ValidateAllowedHosts([]string{"https://example.com"}); err == nil {
			t.Error("expected error for URL entry")
		}
	})
}

func TestEgressProxyConnectFiltering(t *testing.T) {
	allowed := []string{"allowed.example.com", "*.wild.example.com"}
	proxy, addr, err := startEgressProxy(allowed, slog.Default())
	if err != nil {
		t.Fatalf("failed to start egress proxy: %v", err)
	}
	defer proxy.stopEgressProxy()

	t.Run("denied host gets 403", func(t *testing.T) {
		resp := doConnect(t, addr, "denied.example.com:443")
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for denied host, got %d", resp.StatusCode)
		}
	})

	t.Run("allowed host gets 200 (tunnel established)", func(t *testing.T) {
		// allowed.example.com is unlikely to resolve, but the proxy should
		// accept the CONNECT and attempt the upstream dial. We assert on the
		// proxy's response code distinction: allowed => 200, denied => 403.
		// To avoid depending on upstream reachability, we start a local TCP
		// listener and use its address as the allowed host.
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to start local listener: %v", err)
		}
		defer ln.Close()
		go func() {
			conn, cerr := ln.Accept()
			if cerr != nil {
				return
			}
			_ = conn.Close()
		}()

		// Restart proxy with the local listener's host (without port) in the
		// allowlist. The proxy strips the port from the CONNECT target before
		// matching, so the allowlist entry must be the bare host.
		localHost := ln.Addr().String()
		localHostOnly, _, _ := net.SplitHostPort(localHost)
		proxy2, addr2, perr := startEgressProxy([]string{localHostOnly}, slog.Default())
		if perr != nil {
			t.Fatalf("failed to start second proxy: %v", perr)
		}
		defer proxy2.stopEgressProxy()

		resp := doConnect(t, addr2, localHost)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for allowed host, got %d", resp.StatusCode)
		}
	})

	t.Run("non-CONNECT method rejected with 405", func(t *testing.T) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("failed to dial proxy: %v", err)
		}
		defer conn.Close()
		_, _ = fmt.Fprintf(conn, "GET http://allowed.example.com/ HTTP/1.1\r\nHost: allowed.example.com\r\n\r\n")
		resp, rerr := http.ReadResponse(bufio.NewReader(conn), nil)
		if rerr != nil {
			t.Fatalf("failed to read response: %v", rerr)
		}
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 for GET, got %d", resp.StatusCode)
		}
	})
}

// doConnect performs a raw HTTP CONNECT to proxyAddr for target and returns
// the parsed response. It writes the request and reads the status line. On
// parse failure it returns nil.
func doConnect(t *testing.T, proxyAddr, target string) *http.Response {
	t.Helper()
	conn, err := net.DialTimeout("tcp", proxyAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to dial proxy %s: %v", proxyAddr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	if _, werr := conn.Write([]byte(req)); werr != nil {
		t.Fatalf("failed to write CONNECT: %v", werr)
	}
	resp, rerr := http.ReadResponse(bufio.NewReader(conn), nil)
	if rerr != nil {
		t.Logf("failed to read response (may be expected if upstream dial fails): %v", rerr)
		return nil
	}
	return resp
}

func TestEgressProxyStopIdempotent(t *testing.T) {
	proxy, _, err := startEgressProxy([]string{"example.com"}, slog.Default())
	if err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	// Multiple stops must not panic.
	proxy.stopEgressProxy()
	proxy.stopEgressProxy()
}

func TestEgressProxyURL(t *testing.T) {
	got := egressProxyURL("127.0.0.1:12345")
	want := "http://127.0.0.1:12345"
	if got != want {
		t.Errorf("egressProxyURL = %q, want %q", got, want)
	}
}

func TestProvisionEgressNetworkBestEffort(t *testing.T) {
	// With no container runtime available, provisioning should fail
	// gracefully (non-fatal) and return an empty network name. We pass a
	// bogus runtime to force the failure path deterministically.
	nm, err := provisionEgressNetwork("no-such-runtime-xyz", slog.Default())
	if err == nil {
		t.Log("provisioning unexpectedly succeeded (runtime may be present)")
	}
	if nm != "" {
		t.Errorf("expected empty network name on failure, got %q", nm)
	}
}

func TestEgressProxyContextDrain(t *testing.T) {
	// Verify the proxy's stop uses a bounded context timeout and returns.
	proxy, addr, err := startEgressProxy([]string{"example.com"}, slog.Default())
	if err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	// Open a connection and then stop; stop should drain within timeout.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	_ = conn.Close()
	done := make(chan struct{})
	go func() {
		proxy.stopEgressProxy()
		close(done)
	}()
	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("stopEgressProxy did not return within timeout")
	}
}

// Sanity: ensure the package compiles with the new symbols referenced.
var _ = context.Background
var _ = io.EOF
var _ = strings.Contains
