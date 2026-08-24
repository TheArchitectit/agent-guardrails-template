package web

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/thearchitectit/guardrail-mcp/internal/ingest"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/updates"
)

// System handlers

func (s *Server) getStats(c echo.Context) error {
	ctx := c.Request().Context()

	// Fetch all counts concurrently for better performance
	type countResult struct {
		name  string
		count int64
		err   error
	}

	counts := make(chan countResult, 4)

	// Document count
	go func() {
		count, err := s.docStore.Count(ctx, "")
		counts <- countResult{"documents", int64(count), err}
	}()

	// Rule count
	go func() {
		count, err := s.ruleStore.Count(ctx, nil, "")
		counts <- countResult{"rules", int64(count), err}
	}()

	// Project count
	go func() {
		count, err := s.projStore.Count(ctx)
		counts <- countResult{"projects", int64(count), err}
	}()

	// Failure count
	go func() {
		count, err := s.failStore.Count(ctx)
		counts <- countResult{"failures", count, err}
	}()

	// Collect results
	stats := make(map[string]int64)
	for i := 0; i < 4; i++ {
		result := <-counts
		if result.err != nil {
			slog.Error("Failed to get count", "entity", result.name, "error", result.err)
			stats[result.name] = 0
		} else {
			stats[result.name] = result.count
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"documents_count": stats["documents"],
		"rules_count":     stats["rules"],
		"projects_count":  stats["projects"],
		"failures_count":  stats["failures"],
	})
}

func (s *Server) triggerIngest(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse optional request body for ingest options
	var req struct {
		RepoPath    string   `json:"repo_path,omitempty"`
		Force       bool     `json:"force,omitempty"`
		ProjectSlug string   `json:"project_slug,omitempty"`
		Categories  []string `json:"categories,omitempty"`
	}
	_ = c.Bind(&req) // Optional, ignore error

	// Create a new ingest job
	jobID := uuid.New()
	slog.Info("Starting document ingest job", "job_id", jobID, "repo_path", req.RepoPath, "project_slug", req.ProjectSlug)

	// Trigger ingest in background to not block the response
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if req.RepoPath != "" {
			// Custom repo path provided, would need custom implementation
			slog.Info("Custom repo path ingest requested", "job_id", jobID, "path", req.RepoPath)
			// For now, fall through to default sync
		}

		// Trigger the ingest from watched directories
		if err := s.ingestSvc.SyncFromRepo(bgCtx, jobID); err != nil {
			slog.Error("Failed to ingest documents", "job_id", jobID, "error", err)
		} else {
			slog.Info("Document ingest completed", "job_id", jobID)
		}
	}()

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogDocChange(ctx, keyHash, fmt.Sprintf("ingest:%s", jobID), "ingest")

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"job_id":       jobID,
		"status":       "running",
		"message":      "Document ingestion started",
		"repo_path":    req.RepoPath,
		"project_slug": req.ProjectSlug,
	})
}

// uploadFiles handles multipart file uploads for document ingestion
func (s *Server) uploadFiles(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse multipart form with 50MB max memory
	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
	}
	defer form.RemoveAll()

	files := form.File["files"]
	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no files provided"})
	}

	// Create a new ingest job
	jobID := uuid.New()
	slog.Info("Starting rule sync job", "job_id", jobID)
	fileContents := make(map[string][]byte)

	// Process each uploaded file
	var processedFiles []string
	var skippedFiles []string

	for _, fileHeader := range files {
		// Validate file type
		if !ingest.IsMarkdownFile(fileHeader.Filename) {
			skippedFiles = append(skippedFiles, fileHeader.Filename)
			continue
		}

		// Open the uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			slog.Error("Failed to open uploaded file", "filename", fileHeader.Filename, "error", err)
			continue
		}

		// Read file content
		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			slog.Error("Failed to read uploaded file", "filename", fileHeader.Filename, "error", err)
			continue
		}

		fileContents[fileHeader.Filename] = content
		processedFiles = append(processedFiles, fileHeader.Filename)
	}

	// Process the files through the ingest service
	if err := s.ingestSvc.SyncFromUpload(ctx, jobID, fileContents); err != nil {
		slog.Error("Failed to process uploaded files", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to process files"})
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogDocChange(ctx, keyHash, fmt.Sprintf("upload:%s", jobID), "ingest")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"job_id":    jobID,
		"processed": len(processedFiles),
		"skipped":   skippedFiles,
		"files":     processedFiles,
		"status":    "completed",
	})
}

