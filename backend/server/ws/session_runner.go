package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"ElMakina/backend/server"
	"ElMakina/backend/server/replay"
)

// PlayerDisconnectedError signals that a player has left the game.
//
// This is used when a player's WebSocket connection drops unexpectedly.
// The session runner catches this and initiates the forfeit/reconnect flow.
type PlayerDisconnectedError struct {
	PlayerID string // The player who disconnected
}

func (e PlayerDisconnectedError) Error() string {
	return fmt.Sprintf("player %s disconnected", e.PlayerID)
}

const (
	replayRulesetVersion  = "v1"
	replayProtocolVersion = "v1"
	maxChatMessageLength  = 200
)

// eventRecorder handles replay recording for a game session.
//
// Every significant event in a game (actions, challenges, chat, disconnects)
// is recorded for later replay. This struct coordinates between the game
// session and the replay storage backend.
//
// Recording happens asynchronously—the game doesn't wait for disk writes.
// Events are assigned sequence numbers to maintain ordering.
type eventRecorder struct {
	recorder        replay.Recorder // Backend storage interface
	matchID         string          // Unique match identifier
	lobbyID         string          // Originating lobby
	playerIDs       []string        // Player ID mapping (index -> ID)
	rulesetVersion  string          // Game rules version (for compatibility)
	protocolVersion string          // Protocol version (for compatibility)
	seq             atomic.Int64    // Monotonic event sequence counter
}

func newEventRecorder(session *server.GameSession, recorder replay.Recorder) *eventRecorder {
	playerIDs := []string{}
	matchID := ""
	lobbyID := ""
	if session != nil {
		playerIDs = append(playerIDs, session.PlayerIDs...)
		matchID = session.MatchID
		if matchID == "" {
			matchID = session.LobbyID
		}
		lobbyID = session.LobbyID
	}
	return &eventRecorder{
		recorder:        recorder,
		matchID:         matchID,
		lobbyID:         lobbyID,
		playerIDs:       playerIDs,
		rulesetVersion:  replayRulesetVersion,
		protocolVersion: replayProtocolVersion,
	}
}

func (r *eventRecorder) record(ctx context.Context, eventType string, visibility string, playerID *string, payload any) {
	if r.recorder == nil || r.matchID == "" {
		return
	}
	seq := r.seq.Add(1)
	if err := r.recorder.RecordEvent(ctx, replay.EventInput{
		MatchID:         r.matchID,
		Seq:             seq,
		Type:            eventType,
		Visibility:      visibility,
		PlayerID:        playerID,
		RulesetVersion:  r.rulesetVersion,
		ProtocolVersion: r.protocolVersion,
		Payload:         payload,
	}); err != nil {
		log.Printf("replay record event error: %v", err)
	}
}

func (r *eventRecorder) currentSeq() int64 {
	return r.seq.Load()
}

func (r *eventRecorder) playerIDForIndex(index int) *string {
	if index < 0 || index >= len(r.playerIDs) {
		return nil
	}
	id := r.playerIDs[index]
	return &id
}

func (r *eventRecorder) startMatch(ctx context.Context, session *server.GameSession) {
	if r.recorder == nil || session == nil {
		return
	}
	players := make([]replay.PlayerInfo, 0, len(session.PlayerIDs))
	for i, id := range session.PlayerIDs {
		name := ""
		if session.Game != nil && i < len(session.Game.CurrentState.Players) {
			name = session.Game.CurrentState.Players[i].Name
		}
		avatar := ""
		if i < len(session.PlayerAvatars) {
			avatar = session.PlayerAvatars[i]
		}
		players = append(players, replay.PlayerInfo{
			PlayerID:    id,
			PlayerIndex: i,
			Nick:        name,
			Avatar:      avatar,
		})
	}
	if err := r.recorder.StartMatch(ctx, replay.MatchStartInput{
		MatchID:         r.matchID,
		LobbyID:         session.LobbyID,
		Players:         players,
		RulesetVersion:  r.rulesetVersion,
		ProtocolVersion: r.protocolVersion,
		RNGSeed:         session.RNGSeed,
	}); err != nil {
		log.Printf("replay start match error: %v", err)
	}
	r.record(ctx, "match_start", replay.VisibilityPublic, nil, map[string]any{
		"match_id":         r.matchID,
		"lobby_id":         r.lobbyID,
		"player_ids":       session.PlayerIDs,
		"player_names":     extractPlayerNames(session),
		"ruleset_version":  r.rulesetVersion,
		"protocol_version": r.protocolVersion,
		"rng_seed":         session.RNGSeed,
	})
}

