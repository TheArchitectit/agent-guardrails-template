// Package guardrails — Spec 14 (optional-oap-guardrail-checker).
//
// This file implements the portable, observe-only OAP Guardrail Checker
// boundary (gr-oap-01..gr-oap-07). The checker evaluates prompt/output/tool
// content against live Guardrails controls under an asserted, host-authenticated
// context and emits a signed or unsigned cross-product evidence envelope
// without ever mutating host roles, grants, policies, exceptions, adoption
// stages, or runtime effects.
package guardrails

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/controlplane"
	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
)

// ErrUnsupportedPolicyMutation is returned when a caller attempts to ask the
// guardrail checker to mutate a host role, grant, policy, exception, adoption
// stage, or runtime effect (Spec 14 gr-oap-04).
var ErrUnsupportedPolicyMutation = errors.New("oap guardrail checker is observe-only: cross-product policy, role, grant, exception, adoption-stage, or effect mutation is unsupported (gr-oap-04)")

// ErrInvalidOAPContext is returned when the asserted host context is missing
// mandatory tenant, principal, or action bindings (Spec 14 gr-oap-01).
var ErrInvalidOAPContext = errors.New("oap guardrail checker requires asserted host tenant, principal, and action context (gr-oap-01)")

// MaxOAPDiagnosticLen bounds every diagnostic detail string in OAP checker
// responses (Spec 14 gr-oap-06).
const MaxOAPDiagnosticLen = 256

// OAPDigestPrivacyNotice is the explicit privacy disclosure required by Spec 14
// gr-oap-06: SHA-256 digests bind evidence to a specific input/context and are
// NOT anonymization for low-entropy or guessable values.
const OAPDigestPrivacyNotice = "input_digest and context_digest are cryptographic bindings for integrity verification, not anonymization; low-entropy inputs remain guessable."

// OAPHostContext is the typed, host-authenticated execution context asserted by
// the calling host (e.g., OpenAgentPlatform or rad-gateway). Per gr-oap-01, this
// context is read-only input from the host's own security authority and is
// NEVER derived or overridden from prompt, tool, or model content.
type OAPHostContext struct {
	TenantID         string   `json:"tenant_id"`
	PrincipalID      string   `json:"principal_id"`
	SessionID        string   `json:"session_id,omitempty"`
	Action           string   `json:"action"`
	Resource         string   `json:"resource,omitempty"`
	AssertedRoles    []string `json:"asserted_roles,omitempty"`
	HostDecision     string   `json:"host_decision,omitempty"` // "allow" | "deny" | ""
	TransportSecured bool     `json:"transport_secured"`
}

// Validate checks that the host context carries the minimum asserted bindings.
func (c OAPHostContext) Validate() error {
	if strings.TrimSpace(c.TenantID) == "" {
		return fmt.Errorf("%w: missing tenant_id", ErrInvalidOAPContext)
	}
	if strings.TrimSpace(c.PrincipalID) == "" {
		return fmt.Errorf("%w: missing principal_id", ErrInvalidOAPContext)
	}
	if strings.TrimSpace(c.Action) == "" {
		return fmt.Errorf("%w: missing action", ErrInvalidOAPContext)
	}
	return nil
}

