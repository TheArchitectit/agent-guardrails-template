package web

import (
	"net/http"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// TestAPIKeyAuth_LegacyAcceptanceRemovalSwitch covers Spec 11 §4.5.2 /
// acceptance criterion 8 on the REST surface: the legacy-acceptance removal
// switch is testable. Before the deadline the legacy key keeps its surface;
// once the deadline has passed the legacy key is refused even though nothing
// else changed. A malformed deadline fails closed rather than passing.
func TestAPIKeyAuth_LegacyAcceptanceRemovalSwitch(t *testing.T) {
	t.Run("before deadline legacy key accepted", func(t *testing.T) {
		cfg := &config.Config{
			MCPAPIKey:             "legacy-mcp-key",
			IDEAPIKey:             "legacy-ide-key",
			LegacyAcceptanceUntil: "2999-01-01T00:00:00Z",
		}
		code, _ := runAuth(cfg, http.MethodPost, "/api/rules", "legacy-mcp-key")
		if code != http.StatusOK {
			t.Fatalf("legacy key before deadline: got %d want 200", code)
		}
	})

	t.Run("after deadline legacy key refused", func(t *testing.T) {
		cfg := &config.Config{
			MCPAPIKey:             "legacy-mcp-key",
			IDEAPIKey:             "legacy-ide-key",
			LegacyAcceptanceUntil: "2000-01-01T00:00:00Z",
		}
		code, _ := runAuth(cfg, http.MethodPost, "/api/rules", "legacy-mcp-key")
		if code != http.StatusForbidden {
			t.Fatalf("legacy key after deadline: got %d want 403", code)
		}
	})

	t.Run("malformed deadline fails closed", func(t *testing.T) {
		cfg := &config.Config{
			MCPAPIKey:             "legacy-mcp-key",
			IDEAPIKey:             "legacy-ide-key",
			LegacyAcceptanceUntil: "not-a-time",
		}
		code, _ := runAuth(cfg, http.MethodPost, "/api/rules", "legacy-mcp-key")
		if code != http.StatusServiceUnavailable {
			t.Fatalf("malformed deadline: got %d want 503", code)
		}
	})

	t.Run("unconfigured switch never retires legacy keys", func(t *testing.T) {
		cfg := &config.Config{MCPAPIKey: "legacy-mcp-key", IDEAPIKey: "legacy-ide-key"}
		code, _ := runAuth(cfg, http.MethodPost, "/api/rules", "legacy-mcp-key")
		if code != http.StatusOK {
			t.Fatalf("unconfigured switch: got %d want 200", code)
		}
	})

	t.Run("removal switch does not affect a registered credential", func(t *testing.T) {
		cfg := registryConfig(t, "registered-secret", "principal-alpha")
		// Deadline already passed: only the legacy path is retired.
		cfg.LegacyAcceptanceUntil = "2000-01-01T00:00:00Z"
		code, principal := runAuth(cfg, http.MethodPost, "/api/rules", "registered-secret")
		if code != http.StatusOK {
			t.Fatalf("registered credential after deadline: got %d want 200", code)
		}
		if principal != "principal-alpha" {
			t.Fatalf("registered principal=%q want principal-alpha", principal)
		}
	})
}
