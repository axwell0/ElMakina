-- Migration: 001_create_normalized_lobby_schema_rollback.sql
-- Rollback: Drop normalized tables

-- Note: This will lose all data in the normalized schema
-- Only run this if you want to completely reset

DROP TABLE IF EXISTS lobby_players;
DROP TABLE IF EXISTS lobbies;
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS _migration_status;
