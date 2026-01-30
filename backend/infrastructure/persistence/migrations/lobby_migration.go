package migrations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ElMakina/backend/infrastructure/persistence/models"
	"ElMakina/backend/server"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LobbyMigration handles migrating JSON blob data to normalized schema.
type LobbyMigration struct {
	db *gorm.DB
}

// NewLobbyMigration creates a new migration handler.
func NewLobbyMigration(db *gorm.DB) *LobbyMigration {
	return &LobbyMigration{db: db}
}

// Phase represents the current migration state.
type Phase string

const (
	PhaseNotStarted    Phase = "not_started"
	PhaseSchemaCreated Phase = "schema_created"
	PhaseDataMigrated  Phase = "data_migrated"
	PhaseComplete      Phase = "complete"
)

// GetPhase returns the current migration phase.
func (m *LobbyMigration) GetPhase(ctx context.Context) (Phase, error) {
	var status models.MigrationStatus
	result := m.db.WithContext(ctx).First(&status, "key = ?", "lobby_migration_phase")
	if result.Error == gorm.ErrRecordNotFound {
		return PhaseNotStarted, nil
	}
	if result.Error != nil {
		return "", result.Error
	}
	return Phase(status.Value), nil
}

// setPhase updates the migration phase.
func (m *LobbyMigration) setPhase(ctx context.Context, phase Phase) error {
	return m.db.WithContext(ctx).Exec(`
		INSERT INTO _migration_status (key, value, updated_at)
		VALUES ('lobby_migration_phase', ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`, string(phase), time.Now()).Error
}

// MigrateData migrates existing JSON blob data to normalized tables.
func (m *LobbyMigration) MigrateData(ctx context.Context) error {
	phase, err := m.GetPhase(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration phase: %w", err)
	}
	if phase != PhaseSchemaCreated {
		return fmt.Errorf("migration not in correct phase: %s", phase)
	}

	// Load latest snapshot from legacy table
	var record struct {
		Data datatypes.JSON
	}
	result := m.db.WithContext(ctx).Table("lobby_snapshots").
		Order("updated_at desc").
		First(&record)

	if result.Error == gorm.ErrRecordNotFound {
		// No data to migrate
		return m.setPhase(ctx, PhaseDataMigrated)
	}
	if result.Error != nil {
		return fmt.Errorf("failed to load legacy snapshot: %w", result.Error)
	}

	var snapshot server.LobbySnapshot
	if err := json.Unmarshal(record.Data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	// Begin transaction for data migration
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Migrate players
		for id, player := range snapshot.Players {
			playerModel := models.PlayerModel{
				ID:        id,
				Nick:      player.Nick,
				Token:     player.Token,
				Avatar:    player.Avatar,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := tx.Create(&playerModel).Error; err != nil {
				return fmt.Errorf("failed to migrate player %s: %w", id, err)
			}
		}

		// Migrate lobbies
		for id, lobby := range snapshot.Lobbies {
			lobbyModel := models.LobbyModel{
				ID:        id,
				LeaderID:  lobby.LeaderID,
				Status:    string(lobby.Status),
				Version:   1,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := tx.Create(&lobbyModel).Error; err != nil {
				return fmt.Errorf("failed to migrate lobby %s: %w", id, err)
			}

			// Migrate lobby player associations
			for _, playerID := range lobby.PlayerIDs {
				lobbyPlayer := models.LobbyPlayerModel{
					LobbyID:  id,
					PlayerID: playerID,
					JoinedAt: time.Now(),
				}
				if err := tx.Create(&lobbyPlayer).Error; err != nil {
					return fmt.Errorf("failed to migrate lobby_player %s/%s: %w", id, playerID, err)
				}
			}
		}

		return nil
	})
}

// EnableDualWrite enables writing to both old and new schemas.
func (m *LobbyMigration) EnableDualWrite(ctx context.Context) error {
	// Set feature flag in migration status
	return m.db.WithContext(ctx).Exec(`
		INSERT INTO _migration_status (key, value, updated_at)
		VALUES ('lobby_dual_write', 'true', ?)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`, time.Now()).Error
}

// IsDualWriteEnabled checks if dual write mode is active.
func (m *LobbyMigration) IsDualWriteEnabled(ctx context.Context) (bool, error) {
	var status models.MigrationStatus
	result := m.db.WithContext(ctx).First(&status, "key = ?", "lobby_dual_write")
	if result.Error == gorm.ErrRecordNotFound {
		return false, nil
	}
	if result.Error != nil {
		return false, result.Error
	}
	return status.Value == "true", nil
}

// CompleteMigration finalizes the migration and disables legacy writes.
func (m *LobbyMigration) CompleteMigration(ctx context.Context) error {
	// Verify all data is migrated
	var legacyCount, newCount int64
	m.db.WithContext(ctx).Table("lobby_snapshots").Count(&legacyCount)
	m.db.WithContext(ctx).Model(&models.LobbyModel{}).Count(&newCount)

	if newCount == 0 && legacyCount > 0 {
		return fmt.Errorf("migration incomplete: no lobbies in new schema")
	}

	// Disable dual write
	if err := m.db.WithContext(ctx).Exec(`
		UPDATE _migration_status SET value = 'false', updated_at = ? WHERE key = 'lobby_dual_write'
	`, time.Now()).Error; err != nil {
		return err
	}

	// Mark complete
	return m.setPhase(ctx, PhaseComplete)
}

// Rollback performs a rollback of the migration.
// WARNING: This will lose any data created after migration.
func (m *LobbyMigration) Rollback(ctx context.Context) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Drop new tables
		if err := tx.Migrator().DropTable("lobby_players", "lobbies", "players"); err != nil {
			return fmt.Errorf("failed to drop tables: %w", err)
		}

		// Reset migration status
		if err := tx.Delete(&models.MigrationStatus{}, "key LIKE 'lobby_%'").Error; err != nil {
			return fmt.Errorf("failed to reset migration status: %w", err)
		}

		return nil
	})
}
