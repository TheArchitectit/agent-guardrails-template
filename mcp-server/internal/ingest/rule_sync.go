package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

type RuleSyncResult struct {
	Added    int
	Updated  int
	Disabled int
	Errors   []string
}

// JSONRuleFile represents the structure of JSON rule files
type JSONRuleFile struct {
	Schema      string     `json:"$schema"`
	Description string     `json:"description"`
	Version     string     `json:"version"`
	Rules       []JSONRule `json:"rules"`
}

// JSONRule represents a single rule in JSON format
type JSONRule struct {
	RuleID           string   `json:"rule_id"`
	FailureID        *string  `json:"failure_id"`
	Name             string   `json:"name"`
	Enabled          bool     `json:"enabled"`
	Pattern          string   `json:"pattern"`
	ForbiddenContext *string  `json:"forbidden_context"`
	Message          string   `json:"message"`
	Severity         string   `json:"severity"`
	FileGlob         []string `json:"file_glob"`
	Suggestion       string   `json:"suggestion"`
	Category         string   `json:"category"`
}

// ParseJSONRuleFile parses a JSON rule file and extracts rules
func (p *RuleParser) ParseJSONRuleFile(path string) ([]ParsedRule, error) {
	slog.Info("Parsing JSON rule file", "file", path)

	content, err := os.ReadFile(path)
	if err != nil {
		slog.Error("Failed to read JSON rule file", "file", path, "error", err)
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var jsonFile JSONRuleFile
	if err := json.Unmarshal(content, &jsonFile); err != nil {
		slog.Error("Failed to unmarshal JSON rule file", "file", path, "error", err)
		return nil, fmt.Errorf("failed to parse JSON file %s: %w", path, err)
	}

	var rules []ParsedRule
	for _, jsonRule := range jsonFile.Rules {
		// Skip disabled rules
		if !jsonRule.Enabled {
			slog.Debug("Skipping disabled rule", "rule_id", jsonRule.RuleID)
			continue
		}

		rule := ParsedRule{
			RuleID:   jsonRule.RuleID,
			Name:     jsonRule.Name,
			Pattern:  jsonRule.Pattern,
			Message:  jsonRule.Message,
			Severity: jsonRule.Severity,
			Category: jsonRule.Category,
		}

		// Default category if not set
		if rule.Category == "" {
			rule.Category = "general"
		}

		// Compute hash
		hash := sha256.Sum256([]byte(jsonRule.Pattern + jsonRule.Message))
		rule.PatternHash = fmt.Sprintf("%x", hash[:8])

		// Validate
		if err := p.validateRule(&rule); err != nil {
			slog.Error("Invalid JSON rule", "rule_id", jsonRule.RuleID, "error", err)
			continue
		}

		rules = append(rules, rule)
	}

	slog.Info("Successfully parsed JSON rule file", "file", path, "rules_found", len(rules))
	return rules, nil
}

// RuleSyncService handles syncing parsed rules to the database
type RuleSyncService struct {
	ruleStore *database.RuleStore
	parser    *RuleParser
}

// NewRuleSyncService creates a new rule sync service
func NewRuleSyncService(ruleStore *database.RuleStore) *RuleSyncService {
	return &RuleSyncService{
		ruleStore: ruleStore,
		parser:    NewRuleParser(),
	}
}

// SyncRulesFromDirectory syncs all rules from markdown and JSON files in a directory
func (s *RuleSyncService) SyncRulesFromDirectory(ctx context.Context, dir string) (*RuleSyncResult, error) {
	slog.Info("Syncing rules from directory", "dir", dir)
	result := &RuleSyncResult{}
	fileCount := 0
	processedRuleIDs := make(map[string]bool)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		var rules []ParsedRule
		var parseErr error

		// Handle markdown files
		if IsMarkdownFile(path) {
			fileCount++
			slog.Debug("Processing markdown file", "file", path)
			rules, parseErr = s.parser.ParseRuleFile(path)
		} else if strings.HasSuffix(strings.ToLower(path), ".json") {
			// Handle JSON rule files
			fileCount++
			slog.Debug("Processing JSON rule file", "file", path)
			rules, parseErr = s.parser.ParseJSONRuleFile(path)
		} else {
			return nil // Skip other files
		}

		if parseErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to parse %s: %v", path, parseErr))
			return nil
		}

		for _, parsedRule := range rules {
			processedRuleIDs[parsedRule.RuleID] = true

			if err := s.syncRule(ctx, parsedRule, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to sync %s: %v", parsedRule.RuleID, err))
			}
		}

		return nil
	})

	if err != nil {
		return result, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Disable rules that no longer exist in markdown files
	if err := s.disableOrphanedRules(ctx, processedRuleIDs, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to disable orphaned rules: %v", err))
	}

	return result, nil
}

