package websocket

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func intPtr(v int) *int { return &v }

type fakeConn struct {
	readCh  chan Envelope
	writeCh chan Envelope
	closed  chan struct{}
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		readCh:  make(chan Envelope, 16),
		writeCh: make(chan Envelope, 16),
		closed:  make(chan struct{}),
	}
}

func (c *fakeConn) ReadJSON(v any) error {
	select {
	case <-c.closed:
		return errors.New("closed")
	case env, ok := <-c.readCh:
		if !ok {
			return errors.New("closed")
		}
		b, _ := json.Marshal(env)
		return json.Unmarshal(b, v)
	}
}

func (c *fakeConn) WriteJSON(v any) error {
	select {
	case <-c.closed:
		return errors.New("closed")
	default:
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return err
	}
	c.writeCh <- env
	return nil
}

func (c *fakeConn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
		close(c.readCh)
		close(c.writeCh)
	}
	return nil
}

func registerFake(t *testing.T, p *Provider, playerIndex int) *fakeConn {
	t.Helper()
	conn := newFakeConn()
	if err := p.registerConn(playerIndex, conn); err != nil {
		t.Fatalf("register: %v", err)
	}
	return conn
}

func mustJSON(t *testing.T, v any) RawMessage {
	t.Helper()
	raw, err := Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestProviderRequestAction(t *testing.T) {
	p := NewProvider(nil)
	conn := registerFake(t, p, 0)

	resCh := make(chan *models.PlayerAction, 1)
	errCh := make(chan error, 1)
	go func() {
		cmd, err := p.RequestAction(context.Background(), 0, &state.GameState{})
		resCh <- cmd
		errCh <- err
	}()

	req := <-conn.writeCh
	if req.Type != "request_action" || req.RequestID == "" {
		t.Fatalf("unexpected request: %+v", req)
	}

	conn.readCh <- Envelope{
		Type:      "action",
		RequestID: req.RequestID,
		Payload: mustJSON(t, actionPayload{
			ID:          models.Income,
			SourceIndex: intPtr(0),
		}),
	}

	cmd := <-resCh
	err := <-errCh
	if err != nil {
		t.Fatalf("request action error: %v", err)
	}
	if cmd.ID != models.Income || cmd.SourceIndex != 0 {
		t.Fatalf("unexpected command: %+v", cmd)
	}
}

func TestProviderChallengeResponses(t *testing.T) {
	p := NewProvider(nil)
	conn := registerFake(t, p, 1)

	window := engine.ChallengeWindow{
		Kind:               engine.ChallengeMain,
		Action:             &models.PlayerAction{ID: models.Investigate, SourceIndex: 0},
		ActorIndex:         0,
		ClaimedRole:        models.Policewoman,
		AllowedChallengers: []int{1},
	}

	ch, err := p.ChallengeResponses(context.Background(), window)
	if err != nil {
		t.Fatalf("challenge responses error: %v", err)
	}

	req := <-conn.writeCh
	if req.Type != "challenge_window" || req.RequestID == "" {
		t.Fatalf("unexpected challenge request: %+v", req)
	}

	conn.readCh <- Envelope{
		Type:      "challenge",
		RequestID: req.RequestID,
		Payload: mustJSON(t, challengePayload{
			ChallengerIndex:   intPtr(1),
			ActorDiscardIndex: 0,
		}),
	}

	decision, ok := <-ch
	if !ok {
		t.Fatalf("expected decision")
	}
	if decision.ChallengerIndex != 1 || decision.ActorDiscardIndex != 0 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestProviderCounterResponses(t *testing.T) {
	p := NewProvider(nil)
	conn := registerFake(t, p, 1)

	window := engine.CounterWindow{
		MainAction:     &models.PlayerAction{ID: models.Steal, SourceIndex: 0},
		AllowedPlayers: []int{1},
		AllowedActions: []models.ActionID{models.BlockSteal},
	}

	ch, err := p.CounterResponses(context.Background(), window)
	if err != nil {
		t.Fatalf("counter responses error: %v", err)
	}

	req := <-conn.writeCh
	if req.Type != "counter_window" || req.RequestID == "" {
		t.Fatalf("unexpected counter request: %+v", req)
	}

	conn.readCh <- Envelope{
		Type:      "counter",
		RequestID: req.RequestID,
		Payload: mustJSON(t, actionPayload{
			ID:          models.BlockSteal,
			SourceIndex: intPtr(1),
		}),
	}

	decision, ok := <-ch
	if !ok {
		t.Fatalf("expected counter command")
	}
	if decision.Command == nil {
		t.Fatalf("expected counter command")
	}
	if decision.Command.ID != models.BlockSteal || decision.Command.SourceIndex != 1 {
		t.Fatalf("unexpected counter command: %+v", decision.Command)
	}
}

func TestProviderDisconnectAbortsRequest(t *testing.T) {
	p := NewProvider(nil)
	conn := registerFake(t, p, 0)

	done := make(chan error, 1)
	go func() {
		_, err := p.RequestAction(context.Background(), 0, &state.GameState{})
		done <- err
	}()

	_ = conn.Close()

	if err := <-done; err == nil {
		t.Fatalf("expected error on disconnect")
	}
}

func TestProviderUnknownRequestIDReportsError(t *testing.T) {
	p := NewProvider(nil)
	_ = registerFake(t, p, 0)

	p.dispatch(0, Envelope{
		Type:      "action",
		RequestID: "unknown",
	})

	err := <-p.Errors()
	if err == nil {
		t.Fatalf("expected error for unknown request id")
	}
}

func TestProviderActionMissingSourceIndexFails(t *testing.T) {
	p := NewProvider(nil)
	conn := registerFake(t, p, 0)

	resCh := make(chan *models.PlayerAction, 1)
	errCh := make(chan error, 1)
	go func() {
		cmd, err := p.RequestAction(context.Background(), 0, &state.GameState{})
		resCh <- cmd
		errCh <- err
	}()

	req := <-conn.writeCh
	conn.readCh <- Envelope{
		Type:      "action",
		RequestID: req.RequestID,
		Payload: mustJSON(t, actionPayload{
			ID: models.Income,
		}),
	}

	err := <-errCh
	if err == nil {
		t.Fatalf("expected error for missing source_index")
	}
}

func TestProviderActionGuessWithoutTargetFails(t *testing.T) {
	p := NewProvider(nil)
	conn := registerFake(t, p, 0)

	resCh := make(chan *models.PlayerAction, 1)
	errCh := make(chan error, 1)
	go func() {
		cmd, err := p.RequestAction(context.Background(), 0, &state.GameState{})
		resCh <- cmd
		errCh <- err
	}()

	req := <-conn.writeCh
	guess := "Thief"
	conn.readCh <- Envelope{
		Type:      "action",
		RequestID: req.RequestID,
		Payload: mustJSON(t, actionPayload{
			ID:          models.Accuse,
			SourceIndex: intPtr(0),
			Guess:       &guess,
		}),
	}

	err := <-errCh
	if err == nil {
		t.Fatalf("expected error for guess without target_index")
	}
}

func TestProviderChallengeMissingChallengerIndexReportsError(t *testing.T) {
	p := NewProvider(nil)
	conn := registerFake(t, p, 1)

	window := engine.ChallengeWindow{
		Kind:               engine.ChallengeMain,
		Action:             &models.PlayerAction{ID: models.Investigate, SourceIndex: 0},
		ActorIndex:         0,
		ClaimedRole:        models.Policewoman,
		AllowedChallengers: []int{1},
	}

	ch, err := p.ChallengeResponses(context.Background(), window)
	if err != nil {
		t.Fatalf("challenge responses error: %v", err)
	}

	req := <-conn.writeCh
	conn.readCh <- Envelope{
		Type:      "challenge",
		RequestID: req.RequestID,
		Payload:   mustJSON(t, challengePayload{}),
	}

	<-ch
	if err := <-p.Errors(); err == nil {
		t.Fatalf("expected provider error for missing challenger_index")
	}
}
