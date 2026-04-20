package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	sessionapp "example.com/elmakina/app/session"
	"example.com/elmakina/engine/ports"
	runtimeturn "example.com/elmakina/engine/runtime/turn"
	"example.com/elmakina/server/httpapi"
)

// Room is the hosted runtime wrapper around one game session.
type Room struct {
	session    *sessionapp.GameSession
	connections *ConnectionRegistry
	logger     *slog.Logger
	turnConfig ports.TurnConfig

	mu          sync.Mutex
	loopRunning bool
}

// newRoom wires a session into one hosted room runtime.
// TODO: remove logger from the param
func newRoom(session *sessionapp.GameSession, cfg ports.TurnConfig, logger *slog.Logger) (*Room, error) {
	// Default to the process logger when callers do not provide one.
	if logger == nil {
		logger = slog.Default()
	}
	return &Room{
		session:    session,
		connections: NewConnectionRegistry(),
		logger:     logger,
		turnConfig: cfg,
	}, nil
}

// startGameLoop starts the turn loop once the room is ready.
func (r *Room) startGameLoop() {
	// Serialize loop startup so only one background runner can own the room at a time.
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loopRunning || !r.readyToRun() {
		return
	}
	r.loopRunning = true
	r.logger.Info("starting hosted session loop", "room_id", r.session.ID)

	go func() {
		// When the loop exits, clear the guard so a future start can happen again.
		defer func() {
			r.mu.Lock()
			r.loopRunning = false
			r.mu.Unlock()
			r.logger.Info("stopped hosted session loop", "room_id", r.session.ID)
		}()

		for {
			// If the session already has a winner, publish the terminal state and stop looping.
			if winner := r.session.Winner(); winner != nil {
				r.session.MarkGameOver()
				_ = r.broadcastRoomSync()
				_ = r.broadcastGameOver(winner.ID)
				r.logger.Info("game over reached", "room_id", r.session.ID, "winner_index", winner.ID)
				return
			}

			// Run one turn using the room's shared turn configuration.
			if _, err := r.RunTurn(context.Background()); err != nil {
				r.logger.Warn("turn loop failed", "room_id", r.session.ID, "error", err)
				_ = r.broadcastError(err)
				return
			}

			// Re-check for a winner after the turn finishes because the turn may have ended the game.
			if winner := r.session.Winner(); winner != nil {
				r.session.MarkGameOver()
				_ = r.broadcastRoomSync()
				_ = r.broadcastGameOver(winner.ID)
				r.logger.Info("game over reached", "room_id", r.session.ID, "winner_index", winner.ID)
				return
			}

			// Advance the session into the next turn and push the new snapshot to clients.
			r.session.MarkInTurn()
			r.session.IncrementTurn()
			_ = r.broadcastRoomSync()
		}
	}()
}

// readyToRun checks whether every player is connected and the room is in the
// active turn phase.
func (r *Room) readyToRun() bool {
	// The game can only spin its turn loop when the session is already in turn phase.
	if r.session.Phase() != sessionapp.PhaseInTurn {
		return false
	}
	// Every player must have an attached websocket before we start advancing turns.
	for playerIndex := range r.session.State.Players {
		session := r.session.SessionForPlayer(playerIndex)
		if session == nil || !r.HasClient(session.ID) {
			return false
		}
	}
	return true
}

// handleIncoming routes one websocket message to the correct session
// controller method.
func (r *Room) handleIncoming(sessionID string, envelope incomingEnvelope) error {
	// Each incoming websocket envelope is dispatched by its declared message type.
	switch envelope.Type {
	case TypeSubmitReady:
		var message SubmitReadyMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		if err := r.HandleReady(sessionID, message); err != nil {
			return err
		}
		// Ready updates are reflected immediately so the lobby view stays current.
		return r.broadcastRoomSync()
	case TypeSubmitStart:
		var message SubmitStartMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		if err := r.HandleStart(sessionID, message); err != nil {
			return err
		}
		// Starting the match changes room state and may unlock the background turn loop.
		if err := r.broadcastRoomSync(); err != nil {
			return err
		}
		r.startGameLoop()
		return nil
	case TypeSubmitAction:
		var message SubmitActionMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		return r.HandleSubmitAction(sessionID, message)
	case TypeCommand:
		var message CommandEnvelopeMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		return r.HandleCommand(sessionID, message)
	case TypeSubmitChallenge:
		var message SubmitChallengeMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		return r.HandleSubmitChallenge(sessionID, message)
	case TypeSubmitCounter:
		var message SubmitCounterMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		return r.HandleSubmitCounter(sessionID, message)
	case TypeSubmitStep:
		var message SubmitStepMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		return r.HandleSubmitStep(sessionID, message)
	default:
		return fmt.Errorf("unsupported message type %q", envelope.Type)
	}
}

