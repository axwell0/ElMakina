package tests

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type scriptedInput struct {
	actions    []*models.PlayerAction
	challenges [][]engine.ChallengeDecision
	counters   [][]engine.CounterDecision
	steps      []any
}

func (s *scriptedInput) RequestAction(ctx context.Context, actor int, gs *state.GameState) (*models.PlayerAction, error) {
	if len(s.actions) == 0 {
		return nil, errors.New("no actions queued")
	}
	cmd := s.actions[0]
	s.actions = s.actions[1:]
	return cmd, nil
}

func (s *scriptedInput) ChallengeResponses(ctx context.Context, window engine.ChallengeWindow) (<-chan engine.ChallengeDecision, error) {
	var items []engine.ChallengeDecision
	if len(s.challenges) > 0 {
		items = s.challenges[0]
		s.challenges = s.challenges[1:]
	}
	ch := make(chan engine.ChallengeDecision, len(items))
	for _, item := range items {
		ch <- item
	}
	close(ch)
	return ch, nil
}

func (s *scriptedInput) CounterResponses(ctx context.Context, window engine.CounterWindow) (<-chan engine.CounterDecision, error) {
	var items []engine.CounterDecision
	if len(s.counters) > 0 {
		items = s.counters[0]
		s.counters = s.counters[1:]
	}
	ch := make(chan engine.CounterDecision, len(items))
	for _, item := range items {
		ch <- item
	}
	close(ch)
	return ch, nil
}

func (s *scriptedInput) RequestStep(ctx context.Context, req state.StepRequest) (any, error) {
	if len(s.steps) == 0 {
		return nil, errors.New("no step responses queued")
	}
	resp := s.steps[0]
	s.steps = s.steps[1:]
	return resp, nil
}

type manualClock struct {
	mu      sync.Mutex
	timers  []chan time.Time
	created chan struct{}
}

func newManualClock() *manualClock {
	return &manualClock{created: make(chan struct{}, 16)}
}

func (c *manualClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	c.mu.Lock()
	c.timers = append(c.timers, ch)
	c.mu.Unlock()
	c.created <- struct{}{}
	return ch
}

func (c *manualClock) FireNext() {
	c.mu.Lock()
	if len(c.timers) == 0 {
		c.mu.Unlock()
		return
	}
	ch := c.timers[0]
	c.timers = c.timers[1:]
	c.mu.Unlock()
	ch <- time.Now()
}

func (c *manualClock) WaitNextTimer() {
	<-c.created
}

type blockingChallengesInput struct {
	action *models.PlayerAction
	steps  []any
}

func (b *blockingChallengesInput) RequestAction(ctx context.Context, actor int, gs *state.GameState) (*models.PlayerAction, error) {
	return b.action, nil
}

func (b *blockingChallengesInput) ChallengeResponses(ctx context.Context, window engine.ChallengeWindow) (<-chan engine.ChallengeDecision, error) {
	ch := make(chan engine.ChallengeDecision)
	return ch, nil
}

func (b *blockingChallengesInput) CounterResponses(ctx context.Context, window engine.CounterWindow) (<-chan engine.CounterDecision, error) {
	ch := make(chan engine.CounterDecision)
	close(ch)
	return ch, nil
}

func (b *blockingChallengesInput) RequestStep(ctx context.Context, req state.StepRequest) (any, error) {
	if len(b.steps) == 0 {
		return nil, errors.New("no step responses queued")
	}
	resp := b.steps[0]
	b.steps = b.steps[1:]
	return resp, nil
}

func newGameWithPlayers(players ...state.Player) *engine.Game {
	return &engine.Game{
		CurrentState: state.GameState{
			TurnNumber:         1,
			Players:            players,
			CurrentPlayerIndex: 0,
		},
	}
}

func TestRunTurnIncome_NoChallengeNoCounters(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Income, SourceIndex: 0},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
	if got := g.CurrentState.Players[0].Coins; got != 3 {
		t.Fatalf("coins: got %d want %d", got, 3)
	}
}

