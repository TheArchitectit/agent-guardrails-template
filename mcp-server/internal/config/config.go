package config

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/thearchitectit/guardrail-mcp/internal/clientip"
	"github.com/thearchitectit/guardrail-mcp/internal/keymaterial"
)

// SchemaVersion tracks the configuration schema version for migrations
const SchemaVersion = "1.0"

// Config holds all application configuration
type Config struct {
	// Schema Version (for config migration tracking)
	SchemaVersion string `env:"CONFIG_SCHEMA_VERSION" envDefault:"1.0"`

	// Server Configuration
	MCPPort        int           `env:"MCP_PORT" envDefault:"8080"`
	MCPBindHost    string        `env:"MCP_BIND_HOST" envDefault:"127.0.0.1"`
	LogLevel       string        `env:"LOG_LEVEL" envDefault:"info"`
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`

	// Graceful Shutdown Configuration
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`

	// Web UI Configuration
	WebPort     int    `env:"WEB_PORT" envDefault:"8081"`
	WebBindHost string `env:"WEB_BIND_HOST" envDefault:"127.0.0.1"`
	WebEnabled  bool   `env:"WEB_ENABLED" envDefault:"true"`

	// CORS Configuration
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	CORSAllowedMethods []string `env:"CORS_ALLOWED_METHODS" envDefault:"GET,POST,PUT,DELETE,OPTIONS"`
	CORSAllowedHeaders []string `env:"CORS_ALLOWED_HEADERS" envDefault:"Authorization,Content-Type,X-Request-ID"`
	CORSMaxAge         int      `env:"CORS_MAX_AGE" envDefault:"86400"`

	// Profiling Configuration (Spec 17 R17-09). Disabled by default; when
	// enabled for diagnosis it must bind a loopback-only address, carry a
	// bounded access window (PPROF_UNTIL) and, outside the local profile, an
	// explicit attestation that no public/tailnet proxy route reaches it.
	PProfEnabled  bool   `env:"PPROF_ENABLED" envDefault:"false"`
	PProfPort     int    `env:"PPROF_PORT" envDefault:"6060"`
	PProfBindHost string `env:"PPROF_BIND_HOST" envDefault:"127.0.0.1"`
	// PProfUntil is the RFC3339 deadline after which the debug endpoint must
	// stop serving. Required when PProfEnabled is true.
	PProfUntil string `env:"PPROF_UNTIL"`
	// PProfNoProxyRouteAttestation is the operator attestation that no
	// public/tailnet proxy route reaches the loopback debug endpoint. Required
	// outside the local profile.
	PProfNoProxyRouteAttestation string `env:"PPROF_NO_PROXY_ROUTE_ATTESTATION"`

	// Health Check Configuration
	HealthCheckTimeout time.Duration `env:"HEALTH_CHECK_TIMEOUT" envDefault:"3s"`

	// Database Configuration
	DBHost string `env:"DB_HOST" envDefault:"localhost"`
	DBPort int    `env:"DB_PORT" envDefault:"5432"`
	DBName string `env:"DB_NAME" envDefault:"guardrails"`
	DBUser string `env:"DB_USER,required"`
	// DBPassword may come from the environment or, as R17-06 prefers for
	// production, from a read-only mounted secret file named by DB_PASSWORD_FILE.
	// The `required` tag is deliberately absent so a file-only deployment can
	// start; ValidateSecretSources enforces that at least one source exists.
	DBPassword        string        `env:"DB_PASSWORD"`
	DBPasswordFile    string        `env:"DB_PASSWORD_FILE"`
	DBSSLMode         string        `env:"DB_SSLMODE" envDefault:"require"`
	DBConnectTimeout  time.Duration `env:"DB_CONNECT_TIMEOUT" envDefault:"10s"`
	DBMaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	DBMaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	DBConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"30m"`
	DBConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"10m"`

	// Redis Configuration
	RedisHost         string        `env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort         int           `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword     string        `env:"REDIS_PASSWORD"`
	RedisPasswordFile string        `env:"REDIS_PASSWORD_FILE"`
	RedisUseTLS       bool          `env:"REDIS_USE_TLS" envDefault:"false"`
	RedisDB           int           `env:"REDIS_DB" envDefault:"0"`
	RedisPoolSize     int           `env:"REDIS_POOL_SIZE" envDefault:"10"`
	RedisMinIdleConns int           `env:"REDIS_MIN_IDLE_CONNS" envDefault:"2"`
	RedisMaxRetries   int           `env:"REDIS_MAX_RETRIES" envDefault:"3"`
	RedisDialTimeout  time.Duration `env:"REDIS_DIAL_TIMEOUT" envDefault:"5s"`
	RedisReadTimeout  time.Duration `env:"REDIS_READ_TIMEOUT" envDefault:"3s"`

	// TLS Configuration
	TLSEnabled    bool   `env:"TLS_ENABLED" envDefault:"false"`
	TLSCertPath   string `env:"TLS_CERT_PATH"`
	TLSKeyPath    string `env:"TLS_KEY_PATH"`
	TLSCAPath     string `env:"TLS_CA_PATH"`
	TLSMinVersion string `env:"TLS_MIN_VERSION" envDefault:"1.3"`

	// Security Configuration. Each legacy key may come from the environment or
	// from a read-only secret file (MCP_API_KEY_FILE / IDE_API_KEY_FILE); see
	// secret_sources.go for the R17-06 file-source rule.
	MCPAPIKey     string `env:"MCP_API_KEY"`
	MCPAPIKeyFile string `env:"MCP_API_KEY_FILE"`
	IDEAPIKey     string `env:"IDE_API_KEY"`
	IDEAPIKeyFile string `env:"IDE_API_KEY_FILE"`

	// Credential-to-principal registry (Spec 15 / gr-xp-01).
	// Records are supplied out of band as an inline JSON list and/or a JSON
	// file; the registry stores only a keyed one-way verifier of each secret,
	// never the secret itself. When no records are configured the legacy
	// MCP_API_KEY/IDE_API_KEY surface is retained unchanged.
	CredentialRegistryJSON string `env:"CREDENTIAL_REGISTRY"`
	// CredentialRegistryFile is a read-only mounted JSON secret file holding
	// the record list (Spec 17 R17-06). It is never serialized to callers.
	CredentialRegistryFile string `env:"CREDENTIAL_REGISTRY_FILE"`
	CredentialVerifierKey  string `env:"CREDENTIAL_VERIFIER_KEY"`
	// CredentialVerifierKeyFile is a read-only mounted secret file holding the
	// verifier key, as an alternative to inline CREDENTIAL_VERIFIER_KEY.
	CredentialVerifierKeyFile string `env:"CREDENTIAL_VERIFIER_KEY_FILE"`

	// JWT Configuration. The signing secret may come from the environment or
	// from a read-only secret file (JWT_SECRET_FILE).
	JWTSecret        string        `env:"JWT_SECRET"`
	JWTSecretFile    string        `env:"JWT_SECRET_FILE"`
	JWTIssuer        string        `env:"JWT_ISSUER" envDefault:"guardrail-mcp"`
	JWTExpiry        time.Duration `env:"JWT_EXPIRY" envDefault:"15m"`
	JWTRotationHours time.Duration `env:"JWT_ROTATION_HOURS" envDefault:"168h"` // 7 days

	// Rate Limiting Configuration (prefixed consistently)
	RateLimitMCP         int           `env:"RATE_LIMIT_MCP" envDefault:"1000"`
	RateLimitIDE         int           `env:"RATE_LIMIT_IDE" envDefault:"500"`
	RateLimitSession     int           `env:"RATE_LIMIT_SESSION" envDefault:"100"`
	RateLimitWindow      time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
	RateLimitBurstFactor float64       `env:"RATE_LIMIT_BURST_FACTOR" envDefault:"1.5"`

	// Cache TTL Configuration
	CacheTTLRules  time.Duration `env:"CACHE_TTL_RULES" envDefault:"5m"`
	CacheTTLDocs   time.Duration `env:"CACHE_TTL_DOCS" envDefault:"10m"`
	CacheTTLSearch time.Duration `env:"CACHE_TTL_SEARCH" envDefault:"2m"`

	// Feature Flags (hot-reloadable)
	EnableValidation   bool `env:"ENABLE_VALIDATION" envDefault:"true"`
	EnableMetrics      bool `env:"ENABLE_METRICS" envDefault:"true"`
	EnableAuditLogging bool `env:"ENABLE_AUDIT_LOGGING" envDefault:"true"`
	EnableCache        bool `env:"ENABLE_CACHE" envDefault:"true"`

	// Audit Logging Configuration
	AuditBufferSize    int           `env:"AUDIT_BUFFER_SIZE" envDefault:"1000"`
	AuditFlushInterval time.Duration `env:"AUDIT_FLUSH_INTERVAL" envDefault:"5s"`

	// Circuit Breaker Configuration
	CircuitBreakerEnabled          bool          `env:"CIRCUIT_BREAKER_ENABLED" envDefault:"true"`
	CircuitBreakerFailureThreshold int           `env:"CIRCUIT_BREAKER_FAILURE_THRESHOLD" envDefault:"5"`
	CircuitBreakerSuccessThreshold int           `env:"CIRCUIT_BREAKER_SUCCESS_THRESHOLD" envDefault:"2"`
	CircuitBreakerTimeout          time.Duration `env:"CIRCUIT_BREAKER_TIMEOUT" envDefault:"30s"`
	CircuitBreakerMaxRequests      int           `env:"CIRCUIT_BREAKER_MAX_REQUESTS" envDefault:"3"`
	CircuitBreakerInterval         time.Duration `env:"CIRCUIT_BREAKER_INTERVAL" envDefault:"10s"`

	// Deployment Profile (Spec 17 §2 / R17-01). Selects the effective
	// exposure profile: local (default, loopback-only), tailnet, or public.
	// The profile is validated against the actual listener bind addresses at
	// startup; a mismatch fails startup instead of downgrading silently.
	DeploymentProfile string `env:"DEPLOYMENT_PROFILE" envDefault:"local"`
	// DeploymentBindAttestation is the operator attestation that a wildcard
	// listen address sits on a verified isolated local backend network with
	// host-loopback-only publishing (local Compose). Required for local +
	// wildcard binds; ignored otherwise.
	DeploymentBindAttestation string `env:"DEPLOYMENT_BIND_ATTESTATION"`
	// TrustedProxies are the proxy peers allowed to supply forwarded
	// identity/scheme headers. Required (non-empty) for the public profile,
	// which may only listen behind a controlled TLS-terminating proxy.
	// Empty by default: forwarded headers are ignored (Spec 12 §3.4).
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:","`
	// DeploymentPreflightFile names the commit-bound deployment preflight
	// record (JSON) that proves the ingress posture for a nonlocal profile
	// (Spec 17 R17-02): TLS termination and direct-backend denial for `public`,
	// verified private ingress and encrypted authenticated transport for
	// `tailnet`. Required outside `local`; a PRODUCTION_MODE flag, TLS_ENABLED
	// or certificate paths alone never satisfy the gate.
	DeploymentPreflightFile string `env:"DEPLOYMENT_PREFLIGHT_FILE"`

	// LegacyAcceptanceUntil bounds how long the legacy shared keys
	// (MCP_API_KEY / IDE_API_KEY) are accepted (Spec 11 §4.5.2). Unset means the
	// removal switch is not configured and legacy keys keep their historical
	// surface. A configured value is an RFC3339 timestamp; once it has passed,
	// a legacy key is refused on both transports regardless of registry state.
	LegacyAcceptanceUntil string `env:"LEGACY_ACCEPTANCE_UNTIL"`

	// Production Mode Indicator
	ProductionMode bool `env:"PRODUCTION_MODE" envDefault:"false"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Resolve any read-only secret-file sources before validation (Spec 17
	// R17-06), so every downstream gate — storage transit, default-credential
	// rejection, key/JWT strength — sees the resolved value rather than an
	// empty field that only looked unset.
	if err := cfg.ResolveSecretSources(); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate performs comprehensive configuration validation
func (c *Config) Validate() error {
	// Resolve any read-only secret-file sources first (Spec 17 R17-06) so every
	// gate below — including storage transit and default-credential rejection —
	// sees the resolved value rather than an empty field that only looked
	// unset. Load() resolves before calling Validate; this keeps a
	// directly-constructed Config on the same path.
	if err := c.ResolveSecretSources(); err != nil {
		return err
	}

	// Require a source for every primary secret (Spec 17 R17-06) before the
	// strength checks below. This replaces the `required` env tags those fields
	// used to carry, which made a file-only deployment impossible to start.
	if err := c.ValidateSecretSources(); err != nil {
		return err
	}

	// Validate JWT secret
	if err := ValidateJWTSecret(c.JWTSecret); err != nil {
		return fmt.Errorf("JWT_SECRET validation failed: %w", err)
	}

	// Validate API keys
	if err := ValidateAPIKey(c.MCPAPIKey, "MCP_API_KEY"); err != nil {
		return err
	}
	if err := ValidateAPIKey(c.IDEAPIKey, "IDE_API_KEY"); err != nil {
		return err
	}

	// Require a source for every primary secret (Spec 17 R17-06). This replaces
	// the `required` env tags those fields used to carry, which made a
	// file-only deployment impossible to start.
	if err := c.ValidateSecretSources(); err != nil {
		return err
	}

	// Validate the credential registry: a configured registry requires the
	// out-of-band verifier key — supplied inline or as a read-only secret file
	// — so verifier digests are keyed one-way values.
	if err := c.validateCredentialRegistry(); err != nil {
		return err
	}

	// Validate the legacy-acceptance removal switch (Spec 11 §4.5.2): a
	// configured value must parse as RFC3339, so an unreadable deadline fails
	// startup rather than silently disabling the retirement of legacy keys.
	if _, _, err := c.LegacyAcceptanceDeadline(); err != nil {
		return err
	}

	// Validate the debug/pprof endpoint posture (Spec 17 R17-09): when enabled
	// it must be loopback-only, time-boxed, and — outside the local profile —
	// carry an explicit no-proxy-route attestation. A misconfigured debug
	// endpoint fails startup rather than serving an administrative surface.
	if err := c.ValidateDebugEndpoint(); err != nil {
		return err
	}

	// Validate timeouts
	if err := ValidateTimeout("SHUTDOWN_TIMEOUT", c.ShutdownTimeout, 5*time.Second, 5*time.Minute); err != nil {
		return err
	}
	if err := ValidateTimeout("REQUEST_TIMEOUT", c.RequestTimeout, 1*time.Second, 5*time.Minute); err != nil {
		return err
	}
	if err := ValidateTimeout("DB_CONNECT_TIMEOUT", c.DBConnectTimeout, 1*time.Second, 2*time.Minute); err != nil {
		return err
	}

	// Validate database connection pool
	if c.DBMaxOpenConns < 1 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be at least 1, got %d", c.DBMaxOpenConns)
	}
	if c.DBMaxOpenConns > 1000 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be at most 1000, got %d", c.DBMaxOpenConns)
	}
	if c.DBMaxIdleConns < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be non-negative, got %d", c.DBMaxIdleConns)
	}
	if c.DBMaxIdleConns > c.DBMaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)",
			c.DBMaxIdleConns, c.DBMaxOpenConns)
	}

	// Validate Redis connection pool
	if c.RedisPoolSize < 1 {
		return fmt.Errorf("REDIS_POOL_SIZE must be at least 1, got %d", c.RedisPoolSize)
	}
	if c.RedisPoolSize > 100 {
		return fmt.Errorf("REDIS_POOL_SIZE must be at most 100, got %d", c.RedisPoolSize)
	}
	if c.RedisMinIdleConns < 0 {
		return fmt.Errorf("REDIS_MIN_IDLE_CONNS must be non-negative, got %d", c.RedisMinIdleConns)
	}
	if c.RedisMinIdleConns > c.RedisPoolSize {
		return fmt.Errorf("REDIS_MIN_IDLE_CONNS (%d) cannot exceed REDIS_POOL_SIZE (%d)",
			c.RedisMinIdleConns, c.RedisPoolSize)
	}

	// Validate rate limits
	if c.RateLimitMCP < 1 {
		return fmt.Errorf("RATE_LIMIT_MCP must be at least 1, got %d", c.RateLimitMCP)
	}
	if c.RateLimitIDE < 1 {
		return fmt.Errorf("RATE_LIMIT_IDE must be at least 1, got %d", c.RateLimitIDE)
	}
	if c.RateLimitSession < 1 {
		return fmt.Errorf("RATE_LIMIT_SESSION must be at least 1, got %d", c.RateLimitSession)
	}
	if c.RateLimitBurstFactor < 1.0 || c.RateLimitBurstFactor > 5.0 {
		return fmt.Errorf("RATE_LIMIT_BURST_FACTOR must be between 1.0 and 5.0, got %.2f", c.RateLimitBurstFactor)
	}

	// Validate TLS configuration
	if c.TLSEnabled {
		if c.TLSCertPath == "" {
			return fmt.Errorf("TLS_CERT_PATH is required when TLS_ENABLED is true")
		}
		if c.TLSKeyPath == "" {
			return fmt.Errorf("TLS_KEY_PATH is required when TLS_ENABLED is true")
		}
		if c.TLSMinVersion != "1.2" && c.TLSMinVersion != "1.3" {
			return fmt.Errorf("TLS_MIN_VERSION must be 1.2 or 1.3, got %s", c.TLSMinVersion)
		}
	}

	// Validate log level
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error, got %s", c.LogLevel)
	}

	// Validate SSL mode
	validSSLModes := map[string]bool{"disable": true, "require": true, "prefer": true, "verify-ca": true, "verify-full": true}
	if !validSSLModes[c.DBSSLMode] {
		return fmt.Errorf("DB_SSLMODE must be one of: disable, require, prefer, verify-ca, verify-full, got %s", c.DBSSLMode)
	}

	// Validate the deployment profile and bind the listener addresses to it
	// (Spec 17 R17-01/R17-02). A profile/bind mismatch fails startup.
	if _, err := NormalizeDeploymentProfile(c.DeploymentProfile); err != nil {
		return err
	}
	if err := c.ValidateDeploymentBinding(c.MCPBindAddr()); err != nil {
		return err
	}
	if c.WebEnabled {
		if err := c.ValidateDeploymentBinding(c.WebBindAddr()); err != nil {
			return err
		}
	}

	// Fail closed on an unproven nonlocal ingress posture (Spec 17 R17-02): a
	// `public` deployment must prove TLS termination and direct-backend denial,
	// a `tailnet` deployment verified private ingress and encrypted
	// authenticated transport, via a commit-bound preflight record. A bind that
	// merely is not a wildcard is not evidence of a safe exposure.
	if err := c.ValidateIngressPosture(); err != nil {
		return err
	}

	// Couple the storage transit posture to the effective profile (Spec 17
	// R17-04): outside the isolated local boundary, DB/Redis transit must be
	// authenticated and encrypted, and a plaintext posture fails startup rather
	// than connecting in the clear.
	if err := c.ValidateStorageTransit(); err != nil {
		return err
	}

	// Reject known placeholder/default credentials outside the local boundary
	// (Spec 17 R17-05): a nonlocal or production deployment must not start with
	// the shipped Compose fallback or another publicly-known default.
	if err := c.ValidateNoProductionDefaultCredentials(); err != nil {
		return err
	}

	// Validate the trusted-proxy stance (Spec 12 §3.4). Every entry must be a
	// valid IP or CIDR: a malformed entry is a configuration error, not a
	// silent narrowing to "trust nothing".
	if _, err := clientip.NewStance(c.TrustedProxies); err != nil {
		return err
	}

	// Validate audit settings
	if c.AuditBufferSize < 100 {
		return fmt.Errorf("AUDIT_BUFFER_SIZE must be at least 100, got %d", c.AuditBufferSize)
	}
	if c.AuditBufferSize > 10000 {
		return fmt.Errorf("AUDIT_BUFFER_SIZE must be at most 10000, got %d", c.AuditBufferSize)
	}

	// Validate CORS settings
	if len(c.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must not be empty")
	}
	// Production must not accept a wildcard CORS origin. `*` permits any
	// origin to read credentialed responses; in production that is a
	// configuration error, not a default. The web layer already falls back
	// to restrictive localhost origins when CORS is unset or `*`, but a
	// production operator who explicitly sets `*` must be told, not silently
	// narrowed. Spec 12-4.1 / audit release gap 6.
	if c.ProductionMode {
		for _, origin := range c.CORSAllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain wildcard '*' in production; list explicit origins")
			}
		}
	}

	// Validate circuit breaker settings
	if c.CircuitBreakerFailureThreshold < 1 {
		return fmt.Errorf("CIRCUIT_BREAKER_FAILURE_THRESHOLD must be at least 1, got %d", c.CircuitBreakerFailureThreshold)
	}
	if c.CircuitBreakerSuccessThreshold < 1 {
		return fmt.Errorf("CIRCUIT_BREAKER_SUCCESS_THRESHOLD must be at least 1, got %d", c.CircuitBreakerSuccessThreshold)
	}
	if err := ValidateTimeout("CIRCUIT_BREAKER_TIMEOUT", c.CircuitBreakerTimeout, 1*time.Second, 5*time.Minute); err != nil {
		return err
	}
	if c.CircuitBreakerMaxRequests < 1 {
		return fmt.Errorf("CIRCUIT_BREAKER_MAX_REQUESTS must be at least 1, got %d", c.CircuitBreakerMaxRequests)
	}
	if err := ValidateTimeout("CIRCUIT_BREAKER_INTERVAL", c.CircuitBreakerInterval, 1*time.Second, 5*time.Minute); err != nil {
		return err
	}

	return nil
}