func (r *eventRecorder) snapshot(ctx context.Context, gs *state.GameState) {
	if r.recorder == nil || gs == nil {
		return
	}
	seq := r.currentSeq()
	stateCopy := gs.Snapshot()
	if err := r.recorder.Snapshot(ctx, replay.SnapshotInput{
		MatchID: r.matchID,
		Seq:     seq,
		State:   stateCopy,
	}); err != nil {
		log.Printf("replay snapshot error: %v", err)
	}
}

func (r *eventRecorder) endMatch(ctx context.Context, winnerIndex int, winnerName string) {
	if r.recorder == nil {
		return
	}
	if err := r.recorder.EndMatch(ctx, replay.MatchEndInput{
		MatchID:         r.matchID,
		WinnerIndex:     winnerIndex,
		WinnerName:      winnerName,
		EndedAt:         time.Now(),
		RulesetVersion:  r.rulesetVersion,
		ProtocolVersion: r.protocolVersion,
	}); err != nil {
		log.Printf("replay end match error: %v", err)
	}
}

func extractPlayerNames(session *server.GameSession) []string {
	if session == nil || session.Game == nil {
		return nil
	}
	names := make([]string, 0, len(session.Game.CurrentState.Players))
	for _, player := range session.Game.CurrentState.Players {
		names = append(names, player.Name)
	}
	return names
}

// sessionRunner orchestrates a single game session from start to finish.
//
// Think of this as the "Game Master" at a tabletop session. It:
//   - Manages player connections (who's here, who disconnected, who got kicked)
//   - Coordinates between the WebSocket layer and the game engine
//   - Handles the game loop (RunTurn → process result → next turn)
//   - Manages pausing (real life happens—people need bathroom breaks)
//   - Records everything for replays
//
// LIFECYCLE:
//  1. Created when lobby leader clicks "Start Game"
//  2. Starts goroutines for message handling and game loop
//  3. Runs until game ends (winner determined) or all players disconnect
//  4. Cleanup and session removal
//
// CONCURRENCY MODEL:
//   - messageLoop(): Receives WebSocket messages from all players
//   - run(): The main game loop, calls engine.RunTurn()
//   - Commands flow through cmdCh to serialize access to shared state
//
// This is the central coordination point for an active game.
//
// See: server.go:88 for connection handling.
// See: orchestrator_turn.go:29 for the engine turn flow.
// See: session_provider.go for InputProvider implementation.
type sessionRunner struct {
	session  *server.GameSession // The game state and player mappings
	provider *sessionProvider    // Bridges WebSocket messages to InputProvider interface
	clock    engine.Clock        // Time source (real or fake for testing)
	pause    *pausableClock      // Wrapper that can pause turn timers
	cfg      engine.TurnConfig   // Timeout durations
	grace    time.Duration       // Reconnection grace period
	errCh    chan error          // Fatal errors (disconnects, etc.)
	ctx      context.Context     // Session lifecycle context
	cancel   context.CancelFunc  // Call to end the session
	wg       sync.WaitGroup      // Waits for goroutines to finish
	cmdCh    chan sessionCommand // Serialized command queue
	mu       sync.Mutex          // Protects offline/kicked/paused maps
	offline  map[string]struct{} // Set of disconnected player IDs
	kicked   map[string]struct{} // Set of kicked player IDs
	paused   *pauseState         // Pause state (who paused, when)
	events   *eventRecorder      // Replay recording
}

// sessionCommand is a message sent to the session's command queue.
//
// The sessionRunner processes commands sequentially from cmdCh. This ensures
// thread-safe access to shared state without complex locking.
//
// Types of commands:
//   - WebSocket messages (env): Player actions, chat, etc.
//   - Functions (fn): Internal operations (pause, resume, kick, etc.)
type sessionCommand struct {
	senderIndex int       // Which player sent this (for WebSocket messages)
	env         *Envelope // The WebSocket message (if this is a player command)
	fn          func()    // Function to execute (if this is an internal command)
}

