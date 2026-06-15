-- Migration: Remove agent lifecycle state machine tables
-- Version: 016

DROP TABLE IF EXISTS agent_state_transitions CASCADE;
DROP TABLE IF EXISTS agent_sessions CASCADE;
