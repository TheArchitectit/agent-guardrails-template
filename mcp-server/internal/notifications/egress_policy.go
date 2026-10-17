package notifications

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"
)

// ErrWebhookEgressDenied marks a delivery refused by the egress policy, as
// opposed to a transient transport failure. A policy denial is terminal: the
// delivery retry loop must not re-attempt it (Spec 13 acceptance 11).
var ErrWebhookEgressDenied = errors.New("webhook egress policy denied")

type egressDeniedError struct{ reason string }

func (e *egressDeniedError) Error() string { return e.reason }
func (e *egressDeniedError) Unwrap() error { return ErrWebhookEgressDenied }

func deny(format string, args ...any) error {
	return &egressDeniedError{reason: fmt.Sprintf(format, args...)}
}

// Webhook delivery egress policy (Spec 13 Sec. 3.2).
//
// The policy is DATA, not predicates: every denied range is a reviewed entry in
// the table below, so a new IANA special-purpose allocation is a one-line
// change with a fixture, not a new conditional. Classification runs on the
// address the dial actually uses, after resolution, and the WHOLE resolved
// answer set is validated — a host that answers with a public and a private
// address is refused outright rather than being allowed to pick the public one.
// A mixed answer is the rebinding attack, not a preference.
//
// The default posture is the spec's first-implementation default (Sec. 7): no
// internal-destination exception, no redirects. HTTPS-only / port-443
// enforcement (Sec. 3.1) is a separate deployment-behaviour decision and is not
// applied here.

// webhookResolver resolves a webhook destination host to its full address set.
// It is a parameter so tests can drive rebinding and mixed-answer cases without
// a network (Spec 13 acceptance 13).
type webhookResolver func(ctx context.Context, host string) ([]netip.Addr, error)

// defaultWebhookResolver uses the process resolver and asks for the complete
// answer set ("ip" family — both A and AAAA), never a single preferred family.
func defaultWebhookResolver(ctx context.Context, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

// deniedWebhookPrefixes is the reviewed snapshot of IANA special-purpose
// address blocks that must never receive a webhook delivery. The label is what
// a refusal reports, so an operator sees the class, not just "denied".
var deniedWebhookPrefixes = []struct {
	prefix netip.Prefix
	label  string
}{
	{netip.MustParsePrefix("0.0.0.0/8"), "ipv4 this-network"},
	{netip.MustParsePrefix("10.0.0.0/8"), "ipv4 private"},
	{netip.MustParsePrefix("100.64.0.0/10"), "ipv4 carrier-grade nat"},
	{netip.MustParsePrefix("127.0.0.0/8"), "ipv4 loopback"},
	{netip.MustParsePrefix("169.254.0.0/16"), "ipv4 link-local"},
	{netip.MustParsePrefix("172.16.0.0/12"), "ipv4 private"},
	{netip.MustParsePrefix("192.0.0.0/24"), "ipv4 ietf protocol assignments"},
	{netip.MustParsePrefix("192.0.2.0/24"), "ipv4 documentation"},
	{netip.MustParsePrefix("192.88.99.0/24"), "ipv4 6to4 relay anycast"},
	{netip.MustParsePrefix("192.168.0.0/16"), "ipv4 private"},
	{netip.MustParsePrefix("198.18.0.0/15"), "ipv4 benchmarking"},
	{netip.MustParsePrefix("198.51.100.0/24"), "ipv4 documentation"},
	{netip.MustParsePrefix("203.0.113.0/24"), "ipv4 documentation"},
	{netip.MustParsePrefix("224.0.0.0/4"), "ipv4 multicast"},
	{netip.MustParsePrefix("240.0.0.0/4"), "ipv4 reserved"},
	{netip.MustParsePrefix("::/128"), "ipv6 unspecified"},
	{netip.MustParsePrefix("::1/128"), "ipv6 loopback"},
	{netip.MustParsePrefix("64:ff9b::/96"), "ipv6 nat64"},
	{netip.MustParsePrefix("64:ff9b:1::/48"), "ipv6 local-use nat64"},
	{netip.MustParsePrefix("100::/64"), "ipv6 discard-only"},
	{netip.MustParsePrefix("2001::/32"), "ipv6 teredo"},
	{netip.MustParsePrefix("2001:2::/48"), "ipv6 benchmarking"},
	{netip.MustParsePrefix("2001:db8::/32"), "ipv6 documentation"},
	{netip.MustParsePrefix("2002::/16"), "ipv6 6to4"},
	{netip.MustParsePrefix("fc00::/7"), "ipv6 unique-local"},
	{netip.MustParsePrefix("fe80::/10"), "ipv6 link-local"},
	{netip.MustParsePrefix("ff00::/8"), "ipv6 multicast"},
}

// metadataWebhookAddrs are cloud/platform metadata and control-plane addresses.
// Address classification already denies them (link-local / unique-local); they
// are named separately so a refusal says "metadata" rather than a generic
// range, and so the intent survives a future range-policy edit.
var metadataWebhookAddrs = []netip.Addr{
	netip.MustParseAddr("169.254.169.254"), // AWS/GCP/Azure IMDS (v4)
	netip.MustParseAddr("fd00:ec2::254"),   // AWS IMDS (v6)
}

// ClassifyWebhookAddr reports whether addr is a permitted webhook delivery
// destination. It fails closed: an invalid, zoned, non-global-unicast, metadata
// or special-purpose address is refused with a reason naming the class.
//
// IPv4-mapped IPv6 addresses are classified by their embedded IPv4 value, so
// ::ffff:127.0.0.1 is refused exactly like 127.0.0.1.
func ClassifyWebhookAddr(addr netip.Addr) error {
	if !addr.IsValid() {
		return deny("refusing webhook delivery: no address to dial")
	}
	if addr.Zone() != "" {
		return deny("refusing webhook delivery: scoped address %q", addr.String())
	}
	if addr.Is4In6() {
		addr = addr.Unmap()
	}
	for _, m := range metadataWebhookAddrs {
		if addr == m {
			return deny("refusing webhook delivery to metadata address %s", addr)
		}
	}
	for _, d := range deniedWebhookPrefixes {
		if d.prefix.Contains(addr) {
			return deny("refusing webhook delivery to %s (%s)", addr, d.label)
		}
	}
	if !addr.IsGlobalUnicast() {
		return deny("refusing webhook delivery to non-global address %s", addr)
	}
	return nil
}

// publicWebhookIP is the net.IP-shaped entry point to the same policy, kept for
// callers that already hold a net.IP (and for the existing rebinding test).
func publicWebhookIP(ip net.IP) error {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return deny("refusing webhook delivery: no address to dial")
	}
	return ClassifyWebhookAddr(addr)
}

