package team

import "fmt"

// GetTeamByID returns a team by ID
func (m *Manager) GetTeamByID(teamID int) (Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, exists := m.teams[teamID]
	if !exists {
		return Team{}, fmt.Errorf("team %d not found", teamID)
	}

	return copyTeam(team), nil
}

// GetAllTeams returns all teams
func (m *Manager) GetAllTeams() []Team {
	m.mu.RLock()
	defer m.mu.RUnlock()

	teams := make([]Team, 0, len(m.teams))
	for _, team := range m.teams {
		teams = append(teams, copyTeam(team))
	}
	return teams
}

// GetTeamsByPhase returns teams filtered by phase
func (m *Manager) GetTeamsByPhase(phase string) []Team {
	m.mu.RLock()
	defer m.mu.RUnlock()

	teams := make([]Team, 0)
	for _, team := range m.teams {
		if team.Phase == phase {
			teams = append(teams, copyTeam(team))
		}
	}
	return teams
}

// GetProjectStatus returns overall project status
func (m *Manager) GetProjectStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := len(m.teams)
	completed := 0
	active := 0
	notStarted := 0

	for _, team := range m.teams {
		switch team.Status {
		case TeamStatusCompleted:
			completed++
		case TeamStatusActive:
			active++
		case TeamStatusNotStarted:
			notStarted++
		}
	}

	progressPct := 0.0
	if total > 0 {
		progressPct = float64(completed) / float64(total) * 100
	}

	return map[string]interface{}{
		"project":      m.projectName,
		"total_teams":  total,
		"completed":    completed,
		"active":       active,
		"not_started":  notStarted,
		"progress_pct": progressPct,
	}
}

// QueryTeams queries teams with filters
func (m *Manager) QueryTeams(status, phase, assignee, roleName string) ([]Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]Team, 0)
	for _, team := range m.teams {
		// Check status filter
		if status != "" && string(team.Status) != status {
			continue
		}

		// Check phase filter
		if phase != "" && team.Phase != phase {
			continue
		}

		// Check assignee filter
		if assignee != "" {
			found := false
			for _, role := range team.Roles {
				if role.AssignedTo != nil && *role.AssignedTo == assignee {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check role name filter
		if roleName != "" {
			found := false
			for _, role := range team.Roles {
				if role.Name == roleName {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		results = append(results, copyTeam(team))
	}

	return results, nil
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetProjectName returns the project name
func (m *Manager) GetProjectName() string {
	return m.projectName
}
