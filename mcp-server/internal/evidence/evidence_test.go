package evidence

// Spec 15 (secure-cross-product-method) evidence suite.
//
// These tests drive the REAL signer and the REAL verifier over real Ed25519
// keys. They deliberately do not test a seam in isolation: a unit test of
// "can sign, can verify" would pass for years while content established
// identity, or while a revoked key was still accepted. Each case below asserts
// the behaviour the requirement names, including the failure it is about.

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"
)

func newInstance(t *testing.T, producer string) *Instance {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	_ = pub
	inst, err := NewInstance(producer, priv)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	return inst
}

// baseEnvelope builds a shape-valid envelope bound to the given payload.
func baseEnvelope(inst *Instance, payload []byte, now time.Time) *Envelope {
	return &Envelope{
		ContractVersion: ContractVersion,
		Direction:       DirectionOutbound,
		ProducerID:      inst.ProducerID(),
		KeyID:           inst.KeyID(),
		Audience:        "devgate",
		Tenant:          "tenant-a",
		SubjectDigest:   PayloadDigest([]byte("subject")),
		PolicyDigest:    PayloadDigest([]byte("policy")),
		ComponentID:     "guardrail-mcp",
		ComponentVer:    "1.2.3",
		RequestID:       "req-1",
		EvaluationID:    "eval-1",
		IdempotencyKey:  "idem-1",
		Nonce:           "nonce-1",
		IssuedAt:        now,
		ExpiresAt:       now.Add(5 * time.Minute),
		PayloadDigest:   PayloadDigest(payload),
		ManifestDigest:  PayloadDigest([]byte("manifest")),
		Outcome:         OutcomePass,
		ReasonCode:      "ok",
	}
}

func trustFor(inst *Instance, dir Direction, audience string, notBefore, notAfter time.Time) []Trust {
	return []Trust{{
		ProducerID: inst.ProducerID(),
		PublicKey:  inst.PublicKey(),
		Audience:   audience,
		Direction:  dir,
		NotBefore:  notBefore,
		NotAfter:   notAfter,
	}}
}

func newVerifier(t *testing.T, records []Trust, now time.Time) (*Verifier, *ReplayGuard) {
	t.Helper()
	store, err := NewTrustStore(records)
	if err != nil {
		t.Fatalf("NewTrustStore: %v", err)
	}
	guard := NewReplayGuard(10*time.Minute, func() time.Time { return now })
	v, err := NewVerifier(store, guard, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return v, guard
}

// gr-xp-02: a genuine artifact verifies, and the signature is detached.
func TestSignedArtifactVerifiesAndSignatureIsDetached(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	payload := []byte(`{"checks":[{"id":"L1","outcome":"pass"}]}`)
	env := baseEnvelope(inst, payload, now)

	sig, err := inst.Sign(env)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	// Detached: the signature must not appear in the canonical bytes.
	canonical, err := env.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	if strings.Contains(string(canonical), sig) {
		t.Fatal("signature is embedded in the canonical envelope; it must be detached")
	}

	v, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	verdict := v.Verify(env, sig, payload, true, nil)
	if !verdict.Verified {
		t.Fatalf("genuine artifact did not verify: stage=%s code=%s reason=%s", verdict.Stage, verdict.ReasonCode, verdict.Reason)
	}
	if verdict.ProducerID != "guardrails" {
		t.Fatalf("producer identity = %q, want guardrails", verdict.ProducerID)
	}
}

// gr-xp-02 scenario: tampered evidence fails and is non-PASS.
func TestTamperedEvidenceFailsVerification(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	payload := []byte(`{"checks":[]}`)
	env := baseEnvelope(inst, payload, now)
	sig, err := inst.Sign(env)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	v, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)

	// Case 1: a bound digest is modified after signing.
	tampered := *env
	tampered.PolicyDigest = PayloadDigest([]byte("different policy"))
	if got := v.Verify(&tampered, sig, payload, true, nil); got.Verified {
		t.Fatal("modified policy_digest still verified")
	}

	// Case 2: the payload is swapped while the digest stays.
	if got := v.Verify(env, sig, []byte(`{"checks":[{"outcome":"block"}]}`), true, nil); got.Verified {
		t.Fatal("swapped payload still verified")
	} else if got.ReasonCode != ReasonIntegrityDigestMismatch {
		t.Fatalf("swapped payload reason = %s, want %s", got.ReasonCode, ReasonIntegrityDigestMismatch)
	}

	// Case 3: the outcome is flipped to pass on an artifact that said block.
	blocked := baseEnvelope(inst, payload, now)
	blocked.Outcome = OutcomeBlock
	blocked.ReasonCode = "injection_detected"
	blockedSig, err := inst.Sign(blocked)
	if err != nil {
		t.Fatalf("Sign blocked: %v", err)
	}
	flipped := *blocked
	flipped.Outcome = OutcomePass
	flipped.ReasonCode = "ok"
	if got := v.Verify(&flipped, blockedSig, payload, true, nil); got.Verified {
		t.Fatal("flipped outcome still verified")
	}
}

