package config

import "time"

// PublicConfig is the explicitly reviewed non-secret projection of Config,
// exposed to callers via the guardrail://config MCP resource.
//
// Review rule: every field here MUST be non-secret. Never add passwords, API
// keys, JWT secrets, verifier keys, credential-registry material, raw
// credentials, or secret file paths. When in doubt, leave it out.
type PublicConfig struct {
	SchemaVersion        string        `json:"schema_version"`
	MCPPort              int           `json:"mcp_port"`
	LogLevel             string        `json:"log_level"`
	RequestTimeout       time.Duration `json:"request_timeout"`
	ShutdownTimeout      time.Duration `json:"shutdown_timeout"`
	WebPort              int           `json:"web_port"`
	WebEnabled           bool          `json:"web_enabled"`
	CORSAllowedOrigins   []string      `json:"cors_allowed_origins"`
	CORSAllowedMethods   []string      `json:"cors_allowed_methods"`
	CORSAllowedHeaders   []string      `json:"cors_allowed_headers"`
	CORSMaxAge           int           `json:"cors_max_age"`
	PProfEnabled         bool          `json:"pprof_enabled"`
	HealthCheckTimeout   time.Duration `json:"health_check_timeout"`
	TLSEnabled           bool          `json:"tls_enabled"`
	TLSMinVersion        string        `json:"tls_min_version"`
	JWTIssuer            string        `json:"jwt_issuer"`
	JWTExpiry            time.Duration `json:"jwt_expiry"`
	JWTRotationHours     time.Duration `json:"jwt_rotation_hours"`
	RateLimitMCP         int           `json:"rate_limit_mcp"`
	RateLimitIDE         int           `json:"rate_limit_ide"`
	RateLimitSession     int           `json:"rate_limit_session"`
	RateLimitWindow      time.Duration `json:"rate_limit_window"`
	RateLimitBurstFactor float64       `json:"rate_limit_burst_factor"`
	CacheTTLRules        time.Duration `json:"cache_ttl_rules"`
	CacheTTLDocs         time.Duration `json:"cache_ttl_docs"`
	CacheTTLSearch       time.Duration `json:"cache_ttl_search"`
	EnableValidation     bool          `json:"enable_validation"`
	EnableMetrics        bool          `json:"enable_metrics"`
	EnableAuditLogging   bool          `json:"enable_audit_logging"`
	EnableCache          bool          `json:"enable_cache"`
	AuditBufferSize      int           `json:"audit_buffer_size"`
	AuditFlushInterval   time.Duration `json:"audit_flush_interval"`

	CircuitBreakerEnabled          bool          `json:"circuit_breaker_enabled"`
	CircuitBreakerFailureThreshold int           `json:"circuit_breaker_failure_threshold"`
	CircuitBreakerSuccessThreshold int           `json:"circuit_breaker_success_threshold"`
	CircuitBreakerTimeout          time.Duration `json:"circuit_breaker_timeout"`
	CircuitBreakerMaxRequests      int           `json:"circuit_breaker_max_requests"`
	CircuitBreakerInterval         time.Duration `json:"circuit_breaker_interval"`

	ProductionMode bool `json:"production_mode"`
}

// PublicView returns the reviewed non-secret projection of this Config.
// Deliberately allowlist-only: fields are copied one by one so a newly added
// secret on Config cannot leak by default.
func (c *Config) PublicView() PublicConfig {
	if c == nil {
		return PublicConfig{}
	}
	return PublicConfig{
		SchemaVersion:        c.SchemaVersion,
		MCPPort:              c.MCPPort,
		LogLevel:             c.LogLevel,
		RequestTimeout:       c.RequestTimeout,
		ShutdownTimeout:      c.ShutdownTimeout,
		WebPort:              c.WebPort,
		WebEnabled:           c.WebEnabled,
		CORSAllowedOrigins:   c.CORSAllowedOrigins,
		CORSAllowedMethods:   c.CORSAllowedMethods,
		CORSAllowedHeaders:   c.CORSAllowedHeaders,
		CORSMaxAge:           c.CORSMaxAge,
		PProfEnabled:         c.PProfEnabled,
		HealthCheckTimeout:   c.HealthCheckTimeout,
		TLSEnabled:           c.TLSEnabled,
		TLSMinVersion:        c.TLSMinVersion,
		JWTIssuer:            c.JWTIssuer,
		JWTExpiry:            c.JWTExpiry,
		JWTRotationHours:     c.JWTRotationHours,
		RateLimitMCP:         c.RateLimitMCP,
		RateLimitIDE:         c.RateLimitIDE,
		RateLimitSession:     c.RateLimitSession,
		RateLimitWindow:      c.RateLimitWindow,
		RateLimitBurstFactor: c.RateLimitBurstFactor,
		CacheTTLRules:        c.CacheTTLRules,
		CacheTTLDocs:         c.CacheTTLDocs,
		CacheTTLSearch:       c.CacheTTLSearch,
		EnableValidation:     c.EnableValidation,
		EnableMetrics:        c.EnableMetrics,
		EnableAuditLogging:   c.EnableAuditLogging,
		EnableCache:          c.EnableCache,
		AuditBufferSize:      c.AuditBufferSize,
		AuditFlushInterval:   c.AuditFlushInterval,

		CircuitBreakerEnabled:          c.CircuitBreakerEnabled,
		CircuitBreakerFailureThreshold: c.CircuitBreakerFailureThreshold,
		CircuitBreakerSuccessThreshold: c.CircuitBreakerSuccessThreshold,
		CircuitBreakerTimeout:          c.CircuitBreakerTimeout,
		CircuitBreakerMaxRequests:      c.CircuitBreakerMaxRequests,
		CircuitBreakerInterval:         c.CircuitBreakerInterval,

		ProductionMode: c.ProductionMode,
	}
}
