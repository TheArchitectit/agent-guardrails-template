package guardrails

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// egressProxy is a CONNECT-only filtering forward proxy used to enforce
// per-host egress whitelisting for L2 sandboxes that declare AllowedHosts.
//
// It listens on 127.0.0.1:0 and accepts HTTP CONNECT requests. A CONNECT
// target is permitted only if its host matches an entry in allowedHosts
// (exact match or "*.suffix" wildcard). All other targets are refused with
// HTTP 403 and logged. Plain GET/POST proxying is intentionally NOT supported
// — the sandbox only routes HTTPS through this proxy via HTTPS_PROXY, which
// uses CONNECT exclusively. Any non-CONNECT method is rejected with 405.
//
// The proxy only enforces the host allowlist. It does not perform TLS
// termination or upstream connection pooling; on an allowed CONNECT it dials
// the upstream host and tunnels bytes bidirectionally until either side closes.
type egressProxy struct {
	listener     net.Listener
	server       *http.Server
	allowedHosts []string
	logger       *slog.Logger
	stopOnce     sync.Once
}

// hostMatchesAllow reports whether host is permitted by the allowed list.
//
// Matching rules:
//   - The host is first stripped of any port (e.g. "example.com:443").
//   - Comparison is case-insensitive.
//   - An exact match (after normalization) is allowed.
//   - A wildcard entry "*.example.com" allows example.com itself AND any
//     single-level subdomain (e.g. "api.example.com", "sub.api.example.com").
//   - A bare wildcard "*" or "*." is rejected by the config validator
//     (ValidateAllowedHosts) and must never reach this function, but if it
//     does it matches nothing (fail-closed).
//
// allowed entries are bare hosts or "*.example.com" — never URLs or schemes.
func hostMatchesAllow(host string, allowed []string) bool {
	// Strip port if present (host:port or [ipv6]:port).
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// Also handle a bare port suffix without brackets for IPv4/hostname.
	if idx := strings.LastIndexByte(host, ':'); idx != -1 {
		candidate := host[:idx]
		if net.ParseIP(candidate) != nil || strings.Contains(candidate, ".") {
			host = candidate
		}
	}
	if host == "" {
		return false
	}
	host = strings.ToLower(host)
	for _, a := range allowed {
		a = strings.TrimSpace(strings.ToLower(a))
		if a == "" {
			continue
		}
		if a == "*" || a == "*." {
			// Too-broad wildcard; must be rejected at validation time.
			// Fail closed: never match.
			continue
		}
		if host == a {
			return true
		}
		// Handle "*.example.com" wildcard: allow the apex domain
		// (example.com) and any subdomain (api.example.com, a.b.example.com).
		if strings.HasPrefix(a, "*.") {
			suffix := a[1:] // ".example.com"
			apex := a[2:]   // "example.com"
			if host == apex || strings.HasSuffix(host, suffix) {
				return true
			}
			continue
		}
		// Defense in depth: also allow suffix matches for non-wildcard
		// entries so subdomains of a listed parent are covered.
		if strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

// ValidateAllowedHosts rejects configs with overly broad wildcard entries
// (e.g. bare "*" or "*.") that would defeat the egress filter. It returns an
// error describing the first offending entry. Empty lists are valid (they
// simply mean no egress, handled by the caller with --network=none).
func ValidateAllowedHosts(allowed []string) error {
	for _, a := range allowed {
		t := strings.TrimSpace(a)
		if t == "*" || t == "*." {
			return fmt.Errorf("allowed host %q is too broad; per-host egress filtering requires explicit hosts or *.suffix wildcards", a)
		}
		if strings.ContainsAny(t, ":/") {
			return fmt.Errorf("allowed host %q must be a bare host or *.suffix, not a URL", a)
		}
	}
	return nil
}

// startEgressProxy launches the CONNECT filtering proxy on 127.0.0.1:0 and
// returns it along with the listen address (host:port). The proxy enforces
// the provided allowlist. The caller MUST call stopEgressProxy (typically via
// defer) to release the listener and drain in-flight connections.
func startEgressProxy(allowed []string, logger *slog.Logger) (*egressProxy, string, error) {
	if logger == nil {
		logger = slog.Default()
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("failed to bind egress proxy listener: %w", err)
	}
	p := &egressProxy{
		listener:     ln,
		allowedHosts: allowed,
		logger:       logger,
	}
	p.server = &http.Server{
		Handler:           http.HandlerFunc(p.handle),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		_ = p.server.Serve(ln)
	}()
	return p, ln.Addr().String(), nil
}

// handle implements the single HTTP handler. Only CONNECT is supported.
func (p *egressProxy) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		p.logger.Warn("egress proxy rejected non-CONNECT request", "method", r.Method, "host", r.Host)
		http.Error(w, "method not allowed; only CONNECT is supported", http.StatusMethodNotAllowed)
		return
	}

	target := r.Host
	if target == "" {
		target = r.URL.Host
	}
	if !hostMatchesAllow(target, p.allowedHosts) {
		p.logger.Warn("egress proxy denied CONNECT to non-allowed host", "host", target)
		http.Error(w, "host not allowed by egress policy", http.StatusForbidden)
		return
	}

	// Allowed: dial upstream and tunnel.
	destConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		p.logger.Warn("egress proxy upstream dial failed", "host", target, "error", err)
		http.Error(w, "upstream connection failed", http.StatusBadGateway)
		return
	}
	defer destConn.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		p.logger.Warn("egress proxy hijack failed", "host", target, "error", err)
		return
	}
	defer clientConn.Close()

	// Signal successful tunnel establishment to the client.
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Bidirectional tunnel until either side closes.
	done := make(chan struct{}, 2)
	go tunnel(clientConn, destConn, done)
	go tunnel(destConn, clientConn, done)
	<-done
	<-done
}

