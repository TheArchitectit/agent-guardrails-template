package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/validation"
)

func (s *Server) ideHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) validateFile(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse request body
	var req struct {
		FilePath    string `json:"file_path"`
		Content     string `json:"content"`
		ProjectSlug string `json:"project_slug,omitempty"`
		Language    string `json:"language,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.FilePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file_path is required"})
	}

	// Get active rules for the project
	var rules []models.PreventionRule
	var err error

	if req.ProjectSlug != "" {
		proj, err := s.projStore.GetBySlug(ctx, req.ProjectSlug)
		if err == nil && len(proj.ActiveRules) > 0 {
			rules, err = s.ruleStore.GetByRuleIDs(ctx, proj.ActiveRules)
			if err != nil {
				slog.Warn("Failed to get project rules, falling back to all active", "error", err)
			}
		}
	}

	// If no project-specific rules, get all active rules
	if len(rules) == 0 {
		rules, err = s.ruleStore.GetActiveRules(ctx)
		if err != nil {
			slog.Error("Failed to get active rules", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve rules"})
		}
	}

	// Validate content against rules
	violations := validateContentAgainstRules(req.FilePath, req.Content, req.Language, rules)

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogValidation(ctx, keyHash, "validate_file", len(violations) == 0, len(violations))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":         len(violations) == 0,
		"violations":    violations,
		"file_path":     req.FilePath,
		"rules_checked": len(rules),
	})
}

func (s *Server) validateSelection(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		Selection   string `json:"selection"`
		FilePath    string `json:"file_path"`
		ProjectSlug string `json:"project_slug,omitempty"`
		Language    string `json:"language,omitempty"`
		StartLine   int    `json:"start_line"`
		EndLine     int    `json:"end_line"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Selection == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "selection is required"})
	}

	// Get active rules for the project
	var rules []models.PreventionRule
	var err error

	if req.ProjectSlug != "" {
		proj, err := s.projStore.GetBySlug(ctx, req.ProjectSlug)
		if err == nil && len(proj.ActiveRules) > 0 {
			rules, err = s.ruleStore.GetByRuleIDs(ctx, proj.ActiveRules)
			if err != nil {
				slog.Warn("Failed to get project rules, falling back to all active", "error", err)
			}
		}
	}

	// If no project-specific rules, get all active rules
	if len(rules) == 0 {
		rules, err = s.ruleStore.GetActiveRules(ctx)
		if err != nil {
			slog.Error("Failed to get active rules", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve rules"})
		}
	}

	// Validate selection against rules
	violations := validateContentAgainstRules(req.FilePath, req.Selection, req.Language, rules)

	// Adjust line numbers to be relative to the file, not the selection
	for i := range violations {
		violations[i].Line = req.StartLine + violations[i].Line - 1
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogValidation(ctx, keyHash, "validate_selection", len(violations) == 0, len(violations))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":         len(violations) == 0,
		"violations":    violations,
		"file_path":     req.FilePath,
		"start_line":    req.StartLine,
		"end_line":      req.EndLine,
		"rules_checked": len(rules),
	})
}

func (s *Server) getIDERules(c echo.Context) error {
	ctx := c.Request().Context()
	projectSlug := c.QueryParam("project")

	// Try cache first for better performance
	cacheKey := projectSlug
	if cacheKey == "" {
		cacheKey = "default"
	}

	if cached, err := s.cache.GetIDERules(ctx, cacheKey); err == nil && len(cached) > 0 {
		// Return cached JSON directly to avoid re-marshaling
		return c.JSONBlob(http.StatusOK, cached)
	}

	var rules []models.PreventionRule
	var err error

	if projectSlug != "" {
		// Get project to find active rules
		proj, err := s.projStore.GetBySlug(ctx, projectSlug)
		if err == nil && len(proj.ActiveRules) > 0 {
			// Batch fetch all project rules in a single query (prevents N+1)
			rules, err = s.ruleStore.GetByRuleIDs(ctx, proj.ActiveRules)
			if err != nil {
				slog.Warn("Failed to get project rules, falling back to all active", "error", err)
			}
		}
	}

	// If no project-specific rules, get all active rules
	if len(rules) == 0 {
		rules, err = s.ruleStore.GetActiveRules(ctx)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	// Marshal once for both caching and response
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to marshal rules"})
	}

	// Cache the result asynchronously to not block the response
	go func(ctx context.Context, key string, data []byte) {
		// Use a new context with timeout for cache operation
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if cacheErr := s.cache.SetIDERules(cacheCtx, key, data); cacheErr != nil {
			slog.Warn("Failed to cache IDE rules", "error", cacheErr)
		}
	}(ctx, cacheKey, rulesJSON)

	return c.JSONBlob(http.StatusOK, rulesJSON)
}

func (s *Server) getQuickReference(c echo.Context) error {
	ctx := c.Request().Context()

	// Try to find quick-reference document
	doc, err := s.docStore.GetBySlug(ctx, "quick-reference")
	if err != nil {
		// Try alternative slugs
		doc, err = s.docStore.GetBySlug(ctx, "quick-reference-card")
		if err != nil {
			// Search for any document with "quick reference" in title
			docs, searchErr := s.docStore.Search(ctx, "quick reference", 5)
			if searchErr != nil || len(docs) == 0 {
				return c.JSON(http.StatusOK, map[string]string{
					"reference": "Quick reference documentation not found. Please ensure documents are ingested.",
				})
			}
			doc = &docs[0]
		}
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogDocChange(ctx, keyHash, doc.Slug, "quick-reference-access")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"reference": doc.Content,
		"title":     doc.Title,
		"slug":      doc.Slug,
		"category":  doc.Category,
	})
}

