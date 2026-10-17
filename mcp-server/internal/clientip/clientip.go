// Package clientip derives the effective client IP for a request under an
// explicit trusted-proxy stance (Spec 12 Sec. 3.4).
//
// Default posture: trust no proxy. When TRUSTED_PROXIES is empty the socket
// peer address is the client IP and forwarded headers (X-Forwarded-For,
// X-Real-IP, Forwarded) are ignored for rate-limit keys, audit and logging.
//
// Forwarded headers are honoured only when the immediate peer is inside a
// configured CIDR. When the peer is not trusted, a request that carries
// forwarded headers is reported so callers can record the protocol anomaly.
//
// IP-derived values are never an authorization input; this package feeds
// rate-limit bucketing and log attribution only.
package clientip

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// Forwarded headers this package inspects when the peer is trusted.
var forwardedHeaders = []string{
	"X-Forwarded-For",
	"X-Real-IP",
	"Forwarded",
}

// Stance is a parsed, immutable trusted-proxy policy. The zero value trusts no
// proxy, which is the safe default.
type Stance struct {
	nets []*net.IPNet
}

// NewStance parses a list of CIDRs (or bare IPs) into a trusted-proxy stance.
// A bare address is treated as a single-host CIDR. An empty or nil list yields
// a stance that trusts no peer. A malformed entry is a hard error: an operator
// who configures a bad CIDR must be told, not silently narrowed to "trust
// nothing" (a silent narrowing can flip the effective exposure posture).
func NewStance(entries []string) (Stance, error) {
	var s Stance
	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if _, ipnet, err := net.ParseCIDR(entry); err == nil {
			s.nets = append(s.nets, ipnet)
			continue
		}
		// Accept a bare IP as a single-host CIDR.
		if ip := net.ParseIP(entry); ip != nil {
			s.nets = append(s.nets, singleHost(ip))
			continue
		}
		return Stance{}, fmt.Errorf("TRUSTED_PROXIES entry %q is not a valid IP or CIDR", raw)
	}
	return s, nil
}

// Trusts reports whether the peer address is an allowed proxy.
func (s Stance) Trusts(peer string) bool {
	ip := parseHost(peer)
	if ip == nil {
		return false
	}
	for _, n := range s.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// Empty reports whether no proxy is trusted.
func (s Stance) Empty() bool { return len(s.nets) == 0 }

// ForwardedHeadersPresent reports whether the request carries any forwarded
// header, regardless of trust. Callers use it to record a protocol anomaly for
// untrusted peers.
func ForwardedHeadersPresent(r *http.Request) bool {
	for _, h := range forwardedHeaders {
		if strings.TrimSpace(r.Header.Get(h)) != "" {
			return true
		}
	}
	return false
}

// ClientIP returns the effective client IP for the request under the stance.
// The socket peer is used unless the peer is trusted and supplies a valid
// forwarded address. The second result reports whether a forwarded header was
// present from an untrusted peer (a protocol anomaly, not a rejection).
func (s Stance) ClientIP(r *http.Request) (ip string, untrustedForwarded bool) {
	peer := socketPeer(r.RemoteAddr)
	// Canonicalise the peer (drop a bracketed form or an IPv6 zone) so the
	// returned value is comparable and log-stable.
	peerIP := parseHost(peer)
	if s.Trusts(peer) {
		if fwd := forwardedIP(r); fwd != "" {
			return fwd, false
		}
		if peerIP != nil {
			return peerIP.String(), false
		}
		return peer, false
	}
	if ForwardedHeadersPresent(r) {
		return canonicalPeer(peerIP, peer), true
	}
	return canonicalPeer(peerIP, peer), false
}

func canonicalPeer(peerIP net.IP, raw string) string {
	if peerIP != nil {
		return peerIP.String()
	}
	return raw
}

// Extractor adapts a Stance to Echo's IPExtractor type without importing Echo.
// It returns the effective client IP for the request.
func (s Stance) Extractor() func(*http.Request) string {
	return func(r *http.Request) string {
		ip, _ := s.ClientIP(r)
		return ip
	}
}

// socketPeer extracts the host portion of a Go RemoteAddr ("host:port").
func socketPeer(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// forwardedIP returns the first valid client address from the forwarded
// headers, preferring X-Forwarded-For (left-most entry), then X-Real-IP, then
// the `for=` parameter of RFC 7239 Forwarded. Returns "" when none is valid.
func forwardedIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Left-most entry is the originating client.
		for _, part := range strings.Split(xff, ",") {
			if ip := parseHost(strings.TrimSpace(part)); ip != nil {
				return ip.String()
			}
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		if ip := parseHost(xri); ip != nil {
			return ip.String()
		}
	}
	if fwd := r.Header.Get("Forwarded"); fwd != "" {
		for _, element := range strings.Split(fwd, ",") {
			for _, pair := range strings.Split(element, ";") {
				kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
				if len(kv) != 2 || !strings.EqualFold(kv[0], "for") {
					continue
				}
				if ip := parseHost(strings.Trim(strings.TrimSpace(kv[1]), `"`)); ip != nil {
					return ip.String()
				}
			}
		}
	}
	return ""
}

// parseHost parses an IP, tolerating a bracketed IPv6 literal, an optional
// port suffix, and an IPv6 zone. Returns nil when the value is not an IP.
func parseHost(value string) net.IP {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	v = strings.TrimPrefix(v, "[")
	if i := strings.IndexByte(v, ']'); i >= 0 {
		v = v[:i]
	}
	// Strip an IPv6 zone ("fe80::1%eth0") before parsing.
	if i := strings.IndexByte(v, '%'); i >= 0 {
		v = v[:i]
	}
	if ip := net.ParseIP(v); ip != nil {
		return ip
	}
	// Try host:port form (IPv4 with port).
	if host, _, err := net.SplitHostPort(value); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip
		}
	}
	return nil
}

func singleHost(ip net.IP) *net.IPNet {
	if v4 := ip.To4(); v4 != nil {
		return &net.IPNet{IP: v4, Mask: net.CIDRMask(32, 32)}
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
}
