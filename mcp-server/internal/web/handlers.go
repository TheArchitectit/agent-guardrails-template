package web

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/security"
)

// Document handlers

func (s *Server) listDocuments(c echo.Context) error {
	category := c.QueryParam("category")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	docs, err := s.docStore.List(c.Request().Context(), category, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, docs)
}

func (s *Server) getDocument(c echo.Context) error {
	id := c.Param("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	doc, err := s.docStore.GetByID(c.Request().Context(), uuid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, doc)
}

func (s *Server) updateDocument(c echo.Context) error {
	id := c.Param("id")
	uuid, err := uuid.Parse(id)
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

	doc.ID = uuid
	if err := s.docStore.Update(c.Request().Context(), &doc); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnDocumentChange(c.Request().Context(), doc.Slug)

	// Audit log
	keyHash, _ := c.Get("api_key_hash").(string)
	s.auditLogger.LogDocChange(c.Request().Context(), keyHash, doc.Slug, "update")

	return c.JSON(http.StatusOK, doc)
}

func (s *Server) searchDocuments(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query parameter required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	docs, err := s.docStore.Search(c.Request().Context(), query, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, docs)
}

// Rule handlers

func (s *Server) listRules(c echo.Context) error {
	var enabled *bool
	if enabledParam := c.QueryParam("enabled"); enabledParam != "" {
		e := enabledParam == "true"
		enabled = &e
	}
	category := c.QueryParam("category")

	rules, err := s.ruleStore.List(c.Request().Context(), enabled, category)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, rules)
}

func (s *Server) getRule(c echo.Context) error {
	id := c.Param("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	rule, err := s.ruleStore.GetByID(c.Request().Context(), uuid)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, rule)
}

func (s *Server) createRule(c echo.Context) error {
	var rule models.PreventionRule
	if err := c.Bind(&rule); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.ruleStore.Create(c.Request().Context(), &rule); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID)

	// Audit log
	keyHash, _ := c.Get("api_key_hash").(string)
	s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "create")

	return c.JSON(http.StatusCreated, rule)
}

func (s *Server) updateRule(c echo.Context) error {
	id := c.Param("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var rule models.PreventionRule
	if err := c.Bind(&rule); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	rule.ID = uuid
	if err := s.ruleStore.Update(c.Request().Context(), &rule); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID)

	// Audit log
	keyHash, _ := c.Get("api_key_hash").(string)
	s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "update")

	return c.JSON(http.StatusOK, rule)
}

func (s *Server) deleteRule(c echo.Context) error {
	id := c.Param("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	// Get rule for cache invalidation before deleting
	rule, _ := s.ruleStore.GetByID(c.Request().Context(), uuid)

	if err := s.ruleStore.Delete(c.Request().Context(), uuid); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	if rule != nil {
		s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID)
	}

	// Audit log
	keyHash, _ := c.Get("api_key_hash").(string)
	if rule != nil {
		s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "delete")
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) toggleRule(c echo.Context) error {
	id := c.Param("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.ruleStore.Toggle(c.Request().Context(), uuid, req.Enabled); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Get rule for cache invalidation
	rule, _ := s.ruleStore.GetByID(c.Request().Context(), uuid)
	if rule != nil {
		s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID)

		// Audit log
		keyHash, _ := c.Get("api_key_hash").(string)
		s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "toggle")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

// Project handlers

func (s *Server) listProjects(c echo.Context) error {
	projects, err := s.projStore.List(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, projects)
}

func (s *Server) getProject(c echo.Context) error {
	slug := c.Param("slug")

	proj, err := s.projStore.GetBySlug(c.Request().Context(), slug)
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
	slug := c.Param("slug")

	var proj models.Project
	if err := c.Bind(&proj); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	proj.Slug = slug
	if err := s.projStore.Update(c.Request().Context(), &proj); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnProjectChange(c.Request().Context(), slug)

	return c.JSON(http.StatusOK, proj)
}

func (s *Server) deleteProject(c echo.Context) error {
	slug := c.Param("slug")

	if err := s.projStore.Delete(c.Request().Context(), slug); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnProjectChange(c.Request().Context(), slug)

	return c.NoContent(http.StatusNoContent)
}

// Failure handlers

func (s *Server) listFailures(c echo.Context) error {
	status := c.QueryParam("status")
	category := c.QueryParam("category")
	projectSlug := c.QueryParam("project")

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	failures, err := s.failStore.List(c.Request().Context(), status, category, projectSlug, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, failures)
}

func (s *Server) getFailure(c echo.Context) error {
	id := c.Param("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	failure, err := s.failStore.GetByID(c.Request().Context(), uuid)
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
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	var failure models.FailureEntry
	if err := c.Bind(&failure); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	failure.ID = uuid
	if err := s.failStore.Update(c.Request().Context(), &failure); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, failure)
}

// System handlers

func (s *Server) getStats(c echo.Context) error {
	ctx := c.Request().Context()

	failCount, _ := s.failStore.Count(ctx)

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
	projectSlug := c.QueryParam("project")

	var rules []models.PreventionRule
	var err error

	if projectSlug != "" {
		// Get project to find active rules
		proj, err := s.projStore.GetBySlug(c.Request().Context(), projectSlug)
		if err == nil && len(proj.ActiveRules) > 0 {
			// Get specific active rules for project
			for _, ruleID := range proj.ActiveRules {
				rule, err := s.ruleStore.GetByRuleID(c.Request().Context(), ruleID)
				if err == nil && rule.Enabled {
					rules = append(rules, *rule)
				}
			}
		}
	}

	// If no project-specific rules, get all active rules
	if len(rules) == 0 {
		rules, err = s.ruleStore.GetActiveRules(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, rules)
}

func (s *Server) getQuickReference(c echo.Context) error {
	// TODO: Implement quick reference - get from documents
	return c.JSON(http.StatusOK, map[string]string{
		"reference": "Quick reference documentation - TODO: Load from docs",
	})
}
