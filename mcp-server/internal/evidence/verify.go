// Verification for the cross-product evidence contract (Spec 15 gr-xp-02,
// gr-xp-04, gr-xp-06).
//
// The four stages are deliberately separate functions AND separately reported
// in the verdict, because the failure this contract exists to prevent is
// "transport authenticated, therefore the artifact is trustworthy". A caller
// that only ever looks at Verified cannot tell an integrity failure from an
// authentication failure, so the verdict carries a distinct reason per stage.
package evidence

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Stage names, reported in the verdict.
const (
	StageTransport     = "transport"
	StageIntegrity     = "integrity"
	StageProducer      = "producer"
	StageAuthorization = "authorization"
)

// Reason codes. Distinct per stage: a consumer can act on them without parsing
// prose, and a test can assert the exact stage that failed.
const (
	ReasonTransportUnauthenticated = "transport_unauthenticated"
	ReasonIntegrityShape           = "integrity_shape_invalid"
	ReasonIntegritySignature       = "integrity_signature_invalid"
	ReasonIntegrityDigestMismatch  = "integrity_digest_mismatch"
	ReasonProducerUntrusted        = "producer_untrusted"
	ReasonProducerAudience         = "producer_audience_mismatch"
	ReasonProducerDirection        = "producer_direction_mismatch"
	ReasonKeyRevoked               = "producer_key_revoked"
	ReasonKeyExpired               = "producer_key_expired"
	ReasonExpired                  = "artifact_expired"
	ReasonNotYetValid              = "artifact_not_yet_valid"
	ReasonReplayed                 = "artifact_replayed"
	ReasonAuthorizationDenied      = "receiving_policy_denied"
	ReasonOutcomeNonPass           = "outcome_non_pass"
	ReasonOK                       = "ok"
)

// Trust is one peer key the receiver accepted out of band. Trust is never
// learned from the payload: this record is provisioned by the operator, and a
// key that is not here is untrusted no matter how valid its signature
// (gr-xp-03/gr-xp-04).
type Trust struct {
	ProducerID string
	PublicKey  ed25519.PublicKey
	Audience   string
	Direction  Direction
	// NotBefore/NotAfter bound the key's validity. Zero NotAfter means the key
	// has no declared expiry, which the load boundary refuses for a nonlocal
	// profile — an unbounded peer key is an unbounded trust grant.
	NotBefore time.Time
	NotAfter  time.Time
	Revoked   bool
}

// TrustStore is the set of provisioned peer keys, indexed by key ID. It is
// immutable once built; revocation replaces the snapshot (a mutation in place
// would leave a concurrent verifier reading half a change).
type TrustStore struct {
	byKeyID map[string]Trust
}

// NewTrustStore validates and indexes the provisioned records. A record with a
// missing producer, a non-Ed25519 key, an unrecognised direction, a
// contradictory validity window or a duplicate key ID is refused: a trust store
// that silently drops a bad record is a store whose contents cannot be audited.
func NewTrustStore(records []Trust) (*TrustStore, error) {
	if len(records) == 0 {
		return nil, errors.New("trust store is empty; a verifier with no trusted keys would reject every artifact for the wrong reason")
	}
	byKeyID := make(map[string]Trust, len(records))
	for i, rec := range records {
		if rec.ProducerID == "" {
			return nil, fmt.Errorf("trust record %d has no producer_id", i)
		}
		if len(rec.PublicKey) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("trust record %d (%s) key is not Ed25519", i, rec.ProducerID)
		}
		if rec.Direction != DirectionInbound && rec.Direction != DirectionOutbound {
			return nil, fmt.Errorf("trust record %d (%s) direction %q is not inbound/outbound", i, rec.ProducerID, rec.Direction)
		}
		if !rec.NotAfter.IsZero() && !rec.NotAfter.After(rec.NotBefore) {
			return nil, fmt.Errorf("trust record %d (%s) NotAfter must be after NotBefore", i, rec.ProducerID)
		}
		id := KeyIDFor(rec.PublicKey)
		if _, dup := byKeyID[id]; dup {
			return nil, fmt.Errorf("duplicate key id %s (two records share one public key)", id)
		}
		byKeyID[id] = rec
	}
	return &TrustStore{byKeyID: byKeyID}, nil
}

// Lookup returns the record for a key ID.
func (t *TrustStore) Lookup(keyID string) (Trust, bool) {
	rec, ok := t.byKeyID[keyID]
	return rec, ok
}

// Len reports the number of trusted keys (used to assert non-vacuity).
func (t *TrustStore) Len() int { return len(t.byKeyID) }