// validateCredentialRegistry enforces the load-boundary rule for a configured
// credential registry: verifier material must come from exactly one of the two
// approved sources (inline or a read-only secret file), and any inline key must
// clear the Spec 17 R17-07 strength floor. R17-07 states plainly that "a length
// check is not an entropy proof", so the length floor below is only the cheap
// first gate; the real rule is the entropy bound (>=256 bits for a nonlocal or
// production deployment, the >=128-bit floor for `local`).
func (c *Config) validateCredentialRegistry() error {
	if !SourcesConfiguredRegistry(c) {
		return nil
	}
	if strings.TrimSpace(c.CredentialVerifierKey) == "" && strings.TrimSpace(c.CredentialVerifierKeyFile) == "" {
		return fmt.Errorf("a configured credential registry requires CREDENTIAL_VERIFIER_KEY or CREDENTIAL_VERIFIER_KEY_FILE")
	}
	if c.CredentialVerifierKey != "" && len(c.CredentialVerifierKey) < 16 {
		return fmt.Errorf("CREDENTIAL_VERIFIER_KEY must be at least 16 characters when a credential registry is configured")
	}
	// Spec 17 R17-07 strength floor. Only the inline value is checked here: a
	// file-sourced key is validated at the registry load boundary (auth.New),
	// which reads and applies the same rule. `local` keeps the published
	// floor; a nonlocal or production deployment must clear the >=256-bit
	// bound R17-07 names, so a weak key fails startup rather than becoming the
	// HMAC key behind every stored credential verifier.
	if c.CredentialVerifierKey != "" {
		minBits := keymaterial.MinVerifierKeyBits
		if c.requiresProductionCredentials() {
			minBits = keymaterial.ProductionVerifierKeyBits
		}
		if err := keymaterial.ValidateVerifierKeyEntropy(c.CredentialVerifierKey, minBits); err != nil {
			return fmt.Errorf("CREDENTIAL_VERIFIER_KEY is too weak for profile %q: %w", c.effectiveProfileName(), err)
		}
	}
	return nil
}

