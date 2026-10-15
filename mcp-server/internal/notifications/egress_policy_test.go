package notifications

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
)

// staticResolver returns a fixed answer set, or an error, for every host.
func staticResolver(addrs ...string) webhookResolver {
	return func(context.Context, string) ([]netip.Addr, error) {
		out := make([]netip.Addr, 0, len(addrs))
		for _, a := range addrs {
			out = append(out, netip.MustParseAddr(a))
		}
		return out, nil
	}
}

func errResolver(err error) webhookResolver {
	return func(context.Context, string) ([]netip.Addr, error) { return nil, err }
}

// --- Spec 13 acceptance 5/6: the address matrix -----------------------------

func TestClassifyWebhookAddrDeniesSpecialPurposeAndAllowsPublic(t *testing.T) {
	denied := []struct{ name, addr string }{
		{"ipv4 loopback", "127.0.0.1"},
		{"ipv4 this-network", "0.1.2.3"},
		{"ipv4 private 10/8", "10.1.2.3"},
		{"ipv4 private 172.16/12", "172.20.1.1"},
		{"ipv4 private 192.168/16", "192.168.0.1"},
		{"ipv4 link-local", "169.254.10.10"},
		{"imds v4", "169.254.169.254"},
		{"ipv4 cgnat", "100.64.0.1"},
		{"ipv4 benchmarking", "198.18.0.1"},
		{"ipv4 documentation", "203.0.113.9"},
		{"ipv4 multicast", "224.0.0.1"},
		{"ipv4 reserved", "240.0.0.1"},
		{"ipv4 broadcast", "255.255.255.255"},
		{"ipv6 loopback", "::1"},
		{"ipv6 unspecified", "::"},
		{"ipv6 unique-local", "fd00::1"},
		{"ipv6 link-local", "fe80::1"},
		{"ipv6 multicast", "ff02::1"},
		{"ipv6 documentation", "2001:db8::1"},
		{"ipv6 6to4", "2002:c0a8:101::1"},
		{"ipv6 teredo", "2001::1"},
		{"ipv6 nat64", "64:ff9b::a00:1"},
		{"imds v6", "fd00:ec2::254"},
		{"mapped ipv4 loopback", "::ffff:127.0.0.1"},
		{"mapped ipv4 private", "::ffff:10.0.0.1"},
	}
	for _, c := range denied {
		if err := ClassifyWebhookAddr(netip.MustParseAddr(c.addr)); err == nil {
			t.Errorf("%s (%s): must be denied, was allowed", c.name, c.addr)
		} else if !errors.Is(err, ErrWebhookEgressDenied) {
			t.Errorf("%s (%s): denial must be terminal (ErrWebhookEgressDenied), got %v", c.name, c.addr, err)
		}
	}

	allowed := []string{"93.184.216.34", "1.1.1.1", "8.8.8.8", "2606:4700:4700::1111", "2001:4860:4860::8888"}
	for _, a := range allowed {
		if err := ClassifyWebhookAddr(netip.MustParseAddr(a)); err != nil {
			t.Errorf("public address %s must be allowed, got %v", a, err)
		}
	}

	if err := ClassifyWebhookAddr(netip.Addr{}); err == nil {
		t.Error("an invalid address must be denied, not dialled")
	}
}

func TestClassifyWebhookAddrDeniesScopedAddress(t *testing.T) {
	// fe80::1%eth0 — a zone identifier is not a delivery destination.
	if err := ClassifyWebhookAddr(netip.MustParseAddr("fe80::1%eth0")); err == nil {
		t.Fatal("a scoped address must be denied")
	}
}

// --- Spec 13 acceptance 4: mixed answer set ---------------------------------

func TestMixedAnswerSetIsDeniedWhole(t *testing.T) {
	for _, order := range [][]string{
		{"93.184.216.34", "10.0.0.1"},
		{"10.0.0.1", "93.184.216.34"},
	} {
		_, err := safeDialContextWithResolver(staticResolver(order...))(
			context.Background(), "tcp", "webhook.example:443")
		if err == nil {
			t.Fatalf("answer set %v: a mixed public/private set must be denied whole", order)
		}
		if !errors.Is(err, ErrWebhookEgressDenied) {
			t.Fatalf("answer set %v: denial must be terminal, got %v", order, err)
		}
	}
}

// --- Spec 13 acceptance 2/3: stored-URL rebinding at the dial ---------------

func TestDialGuardRefusesRebindingAnswer(t *testing.T) {
	// The listener is real and public-looking at configuration time; the
	// resolver now answers with a private address. Nothing may reach it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("a rebinding answer must not reach the target")
	}))
	defer srv.Close()

	dial := safeDialContextWithResolver(staticResolver("10.0.0.1"))
	if _, err := dial(context.Background(), "tcp", "rebind.example:80"); err == nil {
		t.Fatal("a rebound private answer must be refused at dial time")
	}
}

