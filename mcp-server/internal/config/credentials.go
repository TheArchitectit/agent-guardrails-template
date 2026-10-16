package config

import (
	"fmt"
	"strings"
)

// Production default-credential policy (Spec 17 R17-05).
//
// R17-05: nonlocal or production profiles require unique high-entropy
// operator-generated DB, Redis, MCP/IDE (during migration), scoped registry and
// JWT credentials; unset, empty, known placeholder/default, reused or test
// credentials are rejected before startup. No checked-in sample is a production
// credential. DB and Redis account privilege is service-specific, with separate
// accounts per environment.
//
// The tree ships a fixed, publicly-known Compose fallback (`changeme`) for the
// DB and Redis passwords; `docker-compose.yml` and the deploy examples use it
// via `${DB_PASSWORD:-changeme}` / `${REDIS_PASSWORD:-changeme}`. That fallback
// is fine for a loopback-only local run but must never be a production
// credential. Before this existed, `Validate` checked only length/character
// class for the API keys and the JWT secret and checked DB/Redis passwords not
// at all, so a `public`/`tailnet` deployment could start with `changeme` and
// connect with a credential any reader of this repository already knows.
//
// The check is a hard error naming the requirement and the offending variable,
// never a silent replacement: a misconfiguration must stop the process rather
// than start with a default secret.

// knownPlaceholderCredentials is the reviewed set of literal credential values
// that are shipped as samples/defaults in this repository (or are otherwise
// publicly known), and therefore must never appear as a production credential.
// Matching is case-insensitive after trimming. The set is deliberately small
// and named so it stays auditable; it is a floor, not an entropy oracle.
var knownPlaceholderCredentials = []string{
	"changeme",
	"change-me",
	"password",
	"passwd",
	"secret",
	"example",
	"placeholder",
	"default",
	"guardrail",
	"guardrails",
	"admin",
	"root",
	"test",
	"dev",
	"development",
	"local",
	"redis",
	"postgres",
	"postgresql",
	"mcp",
	"ide",
	"jwt",
	"your-password-here",
	"yourpassword",
	"xxxxxxxx",
}

// IsPlaceholderCredential reports whether value is a known placeholder/default
// credential (empty, a known literal, or an obvious sample form). It is the
// single decision used by the R17-05 gate.
func IsPlaceholderCredential(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return true
	}
	for _, p := range knownPlaceholderCredentials {
		if v == p {
			return true
		}
	}
	// Obvious sample forms: a `your-…-here`/`replace-…` template, or a value
	// that is a single repeated character.
	if strings.HasPrefix(v, "your-") || strings.HasPrefix(v, "your_") ||
		strings.HasPrefix(v, "replace-") || strings.HasPrefix(v, "replace_") ||
		strings.Contains(v, "<") || strings.Contains(v, ">") {
		return true
	}
	if len(v) > 1 {
		first := v[0]
		allSame := true
		for i := 1; i < len(v); i++ {
			if v[i] != first {
				allSame = false
				break
			}
		}
		if allSame {
			return true
		}
	}
	return false
}

// ValidateNoProductionDefaultCredentials fails startup when a production or
// nonlocal profile carries a known placeholder/default credential. The `local`
// profile is exempt: R17-05 explicitly keeps the loopback development posture
// working (the Compose fallback is documented as non-production).
func (c *Config) ValidateNoProductionDefaultCredentials() error {
	if !c.requiresProductionCredentials() {
		return nil
	}

	// DB and Redis credentials must be present and not a known default.
	if IsPlaceholderCredential(c.DBPassword) {
		return fmt.Errorf(
			"profile %q requires a unique high-entropy operator-generated database credential (Spec 17 R17-05): DB_PASSWORD is unset, empty, or a known placeholder/default; the Compose fallback is not a production credential",
			c.effectiveProfileName())
	}
	if IsPlaceholderCredential(c.RedisPassword) {
		return fmt.Errorf(
			"profile %q requires a unique high-entropy operator-generated Redis credential (Spec 17 R17-05): REDIS_PASSWORD is unset, empty, or a known placeholder/default; the Compose fallback is not a production credential",
			c.effectiveProfileName())
	}

	// Legacy shared keys are still accepted during migration (Spec 11 §4.5.2)
	// but must not themselves be a default/placeholder.
	if IsPlaceholderCredential(c.MCPAPIKey) {
		return fmt.Errorf(
			"profile %q rejects a placeholder MCP credential (Spec 17 R17-05): MCP_API_KEY is a known default; generate a unique operator credential",
			c.effectiveProfileName())
	}
	if IsPlaceholderCredential(c.IDEAPIKey) {
		return fmt.Errorf(
			"profile %q rejects a placeholder IDE credential (Spec 17 R17-05): IDE_API_KEY is a known default; generate a unique operator credential",
			c.effectiveProfileName())
	}

	// JWT signing material must not be a default/placeholder either.
	if IsPlaceholderCredential(c.JWTSecret) {
		return fmt.Errorf(
			"profile %q rejects a placeholder JWT signing secret (Spec 17 R17-05): JWT_SECRET is a known default; generate unique signing material",
			c.effectiveProfileName())
	}

	return nil
}

// requiresProductionCredentials reports whether the effective posture demands
// non-default credentials: any nonlocal profile, or an explicit production
// mode. The local profile (the safe default) is exempt.
func (c *Config) requiresProductionCredentials() bool {
	if c.ProductionMode {
		return true
	}
	profile, err := NormalizeDeploymentProfile(c.DeploymentProfile)
	if err != nil {
		// An unparsable profile is rejected elsewhere; treat it as demanding
		// (fail closed) rather than exempt.
		return true
	}
	return profile != DeploymentProfileLocal
}

// effectiveProfileName returns the normalized profile name for error messages.
func (c *Config) effectiveProfileName() string {
	if profile, err := NormalizeDeploymentProfile(c.DeploymentProfile); err == nil {
		return profile
	}
	return strings.TrimSpace(c.DeploymentProfile)
}