// LegacyAcceptanceDeadline parses LEGACY_ACCEPTANCE_UNTIL. An unset value
// returns (zero, false, nil): the removal switch is not configured, and legacy
// keys are not retired by it. A malformed value is a hard error, never a
// silent "no deadline".
func (c *Config) LegacyAcceptanceDeadline() (time.Time, bool, error) {
	if c == nil || strings.TrimSpace(c.LegacyAcceptanceUntil) == "" {
		return time.Time{}, false, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(c.LegacyAcceptanceUntil))
	if err != nil {
		return time.Time{}, false, fmt.Errorf("LEGACY_ACCEPTANCE_UNTIL must be an RFC3339 timestamp, got %q", c.LegacyAcceptanceUntil)
	}
	return t, true, nil
}

// LegacyAcceptanceExpired reports whether a configured legacy-acceptance
// deadline has passed as of now. An unconfigured switch never expires; a
// malformed one is an error (fail closed at the caller). This is the testable
// removal-date switch required by Spec 11 acceptance criterion 8.
func (c *Config) LegacyAcceptanceExpired(now time.Time) (bool, error) {
	deadline, configured, err := c.LegacyAcceptanceDeadline()
	if err != nil {
		return false, err
	}
	if !configured {
		return false, nil
	}
	return now.After(deadline), nil
}

