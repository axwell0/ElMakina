package ws

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"ElMakina/backend/server"
	"context"
	"encoding/json"
	"testing"
	"time"
)

type recordingConn struct {
	writeCh chan Envelope
}

func newRecordingConn() *recordingConn {
	return &recordingConn{writeCh: make(chan Envelope, 4)}
}

func (c *recordingConn) ReadJSON(v any) error { return nil }
func (c *recordingConn) WriteJSON(v any) error {
	b, _ := json.Marshal(v)
	var env Envelope
	_ = json.Unmarshal(b, &env)
	c.writeCh <- env
	return nil
}
func (c *recordingConn) Close() error { return nil }

func intPtr(v int) *int { return &v }

type manualClock struct {
	ch chan time.Time
}

func newManualClock() *manualClock {
	return &manualClock{ch: make(chan time.Time, 1)}
}

func (c *manualClock) After(d time.Duration) <-chan time.Time {
	return c.ch
}

func (c *manualClock) Fire() {
	c.ch <- time.Now()
}

func TestRequestActionFiltersUnaffordable(t *testing.T) {
	t.Parallel()
	game := &engine.Game{
		CurrentState: state.GameState{
			TurnNumber:         1,
			Players:            []state.Player{{ID: 0, Name: "A", Hand: []state.Card{{ID: 1}}, Coins: 2}},
			CurrentPlayerIndex: 0,
		},
	}
	session := &server.GameSession{Game: game}
	provider := newSessionProvider(session, engine.TurnConfig{}, engine.RealClock{}, nil)
	conn := newRecordingConn()
	provider.attach(0, &clientConn{conn: conn})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := provider.RequestAction(ctx, 0, &game.CurrentState)
		done <- err
	}()

	var req Envelope
	select {
	case req = <-conn.writeCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for request_action")
	}

	var payload RequestActionPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatalf("payload parse: %v", err)
	}
	for _, id := range payload.AllowedAction {
		if id == models.Coup || id == models.Assassinate {
			t.Fatalf("expected %s to be filtered out", id)
		}
	}

	provider.dispatch(0, Envelope{
		Type:      MsgAction,
		RequestID: req.RequestID,
		Payload: mustJSON(ActionPayload{
			ID:          models.Income,
			SourceIndex: intPtr(0),
		}),
	})

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("request action error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for RequestAction completion")
	}
}

func TestRequestActionFiltersIneligibleTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		players       []state.Player
		expectMissing []models.ActionID
		expectAllowed []models.ActionID
	}{
		{
			name: "steal filtered when no target has 2 coins",
			players: []state.Player{
				{ID: 0, Name: "A", Hand: []state.Card{{ID: 1}}, Coins: 2},
				{ID: 1, Name: "B", Hand: []state.Card{{ID: 2}}, Coins: 1},
			},
			expectMissing: []models.ActionID{models.Steal, models.Coup},
			expectAllowed: []models.ActionID{models.Income, models.ForeignAid},
		},
		{
			name: "targeted actions filtered when no other alive player",
			players: []state.Player{
				{ID: 0, Name: "A", Hand: []state.Card{{ID: 1}}, Coins: 7},
				{ID: 1, Name: "B", Hand: nil, Coins: 2},
			},
			expectMissing: []models.ActionID{models.Coup, models.Steal, models.Accuse, models.Assassinate, models.Investigate},
			expectAllowed: []models.ActionID{models.Income, models.ForeignAid},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			game := &engine.Game{
				CurrentState: state.GameState{
					TurnNumber:         1,
					Players:            tc.players,
					CurrentPlayerIndex: 0,
				},
			}
			session := &server.GameSession{Game: game}
			provider := newSessionProvider(session, engine.TurnConfig{}, engine.RealClock{}, nil)
			conn := newRecordingConn()
			provider.attach(0, &clientConn{conn: conn})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan error, 1)
			go func() {
				_, err := provider.RequestAction(ctx, 0, &game.CurrentState)
				done <- err
			}()

			var req Envelope
			select {
			case req = <-conn.writeCh:
			case <-time.After(200 * time.Millisecond):
				t.Fatalf("timeout waiting for request_action")
			}

			var payload RequestActionPayload
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				t.Fatalf("payload parse: %v", err)
			}

			for _, id := range tc.expectMissing {
				if containsAction(payload.AllowedAction, id) {
					t.Fatalf("expected %s to be filtered out", id)
				}
			}
			for _, id := range tc.expectAllowed {
				if !containsAction(payload.AllowedAction, id) {
					t.Fatalf("expected %s to be allowed", id)
				}
			}

			provider.dispatch(0, Envelope{
				Type:      MsgAction,
				RequestID: req.RequestID,
				Payload: mustJSON(ActionPayload{
					ID:          models.Income,
					SourceIndex: intPtr(0),
				}),
			})

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("request action error: %v", err)
				}
			case <-time.After(200 * time.Millisecond):
				t.Fatalf("timeout waiting for RequestAction completion")
			}
		})
	}
}

func containsAction(actions []models.ActionID, want models.ActionID) bool {
	for _, id := range actions {
		if id == want {
			return true
		}
	}
	return false
}

func TestRequestActionTimeoutReturnsPass(t *testing.T) {
	t.Parallel()
	game := &engine.Game{
		CurrentState: state.GameState{
			TurnNumber:         1,
			Players:            []state.Player{{ID: 0, Name: "A", Hand: []state.Card{{ID: 1}}, Coins: 2}},
			CurrentPlayerIndex: 0,
		},
	}
	session := &server.GameSession{Game: game}
	clock := newManualClock()
	provider := newSessionProvider(session, engine.TurnConfig{TurnTimeout: 20 * time.Second}, clock, nil)
	conn := newRecordingConn()
	provider.attach(0, &clientConn{conn: conn})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		cmd *models.PlayerAction
		err error
	}
	done := make(chan result, 1)
	go func() {
		cmd, err := provider.RequestAction(ctx, 0, &game.CurrentState)
		done <- result{cmd: cmd, err: err}
	}()

	select {
	case <-conn.writeCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for request_action")
	}

	clock.Fire()

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("request action error: %v", res.err)
		}
		if res.cmd == nil {
			t.Fatalf("expected pass command, got nil")
		}
		if res.cmd.ID != models.Pass {
			t.Fatalf("expected pass command, got %s", res.cmd.ID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for RequestAction timeout")
	}
}
