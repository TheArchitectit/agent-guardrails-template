package notifications

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Webhook signing-key rotation (Spec 17 R17-10).
//
// R17-10 requires that rotation cover webhook material with "a staged
// replacement and explicit cutover", that old material be "invalidated after a
// bounded overlap", that service continuity be preserved "only while both are
// valid", and — the part this file is the single decision for — that "webhook
// rotation must identify the active signing key to receivers without leaking
// it".
//
// Before this contract the tree could only replace a webhook HMAC secret in
// place: `webhook_configs.secret_hmac` was overwritten, the very next delivery
// signed with the new value, and the receiver had no way to know a rotation had
// happened or which key signed a payload — a receiver that had not yet been
// updated simply rejected every delivery (or, worse, a receiver that never
// rotated kept accepting a value the operator believed was retired). There was
// no previous-key window, no key identity on the wire and no rotation record.
//
// The contract below is deliberately opaque: a key is identified on the wire
// and in the audit log only by a non-reversible fingerprint, never by the
// secret or anything reversible to it.

// WebhookKeyIDHeader names the signing key a delivery used, by fingerprint.
// A receiver uses it to select the matching verification key without either
// side transmitting the key.
const WebhookKeyIDHeader = "X-Guardrail-Key-Id"

// WebhookPreviousSignatureHeader carries the outgoing key's signature during a
// bounded rotation overlap, so a receiver that has not yet cut over still
// verifies (R17-10: "preserve service continuity only while both are valid").
const WebhookPreviousSignatureHeader = "X-Guardrail-Signature-Previous"

// WebhookPreviousKeyIDHeader names the outgoing key by fingerprint during the
// overlap window.
const WebhookPreviousKeyIDHeader = "X-Guardrail-Previous-Key-Id"

// WebhookRotationOverlap is the bounded window during which the outgoing
// signing key stays valid after a rotation. It matches the <=24-hour planned
// overlap Spec 16 R16-09 names for registered credentials; the two must not
// drift, so the value is a named constant rather than a per-call literal.
const WebhookRotationOverlap = 24 * time.Hour

// WebhookKeyFingerprint returns a short, non-reversible identifier for a
// webhook signing secret. Domain separation keeps it from colliding with the
// credential-registry verifier-key fingerprint (auth.keyFingerprint); the
// result is truncated because it is an identifier, not an authentication
// artifact — it cannot be used to sign or to verify anything.
func WebhookKeyFingerprint(secret string) string {
	sum := sha256.Sum256([]byte("guardrail-webhook-hmac-key-fingerprint:" + secret))
	return hex.EncodeToString(sum[:])[:12]
}

// WebhookSigningKey is one key valid for a delivery: its non-reversible
// fingerprint and the secret used to compute the signature. The secret never
// leaves this process.
type WebhookSigningKey struct {
	KeyID  string
	Secret string
}

// WebhookSigningKeys is the single decision for which keys are valid at a
// given instant. The active key is always first; the outgoing key is appended
// only while the rotation is inside the bounded overlap. A cutover clears the
// previous key outright, so it is absent here even inside the window. This is
// the rule both delivery paths (event and test) and the receiver-facing key
// identity use, so they cannot disagree about "which key is live".
func WebhookSigningKeys(wh *WebhookConfig, now time.Time) []WebhookSigningKey {
	if wh == nil || wh.SecretHMAC == "" {
		return nil
	}
	keys := []WebhookSigningKey{{KeyID: WebhookKeyFingerprint(wh.SecretHMAC), Secret: wh.SecretHMAC}}
	if wh.SecretHMACPrevious == "" || wh.SecretHMACRotatedAt == nil {
		return keys
	}
	if !now.Before(wh.SecretHMACRotatedAt.Add(WebhookRotationOverlap)) {
		// The overlap has elapsed: the outgoing key is invalid. The record is
		// left in place (a caller may still cut it over explicitly) but it is
		// not offered for signing.
		return keys
	}
	keys = append(keys, WebhookSigningKey{
		KeyID:  WebhookKeyFingerprint(wh.SecretHMACPrevious),
		Secret: wh.SecretHMACPrevious,
	})
	return keys
}

// WebhookRotationEvent describes one applied webhook signing-key rotation. It
// carries no key material and no secret: the key is identified only by its
// non-reversible fingerprint.
type WebhookRotationEvent struct {
	// WebhookID is the rotated endpoint.
	WebhookID string
	// Operation is "rotate" (a staged replacement that starts the overlap) or
	// "cutover" (the outgoing key invalidated immediately).
	Operation string
	// KeyID is the fingerprint of the key in force after the operation.
	KeyID string
	// PreviousKeyID is the fingerprint of the outgoing key, empty on cutover
	// of an already-expired window or when there was no previous key.
	PreviousKeyID string
	// OverlapSeconds is the bounded overlap window (0 for a cutover).
	OverlapSeconds int
	// OccurredAt is when the rotation was applied (UTC).
	OccurredAt time.Time
}