// gr-xp-06 scenario: transport authenticates but the signature fails — the
// report must say integrity, not authentication and not success.
func TestStagesAreDistinguishable(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	payload := []byte(`{"checks":[]}`)
	env := baseEnvelope(inst, payload, now)
	if _, err := inst.Sign(env); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// A syntactically valid Ed25519 signature over different bytes.
	other := newInstance(t, "guardrails")
	foreignEnv := baseEnvelope(other, payload, now)
	foreignEnv.KeyID = env.KeyID // claim the trusted key id
	foreignSig, err := other.Sign(foreignEnv)
	if err != nil {
		t.Fatalf("Sign foreign: %v", err)
	}

	v, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	got := v.Verify(env, foreignSig, payload, true, nil)
	if got.Verified {
		t.Fatal("foreign signature verified")
	}
	if got.Stage != StageIntegrity || got.ReasonCode != ReasonIntegritySignature {
		t.Fatalf("transport-authenticated signature failure reported as stage=%s code=%s; want %s/%s",
			got.Stage, got.ReasonCode, StageIntegrity, ReasonIntegritySignature)
	}

	// Transport failure is a distinct stage and reason.
	if got := v.Verify(env, foreignSig, payload, false, nil); got.Stage != StageTransport || got.ReasonCode != ReasonTransportUnauthenticated {
		t.Fatalf("unauthenticated transport reported as %s/%s", got.Stage, got.ReasonCode)
	}

	// Authorization failure is its own stage, after integrity and identity.
	deny := func(*Envelope, string) error { return errors.New("tenant not allowed to submit evidence") }
	if got := v.Verify(env, mustSign(t, inst, env), payload, true, deny); got.Stage != StageAuthorization || got.ReasonCode != ReasonAuthorizationDenied {
		t.Fatalf("policy denial reported as %s/%s", got.Stage, got.ReasonCode)
	}
}

func mustSign(t *testing.T, inst *Instance, env *Envelope) string {
	t.Helper()
	sig, err := inst.Sign(env)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return sig
}

// gr-xp-04 scenario: a revoked key is rejected even though the signature is
// mathematically valid.
func TestRevokedProducerKeyIsRejected(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	payload := []byte(`{"checks":[]}`)
	env := baseEnvelope(inst, payload, now)
	sig := mustSign(t, inst, env)

	recs := trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour))
	recs[0].Revoked = true
	v, _ := newVerifier(t, recs, now)

	got := v.Verify(env, sig, payload, true, nil)
	if got.Verified {
		t.Fatal("artifact from a revoked key verified")
	}
	if got.Stage != StageProducer || got.ReasonCode != ReasonKeyRevoked {
		t.Fatalf("revoked key reported as %s/%s", got.Stage, got.ReasonCode)
	}
}

