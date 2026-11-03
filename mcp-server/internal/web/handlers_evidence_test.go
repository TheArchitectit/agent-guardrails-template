package web

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
)

func TestVerifyEvidenceEndpoint_FourStageContract(t *testing.T) {
	now := time.Now().UTC()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	inst, err := evidence.NewInstance("devgate-peer", priv)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}

	tmpDir := t.TempDir()
	trustPath := filepath.Join(tmpDir, "trust.json")
	trustDoc := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"producer_id":       "devgate-peer",
				"public_key_base64": base64.StdEncoding.EncodeToString(inst.PublicKey()),
				"audience":          "guardrails",
				"direction":         "inbound",
				"not_before":        now.Add(-time.Hour).Format(time.RFC3339),
				"not_after":         now.Add(time.Hour).Format(time.RFC3339),
			},
		},
	}
	trustBytes, _ := json.Marshal(trustDoc)
	if err := os.WriteFile(trustPath, trustBytes, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := &config.Config{
		EvidenceTrustStoreFile: trustPath,
	}
	verifier, err := buildEvidenceVerifier(cfg)
	if err != nil {
		t.Fatalf("buildEvidenceVerifier: %v", err)
	}

	s := &Server{
		cfg:              cfg,
		evidenceVerifier: verifier,
	}
	e := echo.New()

	payload := []byte(`{"gate":"spec_audit","passed":true}`)
	env := &evidence.Envelope{
		ContractVersion: evidence.ContractVersion,
		Direction:       evidence.DirectionInbound,
		ProducerID:      inst.ProducerID(),
		KeyID:           inst.KeyID(),
		Audience:        "guardrails",
		Tenant:          "tenant-a",
		SubjectDigest:   "sha256:subject-1",
		PolicyDigest:    "sha256:policy-1",
		ComponentID:     "devgate-runner",
		ComponentVer:    "v1",
		RequestID:       "req-web-1",
		EvaluationID:    "eval-web-1",
		IdempotencyKey:  "idem-web-1",
		Nonce:           "nonce-web-1",
		IssuedAt:        now.Add(-time.Minute),
		ExpiresAt:       now.Add(5 * time.Minute),
		PayloadDigest:   evidence.PayloadDigest(payload),
		ManifestDigest:  "sha256:manifest-1",
		Outcome:         evidence.OutcomePass,
		ReasonCode:      "all_checks_passed",
	}
	sig, err := inst.Sign(env)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	bodyBytes, _ := json.Marshal(EvidenceVerifyRequest{
		Envelope:          env,
		DetachedSignature: sig,
		Payload:           payload,
	})

	// 1. Authenticated transport + valid signature -> 200 Verified=true.
	req := httptest.NewRequest(http.MethodPost, "/api/evidence/verify", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("api_key_type", "registered")

	if err := s.verifyEvidence(c); err != nil {
		t.Fatalf("verifyEvidence: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var verdict evidence.Verdict
	if err := json.Unmarshal(rec.Body.Bytes(), &verdict); err != nil {
		t.Fatalf("Unmarshal verdict: %v", err)
	}
	if !verdict.Verified || verdict.ProducerID != "devgate-peer" {
		t.Fatalf("expected verified verdict from devgate-peer, got %+v", verdict)
	}

	// 2. Replay of the same envelope -> 422 ReasonReplayed.
	req2 := httptest.NewRequest(http.MethodPost, "/api/evidence/verify", bytes.NewReader(bodyBytes))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.Set("api_key_type", "registered")

	if err := s.verifyEvidence(c2); err != nil {
		t.Fatalf("verifyEvidence replay: %v", err)
	}
	if rec2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on replay, got %d", rec2.Code)
	}

	// 3. Unconfigured verifier fails closed with 503.
	sUnconf := &Server{cfg: &config.Config{}}
	req3 := httptest.NewRequest(http.MethodPost, "/api/evidence/verify", bytes.NewReader(bodyBytes))
	req3.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec3 := httptest.NewRecorder()
	c3 := e.NewContext(req3, rec3)
	c3.Set("api_key_type", "registered")
	if err := sUnconf.verifyEvidence(c3); err != nil {
		t.Fatalf("verifyEvidence unconfigured: %v", err)
	}
	if rec3.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when trust store not configured, got %d", rec3.Code)
	}
}
