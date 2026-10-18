package mcp

// Spec 15 (secure-cross-product-method) request-path wiring.
//
// The evidence package proves an artifact can be signed and verified; that
// alone would be another library with no caller — the exact defect this
// program keeps finding. So the signer is attached to the tools whose results
// ARE peer-consumable evidence: the prompt-injection tools. Their verdicts
// already carry everything gr-oap-03 requires (entry path, component identity,
// policy digest, subject digest, outcome, reason), and a peer product
// (OpenAgentPlatform, DevGate) is the natural consumer.
//
// Why these tools and not the webhook path: a webhook delivery is a
// notification, and manufacturing an `outcome` for it would be inventing
// semantics. An injection verdict has a real outcome, so the envelope reports
// what actually happened instead of a plausible guess.

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// EvidenceTTL bounds how long a produced artifact stays verifiable. A peer that
// is offline past this window must re-request rather than accept stale
// evidence (gr-xp-04 freshness).
const EvidenceTTL = 10 * time.Minute

// evidencePolicyDigest names the policy/profile identity the verdict was
// reached under. It is a digest of the declared policy identity rather than of
// the input text, because the artifact must say WHICH policy produced the
// answer — that is the half a reviewer needs and the half a payload digest
// cannot supply.
//
// The value is stable per build+policy set: the R17-10 config digest fields and
// the seeded default pattern set identify it. A change to either changes this
// string, so a peer can detect that an artifact was produced under a different
// policy than the one it is evaluating against.
func evidencePolicyDigest(engine *guardrails.Engine) string {
	identity := "guardrail-mcp/policy/injection"
	if engine != nil {
		identity += "/l1-seeded"
	}
	return evidence.PayloadDigest([]byte(identity))
}

// signedEvidence is the block attached to a tool result when an instance key is
// configured. The signature is detached from the envelope by construction: it
// is a sibling field, never part of the bytes it attests.
// signedEvidence is the block attached to a result when an instance key is
// configured.
//
// Signed is always emitted — on both the signed and the unsigned path — so a
// consumer tests one field instead of inferring provenance from whether a
// signature happens to be present. An absent key is not a missing field; it is
// a stated fact about the artifact.
type signedEvidence struct {
	Signed          bool               `json:"signed"`
	ContractVersion string             `json:"contract_version"`
	Envelope        *evidence.Envelope `json:"envelope"`
	Signature       string             `json:"signature"`
	KeyID           string             `json:"key_id"`
}

// buildEvidence constructs and signs an envelope for one verdict.
//
// The caller supplies the verified facts; this function never invents an
// outcome. A scan that did not run must not reach here at all — the handlers
// fail closed before this point.
func (s *MCPServer) buildEvidence(entryPath, subject string, outcome evidence.Outcome, reasonCode, reason string) (*signedEvidence, error) {
	if s.evidenceSigner == nil {
		return nil, nil
	}
	now := time.Now().UTC()

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		// A nonce that cannot be generated means the artifact cannot be made
		// replay-resistant; refuse rather than emit a predictable one.
		return nil, err
	}
	nonce := hex.EncodeToString(nonceBytes)

	evaluationID := uuid.New().String()
	env := &evidence.Envelope{
		ContractVersion: evidence.ContractVersion,
		Direction:       evidence.DirectionOutbound,
		ProducerID:      s.evidenceSigner.ProducerID(),
		KeyID:           s.evidenceSigner.KeyID(),
		Audience:        s.evidenceAudience,
		SubjectDigest:   evidence.PayloadDigest([]byte(subject)),
		PolicyDigest:    evidencePolicyDigest(s.guardrailsEngine),
		ComponentID:     "guardrail-mcp:" + entryPath,
		ComponentVer:    s.version,
		RequestID:       evaluationID,
		EvaluationID:    evaluationID,
		IdempotencyKey:  evaluationID,
		Nonce:           nonce,
		IssuedAt:        now,
		ExpiresAt:       now.Add(EvidenceTTL),
		// The payload is the verdict block itself, so the digest binds the
		// signature to the answer the caller received, not to a summary of it.
		PayloadDigest:  evidence.PayloadDigest([]byte(outcomePayload(outcome, reasonCode))),
		ManifestDigest: evidence.PayloadDigest([]byte(evidencePolicyDigest(s.guardrailsEngine))),
		Outcome:        outcome,
		ReasonCode:     reasonCode,
		Reason:         reason,
	}

	signature, err := s.evidenceSigner.Sign(env)
	if err != nil {
		return nil, err
	}
	return &signedEvidence{
		Signed:          true,
		ContractVersion: evidence.ContractVersion,
		Envelope:        env,
		Signature:       signature,
		KeyID:           env.KeyID,
	}, nil
}

// outcomePayload is the canonical bytes a verdict's payload digest covers. It
// is deliberately the minimal (outcome, reason) pair: binding the whole result
// struct would make the digest depend on formatting, and a peer that reformats
// its own copy would appear to hold different evidence.
func outcomePayload(outcome evidence.Outcome, reasonCode string) string {
	return string(outcome) + "\x00" + reasonCode
}

// verificationHint is the note included with a signed artifact telling the peer
// which stages to run. It is documentation, not authority: a peer that skips a
// stage is not thereby authorized (gr-xp-03).
const verificationHint = "verify transport, artifact integrity, producer identity, then your own authorization; a pass here is evidence only and never permits an effect"

// base64Length is exported for tests that assert the signature shape.
func signatureIsDetached(sig string) bool {
	if sig == "" {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(sig)
	return err == nil
}

// reasonCodeOf derives a stable machine-readable reason token from a verdict.
//
// The pipeline's own `Reason` is human prose that changes with the matched
// pattern; a peer needs a token it can branch on. The token is derived from the
// decision and the matched categories, both of which are closed sets, so a
// consumer sees a small vocabulary rather than an unbounded string.
func reasonCodeOf(result guardrails.InjectionResult) string {
	if result.Safe {
		return "clean"
	}
	if result.Decision != "" {
		return "injection_detected:" + result.Decision
	}
	if len(result.Categories) > 0 {
		return "injection_detected:" + result.Categories[0]
	}
	return "injection_detected"
}

// withEvidence returns the tool's original flat JSON object with one additive
// "evidence" key.
//
// The first version of this wiring wrapped the verdict as {"verdict": …,
// "evidence": …}. The existing tool tests caught it, and they were right: a
// client parsing the shipped flat result would decode every field as zero and
// see `safe=false` for benign text — a silent contract break introduced to add
// provenance. Additive-only is the rule: an existing consumer keeps working,
// and a new consumer reads the evidence block.
func withEvidence(result any, block any) (map[string]any, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var merged map[string]any
	if err := json.Unmarshal(raw, &merged); err != nil {
		return nil, err
	}
	merged["evidence"] = block
	return merged, nil
}

// unsignedEvidence is the explicit statement that an artifact carries no
// provenance. It is emitted rather than omitted so a consumer tests a field
// instead of inferring signing from the presence of a signature.
func unsignedEvidence() map[string]any {
	return map[string]any{
		"signed": false,
		"reason": "no instance signing key configured (EVIDENCE_PRIVATE_KEY); this verdict has no provenance",
		"hint":   verificationHint,
	}
}