// ReplayGuard remembers consumed (evaluation_id, nonce, idempotency_key)
// triples so a replayed artifact cannot be accepted twice (gr-xp-04).
//
// The store is in-memory and bounded by the artifact freshness window: an entry
// older than the longest accepted window can never match a still-valid
// artifact, so pruning it cannot reopen a replay.
type ReplayGuard struct {
	mu      sync.Mutex
	seen    map[string]time.Time
	maxAge  time.Duration
	nowFunc func() time.Time
}

// NewReplayGuard builds a guard retaining entries for maxAge.
func NewReplayGuard(maxAge time.Duration, now func() time.Time) *ReplayGuard {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if maxAge <= 0 {
		maxAge = 10 * time.Minute
	}
	return &ReplayGuard{seen: make(map[string]time.Time), maxAge: maxAge, nowFunc: now}
}

func replayKey(e *Envelope) string {
	h := sha256.New()
	h.Write([]byte("guardrail-evidence-replay-v1\x00"))
	for _, part := range []string{e.ProducerID, e.EvaluationID, e.Nonce, e.IdempotencyKey} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// CheckAndRecord returns an error when the artifact has already been consumed,
// and otherwise records it. It is one operation on purpose: a check that a
// caller may forget to follow with a record is not a replay guard.
func (g *ReplayGuard) CheckAndRecord(e *Envelope) error {
	key := replayKey(e)
	now := g.nowFunc()

	g.mu.Lock()
	defer g.mu.Unlock()

	// Prune before checking so an expired entry cannot mask a live replay.
	for k, at := range g.seen {
		if now.Sub(at) > g.maxAge {
			delete(g.seen, k)
		}
	}
	if _, seen := g.seen[key]; seen {
		return errors.New("artifact replay: this evaluation_id/nonce/idempotency_key was already consumed")
	}
	g.seen[key] = now
	return nil
}

// Forget removes a recorded artifact. It exists for the path where a later
// stage fails: an artifact that was rejected downstream must not consume its
// idempotency key, or a corrected retry would be refused as a replay.
func (g *ReplayGuard) Forget(e *Envelope) {
	key := replayKey(e)
	g.mu.Lock()
	delete(g.seen, key)
	g.mu.Unlock()
}

// Verdict is the four-stage result. Each stage is reported independently.
type Verdict struct {
	// Verified is true only when every stage passed.
	Verified bool
	// Stage is the first stage that failed, or "ok".
	Stage      string
	ReasonCode string
	Reason     string
	// Outcome echoes the artifact's own verdict. It is NOT an authorization.
	Outcome Outcome
	// ProducerID is the identity established by the producer stage; empty when
	// that stage did not pass.
	ProducerID string
}

// Policy is the receiving side's authorization decision for evidence
// (gr-xp-06 stage 4). It is deliberately a function of the VERIFIED envelope
// only: content cannot influence it, and a peer artifact can never widen it.
type Policy func(env *Envelope, producerID string) error

// Verify runs the four stages in order over an artifact.
//
// transportAuthenticated is supplied by the caller, because transport
// authentication belongs to the connection (mTLS), not to the artifact. Passing
// false fails the first stage; passing true does NOT skip any later stage.
func (v *Verifier) Verify(env *Envelope, signatureB64 string, payload []byte, transportAuthenticated bool, policy Policy) Verdict {
	// Stage 1 — transport authentication.
	if !transportAuthenticated {
		return Verdict{Stage: StageTransport, ReasonCode: ReasonTransportUnauthenticated,
			Reason:  "transport peer was not authenticated; an artifact is not accepted over an unauthenticated connection",
			Outcome: env.Outcome}
	}

	// Stage 2 — artifact integrity (shape, then signature, then digests).
	if err := env.ValidateShape(); err != nil {
		return Verdict{Stage: StageIntegrity, ReasonCode: ReasonIntegrityShape, Reason: err.Error(), Outcome: env.Outcome}
	}
	canonical, err := env.CanonicalBytes()
	if err != nil {
		return Verdict{Stage: StageIntegrity, ReasonCode: ReasonIntegrityShape, Reason: err.Error(), Outcome: env.Outcome}
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return Verdict{Stage: StageIntegrity, ReasonCode: ReasonIntegritySignature,
			Reason: "signature is not a valid Ed25519 detached signature", Outcome: env.Outcome}
	}

	// Stage 3 — producer identity: the signature must verify under a key the
	// receiver trusts for THIS audience and THIS direction, at a valid time,
	// and the key must not be revoked or expired.
	trust, ok := v.trust.Lookup(env.KeyID)
	if !ok {
		return Verdict{Stage: StageProducer, ReasonCode: ReasonProducerUntrusted,
			Reason: "signing key is not a provisioned trust record", Outcome: env.Outcome}
	}
	// Verify the signature against the trusted public key. Note the order: the
	// trust lookup happens BEFORE verification so an unknown key never gets a
	// cryptographic verdict it could use to probe.
	signed := append([]byte(sigDomain), canonical...)
	if !ed25519.Verify(trust.PublicKey, signed, sig) {
		return Verdict{Stage: StageIntegrity, ReasonCode: ReasonIntegritySignature,
			Reason: "detached signature does not verify over the canonical envelope bytes", Outcome: env.Outcome}
	}
	if trust.ProducerID != env.ProducerID {
		return Verdict{Stage: StageProducer, ReasonCode: ReasonProducerUntrusted,
			Reason: "envelope producer_id does not match the trusted key's producer", Outcome: env.Outcome}
	}
	if trust.Direction != env.Direction {
		return Verdict{Stage: StageProducer, ReasonCode: ReasonProducerDirection,
			Reason: fmt.Sprintf("key is trusted for %q but the envelope declares %q", trust.Direction, env.Direction), Outcome: env.Outcome}
	}
	if trust.Audience != env.Audience {
		return Verdict{Stage: StageProducer, ReasonCode: ReasonProducerAudience,
			Reason: fmt.Sprintf("key is trusted for audience %q but the envelope addresses %q", trust.Audience, env.Audience), Outcome: env.Outcome}
	}
	now := v.now()
	if trust.Revoked {
		return Verdict{Stage: StageProducer, ReasonCode: ReasonKeyRevoked,
			Reason: "signing key is revoked; a mathematically valid signature from a revoked key is not usable", Outcome: env.Outcome}
	}
	if !trust.NotAfter.IsZero() && now.After(trust.NotAfter) {
		return Verdict{Stage: StageProducer, ReasonCode: ReasonKeyExpired,
			Reason: "signing key is past its validity window", Outcome: env.Outcome}
	}
	if now.Before(trust.NotBefore) && !trust.NotBefore.IsZero() {
		return Verdict{Stage: StageProducer, ReasonCode: ReasonKeyExpired,
			Reason: "signing key is not yet valid", Outcome: env.Outcome}
	}

	// Freshness and payload binding. These are integrity of the CLAIM rather
	// than of the bytes, but they belong to the same stage: an artifact whose
	// window has closed or whose payload no longer matches its digest cannot be
	// treated as evidence about the present.
	if now.After(env.ExpiresAt) {
		return Verdict{Stage: StageIntegrity, ReasonCode: ReasonExpired,
			Reason: "artifact is past expires_at", Outcome: env.Outcome}
	}
	if now.Before(env.IssuedAt) {
		return Verdict{Stage: StageIntegrity, ReasonCode: ReasonNotYetValid,
			Reason: "artifact issued_at is in the future", Outcome: env.Outcome}
	}
	if env.PayloadDigest != PayloadDigest(payload) {
		return Verdict{Stage: StageIntegrity, ReasonCode: ReasonIntegrityDigestMismatch,
			Reason: "payload does not match payload_digest", Outcome: env.Outcome}
	}

	// Replay, after the artifact is proven genuine: recording a forged artifact
	// would let an attacker burn an idempotency key.
	if err := v.replay.CheckAndRecord(env); err != nil {
		return Verdict{Stage: StageIntegrity, ReasonCode: ReasonReplayed, Reason: err.Error(), Outcome: env.Outcome}
	}

	// Stage 4 — receiving-policy authorization.
	if policy != nil {
		if err := policy(env, trust.ProducerID); err != nil {
			v.replay.Forget(env)
			return Verdict{Stage: StageAuthorization, ReasonCode: ReasonAuthorizationDenied,
				Reason: err.Error(), Outcome: env.Outcome, ProducerID: trust.ProducerID}
		}
	}

	// A non-PASS outcome is a successful VERIFICATION of a non-PASS claim: the
	// artifact is genuine, and it says the check did not pass. It is reported
	// as verified with the real outcome so the caller cannot mistake genuine
	// evidence of a block for a failure to verify.
	return Verdict{Verified: true, Stage: "ok", ReasonCode: ReasonOK, Reason: "all four stages passed",
		Outcome: env.Outcome, ProducerID: trust.ProducerID}
}

// Verifier holds the trust snapshot and replay guard. It is safe for
// concurrent use.
type Verifier struct {
	trust  *TrustStore
	replay *ReplayGuard
	now    func() time.Time
}

// NewVerifier builds a verifier. A nil replay guard gets a bounded default, and
// a nil clock uses UTC now.
func NewVerifier(trust *TrustStore, replay *ReplayGuard, now func() time.Time) (*Verifier, error) {
	if trust == nil || trust.Len() == 0 {
		return nil, errors.New("verifier requires a non-empty trust store")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if replay == nil {
		replay = NewReplayGuard(10*time.Minute, now)
	}
	return &Verifier{trust: trust, replay: replay, now: now}, nil
}

// ConstantTimeEqualStrings compares two strings without leaking length-by-timing
// beyond the length itself. Used where a digest comparison is a security
// boundary rather than a convenience.
func ConstantTimeEqualStrings(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
