// Package team provides team management functionality for MCP server.
// This is a Go port of the Python team_manager.py core functionality.
package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TeamMember represents a person assigned to a team role
type TeamMember struct {
	Person     string `json:"person"`
	Role       string `json:"role"`
	AssignedAt string `json:"assigned_at"`
}

// TeamAssignment represents a role assignment
type TeamAssignment struct {
	TeamID     int    `json:"team_id"`
	RoleName   string `json:"role_name"`
	Person     string `json:"person"`
	AssignedAt string `json:"assigned_at"`
}

// PhaseStatus represents the status of a phase
type PhaseStatus struct {
	Phase       string  `json:"phase"`
	TotalTeams  int     `json:"total_teams"`
	Completed   int     `json:"completed"`
	Active      int     `json:"active"`
	NotStarted  int     `json:"not_started"`
	ProgressPct float64 `json:"progress_pct"`
}

// TeamListItem represents a team in list output
type TeamListItem struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Phase         string `json:"phase"`
	Status        string `json:"status"`
	AssignedCount int    `json:"assigned_count"`
	TotalRoles    int    `json:"total_roles"`
}

// Manager handles team operations with thread-safe access
type Manager struct {
	projectName string
	baseDir     string
	configPath  string
	lockPath    string
	teams       map[int]Team
	mu          sync.RWMutex
}

// ManagerOption configures the Manager
type ManagerOption func(*Manager)

// WithBaseDir sets the base directory for team files
func WithBaseDir(dir string) ManagerOption {
	return func(m *Manager) {
		m.baseDir = dir
	}
}

// WithTestMode enables test mode (no auth checks)
func WithTestMode(enabled bool) ManagerOption {
	return func(m *Manager) {
		// Test mode is handled by the caller
		_ = enabled
	}
}

// NewManager creates a new team manager
func NewManager(projectName string, opts ...ManagerOption) (*Manager, error) {
	if err := ValidateProjectName(projectName); err != nil {
		return nil, fmt.Errorf("invalid project name: %w", err)
	}

	m := &Manager{
		projectName: projectName,
		baseDir:     ".teams",
		teams:       make(map[int]Team),
	}

	for _, opt := range opts {
		opt(m)
	}

	// Validate and set paths
	configPath, err := ValidateProjectPath(projectName, m.baseDir)
	if err != nil {
		return nil, err
	}
	m.configPath = configPath
	m.lockPath = strings.TrimSuffix(configPath, ".json") + ".lock"

	return m, nil
}

// InitProject initializes a new project with all standard teams (deprecated, use InitializeProject)
func (m *Manager) InitProject() error {
	return m.InitializeProject()
}

// InitializeProject initializes a new project with all standard teams
func (m *Manager) InitializeProject() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize with standard teams (deep copy)
	for id, team := range StandardTeams {
		m.teams[id] = copyTeam(team)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := m.save(); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// Load loads team configuration from disk
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return fmt.Errorf("project not found: %s", m.projectName)
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var projectData ProjectData
	if err := json.Unmarshal(data, &projectData); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	m.teams = make(map[int]Team)
	for _, team := range projectData.Teams {
		m.teams[team.ID] = team
	}

	return nil
}

// save persists team configuration to disk (must hold write lock)
func (m *Manager) save() error {
	projectData := ProjectData{
		ProjectName: m.projectName,
		Version:     "1.0.0",
		UpdatedAt:   time.Now(),
		Teams:       make([]Team, 0, len(m.teams)),
	}

	for _, team := range m.teams {
		projectData.Teams = append(projectData.Teams, team)
	}

	data, err := json.MarshalIndent(projectData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tempPath := m.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, m.configPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