// sendRoomSync sends the current room snapshot to one player.
func (r *Room) sendRoomSync(playerIndex int) error {
	// Resolve the target player first so we can fail cleanly if the session vanished.
	session := r.session.SessionForPlayer(playerIndex)
	if session == nil {
		return fmt.Errorf("player %d has no session", playerIndex)
	}
	message, err := BuildRoomSyncMessage(r.session, playerIndex)
	if err != nil {
		return err
	}
	return r.sendToSession(session.ID, message)
}

// broadcastRoomSync sends the current room snapshot to every connected player.
func (r *Room) broadcastRoomSync() error {
	// Walk the logical session list so snapshot generation stays aligned with session order.
	for _, session := range r.session.SessionList() {
		if session == nil {
			continue
		}
		message, err := BuildRoomSyncMessage(r.session, session.PlayerIndex)
		if err != nil {
			return err
		}
		if _, err := r.SendToSessionIfConnected(session.ID, message); err != nil {
			return err
		}
	}
	return nil
}

// broadcastGameOver sends the final winner payload to every connected player.
func (r *Room) broadcastGameOver(winnerIndex int) error {
	// Build the game-over message once, then fan it out to every live socket.
	message, err := BuildGameOverMessage(r.session, winnerIndex)
	if err != nil {
		return err
	}
	for _, session := range r.session.SessionList() {
		if session == nil {
			continue
		}
		if _, err := r.SendToSessionIfConnected(session.ID, message); err != nil {
			return err
		}
	}
	return nil
}

// broadcastError sends a shared error message to every connected player.
func (r *Room) broadcastError(err error) error {
	// Nil errors are a no-op so callers can forward failures without extra branching.
	if err == nil {
		return nil
	}
	for _, session := range r.session.SessionList() {
		if session == nil {
			continue
		}
		if _, sendErr := r.SendToSessionIfConnected(session.ID, httpapi.ErrorResponse{Error: err.Error()}); sendErr != nil {
			return sendErr
		}
	}
	return nil
}

// Register stores a websocket client in the session registry.
func (r *Room) Register(client sessionapp.Client) error {
	// Reject nil clients early so we never store broken registry entries.
	if client == nil {
		return fmt.Errorf("client is nil")
	}
	// Delegate attachment to the backing session so its own invariants stay in one place.
	if err := r.session.AttachClient(client); err != nil {
		return err
	}
	session := client.Session()
	r.connections.Attach(session.ID, session.PlayerIndex, client)
	r.logger.Info("registered websocket session", "room_id", r.session.ID, "session_id", session.ID, "player_index", session.PlayerIndex)
	return nil
}

// UnregisterIfCurrent removes a session only if this client is still active.
func (r *Room) UnregisterIfCurrent(sessionID string, client sessionapp.Client) bool {
	session := r.session.Session(sessionID)
	if session == nil {
		return false
	}
	if !r.connections.DetachIfCurrent(sessionID, session.PlayerIndex, client) {
		return false
	}
	if !r.session.DetachIfCurrent(sessionID, client) {
		return false
	}
	r.logger.Info("unregistered websocket client", "room_id", r.session.ID, "session_id", sessionID)
	return true
}

// Client returns the active websocket client for a session ID.
func (r *Room) Client(sessionID string) sessionapp.Client {
	connection, ok := r.connections.Connection(sessionID)
	if !ok {
		return nil
	}
	client, ok := connection.(sessionapp.Client)
	if !ok {
		return nil
	}
	return client
}

