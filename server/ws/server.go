package ws

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	coreconfig "github.com/axwell0/elmakina/engine/core/config"
	runtimeturn "github.com/axwell0/elmakina/engine/runtime/turn"
	httpapi "github.com/axwell0/elmakina/internal/api"
	sessionapp "github.com/axwell0/elmakina/internal/app/session"
	"github.com/axwell0/elmakina/internal/apperrors"
	"github.com/axwell0/elmakina/internal/store"
	"github.com/axwell0/elmakina/internal/store/memory"
	wstransport "github.com/axwell0/elmakina/internal/transport/ws"
	serverlogging "github.com/axwell0/elmakina/server/logging"
)

//go:embed playground.html
var playgroundFS embed.FS

// Server owns the HTTP routes and the active hosted rooms.
//
// rooms is the central registry: every created room lives here until it is
// explicitly deleted or the process restarts.  roomsMu guards it because
// multiple HTTP requests can create/delete rooms concurrently.
//
// stores holds the persistence adapters (memory-backed by default).
// They are nil-safe so callers can skip persisting in unit tests.
type Server struct {
	roomsMu    sync.RWMutex       // guards reads/writes to the rooms map
	rooms      map[string]*Room   // roomID → live Room
	turnConfig runtimeturn.Config // default timeouts for turn/counter/challenge windows
	logger     *slog.Logger
	stores     store.Bundle // optional persistence (rooms, sessions, events)
}

// NewServer creates a room registry with the default turn timing and logger.
func NewServer() *Server {
	stores := memory.NewBundle()
	return &Server{
		rooms: make(map[string]*Room),
		turnConfig: runtimeturn.Config{
			TurnTimeout:      60 * time.Second,
			CounterTimeout:   30 * time.Second,
			ChallengeTimeout: 30 * time.Second,
		},
		logger: serverlogging.NewLogger(),
		stores: stores,
	}
}

