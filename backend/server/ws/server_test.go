package ws

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/server"
	"encoding/json"
	"testing"
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
	manager := server.NewLobbyManager(2, 9)
	srv := NewServer(manager, engine.RealClock{}, engine.TurnConfig{}, 0)
	srv.start = func(*sessionRunner) {}

	leader, _ := manager.RegisterPlayer("leader")
	p2, _ := manager.RegisterPlayer("p2")

	leaderConn := &fakeConn{}
	p2Conn := &fakeConn{}

	leaderClient := &clientConn{player: leader, conn: leaderConn}
	p2Client := &clientConn{player: p2, conn: p2Conn}
	srv.clients[leader.ID] = leaderClient
	srv.clients[p2.ID] = p2Client

	if err := srv.handleLobbyMessage(leaderClient, Envelope{Type: MsgLobbyCreate, RequestID: "1"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(leaderConn.writes) == 0 || leaderConn.writes[0].Type != MsgLobbyCreated {
		t.Fatalf("expected lobby_created response")
	}

	var created LobbyCreatedPayload
	_ = json.Unmarshal(leaderConn.writes[0].Payload, &created)

	joinPayload, _ := json.Marshal(LobbyJoinPayload{LobbyID: created.LobbyID})
	if err := srv.handleLobbyMessage(p2Client, Envelope{Type: MsgLobbyJoin, RequestID: "2", Payload: joinPayload}); err != nil {
		t.Fatalf("join: %v", err)
	}

	startPayload, _ := json.Marshal(LobbyStartPayload{LobbyID: created.LobbyID})
	if err := srv.handleLobbyMessage(leaderClient, Envelope{Type: MsgLobbyStart, RequestID: "3", Payload: startPayload}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !containsType(leaderConn.writes, MsgLobbyStarted) {
		t.Fatalf("expected lobby_started for leader")
	}
	if !containsType(p2Conn.writes, MsgLobbyStarted) {
		t.Fatalf("expected lobby_started for p2")
	}
}

func containsType(envs []Envelope, want string) bool {
	for _, env := range envs {
		if env.Type == want {
			return true
		}
	}
	return false
}
