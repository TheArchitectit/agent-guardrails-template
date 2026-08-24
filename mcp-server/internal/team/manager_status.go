package team

import (
	"fmt"
	"time"
)

// GetTeamStatus returns the current status of a team
func (m *Manager) GetTeamStatus(teamID int) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, exists := m.teams[teamID]
	if !exists {
		return nil, fmt.Errorf("team %d not found", teamID)
	}

	assignedCount := 0
	assignedRoles := make([]map[string]string, 0)
	for _, role := range team.Roles {
		if role.AssignedTo != nil {
			assignedCount++
			assignedRoles = append(assignedRoles, map[string]string{
				"name":        role.Name,
				"assigned_to": *role.AssignedTo,
			})
		}
	}

	return map[string]any{
		"id":             team.ID,
		"name":           team.Name,
		"phase":          team.Phase,
		"description":    team.Description,
		"status":         team.Status,
		"started_at":     team.StartedAt,
		"completed_at":   team.CompletedAt,
		"assigned_count": assignedCount,
		"total_roles":    len(team.Roles),
		"assigned_roles": assignedRoles,
		"exit_criteria":  team.ExitCriteria,
	}, nil
}

// ListTeams returns all teams, optionally filtered by phase
func (m *Manager) ListTeams(phase string) ([]TeamListItem, error) {
	if phase != "" {
		if err := ValidatePhase(phase); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]TeamListItem, 0, len(m.teams))
	for _, team := range m.teams {
		if phase != "" && team.Phase != phase {
			continue
		}

		assignedCount := 0
		for _, role := range team.Roles {
			if role.AssignedTo != nil {
				assignedCount++
			}
		}

		items = append(items, TeamListItem{
			ID:            team.ID,
			Name:          team.Name,
			Phase:         team.Phase,
			Status:        string(team.Status),
			AssignedCount: assignedCount,
			TotalRoles:    len(team.Roles),
		})
	}

	return items, nil
}

// GetPhaseStatus returns status summary for a phase
func (m *Manager) GetPhaseStatus(phase string) (PhaseStatus, error) {
	if err := ValidatePhase(phase); err != nil {
		return PhaseStatus{}, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var total, completed, active int
	for _, team := range m.teams {
		if team.Phase != phase {
			continue
		}
		total++
		switch team.Status {
		case TeamStatusCompleted:
			completed++
		case TeamStatusActive:
			active++
		}
	}

	progressPct := 0.0
	if total > 0 {
		progressPct = float64(completed) / float64(total) * 100
	}

	return PhaseStatus{
		Phase:       phase,
		TotalTeams:  total,
		Completed:   completed,
		Active:      active,
		NotStarted:  total - completed - active,
		ProgressPct: progressPct,
	}, nil
}

// GetAllPhaseStatuses returns status for all phases
func (m *Manager) GetAllPhaseStatuses() ([]PhaseStatus, error) {
	phases := make(map[string]bool)
	for _, team := range m.teams {
		phases[team.Phase] = true
	}

	results := make([]PhaseStatus, 0, len(phases))
	for phase := range phases {
		status, err := m.GetPhaseStatus(phase)
		if err != nil {
			return nil, err
		}
		results = append(results, status)
	}

	return results, nil
}

// GetTeamAssignments returns all assignments for a team
func (m *Manager) GetTeamAssignments(teamID int) ([]TeamAssignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, exists := m.teams[teamID]
	if !exists {
		return nil, fmt.Errorf("team %d not found", teamID)
	}

	assignments := make([]TeamAssignment, 0)
	var assignedAt string
	if team.StartedAt != nil {
		assignedAt = team.StartedAt.Format(time.RFC3339)
	}

	for _, role := range team.Roles {
		if role.AssignedTo != nil {
			assignments = append(assignments, TeamAssignment{
				TeamID:     teamID,
				RoleName:   role.Name,
				Person:     *role.AssignedTo,
				AssignedAt: assignedAt,
			})
		}
	}

	return assignments, nil
}

// GetPersonAssignments returns all assignments for a person
func (m *Manager) GetPersonAssignments(person string) ([]TeamAssignment, error) {
	if err := ValidatePersonName(person); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	assignments := make([]TeamAssignment, 0)
	for _, team := range m.teams {
		for _, role := range team.Roles {
			if role.AssignedTo != nil && *role.AssignedTo == person {
				var assignedAt string
				if team.StartedAt != nil {
					assignedAt = team.StartedAt.Format(time.RFC3339)
				}
				assignments = append(assignments, TeamAssignment{
					TeamID:     team.ID,
					RoleName:   role.Name,
					Person:     person,
					AssignedAt: assignedAt,
				})
			}
		}
	}

	return assignments, nil
}
