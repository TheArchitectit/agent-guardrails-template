package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// Spec 07 R07-4.4.1 — the active policy set must be exposed "at startup and
// through a read-only diagnostic surface".
//
// Before this file the server had no runtime policy-pack consumer at all: the
// engine resolved policy from its own configured rule set, the effective digest
// was never exposed anywhere, and no enforcement decision could be attributed
// to a pack rule. The CI gate proved the *record* was honest; it could not make
// the running server obey the pack.
//
// The diagnostic added here is read-only and non-secret. It reports one of
// three states honestly rather than collapsing them:
//
//   - "not_configured": no pack directory was supplied. The server is not
//     enforcing a pack, and saying "active" would be a false claim.
//   - "active": a candidate resolved, validated against the checked-in lock, and
//     was swapped in atomically.
//   - "failed": a configured directory did not resolve. The previous set (if
//     any) stays in force; the error is reported so an operator can see why.

// policyPackStatus is the JSON shape of guardrail://policy-packs.
type policyPackStatus struct {
	Status          string                  `json:"status"`
	Source          string                  `json:"source,omitempty"`
	EffectiveDigest string                  `json:"effective_digest,omitempty"`
	ResolverVersion string                  `json:"resolver_version,omitempty"`
	Packs           []guardrails.PackRecord `json:"packs,omitempty"`
	Dependencies    []guardrails.PackEdge   `json:"dependencies,omitempty"`
	// Spec 07 R07-4.2 — the precedence decision, the per-tool composition and
	// the reviewed exceptions. Reported from the resolved snapshot, never
	// recomputed here: a surface that re-derives precedence can disagree with
	// the set the server is enforcing.
	RuleSetDigest string                       `json:"rule_set_digest,omitempty"`
	Applications  []guardrails.RuleApplication `json:"applications,omitempty"`
	Conflicts     []guardrails.RuleConflict    `json:"conflicts,omitempty"`
	Exceptions    []guardrails.Exception       `json:"exceptions,omitempty"`
	Rules         []guardrails.PackRule        `json:"rules,omitempty"`
	// Spec 07 R07-4.3.5 — one provenance row per pack: verified, pinned or
	// locally_modified. It is a separate field from the pack records because
	// "the digest matched" and "a trusted key vouched for these bytes" are
	// different facts and must not collapse into one.
	Trust []guardrails.PackTrust `json:"trust,omitempty"`
	// SignatureVerification reports whether a trusted key set was configured,
	// so `pinned` is not misread as "this deployment has no signature feature".
	SignatureVerification bool                         `json:"signature_verification"`
	Attribution           map[string]policyAttribution `json:"attribution,omitempty"`
	Error                 string                       `json:"error,omitempty"`
}

// policyAttribution names the pack and rule that govern a tool. An enforcement
// decision can then be traced to the exact rule that produced it (R07-4.4.3)
// without re-resolving anything.
type policyAttribution struct {
	PackDigest string `json:"pack_digest"`
	RuleID     string `json:"rule_id"`
	Pack       string `json:"pack"`
}

// SetPolicyPacks attaches the Spec 07 runtime policy-pack manager. A nil manager
// makes the diagnostic report "not_configured" and makes attribution empty;
// it never reads as an active policy set with no rules.
func (s *MCPServer) SetPolicyPacks(m *guardrails.PolicyPackManager) {
	s.policyPacks = m
	s.policyPacksErr = nil
}

// SetPolicyPacksError records a configured pack set that did not resolve, so
// guardrail://policy-packs reports "failed" WITH the reason instead of an
// inactive manager. The two are different conditions: one means nothing was
// configured, the other means something was rejected.
func (s *MCPServer) SetPolicyPacksError(err error) {
	s.policyPacksErr = err
}

// policyPackStatusSnapshot builds the diagnostic payload from the live manager.
func (s *MCPServer) policyPackStatusSnapshot(err error) policyPackStatus {
	if err == nil {
		err = s.policyPacksErr
	}
	if s.policyPacks == nil {
		status := policyPackStatus{Status: "not_configured"}
		if err != nil {
			status.Status = "failed"
			status.Error = err.Error()
		}
		return status
	}

	status := policyPackStatus{Source: s.policyPacks.Source()}
	active := s.policyPacks.Active()
	if !active.Active() {
		status.Status = "failed"
		if err != nil {
			status.Error = err.Error()
		} else {
			status.Error = "no policy set has been activated"
		}
		return status
	}

	status.Status = "active"
	status.EffectiveDigest = active.EffectiveDigest()
	status.ResolverVersion = active.ResolverVersion()
	status.Packs = active.Packs()
	// The resolved dependency graph (R07-4.1.4) is reported from the snapshot
	// resolution produced, never re-derived here: a surface that recomputes the
	// graph can disagree with the set the server is enforcing.
	status.Dependencies = active.Dependencies()
	status.Rules = active.Rules()
	status.Trust = active.TrustStates()
	status.SignatureVerification = s.policyPacks.VerificationEnabled()
	status.RuleSetDigest = active.RuleSetDigest()
	// Applications always carries one row per effective rule id, so "no
	// override happened" is visible as a row with weakened=false rather than as
	// a missing field. Conflicts and exceptions are omitempty because their
	// absence is the normal state of a single-pack tree; the rule-set digest
	// still proves which set they were absent from.
	status.Applications = active.Applications()
	status.Conflicts = active.Conflicts()
	status.Exceptions = active.Exceptions()

	attribution := map[string]policyAttribution{}
	for _, rule := range active.Rules() {
		if rule.Tool == "" {
			continue
		}
		digest, ruleID, ok := active.Attribution(rule.Tool)
		if !ok {
			continue
		}
		attribution[rule.Tool] = policyAttribution{
			PackDigest: digest,
			RuleID:     ruleID,
			Pack:       active.PackForRule(ruleID),
		}
	}
	status.Attribution = attribution
	return status
}

// readPolicyPacksResource renders guardrail://policy-packs.
func (s *MCPServer) readPolicyPacksResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	payload, err := json.MarshalIndent(s.policyPackStatusSnapshot(nil), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy pack status: %w", err)
	}
	return &mcp.ReadResourceResult{
		Contents: []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(payload),
			},
		},
	}, nil
}

// policyAttributionFields returns the audit fields that attribute an enforcement
// decision to the active policy set. An uncovered tool yields empty values, and
// the caller records that absence rather than borrowing a neighbouring rule.
func (s *MCPServer) policyAttributionFields(tool string) (digest, ruleID string, ok bool) {
	if s.policyPacks == nil {
		return "", "", false
	}
	return s.policyPacks.Active().Attribution(tool)
}

// RegisteredToolNames returns the set of tool names this server advertises.
// Startup uses it to refuse a policy rule naming a tool that does not exist —
// such a rule can never fire, so shipping it would be a policy that reads as
// enforcement and does nothing.
func (s *MCPServer) RegisteredToolNames() map[string]bool {
	tools := s.toolList()
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}
	return names
}
