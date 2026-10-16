package config

import "testing"

// TestEffectiveCORSOrigins covers Spec 12 §3.3.2: production wildcard/empty is
// a hard error (never a substitution); non-production substitutes restrictive
// localhost defaults and reports the substitution so the caller can warn.
func TestEffectiveCORSOrigins(t *testing.T) {
	cases := []struct {
		name         string
		origins      []string
		production   bool
		wantErr      bool
		wantSubst    bool
		wantContains string
	}{
		{
			name:         "explicit origins pass through",
			origins:      []string{"https://app.example.com", "https://admin.example.com"},
			wantContains: "https://app.example.com",
		},
		{
			name:       "empty in production fails closed",
			origins:    nil,
			production: true,
			wantErr:    true,
		},
		{
			name:       "wildcard in production fails closed",
			origins:    []string{"*"},
			production: true,
			wantErr:    true,
		},
		{
			name:         "empty outside production substitutes dev defaults",
			origins:      nil,
			wantSubst:    true,
			wantContains: "http://localhost:*",
		},
		{
			name:         "wildcard outside production substitutes dev defaults",
			origins:      []string{"*"},
			wantSubst:    true,
			wantContains: "https://localhost:*",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{CORSAllowedOrigins: tc.origins, ProductionMode: tc.production}
			got, substituted, err := cfg.EffectiveCORSOrigins()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got origins=%v substituted=%v", got, substituted)
				}
				if got != nil {
					t.Fatalf("on error origins must be nil, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if substituted != tc.wantSubst {
				t.Fatalf("substituted = %v, want %v", substituted, tc.wantSubst)
			}
			if tc.wantContains != "" {
				found := false
				for _, o := range got {
					if o == tc.wantContains {
						found = true
					}
				}
				if !found {
					t.Fatalf("origins %v do not contain %q", got, tc.wantContains)
				}
			}
		})
	}
}

// TestEffectiveCORSOriginsDoesNotAliasCaller guards against the returned slice
// sharing backing storage with the config, which would let a caller mutate the
// validated posture.
func TestEffectiveCORSOriginsDoesNotAliasCaller(t *testing.T) {
	cfg := &Config{CORSAllowedOrigins: []string{"https://app.example.com"}}
	got, _, err := cfg.EffectiveCORSOrigins()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got[0] = "https://evil.example.com"
	if cfg.CORSAllowedOrigins[0] != "https://app.example.com" {
		t.Fatal("EffectiveCORSOrigins must not alias the caller's slice")
	}
}
