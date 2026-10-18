// Package evidence implements the Secure Cross-Product Evidence Method
// (Spec 15, requirements gr-xp-01 .. gr-xp-06).
//
// The contract exists because "a library exists" is not evidence that a control
// ran, and because two products must be able to exchange bounded evidence
// without either one acquiring authority over the other. Three rules drive the
// whole package:
//
//  1. Identity is never derived from content. A principal comes from a
//     registered trust record, not from a claim inside a payload, a policy
//     text, a tool output or a model response (gr-xp-05).
//  2. A signed artifact is evidence, never an authorization grant. Passing
//     verification says "this producer said this"; it says nothing about
//     whether the receiving host may perform an effect (gr-xp-03).
//  3. The four verification stages are separate and fail with distinct
//     reasons: transport authentication, artifact integrity, producer
//     identity, receiving-policy authorization. Collapsing them into one
//     boolean is how "authenticated" ends up standing in for "integrity
//     verified" (gr-xp-06).
package evidence

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ContractVersion is the envelope contract this build implements. A consumer
// refuses an envelope whose version it does not implement rather than
// reinterpreting unknown fields (fail closed).
const ContractVersion = "1"

// Direction is the flow of an artifact. Each direction uses its own credential
// (gr-xp-03); a key trusted to submit evidence inbound is not thereby trusted
// to accept evidence outbound.
type Direction string

const (
	DirectionInbound  Direction = "inbound"  // peer -> guardrails
	DirectionOutbound Direction = "outbound" // guardrails -> peer
)

// Outcome is the guardrail verdict carried by an artifact. It is deliberately
// distinct from a host authorization decision: `pass` here means the checks
// named in the artifact passed, not that any effect is permitted.
type Outcome string

const (
	OutcomePass         Outcome = "pass"
	OutcomeBlock        Outcome = "block"
	OutcomeInconclusive Outcome = "inconclusive"
	OutcomeError        Outcome = "error"
	OutcomeSkipped      Outcome = "skipped"
)

// Terminal reports whether an outcome may satisfy a mandatory check. Only
// `pass` does; every other value — including the ones a producer might be
// tempted to treat as "not a failure" — is non-PASS (gr-xp-04/gr-oap-03).
func (o Outcome) Terminal() bool { return o == OutcomePass }

// Domain separation for the key identifier and the signature. Without it the
// same 32 bytes could be reinterpreted as a signature over a different
// structure, and a key ID could collide with some other digest of the same
// material.
const (
	keyIDDomain = "guardrail-evidence-key-v1\x00"
	sigDomain   = "guardrail-evidence-envelope-v1\x00"
)

