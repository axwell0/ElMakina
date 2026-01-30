package ws

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/server"
	"encoding/json"
	"testing"
)

type captureConn struct {
	writes []Envelope
}

func (c *captureConn) ReadJSON(v any) error { return nil }
func (c *captureConn) WriteJSON(v any) error {
	b, _ := json.Marshal(v)
	var env Envelope
	_ = json.Unmarshal(b, &env)
	c.writes = append(c.writes, env)
	return nil
}
func (c *captureConn) Close() error { return nil }

func TestBroadcastLogsSendsInvestigateResult(t *testing.T) {
	session := &server.GameSession{
		Game: &engine.Game{CurrentState: state.GameState{Players: []state.Player{{ID: 0, Name: "A", Hand: []state.Card{{ID: 1}}}}}},
	}
	runner := &sessionRunner{session: session, provider: newSessionProvider(session, engine.TurnConfig{}, engine.RealClock{}, nil)}

	p0 := &captureConn{}
	runner.provider.attach(0, &clientConn{conn: p0})

	res := &engine.TurnResult{
		PrivateLogs: map[int][]string{0: {"You see Bob has Thief"}},
	}

	runner.broadcastLogs(1, res)

	found := false
	for _, env := range p0.writes {
		if env.Type != MsgInvestigateRes {
			continue
		}
		var payload InvestigateResultPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			t.Fatalf("payload parse: %v", err)
		}
		if payload.TargetName != "Bob" || payload.Role != "Thief" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("expected investigate_result")
	}
}

func TestBroadcastLogsSendsHandState(t *testing.T) {
	session := &server.GameSession{
		Game: &engine.Game{CurrentState: state.GameState{Players: []state.Player{
			{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: "Thief"}}},
			{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: "Colonel"}, {ID: 3, Role: "Politician"}}},
		}}},
	}
	runner := &sessionRunner{session: session, provider: newSessionProvider(session, engine.TurnConfig{}, engine.RealClock{}, nil)}

	p0 := &captureConn{}
	p1 := &captureConn{}
	runner.provider.attach(0, &clientConn{conn: p0})
	runner.provider.attach(1, &clientConn{conn: p1})

	runner.broadcastLogs(1, &engine.TurnResult{})

	findHand := func(envs []Envelope) (HandStatePayload, bool) {
		for _, env := range envs {
			if env.Type != MsgHandState {
				continue
			}
			var payload HandStatePayload
			if err := json.Unmarshal(env.Payload, &payload); err != nil {
				t.Fatalf("payload parse: %v", err)
			}
			return payload, true
		}
		return HandStatePayload{}, false
	}

	p0Payload, ok := findHand(p0.writes)
	if !ok {
		t.Fatalf("expected hand_state for p0")
	}
	if p0Payload.PlayerIndex != 0 || len(p0Payload.Hand) != 1 || p0Payload.Hand[0] != "Thief" {
		t.Fatalf("unexpected p0 hand: %+v", p0Payload)
	}

	p1Payload, ok := findHand(p1.writes)
	if !ok {
		t.Fatalf("expected hand_state for p1")
	}
	if p1Payload.PlayerIndex != 1 || len(p1Payload.Hand) != 2 {
		t.Fatalf("unexpected p1 hand: %+v", p1Payload)
	}
}

func TestBroadcastEliminationsSendsPlayerEliminated(t *testing.T) {
	session := &server.GameSession{
		Game: &engine.Game{CurrentState: state.GameState{Players: []state.Player{
			{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: "Thief"}}},
			{ID: 1, Name: "B", Hand: []state.Card{}},
		}}},
	}
	runner := &sessionRunner{session: session, provider: newSessionProvider(session, engine.TurnConfig{}, engine.RealClock{}, nil)}

	p0 := &captureConn{}
	p1 := &captureConn{}
	runner.provider.attach(0, &clientConn{conn: p0})
	runner.provider.attach(1, &clientConn{conn: p1})

	runner.broadcastEliminations([]int{1, 1}, 7, nil)

	found := false
	for _, env := range p0.writes {
		if env.Type != MsgPlayerEliminated {
			continue
		}
		var payload PlayerEliminatedPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			t.Fatalf("payload parse: %v", err)
		}
		if payload.PlayerIndex != 1 || payload.Reason != "lost_last_card" || payload.Turn != 7 {
			t.Fatalf("unexpected payload: %+v", payload)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("expected player_eliminated event")
	}
}