// HasClient reports whether a session currently has an attached websocket.
func (r *Room) HasClient(sessionID string) bool {
	_, ok := r.connections.Connection(sessionID)
	return ok
}

// IsCurrent reports whether the supplied client is still the active websocket.
func (r *Room) IsCurrent(sessionID string, client sessionapp.Client) bool {
	current, ok := r.connections.Connection(sessionID)
	return ok && current == client
}

// sendToSession sends a message to one connected session.
func (r *Room) sendToSession(sessionID string, message any) error {
	// Look up the live client first; disconnected sessions cannot receive messages.
	connection, ok := r.connections.Connection(sessionID)
	if !ok {
		return fmt.Errorf("session %q is not connected", sessionID)
	}
	// Use a background context here because message sending is not tied to an HTTP deadline.
	if err := connection.Send(context.Background(), message); err != nil {
		r.logger.Warn("failed sending websocket message", "room_id", r.session.ID, "session_id", sessionID, "error", err)
		if client, ok := connection.(sessionapp.Client); ok && r.UnregisterIfCurrent(sessionID, client) {
			if session := r.session.Session(sessionID); session != nil {
				session.Disconnect(time.Now().UTC())
			}
		}
		_ = connection.Close()
		return err
	}
	return nil
}

// SendToSessionIfConnected sends a message when the session has a live websocket.
func (r *Room) SendToSessionIfConnected(sessionID string, message any) (bool, error) {
	// This helper lets callers avoid treating a disconnected socket as a hard failure.
	if !r.HasClient(sessionID) {
		return false, nil
	}
	if err := r.sendToSession(sessionID, message); err != nil {
		return false, err
	}
	return true, nil
}

// SendToPlayerIfConnected resolves a player session and sends a message.
func (r *Room) SendToPlayerIfConnected(playerIndex int, message any) (bool, error) {
	// Convert player index to the current session ID before attempting transport.
	session := r.session.SessionForPlayer(playerIndex)
	if session == nil {
		return false, nil
	}
	return r.SendToSessionIfConnected(session.ID, message)
}

// CloseAllClients disconnects all registered websocket clients and clears the registry.
func (r *Room) CloseAllClients() {
	// Snapshot the connected client objects first so we can safely disconnect them after registry cleanup.
	type connectedClient struct {
		sessionID   string
		playerIndex int
		client      sessionapp.Client
	}
	clients := make([]connectedClient, 0, len(r.session.SessionList()))
	for _, session := range r.session.SessionList() {
		if session == nil {
			continue
		}
		connection, ok := r.connections.Connection(session.ID)
		if !ok {
			continue
		}
		client, ok := connection.(sessionapp.Client)
		if !ok {
			continue
		}
		clients = append(clients, connectedClient{
			sessionID:   session.ID,
			playerIndex: session.PlayerIndex,
			client:      client,
		})
	}
	_ = r.session.CloseAllClients()
	// Mark each underlying session as disconnected so the room snapshot reflects reality.
	for _, item := range clients {
		r.connections.DetachIfCurrent(item.sessionID, item.playerIndex, item.client)
		if session := item.client.Session(); session != nil {
			session.Disconnect(time.Now().UTC())
		}
	}
	r.logger.Info("closed all websocket clients", "room_id", r.session.ID, "count", len(clients))
}