func newSessionRunner(session *server.GameSession, clock engine.Clock, cfg engine.TurnConfig, grace time.Duration, recorder replay.Recorder) *sessionRunner {
	pausedClock := newPausableClock(clock)
	events := newEventRecorder(session, recorder)
	return &sessionRunner{
		session:  session,
		provider: newSessionProvider(session, cfg, pausedClock, events),
		clock:    pausedClock,
		pause:    pausedClock,
		cfg:      cfg,
		grace:    grace,
		errCh:    make(chan error, 4),
		cmdCh:    make(chan sessionCommand, 64),
		offline:  make(map[string]struct{}),
		kicked:   make(map[string]struct{}),
		events:   events,
	}
}

func (r *sessionRunner) Start() {
	if r.ctx == nil {
		r.ctx, r.cancel = context.WithCancel(context.Background())
	}
	if r.events != nil {
		r.events.startMatch(r.ctx, r.session)
		if r.session != nil && r.session.Game != nil {
			r.events.snapshot(r.ctx, &r.session.Game.CurrentState)
		}
	}
	r.wg.Add(2)
	go func() {
		defer r.wg.Done()
		r.messageLoop()
	}()
	go func() {
		defer r.wg.Done()
		r.run()
	}()
}

func (r *sessionRunner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
}

func (r *sessionRunner) EnqueueClientEnvelope(senderIndex int, env Envelope) {
	select {
	case r.cmdCh <- sessionCommand{senderIndex: senderIndex, env: &env}:
	default:
	}
}

func (r *sessionRunner) EnqueueOffline(playerID string) {
	select {
	case r.cmdCh <- sessionCommand{fn: func() { r.markOffline(playerID) }}:
	default:
	}
}

func (r *sessionRunner) EnqueueOnline(playerID string) {
	select {
	case r.cmdCh <- sessionCommand{fn: func() { r.markOnline(playerID) }}:
	default:
	}
}

func (r *sessionRunner) EnqueueForfeit(playerID string) {
	select {
	case r.cmdCh <- sessionCommand{fn: func() { r.Forfeit(playerID) }}:
	default:
	}
}

func (r *sessionRunner) messageLoop() {
	for {
		select {
		case cmd := <-r.cmdCh:
			if cmd.fn != nil {
				cmd.fn()
				continue
			}
			if cmd.env == nil {
				continue
			}
			if cmd.env.RequestID != "" {
				// dispatch() blocks while paused; keep the message loop responsive so it
				// can process resume/unpause commands.
				if r.provider.isPaused() && r.provider.hasPending(cmd.env.RequestID) {
					go r.provider.dispatch(cmd.senderIndex, *cmd.env)
					continue
				}
				if r.provider.dispatch(cmd.senderIndex, *cmd.env) {
					continue
				}
			}
			_ = r.HandleClientMessage(cmd.senderIndex, *cmd.env)
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *sessionRunner) run() {
	defer func() {
		if r.cancel != nil {
			r.cancel()
		}
	}()
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}
		if winner := r.session.Game.CheckGameOver(); winner != nil {
			r.broadcastGameOver(winner)
			return
		}
		r.broadcastGameState()
		if r.events != nil {
			gs := &r.session.Game.CurrentState
			r.events.record(r.ctx, "turn_start", replay.VisibilityPublic, nil, map[string]any{
				"turn_number":         gs.TurnNumber,
				"active_player_index": gs.CurrentPlayerIndex,
			})
			r.events.snapshot(r.ctx, gs)
		}
		turnNumber := r.session.Game.CurrentState.TurnNumber
		prevCounts := snapshotCardCounts(&r.session.Game.CurrentState)
		res, err := r.session.Game.RunTurn(r.ctx, r.provider, r.clock, r.cfg)
		if err != nil {
			if _, ok := err.(PlayerDisconnectedError); ok {
				r.session.Game.IncrementTurn()
				r.broadcastGameState()
				continue
			}
			return
		}
		if r.events != nil && res != nil {
			r.events.record(r.ctx, "turn_resolved", replay.VisibilityPublic, nil, buildTurnResolvedPayload(turnNumber, res))
			for index, entries := range res.PrivateLogs {
				playerID := r.events.playerIDForIndex(index)
				r.events.record(r.ctx, "private_log", replay.VisibilityPrivate, playerID, map[string]any{
					"player_index": index,
					"logs":         entries,
				})
			}
		}
		r.broadcastLogs(turnNumber, res)
		r.broadcastEliminations(prevCounts, turnNumber, nil)
		if winner := r.session.Game.CheckGameOver(); winner != nil {
			r.broadcastGameOver(winner)
			return
		}
		r.session.Game.IncrementTurn()
		if r.events != nil {
			r.events.snapshot(r.ctx, &r.session.Game.CurrentState)
		}
		r.broadcastGameState()
	}
}

