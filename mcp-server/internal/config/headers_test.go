package config

import "testing"

// TestEmitsHSTS covers Spec 12 §3.6 / Spec 17 R17-03: HSTS is emitted only for
// a verified public HTTPS origin — TLS enabled AND the effective deployment
// profile is `public`. Every other combination must not emit the header, so the
// header cannot be used to assert a posture the operator never declared.
func TestEmitsHSTS(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		tls     bool
		want    bool
	}{
		{"public + TLS emits", DeploymentProfilePublic, true, true},
		{"public without TLS does not emit", DeploymentProfilePublic, false, false},
		{"local with TLS does not emit", DeploymentProfileLocal, true, false},
		{"tailnet with TLS does not emit", DeploymentProfileTailnet, true, false},
		{"local without TLS does not emit", DeploymentProfileLocal, false, false},
		{"empty profile (defaults local) with TLS does not emit", "", true, false},
		{"unknown profile with TLS does not emit", "staging", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{DeploymentProfile: tt.profile, TLSEnabled: tt.tls}
			if got := cfg.EmitsHSTS(); got != tt.want {
				t.Fatalf("EmitsHSTS() = %v, want %v (profile=%q tls=%v)", got, tt.want, tt.profile, tt.tls)
			}
		})
	}
}

// TestHSTSHeaderValueIsBounded pins the emitted policy: a positive max-age with
// includeSubDomains and no preload assertion (preload is a commitment the
// operator has not made).
func TestHSTSHeaderValueIsBounded(t *testing.T) {
	if HSTSHeaderValue == "" {
		t.Fatal("HSTSHeaderValue must not be empty")
	}
	if got := HSTSHeaderValue; got != "max-age=31536000; includeSubDomains" {
		t.Fatalf("HSTSHeaderValue = %q; update this pin deliberately, it is a client-visible commitment", got)
	}
}
