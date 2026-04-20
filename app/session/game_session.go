package session

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"

	corestate "example.com/elmakina/engine/core/state"
	coretypes "example.com/elmakina/engine/core/types"
	ports "example.com/elmakina/engine/ports"
	runtimeturn "example.com/elmakina/engine/runtime/turn"
)

type activeTurn struct {
	// activeTurn ties one running engine invocation to the broker that receives player input.
	provider *InputBroker
	done     chan struct{}
	result   *runtimeturn.TurnResult
	err      error
}

// GameSession is the application aggregate for one hosted match.
type GameSession struct {
	ID       string
	Lobby    Lobby
	Sessions map[string]*ClientSession
	Inputs   *InputBroker
	// Interactions owns the canonical app-facing live interaction state.
	Interactions *InteractionCoordinator
	State    corestate.GameState

	mu                 sync.RWMutex
	clientsBySessionID map[string]Client
	rng                *rand.Rand
	activeTurn         *activeTurn
	openInput          *OpenInput
	openInputEvents    chan OpenInput
	lastTurnResult     *runtimeturn.TurnResult
	lastTurnErr        error
}

// NewGameSession creates a hosted session with initialized state.
func NewGameSession(id string, leaderName string, playerCount int, rng *rand.Rand) (*GameSession, error) {
	if id == "" {
		return nil, fmt.Errorf("session id is required")
	}
	state, err := newGameState(playerCount, rng)
	if err != nil {
		return nil, err
	}
	state.Players[0].Name = leaderName
	return &GameSession{
		ID:                 id,
		Lobby:              NewLobby(playerCount),
		Sessions:           make(map[string]*ClientSession),
		Inputs:             NewInputBroker(),
		Interactions:       NewInteractionCoordinator(),
		State:              state,
		clientsBySessionID: make(map[string]Client),
		rng:                rng,
		openInputEvents:    make(chan OpenInput, 8),
	}, nil
}

// AddSession registers a durable player session, even when it is disconnected.
func (s *GameSession) AddSession(session *ClientSession) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	if session == nil {
		return fmt.Errorf("client session is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.Sessions[session.ID]; exists {
		return fmt.Errorf("session %q already exists", session.ID)
	}
	s.Sessions[session.ID] = session
	return nil
}

// Session returns the stored durable session by ID.
func (s *GameSession) Session(id string) *ClientSession {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Sessions[id]
}

// SessionForPlayer finds the current durable session for a player.
func (s *GameSession) SessionForPlayer(playerIndex int) *ClientSession {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, session := range s.Sessions {
		if session != nil && session.PlayerIndex == playerIndex {
			return session
		}
	}
	return nil
}

// SessionList returns a stable snapshot of registered sessions.
func (s *GameSession) SessionList() []*ClientSession {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions := make([]*ClientSession, 0, len(s.Sessions))
	for _, session := range s.Sessions {
		if session != nil {
			sessions = append(sessions, session)
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].PlayerIndex < sessions[j].PlayerIndex
	})
	return sessions
}

// AttachClient registers a live client and closes replaced player attachments.
func (s *GameSession) AttachClient(client Client) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	if client == nil {
		return fmt.Errorf("client is nil")
	}
	attached := client.Session()
	if attached == nil {
		return fmt.Errorf("client session is nil")
	}

	s.mu.Lock()
	stored := s.Sessions[attached.ID]
	if stored == nil {
		stored = &ClientSession{
			ID:          attached.ID,
			PlayerIndex: attached.PlayerIndex,
			PlayerName:  attached.PlayerName,
		}
		s.Sessions[attached.ID] = stored
	}
	stored.PlayerIndex = attached.PlayerIndex
	stored.PlayerName = attached.PlayerName
	stored.MarkSeen(attached.LastSeenAt)

	replaced := make([]Client, 0, 2)
	for sessionID, current := range s.clientsBySessionID {
		if current == nil {
			delete(s.clientsBySessionID, sessionID)
			continue
		}
		if sessionID == attached.ID {
			if current != client {
				replaced = appendUniqueClient(replaced, current)
			}
			delete(s.clientsBySessionID, sessionID)
			continue
		}
		currentSession := current.Session()
		if currentSession != nil && currentSession.PlayerIndex == attached.PlayerIndex {
			replaced = appendUniqueClient(replaced, current)
			delete(s.clientsBySessionID, sessionID)
		}
	}
	s.clientsBySessionID[attached.ID] = client
	s.mu.Unlock()

	for _, previous := range replaced {
		if previous != nil && previous != client {
			_ = previous.Close()
		}
	}
	return nil
}

