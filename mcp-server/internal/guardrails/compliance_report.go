package guardrails

// Spec 06 — measured compliance reporting.
//
// The previous GenerateReport produced a score by counting requirements whose
// hand-written compliance_status string equalled "full". That number restated
// the requirement database; it measured nothing. The spec says so itself:
//
//	"Do not quote a score from this code."
//
// The report below counts a requirement only when every control it names was
// OBSERVED live (see compliance_probe.go). Anything unmeasured is reported as
// unmeasured, and the methodology label says which mode produced the number so
// a consumer cannot mistake a restatement for a measurement.

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// MeasuredMethodology is stamped on reports whose score counts only
// live-observed controls. It replaces UnimplementedMethodology on that path.
const MeasuredMethodology = "measured-runtime-probes-v1"

//go:embed compliance_requirements.json
var embeddedRequirementDB []byte

// ComplianceArtifact is the on-disk report. Requirement-level detail is kept
// out of the score path: a reader can see exactly which control was observed.
type ComplianceArtifact struct {
	Framework        string                  `json:"framework"`
	GeneratedAt      time.Time               `json:"generated_at"`
	Methodology      string                  `json:"methodology"`
	OverallScore     float64                 `json:"overall_score"`
	Requirements     []RequirementAssessment `json:"requirements"`
	CriticalGaps     []string                `json:"critical_gaps"`
	Recommendations  []string                `json:"recommendations"`
	OverDeclarations []string                `json:"over_declarations,omitempty"`
	Unmeasured       []string                `json:"unmeasured_requirements,omitempty"`
}

// NewEmbeddedComplianceMapper loads the requirement database compiled into the
// binary. It exists because the file path in DefaultComplianceConfig is
// relative to a source checkout: a deployed binary would otherwise fail to
// open its own requirement DB and the tool would be unusable exactly where an
// auditor needs it.
func NewEmbeddedComplianceMapper() (*ComplianceMapper, error) {
	var db FrameworkDb
	if err := json.Unmarshal(embeddedRequirementDB, &db); err != nil {
		return nil, fmt.Errorf("failed to parse embedded requirement db: %w", err)
	}
	return &ComplianceMapper{db: &db}, nil
}

// Frameworks lists the framework ids in the database, sorted.
func (m *ComplianceMapper) Frameworks() []string {
	out := make([]string, 0, len(*m.db))
	for fw := range *m.db {
		out = append(out, fw)
	}
	sort.Strings(out)
	return out
}

