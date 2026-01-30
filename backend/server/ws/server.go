package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

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
	upgrader gws.Upgrader
	manager  *server.LobbyManager
	clock    engine.Clock
	cfg      engine.TurnConfig
	start    func(*sessionRunner)
	grace    time.Duration
	forfeit  *ForfeitScheduler
	recorder replay.Recorder

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
	player      *server.Player
	conn        wsConn
	sendMu      sync.Mutex
	lobbyID     string
	session     *sessionRunner
	playerIndex int
}

// NewServer constructs a lobby server.
func NewServer(manager *server.LobbyManager, clock engine.Clock, cfg engine.TurnConfig, grace time.Duration) *Server {
	if manager == nil {
		manager = server.NewLobbyManager(2, 9)
	}
	if clock == nil {
		clock = engine.RealClock{}
	}
	return &Server{
		upgrader: gws.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		manager:  manager,
		clock:    clock,
		cfg:      cfg,
		grace:    grace,
		start:    func(r *sessionRunner) { r.Start() },
		clients:  make(map[string]*clientConn),
		sessions: make(map[string]*sessionRunner),
		forfeit:  NewForfeitScheduler(clock, grace),
	}
}

// SetRecorder attaches a match recorder for replay logging.
func (s *Server) SetRecorder(recorder replay.Recorder) {
	s.recorder = recorder
}

// ServeHTTP upgrades to WebSocket and starts the lobby/game loop for a client.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	var player *server.Player
	if payload.ReconnectToken != "" {
		player, err = s.manager.ReconnectPlayer(payload.ReconnectToken)
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
		player, err = s.manager.RegisterPlayer(payload.Nickname)
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
		if err := s.manager.SetPlayerAvatar(player.ID, payload.Avatar); err == nil {
			player.Avatar = payload.Avatar
		}
	}

	client := &clientConn{
		player: player,
		conn:   conn,
	}
	log.Printf("client connected: %s (%s)", player.Nick, player.ID)
	s.registerClient(client)
	_ = s.send(client, Envelope{
		Type:    MsgHelloAck,
		Payload: mustJSON(HelloAckPayload{PlayerID: player.ID, Token: player.Token}),
	})
	_ = s.send(client, Envelope{
		Type:    MsgGameConfig,
		Payload: mustJSON(buildGameConfigPayload()),
	})
	s.broadcastLobbyList()
	s.forfeit.Cancel(player.ID)
	s.reattachToLobby(client)
	go s.readLoop(client)
}

func (s *Server) registerClient(client *clientConn) {
	var prior *clientConn
	s.mu.Lock()
	prior = s.clients[client.player.ID]
	s.clients[client.player.ID] = client
	s.mu.Unlock()
	if prior != nil && prior != client {
		_ = prior.conn.Close()
	}
}

func (s *Server) unregisterClient(client *clientConn, err error) {
	s.mu.Lock()
	current := s.clients[client.player.ID]
	if current != nil && current != client {
		s.mu.Unlock()
		return
	}
	delete(s.clients, client.player.ID)
	s.mu.Unlock()

	if client.session != nil {
		client.session.provider.detach(client.playerIndex, client)
		client.session.EnqueueOffline(client.player.ID)
		s.forfeit.Schedule(client.player.ID, func() { client.session.Forfeit(client.player.ID) })
		return
	}
	if client.lobbyID != "" {
		_ = s.manager.RemovePlayer(client.lobbyID, client.player.ID)
		s.broadcastLobbyState(client.lobbyID)
		s.broadcastLobbyList()
		log.Printf("client left lobby: %s (%s)", client.player.Nick, client.player.ID)
		return
	}
	// Keep player registration to preserve reconnect token across restarts.
	log.Printf("client disconnected: %s (%s)", client.player.Nick, client.player.ID)
	s.broadcastLobbyList()
}

func (s *Server) readLoop(client *clientConn) {
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
		if err := s.handleLobbyMessage(client, env); err != nil {
			_ = s.send(client, Envelope{Type: MsgLobbyError, RequestID: env.RequestID, Payload: mustJSON(map[string]string{"error": err.Error()})})
		}
	}
}