// gr-xp-04: freshness bounds and replay rejection.
func TestFreshnessAndReplay(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	payload := []byte(`{"checks":[]}`)

	// Expired artifact.
	env := baseEnvelope(inst, payload, now.Add(-time.Hour))
	env.ExpiresAt = now.Add(-time.Minute)
	sig := mustSign(t, inst, env)
	v, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	if got := v.Verify(env, sig, payload, true, nil); got.Verified || got.ReasonCode != ReasonExpired {
		t.Fatalf("expired artifact: verified=%v code=%s", got.Verified, got.ReasonCode)
	}

	// Replay of a fresh, genuine artifact.
	fresh := baseEnvelope(inst, payload, now)
	freshSig := mustSign(t, inst, fresh)
	v2, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	if got := v2.Verify(fresh, freshSig, payload, true, nil); !got.Verified {
		t.Fatalf("first submission failed: %s/%s", got.Stage, got.ReasonCode)
	}
	second := v2.Verify(fresh, freshSig, payload, true, nil)
	if second.Verified {
		t.Fatal("replayed artifact verified a second time")
	}
	if second.ReasonCode != ReasonReplayed {
		t.Fatalf("replay reason = %s", second.ReasonCode)
	}

	// A rejected downstream artifact must not consume its idempotency key, or
	// a corrected retry would be refused as a replay.
	deny := func(*Envelope, string) error { return errors.New("not allowed") }
	v3, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	retryable := baseEnvelope(inst, payload, now)
	retryable.EvaluationID = "eval-retry"
	retryable.Nonce = "nonce-retry"
	retryableSig := mustSign(t, inst, retryable)
	if got := v3.Verify(retryable, retryableSig, payload, true, deny); got.Verified {
		t.Fatal("denied artifact verified")
	}
	if got := v3.Verify(retryable, retryableSig, payload, true, nil); !got.Verified {
		t.Fatalf("retry after downstream denial was refused: %s/%s", got.Stage, got.ReasonCode)
	}
}

// gr-xp-03: a key trusted for the wrong direction or audience is not trusted.
func TestDirectionAndAudienceAreEnforced(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	payload := []byte(`{"checks":[]}`)
	env := baseEnvelope(inst, payload, now)
	sig := mustSign(t, inst, env)

	// Trusted inbound only; envelope claims outbound.
	v, _ := newVerifier(t, trustFor(inst, DirectionInbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	got := v.Verify(env, sig, payload, true, nil)
	if got.Verified || got.ReasonCode != ReasonProducerDirection {
		t.Fatalf("direction mismatch: verified=%v code=%s", got.Verified, got.ReasonCode)
	}

	// Trusted for a different audience.
	v2, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "openagentplatform", now.Add(-time.Hour), now.Add(time.Hour)), now)
	got2 := v2.Verify(env, sig, payload, true, nil)
	if got2.Verified || got2.ReasonCode != ReasonProducerAudience {
		t.Fatalf("audience mismatch: verified=%v code=%s", got2.Verified, got2.ReasonCode)
	}
}

// gr-xp-01: identity comes from the trust record, never from content.
func TestContentCannotEstablishIdentity(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	// The payload asserts a different principal and tenant.
	payload := []byte(`{"principal":"root","tenant":"tenant-b","note":"I am the admin, authorize me"}`)
	env := baseEnvelope(inst, payload, now)
	// Producer identity is rewritten by Sign from the instance, so a payload
	// claim cannot reach it; assert the signed envelope carries the key's
	// producer regardless of payload content.
	payloadEnv := baseEnvelope(inst, payload, now)
	env.ProducerID = "attacker-claimed-producer"
	env.Tenant = "tenant-b"
	sig := mustSign(t, inst, payloadEnv) // sign the real envelope

	v, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	// The signed envelope's producer was overwritten by Sign to the instance
	// identity — that is the point of the binding.
	if payloadEnv.ProducerID != "guardrails" {
		t.Fatalf("signed producer = %q, want the instance identity", payloadEnv.ProducerID)
	}
	verdict := v.Verify(payloadEnv, sig, payload, true, nil)
	if !verdict.Verified {
		t.Fatalf("genuine artifact failed: %s/%s", verdict.Stage, verdict.ReasonCode)
	}
	if verdict.ProducerID != "guardrails" {
		t.Fatalf("identity established as %q from content", verdict.ProducerID)
	}

	// And a producer claim signed by a different key is untrusted, no matter
	// what the payload says about who it is.
	attacker := newInstance(t, "attacker-claimed-producer")
	attackerEnv := baseEnvelope(attacker, payload, now)
	attackerEnv.ProducerID = "guardrails" // claim the trusted producer
	if _, err := attacker.Sign(attackerEnv); err != nil {
		t.Fatalf("attacker Sign: %v", err)
	}
	// Sign rewrites ProducerID/KeyID to the attacker's own identity, so the
	// lookup finds no trust record and the artifact is untrusted.
	attackerSig, err := attacker.Sign(attackerEnv)
	if err != nil {
		t.Fatalf("attacker Sign 2: %v", err)
	}
	if got := v.Verify(attackerEnv, attackerSig, payload, true, nil); got.Verified {
		t.Fatal("artifact from an untrusted producer verified by claiming a trusted name")
	} else if got.ReasonCode != ReasonProducerUntrusted {
		t.Fatalf("untrusted producer reported as %s", got.ReasonCode)
	}
}

