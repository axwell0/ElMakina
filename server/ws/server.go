package ws

// This file owns the HTTP endpoints and websocket room runtime for hosted sessions.
import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	mrand "math/rand/v2"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	sessionapp "example.com/elmakina/app/session"
	"example.com/elmakina/engine/ports"
	"example.com/elmakina/server/httpapi"
	serverlogging "example.com/elmakina/server/logging"
	wstransport "example.com/elmakina/server/ws/transport"
)

//go:embed playground.html
var playgroundFS embed.FS

// Server owns the HTTP routes and the active hosted rooms.
type Server struct {
	roomsMu    sync.RWMutex
	rooms      map[string]*Room
	turnConfig ports.TurnConfig
	logger     *slog.Logger
}

func NewServer() *Server {
	// Start with an empty room registry and a server-side turn configuration.
	return &Server{
		rooms: make(map[string]*Room),
		// for now this turn config is server-side. TODO: make it configurable by the players at the room level.
		turnConfig: ports.TurnConfig{
			TurnTimeout:      60 * time.Second,
			CounterTimeout:   30 * time.Second,
			ChallengeTimeout: 30 * time.Second,
		},
		logger: serverlogging.NewLogger(),
	}
}

// HandleIndex serves the embedded browser playground.
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// The page is embedded so the binary can serve the playground without reading from disk.
	page, err := playgroundFS.ReadFile("playground.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
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
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "deviceId is required"})
		return
	}
	if req.PlayerCount < 2 {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "at least 2 players are required"})
		return
	}

	// Generate a fresh room code before allocating any game state.
	roomID, err := randomRoomCode()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
	}

	// Build the actual game session with a unique PRNG instance.
	gameSession, err := sessionapp.NewGameSession(roomID, leaderName, req.PlayerCount, mrand.New(mrand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()+17))))
	if err != nil {
		s.logger.Warn("failed creating hosted session", "room_id", roomID, "error", err)
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	room, err := newRoom(gameSession, s.turnConfig, s.logger)
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
	leaderSession, err := room.session.JoinPlayer(req.LeaderName, time.Now().UTC())
	if err != nil {
		s.roomsMu.Lock()
		delete(s.rooms, roomID)
		s.roomsMu.Unlock()
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	resumeToken, err := generateResumeToken()
	if err != nil {
		s.roomsMu.Lock()
		delete(s.rooms, roomID)
		s.roomsMu.Unlock()
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	leaderSession.DeviceID = deviceID
	leaderSession.SetResumeToken(resumeToken)
	// This log line records the successful room creation and its configured lobby size.
	s.logger.Info("room created", "room_id", roomID, "players", req.PlayerCount)

	writeJSON(w, http.StatusCreated, httpapi.CreateRoomResponse{
		RoomID:      roomID,
		RoomCode:    roomID,
		JoinURL:     roomJoinURL(r, roomID),
		SessionID:   leaderSession.ID,
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
	sessions := make([]*sessionapp.GameSession, 0, len(s.rooms))
	for _, hosted := range s.rooms {
		if hosted != nil && hosted.session != nil {
			sessions = append(sessions, hosted.session)
		}
	}
	s.roomsMu.RUnlock()
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})
	response := httpapi.ListRoomsResponse{Rooms: make([]httpapi.RoomSnapshotResponse, 0, len(sessions))}
	for _, session := range sessions {
		response.Rooms = append(response.Rooms, roomSnapshot(session))
	}
	writeJSON(w, http.StatusOK, response)
}

// HandleListResumableGames returns resumable sessions for a browser device.
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

	response := httpapi.ListResumableGamesResponse{Games: make([]httpapi.ResumableGameSummary, 0)}
	for _, room := range rooms {
		for _, session := range room.session.SessionList() {
			if session == nil || session.DeviceID != deviceID {
				continue
			}
			response.Games = append(response.Games, resumableGameSummary(r, room.session, session))
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
		writeJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Error: fmt.Sprintf("room %q not found", roomID)})
		return
	}
	writeJSON(w, http.StatusOK, roomSnapshot(room.session))
}

