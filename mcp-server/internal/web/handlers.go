package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/security"
)

// Pagination and validation constants
const (
	defaultPageLimit   = 20
	maxPageLimit       = 100
	maxSearchResults   = 50
	defaultSearchLimit = 20
)

// Document handlers

func (s *Server) listDocuments(c echo.Context) error {
	ctx := c.Request().Context()
	category := c.QueryParam("category")
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit <= 0 || limit > maxPageLimit {
		limit = defaultPageLimit
	}
	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	docs, err := s.docStore.List(ctx, category, limit, offset)
	if err != nil {
		slog.Error("Failed to list documents", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve documents"})
	}

	total, err := s.docStore.Count(ctx, category)
	if err != nil {
		slog.Warn("Failed to count documents", "error", err)
		total = len(docs) // Fallback to current page size
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": docs,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (s *Server) getDocument(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	doc, err := s.docStore.GetByID(c.Request().Context(), parsedUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, doc)
}

func (s *Server) updateDocument(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var doc models.Document
	if err := c.Bind(&doc); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Scan for secrets before saving
	if findings := security.ScanContent(doc.Content); len(findings) > 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":    "Potential secrets detected in content",
			"findings": findings,
		})
	}

	doc.ID = parsedUUID
	if err := s.docStore.Update(c.Request().Context(), &doc); err != nil {
		slog.Error("Failed to update document", "doc_id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update document"})
	}

	// Invalidate cache - log error but don't fail the request
	if err := s.cache.InvalidateOnDocumentChange(c.Request().Context(), doc.Slug); err != nil {
		slog.Warn("Failed to invalidate document cache", "slug", doc.Slug, "error", err)
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogDocChange(c.Request().Context(), keyHash, doc.Slug, "update")

	return c.JSON(http.StatusOK, doc)
}

func (s *Server) searchDocuments(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query parameter required"})
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit <= 0 || limit > maxSearchResults {
		limit = defaultSearchLimit
	}

	docs, err := s.docStore.Search(c.Request().Context(), query, limit)
	if err != nil {
		slog.Error("Failed to search documents", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to search documents"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":  docs,
		"query": query,
		"pagination": map[string]interface{}{
			"limit": limit,
		},
	})
}

// Rule handlers

func (s *Server) listRules(c echo.Context) error {
	var enabled *bool
	if enabledParam := c.QueryParam("enabled"); enabledParam != "" {
		e := enabledParam == "true"
		enabled = &e
	}
	category := c.QueryParam("category")
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit <= 0 || limit > maxPageLimit {
		limit = defaultPageLimit
	}
	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	rules, err := s.ruleStore.List(c.Request().Context(), enabled, category, limit, offset)
	if err != nil {
		slog.Error("Failed to list rules", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve rules"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": rules,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (s *Server) getRule(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	rule, err := s.ruleStore.GetByID(c.Request().Context(), parsedUUID)
	if err != nil {
		if err.Error() == fmt.Sprintf("rule not found: %s", parsedUUID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "rule not found"})
		}
		slog.Error("Failed to get rule", "rule_id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve rule"})
	}

	return c.JSON(http.StatusOK, rule)
}

func (s *Server) createRule(c echo.Context) error {
	var rule models.PreventionRule
	if err := c.Bind(&rule); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.ruleStore.Create(c.Request().Context(), &rule); err != nil {
		slog.Error("Failed to create rule", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create rule"})
	}

	// Invalidate cache - log error but don't fail the request
	if err := s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID); err != nil {
		slog.Warn("Failed to invalidate rule cache", "rule_id", rule.RuleID, "error", err)
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "create")

	return c.JSON(http.StatusCreated, rule)
}

func (s *Server) updateRule(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var rule models.PreventionRule
	if err := c.Bind(&rule); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	rule.ID = parsedUUID
	if err := s.ruleStore.Update(c.Request().Context(), &rule); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID)

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "update")

	return c.JSON(http.StatusOK, rule)
}

