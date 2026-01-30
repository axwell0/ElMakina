package server

import (
	"ElMakina/backend/engine"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
)

// LobbyStatus tracks where a lobby is in its lifecycle.
//
// The flow is: Open → InGame → Closed
//   - Open: Players can join, leader can start the game
//   - InGame: Game is running, no new players can join
//   - Closed: Game ended, lobby is being cleaned up
type LobbyStatus string

const (
	LobbyOpen   LobbyStatus = "open"    // Waiting for players
	LobbyInGame LobbyStatus = "in_game" // Game is active
	LobbyClosed LobbyStatus = "closed"  // Game ended
)

// Player represents a human being sitting at their computer wanting to play.
//
// This is the "Identity Layer" of the system. While the engine only cares about
// "Player 0" and "Player 1", the lobby system cares about "Alice" and "Bob".
// The Player struct bridges that gap.
//
// ID vs Token:
//   - ID is the public identifier (shown in UI, used in APIs)
//   - Token is a SECRET used for reconnection. If Alice's browser refreshes,
//     she uses her token to prove "I'm still Alice" and rejoin her game.
//
// Nick is the display name shown to other players. Avatar is an optional
// cosmetic identifier (could be an image URL, an emoji, etc.).
type Player struct {
	ID     string // Public unique identifier
	Nick   string // Display name (e.g., "Alice", "xX_DarkSlayer_Xx")
	Token  string // Secret reconnection token (keep this safe!)
	Avatar string // Optional cosmetic identifier
}

// Lobby is a waiting room for players before a game starts.
type Lobby struct {
	ID        string
	LeaderID  string
	PlayerIDs []string
	Status    LobbyStatus
}

// LobbySummary is a read-only view for lobby listing.
type LobbySummary struct {
	ID          string
	LeaderNick  string
	LeaderID    string
	PlayerCount int
	PlayerNicks []string
	PlayerIDs   []string
	Status      LobbyStatus
}

// LobbyManager is the "Gateway" between the outside world and the Engine.
//
// Think of this as the bridge. On one side, you have the chaotic internet: Users with
// arbitrary strings for IDs, sessions that drop, and nicknames like "DreadPirateRoberts".
// On the other side, you have the Game Engine, which is a sterile environment that
// only understands "Player 0", "Player 1", etc.
//
// LobbyManager's job is to:
//  1. Manage the waiting rooms (Lobbies).
//  2. Map UserIDs ("alice-123") to Engine Indices (0).
//  3. Keep the "persistence" alive. We use a snapshotting mechanism here: every time
//     something important happens (someone joins, a game starts), we save the entire
//     state to the Store. This way, if the server crashes, we can wake up and know
//     exactly who was in what room.
//
// Thread Safety: Everything here is wrapped in a sync.RWMutex. Because many people
// can list lobbies at once, but only one can join a specific lobby at a time, we
// use the RLock/Lock pattern to keep things performant but safe.
//
// See: server/ws/server.go:88 for WebSocket integration.
// See: server/ws/session_runner.go:201 for game session management.
type LobbyManager struct {
	mu          sync.RWMutex
	lobbies     map[string]*Lobby
	players     map[string]*Player
	byToken     map[string]string // Maps secret tokens back to IDs (for reconnection).
	nextLobbyID uint64
	minCount    int
	maxCount    int
	store       LobbyStore // The persistent "Memory" of the manager.
}

// NewLobbyManager constructs a lobby manager with min/max constraints.
func NewLobbyManager(minPlayers, maxPlayers int) *LobbyManager {
	manager, _ := NewLobbyManagerWithStore(minPlayers, maxPlayers, nil)
	return manager
}

// NewLobbyManagerWithStore constructs a lobby manager that persists to the given store.
func NewLobbyManagerWithStore(minPlayers, maxPlayers int, store LobbyStore) (*LobbyManager, error) {
	manager := &LobbyManager{
		lobbies:  make(map[string]*Lobby),
		players:  make(map[string]*Player),
		byToken:  make(map[string]string),
		minCount: minPlayers,
		maxCount: maxPlayers,
		store:    store,
	}
	if store == nil {
		return manager, nil
	}
	snapshot, err := store.Load(context.Background())
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return manager, nil
	}
	manager.restoreSnapshot(*snapshot)
	return manager, nil
}

// RegisterPlayer registers a nickname and returns a stable player ID and token.
func (m *LobbyManager) RegisterPlayer(nick string) (*Player, error) {
	if nick == "" {
		return nil, fmt.Errorf("nickname is required")
	}
	m.mu.Lock()

	id, err := newUUID()
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	token, err := newToken()
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	player := &Player{ID: id, Nick: nick, Token: token}
	m.players[id] = player
	m.byToken[token] = id
	// Return a copy so callers never observe concurrent mutations after unlocking.
	copyPlayer := *player
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	if err := m.saveSnapshot(snapshot); err != nil {
		return &copyPlayer, err
	}
	return &copyPlayer, nil
}

