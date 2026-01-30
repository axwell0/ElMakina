package entities

import "errors"

// Domain errors for entity validation and business rules.
var (
	// Entity validation errors
	ErrInvalidEntity = errors.New("invalid entity")

	// Lobby operation errors
	ErrLobbyNotFound           = errors.New("lobby not found")
	ErrLobbyNotOpen            = errors.New("lobby is not open")
	ErrLobbyFull               = errors.New("lobby is full")
	ErrLobbyClosed             = errors.New("lobby is closed")
	ErrPlayerAlreadyInLobby    = errors.New("player already in lobby")
	ErrPlayerNotInLobby        = errors.New("player not in lobby")
	ErrNotEnoughPlayers        = errors.New("not enough players")
	ErrNotLeader               = errors.New("only leader can perform this action")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrConcurrentModification  = errors.New("concurrent modification detected")

	// Player operation errors
	ErrPlayerNotFound = errors.New("player not found")
	ErrInvalidToken   = errors.New("invalid token")
	ErrPlayerInLobby  = errors.New("player is in a lobby")
)

// IsNotFound returns true if the error is a "not found" error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrLobbyNotFound) || errors.Is(err, ErrPlayerNotFound)
}

// IsConflict returns true if the error is a conflict error.
func IsConflict(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrConcurrentModification) ||
		errors.Is(err, ErrPlayerAlreadyInLobby) ||
		errors.Is(err, ErrLobbyFull)
}
