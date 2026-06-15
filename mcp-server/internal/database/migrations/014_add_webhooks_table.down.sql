-- Migration: Remove webhook notification tables
-- Version: 014

DROP TABLE IF EXISTS webhook_deliveries CASCADE;
DROP TABLE IF EXISTS webhook_configs CASCADE;
