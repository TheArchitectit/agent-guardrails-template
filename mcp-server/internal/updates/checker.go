package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

const (
	// Default timeout for external checks
	checkTimeout = 10 * time.Second

	// Docker Hub API endpoint for guardrail-mcp
	dockerHubTagsURL = "https://hub.docker.com/v2/repositories/thearchitectit/guardrail-mcp/tags/?page_size=1&ordering=last_updated"

	// GitHub API endpoint for latest commit
	githubCommitsURL = "https://api.github.com/repos/thearchitectit/guardrail-mcp/commits/main"
)

// Checker handles update checking operations
type Checker struct {
	db         *database.DB
	httpClient *http.Client
	version    string
	gitCommit  string
}

// NewChecker creates a new update checker
func NewChecker(db *database.DB, version, gitCommit string) *Checker {
	return &Checker{
		db: db,
		httpClient: &http.Client{
			Timeout: checkTimeout,
		},
		version:   version,
		gitCommit: gitCommit,
	}
}

// CheckResult contains the results of an update check
type CheckResult struct {
	DockerCurrentVersion     string
	DockerLatestVersion      string
	DockerReleaseNotes       string
	DockerUpdateAvailable    bool
	GuardrailCurrentCommit   string
	GuardrailLatestCommit    string
	GuardrailNewFiles        int
	GuardrailModifiedFiles   int
	GuardrailDeletedFiles    int
	GuardrailUpdateAvailable bool
	Metadata                 map[string]any
}

// Check performs a full update check and saves results to database
func (c *Checker) Check(ctx context.Context) (*models.UpdateCheck, error) {
	result, err := c.performCheck(ctx)
	if err != nil {
		slog.Error("Update check failed", "error", err)
		// Continue to save partial results
	}

	// Create update check record
	check := &models.UpdateCheck{
		ID:                       uuid.New(),
		CheckedAt:                time.Now().UTC(),
		DockerCurrentVersion:     result.DockerCurrentVersion,
		DockerLatestVersion:      result.DockerLatestVersion,
		DockerReleaseNotes:       result.DockerReleaseNotes,
		DockerUpdateAvailable:    result.DockerUpdateAvailable,
		GuardrailCurrentCommit:   result.GuardrailCurrentCommit,
		GuardrailLatestCommit:    result.GuardrailLatestCommit,
		GuardrailNewFiles:        result.GuardrailNewFiles,
		GuardrailModifiedFiles:   result.GuardrailModifiedFiles,
		GuardrailDeletedFiles:    result.GuardrailDeletedFiles,
		GuardrailUpdateAvailable: result.GuardrailUpdateAvailable,
		Metadata:                 result.Metadata,
	}

	// Save to database
	if err := c.saveCheckResult(ctx, check); err != nil {
		slog.Error("Failed to save update check result", "error", err)
		return nil, fmt.Errorf("failed to save check result: %w", err)
	}

	slog.Info("Update check completed",
		"docker_update_available", check.DockerUpdateAvailable,
		"guardrail_update_available", check.GuardrailUpdateAvailable,
	)

	return check, nil
}

// performCheck executes all update checks
func (c *Checker) performCheck(ctx context.Context) (*CheckResult, error) {
	result := &CheckResult{
		Metadata: make(map[string]any),
	}

	// Check Docker version
	dockerErr := c.checkDockerVersion(ctx, result)
	if dockerErr != nil {
		result.Metadata["docker_check_error"] = dockerErr.Error()
		slog.Warn("Docker version check failed", "error", dockerErr)
	}

	// Check Git repository for updates
	gitErr := c.checkGitUpdates(ctx, result)
	if gitErr != nil {
		result.Metadata["git_check_error"] = gitErr.Error()
		slog.Warn("Git update check failed", "error", gitErr)
	}

	return result, nil
}

// checkDockerVersion checks for available Docker image updates
func (c *Checker) checkDockerVersion(ctx context.Context, result *CheckResult) error {
	// Get current version
	result.DockerCurrentVersion = c.getCurrentDockerVersion()

	// Fetch latest version from Docker Hub
	latestVersion, releaseNotes, err := c.fetchLatestDockerVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest Docker version: %w", err)
	}

	result.DockerLatestVersion = latestVersion
	result.DockerReleaseNotes = releaseNotes
	result.DockerUpdateAvailable = c.isNewerVersion(result.DockerCurrentVersion, result.DockerLatestVersion)

	return nil
}

// getCurrentDockerVersion returns the current Docker version
func (c *Checker) getCurrentDockerVersion() string {
	// First check environment variable
	if version := os.Getenv("DOCKER_IMAGE_VERSION"); version != "" {
		return version
	}

	// Check version file
	if data, err := os.ReadFile("/app/version"); err == nil {
		return strings.TrimSpace(string(data))
	}

	// Fallback to build version
	if c.version != "" && c.version != "dev" {
		return c.version
	}

	return "unknown"
}

// fetchLatestDockerVersion fetches the latest version from Docker Hub
func (c *Checker) fetchLatestDockerVersion(ctx context.Context) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dockerHubTagsURL, nil)
	if err != nil {
		return "", "", err
	}

	// Set headers to avoid rate limiting
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "guardrail-mcp-updater/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Docker Hub API returned status %d", resp.StatusCode)
	}

	var dockerResponse struct {
		Results []struct {
			Name        string `json:"name"`
			LastUpdated string `json:"last_updated"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&dockerResponse); err != nil {
		return "", "", fmt.Errorf("failed to decode Docker Hub response: %w", err)
	}

	if len(dockerResponse.Results) == 0 {
		return "", "", fmt.Errorf("no tags found in Docker Hub response")
	}

	latestTag := dockerResponse.Results[0]
	releaseNotes := fmt.Sprintf("https://hub.docker.com/r/thearchitectit/guardrail-mcp/tags?name=%s",
		latestTag.Name)

	return latestTag.Name, releaseNotes, nil
}

// checkGitUpdates checks for git repository updates
func (c *Checker) checkGitUpdates(ctx context.Context, result *CheckResult) error {
	// Get current commit
	result.GuardrailCurrentCommit = c.getCurrentGitCommit()

	// Try to get latest commit from local git if available
	if latest, err := c.getLatestLocalCommit(ctx); err == nil && latest != "" {
		result.GuardrailLatestCommit = latest
	} else {
		// Fall back to GitHub API
		latest, err := c.fetchLatestGitHubCommit(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch latest commit: %w", err)
		}
		result.GuardrailLatestCommit = latest
	}

	// Check if commits differ
	if result.GuardrailCurrentCommit != "" && result.GuardrailLatestCommit != "" {
		result.GuardrailUpdateAvailable = result.GuardrailCurrentCommit != result.GuardrailLatestCommit
	}

	// Count file changes if update is available
	if result.GuardrailUpdateAvailable {
		newFiles, modifiedFiles, deletedFiles, err := c.countGitChanges(ctx,
			result.GuardrailCurrentCommit, result.GuardrailLatestCommit)
		if err == nil {
			result.GuardrailNewFiles = newFiles
			result.GuardrailModifiedFiles = modifiedFiles
			result.GuardrailDeletedFiles = deletedFiles
		}
	}

	return nil
}

// getCurrentGitCommit returns the current git commit hash
