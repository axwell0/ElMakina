// Package memory provides an in-memory implementation of the store.Bundle interfaces.
//
// This is intended for local development and testing only — all data is lost on restart.
// A production deployment would replace this with a database-backed implementation that
// satisfies the same store.RoomStore and store.SessionStore interfaces.
//
// All three stores share a single sync.RWMutex. Reads are concurrent (RLock),
// writes are exclusive (Lock). This is safe because all writes are short and don't call
// back into external code while holding the lock.
//
//nolint:err113,gocritic // In-memory adapter favors compact runtime validation; stricter shaping deferred.
package memory

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"

	sessionapp "github.com/axwell0/elmakina/internal/app/session"
	"github.com/axwell0/elmakina/internal/apperrors"
	"github.com/axwell0/elmakina/internal/store"
)

// Store is the default in-memory persistence adapter for local rooms.
type Store struct {
	mu       sync.RWMutex
	rooms    map[string]sessionapp.LobbySnapshot
	sessions map[string]sessionapp.ClientSessionSnapshot
}

func NewStore() *Store {
	return &Store{
		rooms:    make(map[string]sessionapp.LobbySnapshot),
		sessions: make(map[string]sessionapp.ClientSessionSnapshot),
	}
}

func NewBundle() store.Bundle {
	memoryStore := NewStore()
	return store.Bundle{
		Rooms:    memoryStore,
		Sessions: memoryStore,
	}
}

func (s *Store) SaveRoom(_ context.Context, room sessionapp.LobbySnapshot) error {
	if room.ID == "" {
		return fmt.Errorf("room id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.ID] = cloneLobbySnapshot(room)
	return nil
}

func (s *Store) LoadRoom(_ context.Context, id string) (sessionapp.LobbySnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.rooms[id]
	if !ok {
		return sessionapp.LobbySnapshot{}, fmt.Errorf("load room %q: %w", id, apperrors.ErrRoomNotFound)
	}
	return cloneLobbySnapshot(room), nil
}

func (s *Store) ListRooms(_ context.Context) ([]sessionapp.LobbySnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.rooms))
	for id := range s.rooms {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rooms := make([]sessionapp.LobbySnapshot, 0, len(ids))
	for _, id := range ids {
		rooms = append(rooms, cloneLobbySnapshot(s.rooms[id]))
	}
	return rooms, nil
}

func (s *Store) DeleteRoom(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, id)
	prefix := sessionKeyPrefix(id)
	for key := range s.sessions {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(s.sessions, key)
		}
	}
	return nil
}

func (s *Store) SaveSession(_ context.Context, roomID string, session sessionapp.ClientSessionSnapshot) error {
	if roomID == "" {
		return fmt.Errorf("room id is required")
	}
	if session.ClientSessionID == "" {
		return fmt.Errorf("session id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionKey(roomID, session.ClientSessionID)] = session
	return nil
}

func (s *Store) LoadSession(_ context.Context, roomID string, sessionID string) (sessionapp.ClientSessionSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionKey(roomID, sessionID)]
	if !ok {
		return sessionapp.ClientSessionSnapshot{}, fmt.Errorf("load session %q in room %q: %w", sessionID, roomID, apperrors.ErrSessionNotFound)
	}
	return session, nil
}

func (s *Store) DeleteSession(_ context.Context, roomID string, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionKey(roomID, sessionID))
	return nil
}

func cloneLobbySnapshot(snapshot sessionapp.LobbySnapshot) sessionapp.LobbySnapshot {
	clone := snapshot
	clone.ReadyPlayers = slices.Clone(snapshot.ReadyPlayers)
	clone.Sessions = slices.Clone(snapshot.Sessions)
	clone.Game = *snapshot.Game.Snapshot()
	return clone
}

func sessionKey(roomID string, sessionID string) string {
	return sessionKeyPrefix(roomID) + sessionID
}

func sessionKeyPrefix(roomID string) string {
	return roomID + "\x00"
}