// Digest computes a deterministic domain-separated SHA-256 binding over the
// asserted host context.
func (c OAPHostContext) Digest() string {
	type canonicalCtx struct {
		TenantID      string   `json:"tenant_id"`
		PrincipalID   string   `json:"principal_id"`
		SessionID     string   `json:"session_id,omitempty"`
		Action        string   `json:"action"`
		Resource      string   `json:"resource,omitempty"`
		AssertedRoles []string `json:"asserted_roles,omitempty"`
		HostDecision  string   `json:"host_decision,omitempty"`
	}
	raw, _ := json.Marshal(canonicalCtx{
		TenantID:      strings.TrimSpace(c.TenantID),
		PrincipalID:   strings.TrimSpace(c.PrincipalID),
		SessionID:     strings.TrimSpace(c.SessionID),
		Action:        strings.TrimSpace(c.Action),
		Resource:      strings.TrimSpace(c.Resource),
		AssertedRoles: c.AssertedRoles,
		HostDecision:  strings.ToLower(strings.TrimSpace(c.HostDecision)),
	})
	sum := sha256.Sum256(append([]byte("guardrails.oap.context.v1\x00"), raw...))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// OAPMutationAttempt represents any forbidden attempt to use the checker as a
// policy/role/grant/exception/adoption-stage/effect mutation surface (gr-oap-04).
type OAPMutationAttempt struct {
	MutateRole          string `json:"mutate_role,omitempty"`
	MutateGrant         string `json:"mutate_grant,omitempty"`
	MutatePolicy        string `json:"mutate_policy,omitempty"`
	MutateException     string `json:"mutate_exception,omitempty"`
	MutateAdoptionStage string `json:"mutate_adoption_stage,omitempty"`
	AuthorizeEffect     bool   `json:"authorize_effect,omitempty"`
}

func (m *OAPMutationAttempt) hasAny() bool {
	if m == nil {
		return false
	}
	return strings.TrimSpace(m.MutateRole) != "" ||
		strings.TrimSpace(m.MutateGrant) != "" ||
		strings.TrimSpace(m.MutatePolicy) != "" ||
		strings.TrimSpace(m.MutateException) != "" ||
		strings.TrimSpace(m.MutateAdoptionStage) != "" ||
		m.AuthorizeEffect
}

// OAPCheckRequest is the input to the OAP Guardrail Checker.
type OAPCheckRequest struct {
	EvaluationID      string              `json:"evaluation_id"`
	Nonce             string              `json:"nonce"`
	IdempotencyKey    string              `json:"idempotency_key,omitempty"`
	HostContext       OAPHostContext      `json:"host_context"`
	Content           string              `json:"content"`
	Source            Source              `json:"source"`
	Direction         ContentDirection    `json:"direction"`
	RequiredChecks    []string            `json:"required_checks,omitempty"`
	RequestedMutation *OAPMutationAttempt `json:"requested_mutation,omitempty"`
}

// OAPCheckItem is one individual control evaluation inside an OAPCheckResult.
type OAPCheckItem struct {
	CheckID    string           `json:"check_id"`
	Mandatory  bool             `json:"mandatory"`
	Outcome    evidence.Outcome `json:"outcome"`
	ReasonCode string           `json:"reason_code,omitempty"`
	Diagnostic string           `json:"diagnostic,omitempty"`
}

// OAPCheckResult is the observe-only guardrail evaluation returned to the host.
//
// AuthorizesEffect is unconditionally false (gr-oap-02): a guardrail PASS is
// evidence that the evaluated checks passed on the input, never permission to
// execute the action when the host security authority denies it.
type OAPCheckResult struct {
	EvaluationID      string              `json:"evaluation_id"`
	Outcome           evidence.Outcome    `json:"outcome"`
	ReasonCode        string              `json:"reason_code"`
	AuthorizesEffect  bool                `json:"authorizes_effect"`
	HostDecision      string              `json:"host_decision"`
	EffectiveAllow    bool                `json:"effective_allow"`
	AssertedTenant    string              `json:"asserted_tenant"`
	AssertedPrincipal string              `json:"asserted_principal"`
	ContextDigest     string              `json:"context_digest"`
	InputDigest       string              `json:"input_digest"`
	Checks            []OAPCheckItem      `json:"checks"`
	NotEvaluated      []string            `json:"not_evaluated"`
	ContestedClaims   []string            `json:"contested_claims,omitempty"`
	PrivacyNotice     string              `json:"privacy_notice"`
	Envelope          *evidence.Envelope  `json:"envelope,omitempty"`
	DetachedSignature string              `json:"detached_signature,omitempty"`
	Signed            bool                `json:"signed"`
}

// OAPChecker evaluates OAPCheckRequests against a live Guardrails Engine.
type OAPChecker struct {
	engine   *Engine
	instance *evidence.Instance
	audience string
	ttl      time.Duration
	now      func() time.Time
}

// NewOAPChecker constructs an observe-only OAP Guardrail Checker.
func NewOAPChecker(engine *Engine, instance *evidence.Instance, audience string, ttl time.Duration) *OAPChecker {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if strings.TrimSpace(audience) == "" {
		audience = "openagentplatform"
	}
	return &OAPChecker{
		engine:   engine,
		instance: instance,
		audience: audience,
		ttl:      ttl,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// boundOAPDiagnostic scrubs secrets and truncates diagnostic strings (gr-oap-06).
func boundOAPDiagnostic(s string) string {
	redacted := controlplane.RedactDetail(strings.TrimSpace(s))
	if len(redacted) > MaxOAPDiagnosticLen {
		return redacted[:MaxOAPDiagnosticLen]
	}
	return redacted
}

// detectContentAuthorityInjection detects untrusted content attempting to assert
// tenant, role, principal, or authorization authority (gr-oap-01).
func detectContentAuthorityInjection(content string, host OAPHostContext) []string {
	lower := strings.ToLower(content)
	var claims []string
	markers := []struct {
		needle string
		label  string
	}{
		{"\"tenant_id\"", "content_claims_tenant_id"},
		{"\"principal_id\"", "content_claims_principal_id"},
		{"\"role\":\"admin\"", "content_claims_admin_role"},
		{"\"role\": \"admin\"", "content_claims_admin_role"},
		{"i am the admin", "content_claims_admin_identity"},
		{"authorize me", "content_claims_self_authorization"},
		{"override host policy", "content_claims_policy_override"},
		{"grant_role", "content_claims_role_grant"},
	}
	for _, m := range markers {
		if strings.Contains(lower, m.needle) {
			claims = append(claims, m.label)
		}
	}
	_ = host
	return claims
}

// Check evaluates req on the request path and returns an observe-only
// OAPCheckResult (gr-oap-01..gr-oap-07).
func (c *OAPChecker) Check(ctx context.Context, req OAPCheckRequest) (*OAPCheckResult, error) {
	// gr-oap-04: refuse any cross-product mutation request up front.
	if req.RequestedMutation.hasAny() {
		return nil, ErrUnsupportedPolicyMutation
	}
	// gr-oap-01: require asserted host context; never infer from content.
	if err := req.HostContext.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.EvaluationID) == "" {
		return nil, errors.New("oap guardrail checker requires evaluation_id")
	}
	if strings.TrimSpace(req.Nonce) == "" {
		return nil, errors.New("oap guardrail checker requires nonce")
	}

	// Preserve immutable copies of the host-asserted tenant and principal so
	// nothing in content evaluation can mutate them (gr-oap-01).
	assertedTenant := strings.TrimSpace(req.HostContext.TenantID)
	assertedPrincipal := strings.TrimSpace(req.HostContext.PrincipalID)
	ctxDigest := req.HostContext.Digest()
	inSum := sha256.Sum256(append([]byte("guardrails.oap.input.v1\x00"), []byte(req.Content)...))
	inDigest := "sha256:" + hex.EncodeToString(inSum[:])

	contested := detectContentAuthorityInjection(req.Content, req.HostContext)

	required := req.RequiredChecks
	if len(required) == 0 {
		required = []string{"injection_detection", "content_filter"}
	}

	var items []OAPCheckItem
	var notEvaluated []string

	// Always list cross-product surfaces that this observe-only checker does
	// not evaluate (gr-oap-06).
	notEvaluated = append(notEvaluated,
		"host_rbac_authorization",
		"host_policy_mutation",
		"sandbox_command_execution",
	)

	overall := evidence.OutcomePass
	overallReason := evidence.ReasonOK

	promoteOutcome := func(candidate evidence.Outcome, reason string) {
		if candidate == evidence.OutcomePass {
			return
		}
		// Precedence: block > error > inconclusive > skipped > pass.
		rank := func(o evidence.Outcome) int {
			switch o {
			case evidence.OutcomeBlock:
				return 5
			case evidence.OutcomeError:
				return 4
			case evidence.OutcomeInconclusive:
				return 3
			case evidence.OutcomeSkipped:
				return 2
			default:
				return 1
			}
		}
		if rank(candidate) > rank(overall) {
			overall = candidate
			overallReason = reason
		}
	}

	if err := ctx.Err(); err != nil {
		promoteOutcome(evidence.OutcomeError, "context_cancelled")
		items = append(items, OAPCheckItem{
			CheckID:    "request_context",
			Mandatory:  true,
			Outcome:    evidence.OutcomeError,
			ReasonCode: "context_cancelled",
			Diagnostic: boundOAPDiagnostic(err.Error()),
		})
	} else if c == nil || c.engine == nil {
		// gr-oap-03 / gr-oap-05: unwired engine must never report PASS.
		for _, chk := range required {
			notEvaluated = append(notEvaluated, chk)
			items = append(items, OAPCheckItem{
				CheckID:    chk,
				Mandatory:  true,
				Outcome:    evidence.OutcomeInconclusive,
				ReasonCode: "engine_unavailable",
				Diagnostic: "guardrail engine is not wired on this instance",
			})
		}
		promoteOutcome(evidence.OutcomeInconclusive, "engine_unavailable")
	} else {
		for _, chk := range required {
			switch chk {
			case "injection_detection":
				if c.engine.pipeline == nil {
					notEvaluated = append(notEvaluated, chk)
					items = append(items, OAPCheckItem{
						CheckID:    chk,
						Mandatory:  true,
						Outcome:    evidence.OutcomeInconclusive,
						ReasonCode: "injection_pipeline_unavailable",
						Diagnostic: "injection pipeline is not wired",
					})
					promoteOutcome(evidence.OutcomeInconclusive, "injection_pipeline_unavailable")
					continue
				}
				src := req.Source
				if src == "" {
					src = SourceUser
				}
				inj := c.engine.pipeline.Detect(ctx, req.Content, src)
				if !inj.Safe {
					items = append(items, OAPCheckItem{
						CheckID:    chk,
						Mandatory:  true,
						Outcome:    evidence.OutcomeBlock,
						ReasonCode: "injection_detected",
						Diagnostic: boundOAPDiagnostic(fmt.Sprintf("layer=%s confidence=%.2f", inj.Layer, inj.Confidence)),
					})
					promoteOutcome(evidence.OutcomeBlock, "injection_detected")
				} else {
					items = append(items, OAPCheckItem{
						CheckID:    chk,
						Mandatory:  true,
						Outcome:    evidence.OutcomePass,
						ReasonCode: evidence.ReasonOK,
						Diagnostic: boundOAPDiagnostic(fmt.Sprintf("layer=%s confidence=%.2f", inj.Layer, inj.Confidence)),
					})
				}

			case "content_filter":
				if c.engine.filter == nil || c.engine.filter.BackendCount() == 0 {
					// gr-oap-03 / gr-oap-05: no live classifier backend -> Honest non-PASS.
					notEvaluated = append(notEvaluated, chk)
					items = append(items, OAPCheckItem{
						CheckID:    chk,
						Mandatory:  true,
						Outcome:    evidence.OutcomeInconclusive,
						ReasonCode: "content_classifier_unavailable",
						Diagnostic: "no semantic content classifier backend is wired",
					})
					promoteOutcome(evidence.OutcomeInconclusive, "content_classifier_unavailable")
					continue
				}
				dir := req.Direction
				if dir == "" {
					dir = DirectionInput
				}
				cfRes, err := c.engine.filter.Classify(ctx, req.Content, dir)
				if err != nil {
					items = append(items, OAPCheckItem{
						CheckID:    chk,
						Mandatory:  true,
						Outcome:    evidence.OutcomeError,
						ReasonCode: "content_filter_error",
						Diagnostic: boundOAPDiagnostic(err.Error()),
					})
					promoteOutcome(evidence.OutcomeError, "content_filter_error")
				} else if !cfRes.Safe || cfRes.IsBlocked() {
					items = append(items, OAPCheckItem{
						CheckID:    chk,
						Mandatory:  true,
						Outcome:    evidence.OutcomeBlock,
						ReasonCode: "content_policy_blocked",
						Diagnostic: boundOAPDiagnostic(fmt.Sprintf("backend=%s risk=%.2f", cfRes.Backend, cfRes.OverallRisk)),
					})
					promoteOutcome(evidence.OutcomeBlock, "content_policy_blocked")
				} else {
					items = append(items, OAPCheckItem{
						CheckID:    chk,
						Mandatory:  true,
						Outcome:    evidence.OutcomePass,
						ReasonCode: evidence.ReasonOK,
						Diagnostic: boundOAPDiagnostic(fmt.Sprintf("backend=%s risk=%.2f", cfRes.Backend, cfRes.OverallRisk)),
					})
				}

			default:
				// Unknown/unsupported mandatory check must never silently pass (gr-oap-05).
				notEvaluated = append(notEvaluated, chk)
				items = append(items, OAPCheckItem{
					CheckID:    chk,
					Mandatory:  true,
					Outcome:    evidence.OutcomeSkipped,
					ReasonCode: "check_not_supported",
					Diagnostic: boundOAPDiagnostic(fmt.Sprintf("check %q is not implemented", chk)),
				})
				promoteOutcome(evidence.OutcomeSkipped, "check_not_supported")
			}
		}
	}

	// gr-oap-02: Guardrail outcome is separate from host authorization.
	// AuthorizesEffect is ALWAYS false, and EffectiveAllow is true ONLY if the
	// host explicitly decided "allow" AND the guardrail outcome is terminal PASS.
	hostDecision := strings.ToLower(strings.TrimSpace(req.HostContext.HostDecision))
	if hostDecision == "" {
		hostDecision = "unspecified"
	}
	effectiveAllow := hostDecision == "allow" && overall.Terminal()

	res := &OAPCheckResult{
		EvaluationID:      req.EvaluationID,
		Outcome:           overall,
		ReasonCode:        overallReason,
		AuthorizesEffect:  false,
		HostDecision:      hostDecision,
		EffectiveAllow:    effectiveAllow,
		AssertedTenant:    assertedTenant,
		AssertedPrincipal: assertedPrincipal,
		ContextDigest:     ctxDigest,
		InputDigest:       inDigest,
		Checks:            items,
		NotEvaluated:      notEvaluated,
		ContestedClaims:   contested,
		PrivacyNotice:     OAPDigestPrivacyNotice,
	}

	// Build payload bytes for the cross-product evidence envelope (Spec 15).
	payloadBytes, err := json.Marshal(struct {
		EvaluationID  string         `json:"evaluation_id"`
		ContextDigest string         `json:"context_digest"`
		InputDigest   string         `json:"input_digest"`
		Outcome       string         `json:"outcome"`
		Checks        []OAPCheckItem `json:"checks"`
		NotEvaluated  []string       `json:"not_evaluated"`
	}{
		EvaluationID:  res.EvaluationID,
		ContextDigest: res.ContextDigest,
		InputDigest:   res.InputDigest,
		Outcome:       string(res.Outcome),
		Checks:        res.Checks,
		NotEvaluated:  res.NotEvaluated,
	})
	if err != nil {
		return nil, err
	}

	now := c.now()
	idem := strings.TrimSpace(req.IdempotencyKey)
	if idem == "" {
		idem = req.EvaluationID + ":" + req.Nonce
	}
	reasonForEnv := res.ReasonCode
	if res.Outcome == evidence.OutcomePass {
		reasonForEnv = "all_required_checks_passed"
	}
	env := &evidence.Envelope{
		ContractVersion: evidence.ContractVersion,
		Direction:       evidence.DirectionOutbound,
		Audience:        c.audience,
		Tenant:          assertedTenant,
		SubjectDigest:   ctxDigest,
		PolicyDigest:    inDigest,
		ComponentID:     "oap-guardrail-checker",
		ComponentVer:    "v1",
		RequestID:       req.EvaluationID,
		EvaluationID:    req.EvaluationID,
		IdempotencyKey:  idem,
		Nonce:           req.Nonce,
		IssuedAt:        now,
		ExpiresAt:       now.Add(c.ttl),
		PayloadDigest:   evidence.PayloadDigest(payloadBytes),
		ManifestDigest:  ctxDigest,
		Outcome:         res.Outcome,
		ReasonCode:      reasonForEnv,
	}
	if c.instance != nil {
		env.ProducerID = c.instance.ProducerID()
		env.KeyID = c.instance.KeyID()
		sig, err := c.instance.Sign(env)
		if err != nil {
			return nil, fmt.Errorf("sign oap evidence envelope: %w", err)
		}
		res.DetachedSignature = sig
		res.Signed = true
	}
	res.Envelope = env
	return res, nil
}
