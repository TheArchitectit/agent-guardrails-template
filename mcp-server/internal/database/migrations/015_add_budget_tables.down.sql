-- Migration: Remove budget tracking tables
-- Version: 015

DROP TABLE IF EXISTS budget_entries CASCADE;
DROP TABLE IF EXISTS budget_configs CASCADE;