func TestRunTurnMainChallengeFailsActorProves(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Policewoman}, {ID: 3, Role: models.Thief}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Businesswoman}}, Coins: 2},
		state.Player{ID: 2, Name: "C", Hand: []state.Card{{ID: 4, Role: models.Politician}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Investigate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		challenges: [][]engine.ChallengeDecision{
			{{
				ChallengerIndex:        2,
				ActorProvingCardIndex:  0,
				ChallengerDiscardIndex: 0,
			}},
		},
		steps: []any{
			[]int{0},
			[]int{0},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected action applied")
	}
	if got := len(g.CurrentState.Players[0].Hand); got != 2 {
		t.Fatalf("actor hand: got %d want %d", got, 2)
	}
	if got := len(g.CurrentState.Players[2].Hand); got != 0 {
		t.Fatalf("challenger hand: got %d want %d", got, 0)
	}
	if got := len(res.ChallengeResults); got != 1 {
		t.Fatalf("challenge results: got %d want %d", got, 1)
	}
}

func TestRunTurnMainChallengeSucceedsActionCanceled(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}}, Coins: 4},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		challenges: [][]engine.ChallengeDecision{
			{{
				ChallengerIndex:   1,
				ActorDiscardIndex: 0,
			}},
		},
		steps: []any{
			[]int{0},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if res.MainApplied {
		t.Fatalf("expected action canceled")
	}
	if got := g.CurrentState.Players[0].Coins; got != 2 {
		t.Fatalf("actor coins: got %d want %d", got, 2)
	}
	if got := g.CurrentState.Players[1].Coins; got != 4 {
		t.Fatalf("target coins: got %d want %d", got, 4)
	}
	if got := len(g.CurrentState.Players[0].Hand); got != 0 {
		t.Fatalf("actor hand: got %d want %d", got, 0)
	}
}

func TestRunTurnCounterBlocksForeignAid(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.TaxCollector}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.ForeignAid, SourceIndex: 0},
		},
		counters: [][]engine.CounterDecision{
			{{Command: &models.PlayerAction{ID: models.BlockForeignAid, SourceIndex: 1}}},
		},
		challenges: [][]engine.ChallengeDecision{
			{},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if res.MainApplied {
		t.Fatalf("expected main action canceled by counter")
	}
	if got := g.CurrentState.Players[0].Coins; got != 2 {
		t.Fatalf("coins: got %d want %d", got, 2)
	}
	if got := len(res.CounterResults); got != 1 {
		t.Fatalf("counter results: got %d want %d", got, 1)
	}
}

func TestRunTurnTaxCountersTake4(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "Biz", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 2},
		state.Player{ID: 1, Name: "T1", Hand: []state.Card{{ID: 2, Role: models.TaxCollector}}, Coins: 2},
		state.Player{ID: 2, Name: "T2", Hand: []state.Card{{ID: 3, Role: models.TaxCollector}}, Coins: 2},
		state.Player{ID: 3, Name: "T3", Hand: []state.Card{{ID: 4, Role: models.TaxCollector}}, Coins: 2},
		state.Player{ID: 4, Name: "T4", Hand: []state.Card{{ID: 5, Role: models.TaxCollector}}, Coins: 2},
		state.Player{ID: 5, Name: "T5", Hand: []state.Card{{ID: 6, Role: models.TaxCollector}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Business, SourceIndex: 0},
		},
		counters: [][]engine.CounterDecision{
			{
				{Command: &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: 1}},
				{Command: &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: 2}},
				{Command: &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: 3}},
				{Command: &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: 4}},
				{Command: &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: 5}},
			},
		},
		challenges: [][]engine.ChallengeDecision{
			{}, {}, {}, {}, {},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
	if got := g.CurrentState.Players[0].Coins; got != 2 {
		t.Fatalf("coins after tax: got %d want %d", got, 2)
	}
}

func TestRunTurnCounterChallengeSucceedsMainProceeds(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Thief}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Businesswoman}}, Coins: 2},
		state.Player{ID: 2, Name: "C", Hand: []state.Card{{ID: 3, Role: models.Politician}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		counters: [][]engine.CounterDecision{
			{{Command: &models.PlayerAction{ID: models.BlockSteal, SourceIndex: 1}}},
		},
		challenges: [][]engine.ChallengeDecision{
			{}, // no challenge to main action
			{{
				ChallengerIndex:   2,
				ActorDiscardIndex: 0,
			}},
		},
		steps: []any{
			[]int{0},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied after counter fails")
	}
	if got := g.CurrentState.Players[0].Coins; got != 4 {
		t.Fatalf("actor coins: got %d want %d", got, 4)
	}
	if got := g.CurrentState.Players[1].Coins; got != 0 {
		t.Fatalf("target coins: got %d want %d", got, 0)
	}
	if got := len(res.CounterResults); got != 1 {
		t.Fatalf("counter results: got %d want %d", got, 1)
	}
	if res.CounterResults[0].Applied {
		t.Fatalf("expected counter not applied")
	}
}

func TestRunTurnCoupStep(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 7},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}, {ID: 3, Role: models.Colonel}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Coup, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		steps: []any{
			[]int{0},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
	if got := len(g.CurrentState.Players[1].Hand); got != 1 {
		t.Fatalf("target hand: got %d want %d", got, 1)
	}
}

