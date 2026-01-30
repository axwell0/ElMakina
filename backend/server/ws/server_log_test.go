package ws

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/server"
	"encoding/json"
	"testing"
)

type logConn struct {
	writes []Envelope
}

func (c *logConn) ReadJSON(v any) error { return nil }
func (c *logConn) WriteJSON(v any) error {
	b, _ := json.Marshal(v)
	var env Envelope
	_ = json.Unmarshal(b, &env)
	c.writes = append(c.writes, env)
	return nil
}
func (c *logConn) Close() error { return nil }

func TestSessionRunnerBroadcastsLogs(t *testing.T) {
	session := &server.GameSession{}
	runner := &sessionRunner{
		session:  session,
		provider: newSessionProvider(session, engine.TurnConfig{}, engine.RealClock{}, nil),
	}

	p0 := &logConn{}
	p1 := &logConn{}
	runner.provider.attach(0, &clientConn{conn: p0})
	runner.provider.attach(1, &clientConn{conn: p1})

	res := &engine.TurnResult{
		Logs: []string{"public 1", "public 2"},
		PrivateLogs: map[int][]string{
			0: {"secret 0"},
		},
	}

	runner.broadcastLogs(3, res)

	if len(p0.writes) != 3 {
		t.Fatalf("expected 3 messages for p0, got %d", len(p0.writes))
	}
	if len(p1.writes) != 2 {
		t.Fatalf("expected 2 messages for p1, got %d", len(p1.writes))
	}
	for _, env := range p1.writes {
		if env.Type != MsgGameLog {
			t.Fatalf("unexpected message type: %s", env.Type)
		}
	}
}
