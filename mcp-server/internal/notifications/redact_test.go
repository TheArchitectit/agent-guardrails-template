package notifications

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// The R17-08 delivery-log redaction suite. It pins the rule (RedactURLsInText)
// and drives the REAL delivery path through the real egress dial guard with a
// canary in the target's query string, asserting the canary never reaches the
// log or the stored error message.

type redactBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (p *redactBuf) Write(x []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.b.Write(x)
}
func (p *redactBuf) String() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.b.String()
}

// TestRedactURLsInText pins the rule: scheme/host/path survive, the whole query
// is withheld, and text that carries no URL is unchanged. A name-based
// allowlist would fail open on the first unlisted parameter, so the entire
// query is replaced.
func TestRedactURLsInText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no url unchanged", "webhook not found: wh1", "webhook not found: wh1"},
		{"empty is empty", "", ""},
		{"bare url no query unchanged", "https://example.com/hook", "https://example.com/hook"},
		{"query withheld", "https://example.com/hook?token=abc", "https://example.com/hook?<redacted>"},
		{
			"url inside wrapped error text is redacted",
			`request failed: Post "https://example.com/hook?api_key=abc": dial tcp: refused`,
			`request failed: Post "https://example.com/hook?<redacted>": dial tcp: refused`,
		},
		{
			"http scheme is covered too",
			"failed: http://10.0.0.1/hook?secret=xyz",
			"failed: http://10.0.0.1/hook?<redacted>",
		},
		{"two urls both redacted", "a https://x/y?t=1 b https://z/w?t=2", "a https://x/y?<redacted> b https://z/w?<redacted>"},
		{"non-http scheme untouched", "redis://user:pw@host:6379/0", "redis://user:pw@host:6379/0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RedactURLsInText(tc.in); got != tc.want {
				t.Fatalf("RedactURLsInText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

type redactStore struct {
	wh       *WebhookConfig
	last     *WebhookDelivery
	lastErr  string
	recorded bool
}

func (s *redactStore) Create(context.Context, *WebhookConfig) error { return nil }
func (s *redactStore) GetByID(context.Context, string) (*WebhookConfig, error) {
	return s.wh, nil
}
func (s *redactStore) ListByTeam(context.Context, string) ([]*WebhookConfig, error) {
	return nil, nil
}
func (s *redactStore) GetByEventType(context.Context, string) ([]*WebhookConfig, error) {
	return []*WebhookConfig{s.wh}, nil
}
func (s *redactStore) Update(context.Context, *WebhookConfig) error { return nil }
func (s *redactStore) Delete(context.Context, string) error         { return nil }
func (s *redactStore) RecordDelivery(_ context.Context, d *WebhookDelivery) error {
	s.last = d
	s.lastErr = d.ErrorMessage
	s.recorded = true
	return nil
}

// TestWebhookDeliveryFailureLogWithholdsTargetQuery drives the REAL deliver
// path: a target whose query carries a canary is refused by the egress dial
// guard (loopback), which makes http.Client.Do return a *url.Error embedding
// the full URL. Both the log line and the stored delivery error must withhold
// the canary while still naming the path for correlation.
func TestWebhookDeliveryFailureLogWithholdsTargetQuery(t *testing.T) {
	const canary = "SECRET_CANARY_WEBHOOK_9f2a71"

	buf := &redactBuf{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	target := srv.URL + "/hook?token=" + canary
	store := &redactStore{wh: &WebhookConfig{
		ID: "wh1", URL: target, Events: []string{"violation.detected"},
		SecretHMAC: "s", Enabled: true,
	}}
	d := newDispatcherWithResolver(store, defaultWebhookResolver)
	d.deliver(store.wh, "violation.detected", []byte(`{"a":1}`))

	out := buf.String()
	// Positive control: the failure log really ran.
	if !strings.Contains(out, "webhook delivery failed after retries") {
		t.Fatalf("delivery-failure log was not emitted; log was %q", out)
	}
	// The path is preserved for correlation.
	if !strings.Contains(out, "/hook") {
		t.Fatalf("the target path should be preserved in the log; log was %q", out)
	}
	if strings.Contains(out, canary) {
		t.Fatalf("target-query credential leaked into the delivery log: %s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Fatalf("expected the withheld-query marker in the log: %s", out)
	}
	// The stored/returned error message must not carry the secret either.
	if !store.recorded {
		t.Fatal("delivery failure was not recorded")
	}
	if strings.Contains(store.lastErr, canary) {
		t.Fatalf("target-query credential leaked into the stored error: %s", store.lastErr)
	}
}

// TestWebhookTestEventErrorWithholdsTargetQuery drives the test_webhook path:
// SendTestEvent returns and stores the transport error text, which is surfaced
// through the test_webhook and get_webhook_deliveries tool results.
func TestWebhookTestEventErrorWithholdsTargetQuery(t *testing.T) {
	const canary = "SECRET_CANARY_TESTEVENT_4c1e88"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	target := srv.URL + "/hook?access_token=" + canary
	store := &redactStore{wh: &WebhookConfig{
		ID: "wh1", URL: target, Events: []string{"violation.detected"},
		SecretHMAC: "s", Enabled: true,
	}}
	d := newDispatcherWithResolver(store, defaultWebhookResolver)

	delivery, err := d.SendTestEvent(context.Background(), "wh1")
	if err != nil {
		t.Fatalf("SendTestEvent returned an unexpected error: %v", err)
	}
	if delivery == nil {
		t.Fatal("SendTestEvent returned no delivery")
	}
	if strings.Contains(delivery.ErrorMessage, canary) {
		t.Fatalf("target-query credential leaked into the test-webhook error: %s", delivery.ErrorMessage)
	}
	if !strings.Contains(delivery.ErrorMessage, "/hook") {
		t.Fatalf("the target path should be preserved: %s", delivery.ErrorMessage)
	}
	if strings.Contains(store.lastErr, canary) {
		t.Fatalf("target-query credential leaked into the stored delivery: %s", store.lastErr)
	}
}