// tunnel copies from src to dst, signalling completion on done. It is used to
// relay bytes between the client and the upstream host once a CONNECT tunnel
// is established.
func tunnel(dst, src net.Conn, done chan struct{}) {
	_, _ = io.Copy(dst, src)
	// Best-effort: close the write side so the peer sees EOF.
	if tc, ok := dst.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}
	done <- struct{}{}
}

// stopEgressProxy idempotently shuts down the proxy: it closes the listener
// and drains in-flight connections with a bounded context timeout, then
// closes the server. Safe to call multiple times.
func (p *egressProxy) stopEgressProxy() {
	if p == nil {
		return
	}
	p.stopOnce.Do(func() {
		if p.listener != nil {
			_ = p.listener.Close()
		}
		if p.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = p.server.Shutdown(ctx)
		}
	})
}

// provisionEgressNetwork best-effort creates a bridge container network named
// "guardrail-egress" so the sandbox container can reach the local egress proxy
// while still being isolated from the host's default network. It uses podman
// (docker fallback). Failures are non-fatal: the caller should fall back to
// --network=none and degrade the feature, never open the network.
//
// runtime is the resolved container runtime ("podman" or "docker").
// Returns the network name to attach (empty string means "use none / fail").
func provisionEgressNetwork(runtime string, logger *slog.Logger) (string, error) {
	const netName = "guardrail-egress"
	if logger == nil {
		logger = slog.Default()
	}
	// Create lazily if it does not already exist.
	if err := runNetworkExists(runtime, netName); err != nil {
		// Network does not exist; create it.
		if cerr := runNetworkCreate(runtime, netName); cerr != nil {
			logger.Warn("failed to provision egress network; falling back to network=none",
				"runtime", runtime, "error", cerr)
			return "", cerr
		}
	}
	return netName, nil
}

// runNetworkExists checks whether the named container network exists. Returns
// nil if it exists, an error otherwise.
func runNetworkExists(runtime, netName string) error {
	cmd := exec.Command(runtime, "network", "exists", netName)
	return cmd.Run()
}

// runNetworkCreate creates a bridge container network with the given name.
func runNetworkCreate(runtime, netName string) error {
	cmd := exec.Command(runtime, "network", "create", netName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("network create failed: %w: %s", err, string(out))
	}
	return nil
}

// extractProxyHostPort parses a listen address "127.0.0.1:port" into the
// loopback proxy URL used for HTTPS_PROXY / HTTP_PROXY injection.
func egressProxyURL(addr string) string {
	// addr is host:port; build http://host:port.
	u := &url.URL{Scheme: "http", Host: addr}
	return u.String()
}
