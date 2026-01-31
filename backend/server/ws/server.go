package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"ElMakina/backend/domain/entities"
	"ElMakina/backend/engine"
	"ElMakina/backend/server"
	"ElMakina/backend/server/replay"

	gws "github.com/gorilla/websocket"
)

// Server hosts lobby commands and runs game sessions over WebSocket connections.
//
// Intent:
//
//	Provide a single entrypoint for multi-lobby play while keeping the engine
//	logic isolated and testable. The server owns lobby state, player identity,
//	and session lifecycles.
//
// Safety:
//   - Each connection is bound to one player.
//   - Disconnects forfeit the player and abort the current turn.
type Server struct {
	upgrader       gws.Upgrader
	manager        *server.LobbyManager
	clock          engine.Clock
	cfg            engine.TurnConfig
	start          func(*sessionRunner)
	grace          time.Duration
	forfeit        *ForfeitScheduler
	recorder       replay.Recorder
	allowedOrigins map[string]struct{}

	mu       sync.Mutex
	clients  map[string]*clientConn
	sessions map[string]*sessionRunner
}

type wsConn interface {
	ReadJSON(v any) error
	WriteJSON(v any) error
	Close() error
}

type clientConn struct {
	playerID     entities.PlayerID
	playerNick   string
	playerAvatar string
	conn         wsConn
	sendMu       sync.Mutex
	lobbyID      entities.LobbyID
	session      *sessionRunner
	playerIndex  int
}

// NewServer constructs a lobby server with the given allowed origins.
// If allowedOrigins is empty, all origins are permitted (development mode).
func NewServer(manager *server.LobbyManager, clock engine.Clock, cfg engine.TurnConfig, grace time.Duration, allowedOrigins []string) *Server {
	if manager == nil {
		panic("LobbyManager is required")
	}
	if clock == nil {
		clock = engine.RealClock{}
	}
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}
	return &Server{
		upgrader: gws.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				if len(originSet) == 0 {
					return true
				}
				origin := r.Header.Get("Origin")
				_, ok := originSet[origin]
				return ok
			},
		},
		manager:        manager,
		clock:          clock,
		cfg:            cfg,
		grace:          grace,
		start:          func(r *sessionRunner) { r.Start() },
		clients:        make(map[string]*clientConn),
		sessions:       make(map[string]*sessionRunner),
		forfeit:        NewForfeitScheduler(clock, grace),
		allowedOrigins: originSet,
	}
}

// SetRecorder attaches a match recorder for replay logging.
func (s *Server) SetRecorder(recorder replay.Recorder) {
	s.recorder = recorder
}

// ServeHTTP upgrades to WebSocket and starts the lobby/game loop for a client.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}

	var hello Envelope
	if err := conn.ReadJSON(&hello); err != nil {
		log.Printf("hello read failed: %v", err)
		_ = conn.Close()
		return
	}
	if hello.Type != MsgHello {
		log.Printf("unexpected first message: %s", hello.Type)
		_ = conn.Close()
		return
	}
	var payload HelloPayload
	if err := json.Unmarshal(hello.Payload, &payload); err != nil {
		log.Printf("hello payload decode failed: %v", err)
		_ = s.sendRaw(conn, Envelope{
			Type:    MsgHelloError,
			Payload: mustJSON(HelloErrorPayload{Error: "invalid hello payload"}),
		})
		_ = conn.Close()
		return
	}
	var player *entities.Player
	if payload.ReconnectToken != "" {
		player, err = s.manager.ReconnectPlayer(ctx, payload.ReconnectToken)
		if err != nil {
			log.Printf("reconnect failed: %v", err)
			_ = s.sendRaw(conn, Envelope{
				Type:    MsgHelloError,
				Payload: mustJSON(HelloErrorPayload{Error: err.Error()}),
			})
			_ = conn.Close()
			return
		}
	} else {
		player, err = s.manager.RegisterPlayer(ctx, payload.Nickname)
		if err != nil {
			log.Printf("register failed: %v", err)
			_ = s.sendRaw(conn, Envelope{
				Type:    MsgHelloError,
				Payload: mustJSON(HelloErrorPayload{Error: err.Error()}),
			})
			_ = conn.Close()
			return
		}
	}
	if payload.Avatar != "" {
		if err := s.manager.SetPlayerAvatar(ctx, player.ID(), payload.Avatar); err == nil {
			player.SetAvatar(payload.Avatar)
		}
	}

	client := &clientConn{
		playerID:     player.ID(),
		playerNick:   player.Nick(),
		playerAvatar: player.Avatar(),
		conn:         conn,
	}
	log.Printf("client connected: %s (%s)", player.Nick(), player.ID())
	s.registerClient(client)
	_ = s.send(client, Envelope{
		Type:    MsgHelloAck,
		Payload: mustJSON(HelloAckPayload{PlayerID: player.ID().String(), Token: player.Token()}),
	})
	_ = s.send(client, Envelope{
		Type:    MsgGameConfig,
		Payload: mustJSON(buildGameConfigPayload()),
	})
	s.broadcastLobbyList(ctx)
	s.forfeit.Cancel(player.ID().String())
	s.reattachToLobby(client)
	go s.readLoop(client)
}

