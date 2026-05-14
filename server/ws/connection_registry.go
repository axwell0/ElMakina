// connection_registry.go — thread-safe mapping of live WebSocket clients.
//
// # Why two maps?
//
// The game has two common access patterns:
//   - "Send a message to session abc-123"  → look up by session ID (bySession)
//   - "Send a message to player 2"         → look up by player index (sessionByPlayer)
//
// Maintaining both indexes lets both lookups be O(1) without scanning.
// Attach and DetachIfCurrent always update both together, so they never diverge.
//
// # Thread safety
//
// All writes are guarded by a sync.RWMutex (mu). Reads that just need a
// single value use RLock so multiple goroutines can look up concurrently.
// The Attach method releases the write lock before closing stale connections
// so it doesn't hold a lock while doing I/O.
package ws

import (
	"sort"
	"sync"
)

// ConnectionRegistry keeps the active session-to-socket and player-to-socket
// mappings in one place so the room can update them atomically.
//
// Two maps are maintained for two different query patterns:
//   - bySession (sessionID → socket): used when sending to a specific session,
//     e.g. "send the prompt only to session abc-123"
//   - sessionByPlayer (playerIndex → sessionID): used when broadcasting by player number,
//     e.g. "send the turn result to player 0, 1, 2..."
//
// Both maps always reflect the same live connections — they're two indexes over the same data.
// Attach and DetachIfCurrent keep them in sync together, so callers never see them diverge.
type ConnectionRegistry struct {
	mu              sync.RWMutex
	bySession       map[string]LiveClient
	sessionByPlayer map[int]string
}

// NewConnectionRegistry starts with empty maps so lookups stay nil-safe.
func NewConnectionRegistry() *ConnectionRegistry {
	// Create the backing maps eagerly so attach and lookup never have to guard against nil maps.
	return &ConnectionRegistry{
		bySession:       make(map[string]LiveClient),
		sessionByPlayer: make(map[int]string),
	}
}

// Attach records the live connection and closes any stale connection for the
// same session or player.
func (r *ConnectionRegistry) Attach(sessionID string, playerIndex int, connection LiveClient) error {
	if r == nil || connection == nil {
		return nil
	}
	r.mu.Lock()
	replaced := make([]LiveClient, 0, 2)
	if current := r.bySession[sessionID]; current != nil && current != connection {
		replaced = appendUniqueLiveClient(replaced, current)
	}
	if previousSessionID := r.sessionByPlayer[playerIndex]; previousSessionID != "" && previousSessionID != sessionID {
		if current := r.bySession[previousSessionID]; current != nil && current != connection {
			replaced = appendUniqueLiveClient(replaced, current)
		}
		delete(r.bySession, previousSessionID)
	}
	r.bySession[sessionID] = connection
	r.sessionByPlayer[playerIndex] = sessionID
	r.mu.Unlock()

	var firstErr error
	for _, previous := range replaced {
		if err := previous.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// DetachIfCurrent removes the mapping only when the supplied connection is
// still the one currently registered for that session.
func (r *ConnectionRegistry) DetachIfCurrent(sessionID string, playerIndex int, connection LiveClient) bool {
	// Hold the lock while we verify that the session still points at the same socket.
	r.mu.Lock()
	defer r.mu.Unlock()
	currentSession := r.bySession[sessionID]
	if currentSession == nil || currentSession != connection {
		return false
	}
	// Remove the session mapping first, then clean up the player mapping if it matches too.
	delete(r.bySession, sessionID)
	if currentSessionID := r.sessionByPlayer[playerIndex]; currentSessionID == sessionID {
		delete(r.sessionByPlayer, playerIndex)
	}
	return true
}

// HasSession reports whether the session currently has a live transport.
func (r *ConnectionRegistry) HasSession(sessionID string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bySession[sessionID] != nil
}

// Connection returns the active transport for a session if one is present.
func (r *ConnectionRegistry) Connection(sessionID string) (LiveClient, bool) {
	// Read lock keeps lookups cheap while still safe during concurrent reconnects.
	r.mu.RLock()
	defer r.mu.RUnlock()
	connection, ok := r.bySession[sessionID]
	return connection, ok
}

// ConnectionForPlayer resolves the current transport from the player index.
func (r *ConnectionRegistry) ConnectionForPlayer(playerIndex int) (LiveClient, bool) {
	// Player lookups share the same registry data, just indexed by player number instead of session ID.
	r.mu.RLock()
	defer r.mu.RUnlock()
	sessionID := r.sessionByPlayer[playerIndex]
	if sessionID == "" {
		return nil, false
	}
	connection, ok := r.bySession[sessionID]
	return connection, ok
}

// ConnectedPlayers returns sorted player indices that currently have a live transport.
func (r *ConnectionRegistry) ConnectedPlayers() []int {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	players := make([]int, 0, len(r.sessionByPlayer))
	for playerIndex, sessionID := range r.sessionByPlayer {
		if r.bySession[sessionID] != nil {
			players = append(players, playerIndex)
		}
	}
	sort.Ints(players)
	return players
}

// CloseAll closes every current transport and clears the registry.
func (r *ConnectionRegistry) CloseAll() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	connections := make([]LiveClient, 0, len(r.bySession))
	for _, connection := range r.bySession {
		if connection != nil {
			connections = append(connections, connection)
		}
	}
	r.bySession = make(map[string]LiveClient)
	r.sessionByPlayer = make(map[int]string)
	r.mu.Unlock()

	var firstErr error
	for _, connection := range connections {
		if err := connection.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func appendUniqueLiveClient(connections []LiveClient, candidate LiveClient) []LiveClient {
	for _, existing := range connections {
		if existing == candidate {
			return connections
		}
	}
	return append(connections, candidate)
}
