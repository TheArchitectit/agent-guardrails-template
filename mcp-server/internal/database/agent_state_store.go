package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// AgentStateStore handles agent session database operations.
type AgentStateStore struct {
	db *DB
}

// NewAgentStateStore creates a new agent state store.
func NewAgentStateStore(db *DB) *AgentStateStore {
	return &AgentStateStore{db: db}
}

// CreateSession creates a new agent session in idle state.
func (s *AgentStateStore) CreateSession(ctx context.Context, session *models.AgentSession) error {
	if session.ID == "" {
		session.ID = generateUUID()
	}
	now := time.Now().UTC()
	session.StartedAt = now
	session.LastTransitionAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_sessions (id, team_id, agent_name, current_state, previous_state, project_slug, started_at, last_transition_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, session.ID, session.TeamID, session.AgentName, string(session.CurrentState),
		string(session.PreviousState), session.ProjectSlug, session.StartedAt, session.LastTransitionAt)
	if err != nil {
		return fmt.Errorf("failed to create agent session: %w", err)
	}

	// Record the initial transition
	s.recordTransition(ctx, session.ID, "", session.CurrentState, "session created", "system")
	return nil
}

// GetSession retrieves an agent session by ID.
func (s *AgentStateStore) GetSession(ctx context.Context, sessionID string) (*models.AgentSession, error) {
	var session models.AgentSession
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, agent_name, current_state, previous_state, project_slug, started_at, last_transition_at
		FROM agent_sessions WHERE id = $1
	`, sessionID).Scan(
		&session.ID, &session.TeamID, &session.AgentName, &session.CurrentState,
		&session.PreviousState, &session.ProjectSlug, &session.StartedAt, &session.LastTransitionAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("agent session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get agent session: %w", err)
	}
	return &session, nil
}

// Transition validates and performs a state transition.
func (s *AgentStateStore) Transition(ctx context.Context, sessionID string, toState models.AgentState, reason, triggeredBy string) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if err := models.ValidateTransition(session.CurrentState, toState); err != nil {
		return err
	}

	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE agent_sessions
		SET previous_state = current_state, current_state = $2, last_transition_at = $3
		WHERE id = $1
	`, sessionID, string(toState), now)
	if err != nil {
		return fmt.Errorf("failed to transition agent session: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("agent session not found: %s", sessionID)
	}

	s.recordTransition(ctx, sessionID, session.CurrentState, toState, reason, triggeredBy)
	return nil
}

// ForceState bypasses transition validation and forces the agent to a new state.
func (s *AgentStateStore) ForceState(ctx context.Context, sessionID string, toState models.AgentState, reason string) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE agent_sessions
		SET previous_state = current_state, current_state = $2, last_transition_at = $3
		WHERE id = $1
	`, sessionID, string(toState), now)
	if err != nil {
		return fmt.Errorf("failed to force state: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("agent session not found: %s", sessionID)
	}

	s.recordTransition(ctx, sessionID, session.CurrentState, toState, reason, "admin_override")
	return nil
}

// GetTransitions returns the audit trail for a session.
func (s *AgentStateStore) GetTransitions(ctx context.Context, sessionID string) ([]*models.StateTransition, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, from_state, to_state, reason, triggered_by, created_at
		FROM agent_state_transitions
		WHERE session_id = $1
		ORDER BY created_at ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transitions: %w", err)
	}
	defer rows.Close()

	var transitions []*models.StateTransition
	for rows.Next() {
		var t models.StateTransition
		if err := rows.Scan(
			&t.ID, &t.SessionID, &t.FromState, &t.ToState,
			&t.Reason, &t.TriggeredBy, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transition: %w", err)
		}
		transitions = append(transitions, &t)
	}
	return transitions, rows.Err()
}

// ListByTeam returns all agent sessions for a team.
func (s *AgentStateStore) ListByTeam(ctx context.Context, teamID string) ([]*models.AgentSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, agent_name, current_state, previous_state, project_slug, started_at, last_transition_at
		FROM agent_sessions
		WHERE team_id = $1
		ORDER BY last_transition_at DESC
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.AgentSession
	for rows.Next() {
		var session models.AgentSession
		if err := rows.Scan(
			&session.ID, &session.TeamID, &session.AgentName, &session.CurrentState,
			&session.PreviousState, &session.ProjectSlug, &session.StartedAt, &session.LastTransitionAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan agent session: %w", err)
		}
		sessions = append(sessions, &session)
	}
	return sessions, rows.Err()
}

func (s *AgentStateStore) recordTransition(ctx context.Context, sessionID string, from, to models.AgentState, reason, triggeredBy string) {
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO agent_state_transitions (id, session_id, from_state, to_state, reason, triggered_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, generateUUID(), sessionID, string(from), string(to), reason, triggeredBy, time.Now().UTC())
}
