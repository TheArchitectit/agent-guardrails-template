-- Migration: Add budget tracking tables
-- Version: 015

CREATE TABLE IF NOT EXISTS budget_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id VARCHAR(100) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    max_tokens BIGINT NOT NULL DEFAULT 0,
    max_cost_cents BIGINT NOT NULL DEFAULT 0,
    period VARCHAR(20) NOT NULL DEFAULT 'daily',
    alert_threshold DOUBLE PRECISION NOT NULL DEFAULT 0.8,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, model_name)
);

CREATE INDEX IF NOT EXISTS idx_budget_configs_team_id ON budget_configs(team_id);
CREATE INDEX IF NOT EXISTS idx_budget_configs_enabled ON budget_configs(enabled) WHERE enabled = true;

CREATE TABLE IF NOT EXISTS budget_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id VARCHAR(100) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    cost_cents BIGINT NOT NULL DEFAULT 0,
    endpoint VARCHAR(200),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_budget_entries_team_model ON budget_entries(team_id, model_name);
CREATE INDEX IF NOT EXISTS idx_budget_entries_created_at ON budget_entries(created_at);
