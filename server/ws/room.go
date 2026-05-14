// room.go — the hosted runtime wrapper around one game lobby.
//
// # What a Room does
//
// A Room ties together three concerns:
//   - Lobby / game state  (r.lobby  — the session/app layer)
//   - Live WebSocket clients (r.connections — who is currently connected)
//   - Background turn loop  (startGameLoop — the goroutine that runs turns)
//
// # Concurrency model
//
// Almost every public method on Room funnels through r.do() / r.doBool(),
// which serialize work through r.actor (an approom.Actor — a bounded channel
// queue).  This means the lobby and connection registry are always mutated by
// one goroutine at a time, even when multiple WebSocket handlers fire at once.
//
// The one exception is RunTurn: it starts its own goroutine (resultCh) and
// blocks on a select loop so it can react to prompt events while the engine
// is running.  RunTurn does NOT go through the actor because it is already
// the sole owner of the game loop at that point.
//
// # Room lifecycle
//
//	newRoom → JoinPlayer (×N) → HandleStart → startGameLoop → [RunTurn × turns] → game over / RestartGame / CloseAllClients
//
//nolint:err113,wrapcheck,contextcheck,sloglint,revive,unparam,gocognit,cyclop,errcheck,inamedparam // Legacy websocket room runtime; full context+slog refactor is deferred.
package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	runtimeturn "github.com/axwell0/elmakina/engine/runtime/turn"
	turnmodel "github.com/axwell0/elmakina/engine/turn"
	httpapi "github.com/axwell0/elmakina/internal/api"
	approom "github.com/axwell0/elmakina/internal/app/room"
	sessionapp "github.com/axwell0/elmakina/internal/app/session"
	"github.com/axwell0/elmakina/internal/apperrors"
	"github.com/axwell0/elmakina/internal/store"
)

// Room is the hosted runtime wrapper around one lobby.
//
// Fields:
//   - lobby:       the game session and player list (mutated by the actor)
//   - connections: live WebSocket clients keyed by session/player
//   - actor:       serializes all state mutations through a channel queue
//   - mu/loopRunning: guards the single background turn goroutine
type Room struct {
	lobby       *sessionapp.Lobby
	connections *ConnectionRegistry
	logger      *slog.Logger
	turnConfig  runtimeturn.Config
	actor       *approom.Actor // single-writer serializer for lobby mutations
	stores      store.Bundle

	mu          sync.Mutex // guards loopRunning
	loopRunning bool       // true while the background turn goroutine is alive
}

// LiveClient is the live transport contract a hosted room needs.
type LiveClient interface {
	Session() *sessionapp.ClientSession
	Send(context.Context, any) error
	Close() error
}

func newRoom(lobby *sessionapp.Lobby, cfg runtimeturn.Config, logger *slog.Logger, stores ...store.Bundle) (*Room, error) {
	if logger == nil {
		logger = slog.Default()
	}
	var bundle store.Bundle
	if len(stores) > 0 {
		bundle = stores[0]
	}
	room := &Room{
		lobby:       lobby,
		connections: NewConnectionRegistry(),
		logger:      logger,
		turnConfig:  cfg,
		actor:       approom.NewActor(64),
		stores:      bundle,
	}
	return room, nil
}

func (r *Room) do(ctx context.Context, fn func() error) error {
	return r.actor.Do(ctx, fn)
}

func (r *Room) doBool(ctx context.Context, fn func() bool) bool {
	return r.actor.DoBool(ctx, fn)
}

