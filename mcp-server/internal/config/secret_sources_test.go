package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/secretfile"
)

// writeSecretFile writes a secret file with the restricted mode the R17-06 rule
// requires and returns its path.
func writeSecretFile(t *testing.T, dir, name, value string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(value), secretfile.Mode); err != nil {
		t.Fatalf("write secret file %s: %v", name, err)
	}
	return path
}

// TestResolveSecretSourcesFromFiles proves a file-only deployment resolves every
// primary secret and then passes Validate — the deployment shape the previous
// `required` env tags made impossible.
func TestResolveSecretSourcesFromFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX secret-file mode rule is not enforced on Windows")
	}
	dir := t.TempDir()

	cfg := validMinimalConfig()
	// Nothing inline: every secret comes from a file.
	cfg.MCPAPIKey, cfg.IDEAPIKey, cfg.JWTSecret = "", "", ""
	cfg.DBPassword, cfg.RedisPassword = "", ""
	cfg.MCPAPIKeyFile = writeSecretFile(t, dir, "mcp.key", "AbCdEfGhIjKlMnOpQrStUvWxYz123456")
	cfg.IDEAPIKeyFile = writeSecretFile(t, dir, "ide.key", "ZyXwVuTsRqPoNmLkJiHgFeDcBa654321")
	cfg.JWTSecretFile = writeSecretFile(t, dir, "jwt.key", "abcdefghijklmnopqrstuvwxyz123456")
	cfg.DBPasswordFile = writeSecretFile(t, dir, "db.pass", "operator-db-pass-01")
	cfg.RedisPasswordFile = writeSecretFile(t, dir, "redis.pass", "operator-redis-pass-01\n")

	if err := cfg.ResolveSecretSources(); err != nil {
		t.Fatalf("ResolveSecretSources: %v", err)
	}

	// The trailing newline of a mounted secret must be trimmed, not carried.
	if cfg.RedisPassword != "operator-redis-pass-01" {
		t.Fatalf("RedisPassword = %q, want the trimmed file value", cfg.RedisPassword)
	}
	for name, got := range map[string]string{
		"MCPAPIKey":     cfg.MCPAPIKey,
		"IDEAPIKey":     cfg.IDEAPIKey,
		"JWTSecret":     cfg.JWTSecret,
		"DBPassword":    cfg.DBPassword,
		"RedisPassword": cfg.RedisPassword,
	} {
		if got == "" {
			t.Fatalf("%s was not resolved from its secret file", name)
		}
	}

	// A file-only deployment must pass the startup gate end to end.
	if err := cfg.Validate(); err != nil {
		t.Fatalf("file-only config must pass Validate, got %v", err)
	}
}

// TestResolveSecretSourcesInlineWins proves inline environment material keeps
// precedence, so an existing environment-only deployment is unaffected.
func TestResolveSecretSourcesInlineWins(t *testing.T) {
	dir := t.TempDir()
	cfg := validMinimalConfig()
	cfg.MCPAPIKeyFile = writeSecretFile(t, dir, "mcp.key", "file-supplied-value-0000000000")

	if err := cfg.ResolveSecretSources(); err != nil {
		t.Fatalf("ResolveSecretSources: %v", err)
	}
	if cfg.MCPAPIKey != "AbCdEfGhIjKlMnOpQrStUvWxYz123456" {
		t.Fatalf("inline MCP_API_KEY must win over the file, got %q", cfg.MCPAPIKey)
	}
}

