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

	// encryption seals team data at rest (SEC-007 / Spec 17 R17-06). A disabled
	// manager (no key configured) writes plaintext and reads plaintext only;
	// a sealed value read without a key is an error, never returned as if it
	// were the assignee name.
	encryption    *EncryptionManager
	encryptionKey string
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

// WithEncryptionKey configures the key that seals team data at rest
// (TEAM_ENCRYPTION_KEY, SEC-007 / Spec 17 R17-06). An empty key leaves the
// manager unsealed, which is the documented `local` posture. A key that cannot
// be used fails NewManager rather than producing a manager that silently
// protects nothing.
func WithEncryptionKey(key string) ManagerOption {
	return func(m *Manager) {
		m.encryptionKey = key
	}
}

// WithEncryptionManager installs a process-wide encryption manager so key
// resolution happens once at startup instead of per request.
func WithEncryptionManager(em *EncryptionManager) ManagerOption {
	return func(m *Manager) {
		m.encryption = em
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

	// Resolve the at-rest key once, here, so every save/load path in this
	// process uses the same manager and an unusable key stops the manager at
	// construction rather than on the first write.
	if m.encryption == nil {
		em, err := NewEncryptionManager(m.encryptionKey)
		if err != nil {
			return nil, err
		}
		m.encryption = em
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
		opened, err := m.encryption.openTeam(team)
		if err != nil {
			return fmt.Errorf("failed to open team %d: %w", team.ID, err)
		}
		m.teams[opened.ID] = opened
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
		sealed, err := m.encryption.sealTeam(team)
		if err != nil {
			return err
		}
		projectData.Teams = append(projectData.Teams, sealed)
	}

	data, err := json.MarshalIndent(projectData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Atomic write: write to temp file, then rename. Team data carries personal
	// names, so the file is owner-only (R17-06); 0644 left every local account
	// able to read the assignments.
	tempPath := m.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to restrict temp file mode: %w", err)
	}

	if err := os.Rename(tempPath, m.configPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
