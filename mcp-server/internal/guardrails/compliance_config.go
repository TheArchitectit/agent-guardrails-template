// Package guardrails provides tools for mapping system features to regulatory compliance frameworks.
package guardrails

import "fmt"

// ComplianceConfig defines the configuration for compliance reporting and mapping.
type ComplianceConfig struct {
	Enabled                bool              `yaml:"enabled" json:"enabled"`
	ActiveFrameworks       []string          `yaml:"active_frameworks" json:"active_frameworks"`
	ReportFormat           string            `yaml:"report_format" json:"report_format"` // "json", "markdown"
	EvidenceSources        []string          `yaml:"evidence_sources" json:"evidence_sources"`
	ExportPath             string            `yaml:"export_path" json:"export_path"`
	RequirementDbPath      string            `yaml:"requirement_db_path" json:"requirement_db_path"`
}

// DefaultComplianceConfig returns a secure base configuration for compliance.
func DefaultComplianceConfig() *ComplianceConfig {
	return &ComplianceConfig{
		Enabled:           true,
		ActiveFrameworks:  []string{"eu_ai_act", "nist_rmf"},
		ReportFormat:      "json",
		EvidenceSources:   []string{"audit_logs", "guardrail_events", "config_history"},
		ExportPath:        "reports/compliance/",
		RequirementDbPath: "internal/guardrails/compliance_requirements.json",
	}
}

// Validate checks the config for errors.
func (c *ComplianceConfig) Validate() error {
	if c.ReportFormat != "json" && c.ReportFormat != "markdown" {
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
	FrameworkEUAIAct   Framework = "eu_ai_act"
	FrameworkNISTRMF   Framework = "nist_rmf"
	FrameworkISO42001  Framework = "iso_42001"
)

// ReportFormat is the output format for compliance reports.
type ReportFormat string

const (
	FormatJSON     ReportFormat = "json"
	FormatMarkdown ReportFormat = "markdown"
)