func (r *sessionRunner) Forfeit(playerID string) {
	r.forfeit(playerID, "timeout")
}

func (r *sessionRunner) forfeit(playerID string, reason string) {
	if playerID == "" {
		return
	}
	if r.markKicked(playerID) {
		return
	}
	prevCounts := snapshotCardCounts(&r.session.Game.CurrentState)
	_ = r.session.Forfeit(playerID)
	reasons := map[int]string{}
	if index, ok := r.session.IndexByPlayer[playerID]; ok {
		reasons[index] = reason
	}
	r.broadcastEliminations(prevCounts, r.session.Game.CurrentState.TurnNumber, reasons)
	r.finishPause(playerID, reason)
	r.broadcastPlayerKicked(playerID, reason)
	r.notifyError(PlayerDisconnectedError{PlayerID: playerID})
	if winner := r.session.Game.CheckGameOver(); winner != nil {
		r.broadcastGameOver(winner)
	}
}

func (r *sessionRunner) notifyError(err error) {
	r.provider.notifyError(err)
}

func (r *sessionRunner) markOffline(playerID string) {
	if playerID == "" {
		return
	}
	var pausePayload *GamePausedPayload
	r.mu.Lock()
	r.offline[playerID] = struct{}{}
	shouldPause := len(r.offline) > 0
	if shouldPause {
		if r.paused == nil || !r.paused.active {
			r.paused = r.buildPauseState(playerID)
			pausePayload = r.buildPausePayloadLocked()
		} else {
			r.refreshPauseVotesLocked()
			pausePayload = r.buildPausePayloadLocked()
		}
	}
	r.mu.Unlock()
	if shouldPause {
		r.provider.Pause()
		r.pause.Pause()
		r.broadcastTurnTimerPause()
		if pausePayload != nil {
			if r.events != nil {
				r.events.record(context.Background(), "game_paused", replay.VisibilityPublic, nil, pausePayload)
			}
			r.provider.broadcast(Envelope{
				Type:    MsgGamePaused,
				Payload: mustJSON(*pausePayload),
			})
		}
	}
}

func (r *sessionRunner) markOnline(playerID string) {
	if playerID == "" {
		return
	}
	var resumePayload *GameResumedPayload
	var pausePayload *GamePausedPayload
	r.mu.Lock()
	delete(r.offline, playerID)
	if r.paused != nil && r.paused.active && r.paused.pausedPlayerID == playerID {
		if len(r.offline) == 0 {
			resumePayload = r.buildResumePayloadLocked(playerID, "reconnected")
			r.clearPauseLocked()
		} else {
			next := r.anyOfflinePlayerLocked()
			if next != "" {
				r.paused = r.buildPauseState(next)
				pausePayload = r.buildPausePayloadLocked()
			}
		}
	} else if r.paused != nil && r.paused.active {
		r.refreshPauseVotesLocked()
		pausePayload = r.buildPausePayloadLocked()
	}
	shouldResume := len(r.offline) == 0 && (r.paused == nil || !r.paused.active)
	r.mu.Unlock()
	if shouldResume {
		r.provider.Resume()
		r.pause.Resume()
		r.broadcastTurnTimerResume()
		if resumePayload != nil {
			if r.events != nil {
				r.events.record(context.Background(), "game_resumed", replay.VisibilityPublic, nil, resumePayload)
			}
			r.provider.broadcast(Envelope{
				Type:    MsgGameResumed,
				Payload: mustJSON(*resumePayload),
			})
		}
	}
	if pausePayload != nil && !shouldResume {
		if r.events != nil {
			r.events.record(context.Background(), "game_paused", replay.VisibilityPublic, nil, pausePayload)
		}
		r.provider.broadcast(Envelope{
			Type:    MsgGamePaused,
			Payload: mustJSON(*pausePayload),
		})
	}
}