// RunTurn starts one turn, waits for it to finish, and pushes prompt updates.
func (r *Room) RunTurn(ctx context.Context) (*runtimeturn.TurnResult, error) {
	// Log the turn start before handing control to the session engine.
	r.logger.Info("starting session turn", "room_id", r.session.ID, "turn_number", r.session.State.TurnNumber)
	if err := r.session.StartTurn(ctx, r.turnConfig); err != nil {
		r.logger.Warn("failed to start session turn", "room_id", r.session.ID, "error", err)
		return nil, err
	}

	// Wait for the engine to finish while also reacting to newly opened prompts.
	resultCh := make(chan turnOutcome, 1)
	go func() {
		result, err := r.session.AwaitTurn(ctx)
		resultCh <- turnOutcome{result: result, err: err}
	}()
	openInputEvents := r.session.OpenInputEvents()
	for {
		select {
		case <-ctx.Done():
			// A canceled context means the turn loop should stop immediately.
			r.logger.Warn("session turn canceled by context", "room_id", r.session.ID, "error", ctx.Err())
			return nil, ctx.Err()
		case outcome := <-resultCh:
			// When the engine completes, publish the result before returning to the caller.
			if outcome.err != nil {
				r.logger.Warn("session turn ended with error", "room_id", r.session.ID, "error", outcome.err)
				return nil, outcome.err
			}
			if err := r.broadcastTurnResult(outcome.result); err != nil {
				r.logger.Warn("failed broadcasting turn result", "room_id", r.session.ID, "error", err)
				return nil, err
			}
			r.logger.Info("session turn completed", "room_id", r.session.ID, "turn_number", r.session.State.TurnNumber)
			return outcome.result, nil
		case input := <-openInputEvents:
			// Newly opened inputs should be pushed out immediately so clients can respond.
			if err := r.dispatchPrompt(); err != nil {
				r.logger.Warn("failed dispatching open input", "room_id", r.session.ID, "input_kind", input.Kind, "error", err)
				return nil, err
			}
		}
	}
}

func (r *Room) HandleSubmitAction(sessionID string, message SubmitActionMessage) error {
	// Validate the room ID before looking at the player's submitted action.
	if err := r.validateRoom(message.RoomID); err != nil {
		return err
	}
	// The websocket must still map to a registered session in this room.
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return err
	}
	openInput, err := r.requireOpenInput(sessionapp.InputMainAction)
	if err != nil {
		return err
	}
	// Only the actor assigned by the current prompt may submit the action.
	if session.PlayerIndex != openInput.Actor {
		return fmt.Errorf("session %q is not allowed to answer main action prompt", sessionID)
	}
	// Decode the payload into the engine's internal action representation.
	action, err := DecodeSubmitAction(message)
	if err != nil {
		return err
	}
	// Prevent spoofed submissions by ensuring the action matches the websocket owner.
	if action.ActorIndex != session.PlayerIndex {
		return fmt.Errorf("action actor %d does not match session player %d", action.ActorIndex, session.PlayerIndex)
	}
	return r.session.SubmitMainAction(sessionID, action)
}

func (r *Room) HandleSubmitChallenge(sessionID string, message SubmitChallengeMessage) error {
	// Challenges are room-scoped, so the room ID must match before we accept the answer.
	if err := r.validateRoom(message.RoomID); err != nil {
		return err
	}
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return err
	}
	openInput, err := r.requireOpenInput(sessionapp.InputChallenge)
	if err != nil {
		return err
	}
	if openInput.Challenge == nil || !containsPlayer(openInput.Challenge.AllowedChallengers, session.PlayerIndex) {
		return fmt.Errorf("session %q is not allowed to answer challenge prompt", sessionID)
	}
	// The challenge response is derived from the submitted payload and then validated against the caller.
	decision := DecodeSubmitChallenge(message)
	if decision.ChallengerIndex != session.PlayerIndex {
		return fmt.Errorf("challenge player %d does not match session player %d", decision.ChallengerIndex, session.PlayerIndex)
	}
	return r.session.SubmitChallengeDecision(sessionID, decision)
}

func (r *Room) HandleSubmitCounter(sessionID string, message SubmitCounterMessage) error {
	// Counter prompts are only valid within the room they were generated for.
	if err := r.validateRoom(message.RoomID); err != nil {
		return err
	}
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return err
	}
	openInput, err := r.requireOpenInput(sessionapp.InputCounter)
	if err != nil {
		return err
	}
	if openInput.Counter == nil || !containsPlayer(openInput.Counter.AllowedPlayers, session.PlayerIndex) {
		return fmt.Errorf("session %q is not allowed to answer counter prompt", sessionID)
	}
	// Decode and then cross-check the response so the payload cannot claim a different player.
	decision, err := DecodeSubmitCounter(message)
	if err != nil {
		return err
	}
	if decision.PlayerIndex != session.PlayerIndex {
		return fmt.Errorf("counter player %d does not match session player %d", decision.PlayerIndex, session.PlayerIndex)
	}
	return r.session.SubmitCounterDecision(sessionID, decision)
}

