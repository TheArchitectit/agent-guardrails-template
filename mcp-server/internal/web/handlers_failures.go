package web

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

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

	failures, err := s.failStore.List(c.Request().Context(), status, category, projectSlug, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": failures,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"count":  len(failures),
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