// TestResolveSecretSourcesFailsClosed covers the negative cases: an absent
// file, a group/world-accessible file, an empty file, and a non-regular file
// (directory) each stop startup rather than yielding an empty secret.
func TestResolveSecretSourcesFailsClosed(t *testing.T) {
	dir := t.TempDir()

	t.Run("absent file", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.MCPAPIKey = ""
		cfg.MCPAPIKeyFile = filepath.Join(dir, "absent.key")
		err := cfg.ResolveSecretSources()
		if err == nil {
			t.Fatal("an absent secret file must fail closed")
		}
		if !strings.Contains(err.Error(), "R17-06") || !strings.Contains(err.Error(), "MCP_API_KEY") {
			t.Fatalf("error must name R17-06 and the variable, got %v", err)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.MCPAPIKey = ""
		cfg.MCPAPIKeyFile = writeSecretFile(t, dir, "empty.key", "\n  \n")
		err := cfg.ResolveSecretSources()
		if err == nil || !strings.Contains(err.Error(), "empty") {
			t.Fatalf("an empty secret file must fail closed, got %v", err)
		}
	})

	t.Run("non-regular file", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.MCPAPIKey = ""
		cfg.MCPAPIKeyFile = dir
		if err := cfg.ResolveSecretSources(); err == nil {
			t.Fatal("a non-regular secret source must fail closed")
		}
	})

	t.Run("group/world accessible file", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("POSIX mode check is not enforced on Windows")
		}
		path := filepath.Join(dir, "permissive.key")
		if err := os.WriteFile(path, []byte("some-secret-value-000000000000"), 0o644); err != nil {
			t.Fatalf("write permissive file: %v", err)
		}
		cfg := validMinimalConfig()
		cfg.MCPAPIKey = ""
		cfg.MCPAPIKeyFile = path
		err := cfg.ResolveSecretSources()
		if err == nil {
			t.Fatal("a group/world-readable secret file must be rejected")
		}
		if !strings.Contains(err.Error(), "group/world") {
			t.Fatalf("error must name the group/world access, got %v", err)
		}
	})
}

// TestValidateRequiresASecretSource proves the check that replaced the `required`
// env tags: an unset secret with no file source fails validation, and naming a
// file source satisfies it.
func TestValidateRequiresASecretSource(t *testing.T) {
	cfg := validMinimalConfig()
	cfg.MCPAPIKey = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("an unset MCP_API_KEY with no file source must fail Validate")
	}
	// The error must be the source-requirement gate (R17-06), not the later
	// key-strength check: a missing source is a different fault from a weak key.
	if !strings.Contains(err.Error(), "R17-06") || !strings.Contains(err.Error(), "MCP_API_KEY") {
		t.Fatalf("error must name R17-06 and MCP_API_KEY, got %v", err)
	}

	// A file source satisfies the requirement.
	dir := t.TempDir()
	cfg.MCPAPIKeyFile = writeSecretFile(t, dir, "mcp.key", "AbCdEfGhIjKlMnOpQrStUvWxYz123456")
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config with a secret file source must pass Validate, got %v", err)
	}
}

// TestSecretFileSourceFeedsTheExistingGates proves the resolved value flows
// through the gates that already exist rather than bypassing them: a file
// supplying the shipped Compose fallback is rejected by R17-05 exactly as the
// inline value would be.
func TestSecretFileSourceFeedsTheExistingGates(t *testing.T) {
	dir := t.TempDir()
	cfg := validMinimalConfig()
	cfg.DeploymentProfile = DeploymentProfilePublic
	cfg.MCPBindHost = "10.0.0.5"
	cfg.WebEnabled = false
	cfg.DBSSLMode = "verify-full"
	cfg.RedisUseTLS = true
	cfg.TrustedProxies = []string{"10.0.0.1"}
	cfg.CORSAllowedOrigins = []string{"https://guardrails.example.com"}
	cfg.DBPassword = ""
	cfg.DBPasswordFile = writeSecretFile(t, dir, "db.pass", "changeme")

	err := cfg.Validate()
	if err == nil {
		t.Fatal("a placeholder credential supplied from a file must still be rejected")
	}
	if !strings.Contains(err.Error(), "R17-05") {
		t.Fatalf("error must name R17-05 (the file source must not bypass the gate), got %v", err)
	}
}