// HandleIndex serves the embedded browser playground.
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	page, err := playgroundFS.ReadFile("playground.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandleCreateRoom creates a new room, match, and hosted websocket bridge.
func (s *Server) HandleCreateRoom(w http.ResponseWriter, r *http.Request) {
	// Decode the request first so malformed JSON fails before we allocate any room state.
	var req httpapi.CreateRoomRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	leaderName := strings.TrimSpace(req.LeaderName)
	if leaderName == "" {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "leader name is required"})
		return
	}
	// deviceId identifies the browser for "what can I resume?" lookups later.
	// The resume token still does the real authentication work.
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "deviceId is required"})
		return
	}
	if req.PlayerCount < 2 {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "at least 2 players are required"})
		return
	}

	roomID, err := randomRoomCodeFunc()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}

	gameConfig := coreconfig.DefaultGameConfig()
	if req.GameConfig != nil {
		gameConfig, err = completeCreateRoomGameConfig(*req.GameConfig)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
			return
		}
	}
	lobby, err := sessionapp.NewLobbyWithConfig(roomID, leaderName, req.PlayerCount, gameConfig)
	if err != nil {
		s.logger.Warn("failed creating hosted session", "room_id", roomID, "error", err)
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	room, err := newRoom(lobby, turnConfig(gameConfig.Turn), s.logger, s.stores)
	if err != nil {
		s.logger.Error("failed initializing hosted session", "room_id", roomID, "error", err)
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}

	// Claim the room ID in the registry before exposing the room to clients.
	s.roomsMu.Lock()
	if _, exists := s.rooms[roomID]; exists {
		s.roomsMu.Unlock()
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: fmt.Sprintf("room %q already exists", roomID)})
		return
	}
	s.rooms[roomID] = room
	s.roomsMu.Unlock()
	// The leader joins immediately so the room has an owner from the start.
	leaderSession, err := room.JoinPlayer(req.LeaderName, time.Now().UTC())
	if err != nil {
		s.roomsMu.Lock()
		delete(s.rooms, roomID)
		s.roomsMu.Unlock()
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	resumeToken, err := generateResumeTokenFunc()
	if err != nil {
		s.roomsMu.Lock()
		delete(s.rooms, roomID)
		s.roomsMu.Unlock()
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	// Store the resume token on the session so reconnects can authenticate later.
	leaderSession.DeviceID = deviceID
	leaderSession.SetResumeToken(resumeToken)
	if err := room.persistSession(r.Context(), leaderSession); err != nil {
		s.roomsMu.Lock()
		delete(s.rooms, roomID)
		s.roomsMu.Unlock()
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	if err := room.persistRoom(r.Context()); err != nil {
		s.roomsMu.Lock()
		delete(s.rooms, roomID)
		s.roomsMu.Unlock()
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	// This log line records the successful room creation and its configured lobby size.
	s.logger.Info("room created", "room_id", roomID, "players", req.PlayerCount)

	writeJSON(w, http.StatusCreated, httpapi.CreateRoomResponse{
		RoomID:      roomID,
		RoomCode:    roomID,
		JoinURL:     roomJoinURL(r, roomID),
		SessionID:   leaderSession.ClientSessionID,
		ResumeToken: resumeToken,
		PlayerIndex: leaderSession.PlayerIndex,
		PlayerName:  leaderSession.PlayerName,
		Resumed:     false,
	})
}

// HandleListRooms returns a snapshot for every active room.
func (s *Server) HandleListRooms(w http.ResponseWriter, _ *http.Request) {
	// Copy the active sessions while holding the lock, then sort and render after unlocking.
	s.roomsMu.RLock()
	sessions := make([]*sessionapp.Lobby, 0, len(s.rooms))
	for _, hosted := range s.rooms {
		if hosted != nil && hosted.lobby != nil {
			sessions = append(sessions, hosted.lobby)
		}
	}
	s.roomsMu.RUnlock()
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID() < sessions[j].ID()
	})
	response := httpapi.ListRoomsResponse{Rooms: make([]httpapi.RoomSnapshotResponse, 0, len(sessions))}
	for _, session := range sessions {
		if room, ok := s.getRoom(session.ID()); ok {
			response.Rooms = append(response.Rooms, roomSnapshot(session, room.connections.ConnectedPlayers()))
		}
	}
	writeJSON(w, http.StatusOK, response)
}

// HandleListResumableGames returns resumable sessions for a browser device.
// deviceId is only a grouping key for discovery; clients still need the
// per-session resume token to actually reconnect.
func (s *Server) HandleListResumableGames(w http.ResponseWriter, r *http.Request) {
	deviceID := strings.TrimSpace(r.URL.Query().Get("deviceId"))
	if deviceID == "" {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "deviceId is required"})
		return
	}

	s.roomsMu.RLock()
	rooms := make([]*Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		if room != nil {
			rooms = append(rooms, room)
		}
	}
	s.roomsMu.RUnlock()

	response := httpapi.ListResumableGamesResponse{Games: make([]httpapi.ResumableGameSnapshot, 0)}
	for _, room := range rooms {
		for _, session := range room.lobby.SessionList() {
			// Match sessions by browser/device so the UI can offer possible resumes
			// without already knowing the exact session ID up front.
			if session == nil || session.DeviceID != deviceID {
				continue
			}
			response.Games = append(response.Games, resumableGameSummary(r, room.lobby, session, room.HasClient(session.ClientSessionID)))
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleGetRoom returns the snapshot for a single room.
func (s *Server) HandleGetRoom(w http.ResponseWriter, r *http.Request) {
	// Fetch the room first so a missing room becomes a clean 404.
	roomID := r.PathValue("roomID")
	room, ok := s.getRoom(roomID)
	if !ok {
		writeError(w, roomNotFoundError(roomID))
		return
	}
	writeJSON(w, http.StatusOK, roomSnapshot(room.lobby, room.connections.ConnectedPlayers()))
}

// HandleJoinRoom registers a new player in the next open lobby slot.
func (s *Server) HandleJoinRoom(w http.ResponseWriter, r *http.Request) {
	// Resolve the room before parsing the body so the lookup failure stays cheap.
	roomID := r.PathValue("roomID")
	room, ok := s.getRoom(roomID)
	if !ok {
		writeError(w, roomNotFoundError(roomID))
		return
	}

	var req httpapi.JoinRoomRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "deviceId is required"})
		return
	}
	// Only after the join succeeds do we mint a resume token for the new session.
	session, err := room.JoinPlayer(req.PlayerName, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	resumeToken, err := generateResumeTokenFunc()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	// Attach the browser identity before returning so the client can resume the session later.
	session.DeviceID = deviceID
	session.SetResumeToken(resumeToken)
	if err := room.persistSession(r.Context(), session); err != nil {
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	if err := room.persistRoom(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	// Log the join so websocket reconnects can be correlated back to the room and player.
	s.logger.Info("session created for room player", "room_id", roomID, "session_id", session.ClientSessionID, "player_index", session.PlayerIndex)

	writeJSON(w, http.StatusOK, httpapi.JoinRoomResponse{
		RoomID:      roomID,
		RoomCode:    roomID,
		JoinURL:     roomJoinURL(r, roomID),
		SessionID:   session.ClientSessionID,
		ResumeToken: resumeToken,
		PlayerIndex: session.PlayerIndex,
		PlayerName:  session.PlayerName,
		Resumed:     false,
	})
}

// HandleRestartRoom lets the room leader restart a completed room.
func (s *Server) HandleRestartRoom(w http.ResponseWriter, r *http.Request) {
	// Restart only makes sense for an existing room with an authorized session.
	roomID := r.PathValue("roomID")
	room, ok := s.getRoom(roomID)
	if !ok {
		writeError(w, roomNotFoundError(roomID))
		return
	}

	// Query-string values win because they are cheapest to read and most explicit.
	sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId"))
	if sessionID == "" {
		var req httpapi.RoomMutationRequest
		if err := decodeJSON(r.Body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
			return
		}
		sessionID = strings.TrimSpace(req.SessionID)
		if sessionID == "" {
			writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "sessionId is required"})
			return
		}
	}
	session, err := room.requireAuthorizedSession(r, sessionID)
	if err != nil {
		writeError(w, err)
		return
	}
	// The leader is the only player allowed to restart the room.
	if !room.lobby.IsLeader(session.PlayerIndex) {
		writeJSON(w, http.StatusForbidden, httpapi.ErrorResponse{Error: fmt.Sprintf("player %d is not the lobby leader", session.PlayerIndex)})
		return
	}
	if err := room.RestartGame(); err != nil {
		writeError(w, fmt.Errorf("restart room %q: %w", roomID, err))
		return
	}
	// Push a fresh snapshot so connected clients immediately redraw the restarted state.
	if err := room.broadcastRoomSync(); err != nil {
		s.logger.Warn("broadcast room sync failed", "room_id", roomID, "session_id", sessionID, "player_index", session.PlayerIndex, "err", err)
	}
	writeJSON(w, http.StatusOK, roomSnapshot(room.lobby, room.connections.ConnectedPlayers()))
}