// Envelope is the bounded, canonical artifact header required by gr-xp-02 and
// gr-oap-03. Every field is either an identity the receiver can check against
// its own trust store or a digest binding the artifact to what was evaluated.
//
// It contains no secret and no raw credential. The signature is detached: it
// travels beside the envelope, never inside the bytes it attests.
type Envelope struct {
	ContractVersion string    `json:"contract_version"`
	Direction       Direction `json:"direction"`
	ProducerID      string    `json:"producer_id"`
	KeyID           string    `json:"key_id"`
	Audience        string    `json:"audience"`
	Tenant          string    `json:"tenant,omitempty"`
	SubjectDigest   string    `json:"subject_digest"`
	PolicyDigest    string    `json:"policy_digest"`
	ComponentID     string    `json:"component_id"`
	ComponentVer    string    `json:"component_version"`
	RequestID       string    `json:"request_id"`
	EvaluationID    string    `json:"evaluation_id"`
	IdempotencyKey  string    `json:"idempotency_key"`
	Nonce           string    `json:"nonce"`
	IssuedAt        time.Time `json:"issued_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	PayloadDigest   string    `json:"payload_digest"`
	ManifestDigest  string    `json:"evidence_manifest_digest"`
	Outcome         Outcome   `json:"outcome"`
	ReasonCode      string    `json:"reason_code"`
	Reason          string    `json:"reason,omitempty"`
}

// ValidateShape rejects an envelope that cannot be verified at all: a missing
// binding, an unimplemented contract version, or a non-PASS outcome with no
// reason. Shape problems are integrity failures, reported before any
// cryptography runs, because a signature over an incomplete envelope proves
// nothing about the requirement it claims to satisfy.
func (e *Envelope) ValidateShape() error {
	if e.ContractVersion != ContractVersion {
		return fmt.Errorf("contract_version %q is not implemented by this build (want %q)", e.ContractVersion, ContractVersion)
	}
	if e.Direction != DirectionInbound && e.Direction != DirectionOutbound {
		return fmt.Errorf("direction %q is not one of inbound/outbound", e.Direction)
	}
	required := map[string]string{
		"producer_id":              e.ProducerID,
		"key_id":                   e.KeyID,
		"audience":                 e.Audience,
		"subject_digest":           e.SubjectDigest,
		"policy_digest":            e.PolicyDigest,
		"component_id":             e.ComponentID,
		"component_version":        e.ComponentVer,
		"request_id":               e.RequestID,
		"evaluation_id":            e.EvaluationID,
		"idempotency_key":          e.IdempotencyKey,
		"nonce":                    e.Nonce,
		"payload_digest":           e.PayloadDigest,
		"evidence_manifest_digest": e.ManifestDigest,
	}
	for name, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required and empty", name)
		}
	}
	if e.IssuedAt.IsZero() || e.ExpiresAt.IsZero() {
		return errors.New("issued_at and expires_at are required")
	}
	if !e.ExpiresAt.After(e.IssuedAt) {
		return errors.New("expires_at must be after issued_at")
	}
	if e.Outcome == "" {
		return errors.New("outcome is required")
	}
	// A non-PASS artifact must say why. An unexplained block/error is not
	// reviewable evidence, and an empty reason is how a producer hides the
	// difference between "the check failed" and "the check never ran".
	if !e.Outcome.Terminal() && e.ReasonCode == "" {
		return fmt.Errorf("outcome %q requires a reason_code", e.Outcome)
	}
	return nil
}

// CanonicalBytes returns the byte sequence that is signed and verified. It is
// the JSON encoding of the envelope struct itself: Go marshals struct fields in
// declaration order, so the bytes are stable for a given field set without
// depending on a map's iteration order. No signature, no whitespace variance.
func (e *Envelope) CanonicalBytes() ([]byte, error) {
	return json.Marshal(e)
}

// Instance is one product instance's signing identity: an Ed25519 key pair plus
// the key ID derived from the public half.
//
// The key ID is a domain-separated digest of the PUBLIC key, so it is not a
// secret, and it cannot be confused with a truncated log hash of a credential
// (the identity source gr-xp-01 forbids).
type Instance struct {
	producerID string
	public     ed25519.PublicKey
	private    ed25519.PrivateKey
}

// NewInstance builds the signing identity for a producer. The private key is
// supplied by the deployment (see LoadInstanceKey); this constructor never
// generates one implicitly, so a process cannot quietly mint a new identity and
// invalidate every peer's trust record.
func NewInstance(producerID string, private ed25519.PrivateKey) (*Instance, error) {
	if producerID == "" {
		return nil, errors.New("producer ID is required")
	}
	if len(private) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("signing key must be an Ed25519 private key (%d bytes), got %d", ed25519.PrivateKeySize, len(private))
	}
	pub, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("signing key has no Ed25519 public half")
	}
	return &Instance{producerID: producerID, public: pub, private: private}, nil
}

// KeyID is the public identifier of this instance's signing key.
func (i *Instance) KeyID() string { return KeyIDFor(i.public) }

// ProducerID is the identity this instance signs as.
func (i *Instance) ProducerID() string { return i.producerID }

// PublicKey returns a copy of the verification key, for out-of-band
// provisioning into a peer's trust store.
func (i *Instance) PublicKey() ed25519.PublicKey {
	out := make(ed25519.PublicKey, len(i.public))
	copy(out, i.public)
	return out
}

// KeyIDFor derives the public key identifier: a domain-separated SHA-256 of the
// public key, hex encoded with an explicit algorithm prefix.
func KeyIDFor(pub ed25519.PublicKey) string {
	h := sha256.New()
	h.Write([]byte(keyIDDomain))
	h.Write(pub)
	return "ed25519:" + hex.EncodeToString(h.Sum(nil))
}

// Sign produces the detached signature over the envelope's canonical bytes.
// The envelope is shape-checked first: signing an incomplete envelope would
// produce a valid signature over an artifact that cannot satisfy the contract.
func (i *Instance) Sign(env *Envelope) (string, error) {
	if err := env.ValidateShape(); err != nil {
		return "", err
	}
	// Bind the envelope to the signing identity: a producer cannot sign as
	// somebody else, and a receiver can reject the mismatch before any
	// expensive work.
	env.ProducerID = i.producerID
	env.KeyID = i.KeyID()
	canonical, err := env.CanonicalBytes()
	if err != nil {
		return "", err
	}
	signed := append([]byte(sigDomain), canonical...)
	return base64.StdEncoding.EncodeToString(ed25519.Sign(i.private, signed)), nil
}

// PayloadDigest returns the digest form used for subject and payload binding.
// It is a plain SHA-256 over the bytes; the caller decides what the bytes are.
func PayloadDigest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}
