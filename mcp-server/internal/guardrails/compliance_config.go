// Package guardrails provides tools for mapping system features to regulatory compliance frameworks.
package guardrails

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComplianceConfig defines the configuration for compliance reporting and mapping.
type ComplianceConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	ActiveFrameworks  []string `yaml:"active_frameworks" json:"active_frameworks"`
	ReportFormat      string   `yaml:"report_format" json:"report_format"` // "json", "markdown"
	EvidenceSources   []string `yaml:"evidence_sources" json:"evidence_sources"`
	ExportPath        string   `yaml:"export_path" json:"export_path"`
	RequirementDbPath string   `yaml:"requirement_db_path" json:"requirement_db_path"`
}

// DefaultComplianceConfig returns a secure base configuration for compliance.
//
// The evidence-source list names only sources this tree can OBSERVE. The spec's
// Sec. 2.3 example also lists `config_history`, and no code path in this tree
// records a configuration snapshot (the string occurs nowhere else), so it is
// recorded in the gap-spec STATUS ledger as an unimplemented evidence class
// rather than shipped in a default list nothing can satisfy.
func DefaultComplianceConfig() *ComplianceConfig {
	return &ComplianceConfig{
		Enabled:           true,
		ActiveFrameworks:  []string{"eu_ai_act", "nist_rmf"},
		ReportFormat:      "json",
		EvidenceSources:   []string{"audit_logs", "guardrail_events", "metrics"},
		ExportPath:        "reports/compliance/",
		RequirementDbPath: "internal/guardrails/compliance_requirements.json",
	}
}

// Validate checks the config for errors.
func (c *ComplianceConfig) Validate() error {
	if ReportFormat(c.ReportFormat) != FormatJSON && ReportFormat(c.ReportFormat) != FormatMarkdown {
		return fmt.Errorf("compliance report_format must be 'json' or 'markdown'")
	}
	if len(c.ActiveFrameworks) == 0 && c.Enabled {
		return fmt.Errorf("compliance enabled but no active_frameworks specified")
	}
	return nil
}

// Framework is a label for a regulatory framework.
type Framework string

const (
	FrameworkEUAIAct  Framework = "eu_ai_act"
	FrameworkNISTRMF  Framework = "nist_rmf"
	FrameworkISO42001 Framework = "iso_42001"
)

// ReportFormat is the output format for compliance reports.
type ReportFormat string

const (
	FormatJSON     ReportFormat = "json"
	FormatMarkdown ReportFormat = "markdown"
)

// --- Spec 06 Sec. 2.3 — the `compliance_reports:` configuration file ----------
//
// Spec 06 Sec. 2.3 documents a `compliance_reports:` YAML block, one section per
// framework, and before this file **nothing read it**: `ComplianceConfig` and
// `DefaultComplianceConfig()` had zero non-test call sites (grep-verified), so
// an operator who wrote the documented file changed nothing and the three
// compliance tools ran on hardcoded defaults. This is the same defect class as
// the Spec 01/02/03/04/05 config surfaces fixed earlier in this program, and it
// is the last gap spec carrying a documented configuration surface with no
// consumer.
//
// It matters because the three tools are the ones an auditor is handed. An
// operator who declares `active_frameworks: [nist_rmf]` and gets an EU AI Act
// report anyway, or who writes `enabled: false` for a framework under active
// remediation and still gets it scored, has a review trail asserting a scope the
// server never applied.
//
// The loader is strict, for the same reasons as its five siblings:
//
//  1. Unknown keys are an ERROR (KnownFields). A typo must not leave a setting
//     at its zero value and load clean.
//  2. A file that declares nothing is not a configuration: an empty file, a
//     missing `compliance_reports:` section and a section with no enabled
//     framework are each refused by name.
//  3. A framework id the requirement database does not carry is refused by
//     name, with the known set. Activating a framework that does not exist
//     would otherwise produce a permanent "0% covered" reading that looks like
//     a measurement.
//  4. An evidence source the tree cannot OBSERVE is refused by name. The
//     documented `config_history` has no producer anywhere in this tree (the
//     only occurrence of the string is the default list itself), so declaring
//     it would name evidence nothing can collect.
//  5. `output_format: pdf` is refused by name. Spec Sec. 4.2 offers json/pdf/
//     markdown; this tree renders json and markdown only, and a report that
//     silently came back as JSON when the file asked for PDF is exactly the
//     "declared but not honoured" outcome this file exists to prevent.
//  6. `sections:` is refused by name. The measured report emits a fixed
//     structure (summary, requirements, over-declarations, unmeasured, gaps,
//     recommendations); per-section selection is not implemented, so a declared
//     list would be inert. NotImplementedByDesign, recorded in STATUS.md.
//
// Two spellings load, matching the sibling loaders: the documented per-framework
// map (`compliance_reports: { eu_ai_act: { ... } }`) and the flat scalar form
// the Go type has always carried. One report format and one export path are
// derived from the enabled sections; two enabled frameworks declaring different
// values is refused, because there is no correct merge order and choosing one
// silently makes the report's shape depend on map iteration.