// HandleDeleteRoom removes a room and disconnects its websocket clients.
func (s *Server) HandleDeleteRoom(w http.ResponseWriter, r *http.Request) {
	// The room must still exist before we can authorize and delete it.
	roomID := r.PathValue("roomID")
	hosted, ok := s.getRoom(roomID)
	if !ok {
		writeError(w, roomNotFoundError(roomID))
		return
	}

	sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId"))
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "sessionId is required"})
		return
	}
	session, err := hosted.requireAuthorizedSession(r, sessionID)
	if err != nil {
		writeError(w, err)
		return
	}
	// Deletion is intentionally limited to the room leader and only when the session permits it.
	if !hosted.lobby.IsLeader(session.PlayerIndex) {
		writeJSON(w, http.StatusForbidden, httpapi.ErrorResponse{Error: fmt.Sprintf("player %d is not the lobby leader", session.PlayerIndex)})
		return
	}
	if !hosted.lobby.CanDelete() {
		writeError(w, fmt.Errorf("delete room %q: %w", roomID, apperrors.ErrInvalidPhase))
		return
	}

	// Remove the room from the registry before closing sockets so the room disappears atomically.
	s.roomsMu.Lock()
	delete(s.rooms, roomID)
	s.roomsMu.Unlock()
	if err := s.deletePersistedRoom(r.Context(), hosted); err != nil {
		s.logger.Warn("failed deleting persisted room", "room_id", roomID, "error", err)
	}
	hosted.CloseAllClients()
	s.logger.Info("room deleted", "room_id", roomID)
	w.WriteHeader(http.StatusNoContent)
}