// syncRepo triggers a repository sync
func (s *Server) syncRepo(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse optional request body for sync options
	var req struct {
		Force bool `json:"force,omitempty"`
	}
	_ = c.Bind(&req) // Optional, ignore error

	// Create a new ingest job
	jobID := uuid.New()
	slog.Info("Starting rule sync job", "job_id", jobID)

	// Trigger sync in background to not block the response
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := s.ingestSvc.SyncFromRepo(bgCtx, jobID); err != nil {
			slog.Error("Failed to sync from repo", "job_id", jobID, "error", err)
		}
	}()

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogDocChange(ctx, keyHash, fmt.Sprintf("sync:%s", jobID), "ingest")

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"job_id":  jobID,
		"status":  "running",
		"message": "Repository sync started",
	})
}

// getIngestStatus returns the status of the last sync operation
func (s *Server) getIngestStatus(c echo.Context) error {
	// For now, return a placeholder status
	// In a full implementation, this would query the ingest_jobs table
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":          "completed",
		"last_sync":       time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
		"files_processed": 0,
		"files_added":     0,
		"files_updated":   0,
		"files_orphaned":  0,
		"errors":          []interface{}{},
	})
}

// listOrphans returns a list of orphaned documents
func (s *Server) listOrphans(c echo.Context) error {
	_ = c.Request().Context() // Reserved for future use

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit <= 0 || limit > maxPageLimit {
		limit = defaultPageLimit
	}
	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	// Query orphaned documents using the existing List method with a filter
	// In a full implementation, this would query with orphaned=true filter
	// For now, return an empty list
	docs := []models.Document{}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": docs,
		"pagination": map[string]interface{}{
			"total":  0,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// deleteOrphan deletes a single orphaned document
func (s *Server) deleteOrphan(c echo.Context) error {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id format"})
	}

	ctx := c.Request().Context()

	// Get document to verify it exists and is orphaned
	doc, err := s.docStore.GetByID(ctx, parsedUUID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "document not found"})
	}

	// Verify the document is orphaned
	if !doc.Orphaned {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "document is not orphaned"})
	}

	// Delete the document
	if err := s.docStore.Delete(ctx, parsedUUID); err != nil {
		slog.Error("Failed to delete orphaned document", "doc_id", id, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete document"})
	}

	// Invalidate cache
	if err := s.cache.InvalidateOnDocumentChange(ctx, doc.Slug); err != nil {
		slog.Warn("Failed to invalidate document cache", "slug", doc.Slug, "error", err)
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogDocChange(ctx, keyHash, doc.Slug, "delete_orphan")

	return c.NoContent(http.StatusNoContent)
}

// getUpdateStatus returns the current update status
func (s *Server) getUpdateStatus(c echo.Context) error {
	ctx := c.Request().Context()

	// Get the latest update check from database
	latestCheck, err := s.updateChecker.GetLatestCheck(ctx)
	if err != nil {
		// No check has been performed yet
		return c.JSON(http.StatusOK, map[string]interface{}{
			"last_checked":      nil,
			"updates_available": false,
			"message":           "No update check has been performed yet. Use POST /api/updates/check to trigger a check.",
		})
	}

	// Convert to response format
	response := updates.ToStatusResponse(latestCheck)

	return c.JSON(http.StatusOK, response)
}

// checkForUpdates triggers a manual update check
func (s *Server) checkForUpdates(c echo.Context) error {
	ctx := c.Request().Context()

	// Perform the update check
	check, err := s.updateChecker.Check(ctx)
	if err != nil {
		slog.Error("Update check failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to check for updates",
		})
	}

	// Audit log
	keyHash := getAPIKeyHash(c)
	s.auditLogger.LogDocChange(ctx, keyHash, "update-check", "check")

	// Return the result
	response := updates.ToStatusResponse(check)

	return c.JSON(http.StatusOK, response)
}