// complianceSectionYAML mirrors one framework's keys in Spec 06 Sec. 2.3.
type complianceSectionYAML struct {
	Enabled         *bool    `yaml:"enabled"`
	OutputFormat    string   `yaml:"output_format"`
	EvidenceSources []string `yaml:"evidence_sources"`
	Sections        []string `yaml:"sections"`
	ExportPath      string   `yaml:"export_path"`
}

// complianceReportsFile is the wrapper: Sec. 2.3 nests the frameworks under a
// top-level `compliance_reports:` key.
type complianceReportsFile struct {
	ComplianceReports map[string]complianceSectionYAML `yaml:"compliance_reports"`
}

// flatComplianceFile is the flat form the Go struct has always carried, so a
// programmatic or hand-written flat file keeps loading.
type flatComplianceFile struct {
	Enabled           *bool    `yaml:"enabled"`
	ActiveFrameworks  []string `yaml:"active_frameworks"`
	ReportFormat      string   `yaml:"report_format"`
	EvidenceSources   []string `yaml:"evidence_sources"`
	ExportPath        string   `yaml:"export_path"`
	RequirementDbPath string   `yaml:"requirement_db_path"`
}

// ConfigurableComplianceFormats lists the report formats this tree can render,
// sorted. `pdf` is deliberately absent: spec Sec. 4.2 offers it, the renderer
// does not implement it, and naming it is an error rather than a silent
// substitution.
func ConfigurableComplianceFormats() []string {
	return []string{string(FormatJSON), string(FormatMarkdown)}
}

// ConfigurableComplianceEvidenceSources lists the evidence classes the running
// system can actually OBSERVE, sorted. The documented `config_history` is
// deliberately absent: no code path records a configuration snapshot, so a
// declaration of it would name evidence nothing can collect.
func ConfigurableComplianceEvidenceSources() []string {
	return []string{"audit_logs", "guardrail_events", "metrics"}
}

// LoadComplianceConfigFromYAML reads the Spec 06 Sec. 2.3 configuration.
func LoadComplianceConfigFromYAML(path string) (*ComplianceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Peek at the top-level keys so both spellings can be dispatched without
	// loading a documented file through the wrong shape and reporting a
	// confusing "field not found".
	var probe map[string]any
	if err := yaml.Unmarshal(data, &probe); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("config %s is empty; it declares no compliance_reports section", path)
		}
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if len(probe) == 0 {
		return nil, fmt.Errorf("config %s is empty; it declares no compliance_reports section", path)
	}

	_, nested := probe["compliance_reports"]
	_, flat := probe["active_frameworks"]

	switch {
	case nested && flat:
		// Two spellings of the same settings in one file has no correct
		// precedence: picking one makes policy depend on parse order.
		return nil, fmt.Errorf("config %s declares both the `compliance_reports:` map and the flat `active_frameworks:` form; declare one", path)
	case nested:
		return decodeNestedCompliance(data, path)
	case flat:
		return decodeFlatCompliance(data, path)
	default:
		return nil, fmt.Errorf("config %s declares neither a `compliance_reports:` section nor a flat `active_frameworks:` list", path)
	}
}