func (r *Room) HandleSubmitStep(sessionID string, message SubmitStepMessage) error {
	// Step responses are also tied to the owning room, so reject mismatches early.
	if err := r.validateRoom(message.RoomID); err != nil {
		return err
	}
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return err
	}
	openInput, err := r.requireOpenInput(sessionapp.InputStep)
	if err != nil {
		return err
	}
	if openInput.Step == nil || openInput.Actor != session.PlayerIndex {
		return fmt.Errorf("session %q is not allowed to answer step prompt", sessionID)
	}
	// The step decoder needs the live prompt so it can interpret the response correctly.
	response, err := DecodeSubmitStep(openInput.Step, message)
	if err != nil {
		return err
	}
	return r.session.SubmitStepResponse(sessionID, response)
}

func (r *Room) HandleCommand(sessionID string, message CommandEnvelopeMessage) error {
	command, err := DecodeCommandEnvelope(sessionID, message)
	if err != nil {
		return err
	}
	return r.session.SubmitCommand(command)
}

// HandleReady marks a lobby player as ready.
func (r *Room) HandleReady(sessionID string, message SubmitReadyMessage) error {
	// Ready messages are room-specific, so reject cross-room submissions first.
	if err := r.validateRoom(message.RoomID); err != nil {
		return err
	}
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return err
	}
	if r.session.Phase() != sessionapp.PhaseLobby {
		return fmt.Errorf("room %q is not in lobby", r.session.ID)
	}
	// Mark the player ready in the backing session state.
	return r.session.MarkReady(session.PlayerIndex, true)
}

// HandleStart lets the leader start the match.
func (r *Room) HandleStart(sessionID string, message SubmitStartMessage) error {
	// Start requests are only valid when they refer to this room.
	if err := r.validateRoom(message.RoomID); err != nil {
		return err
	}
	session, err := r.requireLeaderSession(sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("leader session %q is unavailable", sessionID)
	}
	// Once the leader is authenticated, hand control to the session engine to begin play.
	return r.session.StartGame()
}

func (r *Room) dispatchPrompt() error {
	// Prompt dispatch is centralized so the loop can trigger it without knowing the details.
	return r.sendPromptToRelevantPlayers()
}

// SendPromptToPlayer pushes the current prompt to one specific player.
func (r *Room) SendPromptToPlayer(playerIndex int) error {
	// Only send a prompt when the current canonical interaction applies to this player.
	interaction := r.session.InteractionState()
	if interaction == nil || !containsPlayer(interaction.Recipients.Players, playerIndex) {
		return nil
	}

	message, err := BuildPromptEnvelope(interaction)
	if err != nil {
		return err
	}
	_, err = r.SendToPlayerIfConnected(playerIndex, message)
	return err
}

func (r *Room) broadcastTurnResult(result *runtimeturn.TurnResult) error {
	// Recompute the turn result message for each player because the payload is player-specific.
	for _, session := range r.session.SessionList() {
		message, err := BuildTurnResultMessage(r.session, session.PlayerIndex, result)
		if err != nil {
			return err
		}
		if _, err := r.SendToSessionIfConnected(session.ID, message); err != nil {
			return err
		}
	}
	return nil
}

func (r *Room) validateRoom(roomID string) error {
	// Most submission handlers guard against cross-room payloads with this simple equality check.
	if roomID != r.session.ID {
		return fmt.Errorf("room id %q does not match controller room %q", roomID, r.session.ID)
	}
	return nil
}

func (r *Room) sessionForID(sessionID string) (*sessionapp.ClientSession, error) {
	// The websocket message handlers always start by resolving the current session record.
	session := r.session.Session(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session %q is not registered in room %q", sessionID, r.session.ID)
	}
	return session, nil
}

func (r *Room) requireAuthorizedSession(req *http.Request, sessionID string) (*sessionapp.ClientSession, error) {
	// Authorize the request using the bearer token that was issued as the session resume token.
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return nil, err
	}
	token := bearerToken(req)
	if !session.VerifyResumeToken(token) {
		return nil, fmt.Errorf("invalid resume token")
	}
	return session, nil
}