// SourcesConfiguredRegistry reports whether a credential registry source is
// configured. It mirrors the registry load rule so config validation and
// runtime loading agree on what counts as "configured".
func SourcesConfiguredRegistry(c *Config) bool {
	if c == nil {
		return false
	}
	return strings.TrimSpace(c.CredentialRegistryJSON) != "" || strings.TrimSpace(c.CredentialRegistryFile) != ""
}

// ValidateJWTSecret ensures the JWT secret meets security requirements
func ValidateJWTSecret(secret string) error {
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 bytes, got %d", len(secret))
	}

	// Check entropy using Shannon entropy over byte frequencies.
	// Random secrets (hex, base64) score ~4.0 bits; human-readable or
	// repetitive strings score far lower. The previous check used popcount
	// (bits set per byte), which wrongly rejected valid hex secrets.
	freq := make(map[byte]int)
	for i := 0; i < len(secret); i++ {
		freq[secret[i]]++
	}
	n := float64(len(secret))
	entropy := 0.0
	for _, count := range freq {
		p := float64(count) / n
		entropy -= p * math.Log2(p)
	}
	if entropy < 3.0 {
		return fmt.Errorf("JWT_SECRET has insufficient entropy (should be random, not human-readable)")
	}

	return nil
}

// ValidateAPIKey validates an API key meets minimum security requirements
func ValidateAPIKey(key, name string) error {
	if len(key) < 32 {
		return fmt.Errorf("%s must be at least 32 characters, got %d", name, len(key))
	}

	// Check for common weak patterns
	weakPatterns := []string{
		`^[a-zA-Z]+$`,            // Only letters
		`^[0-9]+$`,               // Only numbers
		`^(password|secret|key)`, // Common weak prefixes
	}

	for _, pattern := range weakPatterns {
		matched, err := regexp.MatchString(pattern, key)
		if err != nil {
			continue
		}
		if matched {
			return fmt.Errorf("%s appears to be weak (avoid only letters, only numbers, or common words)", name)
		}
	}

	// Check for reasonable character variety
	var hasLower, hasUpper, hasDigit bool
	for _, c := range key {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}

	if !hasLower || !hasUpper || !hasDigit {
		return fmt.Errorf("%s should contain a mix of uppercase, lowercase, and digits", name)
	}

	return nil
}