// DetachIfCurrent removes a client only when it is still the active mapping.
func (s *GameSession) DetachIfCurrent(sessionID string, client Client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.clientsBySessionID[sessionID]
	if current == nil || current != client {
		return false
	}
	delete(s.clientsBySessionID, sessionID)
	return true
}

// Client returns the active client by session ID.
func (s *GameSession) Client(sessionID string) Client {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clientsBySessionID[sessionID]
}

// ClientForPlayer returns the active client for a player index.
func (s *GameSession) ClientForPlayer(playerIndex int) Client {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, client := range s.clientsBySessionID {
		if client == nil {
			continue
		}
		session := client.Session()
		if session != nil && session.PlayerIndex == playerIndex {
			return client
		}
	}
	return nil
}

// IsCurrent reports whether the provided client is still active for session ID.
func (s *GameSession) IsCurrent(sessionID string, client Client) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clientsBySessionID[sessionID] == client
}

// ConnectedPlayers returns sorted player indices that currently have a client.
func (s *GameSession) ConnectedPlayers() []int {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[int]struct{})
	for _, client := range s.clientsBySessionID {
		if client == nil {
			continue
		}
		session := client.Session()
		if session == nil {
			continue
		}
		seen[session.PlayerIndex] = struct{}{}
	}

	players := make([]int, 0, len(seen))
	for playerIndex := range seen {
		players = append(players, playerIndex)
	}
	sort.Ints(players)
	return players
}

// CloseAllClients closes every currently registered live client and clears them.
func (s *GameSession) CloseAllClients() error {
	if s == nil {
		return nil
	}

	// Snapshot the current clients under lock, then clear the registry before closing them.
	s.mu.Lock()
	clients := make([]Client, 0, len(s.clientsBySessionID))
	for _, client := range s.clientsBySessionID {
		if client != nil {
			clients = append(clients, client)
		}
	}
	s.clientsBySessionID = make(map[string]Client)
	s.mu.Unlock()

	var firstErr error
	for _, client := range clients {
		if err := client.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// MarkReady updates lobby readiness for one player.
func (s *GameSession) MarkReady(playerIndex int, ready bool) error {
	if s == nil {
		return fmt.Errorf("session game is nil")
	}
	if playerIndex < 0 || playerIndex >= len(s.State.Players) {
		return fmt.Errorf("player index %d out of range", playerIndex)
	}
	// Readiness changes only make sense for players that have already claimed a lobby slot.
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.playerJoined(playerIndex) {
		return fmt.Errorf("player %d has not joined this room", playerIndex)
	}
	s.Lobby.MarkReady(playerIndex, ready)
	return nil
}

// CanStart reports whether every player is connected and marked ready.
func (s *GameSession) CanStart() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Starting requires a full lobby, all slots claimed, and every joined player marked ready.
	if s.Lobby.Phase != PhaseLobby {
		return false
	}
	if s.joinedPlayerCount() != s.Lobby.PlayerCount {
		return false
	}
	for playerIndex := range s.State.Players {
		if !s.playerJoined(playerIndex) {
			return false
		}
		session := sessionForPlayerLocked(s.Sessions, playerIndex)
		if session == nil || !session.Connected || !s.Lobby.ReadyPlayers[playerIndex] {
			return false
		}
	}
	return true
}

// StartGame transitions lobby state into active-turn mode when prerequisites hold.
func (s *GameSession) StartGame() error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	if !s.CanStart() {
		return fmt.Errorf("session %q cannot start yet", s.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Lobby.Phase = PhaseInTurn
	clear(s.Lobby.ReadyPlayers)
	return nil
}

// RestartGame creates a fresh game instance while preserving the player roster.
func (s *GameSession) RestartGame() error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}

	// Restarts are only safe when no turn is currently executing.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeTurn != nil {
		return fmt.Errorf("session %q cannot restart during an active turn", s.ID)
	}
	if s.Lobby.Phase != PhaseLobby && s.Lobby.Phase != PhaseGameOver {
		return fmt.Errorf("session %q cannot restart from phase %q", s.ID, s.Lobby.Phase)
	}

	state, err := newGameState(len(s.State.Players), s.rng)
	if err != nil {
		return err
	}
	// Preserve player names while rebuilding the underlying match state.
	for i := range s.State.Players {
		state.Players[i].Name = s.State.Players[i].Name
	}

	s.State = state
	s.Lobby.Phase = PhaseLobby
	clear(s.Lobby.ReadyPlayers)
	s.openInput = nil
	s.lastTurnResult = nil
	s.lastTurnErr = nil
	return nil
}

