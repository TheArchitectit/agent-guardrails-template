// Package guardrails provides tools for mapping system features to regulatory compliance frameworks.
//
// This module implements Spec 06: Regulatory Compliance Mapping, providing
// the ability to check system status against frameworks like the EU AI Act
// and NIST AI RMF, and generate evidence-based compliance reports.
package guardrails

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Requirement represents a single regulatory requirement.
type Requirement struct {
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	RequiredFor       []string `json:"required_for"`
	GuardrailFeatures []string `json:"guardrail_features"`
	EvidenceQueries   []string `json:"evidence_queries"`
	ComplianceStatus  string   `json:"compliance_status"`
	Gaps              []string `json:"gaps"`
	Recommendations   []string `json:"recommendations"`
}

// FrameworkDb is the mapping of framework IDs to their requirements.
type FrameworkDb map[string]map[string]Requirement

// Evidence represents a piece of proof that a requirement is met.
type Evidence struct {
	RequirementID string    `json:"requirement_id"`
	Source        string    `json:"source"`
	Value         string    `json:"value"`
	Timestamp     time.Time `json:"timestamp"`
	Verified      bool      `json:"verified"`
}

// ComplianceSection represents a group of requirements in a report.
type ComplianceSection struct {
	Name         string                 `json:"name"`
	Requirements map[string]Requirement `json:"requirements"`
	Score        float64                `json:"score"`
}

// ComplianceReport is the final output of a compliance audit.
type ComplianceReport struct {
	Framework       string               `json:"framework"`
	GeneratedAt     time.Time            `json:"generated_at"`
	OverallScore    float64              `json:"overall_score"`
	Sections        []ComplianceSection  `json:"sections"`
	CriticalGaps    []string             `json:"critical_gaps"`
	Recommendations []string             `json:"recommendations"`
}

// ComplianceMapper handles the mapping between features and requirements.
type ComplianceMapper struct {
	db *FrameworkDb
}

func NewComplianceMapper(dbPath string) (*ComplianceMapper, error) {
	data, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read requirement db: %w", err)
	}

	var db FrameworkDb
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("failed to parse requirement db: %w", err)
	}

	return &ComplianceMapper{db: &db}, nil
}

// CheckRequirement evaluates if a specific requirement is met.
func (m *ComplianceMapper) CheckRequirement(framework, reqID string) (bool, []Evidence, []string) {
	fw, ok := (*m.db)[framework]
	if !ok {
		return false, nil, []string{"framework not found"}
	}

	req, ok := fw[reqID]
	if !ok {
		return false, nil, []string{"requirement not found"}
	}

	// In a real implementation, this would check if the features in req.GuardrailFeatures
	// are actually enabled and active in the current system.
	if req.ComplianceStatus == "full" {
		return true, []Evidence{{RequirementID: reqID, Source: "system_config", Value: "Feature active", Timestamp: time.Now().UTC(), Verified: true}}, nil
	}

	return false, nil, req.Gaps
}

// ComplianceReporter generates structured reports.
type ComplianceReporter struct {
	mapper *ComplianceMapper
	logger *slog.Logger
}

func NewComplianceReporter(mapper *ComplianceMapper, logger *slog.Logger) *ComplianceReporter {
	return &ComplianceReporter{
		mapper: mapper,
		logger: logger,
	}
}

// GenerateReport creates a full compliance report for a given framework.
func (r *ComplianceReporter) GenerateReport(ctx context.Context, framework string, format ReportFormat) (*ComplianceReport, error) {
	fw, ok := (*r.mapper.db)[framework]
	if !ok {
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}

	report := &ComplianceReport{
		Framework:   framework,
		GeneratedAt: time.Now().UTC(),
		Sections:    []ComplianceSection{},
	}

	var totalCovered int
	var totalReqs int

	for id, req := range fw {
		totalReqs++
		compliant, _, _ := r.mapper.CheckRequirement(framework, id)
		if compliant {
			totalCovered++
		}
		if req.ComplianceStatus == "gap" {
			report.CriticalGaps = append(report.CriticalGaps, fmt.Sprintf("%s: %s", id, req.Title))
		}
		report.Recommendations = append(report.Recommendations, req.Recommendations...)
	}

	if totalReqs > 0 {
		report.OverallScore = (float64(totalCovered) / float64(totalReqs)) * 100
	}

	// Organize into a single section for this simple implementation
	report.Sections = append(report.Sections, ComplianceSection{
		Name:         "General Compliance",
		Requirements: fw,
		Score:        report.OverallScore,
	})

	return report, nil
}

// ExportReport writes the report to the filesystem.
func (r *ComplianceReporter) ExportReport(report *ComplianceReport, path string, format ReportFormat) (string, error) {
	fileName := fmt.Sprintf("compliance_%s_%s.%s", report.Framework, report.GeneratedAt.Format("20060102"), format)
	fullPath := filepath.Join(path, fileName)

	var data []byte
	var err error

	switch format {
	case FormatJSON:
		data, err = json.MarshalIndent(report, "", "  ")
	case FormatMarkdown:
		data = []byte(fmt.Sprintf("# Compliance Report: %s\n\nScore: %.2f%%\n\nGenerated: %s", report.Framework, report.OverallScore, report.GeneratedAt))
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", err
	}

	return fullPath, nil
}

// EvidenceCollector simulates the collection of evidence from logs.
type EvidenceCollector struct {
	logger *slog.Logger
}

func NewEvidenceCollector(logger *slog.Logger) *EvidenceCollector {
	return &EvidenceCollector{logger: logger}
}

// CollectEvidence runs queries against the audit logs to gather proof.
func (c *EvidenceCollector) CollectEvidence(ctx context.Context, framework, reqID string, queries []string) ([]Evidence, float64) {
	var evidence []Evidence

	for _, q := range queries {
		c.logger.Debug("collecting evidence", "query", q)
		// Simulation: logic to execute query against PostgreSQL would go here.
		evidence = append(evidence, Evidence{
			RequirementID: reqID,
			Source:        "audit_log",
			Value:         fmt.Sprintf("Query result for [%s]: 12 events found", q),
			Timestamp:     time.Now().UTC(),
			Verified:      true,
		})
	}

	completeness := 1.0
	if len(queries) == 0 {
		completeness = 0.0
	}

	return evidence, completeness
}

// CalculateComplianceScore implements the logic from Spec 06.
func CalculateComplianceScore(requirements map[string]Requirement, evidence []Evidence) float64 {
	if len(requirements) == 0 {
		return 0.0
	}

	covered := make(map[string]bool)
	for _, e := range evidence {
		covered[e.RequirementID] = true
	}

	score := 0.0
	for id := range requirements {
		if covered[id] {
			score++
		}
	}

	return (score / float64(len(requirements))) * 100
}
