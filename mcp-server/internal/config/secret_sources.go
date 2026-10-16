package config

import (
	"fmt"
	"strings"

	"github.com/thearchitectit/guardrail-mcp/internal/secretfile"
)

// Secret-source resolution (Spec 17 R17-06).
//
// R17-06 makes a managed secret store or a runtime read-only mounted secret
// file the first-choice production source for DB/Redis credentials, the legacy
// MCP/IDE keys and JWT signing material, and requires that file read be
// restricted to the service identity (owner-only mode, no group/world access).
//
// Before this existed the tree had exactly two file-based secret sources
// (CREDENTIAL_REGISTRY_FILE and CREDENTIAL_VERIFIER_KEY_FILE) and no way at all
// to supply the five primary secrets from a file: `MCP_API_KEY`, `IDE_API_KEY`,
// `JWT_SECRET` and `DB_PASSWORD` carried `required` env tags, so a deployment
// that mounted its secrets as files could not even start, and `REDIS_PASSWORD`
// had only an environment source. Environment injection was therefore the only
// available path for every one of them — the exact posture R17-06 says must not
// be the production default.
//
// Each secret now accepts an explicit `<NAME>_FILE` alternative. The file is
// read through the same restricted-file rule the credential registry uses
// (regular file; no group/world access on POSIX; any error is fail-closed), and
// the resolved value then flows through every existing gate unchanged —
// including the R17-04 storage-transit coupling and the R17-05
// placeholder/default-credential rejection.

// secretSource ties a resolved secret field to its optional file alternative.
type secretSource struct {
	// name is the environment variable base name (e.g. "MCP_API_KEY").
	name string
	// value is the resolved secret; file is the read-only secret-file path.
	value *string
	file  *string
}

// secretSources returns the primary secrets that accept a file source. The
// order is stable so error messages are deterministic.
func (c *Config) secretSources() []secretSource {
	return []secretSource{
		{name: "MCP_API_KEY", value: &c.MCPAPIKey, file: &c.MCPAPIKeyFile},
		{name: "IDE_API_KEY", value: &c.IDEAPIKey, file: &c.IDEAPIKeyFile},
		{name: "JWT_SECRET", value: &c.JWTSecret, file: &c.JWTSecretFile},
		{name: "DB_PASSWORD", value: &c.DBPassword, file: &c.DBPasswordFile},
		{name: "REDIS_PASSWORD", value: &c.RedisPassword, file: &c.RedisPasswordFile},
	}
}

// ResolveSecretSources fills each secret from its configured read-only secret
// file when no inline value is present. Inline environment material wins when
// both are supplied (the same precedence the credential-registry loader uses),
// so an existing environment-only deployment is unaffected. It is idempotent:
// once a value is resolved the file is not read again.
//
// A configured-but-unreadable, non-regular, group/world-accessible or empty
// file is a hard error: the process must stop rather than start with a missing
// or silently-ignored secret.
func (c *Config) ResolveSecretSources() error {
	if c == nil {
		return nil
	}
	for _, s := range c.secretSources() {
		filePath := strings.TrimSpace(*s.file)
		if filePath == "" {
			continue
		}
		if strings.TrimSpace(*s.value) != "" {
			// Inline material present: it wins and the file is not read.
			continue
		}
		data, err := secretfile.Read(filePath)
		if err != nil {
			return fmt.Errorf("%s_FILE could not be read (Spec 17 R17-06): %w", s.name, err)
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return fmt.Errorf("%s_FILE is empty (Spec 17 R17-06): a configured secret file must hold the secret", s.name)
		}
		*s.value = value
	}
	return nil
}

// ValidateSecretSources requires every primary secret to have a source: an
// inline environment value or a resolved read-only secret file. It replaces the
// `required` env tags those fields used to carry, which made a file-only
// deployment impossible to start. Validate calls ResolveSecretSources first, so
// this is a pure check on the resolved values.
func (c *Config) ValidateSecretSources() error {
	if c == nil {
		return nil
	}
	for _, s := range c.secretSources() {
		if strings.TrimSpace(*s.value) != "" {
			continue
		}
		if strings.TrimSpace(*s.file) != "" {
			return fmt.Errorf("%s_FILE %q produced no value (Spec 17 R17-06)", s.name, strings.TrimSpace(*s.file))
		}
		return fmt.Errorf("%s is unset (Spec 17 R17-06): set it inline or point %s_FILE at a read-only secret file (owner-only mode, no group/world access)", s.name, s.name)
	}
	return nil
}
