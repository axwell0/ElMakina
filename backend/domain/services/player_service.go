package services

import (
	"context"
	"time"

	"ElMakina/backend/domain/entities"
)

// PlayerService defines the contract for player lifecycle management.
// It handles registration, authentication, and cleanup of players.
type PlayerService interface {
	// RegisterPlayer creates a new player with the given nickname.
	// Generates a unique ID and secure reconnection token.
	// Returns ErrInvalidNickname if the nick is empty or too long.
	RegisterPlayer(ctx context.Context, nick string) (*entities.Player, error)

	// SetPlayerAvatar updates a player's avatar.
	// Returns ErrPlayerNotFound if the player doesn't exist.
	SetPlayerAvatar(ctx context.Context, playerID entities.PlayerID, avatar string) error

	// ReconnectPlayer authenticates a player using their reconnection token.
	// Updates the activity timestamp on successful reconnection.
	// Returns ErrInvalidToken if the token is invalid or expired.
	ReconnectPlayer(ctx context.Context, token string) (*entities.Player, error)

	// UnregisterPlayer removes a player from the system.
	// Only succeeds if the player is not currently in any lobby.
	// Returns ErrPlayerNotFound if the player doesn't exist.
	// Returns ErrPlayerInLobby if the player is in an active lobby.
	UnregisterPlayer(ctx context.Context, playerID entities.PlayerID) error

	// GetPlayerByID retrieves a player by their unique ID.
	// Returns ErrPlayerNotFound if the player doesn't exist.
	GetPlayerByID(ctx context.Context, playerID entities.PlayerID) (*entities.Player, error)

	// PruneInactivePlayers removes players who have been inactive longer than the given duration.
	// Only removes players who are not in any lobby.
	// Returns the list of removed player IDs.
	PruneInactivePlayers(ctx context.Context, olderThan time.Duration) ([]entities.PlayerID, error)

	// TouchPlayerActivity updates the last active timestamp for a player.
	// Used by other services to maintain activity tracking.
	// Returns ErrPlayerNotFound if the player doesn't exist.
	TouchPlayerActivity(ctx context.Context, playerID entities.PlayerID) error
}

// TokenGenerator defines the contract for generating secure tokens.
// This abstraction allows for testing and different token strategies.
type TokenGenerator interface {
	// GeneratePlayerToken creates a new secure random token for player authentication.
	GeneratePlayerToken() (string, error)
}

// IDGenerator defines the contract for generating unique identifiers.
type IDGenerator interface {
	// GeneratePlayerID creates a new unique player identifier.
	GeneratePlayerID() entities.PlayerID
}