// SetPlayerAvatar updates a player's avatar payload.
func (m *LobbyManager) SetPlayerAvatar(playerID, avatar string) error {
	if avatar == "" {
		return nil
	}
	m.mu.Lock()
	player := m.players[playerID]
	if player == nil {
		m.mu.Unlock()
		return fmt.Errorf("player not found")
	}
	player.Avatar = avatar
	snapshot := m.snapshotLocked()
	m.mu.Unlock()
	return m.saveSnapshot(snapshot)
}

// ReconnectPlayer returns the existing player for a reconnect token.
func (m *LobbyManager) ReconnectPlayer(token string) (*Player, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	playerID := m.byToken[token]
	if playerID == "" {
		return nil, fmt.Errorf("invalid token")
	}
	player := m.players[playerID]
	if player == nil {
		return nil, fmt.Errorf("player not found")
	}
	// Return a copy so callers never observe concurrent mutations after unlocking.
	copyPlayer := *player
	return &copyPlayer, nil
}

// UnregisterPlayer removes a player entirely when they are not in a lobby.
func (m *LobbyManager) UnregisterPlayer(playerID string) error {
	m.mu.Lock()
	player := m.players[playerID]
	if player == nil {
		m.mu.Unlock()
		return fmt.Errorf("player not found")
	}
	for _, lobby := range m.lobbies {
		for _, id := range lobby.PlayerIDs {
			if id == playerID {
				m.mu.Unlock()
				return fmt.Errorf("player still in lobby")
			}
		}
	}
	delete(m.players, playerID)
	delete(m.byToken, player.Token)
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	return m.saveSnapshot(snapshot)
}

// CreateLobby creates a new lobby with the given leader as its first member.
func (m *LobbyManager) CreateLobby(leaderID string) (*Lobby, error) {
	m.mu.Lock()

	if _, ok := m.players[leaderID]; !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("leader not registered")
	}
	lobbyID := m.nextLobbyIDValue()
	lobby := &Lobby{
		ID:        lobbyID,
		LeaderID:  leaderID,
		PlayerIDs: []string{leaderID},
		Status:    LobbyOpen,
	}
	m.lobbies[lobbyID] = lobby
	// Return a copy so callers never observe concurrent mutations after unlocking.
	copyLobby := *lobby
	copyLobby.PlayerIDs = append([]string{}, lobby.PlayerIDs...)
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	if err := m.saveSnapshot(snapshot); err != nil {
		return &copyLobby, err
	}
	return &copyLobby, nil
}

// ListLobbies returns a snapshot of all lobbies.
func (m *LobbyManager) ListLobbies() []LobbySummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]LobbySummary, 0, len(m.lobbies))
	for _, lobby := range m.lobbies {
		leader := m.players[lobby.LeaderID]
		leaderNick := ""
		if leader != nil {
			leaderNick = leader.Nick
		}
		playerNicks := make([]string, 0, len(lobby.PlayerIDs))
		playerIDs := make([]string, 0, len(lobby.PlayerIDs))
		for _, id := range lobby.PlayerIDs {
			if player := m.players[id]; player != nil {
				playerNicks = append(playerNicks, player.Nick)
				playerIDs = append(playerIDs, player.ID)
			}
		}
		result = append(result, LobbySummary{
			ID:          lobby.ID,
			LeaderNick:  leaderNick,
			LeaderID:    lobby.LeaderID,
			PlayerCount: len(lobby.PlayerIDs),
			PlayerNicks: playerNicks,
			PlayerIDs:   playerIDs,
			Status:      lobby.Status,
		})
	}
	return result
}

// PruneEmptyOpenLobbies removes offline players from open lobbies, and deletes any
// open lobby that ends up with zero online members.
// onlinePlayers should contain player IDs with active connections.
func (m *LobbyManager) PruneEmptyOpenLobbies(onlinePlayers map[string]struct{}) ([]string, error) {
	m.mu.Lock()
	removed := make([]string, 0)
	changed := false
	for id, lobby := range m.lobbies {
		if lobby.Status != LobbyOpen {
			continue
		}
		pruned := make([]string, 0, len(lobby.PlayerIDs))
		for _, playerID := range lobby.PlayerIDs {
			if _, ok := onlinePlayers[playerID]; ok {
				pruned = append(pruned, playerID)
			}
		}
		if len(pruned) == 0 {
			delete(m.lobbies, id)
			removed = append(removed, id)
			changed = true
			continue
		}
		if len(pruned) != len(lobby.PlayerIDs) {
			lobby.PlayerIDs = pruned
			if _, ok := onlinePlayers[lobby.LeaderID]; !ok {
				lobby.LeaderID = pruned[0]
			}
			changed = true
		}
	}
	if !changed {
		m.mu.Unlock()
		return nil, nil
	}
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	if err := m.saveSnapshot(snapshot); err != nil {
		return removed, err
	}
	return removed, nil
}