func (s *Server) registerClient(client *clientConn) {
	var prior *clientConn
	s.mu.Lock()
	prior = s.clients[client.playerID.String()]
	s.clients[client.playerID.String()] = client
	s.mu.Unlock()
	if prior != nil && prior != client {
		_ = prior.conn.Close()
	}
}

func (s *Server) unregisterClient(client *clientConn, err error) {
	ctx := context.Background()
	s.mu.Lock()
	current := s.clients[client.playerID.String()]
	if current != nil && current != client {
		s.mu.Unlock()
		return
	}
	delete(s.clients, client.playerID.String())
	s.mu.Unlock()

	if client.session != nil {
		client.session.provider.detach(client.playerIndex, client)
		client.session.EnqueueOffline(client.playerID.String())
		s.forfeit.Schedule(client.playerID.String(), func() { client.session.EnqueueForfeit(client.playerID.String()) })
		return
	}
	if client.lobbyID != "" {
		shouldClose, _ := s.manager.LeaveLobby(ctx, client.lobbyID, client.playerID)
		_ = shouldClose
		s.broadcastLobbyState(ctx, client.lobbyID.String())
		s.broadcastLobbyList(ctx)
		log.Printf("client left lobby: %s (%s)", client.playerNick, client.playerID)
		return
	}
	// Keep player registration to preserve reconnect token across restarts.
	log.Printf("client disconnected: %s (%s)", client.playerNick, client.playerID)
	s.broadcastLobbyList(ctx)
}

func (s *Server) readLoop(client *clientConn) {
	ctx := context.Background()
	for {
		var env Envelope
		if err := client.conn.ReadJSON(&env); err != nil {
			_ = client.conn.Close()
			s.unregisterClient(client, err)
			return
		}
		if client.session != nil {
			client.session.EnqueueClientEnvelope(client.playerIndex, env)
			continue
		}
		if err := s.handleLobbyMessage(ctx, client, env); err != nil {
			_ = s.send(client, Envelope{Type: MsgLobbyError, RequestID: env.RequestID, Payload: mustJSON(map[string]string{"error": err.Error()})})
		}
	}
}

