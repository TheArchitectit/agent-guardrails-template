package team

import (
	"fmt"
	"strings"
)

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

// GetTeamsByPhase returns teams filtered by phase.
//
// Accepts either the full phase label ("Phase 1: Strategy, Governance &
// Planning") or the short form ("Phase 1"). Callers validate and pass the
// short form, while Team.Phase always holds the full label, so an exact
// comparison alone returned nothing for every valid request.
func (m *Manager) GetTeamsByPhase(phase string) []Team {
	m.mu.RLock()
	defer m.mu.RUnlock()

	teams := make([]Team, 0)
	for _, team := range m.teams {
		if phaseMatches(team.Phase, phase) {
			teams = append(teams, copyTeam(team))
		}
	}
	return teams
}

// phaseMatches reports whether a team's full phase label satisfies the
// requested phase, which may be either the full label or the short form.
func phaseMatches(teamPhase, requested string) bool {
	if teamPhase == requested {
		return true
	}
	// Short form: "Phase 1" matches "Phase 1: Strategy, ...".
	return strings.HasPrefix(teamPhase, requested+":")
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
