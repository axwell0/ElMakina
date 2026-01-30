package testutil

import (
	"testing"

	"ElMakina/backend/infrastructure/persistence/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestDB provides a PostgreSQL database for testing.
// In production, this would use testcontainers. For now, we use a simple approach.
type TestDB struct {
	DB *gorm.DB
}

// NewTestDB creates a new test database instance.
// Note: This expects an existing database connection to be provided.
func NewTestDB(t *testing.T, db *gorm.DB) *TestDB {
	return &TestDB{DB: db}
}

// RunMigrations runs the GORM auto-migrations for test models.
func (td *TestDB) RunMigrations(t *testing.T) {
	// Auto-migrate GORM models
	err := td.DB.AutoMigrate(
		&models.PlayerModel{},
		&models.LobbyModel{},
		&models.LobbyPlayerModel{},
		&models.MigrationStatus{},
	)
	require.NoError(t, err)
}

// Cleanup cleans up test data.
func (td *TestDB) Cleanup(t *testing.T) {
	// Delete all data from tables
	td.DB.Exec("DELETE FROM lobby_players")
	td.DB.Exec("DELETE FROM lobbies")
	td.DB.Exec("DELETE FROM players")
	td.DB.Exec("DELETE FROM _migration_status")
}
