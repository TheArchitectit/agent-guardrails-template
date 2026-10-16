package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeIngressRecord writes a deployment preflight record (Spec 17 R17-02) with
// the restricted mode the gate requires and returns its path.
func writeIngressRecord(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write preflight record %s: %v", name, err)
	}
	return path
}

// validIngressRecord returns a record body that proves every result for the
// given profile, with a full 40-hex commit and at least one negative probe.
func validIngressRecord(profile string) string {
	return `{
  "contract_version": "` + ingressContractVersion + `",
  "profile": "` + profile + `",
  "commit": "` + strings.Repeat("a", 40) + `",
  "tls_termination": {"proven": true, "detail": "TLS 1.3 handshake verified from the public client"},
  "direct_backend_denied": {"proven": true, "detail": "direct TCP to the backend port from the public zone refused"},
  "private_ingress": {"proven": true, "detail": "only the overlay interface is reachable"},
  "encrypted_transport": {"proven": true, "detail": "peer-verified TLS terminates at the overlay boundary"},
  "negative_probes": ["plaintext public request refused", "direct backend connection refused"]
}`
}

// TestValidateIngressPosture covers Spec 17 R17-02: a nonlocal profile must
// prove its ingress posture from a commit-bound record; every way of presenting
// weaker or absent evidence fails closed.
func TestValidateIngressPosture(t *testing.T) {
	t.Run("local passes without a record", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileLocal
		if err := cfg.ValidateIngressPosture(); err != nil {
			t.Fatalf("local must not require a preflight record: %v", err)
		}
	})

	t.Run("empty profile defaults to local and passes", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = ""
		if err := cfg.ValidateIngressPosture(); err != nil {
			t.Fatalf("empty profile must default to local: %v", err)
		}
	})

	t.Run("public without a record fails", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		err := cfg.ValidateIngressPosture()
		if err == nil {
			t.Fatal("public without a preflight record must fail")
		}
		if !strings.Contains(err.Error(), "R17-02") || !strings.Contains(err.Error(), "DEPLOYMENT_PREFLIGHT_FILE") {
			t.Fatalf("error must name R17-02 and the variable, got %v", err)
		}
	})

	t.Run("tailnet without a record fails", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileTailnet
		if err := cfg.ValidateIngressPosture(); err == nil {
			t.Fatal("tailnet without a preflight record must fail")
		}
	})

	t.Run("public with a proving record passes", func(t *testing.T) {
		dir := t.TempDir()
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.DeploymentPreflightFile = writeIngressRecord(t, dir, "preflight.json", validIngressRecord(DeploymentProfilePublic))
		if err := cfg.ValidateIngressPosture(); err != nil {
			t.Fatalf("a fully proven public record must pass: %v", err)
		}
	})

	t.Run("tailnet with a proving record passes", func(t *testing.T) {
		dir := t.TempDir()
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileTailnet
		cfg.DeploymentPreflightFile = writeIngressRecord(t, dir, "preflight.json", validIngressRecord(DeploymentProfileTailnet))
		if err := cfg.ValidateIngressPosture(); err != nil {
			t.Fatalf("a fully proven tailnet record must pass: %v", err)
		}
	})

	// The named negatives: each is a way of presenting evidence that is not
	// evidence.
	negatives := []struct {
		name    string
		profile string
		mutate  func(string) string
		want    string
	}{
		{
			name:    "public record missing tls_termination",
			profile: DeploymentProfilePublic,
			mutate:  func(s string) string { return removeJSONField(s, "tls_termination") },
			want:    "tls_termination",
		},
		{
			name:    "public tls_termination not proven",
			profile: DeploymentProfilePublic,
			mutate: func(s string) string {
				return replaceJSON(s, `"tls_termination": {"proven": true`, `"tls_termination": {"proven": false`)
			},
			want: "NOT proven",
		},
		{
			name:    "public tls_termination proven without detail",
			profile: DeploymentProfilePublic,
			mutate: func(s string) string {
				return replaceJSON(s, `"tls_termination": {"proven": true, "detail": "TLS 1.3 handshake verified from the public client"}`, `"tls_termination": {"proven": true, "detail": "   "}`)
			},
			want: "no detail",
		},
		{
			name:    "public record missing direct_backend_denied",
			profile: DeploymentProfilePublic,
			mutate:  func(s string) string { return removeJSONField(s, "direct_backend_denied") },
			want:    "direct_backend_denied",
		},
		{
			name:    "tailnet record missing encrypted_transport",
			profile: DeploymentProfileTailnet,
			mutate:  func(s string) string { return removeJSONField(s, "encrypted_transport") },
			want:    "encrypted_transport",
		},
		{
			name:    "tailnet record missing private_ingress",
			profile: DeploymentProfileTailnet,
			mutate:  func(s string) string { return removeJSONField(s, "private_ingress") },
			want:    "private_ingress",
		},
		{
			name:    "record for the wrong profile",
			profile: DeploymentProfilePublic,
			mutate:  func(s string) string { return replaceJSON(s, `"profile": "public"`, `"profile": "tailnet"`) },
			want:    "produced for profile",
		},
		{
			name:    "short commit",
			profile: DeploymentProfilePublic,
			mutate: func(s string) string {
				return replaceJSON(s, `"commit": "`+strings.Repeat("a", 40)+`"`, `"commit": "abc123"`)
			},
			want: "full 40-hex commit",
		},
		{
			name:    "no negative probes",
			profile: DeploymentProfilePublic,
			mutate: func(s string) string {
				return replaceJSON(s, `"negative_probes": ["plaintext public request refused", "direct backend connection refused"]`, `"negative_probes": []`)
			},
			want: "no negative probes",
		},
		{
			name:    "empty negative probe",
			profile: DeploymentProfilePublic,
			mutate: func(s string) string {
				return replaceJSON(s, `"negative_probes": ["plaintext public request refused", "direct backend connection refused"]`, `"negative_probes": ["ok", "  "]`)
			},
			want: "empty negative probe",
		},
		{
			name:    "wrong contract version",
			profile: DeploymentProfilePublic,
			mutate: func(s string) string {
				return replaceJSON(s, `"contract_version": "`+ingressContractVersion+`"`, `"contract_version": "guardrail-ingress-preflight/v2"`)
			},
			want: "contract_version",
		},
		{
			name:    "unknown field is not reinterpreted",
			profile: DeploymentProfilePublic,
			mutate: func(s string) string {
				return replaceJSON(s, `"negative_probes":`, `"tls_terminated": true, "negative_probes":`)
			},
			want: "not a valid",
		},
		{
			name:    "malformed JSON",
			profile: DeploymentProfilePublic,
			mutate:  func(s string) string { return "{not json" },
			want:    "not a valid",
		},
		{
			name:    "empty file",
			profile: DeploymentProfilePublic,
			mutate:  func(s string) string { return "   \n" },
			want:    "empty",
		},
	}

	for _, tt := range negatives {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := validMinimalConfig()
			cfg.DeploymentProfile = tt.profile
			cfg.DeploymentPreflightFile = writeIngressRecord(t, dir, "preflight.json", tt.mutate(validIngressRecord(tt.profile)))
			err := cfg.ValidateIngressPosture()
			if err == nil {
				t.Fatalf("must fail closed")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

// TestValidateIngressPostureRejectsWeakRecordFile proves the record's own
// custody is checked: a record other identities can rewrite, a directory, and a
// missing file are all refused.
func TestValidateIngressPostureRejectsWeakRecordFile(t *testing.T) {
	t.Run("group/world-writable record is not evidence", func(t *testing.T) {
		dir := t.TempDir()
		path := writeIngressRecord(t, dir, "preflight.json", validIngressRecord(DeploymentProfilePublic))
		if err := os.Chmod(path, 0o666); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.DeploymentPreflightFile = path
		err := cfg.ValidateIngressPosture()
		if err == nil || !strings.Contains(err.Error(), "group/world writable") {
			t.Fatalf("group/world-writable record err = %v, want a writable-record refusal", err)
		}
	})

	t.Run("missing record fails closed", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.DeploymentPreflightFile = filepath.Join(t.TempDir(), "absent.json")
		if err := cfg.ValidateIngressPosture(); err == nil {
			t.Fatal("a missing preflight record must fail")
		}
	})

	t.Run("a directory is not a record", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.DeploymentPreflightFile = t.TempDir()
		err := cfg.ValidateIngressPosture()
		if err == nil || !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("directory record err = %v, want a regular-file refusal", err)
		}
	})
}