// ValidateTimeout validates a timeout is within acceptable bounds
func ValidateTimeout(name string, value, min, max time.Duration) error {
	if value < min {
		return fmt.Errorf("%s must be at least %v, got %v", name, min, value)
	}
	if value > max {
		return fmt.Errorf("%s must be at most %v, got %v", name, max, value)
	}
	return nil
}

// DatabaseURL returns the PostgreSQL connection string
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
		int(c.DBConnectTimeout.Seconds()))
}

// RedisAddr returns the Redis connection address
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

// IsHotReloadable returns true if the config key supports hot reloading
func IsHotReloadable(key string) bool {
	hotReloadable := map[string]bool{
		"LOG_LEVEL":               true,
		"RATE_LIMIT_MCP":          true,
		"RATE_LIMIT_IDE":          true,
		"RATE_LIMIT_SESSION":      true,
		"RATE_LIMIT_WINDOW":       true,
		"RATE_LIMIT_BURST_FACTOR": true,
		"CACHE_TTL_RULES":         true,
		"CACHE_TTL_DOCS":          true,
		"CACHE_TTL_SEARCH":        true,
		"ENABLE_VALIDATION":       true,
		"ENABLE_METRICS":          true,
		"ENABLE_AUDIT_LOGGING":    true,
		"ENABLE_CACHE":            true,
		"CORS_ALLOWED_ORIGINS":    true,
		"CORS_MAX_AGE":            true,
	}
	return hotReloadable[key]
}