// safeDialContextWithResolver returns a DialContext that resolves the
// destination once, refuses the whole answer set if any member is disallowed,
// and then dials only from that validated set. Because the dial target is an
// already-validated literal, there is no second, unvalidated lookup between the
// policy check and the socket — that gap is the rebinding seam.
func safeDialContextWithResolver(resolve webhookResolver) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, deny("refusing webhook delivery: unparseable dial address %q", address)
		}

		var addrs []netip.Addr
		if literal, perr := netip.ParseAddr(host); perr == nil {
			addrs = []netip.Addr{literal}
		} else {
			addrs, err = resolve(ctx, host)
			if err != nil {
				return nil, deny("refusing webhook delivery: cannot resolve %q: %v", host, err)
			}
		}
		if len(addrs) == 0 {
			return nil, deny("refusing webhook delivery: no address for %q", host)
		}

		for _, a := range addrs {
			if cerr := ClassifyWebhookAddr(a); cerr != nil {
				return nil, cerr
			}
		}

		var lastErr error
		for _, a := range addrs {
			d := &net.Dialer{Timeout: 10 * time.Second}
			conn, derr := d.DialContext(ctx, network, net.JoinHostPort(a.String(), port))
			if derr == nil {
				return conn, nil
			}
			lastErr = derr
		}
		return nil, fmt.Errorf("webhook delivery dial failed: %w", lastErr)
	}
}

// newWebhookClientWithResolver builds the delivery client both paths share.
//
// The dial context validates and pins the destination; CheckRedirect refuses to
// follow, because a redirect target is never validated at configuration time.
// The transport takes no proxy function, so a proxy environment variable cannot
// route a delivery around the dial guard.
func newWebhookClientWithResolver(resolve webhookResolver) *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: safeDialContextWithResolver(resolve),
			Proxy:       nil,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return fmt.Errorf("refusing webhook redirect: a delivery target must not move after validation")
		},
	}
}

// newWebhookClient is the production client (process resolver).
func newWebhookClient() *http.Client {
	return newWebhookClientWithResolver(defaultWebhookResolver)
}