func (r *sessionRunner) HandleClientMessage(senderIndex int, env Envelope) bool {
	if env.Type != MsgKickVote {
		if env.Type != MsgChatMessage {
			return false
		}
	}
	if env.Type == MsgChatMessage {
		r.handleChatMessage(senderIndex, env)
		return true
	}
	var payload KickVotePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return true
	}
	r.handleKickVote(senderIndex, payload)
	return true
}

type chatIncomingPayload struct {
	Text string `json:"text"`
}

func (r *sessionRunner) handleChatMessage(senderIndex int, env Envelope) {
	if r.session == nil || r.session.Game == nil {
		return
	}
	if senderIndex < 0 || senderIndex >= len(r.session.Game.CurrentState.Players) {
		return
	}
	var payload chatIncomingPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}
	text := strings.TrimSpace(payload.Text)
	if text == "" {
		return
	}
	if utf8.RuneCountInString(text) > maxChatMessageLength {
		runes := []rune(text)
		text = string(runes[:maxChatMessageLength])
	}

	sender := r.session.Game.CurrentState.Players[senderIndex]
	msg := ChatMessagePayload{
		ID:          fmt.Sprintf("chat-%d-%d", time.Now().UnixNano(), senderIndex),
		SenderIndex: senderIndex,
		SenderName:  sender.Name,
		Text:        text,
		Timestamp:   time.Now().UnixMilli(),
	}
	r.provider.broadcast(Envelope{Type: MsgChatMessage, Payload: mustJSON(msg)})
}

func (r *sessionRunner) handleKickVote(senderIndex int, payload KickVotePayload) {
	r.mu.Lock()
	if r.paused == nil || !r.paused.active {
		r.mu.Unlock()
		return
	}
	if payload.TargetIndex != r.paused.pausedPlayerIndex {
		r.mu.Unlock()
		return
	}
	if !r.isEligibleVoterLocked(senderIndex) {
		r.mu.Unlock()
		return
	}
	if _, ok := r.paused.kickVotes[senderIndex]; ok {
		r.mu.Unlock()
		return
	}
	r.paused.kickVotes[senderIndex] = struct{}{}
	r.refreshPauseVotesLocked()
	update := r.buildKickVoteUpdateLocked()
	shouldKick := r.shouldKickLocked()
	playerID := r.paused.pausedPlayerID
	r.mu.Unlock()

	if update != nil {
		r.provider.broadcast(Envelope{
			Type:    MsgKickVoteUpdate,
			Payload: mustJSON(*update),
		})
	}
	if shouldKick {
		r.forfeit(playerID, "vote")
	}
}

func (r *sessionRunner) markKicked(playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.kicked[playerID]; ok {
		return true
	}
	r.kicked[playerID] = struct{}{}
	return false
}

func (r *sessionRunner) finishPause(playerID string, reason string) {
	var resumePayload *GameResumedPayload
	var pausePayload *GamePausedPayload
	r.mu.Lock()
	if r.paused != nil && r.paused.active && r.paused.pausedPlayerID == playerID {
		delete(r.offline, playerID)
		if len(r.offline) == 0 {
			resumePayload = r.buildResumePayloadLocked(playerID, reason)
			r.clearPauseLocked()
		} else {
			next := r.anyOfflinePlayerLocked()
			if next != "" {
				r.paused = r.buildPauseState(next)
				pausePayload = r.buildPausePayloadLocked()
			}
		}
	} else {
		delete(r.offline, playerID)
	}
	shouldResume := len(r.offline) == 0 && (r.paused == nil || !r.paused.active)
	r.mu.Unlock()
	if shouldResume {
		r.provider.Resume()
		r.pause.Resume()
		r.broadcastTurnTimerResume()
		if resumePayload != nil {
			if r.events != nil {
				r.events.record(context.Background(), "game_resumed", replay.VisibilityPublic, nil, resumePayload)
			}
			r.provider.broadcast(Envelope{
				Type:    MsgGameResumed,
				Payload: mustJSON(*resumePayload),
			})
		}
	}
	if pausePayload != nil && !shouldResume {
		if r.events != nil {
			r.events.record(context.Background(), "game_paused", replay.VisibilityPublic, nil, pausePayload)
		}
		r.provider.broadcast(Envelope{
			Type:    MsgGamePaused,
			Payload: mustJSON(*pausePayload),
		})
	}
}

