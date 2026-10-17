package mcp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
)

// TestRequireBearerWithPolicyLegacyRemovalSwitch covers Spec 11 Sec. 4.5.2 /
// acceptance criterion 8 on the MCP surface: once the configured
// legacy-acceptance deadline has passed, the legacy shared key is refused,
// while a registered credential is unaffected. A registered credential keeps
// working after the deadline because only the legacy path is retired.
func TestRequireBearerWithPolicyLegacyRemovalSwitch(t *testing.T) {
	records := []auth.Record{{
		CredentialID: "cred-1",
		PrincipalID:  "principal-alpha",
		Scopes:       []string{auth.ScopeMCPRead},
		Role:         auth.RoleDeveloper,
		Resources:    []string{"*"},
		IssuedAt:     time.Now().Add(-time.Hour),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Verifier:     auth.Digest(fullPathVerifierKey, "registered-secret"),
	}}
	raw, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	registry, err := auth.LoadFromSources(string(raw), "", fullPathVerifierKey)
	if err != nil {
		t.Fatalf("LoadFromSources: %v", err)
	}

	ok := okHandler()
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	cases := []struct {
		name             string
		header           string
		legacyConfigured bool
		deadline         time.Time
		want             int
	}{
		{"legacy key before deadline accepted", "Bearer secret-key", true, future, 200},
		{"legacy key after deadline refused", "Bearer secret-key", true, past, 401},
		{"registered credential after deadline unaffected", "Bearer registered-secret", true, past, 200},
		{"unconfigured switch never retires legacy key", "Bearer secret-key", false, time.Time{}, 200},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			req.Header.Set("Authorization", tc.header)
			rec := httptest.NewRecorder()
			requireBearerWithPolicy("secret-key", registry, nil, tc.legacyConfigured, tc.deadline, ok).ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("got %d want %d", rec.Code, tc.want)
			}
		})
	}
}