// HandleWebSocket upgrades the HTTP request to a WebSocket and runs the
// long-lived read loop for that client.
//
// Authentication flow:
//  1. Validate roomId + sessionId query params.
//  2. Verify the resume token carried in the WebSocket subprotocol header.
//  3. Upgrade the HTTP connection (hijacks the TCP socket).
//  4. Register the live socket in the room's ConnectionRegistry.
//  5. Send an initial room snapshot + any open prompt.
//  6. Enter a blocking read loop: each message is routed through room.handleIncoming.
//
// On disconnect (normal or error) the deferred cleanup runs:
//   - cancel the socket's context
//   - unregister from the room
//   - close the socket
//   - broadcast updated room snapshot so other clients see the player as offline
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Each websocket request must name both the room and the session it is trying to resume.
	roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
	sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId"))
	if roomID == "" || sessionID == "" {
		http.Error(w, "roomId and sessionId are required", http.StatusBadRequest)
		return
	}

	room, ok := s.getRoom(roomID)
	if !ok {
		http.Error(w, roomNotFoundError(roomID).Error(), http.StatusNotFound)
		return
	}
	session := room.lobby.Session(sessionID)
	if session == nil {
		http.Error(w, sessionNotFoundError(roomID, sessionID).Error(), http.StatusNotFound)
		return
	}
	// deviceId is not used here on purpose. Reconnect authorization is based on
	// the exact session ID plus its resume token.
	// The resume token travels in the websocket subprotocol list so the handshake can authenticate the client.
	wsResumeToken, subprotocol, err := websocketResumeAuth(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if !session.VerifyResumeToken(wsResumeToken) {
		http.Error(w, unauthorizedError("invalid resume token").Error(), http.StatusUnauthorized)
		return
	}

	// Upgrade the transport while preserving the negotiated subprotocol.
	connection, err := wstransport.UpgradeConnection(w, r, session, subprotocol)
	if err != nil {
		s.logger.Warn("failed upgrading websocket connection", "room_id", roomID, "session_id", sessionID, "error", err)
		return
	}
	// coder/websocket documents that using r.Context() after Accept can cause
	// surprising behavior because the request has been hijacked. Give the socket
	// its own lifetime context instead.
	connCtx, cancelConn := context.WithCancel(context.Background())
	s.logger.Info("websocket connected", "room_id", roomID, "session_id", sessionID, "player_index", session.PlayerIndex)
	defer func() {
		// Always clean up the connection, unregister the client, and broadcast the latest room state.
		cancelConn()
		room.Unregister(sessionID, connection)
		if err := connection.Close(); err != nil {
			s.logger.Warn("websocket close failed", "room_id", roomID, "session_id", sessionID, "player_index", session.PlayerIndex, "err", err)
		}
		if err := room.broadcastRoomSync(); err != nil {
			s.logger.Warn("broadcast room sync failed", "room_id", roomID, "session_id", sessionID, "player_index", session.PlayerIndex, "err", err)
		}
		s.logger.Info("websocket disconnected", "room_id", roomID, "session_id", sessionID)
	}()

	if err := room.Register(connection); err != nil {
		if closeErr := connection.Close(); closeErr != nil {
			s.logger.Warn("websocket close failed", "room_id", roomID, "session_id", sessionID, "player_index", session.PlayerIndex, "err", closeErr)
		}
		return
	}
	// A successful registration means the session is now visible as connected.
	session.MarkSeen(time.Now().UTC())
	if err := room.persistSession(r.Context(), session); err != nil {
		s.logger.Warn("failed persisting resumed session", "room_id", roomID, "session_id", sessionID, "error", err)
	}

	if err := room.sendRoomSync(session.PlayerIndex); err != nil {
		if closeErr := connection.Close(); closeErr != nil {
			s.logger.Warn("websocket close failed", "room_id", roomID, "session_id", sessionID, "player_index", session.PlayerIndex, "err", closeErr)
		}
		return
	}
	// Send the prompt immediately after the sync snapshot so the browser can render both states in order.
	if err := room.SendPromptToPlayer(session.PlayerIndex); err != nil {
		if closeErr := connection.Close(); closeErr != nil {
			s.logger.Warn("websocket close failed", "room_id", roomID, "session_id", sessionID, "player_index", session.PlayerIndex, "err", closeErr)
		}
		return
	}
	if err := room.broadcastRoomSync(); err != nil {
		s.logger.Warn("broadcast room sync failed", "room_id", roomID, "session_id", sessionID, "player_index", session.PlayerIndex, "err", err)
	}

	// After setup, the websocket becomes a long-running read loop for client commands.
	for {
		var envelope incomingEnvelope
		if err := connection.Read(connCtx, &envelope); err != nil {
			s.logger.Warn("failed reading websocket message", "room_id", roomID, "session_id", sessionID, "error", err)
			return
		}
		if !room.IsCurrent(sessionID, connection) {
			return
		}
		if err := room.handleIncoming(sessionID, envelope); err != nil {
			s.logger.Warn("failed handling websocket message", "room_id", roomID, "session_id", sessionID, "message_type", envelope.Type, "error", err)
			if sendErr := connection.Send(connCtx, httpapi.ErrorResponse{Error: err.Error()}); sendErr != nil {
				s.logger.Warn("websocket error response send failed", "room_id", roomID, "session_id", sessionID, "message_type", envelope.Type, "err", sendErr)
			}
		}
	}
}