// JoinLobby adds a player to an open lobby.
func (m *LobbyManager) JoinLobby(lobbyID, playerID string) error {
	m.mu.Lock()

	lobby := m.lobbies[lobbyID]
	if lobby == nil {
		m.mu.Unlock()
		return fmt.Errorf("lobby not found")
	}
	if lobby.Status != LobbyOpen {
		m.mu.Unlock()
		return fmt.Errorf("lobby not open")
	}
	if _, ok := m.players[playerID]; !ok {
		m.mu.Unlock()
		return fmt.Errorf("player not registered")
	}
	for _, id := range lobby.PlayerIDs {
		if id == playerID {
			m.mu.Unlock()
			return fmt.Errorf("player already in lobby")
		}
	}
	if len(lobby.PlayerIDs) >= m.maxCount {
		m.mu.Unlock()
		return fmt.Errorf("lobby is full")
	}
	lobby.PlayerIDs = append(lobby.PlayerIDs, playerID)
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	return m.saveSnapshot(snapshot)
}

// StartLobby transitions a lobby into a running game session.
func (m *LobbyManager) StartLobby(lobbyID, leaderID string) (*GameSession, error) {
	m.mu.Lock()

	lobby := m.lobbies[lobbyID]
	if lobby == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("lobby not found")
	}
	if lobby.LeaderID != leaderID {
		m.mu.Unlock()
		return nil, fmt.Errorf("only leader can start lobby")
	}
	if lobby.Status != LobbyOpen {
		m.mu.Unlock()
		return nil, fmt.Errorf("lobby not open")
	}
	if len(lobby.PlayerIDs) < m.minCount {
		m.mu.Unlock()
		return nil, fmt.Errorf("not enough players")
	}

	matchID, err := newUUID()
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	matchSeed, err := newMatchSeed()
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	matchSeedHex := hex.EncodeToString(matchSeed[:])
	matchRNG := rand.New(rand.NewChaCha8(matchSeed))

	names := make([]string, 0, len(lobby.PlayerIDs))
	avatars := make([]string, 0, len(lobby.PlayerIDs))
	indexByPlayer := make(map[string]int, len(lobby.PlayerIDs))
	for i, id := range lobby.PlayerIDs {
		player := m.players[id]
		if player == nil {
			m.mu.Unlock()
			return nil, fmt.Errorf("player %s not registered", id)
		}
		names = append(names, player.Nick)
		avatars = append(avatars, player.Avatar)
		indexByPlayer[id] = i
	}
	game, err := engine.NewGame(names, matchRNG)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	lobby.Status = LobbyInGame
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	if err := m.saveSnapshot(snapshot); err != nil {
		return nil, err
	}
	return &GameSession{
		MatchID:       matchID,
		LobbyID:       lobby.ID,
		Game:          game,
		PlayerIDs:     append([]string{}, lobby.PlayerIDs...),
		IndexByPlayer: indexByPlayer,
		PlayerAvatars: avatars,
		RNGSeed:       matchSeedHex,
	}, nil
}

// RemovePlayer removes a player from an open lobby. If the leader leaves, the
// next player becomes leader. Empty lobbies are closed.
func (m *LobbyManager) RemovePlayer(lobbyID, playerID string) error {
	m.mu.Lock()

	lobby := m.lobbies[lobbyID]
	if lobby == nil {
		m.mu.Unlock()
		return fmt.Errorf("lobby not found")
	}
	if lobby.Status != LobbyOpen {
		m.mu.Unlock()
		return fmt.Errorf("lobby not open")
	}
	index := -1
	for i, id := range lobby.PlayerIDs {
		if id == playerID {
			index = i
			break
		}
	}
	if index == -1 {
		m.mu.Unlock()
		return fmt.Errorf("player not in lobby")
	}
	lobby.PlayerIDs = append(lobby.PlayerIDs[:index], lobby.PlayerIDs[index+1:]...)
	if lobby.LeaderID == playerID {
		if len(lobby.PlayerIDs) > 0 {
			lobby.LeaderID = lobby.PlayerIDs[0]
		} else {
			lobby.Status = LobbyClosed
		}
	}
	if len(lobby.PlayerIDs) == 0 {
		delete(m.lobbies, lobbyID)
	}
	snapshot := m.snapshotLocked()
	m.mu.Unlock()

	return m.saveSnapshot(snapshot)
}

