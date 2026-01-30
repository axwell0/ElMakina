package ws

import (
	"context"
	"fmt"
	"os"

	"ElMakina/backend/domain/repositories"
	"ElMakina/backend/domain/services"
	persistence "ElMakina/backend/infrastructure/persistence/repositories"
	"ElMakina/backend/infrastructure/persistence/transactions"
	"ElMakina/backend/server"
	"ElMakina/backend/server/replay"
	"gorm.io/gorm"
)

// NewLobbyManagerFromEnv builds a lobby manager using PostgreSQL persistence.
// This function opens a PostgreSQL connection using the DSN from ELMAKINA_POSTGRES_DSN.
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
	// Create repositories
	playerRepo := persistence.NewPostgresPlayerRepository(db)
	lobbyRepo := persistence.NewPostgresLobbyRepository(db)
	unitOfWork := transactions.NewGormUnitOfWork(db)

	// Create player service
	playerService := services.NewPlayerService(playerRepo, lobbyRepo, unitOfWork, nil, nil)

	// Create lobby manager
	manager := server.NewLobbyManager(minPlayers, maxPlayers, playerService, lobbyRepo, unitOfWork)

	return manager, nil
}

// RepositoryProvider provides access to repository instances.
// This is useful for dependency injection and testing.
type RepositoryProvider struct {
	PlayerRepo repositories.PlayerRepository
	LobbyRepo  repositories.LobbyRepository
	UnitOfWork repositories.UnitOfWork
}

// NewRepositoryProvider creates a new repository provider from a database connection.
func NewRepositoryProvider(db *gorm.DB) *RepositoryProvider {
	return &RepositoryProvider{
		PlayerRepo: persistence.NewPostgresPlayerRepository(db),
		LobbyRepo:  persistence.NewPostgresLobbyRepository(db),
		UnitOfWork: transactions.NewGormUnitOfWork(db),
	}
}