// --- Spec 13 acceptance 1: literal prohibited IPs are refused before dialing -

func TestDialGuardRefusesLiteralProhibitedIPWithoutDialing(t *testing.T) {
	dialed := false
	dial := safeDialContextWithResolver(func(context.Context, string) ([]netip.Addr, error) {
		t.Error("a literal address must not be resolved")
		return nil, nil
	})
	// Wrap: if the policy lets it through, the real dial would attempt loopback.
	conn, err := dial(context.Background(), "tcp", "127.0.0.1:9")
	if err == nil {
		dialed = true
		_ = conn.Close()
	}
	if dialed {
		t.Fatal("a prohibited literal IP must be refused before dialing")
	}
}

// --- Spec 13 acceptance 13: resolution failure fails closed -----------------

func TestResolutionFailureFailsClosed(t *testing.T) {
	dial := safeDialContextWithResolver(errResolver(errors.New("resolver unavailable")))
	if _, err := dial(context.Background(), "tcp", "unknown.example:443"); err == nil {
		t.Fatal("a resolution failure must deny the delivery")
	}
}

// --- Spec 13 acceptance 8/9: both delivery paths share the policy -----------

type fakeStore struct {
	mu         sync.Mutex
	webhook    *WebhookConfig
	deliveries []*WebhookDelivery
}

func (f *fakeStore) Create(context.Context, *WebhookConfig) error { return nil }
func (f *fakeStore) GetByID(_ context.Context, id string) (*WebhookConfig, error) {
	if f.webhook == nil || f.webhook.ID != id {
		return nil, fmt.Errorf("not found")
	}
	return f.webhook, nil
}
func (f *fakeStore) ListByTeam(context.Context, string) ([]*WebhookConfig, error) { return nil, nil }
func (f *fakeStore) GetByEventType(_ context.Context, eventType string) ([]*WebhookConfig, error) {
	if f.webhook != nil && f.webhook.Enabled {
		return []*WebhookConfig{f.webhook}, nil
	}
	return nil, nil
}
func (f *fakeStore) Update(context.Context, *WebhookConfig) error { return nil }
func (f *fakeStore) Delete(context.Context, string) error         { return nil }
func (f *fakeStore) RecordDelivery(_ context.Context, d *WebhookDelivery) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deliveries = append(f.deliveries, d)
	return nil
}

// A blocked destination must produce a failure state, never success, and the
// target must never be contacted — on BOTH the event path and the test path.
func TestBothDeliveryPathsDenyBlockedDestination(t *testing.T) {
	var contacted bool
	var mu sync.Mutex
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		contacted = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	host, port, _ := net.SplitHostPort(strings.TrimPrefix(backend.URL, "http://"))
	_ = host

	// Resolver answers with the backend's loopback address (a private target).
	resolve := staticResolver("127.0.0.1")

	store := &fakeStore{webhook: &WebhookConfig{
		ID: "wh-1", TeamID: "t1", URL: backend.URL, Events: []string{"violation.detected"},
		SecretHMAC: "secret", Enabled: true,
	}}
	d := newDispatcherWithResolver(store, resolve)

	// Test-event path.
	del, err := d.SendTestEvent(context.Background(), "wh-1")
	if err != nil {
		t.Fatalf("SendTestEvent returned a transport error instead of a recorded denial: %v", err)
	}
	if del.Success {
		t.Fatal("test path: a blocked destination must not report success")
	}
	if del.ErrorMessage == "" {
		t.Fatal("test path: a denial must record a sanitized reason")
	}

	// Event path (synchronous form of the async deliver()).
	_, derr := d.doRequest(context.Background(), backend.URL, []byte("{}"), "sig")
	if derr == nil {
		t.Fatal("event path: a blocked destination must be denied")
	}
	if !errors.Is(derr, ErrWebhookEgressDenied) {
		t.Fatalf("event path: denial must be terminal, got %v", derr)
	}

	mu.Lock()
	defer mu.Unlock()
	if contacted {
		t.Fatal("a denied destination must never be contacted on either path")
	}
	_ = port
}

// --- Spec 13 acceptance 12: no secrets in denial evidence -------------------

func TestDenialReasonLeaksNoSecretOrUserinfo(t *testing.T) {
	err := ClassifyWebhookAddr(netip.MustParseAddr("10.0.0.1"))
	if err == nil {
		t.Fatal("expected a denial")
	}
	msg := err.Error()
	for _, leak := range []string{"secret", "token", "password", "@", "hmac"} {
		if strings.Contains(strings.ToLower(msg), leak) {
			t.Errorf("denial reason %q leaks %q", msg, leak)
		}
	}
}