func (r *sessionRunner) broadcastPlayerKicked(playerID string, reason string) {
	index, ok := r.session.IndexByPlayer[playerID]
	if !ok {
		return
	}
	r.provider.broadcast(Envelope{
		Type: MsgPlayerKicked,
		Payload: mustJSON(PlayerKickedPayload{
			PlayerIndex: index,
			Reason:      reason,
		}),
	})
	if r.events != nil {
		r.events.record(context.Background(), "player_kicked", replay.VisibilityPublic, nil, map[string]any{
			"player_index": index,
			"reason":       reason,
		})
	}
}

func (r *sessionRunner) broadcastTurnTimerPause() {
	if r.session == nil || r.session.Game == nil {
		return
	}
	turn := r.session.Game.CurrentState.TurnNumber
	actor := r.session.Game.CurrentState.CurrentPlayerIndex
	r.provider.broadcastTurnTimer("pause", actor, turn, r.cfg.TurnTimeout)
}

func (r *sessionRunner) broadcastTurnTimerResume() {
	if r.session == nil || r.session.Game == nil {
		return
	}
	turn := r.session.Game.CurrentState.TurnNumber
	actor := r.session.Game.CurrentState.CurrentPlayerIndex
	r.provider.broadcastTurnTimer("resume", actor, turn, r.cfg.TurnTimeout)
}

func (r *sessionRunner) buildPauseState(playerID string) *pauseState {
	state := &pauseState{
		active:         true,
		pausedPlayerID: playerID,
		kickVotes:      make(map[int]struct{}),
		reason:         "disconnect",
	}
	index, ok := r.session.IndexByPlayer[playerID]
	if ok {
		state.pausedPlayerIndex = index
		if r.session.Game != nil && index >= 0 && index < len(r.session.Game.CurrentState.Players) {
			state.pausedPlayerName = r.session.Game.CurrentState.Players[index].Name
		}
	}
	duration := r.grace
	if duration <= 0 {
		duration = 60 * time.Second
	}
	state.duration = duration
	state.deadline = time.Now().Add(duration)
	r.refreshPauseVotesLocked()
	return state
}

func (r *sessionRunner) buildPausePayload() *GamePausedPayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buildPausePayloadLocked()
}

func (r *sessionRunner) PauseSnapshot() *GamePausedPayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buildPausePayloadLocked()
}

func (r *sessionRunner) buildPausePayloadLocked() *GamePausedPayload {
	if r.paused == nil || !r.paused.active {
		return nil
	}
	eligible := r.eligibleVotersLocked()
	kickVotes := r.kickVotesLocked()
	return &GamePausedPayload{
		PausedByPlayerID: r.paused.pausedPlayerID,
		PausedByIndex:    r.paused.pausedPlayerIndex,
		PausedByName:     r.paused.pausedPlayerName,
		DeadlineMs:       r.paused.deadline.UnixMilli(),
		DurationMs:       r.paused.duration.Milliseconds(),
		PauseReason:      r.paused.reason,
		EligibleVoters:   eligible,
		KickVotes:        kickVotes,
	}
}

func (r *sessionRunner) buildKickVoteUpdateLocked() *KickVoteUpdatePayload {
	if r.paused == nil || !r.paused.active {
		return nil
	}
	return &KickVoteUpdatePayload{
		EligibleVoters: r.eligibleVotersLocked(),
		KickVotes:      r.kickVotesLocked(),
	}
}

func (r *sessionRunner) buildResumePayloadLocked(playerID string, reason string) *GameResumedPayload {
	index := -1
	name := ""
	if playerID != "" {
		if idx, ok := r.session.IndexByPlayer[playerID]; ok {
			index = idx
			if r.session.Game != nil && idx >= 0 && idx < len(r.session.Game.CurrentState.Players) {
				name = r.session.Game.CurrentState.Players[idx].Name
			}
		}
	}
	return &GameResumedPayload{
		ResumedByPlayerID: playerID,
		ResumedByIndex:    index,
		ResumedByName:     name,
		ResumeReason:      reason,
	}
}

func (r *sessionRunner) clearPauseLocked() {
	r.paused = nil
}

func (r *sessionRunner) eligibleVotersLocked() []int {
	if r.session == nil || r.session.Game == nil {
		return nil
	}
	indices := r.provider.playerIndices()
	gs := &r.session.Game.CurrentState
	eligible := make([]int, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= len(gs.Players) {
			continue
		}
		if len(gs.Players[idx].Hand) == 0 {
			continue
		}
		eligible = append(eligible, idx)
	}
	return eligible
}

