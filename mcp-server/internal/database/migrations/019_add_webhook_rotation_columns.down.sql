-- Migration: Remove webhook signing-key rotation columns
-- Version: 019

ALTER TABLE webhook_configs
    DROP COLUMN IF EXISTS secret_hmac_previous,
    DROP COLUMN IF EXISTS secret_hmac_rotated_at;
