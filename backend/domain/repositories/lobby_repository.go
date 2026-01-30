package repositories

import (
	"context"

	"ElMakina/backend/domain/entities"
)

// LobbyListFilter provides filtering options for lobby queries.
type LobbyListFilter struct {
	Status     *entities.LobbyStatus // Filter by status (nil = all)
	MinPlayers int                   // Minimum player count
	MaxPlayers int                   // Maximum player count
	Limit      int                   // Max results (0 = no limit)
	Offset     int                   // Pagination offset
}

// LobbyWithPlayers bundles a lobby with its full player details.
type LobbyWithPlayers struct {
	Lobby   *entities.Lobby
	Players []*entities.Player
}

// LobbyRepository defines the contract for lobby persistence.
type LobbyRepository interface {
	// GetByID retrieves a lobby by ID without player details.
	// Returns ErrLobbyNotFound if the lobby doesn't exist.
	GetByID(ctx context.Context, id entities.LobbyID) (*entities.Lobby, error)

	// GetByIDWithPlayers retrieves a lobby with all player details populated.
	// Returns ErrLobbyNotFound if the lobby doesn't exist.
	GetByIDWithPlayers(ctx context.Context, id entities.LobbyID) (*LobbyWithPlayers, error)

	// GetByPlayerID finds the lobby containing the given player.
	// Returns ErrLobbyNotFound if the player is not in any lobby.
	GetByPlayerID(ctx context.Context, playerID entities.PlayerID) (*entities.Lobby, error)

	// List retrieves lobbies matching the filter criteria.
	// Returns empty slice (not error) if no lobbies match.
	List(ctx context.Context, filter LobbyListFilter) ([]*entities.Lobby, error)

	// ListWithPlayerCounts retrieves lobbies with player counts for listing.
	// More efficient than full player details for lobby browser.
	ListWithPlayerCounts(ctx context.Context, filter LobbyListFilter) ([]LobbySummary, error)

	// Save persists a lobby (insert or update).
	// Uses optimistic locking: increments version, checks for conflicts.
	Save(ctx context.Context, lobby *entities.Lobby) error

	// Delete removes a lobby and all its player associations.
	// Returns ErrLobbyNotFound if the lobby doesn't exist.
	Delete(ctx context.Context, id entities.LobbyID) error

	// Exists checks if a lobby with the given ID exists.
	Exists(ctx context.Context, id entities.LobbyID) (bool, error)
}

// LobbySummary is a lightweight view for lobby listing.
type LobbySummary struct {
	ID          entities.LobbyID
	LeaderID    entities.PlayerID
	LeaderNick  string
	PlayerCount int
	PlayerNicks []string
	Status      entities.LobbyStatus
	CreatedAt   int64 // Unix timestamp for JSON compatibility
}