// policyCheck handles POST /api/v1/policy/check — CI/CD enforcement endpoint
func (s *Server) policyCheck(c echo.Context) error {
	var req models.PolicyCheckRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Input == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "input is required"})
	}

	ctx := c.Request().Context()
	start := time.Now()

	// Load active rules, optionally filtered by category
	rules, err := s.ruleStore.GetActiveRules(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load rules"})
	}

	// Filter by categories if specified
	if len(req.Categories) > 0 {
		categorySet := make(map[string]bool, len(req.Categories))
		for _, cat := range req.Categories {
			categorySet[cat] = true
		}
		filtered := make([]models.PreventionRule, 0, len(rules))
		for _, rule := range rules {
			if categorySet[rule.Category] || rule.Category == "all" {
				filtered = append(filtered, rule)
			}
		}
		rules = filtered
	}

	// Compile patterns and check for violations
	var violations []models.PolicyViolation
	for _, rule := range rules {
		if !rule.Enabled || rule.Pattern == "" {
			continue
		}

		re, err := validation.CompilePattern(rule.Pattern)
		if err != nil {
			slog.Warn("Invalid rule pattern in policy check", "rule_id", rule.RuleID, "error", err)
			continue
		}

		lines := strings.Split(req.Input, "\n")
		for lineNum, line := range lines {
			matches := re.FindAllStringIndex(line, -1)
			for _, match := range matches {
				violations = append(violations, models.PolicyViolation{
					RuleID:         rule.RuleID,
					RuleName:       rule.Name,
					Severity:       string(rule.Severity),
					Message:        rule.Message,
					Category:       rule.Category,
					MatchedPattern: rule.Pattern,
					Line:           lineNum + 1,
					Column:         match[0] + 1,
				})
			}
		}
	}

	duration := time.Since(start)
	elapsedMs := int(duration.Milliseconds())

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogValidation(ctx, keyHash, "policy_check", len(violations) == 0, len(violations))

	return c.JSON(http.StatusOK, models.PolicyCheckResponse{
		Passed:     len(violations) == 0,
		Violations: violations,
		CheckedAt:  time.Now().UTC(),
		DurationMs: elapsedMs,
		RulesCount: len(rules),
	})
}

// validateContentAgainstRules checks content against prevention rules and returns violations
type ValidationViolation struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Match    string `json:"match"`
}

func validateContentAgainstRules(filePath, content, language string, rules []models.PreventionRule) []ValidationViolation {
	var violations []ValidationViolation
	lines := strings.Split(content, "\n")

	for _, rule := range rules {
		if !rule.Enabled || rule.Pattern == "" {
			continue
		}

		// Skip language-specific rules if language doesn't match
		if rule.Category != "" && language != "" && rule.Category != language {
			continue
		}

		// Compile regex pattern with caching and ReDoS protection
		re, err := validation.CompilePattern(rule.Pattern)
		if err != nil {
			slog.Warn("Invalid rule pattern", "rule_id", rule.RuleID, "error", err)
			continue
		}

		// Check each line
		for lineNum, line := range lines {
			matches := re.FindAllStringIndex(line, -1)
			for _, match := range matches {
				violations = append(violations, ValidationViolation{
					RuleID:   rule.RuleID,
					RuleName: rule.Name,
					Severity: string(rule.Severity),
					Message:  rule.Message,
					Line:     lineNum + 1,
					Column:   match[0] + 1,
					Match:    truncateMatch(line[match[0]:match[1]]),
				})
			}
		}
	}

	return violations
}

// truncateMatch limits the match length for display
func truncateMatch(match string) string {
	if len(match) > 50 {
		return match[:50] + "..."
	}
	return match
}

// getAPIKeyHash safely extracts the API key hash from the context
func getAPIKeyHash(c echo.Context) string {
	keyHash, ok := c.Get("api_key_hash").(string)
	if !ok || keyHash == "" {
		return "unknown"
	}
	return keyHash
}

// isValidSlug validates a project slug to prevent path traversal attacks
// Valid slugs contain only alphanumeric characters, hyphens, and underscores
func isValidSlug(slug string) bool {
	if slug == "" {
		return false
	}
	if len(slug) > 100 {
		return false
	}
	// Check for path traversal attempts
	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return false
	}
	// Only allow alphanumeric, hyphens, and underscores
	for _, r := range slug {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// apiDocs serves the Scalar API reference UI
func (s *Server) apiDocs(c echo.Context) error {
	return c.HTML(http.StatusOK, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Guardrail MCP Server — API Reference</title>
  <style>body { margin: 0; }</style>
</head>
<body>
  <script
    id="api-reference"
    data-url="/openapi.yaml"
    data-configuration='{"theme":"purple"}'
  ></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`)
}

// openAPISpec serves the OpenAPI 3.1 YAML specification
func (s *Server) openAPISpec(c echo.Context) error {
	// Try to serve from the docs directory relative to the binary
	paths := []string{
		"docs/openapi.yaml",
		"internal/web/../../docs/openapi.yaml",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return c.File(p)
		}
	}
	// Fallback: embedded inline error
	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "openapi.yaml not found — ensure docs/openapi.yaml exists relative to the binary",
	})
}
