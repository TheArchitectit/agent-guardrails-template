package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
)

// EvidenceVerifyRequest is the JSON request body for POST /api/evidence/verify
// (Spec 15 gr-xp-02..gr-xp-06).
type EvidenceVerifyRequest struct {
	Envelope          *evidence.Envelope `json:"envelope"`
	DetachedSignature string             `json:"detached_signature"`
	Payload           json.RawMessage    `json:"payload"`
}

func buildEvidenceVerifier(cfg *config.Config) (*evidence.Verifier, error) {
	if cfg == nil {
		return nil, nil
	}
	trustFile := strings.TrimSpace(cfg.EvidenceTrustStoreFile)
	revFile := strings.TrimSpace(cfg.EvidenceRevocationFile)
	if trustFile == "" && revFile == "" {
		return nil, nil
	}
	if trustFile == "" {
		return nil, fmt.Errorf("EVIDENCE_TRUST_STORE_FILE is required when EVIDENCE_REVOCATION_FILE is set")
	}
	store, err := evidence.LoadTrustStoreFile(trustFile)
	if err != nil {
		return nil, err
	}
	if revFile != "" {
		snap, err := evidence.LoadRevocationSnapshotFile(revFile)
		if err != nil {
			return nil, err
		}
		store, err = store.ApplyRevocationSnapshot(snap, time.Now().UTC())
		if err != nil {
			return nil, err
		}
	}
	return evidence.NewVerifier(store, nil, nil)
}

// verifyEvidence handles POST /api/evidence/verify (Spec 15 gr-xp-02..gr-xp-06).
//
// It runs all four stages of the cross-product verification contract:
//  1. transport authentication (verified bearer caller on the protected route);
//  2. artifact integrity (canonical shape, detached Ed25519 signature, freshness,
//     payload digest, replay);
//  3. producer identity (out-of-band trust record, direction, audience,
//     revocation);
//  4. receiving-policy authorization (caller tenant/resource alignment).
func (s *Server) verifyEvidence(c echo.Context) error {
	if s.evidenceVerifierErr != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"verified":    false,
			"stage":       evidence.StageProducer,
			"reason_code": "trust_store_unavailable",
			"reason":      "inbound evidence trust store failed to load",
		})
	}
	if s.evidenceVerifier == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"verified":    false,
			"stage":       evidence.StageProducer,
			"reason_code": "trust_store_not_configured",
			"reason":      "inbound evidence trust store is not configured on this instance",
		})
	}

	var req EvidenceVerifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"verified":    false,
			"stage":       evidence.StageIntegrity,
			"reason_code": evidence.ReasonIntegrityShape,
			"reason":      "invalid JSON request body",
		})
	}

	// Stage 1 — transport authentication: satisfied only when the request
	// passed APIKeyAuth middleware (which populates a non-empty api_key_type).
	caller := callerFromContext(c)
	keyType, _ := c.Get("api_key_type").(string)
	transportAuthed := keyType != ""

	// Stage 4 — receiving-policy authorization: if the authenticated caller is
	// scoped to specific resources/tenants, the envelope's tenant must be
	// permitted.
	policy := func(env *evidence.Envelope, _ string) error {
		if len(caller.Resources) > 0 && env != nil && strings.TrimSpace(env.Tenant) != "" {
			allowed := false
			for _, r := range caller.Resources {
				if r == "*" || r == env.Tenant {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("authenticated caller is not authorized for envelope tenant %q", env.Tenant)
			}
		}
		return nil
	}

	verdict := s.evidenceVerifier.Verify(req.Envelope, req.DetachedSignature, []byte(req.Payload), transportAuthed, policy)
	status := http.StatusOK
	if !verdict.Verified {
		status = http.StatusUnprocessableEntity
	}
	return c.JSON(status, verdict)
}