func (s *Server) handleLobbyMessage(ctx context.Context, client *clientConn, env Envelope) error {
	switch env.Type {
	case MsgLobbyCreate:
		lobby, err := s.manager.CreateLobby(ctx, client.playerID)
		if err != nil {
			return err
		}
		client.lobbyID = lobby.ID()
		if err := s.send(client, Envelope{
			Type:      MsgLobbyCreated,
			RequestID: env.RequestID,
			Payload:   mustJSON(LobbyCreatedPayload{LobbyID: lobby.ID().String()}),
		}); err != nil {
			return err
		}
		s.broadcastLobbyState(ctx, lobby.ID().String())
		s.broadcastLobbyList(ctx)
		return nil
	case MsgLobbyList:
		lobbies, err := s.manager.ListLobbies(ctx)
		if err != nil {
			return err
		}
		items := make([]LobbySummaryPayload, 0, len(lobbies))
		for _, lobby := range lobbies {
			items = append(items, LobbySummaryPayload{
				ID:          lobby.ID().String(),
				LeaderID:    lobby.LeaderID().String(),
				PlayerCount: lobby.PlayerCount(),
				Status:      string(lobby.Status()),
			})
		}
		return s.send(client, Envelope{
			Type:      MsgLobbyListRes,
			RequestID: env.RequestID,
			Payload:   mustJSON(LobbyListPayload{Lobbies: items}),
		})
	case MsgLobbyJoin:
		var payload LobbyJoinPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return err
		}
		lobbyID := entities.LobbyID(payload.LobbyID)
		if err := s.manager.JoinLobby(ctx, lobbyID, client.playerID); err != nil {
			return err
		}
		client.lobbyID = lobbyID
		if err := s.send(client, Envelope{
			Type:      MsgLobbyJoined,
			RequestID: env.RequestID,
			Payload:   mustJSON(LobbyJoinPayload{LobbyID: payload.LobbyID}),
		}); err != nil {
			return err
		}
		s.broadcastLobbyState(ctx, payload.LobbyID)
		s.broadcastLobbyList(ctx)
		return nil
	case MsgLobbyStart:
		var payload LobbyStartPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return err
		}
		lobbyID := entities.LobbyID(payload.LobbyID)
		if err := s.manager.StartLobby(ctx, lobbyID, client.playerID); err != nil {
			return err
		}
		s.broadcastLobbyState(ctx, payload.LobbyID)
		s.broadcastLobbyList(ctx)
		return nil
	case MsgHello:
		// Ignore late hello messages after handshake.
		return nil
	default:
		return fmt.Errorf("unknown lobby message %q", env.Type)
	}
}

func (s *Server) attachSession(session *server.GameSession, runner *sessionRunner) {
	ctx := context.Background()
	playerNames := make([]string, 0, len(session.PlayerIDs))
	playerAvatars := make([]string, 0, len(session.PlayerIDs))
	for _, id := range session.PlayerIDs {
		playerID := entities.PlayerID(id)
		player, err := s.manager.GetPlayerByID(ctx, playerID)
		if err != nil || player == nil {
			playerNames = append(playerNames, "")
			playerAvatars = append(playerAvatars, "")
			continue
		}
		playerNames = append(playerNames, player.Nick())
		playerAvatars = append(playerAvatars, player.Avatar())
	}

	type clientInfo struct {
		client *clientConn
		index  int
	}
	clientsToNotify := make([]clientInfo, 0, len(session.PlayerIDs))

	s.mu.Lock()
	s.sessions[session.LobbyID] = runner
	for _, id := range session.PlayerIDs {
		client := s.clients[id]
		if client == nil {
			continue
		}
		index := session.IndexByPlayer[id]
		client.session = runner
		client.playerIndex = index
		runner.provider.attach(index, client)
		clientsToNotify = append(clientsToNotify, clientInfo{client: client, index: index})
	}
	s.mu.Unlock()

	for _, info := range clientsToNotify {
		_ = s.send(info.client, Envelope{
			Type: MsgLobbyStarted,
			Payload: mustJSON(LobbyStartedPayload{
				LobbyID:       session.LobbyID,
				MatchID:       session.MatchID,
				PlayerIndex:   info.index,
				PlayerCount:   len(session.PlayerIDs),
				PlayerNames:   playerNames,
				PlayerAvatars: playerAvatars,
				IndexMapping:  session.IndexByPlayer,
			}),
		})
		_ = s.send(info.client, Envelope{
			Type: MsgHandState,
			Payload: mustJSON(HandStatePayload{
				PlayerIndex: info.index,
				Hand:        buildHandRoles(session.Game.CurrentState.Players[info.index].Hand),
			}),
		})
	}
}

