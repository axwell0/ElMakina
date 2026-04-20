package ws

import (
	"context"
	"sync"
)

// LiveConnection is the websocket transport capability Room needs.
type LiveConnection interface {
	Send(context.Context, any) error
	Close() error
}

// ConnectionRegistry owns active session/player to live-connection mappings.
type ConnectionRegistry struct {
	mu        sync.RWMutex
	bySession map[string]LiveConnection
	byPlayer  map[int]LiveConnection
}

func NewConnectionRegistry() *ConnectionRegistry {
	return &ConnectionRegistry{
		bySession: make(map[string]LiveConnection),
		byPlayer:  make(map[int]LiveConnection),
	}
}

func (r *ConnectionRegistry) Attach(sessionID string, playerIndex int, connection LiveConnection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bySession[sessionID] = connection
	r.byPlayer[playerIndex] = connection
}

func (r *ConnectionRegistry) DetachIfCurrent(sessionID string, playerIndex int, connection LiveConnection) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	currentSession := r.bySession[sessionID]
	if currentSession == nil || currentSession != connection {
		return false
	}
	delete(r.bySession, sessionID)
	if currentPlayer := r.byPlayer[playerIndex]; currentPlayer == connection {
		delete(r.byPlayer, playerIndex)
	}
	return true
}

func (r *ConnectionRegistry) Connection(sessionID string) (LiveConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	connection, ok := r.bySession[sessionID]
	return connection, ok
}

func (r *ConnectionRegistry) ConnectionForPlayer(playerIndex int) (LiveConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	connection, ok := r.byPlayer[playerIndex]
	return connection, ok
}
