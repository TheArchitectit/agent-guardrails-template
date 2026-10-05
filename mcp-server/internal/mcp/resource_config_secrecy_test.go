package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// secretMarkers are distinctive fake secrets planted in every secret-bearing
// config field. None of them may appear in guardrail://config output.
var secretMarkers = []string{
	"SECRET_MARKER_DB_PASSWORD_7c2e",
	"SECRET_MARKER_MCP_API_KEY_9a1f",
	"SECRET_MARKER_IDE_API_KEY_4b8d",
	"SECRET_MARKER_JWT_SECRET_6e0a",
	"SECRET_MARKER_VERIFIER_KEY_1f5c",
	"SECRET_MARKER_REDIS_PASSWORD_3d9b",
	"SECRET_MARKER_REGISTRY_JSON_8a2e",
	"SECRET_MARKER_TLS_KEY_PATH_5c1d",
	"SECRET_MARKER_REGISTRY_FILE_2b4c",
	"SECRET_MARKER_VERIFIER_KEY_FILE_7a9e",
}

func secretConfig() *config.Config {
	return &config.Config{
		SchemaVersion:             "1.0",
		MCPPort:                   8080,
		LogLevel:                  "info",
		DBPassword:                secretMarkers[0],
		MCPAPIKey:                 secretMarkers[1],
		IDEAPIKey:                 secretMarkers[2],
		JWTSecret:                 secretMarkers[3],
		CredentialVerifierKey:     secretMarkers[4],
		RedisPassword:             secretMarkers[5],
		CredentialRegistryJSON:    secretMarkers[6],
		TLSKeyPath:                secretMarkers[7],
		TLSCertPath:               "/etc/ssl/" + secretMarkers[7],
		CredentialRegistryFile:    "/run/secrets/" + secretMarkers[8],
		CredentialVerifierKeyFile: "/run/secrets/" + secretMarkers[9],
	}
}

// TestGuardrailConfigOmitsSecrets reads guardrail://config and asserts no
// secret marker appears (S-A0 / R16-04).
func TestGuardrailConfigOmitsSecrets(t *testing.T) {
	s := &MCPServer{config: secretConfig()}

	contents := s.readConfigResourceContents("guardrail://config")
	if len(contents) != 1 {
		t.Fatalf("expected 1 resource contents, got %d", len(contents))
	}
	text, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("expected TextResourceContents, got %T", contents[0])
	}

	for _, marker := range secretMarkers {
		if strings.Contains(text.Text, marker) {
			t.Fatalf("guardrail://config leaked secret marker %q", marker)
		}
	}

	// Also assert the structural field names of secrets are absent.
	for _, field := range []string{
		"DBPassword", "db_password",
		"MCPAPIKey", "mcp_api_key",
		"IDEAPIKey", "ide_api_key",
		"JWTSecret", "jwt_secret",
		"CredentialVerifierKey", "credential_verifier_key",
		"RedisPassword", "redis_password",
		"CredentialRegistryJSON", "credential_registry_json",
		"CredentialRegistryFile", "credential_registry_file",
		"CredentialVerifierKeyFile", "credential_verifier_key_file",
		"TLSKeyPath", "tls_key_path",
		"TLSCertPath", "tls_cert_path",
	} {
		if strings.Contains(text.Text, field) {
			t.Fatalf("guardrail://config leaked secret field name %q", field)
		}
	}

	// Positive control: the projection is real JSON and still useful.
	var view map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &view); err != nil {
		t.Fatalf("config projection is not JSON: %v", err)
	}
	if view["schema_version"] != "1.0" {
		t.Fatalf("schema_version = %v, want 1.0", view["schema_version"])
	}
}

// TestConfigPublicViewOmitsSecrets is a belt-and-braces unit check on the
// projection itself.
func TestConfigPublicViewOmitsSecrets(t *testing.T) {
	raw, err := json.Marshal(secretConfig().PublicView())
	if err != nil {
		t.Fatalf("marshal PublicView: %v", err)
	}
	for _, marker := range secretMarkers {
		if strings.Contains(string(raw), marker) {
			t.Fatalf("PublicView leaked secret marker %q", marker)
		}
	}
}
