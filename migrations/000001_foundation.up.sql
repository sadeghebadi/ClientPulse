-- Migration: 000001_foundation.up.sql
-- Description: Enable PostgreSQL cryptographic extensions and schema metadata

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- System metadata table to verify database foundation readiness
CREATE TABLE IF NOT EXISTS system_metadata (
    key VARCHAR(64) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO system_metadata (key, value)
VALUES ('schema_version', '1.0.0'), ('initial_setup', 'Phase 0 Complete')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