func (r *Room) requireLeaderSession(sessionID string) (*sessionapp.ClientSession, error) {
	// The leader check is a second gate after the session lookup succeeds.
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return nil, err
	}
	if !r.session.IsLeader(session.PlayerIndex) {
		return nil, fmt.Errorf("player %d is not the lobby leader", session.PlayerIndex)
	}
	return session, nil
}

func (r *Room) requireOpenInput(kind sessionapp.InputKind) (*sessionapp.OpenInput, error) {
	// Handlers only proceed when the session is currently waiting on the expected prompt type.
	openInput := r.session.OpenInput()
	if openInput == nil {
		return nil, fmt.Errorf("session has no open input")
	}
	if openInput.Kind != kind {
		return nil, fmt.Errorf("session is waiting on %q input, not %q", openInput.Kind, kind)
	}
	return openInput, nil
}

func containsPlayer(players []int, playerIndex int) bool {
	// A linear scan is fine here because the player lists are intentionally small.
	for _, candidate := range players {
		if candidate == playerIndex {
			return true
		}
	}
	return false
}

func (r *Room) sendPromptToRelevantPlayers() error {
	// This helper fans the current canonical interaction out only to allowed recipients.
	interaction := r.session.InteractionState()
	if interaction == nil {
		return nil
	}

	message, err := BuildPromptEnvelope(interaction)
	if err != nil {
		return err
	}
	for _, playerIndex := range interaction.Recipients.Players {
		if _, err := r.SendToPlayerIfConnected(playerIndex, message); err != nil {
			return err
		}
	}
	return nil
}

type turnOutcome struct {
	result *runtimeturn.TurnResult
	err    error
}

func roomJoinURL(r *http.Request, roomID string) string {
	// Prefer the forwarded scheme when present so generated URLs match the public edge.
	scheme := "http"
	if r != nil && r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	return fmt.Sprintf("%s://%s/play?roomId=%s", scheme, r.Host, urlQueryEscape(roomID))
}

func resumableGameSummary(r *http.Request, game *sessionapp.GameSession, session *sessionapp.ClientSession) httpapi.ResumableGameSummary {
	return httpapi.ResumableGameSummary{
		RoomID:       game.ID,
		RoomCode:     game.ID,
		JoinURL:      roomJoinURL(r, game.ID),
		SessionID:    session.ID,
		PlayerName:   session.PlayerName,
		PlayerIndex:  session.PlayerIndex,
		Phase:        string(game.Phase()),
		Connected:    session.Connected,
		CanReconnect: true,
		LastSeenAt:   session.LastSeenAt.UTC().Format(time.RFC3339),
	}
}

// roomSnapshot builds the compact API response for a room listing or lookup.
func roomSnapshot(session *sessionapp.GameSession) httpapi.RoomSnapshotResponse {
	// This translation keeps the HTTP API decoupled from the internal session types.
	roomView := buildRoomViewMessage(session)
	matchSummary := buildMatchSummaryMessage(session)
	players := make([]httpapi.PlayerSummary, 0, len(matchSummary.Players))
	for _, player := range matchSummary.Players {
		players = append(players, httpapi.PlayerSummary{
			PlayerIndex: player.PlayerIndex,
			Name:        player.Name,
			Coins:       player.Coins,
			Cards:       player.Cards,
			Eliminated:  player.Eliminated,
		})
	}
	return httpapi.RoomSnapshotResponse{
		RoomID: session.ID,
		Room: httpapi.RoomView{
			Phase:             roomView.Phase,
			LeaderPlayerIndex: roomView.LeaderPlayerIndex,
			PlayerCount:       roomView.PlayerCount,
			JoinedPlayers:     roomView.JoinedPlayers,
			ConnectedPlayers:  append([]int(nil), roomView.ConnectedPlayers...),
			ReadyPlayers:      append([]int(nil), roomView.ReadyPlayers...),
			CanStart:          roomView.CanStart,
			CanRestart:        roomView.CanRestart,
			CanDelete:         roomView.CanDelete,
		},
		Summary: httpapi.MatchSummary{
			TurnNumber:         matchSummary.TurnNumber,
			CurrentPlayerIndex: matchSummary.CurrentPlayerIndex,
			Players:            players,
		},
	}
}
