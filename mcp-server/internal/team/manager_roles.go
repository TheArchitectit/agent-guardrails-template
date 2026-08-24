package team

import "fmt"

// AssignRole assigns a person to a role
func (m *Manager) AssignRole(teamID int, roleName, person string) error {
	if err := ValidateRoleName(roleName); err != nil {
		return err
	}
	if err := ValidatePersonName(person); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	team, exists := m.teams[teamID]
	if !exists {
		return fmt.Errorf("team %d not found", teamID)
	}

	for i := range team.Roles {
		if team.Roles[i].Name == roleName {
			previous := team.Roles[i].AssignedTo
			team.Roles[i].AssignedTo = &person
			m.teams[teamID] = team

			if err := m.save(); err != nil {
				// Rollback on error
				team.Roles[i].AssignedTo = previous
				m.teams[teamID] = team
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("role '%s' not found in team %d", roleName, teamID)
}

// UnassignRole removes assignment from a role
func (m *Manager) UnassignRole(teamID int, roleName string) error {
	if err := ValidateRoleName(roleName); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	team, exists := m.teams[teamID]
	if !exists {
		return fmt.Errorf("team %d not found", teamID)
	}

	for i := range team.Roles {
		if team.Roles[i].Name == roleName {
			if team.Roles[i].AssignedTo == nil {
				return fmt.Errorf("role '%s' in %s is already unassigned", roleName, team.Name)
			}

			previous := team.Roles[i].AssignedTo
			team.Roles[i].AssignedTo = nil
			m.teams[teamID] = team

			if err := m.save(); err != nil {
				// Rollback on error
				team.Roles[i].AssignedTo = previous
				m.teams[teamID] = team
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("role '%s' not found in team %d", roleName, teamID)
}