// RequirementIDs lists a framework's requirement ids, sorted.
func (m *ComplianceMapper) RequirementIDs(framework string) []string {
	fw, ok := (*m.db)[framework]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(fw))
	for id := range fw {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Requirement returns one requirement and whether it exists.
func (m *ComplianceMapper) Requirement(framework, reqID string) (Requirement, bool) {
	fw, ok := (*m.db)[framework]
	if !ok {
		return Requirement{}, false
	}
	req, ok := fw[reqID]
	return req, ok
}

// CheckMeasured evaluates one requirement against live probes.
func (m *ComplianceMapper) CheckMeasured(ctx context.Context, framework, reqID string, probes ProbeSet) (RequirementAssessment, error) {
	req, ok := m.Requirement(framework, reqID)
	if !ok {
		return RequirementAssessment{}, fmt.Errorf("unknown requirement %q in framework %q", reqID, framework)
	}
	return AssessRequirement(ctx, reqID, req, probes), nil
}

// GenerateMeasuredReport builds a report whose score counts only requirements
// whose every named control was observed active.
//
// Deliberate properties:
//   - a requirement with no probe on any of its features is UNMEASURED and is
//     listed separately — it is not silently a zero, because "we did not look"
//     and "we looked and it is missing" call for different actions;
//   - an over-declaration (database says full, measurement says partial or
//     absent) is surfaced as a finding, not smoothed into the average.
func (r *ComplianceReporter) GenerateMeasuredReport(ctx context.Context, framework string, probes ProbeSet) (*ComplianceArtifact, error) {
	fw, ok := (*r.mapper.db)[framework]
	if !ok {
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}
	if len(probes) == 0 {
		// Same vacuity rule as the repo's other gates: an empty probe set would
		// report every requirement as unknown and could be mistaken for a real
		// audit that found nothing. Refuse instead.
		return nil, fmt.Errorf("refusing to produce a compliance report with no probes registered")
	}

	art := &ComplianceArtifact{
		Framework:   framework,
		GeneratedAt: time.Now().UTC(),
		Methodology: MeasuredMethodology,
	}

	ids := make([]string, 0, len(fw))
	for id := range fw {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var covered, measured int
	for _, id := range ids {
		a := AssessRequirement(ctx, id, fw[id], probes)
		art.Requirements = append(art.Requirements, a)

		if a.Measured != FeatureUnknown {
			measured++
		} else {
			art.Unmeasured = append(art.Unmeasured, id)
		}
		if a.Covered {
			covered++
		}
		if !a.DeclarationAgrees {
			art.OverDeclarations = append(art.OverDeclarations,
				fmt.Sprintf("%s: declared=%q measured=%q (%s)",
					id, a.Declared, a.Measured, a.Title))
		}
		for _, g := range a.Gaps {
			art.CriticalGaps = append(art.CriticalGaps, fmt.Sprintf("%s: %s", id, g))
		}
		art.Recommendations = append(art.Recommendations, fw[id].Recommendations...)
	}

	// The denominator is every requirement in the framework, including the
	// unmeasured ones: excluding them would let an unmeasured control improve
	// the score by being invisible.
	if len(ids) > 0 {
		art.OverallScore = (float64(covered) / float64(len(ids))) * 100
	}

	if r.logger != nil {
		r.logger.Info("compliance report generated",
			"framework", framework,
			"requirements", len(ids),
			"covered", covered,
			"measured", measured,
			"score", art.OverallScore)
	}
	return art, nil
}

// ExportArtifact writes a measured report. Mode 0600: a compliance artifact
// names which controls are missing, which is a map of the deployment's gaps.
func (r *ComplianceReporter) ExportArtifact(art *ComplianceArtifact, path string, format ReportFormat) (string, error) {
	fileName := fmt.Sprintf("compliance_%s_%s.%s",
		art.Framework, art.GeneratedAt.Format("20060102"), format)
	fullPath := filepath.Join(path, fileName)

	var data []byte
	var err error
	switch format {
	case FormatJSON:
		data, err = json.MarshalIndent(art, "", "  ")
	case FormatMarkdown:
		data, err = []byte(renderMarkdown(art)), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(fullPath, data, 0o600); err != nil {
		return "", err
	}
	return fullPath, nil
}

func renderMarkdown(art *ComplianceArtifact) string {
	out := fmt.Sprintf("# Compliance Report: %s\n\n", art.Framework)
	out += fmt.Sprintf("- Generated: %s\n", art.GeneratedAt.Format(time.RFC3339))
	out += fmt.Sprintf("- Methodology: `%s`\n", art.Methodology)
	out += fmt.Sprintf("- Measured coverage: **%.2f%%** (%d requirements)\n\n", art.OverallScore, len(art.Requirements))
	out += "Coverage counts a requirement only when every control it names was observed active at run time. " +
		"A requirement whose controls could not be observed is listed as unmeasured and does not count as covered.\n\n"

	out += "## Requirements\n\n| Requirement | Declared | Measured | Covered |\n|---|---|---|---|\n"
	for _, r := range art.Requirements {
		out += fmt.Sprintf("| %s | %s | %s | %t |\n", r.RequirementID, r.Declared, r.Measured, r.Covered)
	}

	if len(art.OverDeclarations) > 0 {
		out += "\n## Over-declarations\n\nThe requirement database claims a stronger state than was measured.\n\n"
		for _, o := range art.OverDeclarations {
			out += "- " + o + "\n"
		}
	}
	if len(art.Unmeasured) > 0 {
		out += "\n## Unmeasured requirements\n\nNo probe observed the controls these name, so nothing is claimed about them.\n\n"
		for _, u := range art.Unmeasured {
			out += "- " + u + "\n"
		}
	}
	if len(art.CriticalGaps) > 0 {
		out += "\n## Gaps\n\n"
		for _, g := range art.CriticalGaps {
			out += "- " + g + "\n"
		}
	}
	return out
}