// getRoom looks up the room by room ID.
func (s *Server) getRoom(roomID string) (*Room, bool) {
	// Rooms are shared across handlers, so reads are protected by the RW mutex.
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()
	room, ok := s.rooms[roomID]
	return room, ok
}

func (s *Server) deletePersistedRoom(ctx context.Context, room *Room) error {
	if room == nil {
		return nil
	}
	var firstErr error
	if s.stores.Sessions != nil {
		for _, session := range room.lobby.Sessions() {
			if err := s.stores.Sessions.DeleteSession(ctx, room.lobby.ID(), session.ClientSessionID); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	if s.stores.Rooms != nil {
		if err := s.stores.Rooms.DeleteRoom(ctx, room.lobby.ID()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func turnConfig(cfg coreconfig.TurnConfig) runtimeturn.Config {
	return runtimeturn.Config{
		TurnTimeout:      time.Duration(cfg.TurnTimeoutMillis) * time.Millisecond,
		CounterTimeout:   time.Duration(cfg.CounterTimeoutMillis) * time.Millisecond,
		ChallengeTimeout: time.Duration(cfg.ChallengeTimeoutMillis) * time.Millisecond,
	}
}

func completeCreateRoomGameConfig(cfg coreconfig.GameConfig) (coreconfig.GameConfig, error) {
	defaults := coreconfig.DefaultGameConfig()
	if cfg.Turn.TurnTimeoutMillis == 0 {
		cfg.Turn.TurnTimeoutMillis = defaults.Turn.TurnTimeoutMillis
	}
	if cfg.Turn.CounterTimeoutMillis == 0 {
		cfg.Turn.CounterTimeoutMillis = defaults.Turn.CounterTimeoutMillis
	}
	if cfg.Turn.ChallengeTimeoutMillis == 0 {
		cfg.Turn.ChallengeTimeoutMillis = defaults.Turn.ChallengeTimeoutMillis
	}
	if cfg.Setup.DeckCopies == 0 {
		cfg.Setup.DeckCopies = defaults.Setup.DeckCopies
	}
	if cfg.Economy.MaxCoins == 0 {
		cfg.Economy.MaxCoins = defaults.Economy.MaxCoins
	}
	if cfg.Economy.StartingCoins == 0 {
		cfg.Economy.StartingCoins = defaults.Economy.StartingCoins
	}
	if cfg.Economy.MaxCoins < cfg.Economy.StartingCoins {
		return coreconfig.GameConfig{}, fmt.Errorf("maxCoins must be greater than or equal to startingCoins")
	}
	if _, err := sessionapp.NewGameSessionWithConfig("config-check", "A", 2, cfg); err != nil {
		return coreconfig.GameConfig{}, fmt.Errorf("invalid gameConfig: %w", err)
	}
	return cfg, nil
}