// TestValidateCouplesIngressPostureToStartup proves the gate runs inside
// Config.Validate (the startup gate), not only in the standalone helper: a
// public config that satisfies every other rule still fails startup without a
// proving record, and passes once one is supplied. It also pins that
// TLS_ENABLED is not required for the public profile — the proxy terminates
// TLS, and the record is what proves it.
func TestValidateCouplesIngressPostureToStartup(t *testing.T) {
	cfg := validMinimalConfig()
	cfg.DeploymentProfile = DeploymentProfilePublic
	cfg.MCPBindHost = "10.0.0.5"
	cfg.WebEnabled = false
	cfg.DBSSLMode = "verify-full"
	cfg.RedisUseTLS = true
	cfg.TrustedProxies = []string{"10.0.0.1"}
	cfg.CORSAllowedOrigins = []string{"https://guardrails.example.com"}
	cfg.DBPassword = "operator-db-pass-01"
	cfg.RedisPassword = "operator-redis-pass-01"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate must fail on a public profile with no preflight record")
	} else if !strings.Contains(err.Error(), "R17-02") {
		t.Fatalf("Validate error must name R17-02, got %v", err)
	}

	dir := t.TempDir()
	cfg.DeploymentPreflightFile = writeIngressRecord(t, dir, "preflight.json", validIngressRecord(DeploymentProfilePublic))
	if err := cfg.Validate(); err != nil {
		t.Fatalf("public config with a proving preflight record must pass Validate (TLS_ENABLED stays false: the proxy terminates TLS), got %v", err)
	}
}

// TestIngressPreflightPath covers the diagnostics accessor: empty when unset,
// absolute where possible.
func TestIngressPreflightPath(t *testing.T) {
	var nilCfg *Config
	if got := nilCfg.IngressPreflightPath(); got != "" {
		t.Errorf("nil config path = %q, want empty", got)
	}
	cfg := validMinimalConfig()
	if got := cfg.IngressPreflightPath(); got != "" {
		t.Errorf("unset path = %q, want empty", got)
	}
	cfg.DeploymentPreflightFile = "preflight.json"
	if got := cfg.IngressPreflightPath(); !filepath.IsAbs(got) || !strings.HasSuffix(got, "preflight.json") {
		t.Errorf("path = %q, want an absolute path ending in preflight.json", got)
	}
}

// removeJSONField deletes a top-level field (and its trailing comma) from the
// fixture record.
func removeJSONField(s, field string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.Contains(l, `"`+field+`"`) {
			continue
		}
		out = append(out, l)
	}
	joined := strings.Join(out, "\n")
	// Repair a trailing comma left before the closing brace.
	return strings.Replace(joined, ",\n}", "\n}", 1)
}

// replaceJSON performs a single exact replacement in the fixture record.
func replaceJSON(s, old, new string) string {
	return strings.Replace(s, old, new, 1)
}
