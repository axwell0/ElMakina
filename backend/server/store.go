package server

import "context"

// LobbySnapshot captures lobby state for persistence.
// It is a full snapshot; callers should treat it as immutable once saved.
type LobbySnapshot struct {
	Lobbies     map[string]Lobby
	Players     map[string]Player
	ByToken     map[string]string
	NextLobbyID uint64
}

// LobbyStore persists and restores lobby state.
//
// Implementations may be local (file) or remote (Redis). The interface accepts
// context for cancellation and future remote calls.
type LobbyStore interface {
	Load(ctx context.Context) (*LobbySnapshot, error)
	Save(ctx context.Context, snapshot LobbySnapshot) error
}
