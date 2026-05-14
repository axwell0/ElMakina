// Package store defines the persistence boundary for the application.
//
// Room and session storage flow through these interfaces.
// The engine and application layers never talk to a database directly —
// they call these interfaces, making the storage backend entirely swappable.
//
// Currently only one implementation exists (internal/store/memory), which stores
// everything in memory and is reset on restart. A future production deployment
// would implement these interfaces against a real database without changing any
// engine or application code.
package store

import (
	"context"

	sessionapp "github.com/axwell0/elmakina/internal/app/session"
)

// RoomStore persists room-level snapshots at lifecycle boundaries (create, join, start, end).
type RoomStore interface {
	SaveRoom(ctx context.Context, room sessionapp.LobbySnapshot) error
	LoadRoom(ctx context.Context, id string) (sessionapp.LobbySnapshot, error)
	ListRooms(ctx context.Context) ([]sessionapp.LobbySnapshot, error)
	DeleteRoom(ctx context.Context, id string) error
}

// SessionStore persists durable client session snapshots.
type SessionStore interface {
	SaveSession(ctx context.Context, roomID string, session sessionapp.ClientSessionSnapshot) error
	LoadSession(ctx context.Context, roomID string, sessionID string) (sessionapp.ClientSessionSnapshot, error)
	DeleteSession(ctx context.Context, roomID string, sessionID string) error
}

// Bundle groups stores together for convenient dependency injection.
// Passing a single Bundle into the room runtime is cleaner than three separate parameters,
// and makes it easy to swap all stores at once (e.g. in-memory for tests, DB for production).
type Bundle struct {
	Rooms    RoomStore
	Sessions SessionStore
}