func TestRunTurnChallengeTimeoutAllowsAction(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Policewoman}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}}, Coins: 2},
	)

	clock := newManualClock()
	input := &blockingChallengesInput{
		action: &models.PlayerAction{ID: models.Investigate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		steps:  []any{[]int{0}},
	}

	resCh := make(chan *engine.TurnResult, 1)
	errCh := make(chan error, 1)
	go func() {
		res, err := g.RunTurn(context.Background(), input, clock, engine.TurnConfig{
			ChallengeTimeout: time.Second,
			CounterTimeout:   time.Second,
		})
		resCh <- res
		errCh <- err
	}()

	clock.WaitNextTimer()
	clock.FireNext()

	res := <-resCh
	err := <-errCh
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected action applied after timeout")
	}
}

func TestRunTurnAccuseIncorrect(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Colonel}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Accuse, SourceIndex: 0, Payload: models.AccusePayload{TargetIndex: 1, Guess: models.Thief}},
		},
		challenges: [][]engine.ChallengeDecision{
			{},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if res.MainApplied {
		t.Fatalf("expected main action not applied")
	}

	if err == nil {
		t.Fatalf("run turn error: %v", err)
	}
	if got := len(g.CurrentState.Players[1].Hand); got == 0 {
		t.Fatalf("target hand: got %d want %d", got, 0)
	}
}

func TestRunTurnAssassinateStep(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Terrorist}}, Coins: 3},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}, {ID: 3, Role: models.Politician}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Assassinate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		challenges: [][]engine.ChallengeDecision{
			{},
		},
		steps: []any{
			[]int{1},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
	if got := len(g.CurrentState.Players[1].Hand); got != 1 {
		t.Fatalf("target hand: got %d want %d", got, 1)
	}
}

func TestRunTurnExchangeStep(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Politician}, {ID: 2, Role: models.Thief}}, Coins: 2},
	)
	g.CurrentState.Deck = state.Deck{
		{ID: 3, Role: models.Colonel},
		{ID: 4, Role: models.TaxCollector},
	}

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Exchange, SourceIndex: 0},
		},
		challenges: [][]engine.ChallengeDecision{
			{},
		},
		steps: []any{
			[]int{0, 1},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
	if got := len(g.CurrentState.Players[0].Hand); got != 2 {
		t.Fatalf("hand size: got %d want %d", got, 2)
	}
}

func TestRunTurnTaxMainGlobal(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.TaxCollector}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}}, Coins: 7},
		state.Player{ID: 2, Name: "C", Hand: []state.Card{{ID: 3, Role: models.Politician}}, Coins: 8},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Tax, SourceIndex: 0},
		},
		challenges: [][]engine.ChallengeDecision{
			{},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if !res.MainApplied {
		t.Fatalf("expected main action applied")
	}
	if got := g.CurrentState.Players[1].Coins; got != 6 {
		t.Fatalf("player1 coins: got %d want %d", got, 6)
	}
	if got := g.CurrentState.Players[2].Coins; got != 7 {
		t.Fatalf("player2 coins: got %d want %d", got, 7)
	}
}

func TestRunTurnEscapeCounterCancelsCoup(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 7},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Politician}}, Coins: 9},
		state.Player{ID: 2, Name: "C", Hand: []state.Card{{ID: 3, Role: models.Thief}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Coup, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		counters: [][]engine.CounterDecision{
			{{Command: &models.PlayerAction{ID: models.Escape, SourceIndex: 1}}},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if res.MainApplied {
		t.Fatalf("expected coup canceled by escape")
	}
	if got := len(g.CurrentState.Players[1].Hand); got != 1 {
		t.Fatalf("target hand should be unchanged: got %d want %d", got, 1)
	}
}

func TestRunTurnChallengeRefundsCost(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 3},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Assassinate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		challenges: [][]engine.ChallengeDecision{
			{{
				ChallengerIndex:   1,
				ActorDiscardIndex: 0,
			}},
		},
		steps: []any{
			[]int{0},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if res.MainApplied {
		t.Fatalf("expected main action canceled")
	}
	if got := g.CurrentState.Players[0].Coins; got != 3 {
		t.Fatalf("refund expected: got %d want %d", got, 3)
	}
}

func TestRunTurnInvalidCounterFromActor(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Thief}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Businesswoman}}, Coins: 4},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		challenges: [][]engine.ChallengeDecision{
			{},
		},
		counters: [][]engine.CounterDecision{
			{{Command: &models.PlayerAction{ID: models.BlockSteal, SourceIndex: 0}}},
		},
	}

	_, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err == nil {
		t.Fatalf("expected error for actor counter")
	}
}

