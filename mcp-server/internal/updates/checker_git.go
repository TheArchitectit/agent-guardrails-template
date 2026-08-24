package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

func (c *Checker) getCurrentGitCommit() string {
	// First check environment variable
	if commit := os.Getenv("GIT_COMMIT"); commit != "" && commit != "unknown" {
		return commit
	}

	// Check commit file
	if data, err := os.ReadFile("/app/git-commit"); err == nil {
		return strings.TrimSpace(string(data))
	}

	// Fall back to build commit
	if c.gitCommit != "" && c.gitCommit != "unknown" {
		return c.gitCommit
	}

	// Try to get from git command
	if commit, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		return strings.TrimSpace(string(commit))
	}

	return ""
}

// getLatestLocalCommit tries to get the latest commit from local git
func (c *Checker) getLatestLocalCommit(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "origin/main")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// fetchLatestGitHubCommit fetches the latest commit from GitHub API
func (c *Checker) fetchLatestGitHubCommit(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubCommitsURL, nil)
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "guardrail-mcp-updater/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var commitResponse struct {
		SHA string `json:"sha"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commitResponse); err != nil {
		return "", fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return commitResponse.SHA, nil
}

// countGitChanges counts file changes between two commits
func (c *Checker) countGitChanges(ctx context.Context, currentCommit, latestCommit string) (int, int, int, error) {
	if currentCommit == "" || latestCommit == "" {
		return 0, 0, 0, fmt.Errorf("invalid commit hashes")
	}

	cmd := exec.CommandContext(ctx, "git", "diff", "--stat", currentCommit, latestCommit)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}

	// Parse git diff --stat output
	// Example: " file.go | 10 +++++-----"
	lines := strings.Split(string(output), "\n")
	newFiles, modifiedFiles, deletedFiles := 0, 0, 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "|") == false {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}

		filename := strings.TrimSpace(parts[0])
		// Check for special indicators
		if strings.HasPrefix(filename, "{") && strings.Contains(filename, "=>") {
			// Renamed file
			modifiedFiles++
		} else if strings.Contains(line, "Bin") {
			// Binary file
			modifiedFiles++
		} else {
			// Regular file change
			modifiedFiles++
		}
	}

	// Try to get more detailed stats using git diff --numstat
	cmd = exec.CommandContext(ctx, "git", "diff", "--numstat", currentCommit, latestCommit)
	output, err = cmd.Output()
	if err == nil {
		newFiles, modifiedFiles, deletedFiles = c.parseNumstat(string(output))
	}

	return newFiles, modifiedFiles, deletedFiles, nil
}

// parseNumstat parses git diff --numstat output
func (c *Checker) parseNumstat(output string) (int, int, int) {
	lines := strings.Split(output, "\n")
	newFiles, modifiedFiles, deletedFiles := 0, 0, 0

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		added := fields[0]
		deleted := fields[1]

		if added == "-" && deleted == "-" {
			// Binary file
			modifiedFiles++
		} else if added == "0" {
			// Only deletions - file was deleted
			deletedFiles++
		} else if deleted == "0" {
			// Only additions - new file
			newFiles++
		} else {
			// Both additions and deletions - modified
			modifiedFiles++
		}
	}

	return newFiles, modifiedFiles, deletedFiles
}

// isNewerVersion compares two version strings
func (c *Checker) isNewerVersion(current, latest string) bool {
	// Handle "latest" tag specially
	if current == "latest" {
		return false
	}

	// Normalize versions
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// Simple string comparison for now
	// In production, use semantic versioning library
	return current != latest && latest != "" && latest != "unknown"
}

// saveCheckResult saves the update check result to the database
func (c *Checker) saveCheckResult(ctx context.Context, check *models.UpdateCheck) error {
	metadataJSON, err := json.Marshal(check.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO update_checks (
			id, checked_at,
			docker_current_version, docker_latest_version, docker_release_notes, docker_update_available,
			guardrail_current_commit, guardrail_latest_commit,
			guardrail_new_files, guardrail_modified_files, guardrail_deleted_files,
			guardrail_update_available, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = c.db.ExecContext(ctx, query,
		check.ID, check.CheckedAt,
		check.DockerCurrentVersion, check.DockerLatestVersion, check.DockerReleaseNotes, check.DockerUpdateAvailable,
		check.GuardrailCurrentCommit, check.GuardrailLatestCommit,
		check.GuardrailNewFiles, check.GuardrailModifiedFiles, check.GuardrailDeletedFiles,
		check.GuardrailUpdateAvailable, metadataJSON,
	)

	return err
}

// GetLatestCheck retrieves the most recent update check from the database
func (c *Checker) GetLatestCheck(ctx context.Context) (*models.UpdateCheck, error) {
	query := `
		SELECT id, checked_at,
			docker_current_version, docker_latest_version, docker_release_notes, docker_update_available,
			guardrail_current_commit, guardrail_latest_commit,
			guardrail_new_files, guardrail_modified_files, guardrail_deleted_files,
			guardrail_update_available, metadata
		FROM update_checks
		ORDER BY checked_at DESC
		LIMIT 1
	`

	check := &models.UpdateCheck{}
	var metadataJSON []byte

	err := c.db.QueryRowContext(ctx, query).Scan(
		&check.ID, &check.CheckedAt,
		&check.DockerCurrentVersion, &check.DockerLatestVersion, &check.DockerReleaseNotes, &check.DockerUpdateAvailable,
		&check.GuardrailCurrentCommit, &check.GuardrailLatestCommit,
		&check.GuardrailNewFiles, &check.GuardrailModifiedFiles, &check.GuardrailDeletedFiles,
		&check.GuardrailUpdateAvailable, &metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &check.Metadata)
	}

	return check, nil
}

// ToStatusResponse converts an UpdateCheck to an UpdateStatusResponse
func ToStatusResponse(check *models.UpdateCheck) *models.UpdateStatusResponse {
	if check == nil {
		return &models.UpdateStatusResponse{
			LastChecked: time.Time{},
		}
	}

	response := &models.UpdateStatusResponse{
		LastChecked: check.CheckedAt,
	}

	if check.DockerUpdateAvailable {
		response.DockerUpdate = &models.DockerUpdateInfo{
			CurrentVersion: check.DockerCurrentVersion,
			LatestVersion:  check.DockerLatestVersion,
			ReleaseNotes:   check.DockerReleaseNotes,
		}
	}

	if check.GuardrailUpdateAvailable {
		response.GuardrailUpdate = &models.GuardrailUpdateInfo{
			CurrentCommit: check.GuardrailCurrentCommit,
			LatestCommit:  check.GuardrailLatestCommit,
			NewFiles:      check.GuardrailNewFiles,
			ModifiedFiles: check.GuardrailModifiedFiles,
			DeletedFiles:  check.GuardrailDeletedFiles,
		}
	}

	return response
}
