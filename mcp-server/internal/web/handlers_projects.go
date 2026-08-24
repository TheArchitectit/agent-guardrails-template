package web

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// Project handlers

func (s *Server) listProjects(c echo.Context) error {
	ctx := c.Request().Context()
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit <= 0 || limit > maxPageLimit {
		limit = defaultPageLimit
	}
	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	projects, err := s.projStore.List(ctx, limit, offset)
	if err != nil {
		slog.Error("Failed to list projects", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve projects"})
	}

	total, err := s.projStore.Count(ctx)
	if err != nil {
		slog.Warn("Failed to count projects", "error", err)
		total = len(projects) // Fallback to current page size
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": projects,
		"pagination": map[string]interface{}{
			"total":  total,
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

	// Get project to find the slug
	proj, err := s.projStore.GetByID(c.Request().Context(), parsedUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	if err := s.projStore.Delete(c.Request().Context(), proj.Slug); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Invalidate cache
	s.cache.InvalidateOnProjectChange(c.Request().Context(), proj.Slug)

	return c.NoContent(http.StatusNoContent)
}
