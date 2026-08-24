package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project_name is required")
	}
	if len(name) > 64 {
		return fmt.Errorf("project_name must be 64 characters or less")
	}
	// Allow alphanumeric, hyphen, underscore only
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("project_name must contain only letters, numbers, hyphens, and underscores")
		}
	}
	return nil
}

// validRoles is the whitelist of 48 valid roles from TEAM_STRUCTURE.md
var validRoles = map[string]bool{
	// Team 1: Business & Product Strategy
	"Business Relationship Manager": true,
	"Lead Product Manager":          true,
	"Business Systems Analyst":      true,
	"Financial Controller (FinOps)": true,
	// Team 2: Enterprise Architecture
	"Chief Architect":    true,
	"Domain Architect":   true,
	"Solution Architect": true,
	"Standards Lead":     true,
	// Team 3: GRC
	"Compliance Officer": true,
	"Internal Auditor":   true,
	"Privacy Engineer":   true,
	"Policy Manager":     true,
	// Team 4: Infrastructure & Cloud Ops
	"Cloud Architect":           true,
	"IaC Engineer":              true,
	"Network Security Engineer": true,
	"Storage Engineer":          true,
	// Team 5: Platform Engineering
	"Platform Product Manager": true,
	"CI/CD Architect":          true,
	"Kubernetes Administrator": true,
	"Developer Advocate":       true,
	// Team 6: Data Governance & Analytics
	"Data Architect":       true,
	"DBA":                  true,
	"Data Privacy Officer": true,
	"ETL Developer":        true,
	// Team 7: Core Feature Squad
	"Technical Lead":              true,
	"Senior Backend Engineer":     true,
	"Senior Frontend Engineer":    true,
	"Accessibility (A11y) Expert": true,
	"Technical Writer":            true,
	// Team 8: Middleware & Integration
	"API Product Manager":  true,
	"Integration Engineer": true,
	"Messaging Engineer":   true,
	"IAM Specialist":       true,
	// Team 9: Cybersecurity
	"Security Architect":       true,
	"Vulnerability Researcher": true,
	"Penetration Tester":       true,
	"DevSecOps Engineer":       true,
	// Team 10: Quality Engineering
	"QA Architect":                true,
	"SDET":                        true,
	"Performance/Load Engineer":   true,
	"Manual QA / UAT Coordinator": true,
	// Team 11: SRE
	"SRE Lead":               true,
	"Observability Engineer": true,
	"Chaos Engineer":         true,
	"Incident Manager":       true,
	// Team 12: IT Operations & Support
	"NOC Analyst":         true,
	"Change Manager":      true,
	"Release Manager":     true,
	"L3 Support Engineer": true,
}

// validateRoleName validates role name against whitelist (SEC-002)
func validateRoleName(name string) error {
	if name == "" {
		return fmt.Errorf("role_name is required")
	}
	if len(name) > 128 {
		return fmt.Errorf("role_name must be 128 characters or less")
	}
	// Check for control characters
	for _, r := range name {
		if r < 32 || r == 127 {
			return fmt.Errorf("role_name contains invalid control characters")
		}
	}
	// Whitelist validation: must be one of 48 valid roles
	if !validRoles[name] {
		return fmt.Errorf("invalid role_name: '%s'. Must be one of the 48 defined roles in TEAM_STRUCTURE.md", name)
	}
	return nil
}

// validatePersonName validates person/assignee name format (SEC-003)
// Accepts email addresses, usernames, or display names with alphanumeric, spaces, dots, hyphens, underscores, apostrophes
func validatePersonName(name string) error {
	if name == "" {
		return fmt.Errorf("person is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("person must be 256 characters or less")
	}
	// Check for control characters
	for _, r := range name {
		if r < 32 || r == 127 {
			return fmt.Errorf("person contains invalid control characters")
		}
	}
	// Check for potentially dangerous patterns
	dangerousPatterns := []string{";", "|", "&&", "||", "`", "$", "<", ">", "..", "\\"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(name, pattern) {
			return fmt.Errorf("person contains forbidden pattern: %s", pattern)
		}
	}
	// Validate format: email, username, or display name
	// Email pattern: user@domain.com
	// Username pattern: alphanumeric + dots + hyphens + underscores
	// Display name: alphanumeric + spaces + dots + hyphens + underscores + apostrophes
	isEmail := false
	if strings.Contains(name, "@") {
		parts := strings.Split(name, "@")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			// Basic email validation - must have domain with at least one dot
			domainParts := strings.Split(parts[1], ".")
			if len(domainParts) >= 2 {
				isEmail = true
			}
		}
	}
	if !isEmail {
		// Must be username or display name format
		// Allow alphanumeric, spaces, dots, hyphens, underscores, apostrophes (for names like O'Connor)
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
				r == ' ' || r == '.' || r == '-' || r == '_' || r == '\'') {
				return fmt.Errorf("person contains invalid characters")
			}
		}
	}
	return nil
}

// validatePhase validates phase filter value (SEC-010: Phase injection hardening)
// Whitelist: Phase 1, Phase 2, Phase 3 (strict regex validation)
func validatePhase(phase string) error {
	if phase == "" {
		return nil // Phase is optional
	}
	// SEC-010: Strict regex validation - only allow "Phase 1", "Phase 2", "Phase 3"
	// This prevents injection attacks through the phase parameter
	validPhaseRegex := regexp.MustCompile(`^Phase [1-3]$`)
	if !validPhaseRegex.MatchString(phase) {
		return fmt.Errorf("invalid phase: must be 'Phase 1', 'Phase 2', or 'Phase 3'")
	}
	return nil
}

// sanitizePhase sanitizes phase string for safe command execution (SEC-010)
// Returns empty string if phase is invalid, otherwise returns cleaned phase
func sanitizePhase(phase string) string {
	if phase == "" {
		return ""
	}
	// Whitelist only exact phase patterns
	switch phase {
	case "Phase 1":
		return "Phase 1"
	case "Phase 2":
		return "Phase 2"
	case "Phase 3":
		return "Phase 3"
	default:
		return "" // Invalid phase - return empty for safety
	}
}

// Team tool handler implementations for MCP server

// handleTeamInit initializes team structure for a project