// StartTurn launches one turn against the live game state.
func (s *GameSession) StartTurn(ctx context.Context, cfg ports.TurnConfig) error {
	if s == nil {
		return fmt.Errorf("session game is nil")
	}

	// A turn can only begin when no other turn is already active.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeTurn != nil {
		return fmt.Errorf("turn already active")
	}

	provider := NewInputBroker()
	turn := &activeTurn{
		provider: provider,
		done:     make(chan struct{}),
	}
	s.activeTurn = turn

	// Track open-input changes on a side goroutine so the session can expose them to the websocket layer.
	go s.watchOpenInput(turn)

	go func() {
		// The turn runner owns the live game state until it finishes or fails.
		defer close(turn.done)
		turn.result, turn.err = runtimeturn.NewService(&s.State).Run(ctx, provider, ports.RealClock{}, cfg)
		s.mu.Lock()
		s.lastTurnResult = turn.result
		s.lastTurnErr = turn.err
		s.activeTurn = nil
		s.openInput = nil
		s.mu.Unlock()
		if interaction := s.InteractionState(); interaction != nil {
			s.Interactions.Clear(interaction.ID)
		}
	}()

	return nil
}

// InteractionState returns the current canonical interaction snapshot.
func (s *GameSession) InteractionState() *InteractionState {
	if s == nil || s.Interactions == nil {
		return nil
	}
	return s.Interactions.Current()
}

// InteractionEvents streams canonical interaction snapshots.
func (s *GameSession) InteractionEvents() <-chan InteractionState {
	if s == nil || s.Interactions == nil {
		return nil
	}
	return s.Interactions.Events()
}

// AwaitTurn waits for the current turn to finish or returns the last result.
func (s *GameSession) AwaitTurn(ctx context.Context) (*runtimeturn.TurnResult, error) {
	s.mu.RLock()
	turn := s.activeTurn
	lastResult := s.lastTurnResult
	lastErr := s.lastTurnErr
	s.mu.RUnlock()

	// If there is no active turn, return the last known result immediately.
	if turn == nil {
		return lastResult, lastErr
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-turn.done:
		return turn.result, turn.err
	}
}

// OpenInput returns the current input snapshot.
func (s *GameSession) OpenInput() *OpenInput {
	// Expose a cloned snapshot so callers can inspect prompt state without racing the turn engine.
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneOpenInput(s.openInput)
}

// OpenInputEvents streams snapshots whenever the active input changes.
func (s *GameSession) OpenInputEvents() <-chan OpenInput {
	if s == nil {
		return nil
	}
	// Callers subscribe to this channel to learn when a new prompt becomes actionable.
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openInputEvents
}

// SubmitMainAction delivers the main action to the running turn.
func (s *GameSession) SubmitMainAction(sessionID string, action *coretypes.PlayerAction) error {
	// Main actions are routed through the active turn broker, not applied directly to state.
	s.mu.RLock()
	turn := s.activeTurn
	s.mu.RUnlock()
	if turn == nil {
		return fmt.Errorf("no active turn")
	}
	return turn.provider.SubmitMainAction(sessionID, action.ActorIndex, action)
}

// SubmitChallengeDecision delivers the challenge response to the running turn.
func (s *GameSession) SubmitChallengeDecision(sessionID string, decision ports.ChallengeDecision) error {
	// Challenge decisions follow the same path: validate the turn is alive, then forward to the broker.
	s.mu.RLock()
	turn := s.activeTurn
	s.mu.RUnlock()
	if turn == nil {
		return fmt.Errorf("no active turn")
	}
	return turn.provider.SubmitChallenge(sessionID, decision.ChallengerIndex, decision)
}

// SubmitCounterDecision delivers the counter response to the running turn.
func (s *GameSession) SubmitCounterDecision(sessionID string, decision ports.CounterDecision) error {
	// Counter responses are also delegated to the active turn's input broker.
	s.mu.RLock()
	turn := s.activeTurn
	s.mu.RUnlock()
	if turn == nil {
		return fmt.Errorf("no active turn")
	}
	return turn.provider.SubmitCounter(sessionID, decision.PlayerIndex, decision)
}

// SubmitStepResponse delivers the follow-up step response to the running turn.
func (s *GameSession) SubmitStepResponse(sessionID string, response corestate.StepResponse) error {
	// Step responses need the current prompt context so the broker can route them correctly.
	s.mu.RLock()
	turn := s.activeTurn
	s.mu.RUnlock()
	if turn == nil {
		return fmt.Errorf("no active turn")
	}
	openInput := s.OpenInput()
	if openInput == nil || openInput.Kind != InputStep {
		return fmt.Errorf("session is not waiting on step input")
	}
	return turn.provider.SubmitStep(sessionID, openInput.Actor, response)
}

