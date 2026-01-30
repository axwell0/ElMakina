package ws

import (
	"ElMakina/backend/domain/repositories"
	persistence "ElMakina/backend/infrastructure/persistence/repositories"
	"ElMakina/backend/infrastructure/persistence/transactions"
	"ElMakina/backend/server"
	"gorm.io/gorm"
)

// NewLobbyManagerV2WithDB builds a new repository-based lobby manager using the provided database.
// This uses the new repository pattern with transaction support and optimistic locking.
func NewLobbyManagerV2WithDB(minPlayers, maxPlayers int, db *gorm.DB) (*server.LobbyManagerV2, error) {
	// Create repositories
	playerRepo := persistence.NewPostgresPlayerRepository(db)
	lobbyRepo := persistence.NewPostgresLobbyRepository(db)
	unitOfWork := transactions.NewGormUnitOfWork(db)

	// Create manager
	manager := server.NewLobbyManagerV2(minPlayers, maxPlayers, playerRepo, lobbyRepo, unitOfWork)

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
