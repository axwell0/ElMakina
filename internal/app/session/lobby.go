//nolint:err113,gocritic // Legacy lobby API surface; keeping behavior stable while broader refactor is pending.
package session

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	coreconfig "github.com/axwell0/elmakina/engine/core/config"
	corestate "github.com/axwell0/elmakina/engine/core/state"
	"github.com/axwell0/elmakina/internal/apperrors"
)

// Phase is the hosted-session lifecycle state.
type Phase string

const (
	PhaseLobby    Phase = "lobby"
	PhaseInTurn   Phase = "in_turn"
	PhaseGameOver Phase = "game_over"
)

// Lobby is the application aggregate for one hosted room before, during, and
// after its current game.
type Lobby struct {
	id       string
	game     *GameSession
	sessions map[string]*ClientSession

	mu                sync.RWMutex
	phase             Phase
	leaderPlayerIndex int
	playerCount       int
	readyPlayers      map[int]bool
	sessionsByPlayer  map[int]*ClientSession
}

// NewLobby creates a hosted lobby with an initialized current game.
func NewLobby(id string, leaderName string, playerCount int) (*Lobby, error) {
	return NewLobbyWithConfig(id, leaderName, playerCount, coreconfig.DefaultGameConfig())
}

func NewLobbyWithConfig(id string, leaderName string, playerCount int, cfg coreconfig.GameConfig) (*Lobby, error) {
	if id == "" {
		return nil, fmt.Errorf("session id is required")
	}
	game, err := NewGameSessionWithConfig(id, leaderName, playerCount, cfg)
	if err != nil {
		return nil, err
	}
	return &Lobby{
		id:                id,
		game:              game,
		sessions:          make(map[string]*ClientSession),
		phase:             PhaseLobby,
		leaderPlayerIndex: 0,
		playerCount:       playerCount,
		readyPlayers:      make(map[int]bool),
		sessionsByPlayer:  make(map[int]*ClientSession, playerCount),
	}, nil
}

type LobbySnapshot struct {
	ID                string
	Phase             Phase
	LeaderPlayerIndex int
	PlayerCount       int
	JoinedPlayers     int
	ReadyPlayers      []int
	Sessions          []ClientSessionSnapshot
	Game              corestate.GameState
	GameConfig        coreconfig.GameConfig
}

type ClientSessionSnapshot struct {
	ClientSessionID string
	PlayerIndex     int
	PlayerName      string
	DeviceID        string
	LastSeenAt      time.Time
}

// ID returns the stable room/lobby identifier.
func (l *Lobby) ID() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.id
}

// Game returns the current game session owned by this lobby.
func (l *Lobby) Game() *GameSession {
	return l.game
}

// Snapshot returns a deep copy of the lobby and game state.
func (l *Lobby) Snapshot() LobbySnapshot {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return LobbySnapshot{
		ID:                l.id,
		Phase:             l.phase,
		LeaderPlayerIndex: l.leaderPlayerIndex,
		PlayerCount:       l.playerCount,
		JoinedPlayers:     l.joinedPlayerCount(),
		ReadyPlayers:      l.readyPlayerIndicesLocked(),
		Sessions:          l.sessionSnapshotsLocked(),
		Game:              l.game.Snapshot(),
		GameConfig:        l.game.Config(),
	}
}

// Sessions returns stable, read-only snapshots of registered sessions.
func (l *Lobby) Sessions() []ClientSessionSnapshot {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.sessionSnapshotsLocked()
}