// Phase returns the current lifecycle phase.
func (s *GameSession) Phase() Phase {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Lobby.Phase
}

// MarkGameOver moves the session to the terminal phase.
func (s *GameSession) MarkGameOver() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Lobby.Phase = PhaseGameOver
}

// MarkInTurn records that the session is between turns and can continue.
func (s *GameSession) MarkInTurn() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Lobby.Phase = PhaseInTurn
}

// Leader returns the player who owns lobby controls.
func (s *GameSession) Leader() int {
	if s == nil {
		return -1
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Lobby.LeaderPlayerIndex
}

// IsLeader reports whether the player owns the lobby controls.
func (s *GameSession) IsLeader(playerIndex int) bool {
	return s != nil && s.Leader() == playerIndex
}

// CanRestart reports whether the session can be reset safely.
func (s *GameSession) CanRestart() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeTurn == nil && (s.Lobby.Phase == PhaseLobby || s.Lobby.Phase == PhaseGameOver)
}

// CanDelete reports whether the session can be removed safely.
func (s *GameSession) CanDelete() bool {
	return s.CanRestart()
}

// ConnectedPlayerIndices returns a stable list of connected players.
func (s *GameSession) ConnectedPlayerIndices() []int {
	if s == nil {
		return nil
	}
	// This snapshot is used by room views, so keep it stable and sorted.
	s.mu.RLock()
	defer s.mu.RUnlock()
	indices := make([]int, 0, len(s.Sessions))
	for _, session := range s.Sessions {
		if session != nil && session.Connected && s.playerJoined(session.PlayerIndex) {
			indices = append(indices, session.PlayerIndex)
		}
	}
	sort.Ints(indices)
	return indices
}

// ReadyPlayerIndices returns a stable list of ready players.
func (s *GameSession) ReadyPlayerIndices() []int {
	if s == nil {
		return nil
	}
	// Ready players are filtered to joined slots so the UI does not show phantom readiness.
	s.mu.RLock()
	defer s.mu.RUnlock()
	indices := make([]int, 0, len(s.Lobby.ReadyPlayers))
	for playerIndex, ready := range s.Lobby.ReadyPlayers {
		if ready && s.playerJoined(playerIndex) {
			indices = append(indices, playerIndex)
		}
	}
	sort.Ints(indices)
	return indices
}

// PlayerCount returns the configured room capacity.
func (s *GameSession) PlayerCount() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Lobby.PlayerCount
}

// JoinedPlayerCount returns the number of claimed player slots.
func (s *GameSession) JoinedPlayerCount() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.joinedPlayerCount()
}

// JoinPlayer claims the next open lobby slot for the provided player name.
func (s *GameSession) JoinPlayer(playerName string, now time.Time) (*ClientSession, error) {
	if s == nil {
		return nil, fmt.Errorf("session game is nil")
	}
	// Normalize the requested display name before using it as part of uniqueness checks.
	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return nil, fmt.Errorf("player name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Lobby.Phase != PhaseLobby {
		return nil, fmt.Errorf("room %q is not joinable right now", s.ID)
	}
	if s.playerNameTaken(playerName) {
		return nil, fmt.Errorf("player name %q is already taken in room %q", playerName, s.ID)
	}
	playerIndex := s.nextOpenPlayerIndex()
	if playerIndex < 0 {
		return nil, fmt.Errorf("room %q is full", s.ID)
	}

	sessionID := fmt.Sprintf("%s-player-%d", s.ID, playerIndex)
	session := NewClientSession(sessionID, playerIndex, playerName)
	// New joins begin disconnected; the websocket layer later marks them live.
	session.Disconnect(now.UTC())
	s.Sessions[sessionID] = session
	s.State.Players[playerIndex].Name = playerName
	delete(s.Lobby.ReadyPlayers, playerIndex)
	return session, nil
}

func (s *GameSession) playerNameTaken(playerName string) bool {
	// Compare names case-insensitively so the lobby does not accept visually duplicate names.
	for _, session := range s.Sessions {
		if session != nil && strings.EqualFold(strings.TrimSpace(session.PlayerName), playerName) {
			return true
		}
	}
	return false
}

func (s *GameSession) nextOpenPlayerIndex() int {
	// Scan the player array in order so the next join always fills the lowest empty slot.
	for playerIndex := range s.State.Players {
		if !s.playerJoined(playerIndex) {
			return playerIndex
		}
	}
	return -1
}

