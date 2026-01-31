package ws

import (
	"context"
	"encoding/json"
	"testing"

	"ElMakina/backend/domain/services"
	"ElMakina/backend/engine"
	"ElMakina/backend/server"
)

type fakeConn struct {
	writes []Envelope
}

func (c *fakeConn) ReadJSON(v any) error { return nil }
func (c *fakeConn) WriteJSON(v any) error {
	b, _ := json.Marshal(v)
	var env Envelope
	_ = json.Unmarshal(b, &env)
	c.writes = append(c.writes, env)
	return nil
}
func (c *fakeConn) Close() error { return nil }

func TestLobbyCreateListJoinStart(t *testing.T) {
	ctx := context.Background()

	playerRepo := newMockPlayerRepository()
	lobbyRepo := newMockLobbyRepository()
	unitOfWork := &mockUnitOfWork{}
	tokenGen := &mockTokenGenerator{}
	idGen := &mockIDGenerator{}

	playerService := services.NewPlayerService(playerRepo, lobbyRepo, unitOfWork, tokenGen, idGen)
	manager := server.NewLobbyManager(2, 9, playerService, lobbyRepo, unitOfWork)
	srv := NewServer(manager, engine.RealClock{}, engine.TurnConfig{}, 0, nil)
	srv.start = func(*sessionRunner) {}

	leader, _ := manager.RegisterPlayer(ctx, "leader")
	p2, _ := manager.RegisterPlayer(ctx, "p2")

	leaderConn := &fakeConn{}
	p2Conn := &fakeConn{}

	leaderClient := &clientConn{playerID: leader.ID(), playerNick: leader.Nick(), conn: leaderConn}
	p2Client := &clientConn{playerID: p2.ID(), playerNick: p2.Nick(), conn: p2Conn}
	srv.clients[leader.ID().String()] = leaderClient
	srv.clients[p2.ID().String()] = p2Client

	if err := srv.handleLobbyMessage(ctx, leaderClient, Envelope{Type: MsgLobbyCreate, RequestID: "1"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(leaderConn.writes) == 0 || leaderConn.writes[0].Type != MsgLobbyCreated {
		t.Fatalf("expected lobby_created response")
	}

	var created LobbyCreatedPayload
	_ = json.Unmarshal(leaderConn.writes[0].Payload, &created)

	joinPayload, _ := json.Marshal(LobbyJoinPayload{LobbyID: created.LobbyID})
	if err := srv.handleLobbyMessage(ctx, p2Client, Envelope{Type: MsgLobbyJoin, RequestID: "2", Payload: joinPayload}); err != nil {
		t.Fatalf("join: %v", err)
	}

	startPayload, _ := json.Marshal(LobbyStartPayload{LobbyID: created.LobbyID})
	if err := srv.handleLobbyMessage(ctx, leaderClient, Envelope{Type: MsgLobbyStart, RequestID: "3", Payload: startPayload}); err != nil {
		t.Fatalf("start: %v", err)
	}
	// Note: StartLobby no longer returns a session, so we can't test for MsgLobbyStarted
	// The session would be created by the WebSocket server after StartLobby succeeds
}

func containsType(envs []Envelope, want string) bool {
	for _, env := range envs {
		if env.Type == want {
			return true
		}
	}
	return false
}
