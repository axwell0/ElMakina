-- Migration: 001_create_normalized_lobby_schema.sql
-- Created: 2026-01-30
-- Description: Normalize lobby data from JSON blob to relational schema

-- Step 1: Create new normalized tables
CREATE TABLE IF NOT EXISTS players (
    id VARCHAR(64) PRIMARY KEY,
    nick VARCHAR(32) NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    avatar TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_players_token ON players(token);

CREATE TABLE IF NOT EXISTS lobbies (
    id VARCHAR(64) PRIMARY KEY,
    leader_id VARCHAR(64) NOT NULL REFERENCES players(id),
    status VARCHAR(16) NOT NULL CHECK (status IN ('open', 'in_game', 'closed')),
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lobbies_status ON lobbies(status);
CREATE INDEX IF NOT EXISTS idx_lobbies_leader ON lobbies(leader_id);

CREATE TABLE IF NOT EXISTS lobby_players (
    lobby_id VARCHAR(64) NOT NULL REFERENCES lobbies(id) ON DELETE CASCADE,
    player_id VARCHAR(64) NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (lobby_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_lobby_players_player ON lobby_players(player_id);

-- Step 2: Migration tracking table
CREATE TABLE IF NOT EXISTS _migration_status (
    key VARCHAR(64) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO _migration_status (key, value) 
VALUES ('lobby_migration_phase', 'schema_created')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at;
