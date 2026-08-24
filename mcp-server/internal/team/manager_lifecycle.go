package team

import (
	"fmt"
	"time"
)

// StartTeam marks a team as active
func (m *Manager) StartTeam(teamID int, override bool, reason string) error {
	_ = reason // Reserved for future audit logging
	_ = override

	m.mu.Lock()
	defer m.mu.Unlock()

	team, exists := m.teams[teamID]
	if !exists {
		return fmt.Errorf("team %d not found", teamID)
	}

	if team.Status == TeamStatusActive {
		return fmt.Errorf("team %d is already active", teamID)
	}

	if team.Status == TeamStatusCompleted {
		return fmt.Errorf("team %d is already completed", teamID)
	}

	now := time.Now()
	previousStatus := team.Status
	team.Status = TeamStatusActive
	team.StartedAt = &now
	m.teams[teamID] = team

	if err := m.save(); err != nil {
		// Rollback
		team.Status = previousStatus
		team.StartedAt = nil
		m.teams[teamID] = team
		return err
	}

	return nil
}

// CompleteTeam marks a team as completed
func (m *Manager) CompleteTeam(teamID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	team, exists := m.teams[teamID]
	if !exists {
		return fmt.Errorf("team %d not found", teamID)
	}

	if team.Status == TeamStatusCompleted {
		return fmt.Errorf("team %d is already completed", teamID)
	}

	if team.Status != TeamStatusActive {
		return fmt.Errorf("team %d must be active before completing", teamID)
	}

	now := time.Now()
	previousStatus := team.Status
	team.Status = TeamStatusCompleted
	team.CompletedAt = &now
	m.teams[teamID] = team

	if err := m.save(); err != nil {
		// Rollback
		team.Status = previousStatus
		team.CompletedAt = nil
		m.teams[teamID] = team
		return err
	}

	return nil
}