func (r *sessionRunner) kickVotesLocked() []int {
	if r.paused == nil {
		return nil
	}
	votes := make([]int, 0, len(r.paused.kickVotes))
	for idx := range r.paused.kickVotes {
		votes = append(votes, idx)
	}
	sort.Ints(votes)
	return votes
}

func (r *sessionRunner) refreshPauseVotesLocked() {
	if r.paused == nil || !r.paused.active {
		return
	}
	eligible := r.eligibleVotersLocked()
	eligibleSet := make(map[int]struct{}, len(eligible))
	for _, idx := range eligible {
		eligibleSet[idx] = struct{}{}
	}
	for idx := range r.paused.kickVotes {
		if _, ok := eligibleSet[idx]; !ok {
			delete(r.paused.kickVotes, idx)
		}
	}
}

func (r *sessionRunner) isEligibleVoterLocked(index int) bool {
	for _, eligible := range r.eligibleVotersLocked() {
		if eligible == index {
			return true
		}
	}
	return false
}

func (r *sessionRunner) shouldKickLocked() bool {
	eligible := r.eligibleVotersLocked()
	if len(eligible) == 0 {
		return false
	}
	return len(r.paused.kickVotes) == len(eligible)
}

func (r *sessionRunner) anyOfflinePlayerLocked() string {
	for id := range r.offline {
		return id
	}
	return ""
}

type pauseState struct {
	active            bool
	pausedPlayerID    string
	pausedPlayerIndex int
	pausedPlayerName  string
	deadline          time.Time
	duration          time.Duration
	kickVotes         map[int]struct{}
	reason            string
}

func (r *sessionRunner) broadcastLogs(turn int, res *engine.TurnResult) {
	if res == nil {
		return
	}
	for _, entry := range res.Logs {
		payload := GameLogPayload{
			Turn:    turn,
			Scope:   "public",
			Message: entry,
		}
		r.provider.broadcast(Envelope{
			Type:    MsgGameLog,
			Payload: mustJSON(payload),
		})
	}
	for index, entries := range res.PrivateLogs {
		for _, entry := range entries {
			payload := GameLogPayload{
				Turn:        turn,
				Scope:       "private",
				Message:     entry,
				PlayerIndex: &index,
			}
			_ = r.provider.send(index, Envelope{
				Type:    MsgGameLog,
				Payload: mustJSON(payload),
			})
			if target, role, ok := parseInvestigateLog(entry); ok {
				_ = r.provider.send(index, Envelope{
					Type: MsgInvestigateRes,
					Payload: mustJSON(InvestigateResultPayload{
						TargetName: target,
						Role:       role,
					}),
				})
			}
		}
	}
	// Broadcast card discards for visual feedback
	r.broadcastCardDiscards(turn, res)
	r.broadcastHandState()
}

// broadcastCardDiscards sends structured card_discarded messages for each card lost in challenges
func (r *sessionRunner) broadcastCardDiscards(turn int, res *engine.TurnResult) {
	if res == nil || r.session == nil || r.session.Game == nil {
		return
	}
	gs := &r.session.Game.CurrentState

	// Process challenge results for discards
	for _, cr := range res.ChallengeResults {
		if cr.LostCard == nil || cr.LostPlayerIndex < 0 {
			continue
		}

		player := gs.Players[cr.LostPlayerIndex]
		reason := "challenge_lost"
		isElimination := len(player.Hand) == 0

		r.provider.broadcast(Envelope{
			Type: MsgCardDiscarded,
			Payload: mustJSON(CardDiscardedPayload{
				PlayerIndex:   cr.LostPlayerIndex,
				PlayerName:    player.Name,
				CardRole:      string(cr.LostCard.Role),
				Reason:        reason,
				Turn:          turn,
				IsElimination: isElimination,
			}),
		})
	}
}

func (r *sessionRunner) broadcastGameOver(winner *state.Player) {
	if winner == nil {
		return
	}
	payload := buildGameOverPayload(r.session.Game, winner)
	if r.events != nil {
		r.events.record(context.Background(), "match_end", replay.VisibilityPublic, nil, map[string]any{
			"winner_index": payload.WinnerIndex,
			"winner_name":  payload.WinnerName,
		})
		r.events.snapshot(context.Background(), &r.session.Game.CurrentState)
		r.events.endMatch(context.Background(), payload.WinnerIndex, payload.WinnerName)
	}
	r.provider.broadcast(Envelope{
		Type:    MsgGameOver,
		Payload: mustJSON(payload),
	})
}