func decodeNestedCompliance(data []byte, path string) (*ComplianceConfig, error) {
	var root complianceReportsFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if root.ComplianceReports == nil {
		return nil, fmt.Errorf("config %s declares no compliance_reports section", path)
	}

	known, err := knownComplianceFrameworks()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(root.ComplianceReports))
	for id := range root.ComplianceReports {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	enabled := make([]string, 0, len(ids))
	var format, exportPath string
	sources := map[string]bool{}
	formatSet := false
	exportSet := false

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, fmt.Errorf("a compliance_reports entry has an empty framework id")
		}
		if !known[id] {
			return nil, fmt.Errorf("compliance_reports names framework %q, which the requirement database does not carry; known frameworks: %s",
				id, strings.Join(sortedKeys(known), ", "))
		}
		section := root.ComplianceReports[strings.TrimSpace(id)]

		if len(section.Sections) > 0 {
			return nil, fmt.Errorf("compliance_reports.%s declares `sections:`, which this server does not implement: the measured report emits a fixed structure. Remove the key (per-section selection is recorded as an open item in the gap-spec STATUS ledger)", id)
		}

		// A framework is activated unless it is explicitly disabled; an absent
		// `enabled:` on a declared section is not a reason to silently drop it.
		if section.Enabled != nil && !*section.Enabled {
			continue
		}
		enabled = append(enabled, id)

		if section.OutputFormat != "" {
			f := strings.TrimSpace(section.OutputFormat)
			if f != string(FormatJSON) && f != string(FormatMarkdown) {
				if f == "pdf" {
					return nil, fmt.Errorf("compliance_reports.%s declares output_format: pdf, which this server does not implement; available: %s",
						id, strings.Join(ConfigurableComplianceFormats(), ", "))
				}
				return nil, fmt.Errorf("compliance_reports.%s declares unknown output_format %q; available: %s",
					id, section.OutputFormat, strings.Join(ConfigurableComplianceFormats(), ", "))
			}
			if formatSet && format != f {
				return nil, fmt.Errorf("compliance_reports declares two output formats (%q and %q) across enabled frameworks; one report has one format", format, f)
			}
			format, formatSet = f, true
		}

		if section.ExportPath != "" {
			p := strings.TrimSpace(section.ExportPath)
			if exportSet && exportPath != p {
				return nil, fmt.Errorf("compliance_reports declares two export paths (%q and %q) across enabled frameworks; there is no correct merge order", exportPath, p)
			}
			exportPath, exportSet = p, true
		}

		for _, src := range section.EvidenceSources {
			s := strings.TrimSpace(src)
			if s == "" {
				return nil, fmt.Errorf("compliance_reports.%s declares an empty evidence source", id)
			}
			if !contains(ConfigurableComplianceEvidenceSources(), s) {
				return nil, fmt.Errorf("compliance_reports.%s names evidence source %q, which no code path in this server can observe; observable sources: %s",
					id, src, strings.Join(ConfigurableComplianceEvidenceSources(), ", "))
			}
			sources[s] = true
		}
	}

	if len(enabled) == 0 {
		return nil, fmt.Errorf("compliance_reports declares no enabled framework; use `enabled: false` on every section only if you also intend the three compliance tools to report not-configured")
	}
	sort.Strings(enabled)

	cfg := &ComplianceConfig{
		Enabled:          true,
		ActiveFrameworks: enabled,
		ReportFormat:     string(FormatJSON),
		EvidenceSources:  sortedKeys(sources),
		ExportPath:       exportPath,
	}
	if formatSet {
		cfg.ReportFormat = format
	}
	if len(cfg.EvidenceSources) == 0 {
		cfg.EvidenceSources = ConfigurableComplianceEvidenceSources()
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func decodeFlatCompliance(data []byte, path string) (*ComplianceConfig, error) {
	var raw flatComplianceFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg := DefaultComplianceConfig()
	if raw.Enabled != nil {
		cfg.Enabled = *raw.Enabled
	}
	if len(raw.ActiveFrameworks) > 0 {
		cfg.ActiveFrameworks = raw.ActiveFrameworks
	}
	if raw.ReportFormat != "" {
		cfg.ReportFormat = strings.TrimSpace(raw.ReportFormat)
	}
	if len(raw.EvidenceSources) > 0 {
		cfg.EvidenceSources = raw.EvidenceSources
	}
	if raw.ExportPath != "" {
		cfg.ExportPath = strings.TrimSpace(raw.ExportPath)
	}
	if raw.RequirementDbPath != "" {
		cfg.RequirementDbPath = strings.TrimSpace(raw.RequirementDbPath)
	}

	known, err := knownComplianceFrameworks()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, fw := range cfg.ActiveFrameworks {
		f := strings.TrimSpace(fw)
		if f == "" {
			return nil, fmt.Errorf("active_frameworks declares an empty framework id")
		}
		if !known[f] {
			return nil, fmt.Errorf("active_frameworks names framework %q, which the requirement database does not carry; known frameworks: %s",
				f, strings.Join(sortedKeys(known), ", "))
		}
		if seen[f] {
			return nil, fmt.Errorf("active_frameworks names framework %q twice", f)
		}
		seen[f] = true
	}
	sorted := make([]string, 0, len(seen))
	for f := range seen {
		sorted = append(sorted, f)
	}
	sort.Strings(sorted)
	cfg.ActiveFrameworks = sorted

	for _, src := range cfg.EvidenceSources {
		s := strings.TrimSpace(src)
		if !contains(ConfigurableComplianceEvidenceSources(), s) {
			return nil, fmt.Errorf("evidence_sources names %q, which no code path in this server can observe; observable sources: %s",
				src, strings.Join(ConfigurableComplianceEvidenceSources(), ", "))
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// knownComplianceFrameworks returns the framework ids the requirement database
// carries. It reads the EMBEDDED database, so a deployment validates against the
// same set it will report on rather than against a source-checkout file.
func knownComplianceFrameworks() (map[string]bool, error) {
	mapper, err := NewEmbeddedComplianceMapper()
	if err != nil {
		return nil, fmt.Errorf("load requirement database: %w", err)
	}
	out := map[string]bool{}
	for _, fw := range mapper.Frameworks() {
		out[fw] = true
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("the requirement database carries no frameworks; refusing to validate a configuration against an empty set")
	}
	return out, nil
}

// AllowsFramework reports whether a framework is in the activated set. A nil
// config allows every framework the database carries, which is the pre-file
// behaviour; an explicit `enabled: false` on every section yields a config that
// allows none and lets the tools answer not-configured.
func (c *ComplianceConfig) AllowsFramework(framework string) bool {
	if c == nil {
		return true
	}
	if !c.Enabled {
		return false
	}
	return contains(c.ActiveFrameworks, framework)
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
