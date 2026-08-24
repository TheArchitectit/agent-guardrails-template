package team

import (
	"fmt"
	"os"
)

// copyTeam creates a deep copy of a Team
func copyTeam(team Team) Team {
	roles := make([]Role, len(team.Roles))
	for i, r := range team.Roles {
		deliverables := make([]string, len(r.Deliverables))
		copy(deliverables, r.Deliverables)
		roles[i] = Role{
			Name:           r.Name,
			Responsibility: r.Responsibility,
			Deliverables:   deliverables,
			AssignedTo:     r.AssignedTo,
		}
	}

	exitCriteria := make([]string, len(team.ExitCriteria))
	copy(exitCriteria, team.ExitCriteria)

	return Team{
		ID:           team.ID,
		Name:         team.Name,
		Phase:        team.Phase,
		Description:  team.Description,
		Roles:        roles,
		ExitCriteria: exitCriteria,
		Status:       team.Status,
		StartedAt:    team.StartedAt,
		CompletedAt:  team.CompletedAt,
	}
}

// DeleteTeam removes a team from the project (marks as deleted)
func (m *Manager) DeleteTeam(teamID int, confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("deletion requires confirmation")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.teams[teamID]; !exists {
		return fmt.Errorf("team %d not found", teamID)
	}

	// Remove the team from the map
	delete(m.teams, teamID)

	if err := m.save(); err != nil {
		return fmt.Errorf("failed to save after delete: %w", err)
	}

	return nil
}

// DeleteProject removes the entire project configuration file
func (m *Manager) DeleteProject(confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("project deletion requires confirmation")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove the config file
	if err := os.Remove(m.configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove project file: %w", err)
	}

	// Clear the teams map
	m.teams = make(map[int]Team)

	return nil
}

// Health returns the health status of the team manager
func (m *Manager) Health() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalTeams := len(m.teams)
	active := 0
	completed := 0
	notStarted := 0
	assignedRoles := 0

	for _, team := range m.teams {
		switch team.Status {
		case TeamStatusActive:
			active++
		case TeamStatusCompleted:
			completed++
		case TeamStatusNotStarted:
			notStarted++
		}
		for _, role := range team.Roles {
			if role.AssignedTo != nil && *role.AssignedTo != "" {
				assignedRoles++
			}
		}
	}

	return map[string]interface{}{
		"status":         "healthy",
		"project":        m.projectName,
		"total_teams":    totalTeams,
		"active":         active,
		"completed":      completed,
		"not_started":    notStarted,
		"assigned_roles": assignedRoles,
		"config_path":    m.configPath,
	}
}