func (r *sessionRunner) broadcastGameState() {
	if r.session == nil || r.session.Game == nil {
		return
	}
	r.provider.broadcast(Envelope{
		Type:    MsgGameState,
		Payload: mustJSON(buildGameStatePayload(&r.session.Game.CurrentState, r.session.PlayerAvatars)),
	})
}

func (r *sessionRunner) broadcastHandState() {
	if r.session == nil || r.session.Game == nil {
		return
	}
	gs := &r.session.Game.CurrentState
	for index, player := range gs.Players {
		_ = r.provider.send(index, Envelope{
			Type: MsgHandState,
			Payload: mustJSON(HandStatePayload{
				PlayerIndex: index,
				Hand:        buildHandRoles(player.Hand),
			}),
		})
	}
}

func (r *sessionRunner) broadcastEliminations(prevCounts []int, turn int, reasons map[int]string) {
	if r.session == nil || r.session.Game == nil {
		return
	}
	gs := &r.session.Game.CurrentState
	for i, player := range gs.Players {
		prev := 0
		if i < len(prevCounts) {
			prev = prevCounts[i]
		}
		if prev > 0 && len(player.Hand) == 0 {
			reason := "lost_last_card"
			if reasons != nil {
				if override, ok := reasons[i]; ok {
					reason = override
				}
			}
			r.provider.broadcast(Envelope{
				Type: MsgPlayerEliminated,
				Payload: mustJSON(PlayerEliminatedPayload{
					PlayerIndex: i,
					Reason:      reason,
					Turn:        turn,
				}),
			})
			if r.events != nil {
				r.events.record(context.Background(), "player_eliminated", replay.VisibilityPublic, nil, map[string]any{
					"player_index": i,
					"reason":       reason,
					"turn":         turn,
				})
			}
		}
	}
}

func buildGameStatePayload(gs *state.GameState, avatars []string) GameStatePayload {
	if gs == nil {
		return GameStatePayload{}
	}
	players := make([]PlayerStatePayload, 0, len(gs.Players))
	for i, player := range gs.Players {
		cardCount := len(player.Hand)
		avatar := ""
		if i >= 0 && i < len(avatars) {
			avatar = avatars[i]
		}
		players = append(players, PlayerStatePayload{
			Index:     i,
			Name:      player.Name,
			Coins:     player.Coins,
			CardCount: cardCount,
			Alive:     cardCount > 0,
			Avatar:    avatar,
		})
	}
	return GameStatePayload{
		TurnNumber:        gs.TurnNumber,
		ActivePlayerIndex: gs.CurrentPlayerIndex,
		Players:           players,
	}
}

func buildGameConfigPayload() GameConfigPayload {
	roles := make([]string, 0, len(models.Roles))
	for _, role := range models.Roles {
		roles = append(roles, string(role))
	}
	return GameConfigPayload{Roles: roles}
}

func buildGameOverPayload(game *engine.Game, winner *state.Player) GameOverPayload {
	index := -1
	if game != nil && winner != nil {
		for i := range game.CurrentState.Players {
			if &game.CurrentState.Players[i] == winner {
				index = i
				break
			}
		}
	}
	name := ""
	if winner != nil {
		name = winner.Name
	}
	return GameOverPayload{
		WinnerIndex: index,
		WinnerName:  name,
	}
}

var investigateLogRe = regexp.MustCompile(`^You see (.+) has (.+)$`)

func parseInvestigateLog(entry string) (string, string, bool) {
	match := investigateLogRe.FindStringSubmatch(entry)
	if len(match) != 3 {
		return "", "", false
	}
	return match[1], match[2], true
}

func snapshotCardCounts(gs *state.GameState) []int {
	if gs == nil {
		return nil
	}
	counts := make([]int, len(gs.Players))
	for i, player := range gs.Players {
		counts[i] = len(player.Hand)
	}
	return counts
}

func buildHandRoles(hand []state.Card) []string {
	roles := make([]string, 0, len(hand))
	for _, card := range hand {
		roles = append(roles, string(card.Role))
	}
	return roles
}