// GetLobby returns a copy of the lobby data.
func (m *LobbyManager) GetLobby(lobbyID string) (*Lobby, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lobby := m.lobbies[lobbyID]
	if lobby == nil {
		return nil, fmt.Errorf("lobby not found")
	}
	copyLobby := *lobby
	copyLobby.PlayerIDs = append([]string{}, lobby.PlayerIDs...)
	return &copyLobby, nil
}

// GetPlayer returns a copy of a player record.
func (m *LobbyManager) GetPlayer(playerID string) (*Player, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	player := m.players[playerID]
	if player == nil {
		return nil, fmt.Errorf("player not found")
	}
	copyPlayer := *player
	return &copyPlayer, nil
}

// FindLobbyByPlayer returns the lobby ID containing the player, if any.
func (m *LobbyManager) FindLobbyByPlayer(playerID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, lobby := range m.lobbies {
		for _, id := range lobby.PlayerIDs {
			if id == playerID {
				return lobby.ID, true
			}
		}
	}
	return "", false
}

// ResetInGameLobbiesToOpen downgrades any in-game lobbies to open.
// This is useful after a server restart when active sessions are not recoverable.
func (m *LobbyManager) ResetInGameLobbiesToOpen() error {
	m.mu.Lock()
	changed := false
	for _, lobby := range m.lobbies {
		if lobby.Status == LobbyInGame {
			lobby.Status = LobbyOpen
			changed = true
		}
	}
	snapshot := m.snapshotLocked()
	m.mu.Unlock()
	if !changed {
		return nil
	}
	return m.saveSnapshot(snapshot)
}

// ResetLobbyToOpen marks a specific lobby as open if it exists.
func (m *LobbyManager) ResetLobbyToOpen(lobbyID string) error {
	m.mu.Lock()
	lobby := m.lobbies[lobbyID]
	if lobby == nil {
		m.mu.Unlock()
		return fmt.Errorf("lobby not found")
	}
	if lobby.Status == LobbyOpen {
		m.mu.Unlock()
		return nil
	}
	lobby.Status = LobbyOpen
	snapshot := m.snapshotLocked()
	m.mu.Unlock()
	return m.saveSnapshot(snapshot)
}

func (m *LobbyManager) nextLobbyIDValue() string {
	id := atomic.AddUint64(&m.nextLobbyID, 1)
	return fmt.Sprintf("lobby-%d", id)
}

func (m *LobbyManager) snapshotLocked() LobbySnapshot {
	lobbies := make(map[string]Lobby, len(m.lobbies))
	for id, lobby := range m.lobbies {
		copyLobby := *lobby
		copyLobby.PlayerIDs = append([]string{}, lobby.PlayerIDs...)
		lobbies[id] = copyLobby
	}
	players := make(map[string]Player, len(m.players))
	for id, player := range m.players {
		players[id] = *player
	}
	byToken := make(map[string]string, len(m.byToken))
	for token, id := range m.byToken {
		byToken[token] = id
	}
	return LobbySnapshot{
		Lobbies:     lobbies,
		Players:     players,
		ByToken:     byToken,
		NextLobbyID: m.nextLobbyID,
	}
}

func (m *LobbyManager) restoreSnapshot(snapshot LobbySnapshot) {
	m.lobbies = make(map[string]*Lobby, len(snapshot.Lobbies))
	for id, lobby := range snapshot.Lobbies {
		copyLobby := lobby
		m.lobbies[id] = &copyLobby
	}
	m.players = make(map[string]*Player, len(snapshot.Players))
	for id, player := range snapshot.Players {
		copyPlayer := player
		m.players[id] = &copyPlayer
	}
	m.byToken = make(map[string]string, len(snapshot.ByToken))
	for token, id := range snapshot.ByToken {
		m.byToken[token] = id
	}
	m.nextLobbyID = snapshot.NextLobbyID
}

func (m *LobbyManager) saveSnapshot(snapshot LobbySnapshot) error {
	if m.store == nil {
		return nil
	}
	return m.store.Save(context.Background(), snapshot)
}

func newToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func newUUID() (string, error) {
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:]), nil
}

func newMatchSeed() ([32]byte, error) {
	var seed [32]byte
	if _, err := crand.Read(seed[:]); err != nil {
		return seed, err
	}
	return seed, nil
}
