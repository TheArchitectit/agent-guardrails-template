package config

import (
	"strings"
	"testing"
	"time"
)

// TestProfileBindingLocal covers the local profile bind rules: loopback binds
// pass; wildcard and non-loopback binds fail closed unless the isolated local
// bridge attestation is provided (Spec 17 R17-01/R17-02).
func TestProfileBindingLocal(t *testing.T) {
	tests := []struct {
		name        string
		bind        string
		attestation string
		wantErr     bool
	}{
		{"loopback IPv4 passes", "127.0.0.1:8080", "", false},
		{"loopback IPv6 passes", "[::1]:8080", "", false},
		{"localhost name passes", "localhost:8081", "", false},
		{"loopback range passes", "127.9.9.9:8080", "", false},
		{"wildcard without attestation fails", "0.0.0.0:8080", "", true},
		{"IPv6 wildcard without attestation fails", "[::]:8080", "", true},
		{"empty host without attestation fails", ":8080", "", true},
		{"wildcard with attestation passes", "0.0.0.0:8080", "isolated-local-bridge:host-loopback-publish", false},
		{"non-loopback unicast fails", "192.168.1.10:8080", "", true},
		{"non-loopback unicast with attestation still fails", "192.168.1.10:8080", "isolated-local-bridge:host-loopback-publish", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DeploymentProfile:         DeploymentProfileLocal,
				DeploymentBindAttestation: tt.attestation,
			}
			err := cfg.ValidateDeploymentBinding(tt.bind)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateDeploymentBinding(%q) error = %v, wantErr %v", tt.bind, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), "local") {
				t.Errorf("error should name the local profile, got %v", err)
			}
		})
	}
}

// TestProfileBindingPublic covers the public profile: it requires explicit
// proxy configuration (TRUSTED_PROXIES) regardless of bind address.
func TestProfileBindingPublic(t *testing.T) {
	tests := []struct {
		name      string
		bind      string
		proxies   []string
		wantErr   bool
		errSubstr string
	}{
		{"without proxy config fails", "0.0.0.0:8080", nil, true, "TRUSTED_PROXIES"},
		{"without proxy config fails on loopback too", "127.0.0.1:8080", nil, true, "TRUSTED_PROXIES"},
		{"with proxy config passes wildcard bind", "0.0.0.0:8080", []string{"10.0.0.5"}, false, ""},
		{"with proxy config passes specific bind", "10.0.1.2:8080", []string{"10.0.0.5", "10.0.0.6"}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DeploymentProfile: DeploymentProfilePublic,
				TrustedProxies:    tt.proxies,
			}
			err := cfg.ValidateDeploymentBinding(tt.bind)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateDeploymentBinding(%q) error = %v, wantErr %v", tt.bind, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("error = %v, want containing %q", err, tt.errSubstr)
			}
		})
	}
}

// TestProfileBindingTailnet covers the tailnet rule: a wildcard bind is never
// private by name alone; a named interface bind is required.
func TestProfileBindingTailnet(t *testing.T) {
	cfg := &Config{DeploymentProfile: DeploymentProfileTailnet}

	if err := cfg.ValidateDeploymentBinding("0.0.0.0:8080"); err == nil {
		t.Error("tailnet must reject wildcard bind")
	}
	if err := cfg.ValidateDeploymentBinding("100.64.0.5:8080"); err != nil {
		t.Errorf("tailnet must accept a named private interface bind: %v", err)
	}
}

// TestProfileBindingValidationRules covers profile normalization and the
// validate-at-startup wiring through Config.Validate.
func TestProfileBindingValidationRules(t *testing.T) {
	// Unknown profile name is rejected.
	if _, err := NormalizeDeploymentProfile("staging"); err == nil {
		t.Error("unknown profile must be rejected")
	}

	// Empty profile defaults to local.
	p, err := NormalizeDeploymentProfile("")
	if err != nil || p != DeploymentProfileLocal {
		t.Errorf("empty profile = %q, err %v; want local, nil", p, err)
	}

	// A local-profile server with a wildcard MCP bind and no attestation
	// fails Load()-level validation.
	cfg := validMinimalConfig()
	cfg.MCPBindHost = "0.0.0.0"
	cfg.WebEnabled = false
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "DEPLOYMENT_BIND_ATTESTATION") {
		t.Errorf("Validate() with local+wildcard bind and no attestation must fail, got %v", err)
	}

	// Loopback bind passes startup validation.
	cfg.MCPBindHost = "127.0.0.1"
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() with local+loopback bind must pass, got %v", err)
	}
}

// validMinimalConfig returns a config that passes every other Validate rule
// so profile-binding failures are isolated.
func validMinimalConfig() *Config {
	return &Config{
		MCPPort:                        8080,
		MCPBindHost:                    "127.0.0.1",
		LogLevel:                       "info",
		ShutdownTimeout:                30 * time.Second,
		RequestTimeout:                 30 * time.Second,
		DBConnectTimeout:               10 * time.Second,
		DBMaxOpenConns:                 25,
		DBMaxIdleConns:                 5,
		RedisPoolSize:                  10,
		RedisMinIdleConns:              2,
		RateLimitMCP:                   1000,
		RateLimitIDE:                   500,
		RateLimitSession:               100,
		RateLimitBurstFactor:           1.5,
		DBSSLMode:                      "require",
		AuditBufferSize:                1000,
		CircuitBreakerFailureThreshold: 5,
		CircuitBreakerSuccessThreshold: 2,
		CircuitBreakerTimeout:          30 * time.Second,
		CircuitBreakerMaxRequests:      3,
		CircuitBreakerInterval:         10 * time.Second,
		CORSAllowedOrigins:             []string{"*"},
		JWTSecret:                      "abcdefghijklmnopqrstuvwxyz123456",
		MCPAPIKey:                      "AbCdEfGhIjKlMnOpQrStUvWxYz123456",
		IDEAPIKey:                      "AbCdEfGhIjKlMnOpQrStUvWxYz123456",
	}
}

// TestBindHostExtraction covers listener address parsing used by the
// profile binding checks.
func TestBindHostExtraction(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"127.0.0.1:8080", "127.0.0.1"},
		{"[::1]:8080", "::1"},
		{":8080", ""},
		{"0.0.0.0:80", "0.0.0.0"},
		{"localhost", "localhost"},
	}

	for _, tt := range tests {
		if got := BindHostFromAddr(tt.addr); got != tt.want {
			t.Errorf("BindHostFromAddr(%q) = %q, want %q", tt.addr, got, tt.want)
		}
	}
}
