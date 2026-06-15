package models

import (
	"fmt"
	"time"
)

// AgentState represents the current lifecycle phase of an agent session.
type AgentState string

const (
	AgentStateIdle     AgentState = "idle"
	AgentStatePlanning AgentState = "planning"
	AgentStateActive   AgentState = "active"
	AgentStateReview   AgentState = "review"
	AgentStateRelease  AgentState = "release"
	AgentStateHalted   AgentState = "halted"
)

// ValidTransitions defines which state transitions are allowed.
var ValidTransitions = map[AgentState][]AgentState{
	AgentStateIdle:     {AgentStatePlanning},
	AgentStatePlanning: {AgentStateActive, AgentStateIdle},
	AgentStateActive:   {AgentStateReview, AgentStateHalted, AgentStateIdle},
	AgentStateReview:   {AgentStateRelease, AgentStateActive, AgentStateHalted},
	AgentStateRelease:  {AgentStateIdle},
	AgentStateHalted:   {AgentStateIdle},
}

// ValidateTransition checks if a state transition is valid.
func ValidateTransition(from, to AgentState) error {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return fmt.Errorf("unknown source state: %s", from)
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("invalid transition from %s to %s (allowed: %v)", from, to, allowed)
}

// AgentSession tracks an agent's lifecycle through its phases.
type AgentSession struct {
	ID               string     `json:"id"`
	TeamID           string     `json:"team_id"`
	AgentName        string     `json:"agent_name"`
	CurrentState     AgentState `json:"current_state"`
	PreviousState    AgentState `json:"previous_state,omitempty"`
	ProjectSlug      string     `json:"project_slug,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	LastTransitionAt time.Time  `json:"last_transition_at"`
}

// StateTransition records a single state change event for audit trails.
type StateTransition struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	FromState   AgentState `json:"from_state"`
	ToState     AgentState `json:"to_state"`
	Reason      string     `json:"reason,omitempty"`
	TriggeredBy string     `json:"triggered_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