// AddSession registers a durable player session, even when it is disconnected.
func (l *Lobby) AddSession(session *ClientSession) error {
	if session == nil {
		return fmt.Errorf("client session is nil")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.sessions[session.ClientSessionID]; exists {
		return fmt.Errorf("session %q already exists", session.ClientSessionID)
	}
	if existing := l.sessionForPlayerLocked(session.PlayerIndex); existing != nil {
		return fmt.Errorf("player %d already has a session", session.PlayerIndex)
	}
	l.sessions[session.ClientSessionID] = session
	l.ensureSessionIndexLocked()[session.PlayerIndex] = session
	return nil
}

// Session returns the stored durable session by ID.
func (l *Lobby) Session(id string) *ClientSession {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.sessions[id]
}

// SessionForPlayer finds the current durable session for a player.
func (l *Lobby) SessionForPlayer(playerIndex int) *ClientSession {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.sessionForPlayerLocked(playerIndex)
}

// SessionList returns a stable snapshot of registered sessions.
func (l *Lobby) SessionList() []*ClientSession {
	l.mu.RLock()
	defer l.mu.RUnlock()
	sessions := make([]*ClientSession, 0, len(l.sessions))
	for _, session := range l.sessions {
		if session != nil {
			sessions = append(sessions, session)
		}
	}
	slices.SortFunc(sessions, func(a, b *ClientSession) int {
		return cmp.Compare(a.PlayerIndex, b.PlayerIndex)
	})
	return sessions
}

// MarkReady updates lobby readiness for one player.
func (l *Lobby) MarkReady(playerIndex int, ready bool) error {
	if playerIndex < 0 || playerIndex >= l.game.PlayerCount() {
		return fmt.Errorf("player index %d out of range", playerIndex)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.playerJoined(playerIndex) {
		return fmt.Errorf("player %d has not joined this room", playerIndex)
	}
	l.readyPlayers[playerIndex] = ready
	return nil
}

// CanStart reports whether every player is connected and marked ready.
func (l *Lobby) CanStart() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.phase != PhaseLobby {
		return false
	}
	if l.joinedPlayerCount() != l.playerCount {
		return false
	}
	for playerIndex := 0; playerIndex < l.game.PlayerCount(); playerIndex++ {
		if !l.playerJoined(playerIndex) {
			return false
		}
		session := l.sessionForPlayerLocked(playerIndex)
		if session == nil || !l.readyPlayers[playerIndex] {
			return false
		}
	}
	return true
}

// StartGame transitions lobby state into active-turn mode when prerequisites hold.
func (l *Lobby) StartGame() error {
	if !l.CanStart() {
		return fmt.Errorf("session %q cannot start yet", l.id)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.phase = PhaseInTurn
	clear(l.readyPlayers)
	return nil
}

// RestartGame creates a fresh game instance while preserving the player roster.
func (l *Lobby) RestartGame() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.game.hasActiveTurn() {
		return fmt.Errorf("session %q cannot restart during an active turn", l.id)
	}
	if l.phase != PhaseLobby && l.phase != PhaseGameOver {
		return fmt.Errorf("session %q cannot restart from phase %q", l.id, l.phase)
	}

	game, err := NewGameSessionWithConfig(l.id, "", l.game.PlayerCount(), l.game.Config())
	if err != nil {
		return err
	}
	if err := game.copyPlayerNamesFrom(l.game); err != nil {
		return err
	}

	l.game = game
	l.phase = PhaseLobby
	clear(l.readyPlayers)
	return nil
}

// Phase returns the current lifecycle phase.
func (l *Lobby) Phase() Phase {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.phase
}

// MarkGameOver moves the lobby to the terminal phase.
func (l *Lobby) MarkGameOver() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.phase = PhaseGameOver
}

// MarkInTurn records that the lobby is between turns and can continue.
func (l *Lobby) MarkInTurn() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.phase = PhaseInTurn
}

// Leader returns the player who owns lobby controls.
func (l *Lobby) Leader() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.leaderPlayerIndex
}

// IsLeader reports whether the player owns the lobby controls.
func (l *Lobby) IsLeader(playerIndex int) bool {
	return l.Leader() == playerIndex
}

// SetLeader moves lobby controls to a joined player.
func (l *Lobby) SetLeader(playerIndex int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.playerJoined(playerIndex) {
		return fmt.Errorf("player %d has not joined this room", playerIndex)
	}
	l.leaderPlayerIndex = playerIndex
	return nil
}

// CanRestart reports whether the lobby can be reset safely.
func (l *Lobby) CanRestart() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return !l.game.hasActiveTurn() && (l.phase == PhaseLobby || l.phase == PhaseGameOver)
}

// CanDelete reports whether the lobby can be removed safely.
func (l *Lobby) CanDelete() bool {
	return l.CanRestart()
}

// ReadyPlayerIndices returns a stable list of ready players.
func (l *Lobby) ReadyPlayerIndices() []int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	indices := make([]int, 0, len(l.readyPlayers))
	for playerIndex, ready := range l.readyPlayers {
		if ready && l.playerJoined(playerIndex) {
			indices = append(indices, playerIndex)
		}
	}
	slices.Sort(indices)
	return indices
}

// PlayerCount returns the configured room capacity.
func (l *Lobby) PlayerCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.playerCount
}