func (s *GameSession) joinedPlayerCount() int {
	// Count the number of slots that are already claimed by a durable session.
	count := 0
	for playerIndex := range s.State.Players {
		if s.playerJoined(playerIndex) {
			count++
		}
	}
	return count
}

func (s *GameSession) playerJoined(playerIndex int) bool {
	// A player is joined when there is a durable session record for their slot.
	return sessionForPlayerLocked(s.Sessions, playerIndex) != nil
}

// Winner returns the surviving player when the match is over.
func (s *GameSession) Winner() *corestate.Player {
	if s == nil {
		return nil
	}
	// The winner is the last player who still has cards in hand.
	activePlayers := 0
	var lastPlayer *corestate.Player
	for i := range s.State.Players {
		if len(s.State.Players[i].Hand) > 0 {
			activePlayers++
			lastPlayer = &s.State.Players[i]
		}
	}
	if activePlayers == 1 {
		return lastPlayer
	}
	return nil
}

// IncrementTurn advances the turn counter and skips eliminated players.
func (s *GameSession) IncrementTurn() {
	if s == nil || len(s.State.Players) == 0 {
		return
	}

	// Advance until the next player with a remaining hand is found, or wrap around once.
	start := s.State.CurrentPlayerIndex
	for {
		s.State.CurrentPlayerIndex = (s.State.CurrentPlayerIndex + 1) % len(s.State.Players)
		if len(s.State.Players[s.State.CurrentPlayerIndex].Hand) > 0 {
			break
		}
		if s.State.CurrentPlayerIndex == start {
			break
		}
	}

	s.State.TurnNumber++
}

func (s *GameSession) watchOpenInput(turn *activeTurn) {
	if s == nil || turn == nil || turn.provider == nil {
		return
	}
	// Mirror every broker event into the session snapshot so websocket clients can react to it.
	for {
		select {
		case <-turn.done:
			return
		case input := <-turn.provider.Events():
			s.setOpenInput(input)
			interaction, err := interactionFromOpenInput(s, input)
			if err != nil {
				continue
			}
			s.Interactions.Open(interaction)
		}
	}
}

func (s *GameSession) setOpenInput(input OpenInput) {
	// Store a cloned copy under lock, then publish the snapshot to subscribers outside the lock.
	s.mu.Lock()
	s.openInput = cloneOpenInput(&input)
	snapshot := cloneOpenInput(s.openInput)
	events := s.openInputEvents
	s.mu.Unlock()
	publishOpenInputEvent(events, snapshot)
}

func sessionForPlayerLocked(sessions map[string]*ClientSession, playerIndex int) *ClientSession {
	// This helper is the canonical "find session for player" lookup under the session mutex.
	for _, session := range sessions {
		if session != nil && session.PlayerIndex == playerIndex {
			return session
		}
	}
	return nil
}

func publishOpenInputEvent(events chan OpenInput, input *OpenInput) {
	if events == nil || input == nil {
		return
	}
	// Keep the channel responsive by dropping the oldest snapshot if the buffer is already full.
	snapshot := *input
	select {
	case events <- snapshot:
	default:
		select {
		case <-events:
		default:
		}
		select {
		case events <- snapshot:
		default:
		}
	}
}

func appendUniqueClient(clients []Client, candidate Client) []Client {
	// Replacement tracking uses this helper so we only close each displaced client once.
	for _, existing := range clients {
		if existing == candidate {
			return clients
		}
	}
	return append(clients, candidate)
}

func newGameState(playerCount int, rng *rand.Rand) (corestate.GameState, error) {
	if playerCount < 2 {
		return corestate.GameState{}, fmt.Errorf("at least 2 players required")
	}
	if playerCount > 9 {
		return corestate.GameState{}, fmt.Errorf("max 9 players allowed")
	}

	// Smaller tables get a richer opening hand so the game starts with enough action.
	cardsPerPlayer := 2
	if playerCount <= 4 {
		cardsPerPlayer = 3
	}
	deck := corestate.NewDeck(3, rng)

	// Seed every slot with a placeholder player record so lobby snapshots are immediately readable.
	players := make([]corestate.Player, playerCount)
	for i := range players {
		players[i] = corestate.Player{
			ID:    i,
			Name:  placeholderPlayerName(i),
			Hand:  deck.Draw(cardsPerPlayer),
			Coins: 2,
		}
	}

	return corestate.GameState{
		TurnNumber:         1,
		Players:            players,
		Deck:               deck,
		CurrentPlayerIndex: 0,
	}, nil
}

// placeholderPlayerName gives unclaimed lobby slots a readable label before players join.
func placeholderPlayerName(index int) string {
	return fmt.Sprintf("Open slot %d", index+1)
}
