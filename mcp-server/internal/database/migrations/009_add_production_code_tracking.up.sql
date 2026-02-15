-- Migration to create production code tracking table
-- This table tracks code existence for guardrail validation

DO $$ BEGIN
    CREATE TYPE code_type_enum AS ENUM ('production', 'test', 'infrastructure');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS production_code_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    code_type code_type_enum NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP WITH TIME ZONE NULL,

    CONSTRAINT unique_session_file UNIQUE (session_id, file_path)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_production_code_tracking_session_id ON production_code_tracking(session_id);
CREATE INDEX IF NOT EXISTS idx_production_code_tracking_session_id_type ON production_code_tracking(session_id, code_type);
CREATE INDEX IF NOT EXISTS idx_production_code_tracking_created_at ON production_code_tracking(created_at);
COMMENT ON TABLE production_code_tracking IS 'Tracks production code existence for guardrail validation';
