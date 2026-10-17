package notifications

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// The Spec 17 R17-10 webhook-rotation suite. It pins the receiver-facing
// contract: the active signing key is identified on the wire without leaking
// it, the outgoing key is offered only inside the bounded overlap, a cutover
// invalidates it, and the decision is one rule shared by every delivery path.
//
// The wire assertions drive the REAL request builder (doRequest) against a real
// httptest receiver through a plain client, so the headers asserted are the
// bytes a receiver actually sees. (The production client additionally enforces
// the Spec 13 egress policy, which refuses loopback targets — that policy is
// covered by its own suite and is orthogonal to signing.)

// rotReceiver captures the signing headers of the last request.
type rotReceiver struct {
	mu        sync.Mutex
	keyID     string
	sig       string
	prevKeyID string
	prevSig   string
	hits      int
}

func (r *rotReceiver) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.keyID = req.Header.Get(WebhookKeyIDHeader)
		r.sig = req.Header.Get("X-Guardrail-Signature")
		r.prevKeyID = req.Header.Get(WebhookPreviousKeyIDHeader)
		r.prevSig = req.Header.Get(WebhookPreviousSignatureHeader)
		r.hits++
		w.WriteHeader(http.StatusOK)
	}
}

func (r *rotReceiver) snapshot() (keyID, sig, prevKeyID, prevSig string, hits int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.keyID, r.sig, r.prevKeyID, r.prevSig, r.hits
}

// wireDispatcher builds a dispatcher whose client is a plain transport, so a
// loopback receiver can be driven to inspect the signing headers.
func wireDispatcher() *Dispatcher {
	return &Dispatcher{client: &http.Client{Timeout: 5 * time.Second}}
}

// TestWebhookSigningKeyIdentityWithoutOverlap proves a steady-state delivery
// identifies its active key by fingerprint and signs with that key, and carries
// no previous-key headers. The receiver can select the matching verification
// key without either side transmitting the secret.
func TestWebhookSigningKeyIdentityWithoutOverlap(t *testing.T) {
	const active = "active-signing-key-0001"
	payload := []byte(`{"a":1}`)

	rec := &rotReceiver{}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	wh := &WebhookConfig{ID: "wh1", SecretHMAC: active, Enabled: true}
	d := wireDispatcher()
	h := buildWebhookSigningHeaders(WebhookSigningKeys(wh, time.Now().UTC()), payload)
	if _, err := d.doRequest(context.Background(), srv.URL+"/hook", payload, h); err != nil {
		t.Fatalf("doRequest returned an unexpected error: %v", err)
	}

	keyID, sig, prevKeyID, prevSig, hits := rec.snapshot()
	if hits != 1 {
		t.Fatalf("expected exactly one delivery, got %d", hits)
	}
	if keyID != WebhookKeyFingerprint(active) {
		t.Fatalf("active key id = %q, want the fingerprint of the active key", keyID)
	}
	if !VerifySignature(payload, active, sig) {
		t.Fatal("delivery was not signed with the active key")
	}
	if strings.Contains(keyID, active) || keyID == active {
		t.Fatalf("key id must be a fingerprint, never the secret: %q", keyID)
	}
	if prevKeyID != "" || prevSig != "" {
		t.Fatalf("no overlap is in force, so no previous-key headers may be sent (got %q/%q)", prevKeyID, prevSig)
	}
}

// TestWebhookRotationOverlapOffersBothKeys proves a staged rotation keeps the
// outgoing key valid inside the bounded overlap: the delivery identifies the
// INCOMING key as active, still carries a verifiable signature from the
// outgoing key, and names that outgoing key by fingerprint.
func TestWebhookRotationOverlapOffersBothKeys(t *testing.T) {
	const outgoing = "outgoing-signing-key-0002"
	const incoming = "incoming-signing-key-0003"
	now := time.Now().UTC()
	payload := []byte(`{"b":2}`)

	rec := &rotReceiver{}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	wh := &WebhookConfig{
		ID: "wh1", SecretHMAC: incoming,
		SecretHMACPrevious: outgoing, SecretHMACRotatedAt: &now, Enabled: true,
	}
	d := wireDispatcher()
	h := buildWebhookSigningHeaders(WebhookSigningKeys(wh, time.Now().UTC()), payload)
	if _, err := d.doRequest(context.Background(), srv.URL+"/hook", payload, h); err != nil {
		t.Fatalf("doRequest returned an unexpected error: %v", err)
	}

	keyID, sig, prevKeyID, prevSig, hits := rec.snapshot()
	if hits != 1 {
		t.Fatalf("expected exactly one delivery, got %d", hits)
	}
	if keyID != WebhookKeyFingerprint(incoming) {
		t.Fatalf("active key id = %q, want the incoming key's fingerprint", keyID)
	}
	if !VerifySignature(payload, incoming, sig) {
		t.Fatal("active signature must verify with the incoming key")
	}
	if prevKeyID != WebhookKeyFingerprint(outgoing) {
		t.Fatalf("previous key id = %q, want the outgoing key's fingerprint", prevKeyID)
	}
	if !VerifySignature(payload, outgoing, prevSig) {
		t.Fatal("previous signature must verify with the outgoing key during the overlap")
	}
}