func (r *Room) closeActor() {
	r.actor.Close()
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
	r.logger.Info("starting hosted session loop", "room_id", r.lobby.ID())

	go func() {
		// When the loop exits, clear the guard so a future start can happen again.
		defer func() {
			r.mu.Lock()
			r.loopRunning = false
			r.mu.Unlock()
			r.logger.Info("stopped hosted session loop", "room_id", r.lobby.ID())
		}()

		for {
			// If the session already has a winner, publish the terminal state and stop looping.
			if winner := r.lobby.Game().Winner(); winner != nil {
				r.lobby.MarkGameOver()
				if err := r.persistRoom(context.Background()); err != nil {
					r.logger.Warn("failed persisting game-over room", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
				}
				if err := r.broadcastRoomSync(); err != nil {
					r.logger.Warn("broadcast room sync failed", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
				}
				if err := r.broadcastGameOver(winner.ID); err != nil {
					r.logger.Warn("broadcast game over failed", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "player_index", winner.ID, "err", err)
				}
				r.logger.Info("game over reached", "room_id", r.lobby.ID(), "winner_index", winner.ID)
				return
			}

			// Run one turn using the room's shared turn configuration.
			if _, err := r.RunTurn(context.Background()); err != nil {
				r.logger.Warn("turn loop failed", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
				if broadcastErr := r.broadcastError(err); broadcastErr != nil {
					r.logger.Warn("broadcast error failed", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", broadcastErr)
				}
				return
			}

			// Re-check for a winner after the turn finishes because the turn may have ended the game.
			if winner := r.lobby.Game().Winner(); winner != nil {
				r.lobby.MarkGameOver()
				if err := r.persistRoom(context.Background()); err != nil {
					r.logger.Warn("failed persisting game-over room", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
				}
				if err := r.broadcastRoomSync(); err != nil {
					r.logger.Warn("broadcast room sync failed", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
				}
				if err := r.broadcastGameOver(winner.ID); err != nil {
					r.logger.Warn("broadcast game over failed", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "player_index", winner.ID, "err", err)
				}
				r.logger.Info("game over reached", "room_id", r.lobby.ID(), "winner_index", winner.ID)
				return
			}

			// Advance the session into the next turn and push the new snapshot to clients.
			r.lobby.MarkInTurn()
			_ = r.lobby.Game().IncrementTurn()
			if err := r.persistRoom(context.Background()); err != nil {
				r.logger.Warn("failed persisting advanced room", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
			}
			if err := r.broadcastRoomSync(); err != nil {
				r.logger.Warn("broadcast room sync failed", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
			}
		}
	}()
}

// readyToRun checks whether every player is connected and the room is in the
// active turn phase.
func (r *Room) readyToRun() bool {
	if r.lobby.Phase() != sessionapp.PhaseInTurn {
		return false
	}
	return r.allPlayersConnected()
}

func (r *Room) readyToStart() bool {
	return r.lobby.Phase() == sessionapp.PhaseLobby && r.lobby.CanStart() && r.allPlayersConnected()
}

func (r *Room) allPlayersConnected() bool {
	for playerIndex := 0; playerIndex < r.lobby.PlayerCount(); playerIndex++ {
		session := r.lobby.SessionForPlayer(playerIndex)
		if session == nil || !r.HasClient(session.ClientSessionID) {
			return false
		}
	}
	return true
}

// handleIncoming routes one websocket message to the correct lobby
// controller method.
func (r *Room) handleIncoming(sessionID string, envelope incomingEnvelope) error {
	// Each incoming websocket envelope is dispatched by its declared message type.
	switch envelope.Type {
	case httpapi.TypeSubmitReady:
		var message SubmitReadyMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		if err := r.HandleReady(sessionID, message); err != nil {
			return err
		}
		// Ready updates are reflected immediately so the lobby view stays current.
		return r.broadcastRoomSync()
	case httpapi.TypeSubmitStart:
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
	case httpapi.TypeCommand:
		var message CommandEnvelopeMessage
		if err := json.Unmarshal(envelope.Raw, &message); err != nil {
			return err
		}
		return r.HandleCommand(sessionID, message)
	default:
		return fmt.Errorf("unsupported message type %q", envelope.Type)
	}
}

// sendRoomSync sends the current room snapshot to one player.
func (r *Room) sendRoomSync(playerIndex int) error {
	// Resolve the target player first so we can fail cleanly if the session vanished.
	session := r.lobby.SessionForPlayer(playerIndex)
	if session == nil {
		return fmt.Errorf("player %d has no session", playerIndex)
	}
	message, err := BuildRoomSyncMessage(r.lobby, playerIndex, r.connections.ConnectedPlayers())
	if err != nil {
		return err
	}
	return r.sendToSession(session.ClientSessionID, message)
}

// broadcastRoomSync sends the current room snapshot to every connected player.
func (r *Room) broadcastRoomSync() error {
	connectedPlayers := r.connections.ConnectedPlayers()
	return r.forEachSession(func(session *sessionapp.ClientSession) error {
		message, err := BuildRoomSyncMessage(r.lobby, session.PlayerIndex, connectedPlayers)
		if err != nil {
			return err
		}
		_, err = r.SendToSessionIfConnected(session.ClientSessionID, message)
		return err
	})
}

// broadcastGameOver sends the final winner payload to every connected player.
func (r *Room) broadcastGameOver(winnerIndex int) error {
	// Build the game-over message once, then fan it out to every live socket.
	message, err := BuildGameOverMessage(r.lobby, winnerIndex)
	if err != nil {
		return err
	}
	return r.broadcastToSessions(message)
}

// broadcastError sends a shared error message to every connected player.
func (r *Room) broadcastError(err error) error {
	// Nil errors are a no-op so callers can forward failures without extra branching.
	if err == nil {
		return nil
	}
	return r.broadcastToSessions(httpapi.ErrorResponse{Error: err.Error()})
}

// Register stores a websocket client in the session registry.
func (r *Room) Register(client LiveClient) error {
	return r.do(context.Background(), func() error {
		return r.registerDirect(client)
	})
}

func (r *Room) registerDirect(client LiveClient) error {
	// Reject nil clients early so we never store broken registry entries.
	if client == nil {
		return fmt.Errorf("client is nil")
	}
	session := client.Session()
	if session == nil {
		return fmt.Errorf("client session is nil")
	}
	stored := r.lobby.Session(session.ClientSessionID)
	if stored == nil {
		return sessionNotFoundError(r.lobby.ID(), session.ClientSessionID)
	}
	stored.MarkSeen(time.Now().UTC())
	if err := r.connections.Attach(stored.ClientSessionID, stored.PlayerIndex, client); err != nil {
		return err
	}
	r.logger.Info("registered websocket session", "room_id", r.lobby.ID(), "session_id", session.ClientSessionID, "player_index", session.PlayerIndex)
	return nil
}

// Unregister removes a session only if this client is still active.
func (r *Room) Unregister(sessionID string, client LiveClient) bool {
	return r.doBool(context.Background(), func() bool {
		return r.unregisterDirect(sessionID, client)
	})
}

func (r *Room) unregisterDirect(sessionID string, client LiveClient) bool {
	session := r.lobby.Session(sessionID)
	if session == nil {
		return false
	}
	if !r.connections.DetachIfCurrent(sessionID, session.PlayerIndex, client) {
		return false
	}
	session.Disconnect(time.Now().UTC())
	r.logger.Info("unregistered websocket client", "room_id", r.lobby.ID(), "session_id", sessionID)
	return true
}

// Client returns the active websocket client for a session ID.
func (r *Room) Client(sessionID string) LiveClient {
	connection, ok := r.connections.Connection(sessionID)
	if !ok {
		return nil
	}
	return connection
}

// HasClient reports whether a session currently has an attached websocket.
func (r *Room) HasClient(sessionID string) bool {
	_, ok := r.connections.Connection(sessionID)
	return ok
}

// IsCurrent reports whether the supplied client is still the active websocket.
func (r *Room) IsCurrent(sessionID string, client LiveClient) bool {
	// A reconnect should only keep sending if it still matches the active registry entry.
	current, ok := r.connections.Connection(sessionID)
	return ok && current == client
}

// sendToSession sends a message to one connected lobby.
func (r *Room) sendToSession(sessionID string, message any) error {
	connection, ok := r.connections.Connection(sessionID)
	if !ok {
		return fmt.Errorf("session %q is not connected", sessionID)
	}
	if err := connection.Send(context.Background(), message); err != nil {
		r.logger.Warn("failed sending websocket message", "room_id", r.lobby.ID(), "session_id", sessionID, "err", err)
		r.unregisterDirect(sessionID, connection)
		if closeErr := connection.Close(); closeErr != nil {
			r.logger.Warn("websocket close failed after send error", "room_id", r.lobby.ID(), "session_id", sessionID, "err", closeErr)
		}
		return err
	}
	return nil
}

// SendToSessionIfConnected sends a message when the session has a live websocket.
func (r *Room) SendToSessionIfConnected(sessionID string, message any) (bool, error) {
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
	session := r.lobby.SessionForPlayer(playerIndex)
	if session == nil {
		return false, nil
	}
	return r.SendToSessionIfConnected(session.ClientSessionID, message)
}

// CloseAllClients disconnects all registered websocket clients and clears the registry.
func (r *Room) CloseAllClients() {
	_ = r.do(context.Background(), func() error {
		r.closeAllClientsDirect()
		return nil
	})
	r.closeActor()
}

func (r *Room) closeAllClientsDirect() {
	// Snapshot the connected client objects first so we can safely disconnect them after registry cleanup.
	type connectedClient struct {
		sessionID   string
		playerIndex int
		client      LiveClient
	}
	clients := make([]connectedClient, 0, len(r.lobby.SessionList()))
	for _, session := range r.lobby.SessionList() {
		if session == nil {
			continue
		}
		connection, ok := r.connections.Connection(session.ClientSessionID)
		if !ok {
			continue
		}
		clients = append(clients, connectedClient{
			sessionID:   session.ClientSessionID,
			playerIndex: session.PlayerIndex,
			client:      connection,
		})
	}
	if err := r.connections.CloseAll(); err != nil {
		r.logger.Warn("failed closing websocket registry", "room_id", r.lobby.ID(), "err", err)
	}
	// Mark each underlying session as disconnected so the room snapshot reflects reality.
	for _, item := range clients {
		if session := item.client.Session(); session != nil {
			session.Disconnect(time.Now().UTC())
		}
	}
	r.logger.Info("closed all websocket clients", "room_id", r.lobby.ID(), "count", len(clients))
}

// RunTurn drives one complete turn end-to-end:
//
//  1. StartTurn  — hands the turn to the engine; the engine opens input channels.
//  2. AwaitTurn  — blocks in a goroutine until the engine finishes.
//  3. select loop — while waiting, the room reacts to two events:
//     a. promptEvents: a new prompt opened → dispatch it to recipients.
//     b. resultCh:           the turn finished → persist + broadcast the result.
//
// The select loop is necessary because the engine may need multiple rounds of
// player input during a single turn (e.g. challenge → card selection).
// Each prompt arrives on promptEvents; a nil on that channel means the
// prompt was closed (engine moved on) so we send a "cleared" notification.
func (r *Room) RunTurn(ctx context.Context) (*turnmodel.TurnResult, error) {
	// Log the turn start before handing control to the session engine.
	r.logger.Info("starting session turn", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber())
	if err := r.lobby.Game().StartTurn(ctx, r.turnConfig); err != nil {
		r.logger.Warn("failed to start session turn", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
		return nil, err
	}

	// Wait for the engine to finish while also reacting to newly opened prompts.
	resultCh := make(chan turnOutcome, 1)
	go func() {
		result, err := r.lobby.Game().AwaitTurn(ctx)
		resultCh <- turnOutcome{result: result, err: err}
	}()
	promptEvents := r.lobby.Game().PromptEvents()
	var lastPrompt *sessionapp.Prompt
	for {
		select {
		case <-ctx.Done():
			// A canceled context means the turn loop should stop immediately.
			r.logger.Warn("session turn canceled by context", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", ctx.Err())
			return nil, ctx.Err()
		case outcome := <-resultCh:
			// When the engine completes, publish the result before returning to the caller.
			if outcome.err != nil {
				r.logger.Warn("session turn ended with error", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", outcome.err)
				return nil, outcome.err
			}
			if err := r.persistTurnResult(ctx, outcome.result); err != nil {
				r.logger.Warn("failed persisting turn result", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
				return nil, err
			}
			if err := r.broadcastTurnResult(outcome.result); err != nil {
				r.logger.Warn("failed broadcasting turn result", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
				return nil, err
			}
			r.logger.Info("session turn completed", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber())
			return outcome.result, nil
		case prompt := <-promptEvents:
			if prompt == nil {
				if err := r.clearPrompt(lastPrompt); err != nil {
					r.logger.Warn("failed clearing closed prompt", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "err", err)
					return nil, err
				}
				lastPrompt = nil
				continue
			}
			current := *prompt
			current.Recipients = slices.Clone(prompt.Recipients)
			lastPrompt = &current
			if err := r.dispatchPrompt(); err != nil {
				r.logger.Warn("failed dispatching prompt", "room_id", r.lobby.ID(), "turn_number", r.lobby.Game().TurnNumber(), "prompt_kind", prompt.Kind, "err", err)
				return nil, err
			}
		}
	}
}

// HandleCommand decodes a browser command and submits it to the session engine.
func (r *Room) HandleCommand(sessionID string, message CommandEnvelopeMessage) error {
	return r.do(context.Background(), func() error {
		return r.handleCommandDirect(sessionID, message)
	})
}

func (r *Room) handleCommandDirect(sessionID string, message CommandEnvelopeMessage) error {
	// Decode the browser payload into the engine command format first.
	command, err := DecodeCommandEnvelope(sessionID, message)
	if err != nil {
		return err
	}
	return r.lobby.SubmitCommand(command)
}

// HandleReady marks a lobby player as ready.
func (r *Room) HandleReady(sessionID string, message SubmitReadyMessage) error {
	return r.do(context.Background(), func() error {
		return r.handleReadyDirect(sessionID, message)
	})
}

func (r *Room) handleReadyDirect(sessionID string, message SubmitReadyMessage) error {
	// Ready messages are room-specific, so reject cross-room submissions first.
	if err := r.validateRoom(message.RoomID); err != nil {
		return err
	}
	// Resolve the websocket session before mutating lobby readiness.
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return err
	}
	if r.lobby.Phase() != sessionapp.PhaseLobby {
		return fmt.Errorf("ready room %q from phase %q: %w", r.lobby.ID(), r.lobby.Phase(), apperrors.ErrInvalidPhase)
	}
	// Mark the player ready in the backing session state.
	if err := r.lobby.MarkReady(session.PlayerIndex, true); err != nil {
		return err
	}
	return r.persistRoom(context.Background())
}

// HandleStart lets the leader start the match.
func (r *Room) HandleStart(sessionID string, message SubmitStartMessage) error {
	return r.do(context.Background(), func() error {
		return r.handleStartDirect(sessionID, message)
	})
}

func (r *Room) handleStartDirect(sessionID string, message SubmitStartMessage) error {
	// Start requests are only valid when they refer to this room.
	if err := r.validateRoom(message.RoomID); err != nil {
		return err
	}
	// Only the leader can transition the lobby into an active match.
	session, err := r.requireLeaderSession(sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("leader session %q is unavailable", sessionID)
	}
	if !r.readyToStart() {
		return fmt.Errorf("start room %q: %w", r.lobby.ID(), apperrors.ErrInvalidPhase)
	}
	// Once the leader is authenticated, hand control to the session engine to begin play.
	if err := r.lobby.StartGame(); err != nil {
		return err
	}
	return r.persistRoom(context.Background())
}

func (r *Room) JoinPlayer(playerName string, now time.Time) (*sessionapp.ClientSession, error) {
	var session *sessionapp.ClientSession
	err := r.do(context.Background(), func() error {
		joined, err := r.lobby.JoinPlayer(playerName, now)
		if err != nil {
			return err
		}
		session = joined
		return nil
	})
	return session, err
}

func (r *Room) RestartGame() error {
	return r.do(context.Background(), func() error {
		if err := r.lobby.RestartGame(); err != nil {
			return err
		}
		return r.persistRoom(context.Background())
	})
}

// dispatchPromptToPlayers fans the current prompt out to the eligible recipients.
func (r *Room) dispatchPrompt() error {
	return r.sendPromptToRelevantPlayers()
}

// SendPromptToPlayer pushes the current prompt to one specific player.
func (r *Room) SendPromptToPlayer(playerIndex int) error {
	prompt := r.lobby.Game().Prompt()
	if prompt == nil || !slices.Contains(prompt.Recipients, playerIndex) {
		return nil
	}

	message, err := BuildPromptEnvelope(prompt)
	if err != nil {
		return err
	}
	_, err = r.SendToPlayerIfConnected(playerIndex, message)
	return err
}

// broadcastTurnResult sends the turn result payload to every connected player.
func (r *Room) broadcastTurnResult(result *turnmodel.TurnResult) error {
	return r.forEachSession(func(session *sessionapp.ClientSession) error {
		message, err := BuildTurnResultMessage(r.lobby, session.PlayerIndex, result)
		if err != nil {
			return err
		}
		_, err = r.SendToSessionIfConnected(session.ClientSessionID, message)
		return err
	})
}

// validateRoom rejects commands that accidentally target a different room.
func (r *Room) validateRoom(roomID string) error {
	if roomID != r.lobby.ID() {
		return fmt.Errorf("room id %q does not match controller room %q", roomID, r.lobby.ID())
	}
	return nil
}

// sessionForID resolves a lobby ID into the live client lobby record.
func (r *Room) sessionForID(sessionID string) (*sessionapp.ClientSession, error) {
	// The websocket message handlers always start by resolving the current session record.
	session := r.lobby.Session(sessionID)
	if session == nil {
		return nil, sessionNotFoundError(r.lobby.ID(), sessionID)
	}
	return session, nil
}

// requireAuthorizedSession validates the request bearer token before allowing the lobby to act.
func (r *Room) requireAuthorizedSession(req *http.Request, sessionID string) (*sessionapp.ClientSession, error) {
	// Authorize the request using the bearer token that was issued as the session resume token.
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return nil, err
	}
	token := bearerToken(req)
	if !session.VerifyResumeToken(token) {
		return nil, unauthorizedError("invalid resume token")
	}
	return session, nil
}

// requireLeaderSession resolves the lobby and checks that it belongs to the lobby leader.
func (r *Room) requireLeaderSession(sessionID string) (*sessionapp.ClientSession, error) {
	// The leader check is a second gate after the session lookup succeeds.
	session, err := r.sessionForID(sessionID)
	if err != nil {
		return nil, err
	}
	if !r.lobby.IsLeader(session.PlayerIndex) {
		return nil, fmt.Errorf("player %d is not the lobby leader: %w", session.PlayerIndex, apperrors.ErrUnauthorizedCommand)
	}
	return session, nil
}

func (r *Room) sendPromptToRelevantPlayers() error {
	prompt := r.lobby.Game().Prompt()
	if prompt == nil {
		return nil
	}
	return r.dispatchPromptToPlayers(prompt)
}

func (r *Room) dispatchPromptToPlayers(prompt *sessionapp.Prompt) error {
	envelope, err := BuildPromptEnvelope(prompt)
	if err != nil {
		return err
	}
	return r.sendToPlayers(prompt.Recipients, envelope)
}

func (r *Room) clearPrompt(prompt *sessionapp.Prompt) error {
	if prompt == nil {
		return nil
	}
	envelope, err := BuildClearedPromptEnvelope(prompt.ID)
	if err != nil {
		return err
	}
	return r.sendToPlayers(prompt.Recipients, envelope)
}

func (r *Room) broadcastToSessions(message any) error {
	return r.forEachSession(func(session *sessionapp.ClientSession) error {
		_, err := r.SendToSessionIfConnected(session.ClientSessionID, message)
		return err
	})
}

func (r *Room) sendToPlayers(players []int, message any) error {
	for _, playerIndex := range players {
		if _, err := r.SendToPlayerIfConnected(playerIndex, message); err != nil {
			return err
		}
	}
	return nil
}

func (r *Room) forEachSession(fn func(*sessionapp.ClientSession) error) error {
	for _, session := range r.lobby.SessionList() {
		if session == nil {
			continue
		}
		if err := fn(session); err != nil {
			return err
		}
	}
	return nil
}

func (r *Room) persistRoom(ctx context.Context) error {
	if r == nil || r.stores.Rooms == nil {
		return nil
	}
	return r.stores.Rooms.SaveRoom(ctx, r.lobby.Snapshot())
}

func (r *Room) persistSession(ctx context.Context, session *sessionapp.ClientSession) error {
	if r == nil || r.stores.Sessions == nil || session == nil {
		return nil
	}
	return r.stores.Sessions.SaveSession(ctx, r.lobby.ID(), session.Snapshot())
}

func (r *Room) persistTurnResult(ctx context.Context, result *turnmodel.TurnResult) error {
	if r == nil {
		return nil
	}
	return r.persistRoom(ctx)
}

// turnOutcome carries either the completed turn result or the error that stopped it.
type turnOutcome struct {
	result *turnmodel.TurnResult
	err    error
}

func roomJoinURL(r *http.Request, roomID string) string {
	// Prefer the forwarded scheme when present so generated URLs match the public edge.
	scheme := "http"
	if r != nil && r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/play?roomId=%s", scheme, r.Host, urlQueryEscape(roomID))
}

// resumableGameSummary builds the browser-friendly summary for one recoverable session.
func resumableGameSummary(r *http.Request, game *sessionapp.Lobby, session *sessionapp.ClientSession, connected bool) httpapi.ResumableGameSnapshot {
	// Re-express the session in the compact shape used by the resume games endpoint.
	return httpapi.ResumableGameSnapshot{
		RoomID:       game.ID(),
		RoomCode:     game.ID(),
		JoinURL:      roomJoinURL(r, game.ID()),
		SessionID:    session.ClientSessionID,
		PlayerName:   session.PlayerName,
		PlayerIndex:  session.PlayerIndex,
		Phase:        string(game.Phase()),
		Connected:    connected,
		CanReconnect: true,
		LastSeenAt:   session.LastSeenAt.UTC().Format(time.RFC3339),
	}
}

// roomSnapshot builds the compact API response for a room listing or lookup.
func roomSnapshot(session *sessionapp.Lobby, connectedPlayers []int) httpapi.RoomSnapshotResponse {
	// This translation keeps the HTTP API decoupled from the internal session types.
	roomView := buildRoomViewMessage(session, connectedPlayers)
	matchSummary := buildMatchSummaryMessage(session)
	return httpapi.RoomSnapshotResponse{
		RoomID:  session.ID(),
		Room:    roomView,
		Summary: matchSummary,
	}
}
