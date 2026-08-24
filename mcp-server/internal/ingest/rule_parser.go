package ingest

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/thearchitectit/guardrail-mcp/internal/validation"
)

// RuleParser parses markdown files containing prevention rules
type RuleParser struct {
	// ruleHeaderRegex matches rule headers like "## PREVENT-001: Rule Name"
	ruleHeaderRegex *regexp.Regexp
	// metadataRegex matches metadata fields like "**Pattern:** `regex`"
	metadataRegex *regexp.Regexp
	// backtickRegex extracts content from backticks
	backtickRegex *regexp.Regexp
}

// NewRuleParser creates a new rule parser
func NewRuleParser() *RuleParser {
	return &RuleParser{
		ruleHeaderRegex: regexp.MustCompile(`(?m)^##\s+(PREVENT-\d+)\s*:\s*(.+)$`),
		metadataRegex:   regexp.MustCompile(`(?m)^\*\*(\w+):\*\*\s*(.+?)$`),
		backtickRegex:   regexp.MustCompile("`([^`]+)`"),
	}
}

// ParsedRule represents a rule extracted from markdown
type ParsedRule struct {
	RuleID      string
	Name        string
	Pattern     string
	Message     string
	Severity    string
	Category    string
	PatternHash string
}

// ParseRuleFile parses a single markdown file and extracts rules
func (p *RuleParser) ParseRuleFile(path string) ([]ParsedRule, error) {
	slog.Info("Parsing rule file", "file", path)

	content, err := os.ReadFile(path)
	if err != nil {
		slog.Error("Failed to read rule file", "file", path, "error", err)
		slog.Error("Failed to read rule file", "file", path, "error", err)
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	rules, err := p.ParseRuleContent(string(content), path)
	if err != nil {
		slog.Error("Failed to parse rule content", "file", path, "error", err)
		return nil, err
	}

	slog.Info("Successfully parsed rule file", "file", path, "rules_found", len(rules))
	return rules, nil
}

// ParseRuleContent parses markdown content and extracts rules
func (p *RuleParser) ParseRuleContent(content, source string) ([]ParsedRule, error) {
	slog.Debug("Parsing rule content", "source", source, "content_length", len(content))

	var rules []ParsedRule

	// Find all rule sections
	matches := p.ruleHeaderRegex.FindAllStringIndex(content, -1)
	if matches == nil {
		slog.Debug("No rule sections found in content", "source", source)
		return rules, nil
	}

	slog.Debug("Found rule sections in content", "source", source, "section_count", len(matches))

	for i, match := range matches {
		start := match[0]
		end := len(content)
		if i < len(matches)-1 {
			end = matches[i+1][0]
		}

		section := content[start:end]
		rule, err := p.parseRuleSection(section)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rule in %s: %w", source, err)
		}

		if rule != nil {
			// Compute pattern hash for change detection
			hash := sha256.Sum256([]byte(section))
			rule.PatternHash = fmt.Sprintf("%x", hash[:8])
			rules = append(rules, *rule)
		}
	}

	slog.Debug("Completed parsing rule content", "source", source, "rules_extracted", len(rules))

	return rules, nil
}

// parseRuleSection parses a single rule section
func (p *RuleParser) parseRuleSection(section string) (*ParsedRule, error) {
	// Extract rule ID and name from header
	headerMatch := p.ruleHeaderRegex.FindStringSubmatch(section)
	if headerMatch == nil {
		return nil, nil
	}

	rule := &ParsedRule{
		RuleID: headerMatch[1],
		Name:   strings.TrimSpace(headerMatch[2]),
	}

	// Extract metadata fields
	metadata := p.extractMetadata(section)

	// Map metadata to rule fields
	if pattern, ok := metadata["Pattern"]; ok {
		rule.Pattern = p.extractBacktickContent(pattern)
	}
	if message, ok := metadata["Message"]; ok {
		rule.Message = strings.TrimSpace(message)
	}
	if severity, ok := metadata["Severity"]; ok {
		rule.Severity = strings.ToLower(strings.TrimSpace(severity))
	}
	if category, ok := metadata["Category"]; ok {
		rule.Category = strings.ToLower(strings.TrimSpace(category))
	}

	// Extract description (content after metadata, before next section or end)
	// Note: Description is not stored in the database model but can be used for documentation
	_ = p.extractDescription(section)

	// Set default message if not provided
	if rule.Message == "" {
		rule.Message = fmt.Sprintf("Rule violation: %s", rule.Name)
	}

	// Validate the parsed rule
	if err := p.validateRule(rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// extractMetadata extracts all **Key:** Value pairs from content
func (p *RuleParser) extractMetadata(content string) map[string]string {
	metadata := make(map[string]string)
	matches := p.metadataRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])
			metadata[key] = value
		}
	}

	return metadata
}

// extractBacktickContent extracts content from backticks
func (p *RuleParser) extractBacktickContent(content string) string {
	match := p.backtickRegex.FindStringSubmatch(content)
	if len(match) >= 2 {
		return match[1]
	}
	return strings.TrimSpace(content)
}

// extractDescription extracts the description text from a rule section
func (p *RuleParser) extractDescription(section string) string {
	// Split by lines and find description after metadata
	lines := strings.Split(section, "\n")
	var descLines []string
	inDescription := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip header line
		if strings.HasPrefix(trimmed, "## ") {
			continue
		}

		// Skip metadata lines
		if strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, "**:") {
			continue
		}

		// Skip empty lines at start
		if !inDescription && trimmed == "" {
			continue
		}

		inDescription = true

		// Stop at horizontal rules
		if strings.HasPrefix(trimmed, "---") {
			break
		}

		descLines = append(descLines, line)
	}

	// Clean up the description
	description := strings.Join(descLines, "\n")
	description = strings.TrimSpace(description)

	// Remove markdown formatting for plain text description
	description = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(description, "$1")
	description = regexp.MustCompile("`([^`]+)`").ReplaceAllString(description, "$1")

	return description
}

// validateRule validates a parsed rule
func (p *RuleParser) validateRule(rule *ParsedRule) error {
	if rule.RuleID == "" {
		return fmt.Errorf("rule ID is required")
	}

	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if rule.Pattern == "" {
		return fmt.Errorf("pattern is required for rule %s", rule.RuleID)
	}

	// Validate regex pattern
	if err := validation.ValidatePattern(rule.Pattern); err != nil {
		return fmt.Errorf("invalid pattern for rule %s: %w", rule.RuleID, err)
	}

	// Validate severity
	validSeverities := map[string]bool{"error": true, "warning": true, "info": true}
	if !validSeverities[rule.Severity] {
		rule.Severity = "warning" // Default to warning
	}

	// Validate category
	validCategories := map[string]bool{"git": true, "bash": true, "docker": true, "security": true, "general": true}
	if !validCategories[rule.Category] {
		rule.Category = "general" // Default to general
	}

	return nil
}

// RuleSyncResult tracks the results of a rule sync operation