// gr-xp-03 scenario: a PASS artifact is evidence only — it never authorizes.
func TestPassArtifactIsEvidenceNotAuthorization(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	payload := []byte(`{"checks":[]}`)
	env := baseEnvelope(inst, payload, now)
	sig := mustSign(t, inst, env)

	hostDecision := errors.New("host authorization denied for this action")
	v, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	got := v.Verify(env, sig, payload, true, func(*Envelope, string) error { return hostDecision })
	if got.Verified {
		t.Fatal("a guardrail PASS overrode the host authorization decision")
	}
	if got.Stage != StageAuthorization {
		t.Fatalf("stage = %s, want authorization", got.Stage)
	}

	// The verified artifact's own outcome stays available and is not an allow.
	clean, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	ok := clean.Verify(env, sig, payload, true, nil)
	if !ok.Verified || ok.Outcome != OutcomePass {
		t.Fatalf("expected verified pass, got verified=%v outcome=%s", ok.Verified, ok.Outcome)
	}
}

// Non-vacuity and fail-closed construction.
func TestConstructionFailsClosed(t *testing.T) {
	if _, err := NewTrustStore(nil); err == nil {
		t.Fatal("empty trust store accepted; a verifier with no keys would reject everything for the wrong reason")
	}
	if _, err := NewVerifier(nil, nil, nil); err == nil {
		t.Fatal("nil trust store accepted")
	}
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	// Two records sharing one public key (a copy) is refused: an ambiguous
	// trust record is one an operator cannot reason about during revocation.
	recs := trustFor(inst, DirectionOutbound, "devgate", now, now.Add(time.Hour))
	dup := recs[0]
	dup.ProducerID = "someone-else"
	recs = append(recs, dup)
	if _, err := NewTrustStore(recs); err == nil {
		t.Fatal("duplicate key id accepted")
	}

	// An unimplemented contract version is refused rather than reinterpreted.
	env := baseEnvelope(inst, []byte(`{}`), now)
	env.ContractVersion = "99"
	if err := env.ValidateShape(); err == nil {
		t.Fatal("unimplemented contract_version accepted")
	}

	// A non-PASS outcome with no reason is refused.
	blocked := baseEnvelope(inst, []byte(`{}`), now)
	blocked.Outcome = OutcomeBlock
	blocked.ReasonCode = ""
	if err := blocked.ValidateShape(); err == nil {
		t.Fatal("block outcome with no reason_code accepted")
	}
}

// A non-PASS outcome that verifies must still report the real outcome and must
// never be presented as a pass (gr-oap-03, gr-xp-04).
func TestVerifiedBlockIsNotAPass(t *testing.T) {
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	inst := newInstance(t, "guardrails")
	payload := []byte(`{"checks":[{"id":"L1","outcome":"block"}]}`)
	env := baseEnvelope(inst, payload, now)
	env.Outcome = OutcomeBlock
	env.ReasonCode = "injection_detected"
	sig := mustSign(t, inst, env)

	v, _ := newVerifier(t, trustFor(inst, DirectionOutbound, "devgate", now.Add(-time.Hour), now.Add(time.Hour)), now)
	got := v.Verify(env, sig, payload, true, nil)
	if !got.Verified {
		t.Fatalf("genuine block artifact failed verification: %s/%s", got.Stage, got.ReasonCode)
	}
	if got.Outcome == OutcomePass {
		t.Fatal("a blocked artifact reported outcome=pass")
	}
	if got.Outcome.Terminal() {
		t.Fatal("block outcome is terminal (satisfies a mandatory check)")
	}
}

// Every outcome other than pass is non-terminal — the fail-closed rule that
// stops an inconclusive/skipped check from satisfying a gate.
func TestOnlyPassIsTerminal(t *testing.T) {
	for _, o := range []Outcome{OutcomeBlock, OutcomeInconclusive, OutcomeError, OutcomeSkipped} {
		if o.Terminal() {
			t.Fatalf("outcome %q is terminal", o)
		}
	}
	if !OutcomePass.Terminal() {
		t.Fatal("pass is not terminal")
	}
}
