package ws

import (
	"context"
	"fmt"
	"os"

	"ElMakina/backend/server"
	"ElMakina/backend/server/replay"
	"ElMakina/backend/server/store"
	"gorm.io/gorm"
)

const (
	// EnvLobbyStorePath is deprecated. PostgreSQL is now used exclusively.
	EnvLobbyStorePath = "ELMAKINA_LOBBY_STORE_PATH"
)

// NewLobbyManagerFromEnv builds a lobby manager using PostgreSQL persistence.
// This function opens a PostgreSQL connection using the DSN from ELMAKINA_POSTGRES_DSN.
// Deprecated: Use NewLobbyManagerWithDB instead for better control over the database connection.
func NewLobbyManagerFromEnv(minPlayers, maxPlayers int) (*server.LobbyManager, *gorm.DB, error) {
	dsn := os.Getenv("ELMAKINA_POSTGRES_DSN")
	if dsn == "" {
		return nil, nil, fmt.Errorf("ELMAKINA_POSTGRES_DSN environment variable is required")
	}

	ctx := context.Background()
	db, err := replay.OpenPostgres(ctx, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	manager, err := NewLobbyManagerWithDB(minPlayers, maxPlayers, db)
	if err != nil {
		return nil, nil, err
	}
	return manager, db, nil
}

// NewLobbyManagerWithDB builds a lobby manager using the provided PostgreSQL database.
// This is the preferred method for production use.
func NewLobbyManagerWithDB(minPlayers, maxPlayers int, db *gorm.DB) (*server.LobbyManager, error) {
	lobbyStore := store.NewPostgresLobbyStore(db)

	ctx := context.Background()
	if err := lobbyStore.AutoMigrate(ctx); err != nil {
		return nil, fmt.Errorf("failed to migrate lobby tables: %w", err)
	}

	return server.NewLobbyManagerWithStore(minPlayers, maxPlayers, lobbyStore)
}