func (s *Server) send(client *clientConn, env Envelope) error {
	client.sendMu.Lock()
	err := client.conn.WriteJSON(env)
	client.sendMu.Unlock()
	return err
}

func (s *Server) sendRaw(conn wsConn, env Envelope) error {
	return conn.WriteJSON(env)
}

// reattachToLobby rebinds a reconnecting client to any existing lobby/session.
func (s *Server) reattachToLobby(client *clientConn) {
	ctx := context.Background()
	lobby, err := s.manager.GetLobbyByPlayerID(ctx, client.playerID)
	if err != nil {
		return
	}
	lobbyID := lobby.ID()
	client.lobbyID = lobbyID
	s.mu.Lock()
	session := s.sessions[lobbyID.String()]
	s.mu.Unlock()
	if session == nil {
		// Reset lobby to open if no session exists
		lobby.ResetToOpen()
		_, err := s.manager.LeaveLobby(ctx, lobbyID, client.playerID)
		if err != nil {
			log.Printf("failed to reset lobby %s: %v", lobbyID, err)
		}
		s.broadcastLobbyState(ctx, lobbyID.String())
		return
	}
	if session.session == nil {
		log.Printf("session has no game state for lobby %s", lobbyID)
		lobby.ResetToOpen()
		_, err := s.manager.LeaveLobby(ctx, lobbyID, client.playerID)
		if err != nil {
			log.Printf("failed to reset lobby %s: %v", lobbyID, err)
		}
		s.broadcastLobbyState(ctx, lobbyID.String())
		return
	}
	index, ok := session.session.IndexByPlayer[client.playerID.String()]
	if !ok {
		log.Printf("player %s not found in session for lobby %s", client.playerID, lobbyID)
		return
	}
	client.session = session
	client.playerIndex = index
	session.provider.attach(index, client)
	if index >= 0 && index < len(session.session.PlayerAvatars) && client.playerAvatar != "" {
		session.session.PlayerAvatars[index] = client.playerAvatar
	}
	session.EnqueueOnline(client.playerID.String())
	_ = s.send(client, Envelope{
		Type: MsgHandState,
		Payload: mustJSON(HandStatePayload{
			PlayerIndex: index,
			Hand:        buildHandRoles(session.session.Game.CurrentState.Players[index].Hand),
		}),
	})
	_ = s.send(client, Envelope{
		Type:    MsgGameState,
		Payload: mustJSON(buildGameStatePayload(&session.session.Game.CurrentState, session.session.PlayerAvatars)),
	})
	if paused := session.PauseSnapshot(); paused != nil {
		_ = s.send(client, Envelope{
			Type:    MsgGamePaused,
			Payload: mustJSON(*paused),
		})
	}
	if winner := session.session.Game.CheckGameOver(); winner != nil {
		_ = s.send(client, Envelope{
			Type:    MsgGameOver,
			Payload: mustJSON(buildGameOverPayload(session.session.Game, winner)),
		})
	}
	session.provider.resendPrompts(index)
}