func (s *Server) handleLobbyMessage(client *clientConn, env Envelope) error {
	switch env.Type {
	case MsgLobbyCreate:
		lobby, err := s.manager.CreateLobby(client.player.ID)
		if err != nil {
			return err
		}
		client.lobbyID = lobby.ID
		if err := s.send(client, Envelope{
			Type:      MsgLobbyCreated,
			RequestID: env.RequestID,
			Payload:   mustJSON(LobbyCreatedPayload{LobbyID: lobby.ID}),
		}); err != nil {
			return err
		}
		s.broadcastLobbyState(lobby.ID)
		s.broadcastLobbyList()
		return nil
	case MsgLobbyList:
		lobbies := s.manager.ListLobbies()
		items := make([]LobbySummaryPayload, 0, len(lobbies))
		for _, lobby := range lobbies {
			items = append(items, LobbySummaryPayload{
				ID:          lobby.ID,
				LeaderNick:  lobby.LeaderNick,
				LeaderID:    lobby.LeaderID,
				PlayerCount: lobby.PlayerCount,
				PlayerNicks: lobby.PlayerNicks,
				PlayerIDs:   lobby.PlayerIDs,
				Status:      string(lobby.Status),
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
		if err := s.manager.JoinLobby(payload.LobbyID, client.player.ID); err != nil {
			return err
		}
		client.lobbyID = payload.LobbyID
		if err := s.send(client, Envelope{
			Type:      MsgLobbyJoined,
			RequestID: env.RequestID,
			Payload:   mustJSON(LobbyJoinPayload{LobbyID: payload.LobbyID}),
		}); err != nil {
			return err
		}
		s.broadcastLobbyState(payload.LobbyID)
		s.broadcastLobbyList()
		return nil
	case MsgLobbyStart:
		var payload LobbyStartPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return err
		}
		session, err := s.manager.StartLobby(payload.LobbyID, client.player.ID)
		if err != nil {
			return err
		}
		runner := newSessionRunner(session, s.clock, s.cfg, s.grace, s.recorder)
		s.attachSession(session, runner)
		s.start(runner)
		s.broadcastLobbyState(payload.LobbyID)
		s.broadcastLobbyList()
		return nil
	case MsgHello:
		// Ignore late hello messages after handshake.
		return nil
	default:
		return fmt.Errorf("unknown lobby message %q", env.Type)
	}
}

func (s *Server) attachSession(session *server.GameSession, runner *sessionRunner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.LobbyID] = runner

	playerNames := make([]string, 0, len(session.PlayerIDs))
	playerAvatars := make([]string, 0, len(session.PlayerIDs))
	for _, id := range session.PlayerIDs {
		player, err := s.manager.GetPlayer(id)
		if err != nil || player == nil {
			playerNames = append(playerNames, "")
			playerAvatars = append(playerAvatars, "")
			continue
		}
		playerNames = append(playerNames, player.Nick)
		playerAvatars = append(playerAvatars, player.Avatar)
	}

	for _, id := range session.PlayerIDs {
		client := s.clients[id]
		if client == nil {
			continue
		}
		index := session.IndexByPlayer[id]
		client.session = runner
		client.playerIndex = index
		runner.provider.attach(index, client)
		_ = s.send(client, Envelope{
			Type: MsgLobbyStarted,
			Payload: mustJSON(LobbyStartedPayload{
				LobbyID:       session.LobbyID,
				MatchID:       session.MatchID,
				PlayerIndex:   index,
				PlayerCount:   len(session.PlayerIDs),
				PlayerNames:   playerNames,
				PlayerAvatars: playerAvatars,
				IndexMapping:  session.IndexByPlayer,
			}),
		})
		_ = s.send(client, Envelope{
			Type: MsgHandState,
			Payload: mustJSON(HandStatePayload{
				PlayerIndex: index,
				Hand:        buildHandRoles(session.Game.CurrentState.Players[index].Hand),
			}),
		})
	}
}

func (s *Server) send(client *clientConn, env Envelope) error {
	client.sendMu.Lock()
	defer client.sendMu.Unlock()
	return client.conn.WriteJSON(env)
}

func (s *Server) sendRaw(conn wsConn, env Envelope) error {
	if conn == nil {
		return fmt.Errorf("nil connection")
	}
	return conn.WriteJSON(env)
}

// reattachToLobby rebinds a reconnecting client to any existing lobby/session.
func (s *Server) reattachToLobby(client *clientConn) {
	lobbyID, ok := s.manager.FindLobbyByPlayer(client.player.ID)
	if !ok {
		return
	}
	client.lobbyID = lobbyID
	session := s.sessions[lobbyID]
	if session == nil {
		_ = s.manager.ResetLobbyToOpen(lobbyID)
		s.broadcastLobbyState(lobbyID)
		return
	}
	index := session.session.IndexByPlayer[client.player.ID]
	client.session = session
	client.playerIndex = index
	session.provider.attach(index, client)
	if index >= 0 && index < len(session.session.PlayerAvatars) && client.player.Avatar != "" {
		session.session.PlayerAvatars[index] = client.player.Avatar
	}
	session.EnqueueOnline(client.player.ID)
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
func (s *Server) broadcastLobbyState(lobbyID string) {
	lobby, err := s.manager.GetLobby(lobbyID)
	if err != nil {
		return
	}
	playerNicks := make([]string, 0, len(lobby.PlayerIDs))
	playerIDs := make([]string, 0, len(lobby.PlayerIDs))
	playerAvatars := make([]string, 0, len(lobby.PlayerIDs))
	for _, id := range lobby.PlayerIDs {
		player, err := s.manager.GetPlayer(id)
		if err != nil {
			continue
		}
		playerNicks = append(playerNicks, player.Nick)
		playerIDs = append(playerIDs, player.ID)
		playerAvatars = append(playerAvatars, player.Avatar)
	}
	leader, _ := s.manager.GetPlayer(lobby.LeaderID)
	leaderNick := ""
	if leader != nil {
		leaderNick = leader.Nick
	}

	payload := mustJSON(LobbyStatePayload{
		LobbyID:       lobby.ID,
		LeaderNick:    leaderNick,
		LeaderID:      lobby.LeaderID,
		PlayerNicks:   playerNicks,
		PlayerIDs:     playerIDs,
		PlayerAvatars: playerAvatars,
		PlayerCount:   len(lobby.PlayerIDs),
		Status:        string(lobby.Status),
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range lobby.PlayerIDs {
		client := s.clients[id]
		if client == nil {
			continue
		}
		_ = s.send(client, Envelope{Type: MsgLobbyState, Payload: payload})
	}
}

func (s *Server) broadcastLobbyList() {
	online := make(map[string]struct{}, len(s.clients))
	s.mu.Lock()
	for id := range s.clients {
		online[id] = struct{}{}
	}
	s.mu.Unlock()
	_, _ = s.manager.PruneEmptyOpenLobbies(online)
	lobbies := s.manager.ListLobbies()
	items := make([]LobbySummaryPayload, 0, len(lobbies))
	for _, lobby := range lobbies {
		avatars := make([]string, 0, len(lobby.PlayerIDs))
		for _, id := range lobby.PlayerIDs {
			player, err := s.manager.GetPlayer(id)
			if err != nil {
				avatars = append(avatars, "")
				continue
			}
			avatars = append(avatars, player.Avatar)
		}
		items = append(items, LobbySummaryPayload{
			ID:            lobby.ID,
			LeaderNick:    lobby.LeaderNick,
			LeaderID:      lobby.LeaderID,
			PlayerCount:   lobby.PlayerCount,
			PlayerNicks:   lobby.PlayerNicks,
			PlayerIDs:     lobby.PlayerIDs,
			PlayerAvatars: avatars,
			Status:        string(lobby.Status),
		})
	}
	payload := mustJSON(LobbyListPayload{Lobbies: items})

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, client := range s.clients {
		_ = s.send(client, Envelope{Type: MsgLobbyListRes, Payload: payload})
	}
}
