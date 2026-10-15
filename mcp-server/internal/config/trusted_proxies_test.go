package config

import (
	"strings"
	"testing"
)

// TestValidateTrustedProxiesStance covers the Spec 12 §3.4 configuration
// boundary: valid CIDRs and bare IPs pass, an empty list (trust no proxy) is
// the default, and a malformed entry fails closed rather than silently
// narrowing to "trust nothing".
func TestValidateTrustedProxiesStance(t *testing.T) {
	t.Run("empty trusts no proxy and passes", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.TrustedProxies = nil
		if err := cfg.Validate(); err != nil {
			t.Fatalf("empty TRUSTED_PROXIES must pass: %v", err)
		}
	})

	t.Run("valid cidr and bare ip pass", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.TrustedProxies = []string{"10.0.0.0/8", "192.0.2.5", "2001:db8::/32"}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("valid TRUSTED_PROXIES must pass: %v", err)
		}
	})

	t.Run("malformed entry fails closed", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.TrustedProxies = []string{"not-a-cidr"}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("malformed TRUSTED_PROXIES must fail validation")
		}
		if !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
			t.Fatalf("error must name TRUSTED_PROXIES, got %v", err)
		}
	})
}
