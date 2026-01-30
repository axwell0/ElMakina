package ws

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/server"
	"encoding/json"
	"testing"
)

type gameOverConn struct {
	writes []Envelope
}

func (c *gameOverConn) ReadJSON(v any) error { return nil }
func (c *gameOverConn) WriteJSON(v any) error {
	b, _ := json.Marshal(v)
	var env Envelope
	_ = json.Unmarshal(b, &env)
	c.writes = append(c.writes, env)
	return nil
}
func (c *gameOverConn) Close() error { return nil }

func TestSessionRunnerBroadcastsGameOver(t *testing.T) {
	session := &server.GameSession{
		Game: &engine.Game{
			CurrentState: state.GameState{
				Players: []state.Player{{ID: 0, Name: "A", Hand: []state.Card{{ID: 1}}}, {ID: 1, Name: "B", Hand: nil}},
			},
		},
	}
	runner := &sessionRunner{
		session:  session,
		provider: newSessionProvider(session, engine.TurnConfig{}, engine.RealClock{}, nil),
	}

	p0 := &gameOverConn{}
	p1 := &gameOverConn{}
	runner.provider.attach(0, &clientConn{conn: p0})
	runner.provider.attach(1, &clientConn{conn: p1})

	winner := session.Game.CheckGameOver()
	if winner == nil {
		t.Fatalf("expected winner")
	}

	runner.broadcastGameOver(winner)

	if len(p0.writes) != 1 || len(p1.writes) != 1 {
		t.Fatalf("expected game_over to be broadcast to all players")
	}
	if p0.writes[0].Type != MsgGameOver || p1.writes[0].Type != MsgGameOver {
		t.Fatalf("unexpected message type")
	}

	var payload GameOverPayload
	if err := json.Unmarshal(p0.writes[0].Payload, &payload); err != nil {
		t.Fatalf("payload parse: %v", err)
	}
	if payload.WinnerIndex != 0 || payload.WinnerName != "A" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