// TestWebhookRotationOverlapExpires proves the overlap is BOUNDED: once the
// window elapses the outgoing key is no longer offered, so old material is
// invalidated without an operator action.
func TestWebhookRotationOverlapExpires(t *testing.T) {
	const outgoing = "outgoing-signing-key-0004"
	const incoming = "incoming-signing-key-0005"
	stale := time.Now().UTC().Add(-2 * WebhookRotationOverlap)
	payload := []byte(`{"c":3}`)

	rec := &rotReceiver{}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	wh := &WebhookConfig{
		ID: "wh1", SecretHMAC: incoming,
		SecretHMACPrevious: outgoing, SecretHMACRotatedAt: &stale, Enabled: true,
	}
	d := wireDispatcher()
	h := buildWebhookSigningHeaders(WebhookSigningKeys(wh, time.Now().UTC()), payload)
	if _, err := d.doRequest(context.Background(), srv.URL+"/hook", payload, h); err != nil {
		t.Fatalf("doRequest returned an unexpected error: %v", err)
	}

	keyID, _, prevKeyID, prevSig, hits := rec.snapshot()
	if hits != 1 {
		t.Fatalf("expected exactly one delivery, got %d", hits)
	}
	if keyID != WebhookKeyFingerprint(incoming) {
		t.Fatalf("active key id = %q, want the incoming key's fingerprint", keyID)
	}
	if prevKeyID != "" || prevSig != "" {
		t.Fatalf("the overlap has elapsed, so the outgoing key must not be offered (got %q/%q)", prevKeyID, prevSig)
	}
}

// TestWebhookSigningKeysCutoverClearsPrevious proves an explicit cutover
// invalidates the outgoing key immediately, even inside the window: a config
// with no previous key offers exactly one key.
func TestWebhookSigningKeysCutoverClearsPrevious(t *testing.T) {
	const active = "cutover-signing-key-0006"
	now := time.Now().UTC()
	wh := &WebhookConfig{SecretHMAC: active, SecretHMACRotatedAt: &now}

	keys := WebhookSigningKeys(wh, time.Now().UTC())
	if len(keys) != 1 {
		t.Fatalf("a cutover must offer exactly one key, got %d", len(keys))
	}
	if keys[0].KeyID != WebhookKeyFingerprint(active) {
		t.Fatalf("the single key must be the active key, got %q", keys[0].KeyID)
	}
}

// TestWebhookSigningKeysEmptySecretOffersNone proves a config with no signing
// key offers no keys rather than an empty-key signature.
func TestWebhookSigningKeysEmptySecretOffersNone(t *testing.T) {
	if keys := WebhookSigningKeys(&WebhookConfig{}, time.Now().UTC()); len(keys) != 0 {
		t.Fatalf("a keyless config must offer no signing keys, got %d", len(keys))
	}
	if keys := WebhookSigningKeys(nil, time.Now().UTC()); len(keys) != 0 {
		t.Fatalf("a nil config must offer no signing keys, got %d", len(keys))
	}
}

// TestWebhookKeyFingerprintIsStableAndOpaque pins the identifier contract: the
// same key yields the same fingerprint, a different key a different one, and
// the fingerprint is never the secret itself.
func TestWebhookKeyFingerprintIsStableAndOpaque(t *testing.T) {
	const a = "fingerprint-key-aaaa"
	const b = "fingerprint-key-bbbb"

	if WebhookKeyFingerprint(a) != WebhookKeyFingerprint(a) {
		t.Fatal("the fingerprint must be stable for the same key")
	}
	if WebhookKeyFingerprint(a) == WebhookKeyFingerprint(b) {
		t.Fatal("distinct keys must yield distinct fingerprints")
	}
	fp := WebhookKeyFingerprint(a)
	if fp == a || strings.Contains(fp, a) {
		t.Fatalf("the fingerprint must not be the secret or contain it: %q", fp)
	}
}

// TestWebhookSigningHeadersNeverCarrySecret proves the wire header set is
// built from fingerprints only: the signing secrets appear nowhere in the
// header values, so a receiver learns which key signed without learning it.
func TestWebhookSigningHeadersNeverCarrySecret(t *testing.T) {
	const outgoing = "header-secret-outgoing"
	const incoming = "header-secret-incoming"
	now := time.Now().UTC()
	payload := []byte(`{"d":4}`)

	wh := &WebhookConfig{
		SecretHMAC: incoming, SecretHMACPrevious: outgoing, SecretHMACRotatedAt: &now,
	}
	h := buildWebhookSigningHeaders(WebhookSigningKeys(wh, time.Now().UTC()), payload)

	blob := strings.Join([]string{h.activeKeyID, h.previousKeyID}, "|")
	if strings.Contains(blob, incoming) || strings.Contains(blob, outgoing) {
		t.Fatalf("a key id must never be the signing secret: %q", blob)
	}
	// The signatures are HMACs of the payload, not the keys.
	if h.activeSignature == incoming || h.previousSignature == outgoing {
		t.Fatal("a signature must never be the signing secret")
	}
}
