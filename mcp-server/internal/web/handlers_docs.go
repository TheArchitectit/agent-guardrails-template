package web

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/security"
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