// HandleJoinRoom registers a new player in the next open lobby slot.
func (s *Server) HandleJoinRoom(w http.ResponseWriter, r *http.Request) {
	// Resolve the room before parsing the body so the lookup failure stays cheap.
	roomID := r.PathValue("roomID")
	room, ok := s.getRoom(roomID)
	if !ok {
		writeJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Error: fmt.Sprintf("room %q not found", roomID)})
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
	session, err := room.session.JoinPlayer(req.PlayerName, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	resumeToken, err := generateResumeToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	session.DeviceID = deviceID
	session.SetResumeToken(resumeToken)
	// Log the join so websocket reconnects can be correlated back to the room and player.
	s.logger.Info("session created for room player", "room_id", roomID, "session_id", session.ID, "player_index", session.PlayerIndex)

	writeJSON(w, http.StatusOK, httpapi.JoinRoomResponse{
		RoomID:      roomID,
		RoomCode:    roomID,
		JoinURL:     roomJoinURL(r, roomID),
		SessionID:   session.ID,
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
		writeJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Error: fmt.Sprintf("room %q not found", roomID)})
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
		writeJSON(w, http.StatusForbidden, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	// The leader is the only player allowed to restart the room.
	if !room.session.IsLeader(session.PlayerIndex) {
		writeJSON(w, http.StatusForbidden, httpapi.ErrorResponse{Error: fmt.Sprintf("player %d is not the lobby leader", session.PlayerIndex)})
		return
	}
	if err := room.session.RestartGame(); err != nil {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	// Push a fresh snapshot so connected clients immediately redraw the restarted state.
	_ = room.broadcastRoomSync()
	writeJSON(w, http.StatusOK, roomSnapshot(room.session))
}

// HandleDeleteRoom removes a room and disconnects its websocket clients.
func (s *Server) HandleDeleteRoom(w http.ResponseWriter, r *http.Request) {
	// The room must still exist before we can authorize and delete it.
	roomID := r.PathValue("roomID")
	hosted, ok := s.getRoom(roomID)
	if !ok {
		writeJSON(w, http.StatusNotFound, httpapi.ErrorResponse{Error: fmt.Sprintf("room %q not found", roomID)})
		return
	}

	sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId"))
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: "sessionId is required"})
		return
	}
	session, err := hosted.requireAuthorizedSession(r, sessionID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, httpapi.ErrorResponse{Error: err.Error()})
		return
	}
	// Deletion is intentionally limited to the room leader and only when the session permits it.
	if !hosted.session.IsLeader(session.PlayerIndex) {
		writeJSON(w, http.StatusForbidden, httpapi.ErrorResponse{Error: fmt.Sprintf("player %d is not the lobby leader", session.PlayerIndex)})
		return
	}
	if !hosted.session.CanDelete() {
		writeJSON(w, http.StatusBadRequest, httpapi.ErrorResponse{Error: fmt.Sprintf("room %q cannot be deleted right now", roomID)})
		return
	}

	// Remove the room from the registry before closing sockets so the room disappears atomically.
	s.roomsMu.Lock()
	delete(s.rooms, roomID)
	s.roomsMu.Unlock()
	hosted.CloseAllClients()
	s.logger.Info("room deleted", "room_id", roomID)
	w.WriteHeader(http.StatusNoContent)
}

// HandleWebSocket upgrades the request and streams turn updates to the client.
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
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}
	session := room.session.Session(sessionID)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	// The resume token travels in the websocket subprotocol list so the handshake can authenticate the client.
	wsResumeToken, subprotocol, err := websocketResumeAuth(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if !session.VerifyResumeToken(wsResumeToken) {
		http.Error(w, "invalid resume token", http.StatusUnauthorized)
		return
	}

	// Upgrade the transport while preserving the negotiated subprotocol.
	conn, err := wstransport.UpgradeConnection(w, r, session, subprotocol)
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
		if room.socket.unregisterIfCurrent(sessionID, conn) {
			session.Disconnect(time.Now().UTC())
		}
		_ = conn.Close()
		_ = room.broadcastRoomSync()
		s.logger.Info("websocket disconnected", "room_id", roomID, "session_id", sessionID)
	}()

	if err := room.socket.register(conn); err != nil {
		_ = conn.Close()
		return
	}
	// A successful registration means the session is now visible as connected.
	session.MarkSeen(time.Now().UTC())

	if err := room.sendRoomSync(session.PlayerIndex); err != nil {
		_ = conn.Close()
		return
	}
	if err := room.SendPromptToPlayer(session.PlayerIndex); err != nil {
		_ = conn.Close()
		return
	}
	_ = room.broadcastRoomSync()

	// After setup, the websocket becomes a long-running read loop for client commands.
	for {
		var envelope incomingEnvelope
		if err := conn.Read(connCtx, &envelope); err != nil {
			s.logger.Warn("failed reading websocket message", "room_id", roomID, "session_id", sessionID, "error", err)
			return
		}
		if !room.IsCurrent(sessionID, conn) {
			return
		}
		if err := room.handleIncoming(sessionID, envelope); err != nil {
			s.logger.Warn("failed handling websocket message", "room_id", roomID, "session_id", sessionID, "message_type", envelope.Type, "error", err)
			_ = conn.Send(connCtx, httpapi.ErrorResponse{Error: err.Error()})
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
