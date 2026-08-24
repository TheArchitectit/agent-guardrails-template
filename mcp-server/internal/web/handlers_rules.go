package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// Rule handlers

func (s *Server) listRules(c echo.Context) error {
	ctx := c.Request().Context()
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

	rules, err := s.ruleStore.List(ctx, enabled, category, limit, offset)
	if err != nil {
		slog.Error("Failed to list rules", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve rules"})
	}

	total, err := s.ruleStore.Count(ctx, enabled, category)
	if err != nil {
		slog.Warn("Failed to count rules", "error", err)
		total = len(rules) // Fallback to current page size
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": rules,
		"pagination": map[string]interface{}{
			"total":  total,
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
		Enabled  *bool   `json:"enabled,omitempty"`
		Name     *string `json:"name,omitempty"`
		Message  *string `json:"message,omitempty"`
		Pattern  *string `json:"pattern,omitempty"`
		Severity *string `json:"severity,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Get existing rule
	rule, err := s.ruleStore.GetByID(c.Request().Context(), parsedUUID)
	if err != nil {
		if err.Error() == fmt.Sprintf("rule not found: %s", parsedUUID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "rule not found"})
		}
		slog.Error("Failed to get rule for patch", "rule_id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve rule"})
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
		slog.Error("Failed to patch rule", "rule_id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update rule"})
	}

	// Invalidate cache - log error but don't fail the request
	if err := s.cache.InvalidateOnRuleChange(c.Request().Context(), rule.RuleID); err != nil {
		slog.Warn("Failed to invalidate rule cache after patch", "rule_id", rule.RuleID, "error", err)
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogRuleChange(c.Request().Context(), keyHash, rule.RuleID, "patch")

	return c.JSON(http.StatusOK, rule)
}