func (s *Server) deleteRule(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	// Get rule for cache invalidation before deleting
	rule, err := s.ruleStore.GetByID(c.Request().Context(), parsedUUID)
	if err != nil {
		// Rule doesn't exist - return 404
		return c.JSON(http.StatusNotFound, map[string]string{"error": "rule not found"})
	}

	if err := s.ruleStore.Delete(c.Request().Context(), parsedUUID); err != nil {
		slog.Error("Failed to delete rule", "rule_id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete rule"})
	}

	// Invalidate cache - log error but don't fail the request
	if err := s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID); err != nil {
		slog.Warn("Failed to invalidate cache after rule deletion", "rule_id", rule.RuleID, "error", err)
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "delete")

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) patchRule(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var req struct {
		Enabled *bool   `json:"enabled,omitempty"`
		Name    *string `json:"name,omitempty"`
		Message *string `json:"message,omitempty"`
		Pattern *string `json:"pattern,omitempty"`
		Severity *string `json:"severity,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Get existing rule
	rule, err := s.ruleStore.GetByID(c.Request().Context(), parsedUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	// Apply patches
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Message != nil {
		rule.Message = *req.Message
	}
	if req.Pattern != nil {
		rule.Pattern = *req.Pattern
	}
	if req.Severity != nil {
		rule.Severity = models.Severity(*req.Severity)
	}

	if err := s.ruleStore.Update(c.Request().Context(), rule); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID)

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "patch")

	return c.JSON(http.StatusOK, rule)
}

// Project handlers

func (s *Server) listProjects(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > maxPageLimit {
		limit = defaultPageLimit
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	projects, err := s.projStore.List(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": projects,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (s *Server) getProject(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	proj, err := s.projStore.GetByID(c.Request().Context(), parsedUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, proj)
}

func (s *Server) createProject(c echo.Context) error {
	var proj models.Project
	if err := c.Bind(&proj); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.projStore.Create(c.Request().Context(), &proj); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, proj)
}

func (s *Server) updateProject(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var proj models.Project
	if err := c.Bind(&proj); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	proj.ID = parsedUUID
	if err := s.projStore.Update(c.Request().Context(), &proj); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnProjectChange(c.Request().Context(), proj.Slug)

	return c.JSON(http.StatusOK, proj)
}

func (s *Server) deleteProject(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	if err := s.projStore.Delete(c.Request().Context(), parsedUUID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// Failure handlers

func (s *Server) listFailures(c echo.Context) error {
	status := c.QueryParam("status")
	category := c.QueryParam("category")
	projectSlug := c.QueryParam("project")

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > maxPageLimit {
		limit = defaultPageLimit
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	failures, total, err := s.failStore.List(c.Request().Context(), status, category, projectSlug, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": failures,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

func (s *Server) getFailure(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	failure, err := s.failStore.GetByID(c.Request().Context(), parsedUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, failure)
}

func (s *Server) createFailure(c echo.Context) error {
	var failure models.FailureEntry
	if err := c.Bind(&failure); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.failStore.Create(c.Request().Context(), &failure); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, failure)
}

func (s *Server) updateFailure(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var failure models.FailureEntry
	if err := c.Bind(&failure); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	failure.ID = parsedUUID
	if err := s.failStore.Update(c.Request().Context(), &failure); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, failure)
}

// System handlers

func (s *Server) getStats(c echo.Context) error {
	ctx := c.Request().Context()

	failCount, err := s.failStore.Count(ctx)
	if err != nil {
		slog.Error("Failed to get failure count", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve statistics"})
	}

	// TODO: Add counts for other entities
	return c.JSON(http.StatusOK, map[string]interface{}{
		"documents_count": 0, // TODO: Implement count
		"rules_count":     0, // TODO: Implement count
		"projects_count":  0, // TODO: Implement count
		"failures_count":  failCount,
	})
}

func (s *Server) triggerIngest(c echo.Context) error {
	// TODO: Implement ingest trigger
	return c.JSON(http.StatusOK, map[string]string{"status": "ingest started"})
}

// IDE API handlers

func (s *Server) ideHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) validateFile(c echo.Context) error {
	// TODO: Implement file validation using guardrails
	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":      true,
		"violations": []interface{}{},
	})
}

func (s *Server) validateSelection(c echo.Context) error {
	// TODO: Implement selection validation
	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":      true,
		"violations": []interface{}{},
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
	// TODO: Implement quick reference - get from documents
	return c.JSON(http.StatusOK, map[string]string{
		"reference": "Quick reference documentation - TODO: Load from docs",
	})
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
