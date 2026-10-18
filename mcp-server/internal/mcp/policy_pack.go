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
	Status          string                       `json:"status"`
	Source          string                       `json:"source,omitempty"`
	EffectiveDigest string                       `json:"effective_digest,omitempty"`
	ResolverVersion string                       `json:"resolver_version,omitempty"`
	Packs           []guardrails.PackRecord      `json:"packs,omitempty"`
	Rules           []guardrails.PackRule        `json:"rules,omitempty"`
	Attribution     map[string]policyAttribution `json:"attribution,omitempty"`
	Error           string                       `json:"error,omitempty"`
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
}

// policyPackStatusSnapshot builds the diagnostic payload from the live manager.
func (s *MCPServer) policyPackStatusSnapshot(err error) policyPackStatus {
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
	status.Rules = active.Rules()

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
