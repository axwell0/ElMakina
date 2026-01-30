package repositories

import (
	"context"

	"ElMakina/backend/domain/entities"
)

// UnitOfWork provides atomic operations across multiple repositories.
// It ensures that complex operations (like join lobby + update player)
// are persisted atomically.
type UnitOfWork interface {
	// Execute runs the given function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	// If the function returns nil, the transaction is committed.
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}

// AtomicLobbyOperations defines atomic operations that span multiple entities.
// These are implemented using UnitOfWork to ensure consistency.
type AtomicLobbyOperations interface {
	// JoinLobbyAtomically adds a player to a lobby in a single transaction.
	// Validates lobby is open, not full, and player is not already present.
	JoinLobbyAtomically(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID, maxPlayers int) error

	// LeaveLobbyAtomically removes a player from a lobby.
	// Handles leader transfer and lobby closure if empty.
	LeaveLobbyAtomically(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) (lobbyClosed bool, err error)

	// StartLobbyAtomically transitions lobby to in-game and creates session metadata.
	// Returns session data needed to initialize the game.
	StartLobbyAtomically(ctx context.Context, lobbyID entities.LobbyID, leaderID entities.PlayerID, minPlayers int) (*SessionInitData, error)

	// RegisterPlayerAtomically creates a player and optionally adds them to a lobby.
	RegisterPlayerAtomically(ctx context.Context, nick, token string, lobbyID *entities.LobbyID) (*entities.Player, error)
}

// SessionInitData contains all information needed to start a game session.
type SessionInitData struct {
	MatchID       string
	LobbyID       entities.LobbyID
	PlayerIDs     []entities.PlayerID
	PlayerNicks   []string
	PlayerAvatars []string
	RNGSeed       string
}

// UnitOfWorkProvider creates UnitOfWork instances bound to a persistence backend.
type UnitOfWorkProvider interface {
	// NewUnitOfWork creates a new UnitOfWork for atomic operations.
	NewUnitOfWork() UnitOfWork

	// AtomicLobby returns the atomic operations interface.
	AtomicLobby() AtomicLobbyOperations
}
