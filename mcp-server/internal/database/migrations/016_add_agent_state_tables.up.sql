-- Migration: Add agent lifecycle state machine tables
-- Version: 016

CREATE TABLE IF NOT EXISTS agent_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id VARCHAR(100) NOT NULL,
    agent_name VARCHAR(200) NOT NULL,
    current_state VARCHAR(50) NOT NULL DEFAULT 'idle',
    previous_state VARCHAR(50),
    project_slug VARCHAR(100),
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_transition_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_team_id ON agent_sessions(team_id);
CREATE INDEX IF NOT EXISTS idx_agent_sessions_state ON agent_sessions(current_state);

CREATE TABLE IF NOT EXISTS agent_state_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
    from_state VARCHAR(50),
    to_state VARCHAR(50) NOT NULL,
    reason TEXT,
    triggered_by VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_transitions_session_id ON agent_state_transitions(session_id);
