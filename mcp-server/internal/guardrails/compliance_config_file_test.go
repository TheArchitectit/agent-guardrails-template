package guardrails

// Spec 06 Sec. 2.3 — the `compliance_reports:` configuration file.
//
// Before this file the documented block had NO reader: `ComplianceConfig` and
// `DefaultComplianceConfig()` had zero non-test call sites, so an operator who
// wrote the documented YAML changed nothing and the three compliance tools ran
// on hardcoded defaults. These tests drive the REAL loader, and each one asserts
// a property that would be satisfied by a loader that merely parsed.
//
// They also pin the honest limitation: a VERBATIM copy of the spec's own example
// is REFUSED, because it declares `sections:` (per-section selection, not
// implemented) and the evidence source `config_history` (no producer anywhere in
// this tree). Refusing by name is the point — a file that declares a control the
// server does not apply must not load clean.

import (
	"strings"
	"testing"
)

// supportedShape is the spec's documented block with the two things this tree
// does not implement removed. This is what a deployment can actually write.
const supportedShape = `compliance_reports:
  eu_ai_act:
    enabled: true
    output_format: "json"
    evidence_sources:
      - "audit_logs"
      - "guardrail_events"
    export_path: "reports/compliance/"
  nist_rmf:
    enabled: true
    output_format: "json"
    evidence_sources:
      - "metrics"
    export_path: "reports/compliance/"
`

// specExampleVerbatim is the spec's own Sec. 2.3 example, copied character for
// character. Every key is upstream text, not a fixture of mine.
const specExampleVerbatim = `compliance_reports:
  eu_ai_act:
    enabled: true
    output_format: "json"
    evidence_sources:
      - "audit_logs"
      - "guardrail_events"
      - "config_history"
    sections:
      - "risk_assessment"
      - "technical_documentation"
      - "logging_traceability"
      - "human_oversight"
      - "transparency"
    export_path: "reports/compliance/"
  nist_rmf:
    enabled: true
    output_format: "json"
    sections:
      - "govern"
      - "map"
      - "measure"
      - "manage"
    export_path: "reports/compliance/"
`

// writeComplianceTemp reuses the package's existing writeTemp(t, name, body)
// helper from config_file_wiring_test.go rather than redeclaring it; this
// wrapper only supplies the fixture name.
func writeComplianceTemp(t *testing.T, body string) string {
	t.Helper()
	return writeTemp(t, "compliance.yml", body)
}

func TestComplianceConfigFile_SupportedShapeBindsEveryKey(t *testing.T) {
	cfg, err := LoadComplianceConfigFromYAML(writeComplianceTemp(t, supportedShape))
	if err != nil {
		t.Fatalf("the documented supported shape failed to load: %v", err)
	}
	if !cfg.Enabled {
		t.Fatalf("cfg.Enabled = false, want true")
	}
	if got := strings.Join(cfg.ActiveFrameworks, ","); got != "eu_ai_act,nist_rmf" {
		t.Fatalf("ActiveFrameworks = %q, want eu_ai_act,nist_rmf", got)
	}
	if cfg.ReportFormat != string(FormatJSON) {
		t.Fatalf("ReportFormat = %q, want json", cfg.ReportFormat)
	}
	if cfg.ExportPath != "reports/compliance/" {
		t.Fatalf("ExportPath = %q, want reports/compliance/", cfg.ExportPath)
	}
	// Sources are unioned across the enabled sections, deduplicated, sorted.
	if got := strings.Join(cfg.EvidenceSources, ","); got != "audit_logs,guardrail_events,metrics" {
		t.Fatalf("EvidenceSources = %q, want audit_logs,guardrail_events,metrics", got)
	}
}

// The spec's own example must NOT load clean. Two declared things this tree does
// not implement: per-section selection, and the `config_history` evidence class.
func TestComplianceConfigFile_SpecExampleIsRefusedNamingSections(t *testing.T) {
	_, err := LoadComplianceConfigFromYAML(writeComplianceTemp(t, specExampleVerbatim))
	if err == nil {
		t.Fatalf("a verbatim copy of the spec's own example loaded clean; it declares `sections:`, which this server does not implement")
	}
	if !strings.Contains(err.Error(), "sections") {
		t.Fatalf("refusal does not name `sections`: %v", err)
	}
}

// With `sections:` removed, the next unimplemented declaration must be refused
// by name — the config_history evidence class, because nothing in this tree
// observes a configuration snapshot.
func TestComplianceConfigFile_SpecExampleIsRefusedNamingConfigHistory(t *testing.T) {
	body := strings.Replace(specExampleVerbatim, `    sections:
      - "risk_assessment"
      - "technical_documentation"
      - "logging_traceability"
      - "human_oversight"
      - "transparency"
`, "", 1)
	body = strings.Replace(body, `    sections:
      - "govern"
      - "map"
      - "measure"
      - "manage"
`, "", 1)
	_, err := LoadComplianceConfigFromYAML(writeComplianceTemp(t, body))
	if err == nil {
		t.Fatalf("a file declaring evidence_sources: [config_history] loaded clean; no code path in this tree observes a configuration snapshot")
	}
	if !strings.Contains(err.Error(), "config_history") {
		t.Fatalf("refusal does not name config_history: %v", err)
	}
}

