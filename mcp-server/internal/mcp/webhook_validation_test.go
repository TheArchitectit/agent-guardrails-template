package mcp

import (
	"strings"
	"testing"
)

// TestValidateWebhookURL covers the SSRF guard: the dispatcher POSTs to
// whatever URL is stored, so non-public and non-http targets must be refused
// at configuration time.
func TestValidateWebhookURL(t *testing.T) {
	rejected := []struct {
		name string
		url  string
	}{
		{"loopback by name", "http://localhost/hook"},
		{"loopback subdomain", "http://evil.localhost/hook"},
		{"loopback IPv4", "http://127.0.0.1/hook"},
		{"loopback IPv4 range", "http://127.0.0.2/hook"},
		{"loopback IPv6", "http://[::1]/hook"},
		{"IPv4-mapped loopback", "http://[::ffff:127.0.0.1]/hook"},
		{"private IPv4 10/8", "http://10.0.0.5/hook"},
		{"private IPv4 192.168/16", "http://192.168.1.1/hook"},
		{"private IPv4 172.16/12", "http://172.16.0.1/hook"},
		{"link-local metadata", "http://169.254.169.254/latest/meta-data/"},
		{"IPv4-mapped metadata", "http://[::ffff:169.254.169.254]/hook"},
		{"link-local IPv6", "http://[fe80::1]/hook"},
		{"unique-local IPv6", "http://[fd00::1]/hook"},
		{"unspecified", "http://0.0.0.0/hook"},
		{"file scheme", "file:///etc/passwd"},
		{"gopher scheme", "gopher://example.com/"},
		{"no host", "http:///hook"},
		{"not a url", "://nonsense"},
	}

	for _, tc := range rejected {
		t.Run("reject "+tc.name, func(t *testing.T) {
			if err := validateWebhookURL(tc.url); err == nil {
				t.Fatalf("validateWebhookURL(%q) returned nil, want an error", tc.url)
			}
		})
	}

	t.Run("accept public https", func(t *testing.T) {
		// Uses a literal public IP so the test does not depend on DNS.
		if err := validateWebhookURL("https://93.184.216.34/hook"); err != nil {
			t.Fatalf("public https URL rejected: %v", err)
		}
	})

	t.Run("error message names the reason", func(t *testing.T) {
		err := validateWebhookURL("http://169.254.169.254/latest/meta-data/")
		if err == nil {
			t.Fatal("expected an error for the metadata endpoint")
		}
		if !strings.Contains(err.Error(), "non-public") {
			t.Errorf("error should explain the refusal, got %q", err.Error())
		}
	})
}