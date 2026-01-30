package repositories

import (
	"context"
	"time"

	"ElMakina/backend/domain/entities"
)

// PlayerRepository defines the contract for player persistence.
// Implementations must handle context cancellation and transaction boundaries.
type PlayerRepository interface {
	// GetByID retrieves a player by their unique ID.
	// Returns ErrPlayerNotFound if the player doesn't exist.
	GetByID(ctx context.Context, id entities.PlayerID) (*entities.Player, error)

	// GetByToken retrieves a player by their secret reconnection token.
	// Returns ErrPlayerNotFound if the token is invalid.
	GetByToken(ctx context.Context, token string) (*entities.Player, error)

	// GetByIDs retrieves multiple players by their IDs.
	// Returns partial results; missing players are silently skipped.
	GetByIDs(ctx context.Context, ids []entities.PlayerID) ([]*entities.Player, error)

	// Save persists a player (insert or update).
	// For updates, the version field is used for optimistic locking.
	Save(ctx context.Context, player *entities.Player) error

	// Delete removes a player permanently.
	// Returns ErrPlayerNotFound if the player doesn't exist.
	Delete(ctx context.Context, id entities.PlayerID) error

	// Exists checks if a player with the given ID exists.
	Exists(ctx context.Context, id entities.PlayerID) (bool, error)

	// ListInactive retrieves players who have been inactive since the given time.
	// Returns empty slice (not error) if no inactive players found.
	ListInactive(ctx context.Context, since time.Time) ([]*entities.Player, error)

	// DeleteBatch removes multiple players in a single operation.
	// Returns the number of players actually deleted.
	// Does not return error for non-existent players (idempotent).
	DeleteBatch(ctx context.Context, ids []entities.PlayerID) (int, error)
}