func TestComplianceConfigFile_RefusalShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"empty file", "", "empty"},
		{"no section at all", "other_key: 1\n", "compliance_reports"},
		{"unknown key", "compliance_reports:\n  eu_ai_act:\n    enabled: true\n    wibble: 2\n", "wibble"},
		{"unknown framework", "compliance_reports:\n  nist_ai_rmf:\n    enabled: true\n", "nist_ai_rmf"},
		{"output_format pdf", "compliance_reports:\n  eu_ai_act:\n    enabled: true\n    output_format: pdf\n", "pdf"},
		{"unknown output_format", "compliance_reports:\n  eu_ai_act:\n    enabled: true\n    output_format: xml\n", "xml"},
		{"no enabled framework", "compliance_reports:\n  eu_ai_act:\n    enabled: false\n", "no enabled framework"},
		{"unknown evidence source", "compliance_reports:\n  eu_ai_act:\n    enabled: true\n    evidence_sources: [\"config_history\"]\n", "config_history"},
		{"empty evidence source", "compliance_reports:\n  eu_ai_act:\n    enabled: true\n    evidence_sources: [\"\"]\n", "empty evidence source"},
		{"both shapes", "compliance_reports:\n  eu_ai_act:\n    enabled: true\nactive_frameworks: [eu_ai_act]\n", "declare one"},
		{"conflicting export paths", "compliance_reports:\n  eu_ai_act:\n    enabled: true\n    export_path: /a\n  nist_rmf:\n    enabled: true\n    export_path: /b\n", "two export paths"},
		{"conflicting formats", "compliance_reports:\n  eu_ai_act:\n    enabled: true\n    output_format: json\n  nist_rmf:\n    enabled: true\n    output_format: markdown\n", "two output formats"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadComplianceConfigFromYAML(writeComplianceTemp(t, tc.body))
			if err == nil {
				t.Fatalf("expected a refusal naming %q, got a clean load", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal = %q, want it to name %q", err.Error(), tc.want)
			}
		})
	}
}

// A disabled section is dropped, not loaded-and-ignored: the operator removing a
// framework must actually remove it.
func TestComplianceConfigFile_DisabledFrameworkIsNotActivated(t *testing.T) {
	body := `compliance_reports:
  eu_ai_act:
    enabled: false
  nist_rmf:
    enabled: true
`
	cfg, err := LoadComplianceConfigFromYAML(writeComplianceTemp(t, body))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AllowsFramework("eu_ai_act") {
		t.Fatalf("eu_ai_act is activated although its section declares enabled: false")
	}
	if !cfg.AllowsFramework("nist_rmf") {
		t.Fatalf("nist_rmf is not activated although its section declares enabled: true")
	}
}

// The flat form the Go type has always carried keeps loading, so a programmatic
// or hand-written flat file is not broken by the new nested form.
func TestComplianceConfigFile_FlatShapeStillLoads(t *testing.T) {
	body := `enabled: true
active_frameworks: [iso_42001]
report_format: markdown
evidence_sources: [audit_logs]
export_path: /tmp/out
`
	cfg, err := LoadComplianceConfigFromYAML(writeComplianceTemp(t, body))
	if err != nil {
		t.Fatalf("flat shape failed to load: %v", err)
	}
	if got := strings.Join(cfg.ActiveFrameworks, ","); got != "iso_42001" {
		t.Fatalf("ActiveFrameworks = %q, want iso_42001", got)
	}
	if cfg.ReportFormat != string(FormatMarkdown) {
		t.Fatalf("ReportFormat = %q, want markdown", cfg.ReportFormat)
	}
	if cfg.ExportPath != "/tmp/out" {
		t.Fatalf("ExportPath = %q, want /tmp/out", cfg.ExportPath)
	}
}

// The zero-configuration path must remain permissive: a deployment with no
// `COMPLIANCE_CONFIG_FILE` never calls SetComplianceConfig, so the server holds a
// nil config and behaves exactly as it did before this change — every framework
// the database carries is reportable. DefaultComplianceConfig() is a different
// thing: it is the shipped VALUE, and it names an explicit scope.
func TestComplianceConfigFile_NilConfigAllowsEveryFrameworkAndDefaultScopes(t *testing.T) {
	// A nil config is the no-file path: no scope restriction at all.
	var nilCfg *ComplianceConfig
	for _, fw := range []string{"eu_ai_act", "nist_rmf", "iso_42001"} {
		if !nilCfg.AllowsFramework(fw) {
			t.Fatalf("a nil config must not scope anything, but it refused %q", fw)
		}
	}

	// The shipped default value is valid and scopes to the frameworks it names;
	// a framework it does not name is refused, which is the whole point of the
	// file (an operator who scopes the audit must not get another framework).
	cfg := DefaultComplianceConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultComplianceConfig no longer validates: %v", err)
	}
	for _, fw := range cfg.ActiveFrameworks {
		if !cfg.AllowsFramework(fw) {
			t.Fatalf("the default config does not allow the framework it names: %q", fw)
		}
	}
	if cfg.AllowsFramework("iso_42001") {
		t.Fatalf("the default config allows iso_42001, which its ActiveFrameworks does not name; " +
			"either the default changed or AllowsFramework ignores the active set")
	}
}

// The documented evidence-source list is the set of classes the tree can
// observe, and it must NOT contain the unobservable one.
func TestComplianceConfigFile_ObservableSourcesExcludeConfigHistory(t *testing.T) {
	for _, s := range ConfigurableComplianceEvidenceSources() {
		if s == "config_history" {
			t.Fatalf("config_history is offered as an observable evidence source, but no code path in this tree records a configuration snapshot")
		}
	}
	for _, s := range DefaultComplianceConfig().EvidenceSources {
		if s == "config_history" {
			t.Fatalf("the shipped default names config_history, which nothing can collect")
		}
	}
}