// JoinedPlayerCount returns the number of claimed player slots.
func (l *Lobby) JoinedPlayerCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.joinedPlayerCount()
}

// JoinPlayer claims the next open lobby slot for the provided player name.
func (l *Lobby) JoinPlayer(playerName string, now time.Time) (*ClientSession, error) {
	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return nil, fmt.Errorf("player name is required")
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.phase != PhaseLobby {
		return nil, fmt.Errorf("join room %q from phase %q: %w", l.id, l.phase, apperrors.ErrInvalidPhase)
	}
	if l.playerNameTaken(playerName) {
		return nil, fmt.Errorf("player name %q is already taken in room %q", playerName, l.id)
	}
	playerIndex := l.nextOpenPlayerIndex()
	if playerIndex < 0 {
		return nil, fmt.Errorf("room %q is full", l.id)
	}

	sessionID := fmt.Sprintf("%s-player-%d", l.id, playerIndex)
	session := NewClientSession(sessionID, playerIndex, playerName)
	session.MarkSeen(now.UTC())
	l.sessions[sessionID] = session
	l.ensureSessionIndexLocked()[playerIndex] = session
	if err := l.game.setPlayerName(playerIndex, playerName); err != nil {
		return nil, err
	}
	delete(l.readyPlayers, playerIndex)
	return session, nil
}

// SubmitCommand validates the client session before forwarding command input to the game.
func (l *Lobby) SubmitCommand(command CommandEnvelope) error {
	caller := l.Session(command.SessionID)
	if caller == nil {
		return fmt.Errorf("submit command for session %q in room %q: %w", command.SessionID, l.id, apperrors.ErrSessionNotFound)
	}
	return l.game.submitCommandFromSession(command, caller)
}

func (l *Lobby) playerNameTaken(playerName string) bool {
	for _, session := range l.sessions {
		if session != nil && strings.EqualFold(strings.TrimSpace(session.PlayerName), playerName) {
			return true
		}
	}
	return false
}

func (l *Lobby) nextOpenPlayerIndex() int {
	for playerIndex := 0; playerIndex < l.game.PlayerCount(); playerIndex++ {
		if !l.playerJoined(playerIndex) {
			return playerIndex
		}
	}
	return -1
}

func (l *Lobby) joinedPlayerCount() int {
	count := 0
	for playerIndex := 0; playerIndex < l.game.PlayerCount(); playerIndex++ {
		if l.playerJoined(playerIndex) {
			count++
		}
	}
	return count
}

func (l *Lobby) readyPlayerIndicesLocked() []int {
	indices := make([]int, 0, len(l.readyPlayers))
	for playerIndex, ready := range l.readyPlayers {
		if ready && l.playerJoined(playerIndex) {
			indices = append(indices, playerIndex)
		}
	}
	slices.Sort(indices)
	return indices
}

func (l *Lobby) sessionSnapshotsLocked() []ClientSessionSnapshot {
	sessions := make([]ClientSessionSnapshot, 0, len(l.sessions))
	for _, session := range l.sessions {
		if session == nil {
			continue
		}
		sessions = append(sessions, ClientSessionSnapshot{
			ClientSessionID: session.ClientSessionID,
			PlayerIndex:     session.PlayerIndex,
			PlayerName:      session.PlayerName,
			DeviceID:        session.DeviceID,
			LastSeenAt:      session.LastSeenAt,
		})
	}
	slices.SortFunc(sessions, func(a, b ClientSessionSnapshot) int {
		return cmp.Compare(a.PlayerIndex, b.PlayerIndex)
	})
	return sessions
}

func (l *Lobby) playerJoined(playerIndex int) bool {
	return l.sessionForPlayerLocked(playerIndex) != nil
}

func (l *Lobby) sessionForPlayerLocked(playerIndex int) *ClientSession {
	if session := l.sessionsByPlayer[playerIndex]; session != nil && l.sessions[session.ClientSessionID] == session {
		return session
	}
	for _, session := range l.sessions {
		if session != nil && session.PlayerIndex == playerIndex {
			return session
		}
	}
	return nil
}

func (l *Lobby) ensureSessionIndexLocked() map[int]*ClientSession {
	if l.sessionsByPlayer == nil {
		l.sessionsByPlayer = make(map[int]*ClientSession, len(l.sessions))
	}
	return l.sessionsByPlayer
}