// HotReloadableFields returns a list of all hot-reloadable configuration keys
func HotReloadableFields() []string {
	return []string{
		"LOG_LEVEL",
		"RATE_LIMIT_MCP",
		"RATE_LIMIT_IDE",
		"RATE_LIMIT_SESSION",
		"RATE_LIMIT_WINDOW",
		"RATE_LIMIT_BURST_FACTOR",
		"CACHE_TTL_RULES",
		"CACHE_TTL_DOCS",
		"CACHE_TTL_SEARCH",
		"ENABLE_VALIDATION",
		"ENABLE_METRICS",
		"ENABLE_AUDIT_LOGGING",
		"ENABLE_CACHE",
		"CORS_ALLOWED_ORIGINS",
		"CORS_MAX_AGE",
	}
}

// Masked returns a copy of the config with sensitive values masked
func (c *Config) Masked() *Config {
	masked := *c
	masked.DBPassword = "***"
	masked.DBPasswordFile = "***"
	masked.RedisPassword = "***"
	masked.RedisPasswordFile = "***"
	masked.MCPAPIKey = "***"
	masked.MCPAPIKeyFile = "***"
	masked.IDEAPIKey = "***"
	masked.IDEAPIKeyFile = "***"
	masked.JWTSecret = "***"
	masked.JWTSecretFile = "***"
	masked.CredentialVerifierKey = "***"
	masked.CredentialRegistryJSON = "***"
	masked.CredentialRegistryFile = "***"
	masked.CredentialVerifierKeyFile = "***"
	return &masked
}
