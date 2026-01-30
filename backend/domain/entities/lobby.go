package entities

import (
	"fmt"
	"time"
)

// LobbyID is a type-safe identifier for lobbies.
type LobbyID string

func (id LobbyID) String() string { return string(id) }

// LobbyStatus represents the lifecycle state of a lobby.
type LobbyStatus string

const (
	LobbyOpen   LobbyStatus = "open"
	LobbyInGame LobbyStatus = "in_game"
	LobbyClosed LobbyStatus = "closed"
)

// Valid returns true if the status is a known value.
func (s LobbyStatus) Valid() bool {
	switch s {
	case LobbyOpen, LobbyInGame, LobbyClosed:
		return true
	}
	return false
}

// LobbyPlayer represents a player's membership in a lobby.
type LobbyPlayer struct {
	PlayerID PlayerID
	JoinedAt time.Time
}

// Lobby is a waiting room for players before a game starts.
// It enforces business invariants and maintains consistency.
type Lobby struct {
	id        LobbyID
	leaderID  PlayerID
	players   []LobbyPlayer // Ordered list maintains join order
	status    LobbyStatus
	version   int64 // Optimistic locking version
	createdAt time.Time
	updatedAt time.Time
}

// NewLobby creates a new lobby with the given leader.
func NewLobby(id LobbyID, leaderID PlayerID) (*Lobby, error) {
	if id == "" {
		return nil, fmt.Errorf("lobby id is required: %w", ErrInvalidEntity)
	}
	if leaderID == "" {
		return nil, fmt.Errorf("leader id is required: %w", ErrInvalidEntity)
	}

	now := time.Now().UTC()
	return &Lobby{
		id:        id,
		leaderID:  leaderID,
		players:   []LobbyPlayer{{PlayerID: leaderID, JoinedAt: now}},
		status:    LobbyOpen,
		version:   1,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ReconstituteLobby reconstructs a lobby from persistence.
// Use only within repository mappers.
func ReconstituteLobby(
	id LobbyID,
	leaderID PlayerID,
	players []LobbyPlayer,
	status LobbyStatus,
	version int64,
	createdAt, updatedAt time.Time,
) *Lobby {
	// Defensive copy of players slice
	playersCopy := make([]LobbyPlayer, len(players))
	copy(playersCopy, players)

	return &Lobby{
		id:        id,
		leaderID:  leaderID,
		players:   playersCopy,
		status:    status,
		version:   version,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// ID returns the lobby's unique identifier.
func (l *Lobby) ID() LobbyID { return l.id }

// LeaderID returns the current leader's player ID.
func (l *Lobby) LeaderID() PlayerID { return l.leaderID }

// Status returns the lobby's current status.
func (l *Lobby) Status() LobbyStatus { return l.status }

// Version returns the optimistic locking version.
func (l *Lobby) Version() int64 { return l.version }

// CreatedAt returns the creation timestamp.
func (l *Lobby) CreatedAt() time.Time { return l.createdAt }

// UpdatedAt returns the last update timestamp.
func (l *Lobby) UpdatedAt() time.Time { return l.updatedAt }

// PlayerIDs returns a slice of all player IDs in join order.
func (l *Lobby) PlayerIDs() []PlayerID {
	ids := make([]PlayerID, len(l.players))
	for i, p := range l.players {
		ids[i] = p.PlayerID
	}
	return ids
}

// PlayerCount returns the number of players in the lobby.
func (l *Lobby) PlayerCount() int { return len(l.players) }

// HasPlayer returns true if the player is in the lobby.
func (l *Lobby) HasPlayer(playerID PlayerID) bool {
	for _, p := range l.players {
		if p.PlayerID == playerID {
			return true
		}
	}
	return false
}

// IsOpen returns true if the lobby is accepting new players.
func (l *Lobby) IsOpen() bool { return l.status == LobbyOpen }

// IsInGame returns true if the lobby has an active game.
func (l *Lobby) IsInGame() bool { return l.status == LobbyInGame }

// AddPlayer adds a player to the lobby if it's open and not full.
func (l *Lobby) AddPlayer(playerID PlayerID, maxPlayers int) error {
	if !l.IsOpen() {
		return fmt.Errorf("lobby is not open: %w", ErrLobbyNotOpen)
	}
	if l.HasPlayer(playerID) {
		return fmt.Errorf("player already in lobby: %w", ErrPlayerAlreadyInLobby)
	}
	if len(l.players) >= maxPlayers {
		return fmt.Errorf("lobby is full: %w", ErrLobbyFull)
	}

	l.players = append(l.players, LobbyPlayer{
		PlayerID: playerID,
		JoinedAt: time.Now().UTC(),
	})
	l.version++
	l.updatedAt = time.Now().UTC()
	return nil
}

// RemovePlayer removes a player from the lobby.
// If the leader leaves, the next player becomes leader.
// Returns true if the lobby should be closed (empty).
func (l *Lobby) RemovePlayer(playerID PlayerID) (shouldClose bool, err error) {
	if !l.IsOpen() {
		return false, fmt.Errorf("lobby is not open: %w", ErrLobbyNotOpen)
	}

	idx := -1
	for i, p := range l.players {
		if p.PlayerID == playerID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false, fmt.Errorf("player not in lobby: %w", ErrPlayerNotInLobby)
	}

	// Remove player while preserving order
	l.players = append(l.players[:idx], l.players[idx+1:]...)

	// If lobby is now empty, mark for closure
	if len(l.players) == 0 {
		l.status = LobbyClosed
		l.version++
		l.updatedAt = time.Now().UTC()
		return true, nil
	}

	// If leader left, promote next player
	if l.leaderID == playerID {
		l.leaderID = l.players[0].PlayerID
	}

	l.version++
	l.updatedAt = time.Now().UTC()
	return false, nil
}

// StartGame transitions the lobby to in-game status.
func (l *Lobby) StartGame(minPlayers int) error {
	if !l.IsOpen() {
		return fmt.Errorf("lobby is not open: %w", ErrLobbyNotOpen)
	}
	if len(l.players) < minPlayers {
		return fmt.Errorf("not enough players (need %d, have %d): %w",
			minPlayers, len(l.players), ErrNotEnoughPlayers)
	}

	l.status = LobbyInGame
	l.version++
	l.updatedAt = time.Now().UTC()
	return nil
}

// ResetToOpen transitions an in-game lobby back to open.
// Used after server restart when sessions are lost.
func (l *Lobby) ResetToOpen() error {
	if l.status != LobbyInGame {
		return fmt.Errorf("lobby is not in game: %w", ErrInvalidStatusTransition)
	}

	l.status = LobbyOpen
	l.version++
	l.updatedAt = time.Now().UTC()
	return nil
}

// CanStart returns true if the lobby can be started by the given player.
func (l *Lobby) CanStart(playerID PlayerID, minPlayers int) error {
	if l.leaderID != playerID {
		return fmt.Errorf("only leader can start: %w", ErrNotLeader)
	}
	if !l.IsOpen() {
		return fmt.Errorf("lobby is not open: %w", ErrLobbyNotOpen)
	}
	if len(l.players) < minPlayers {
		return fmt.Errorf("not enough players: %w", ErrNotEnoughPlayers)
	}
	return nil
}
