package ws

import (
	"encoding/json"
	"testing"
	"time"

	"ElMakina/backend/engine"
	"ElMakina/backend/server"
)

type pauseCaptureConn struct {
	writes []Envelope
}

func (c *pauseCaptureConn) ReadJSON(v any) error { return nil }
func (c *pauseCaptureConn) WriteJSON(v any) error {
	b, _ := json.Marshal(v)
	var env Envelope
	_ = json.Unmarshal(b, &env)
	c.writes = append(c.writes, env)
	return nil
}
func (c *pauseCaptureConn) Close() error { return nil }

func TestMarkOfflineBroadcastsPauseImmediately(t *testing.T) {
	game, err := engine.NewGame([]string{"Ada", "Ben", "Cara"}, nil)
	if err != nil {
		t.Fatalf("new game: %v", err)
	}
	session := &server.GameSession{
		LobbyID:   "l1",
		Game:      game,
		PlayerIDs: []string{"p1", "p2", "p3"},
		IndexByPlayer: map[string]int{
			"p1": 0,
			"p2": 1,
			"p3": 2,
		},
	}
	runner := newSessionRunner(session, engine.RealClock{}, engine.TurnConfig{}, 60*time.Second, nil)

	connA := &pauseCaptureConn{}
	connC := &pauseCaptureConn{}
	runner.provider.attach(0, &clientConn{conn: connA})
	runner.provider.attach(2, &clientConn{conn: connC})

	runner.markOffline("p2")

	if runner.paused == nil || !runner.paused.active {
		t.Fatalf("expected pause to be active")
	}
	if len(connA.writes) == 0 || connA.writes[len(connA.writes)-1].Type != MsgGamePaused {
		t.Fatalf("expected game_paused to be broadcast to player 0")
	}
	if len(connC.writes) == 0 || connC.writes[len(connC.writes)-1].Type != MsgGamePaused {
		t.Fatalf("expected game_paused to be broadcast to player 2")
	}
}