// SyncRulesFromContent syncs rules from markdown content (for uploaded files)
func (s *RuleSyncService) SyncRulesFromContent(ctx context.Context, content, filename string) (*RuleSyncResult, error) {
	result := &RuleSyncResult{}

	rules, err := s.parser.ParseRuleContent(content, filename)
	if err != nil {
		return result, fmt.Errorf("failed to parse content: %w", err)
	}

	for _, parsedRule := range rules {
		if err := s.syncRule(ctx, parsedRule, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to sync %s: %v", parsedRule.RuleID, err))
		}
	}

	return result, nil
}

// syncRule syncs a single rule to the database
func (s *RuleSyncService) syncRule(ctx context.Context, parsed ParsedRule, result *RuleSyncResult) error {
	// Check if rule already exists
	existing, err := s.ruleStore.GetByRuleID(ctx, parsed.RuleID)
	if err != nil {
		// Check if it's a "not found" error
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to check existing rule: %w", err)
		}
		existing = nil
	}

	if existing != nil {
		// Check if content changed
		if existing.PatternHash != nil && *existing.PatternHash == parsed.PatternHash {
			// Rule unchanged, just ensure it's enabled
			if !existing.Enabled {
				existing.Enabled = true
				if err := s.ruleStore.Update(ctx, existing); err != nil {
					return fmt.Errorf("failed to re-enable rule: %w", err)
				}
				result.Updated++
			}
			return nil
		}

		// Update existing rule
		existing.Name = parsed.Name
		existing.Pattern = parsed.Pattern
		existing.PatternHash = &parsed.PatternHash
		existing.Message = parsed.Message
		existing.Severity = models.Severity(parsed.Severity)
		existing.Category = parsed.Category
		existing.Enabled = true

		if err := s.ruleStore.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update rule: %w", err)
		}
		result.Updated++
	} else {
		// Create new rule
		newRule := &models.PreventionRule{
			ID:          uuid.New(),
			RuleID:      parsed.RuleID,
			Name:        parsed.Name,
			Pattern:     parsed.Pattern,
			PatternHash: &parsed.PatternHash,
			Message:     parsed.Message,
			Severity:    models.Severity(parsed.Severity),
			Category:    parsed.Category,
			Enabled:     true,
		}

		if err := s.ruleStore.Create(ctx, newRule); err != nil {
			return fmt.Errorf("failed to create rule: %w", err)
		}
		result.Added++
	}

	return nil
}

// disableOrphanedRules disables rules that no longer exist in markdown files
func (s *RuleSyncService) disableOrphanedRules(ctx context.Context, processedIDs map[string]bool, result *RuleSyncResult) error {
	// Get all enabled rules (using large limit to get all)
	rules, err := s.ruleStore.List(ctx, boolPtr(true), "", 10000, 0)
	if err != nil {
		return fmt.Errorf("failed to list rules: %w", err)
	}

	for _, rule := range rules {
		if !processedIDs[rule.RuleID] {
			// Rule no longer exists in markdown files
			rule.Enabled = false
			if err := s.ruleStore.Update(ctx, &rule); err != nil {
				return fmt.Errorf("failed to disable rule %s: %w", rule.RuleID, err)
			}
			result.Disabled++
		}
	}

	return nil
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}
