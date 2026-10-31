-- Migration: 000001_foundation.down.sql
-- Description: Rollback foundation extensions and system metadata table

DROP TABLE IF EXISTS system_metadata;
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";