func TestRunTurnInvalidCounterActionNotAllowed(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Thief}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Businesswoman}}, Coins: 4},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		challenges: [][]engine.ChallengeDecision{
			{},
		},
		counters: [][]engine.CounterDecision{
			{{Command: &models.PlayerAction{ID: models.BlockForeignAid, SourceIndex: 1}}},
		},
	}

	_, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err == nil {
		t.Fatalf("expected error for invalid counter action")
	}
}

func TestRunTurnPlayerEliminatedMidTurn(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		},
		challenges: [][]engine.ChallengeDecision{
			{{
				ChallengerIndex: 1,
			}},
		},
		steps: []any{
			[]int{0},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if res.MainApplied {
		t.Fatalf("expected main action canceled by challenge")
	}
	if got := len(g.CurrentState.Players[0].Hand); got != 0 {
		t.Fatalf("actor should be eliminated: got hand size %d want 0", got)
	}
}

func TestRunTurnCancelingCounterStopsProcessing(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.TaxCollector}}, Coins: 2},
		state.Player{ID: 2, Name: "C", Hand: []state.Card{{ID: 3, Role: models.TaxCollector}}, Coins: 2},
	)

	input := &scriptedInput{
		actions: []*models.PlayerAction{
			{ID: models.ForeignAid, SourceIndex: 0},
		},
		counters: [][]engine.CounterDecision{
			{
				{Command: &models.PlayerAction{ID: models.BlockForeignAid, SourceIndex: 1}},
				{Command: &models.PlayerAction{ID: models.BlockForeignAid, SourceIndex: 2}},
			},
		},
		challenges: [][]engine.ChallengeDecision{
			{},
		},
	}

	res, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("run turn error: %v", err)
	}
	if res.MainApplied {
		t.Fatalf("expected main action canceled")
	}
	if got := len(res.CounterResults); got != 1 {
		t.Fatalf("counter results: got %d want %d", got, 1)
	}
	if res.CounterResults[0].Action.ID != models.BlockForeignAid {
		t.Fatalf("expected block foreign aid to apply")
	}
}

type erroringProvider struct {
	action *models.PlayerAction
	errCh  chan error
}

func (p *erroringProvider) Errors() <-chan error { return p.errCh }

func (p *erroringProvider) RequestAction(ctx context.Context, actor int, gs *state.GameState) (*models.PlayerAction, error) {
	return p.action, nil
}

func (p *erroringProvider) ChallengeResponses(ctx context.Context, window engine.ChallengeWindow) (<-chan engine.ChallengeDecision, error) {
	ch := make(chan engine.ChallengeDecision)
	return ch, nil
}

func (p *erroringProvider) CounterResponses(ctx context.Context, window engine.CounterWindow) (<-chan engine.CounterDecision, error) {
	ch := make(chan engine.CounterDecision)
	close(ch)
	return ch, nil
}

func (p *erroringProvider) RequestStep(ctx context.Context, req state.StepRequest) (any, error) {
	return nil, nil
}

func TestRunTurnProviderErrorAborts(t *testing.T) {
	g := newGameWithPlayers(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Policewoman}}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{{ID: 2, Role: models.Thief}}, Coins: 2},
	)

	errCh := make(chan error, 1)
	errCh <- errors.New("disconnect")

	input := &erroringProvider{
		action: &models.PlayerAction{ID: models.Investigate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}},
		errCh:  errCh,
	}

	_, err := g.RunTurn(context.Background(), input, engine.RealClock{}, engine.TurnConfig{
		ChallengeTimeout: time.Second,
		CounterTimeout:   time.Second,
	})
	if err == nil {
		t.Fatalf("expected error on provider failure")
	}
}
