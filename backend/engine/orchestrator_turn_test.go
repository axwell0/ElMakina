package engine

import (
	"context"
	"testing"
	"time"

	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

type stubProvider struct {
	action     *models.PlayerAction
	challenges []ChallengeDecision
	counters   []CounterDecision
	steps      []any
}

func (s *stubProvider) RequestAction(ctx context.Context, actor int, gs *state.GameState) (*models.PlayerAction, error) {
	return s.action, nil
}

func (s *stubProvider) ChallengeResponses(ctx context.Context, window ChallengeWindow) (<-chan ChallengeDecision, error) {
	ch := make(chan ChallengeDecision, 1)
	if len(s.challenges) > 0 {
		ch <- s.challenges[0]
		s.challenges = s.challenges[1:]
	}
	close(ch)
	return ch, nil
}

func (s *stubProvider) CounterResponses(ctx context.Context, window CounterWindow) (<-chan CounterDecision, error) {
	ch := make(chan CounterDecision, 1)
	if len(s.counters) > 0 {
		ch <- s.counters[0]
		s.counters = s.counters[1:]
	}
	close(ch)
	return ch, nil
}

func (s *stubProvider) RequestStep(ctx context.Context, req state.StepRequest) (any, error) {
	if len(s.steps) == 0 {
		return []int{0}, nil
	}
	resp := s.steps[0]
	s.steps = s.steps[1:]
	return resp, nil
}

type testClock struct{}

func (testClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time)
	return ch
}

func TestRunTurnIncome(t *testing.T) {
	game := &Game{CurrentState: state.GameState{
		TurnNumber:         1,
		CurrentPlayerIndex: 0,
		Players: []state.Player{
			{ID: 0, Name: "A", Coins: 0, Hand: []state.Card{{ID: 1, Role: models.TaxCollector}}},
			{ID: 1, Name: "B", Coins: 0, Hand: []state.Card{{ID: 2, Role: models.Thief}}},
		},
	}}
	provider := &stubProvider{action: &models.PlayerAction{ID: models.Income, SourceIndex: 0}}
	res, err := game.RunTurn(context.Background(), provider, testClock{}, TurnConfig{})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
	if game.CurrentState.Players[0].Coins != 1 {
		t.Fatalf("expected coins to increase")
	}
}

func TestRunTurnChallengeBluffCancels(t *testing.T) {
	game := &Game{CurrentState: state.GameState{
		TurnNumber:         1,
		CurrentPlayerIndex: 0,
		Players: []state.Player{
			{ID: 0, Name: "A", Coins: 2, Hand: []state.Card{{ID: 1, Role: models.TaxCollector}}},
			{ID: 1, Name: "B", Coins: 2, Hand: []state.Card{{ID: 2, Role: models.Thief}}},
		},
	}}
	cmd := &models.PlayerAction{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}
	provider := &stubProvider{
		action:     cmd,
		challenges: []ChallengeDecision{{ChallengerIndex: 1, Pass: false}},
		steps:      []any{[]int{0}},
	}
	res, err := game.RunTurn(context.Background(), provider, testClock{}, TurnConfig{})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if res.MainApplied {
		t.Fatalf("expected main action canceled")
	}
	if len(game.CurrentState.Players[0].Hand) != 0 {
		t.Fatalf("expected actor to lose a card")
	}
}

func TestRunTurnChallengeProvedAllowsAction(t *testing.T) {
	game := &Game{CurrentState: state.GameState{
		TurnNumber:         1,
		CurrentPlayerIndex: 0,
		Players: []state.Player{
			{ID: 0, Name: "A", Coins: 2, Hand: []state.Card{{ID: 1, Role: models.Thief}}},
			{ID: 1, Name: "B", Coins: 2, Hand: []state.Card{{ID: 2, Role: models.TaxCollector}}},
		},
	}}
	cmd := &models.PlayerAction{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}
	provider := &stubProvider{
		action:     cmd,
		challenges: []ChallengeDecision{{ChallengerIndex: 1, Pass: false}},
		steps:      []any{[]int{0}},
	}
	res, err := game.RunTurn(context.Background(), provider, testClock{}, TurnConfig{})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
	if game.CurrentState.Players[0].Coins != 4 {
		t.Fatalf("expected coins to increase")
	}
}

func TestRunTurnCounterCancelsMain(t *testing.T) {
	game := &Game{CurrentState: state.GameState{
		TurnNumber:         1,
		CurrentPlayerIndex: 0,
		Players: []state.Player{
			{ID: 0, Name: "A", Coins: 2, Hand: []state.Card{{ID: 1, Role: models.Thief}}},
			{ID: 1, Name: "B", Coins: 2, Hand: []state.Card{{ID: 2, Role: models.Thief}}},
		},
	}}
	cmd := &models.PlayerAction{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}
	counter := &models.PlayerAction{ID: models.BlockSteal, SourceIndex: 1}
	provider := &stubProvider{
		action: cmd,
		challenges: []ChallengeDecision{
			{ChallengerIndex: 1, Pass: true},
			{ChallengerIndex: 0, Pass: true},
		},
		counters: []CounterDecision{{PlayerIndex: 1, Command: counter}},
	}
	res, err := game.RunTurn(context.Background(), provider, testClock{}, TurnConfig{})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if !res.MainCanceled {
		t.Fatalf("expected main action canceled")
	}
}

func TestRunTurnExchangeStep(t *testing.T) {
	game := &Game{CurrentState: state.GameState{
		TurnNumber:         1,
		CurrentPlayerIndex: 0,
		Players: []state.Player{
			{ID: 0, Name: "A", Coins: 2, Hand: []state.Card{{ID: 1, Role: models.Politician}, {ID: 2, Role: models.TaxCollector}}},
			{ID: 1, Name: "B", Coins: 2, Hand: []state.Card{{ID: 3, Role: models.Thief}}},
		},
		Deck: state.Deck{{ID: 4, Role: models.Thief}, {ID: 5, Role: models.Policewoman}},
	}}
	cmd := &models.PlayerAction{ID: models.Exchange, SourceIndex: 0}
	provider := &stubProvider{
		action:     cmd,
		challenges: []ChallengeDecision{{ChallengerIndex: 1, Pass: true}},
		steps:      []any{[]int{0, 1}},
	}
	res, err := game.RunTurn(context.Background(), provider, testClock{}, TurnConfig{})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
}