// broadcastLobbyState emits the latest lobby state to all players in the lobby.
func (s *Server) broadcastLobbyState(ctx context.Context, lobbyID string) {
	lobby, err := s.manager.GetLobby(ctx, entities.LobbyID(lobbyID))
	if err != nil {
		log.Printf("broadcastLobbyState: failed to get lobby %s: %v", lobbyID, err)
		return
	}
	playerIDs := lobby.PlayerIDs()
	playerNicks := make([]string, 0, len(playerIDs))
	playerIDStrs := make([]string, 0, len(playerIDs))
	playerAvatars := make([]string, 0, len(playerIDs))
	for _, id := range playerIDs {
		player, err := s.manager.GetPlayerByID(ctx, id)
		if err != nil {
			log.Printf("broadcastLobbyState: failed to get player %s: %v", id, err)
			continue
		}
		playerNicks = append(playerNicks, player.Nick())
		playerIDStrs = append(playerIDStrs, player.ID().String())
		playerAvatars = append(playerAvatars, player.Avatar())
	}
	leader, err := s.manager.GetPlayerByID(ctx, lobby.LeaderID())
	if err != nil {
		log.Printf("broadcastLobbyState: failed to get leader %s: %v", lobby.LeaderID(), err)
	}
	leaderNick := ""
	if leader != nil {
		leaderNick = leader.Nick()
	}

	payload := mustJSON(LobbyStatePayload{
		LobbyID:       lobby.ID().String(),
		LeaderNick:    leaderNick,
		LeaderID:      lobby.LeaderID().String(),
		PlayerNicks:   playerNicks,
		PlayerIDs:     playerIDStrs,
		PlayerAvatars: playerAvatars,
		PlayerCount:   len(playerIDs),
		Status:        string(lobby.Status()),
	})

	// Collect clients while locked
	clientsToNotify := make([]*clientConn, 0, len(playerIDs))
	s.mu.Lock()
	for _, id := range playerIDs {
		if client := s.clients[id.String()]; client != nil {
			clientsToNotify = append(clientsToNotify, client)
		}
	}
	s.mu.Unlock()

	// Send after releasing lock
	for _, client := range clientsToNotify {
		_ = s.send(client, Envelope{Type: MsgLobbyState, Payload: payload})
	}
}

func (s *Server) broadcastLobbyList(ctx context.Context) {
	s.mu.Lock()
	online := make(map[string]struct{}, len(s.clients))
	for id := range s.clients {
		online[id] = struct{}{}
	}
	s.mu.Unlock()

	// Prune empty lobbies older than 1 hour
	if _, err := s.manager.PruneEmptyLobbies(ctx, 1*time.Hour); err != nil {
		log.Printf("broadcastLobbyList: failed to prune lobbies: %v", err)
	}

	lobbies, err := s.manager.ListLobbies(ctx)
	if err != nil {
		log.Printf("broadcastLobbyList: failed to list lobbies: %v", err)
		return
	}
	items := make([]LobbySummaryPayload, 0, len(lobbies))
	for _, lobby := range lobbies {
		playerIDs := lobby.PlayerIDs()
		avatars := make([]string, 0, len(playerIDs))
		playerNicks := make([]string, 0, len(playerIDs))
		playerIDStrs := make([]string, 0, len(playerIDs))
		for _, id := range playerIDs {
			player, err := s.manager.GetPlayerByID(ctx, id)
			if err != nil {
				log.Printf("broadcastLobbyList: failed to get player %s: %v", id, err)
				avatars = append(avatars, "")
				playerNicks = append(playerNicks, "")
				playerIDStrs = append(playerIDStrs, id.String())
				continue
			}
			avatars = append(avatars, player.Avatar())
			playerNicks = append(playerNicks, player.Nick())
			playerIDStrs = append(playerIDStrs, player.ID().String())
		}
		leader, _ := s.manager.GetPlayerByID(ctx, lobby.LeaderID())
		leaderNick := ""
		if leader != nil {
			leaderNick = leader.Nick()
		}
		items = append(items, LobbySummaryPayload{
			ID:            lobby.ID().String(),
			LeaderNick:    leaderNick,
			LeaderID:      lobby.LeaderID().String(),
			PlayerCount:   lobby.PlayerCount(),
			PlayerNicks:   playerNicks,
			PlayerIDs:     playerIDStrs,
			PlayerAvatars: avatars,
			Status:        string(lobby.Status()),
		})
	}
	payload := mustJSON(LobbyListPayload{Lobbies: items})

	// Collect clients while locked, then send after releasing lock to avoid blocking
	clientsToNotify := make([]*clientConn, 0, len(s.clients))
	s.mu.Lock()
	for _, client := range s.clients {
		clientsToNotify = append(clientsToNotify, client)
	}
	s.mu.Unlock()

	for _, client := range clientsToNotify {
		_ = s.send(client, Envelope{Type: MsgLobbyListRes, Payload: payload})
	}
}
