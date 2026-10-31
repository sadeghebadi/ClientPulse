-- Migration: 000002_worker_heartbeats.up.sql
-- Description: Worker heartbeat registry for liveness verification

CREATE TABLE IF NOT EXISTS worker_heartbeats (
    worker_id VARCHAR(64) PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    pid INT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    last_heartbeat_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_worker_heartbeats_last_heartbeat 
    ON worker_heartbeats (last_heartbeat_at DESC);

