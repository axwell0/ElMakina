package tests

import (
	"ElMakina/backend/actions"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"context"
	"testing"
)

func makeCard(id int, role models.Role) state.Card {
	return state.Card{ID: id, Role: role}
}

func makeState(players ...state.Player) *state.GameState {
	return &state.GameState{
		TurnNumber:         1,
		Players:            players,
		Deck:               state.Deck{},
		CurrentPlayerIndex: 0,
	}
}

func applyWithResponse(t *testing.T, s *state.GameState, turn *state.TurnState, log *state.TurnLog, cmd *models.PlayerAction, assertReq func(state.StepRequest), response any) error {
	t.Helper()
	flow := actions.NewActionFlow()
	errCh := make(chan error, 1)
	go func() {
		errCh <- actions.ApplyAction(flow, s, turn, log, cmd)
	}()

	req := <-flow.Requests
	if assertReq != nil {
		assertReq(req)
	}
	flow.Responses <- response
	return <-errCh
}

func TestValidatePlayerAction(t *testing.T) {
	t.Parallel()
	baseState := makeState(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Businesswoman)}, Coins: 7},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 2},
	)

	tests := []struct {
		name    string
		state   *state.GameState
		cmd     *models.PlayerAction
		wantErr bool
	}{
		{name: "nil command", state: baseState, cmd: nil, wantErr: true},
		{name: "source out of range", state: baseState, cmd: &models.PlayerAction{ID: models.Income, SourceIndex: 99}, wantErr: true},
		{name: "unknown action", state: baseState, cmd: &models.PlayerAction{ID: models.ActionID("nope"), SourceIndex: 0}, wantErr: true},
		{name: "target payload required", state: baseState, cmd: &models.PlayerAction{ID: models.Steal, SourceIndex: 0}, wantErr: true},
		{name: "target out of range", state: baseState, cmd: &models.PlayerAction{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 99}}, wantErr: true},
		{name: "accuse payload required", state: baseState, cmd: &models.PlayerAction{ID: models.Accuse, SourceIndex: 0}, wantErr: true},
		{name: "payload not allowed", state: baseState, cmd: &models.PlayerAction{ID: models.Income, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}, wantErr: true},
		{name: "coup requires 7 coins", state: makeState(
			state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Businesswoman)}, Coins: 6},
			state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 2},
		), cmd: &models.PlayerAction{ID: models.Coup, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}, wantErr: true},
		{name: "steal requires 2 coins", state: makeState(
			state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Businesswoman)}, Coins: 2},
			state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 1},
		), cmd: &models.PlayerAction{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}, wantErr: true},
		{name: "tax business woman requires take4", state: baseState, cmd: &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: 1, MainAction: models.ActionID("nope")}, wantErr: true},
		{name: "tax requires taxable player", state: makeState(
			state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.TaxCollector)}, Coins: 6},
			state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 3},
		), cmd: &models.PlayerAction{ID: models.Tax, SourceIndex: 0}, wantErr: true},
		{name: "tax allowed when player has 7+ coins", state: makeState(
			state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.TaxCollector)}, Coins: 6},
			state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 7},
		), cmd: &models.PlayerAction{ID: models.Tax, SourceIndex: 0}, wantErr: false},
		{name: "valid take4 tax counter", state: baseState, cmd: &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: 1, MainAction: models.Business}, wantErr: false},
		{name: "block foreign aid requires block action", state: baseState, cmd: &models.PlayerAction{ID: models.BlockForeignAid, SourceIndex: 1, MainAction: models.ForeignAid}, wantErr: false},
		{name: "valid accuse", state: baseState, cmd: &models.PlayerAction{ID: models.Accuse, SourceIndex: 0, Payload: models.AccusePayload{TargetIndex: 1, Guess: models.Thief}}, wantErr: false},
		{name: "valid steal", state: baseState, cmd: &models.PlayerAction{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}, wantErr: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := actions.ValidatePlayerAction(tt.state, tt.cmd)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPayActionCost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cmd       *models.PlayerAction
		startCoin int
		wantCoin  int
		wantErr   bool
	}{
		{name: "coup cost", cmd: &models.PlayerAction{ID: models.Coup, SourceIndex: 0}, startCoin: 7, wantCoin: 0},
		{name: "assassinate cost", cmd: &models.PlayerAction{ID: models.Assassinate, SourceIndex: 0}, startCoin: 3, wantCoin: 0},
		{name: "escape cost", cmd: &models.PlayerAction{ID: models.Escape, SourceIndex: 0}, startCoin: 9, wantCoin: 0},
		{name: "accuse cost", cmd: &models.PlayerAction{ID: models.Accuse, SourceIndex: 0}, startCoin: 5, wantCoin: 1},
		{name: "assassinate insufficient coins", cmd: &models.PlayerAction{ID: models.Assassinate, SourceIndex: 0}, startCoin: 2, wantCoin: 2, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := makeState(state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Businesswoman)}, Coins: tt.startCoin})
			err := actions.PayActionCost(s, tt.cmd)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := s.Players[0].Coins; got != tt.wantCoin {
				t.Fatalf("coins: got %d want %d", got, tt.wantCoin)
			}
		})
	}
}

func TestAddCoinsInsufficientFunds(t *testing.T) {
	s := makeState(state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Businesswoman)}, Coins: 1})
	if _, err := s.AddCoins(0, -2); err == nil {
		t.Fatalf("expected error for insufficient coins")
	}
	if got := s.Players[0].Coins; got != 1 {
		t.Fatalf("coins unchanged: got %d want %d", got, 1)
	}
}

func TestApplyTake4AndTaxCounters(t *testing.T) {
	s := makeState(
		state.Player{ID: 0, Name: "Biz", Hand: []state.Card{makeCard(1, models.Businesswoman)}, Coins: 2},
		state.Player{ID: 1, Name: "Tax1", Hand: []state.Card{makeCard(2, models.TaxCollector)}, Coins: 2},
		state.Player{ID: 2, Name: "Tax2", Hand: []state.Card{makeCard(3, models.TaxCollector)}, Coins: 2},
		state.Player{ID: 3, Name: "Tax3", Hand: []state.Card{makeCard(4, models.TaxCollector)}, Coins: 2},
		state.Player{ID: 4, Name: "Tax4", Hand: []state.Card{makeCard(5, models.TaxCollector)}, Coins: 2},
		state.Player{ID: 5, Name: "Tax5", Hand: []state.Card{makeCard(6, models.TaxCollector)}, Coins: 2},
	)
	s.CurrentPlayerIndex = 0
	turn := &state.TurnState{}
	log := &state.TurnLog{}
	flow := actions.NewActionFlow()

	if err := actions.ApplyAction(flow, s, turn, log, &models.PlayerAction{ID: models.Business, SourceIndex: 0}); err != nil {
		t.Fatalf("take4 failed: %v", err)
	}
	if got := s.Players[0].Coins; got != 6 {
		t.Fatalf("take4 coins: got %d want %d", got, 6)
	}

	counters := []int{1, 2, 3, 4, 5}
	for _, idx := range counters {
		if err := actions.ApplyAction(flow, s, turn, log, &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: idx, MainAction: models.Business}); err != nil {
			t.Fatalf("tax counter failed: %v", err)
		}
	}
	if got := s.Players[0].Coins; got != 2 {
		t.Fatalf("tax counters capped: got %d want %d", got, 2)
	}
	if got := s.Players[1].Coins; got != 3 {
		t.Fatalf("first taxer coins: got %d want %d", got, 3)
	}
	if got := s.Players[4].Coins; got != 3 {
		t.Fatalf("fourth taxer coins: got %d want %d", got, 3)
	}
	if got := s.Players[5].Coins; got != 2 {
		t.Fatalf("fifth taxer should not gain: got %d want %d", got, 2)
	}
}

func TestApplyTaxBlocksForeignAid(t *testing.T) {
	s := makeState(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.TaxCollector)}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Businesswoman)}, Coins: 2},
	)
	s.CurrentPlayerIndex = 1
	turn := &state.TurnState{}
	log := &state.TurnLog{}
	flow := actions.NewActionFlow()

	before := s.Players[1].Coins
	if err := actions.ApplyAction(flow, s, turn, log, &models.PlayerAction{ID: models.BlockForeignAid, SourceIndex: 0, MainAction: models.ForeignAid}); err != nil {
		t.Fatalf("block foreign aid failed: %v", err)
	}
	if got := s.Players[1].Coins; got != before {
		t.Fatalf("foreign aid block should not change coins: got %d want %d", got, before)
	}
}

func TestApplyCoupDiscardsCard(t *testing.T) {
	s := makeState(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Businesswoman)}, Coins: 7},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief), makeCard(3, models.Colonel)}, Coins: 2},
	)
	log := &state.TurnLog{}
	turn := &state.TurnState{}
	cmd := &models.PlayerAction{ID: models.Coup, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}
	err := applyWithResponse(t, s, turn, log, cmd, func(req state.StepRequest) {
		if req.ActionID != models.Coup || req.Actor != 1 || req.Count != 1 || req.Context != state.ContextDiscard {
			t.Fatalf("unexpected request: %+v", req)
		}
	}, []int{0})
	if err != nil {
		t.Fatalf("apply coup error: %v", err)
	}
	if got := len(s.Players[1].Hand); got != 1 {
		t.Fatalf("target hand size: got %d want %d", got, 1)
	}
	if got := len(s.DrainDiscardEvents()); got != 1 {
		t.Fatalf("discard events: got %d want %d", got, 1)
	}
}

func TestApplyInvestigateLogsPrivate(t *testing.T) {
	s := makeState(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Policewoman)}, Coins: 2},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 2},
	)
	turn := &state.TurnState{}
	log := &state.TurnLog{}
	cmd := &models.PlayerAction{ID: models.Investigate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}

	flow := actions.NewActionFlow()
	err := actions.ApplyAction(flow, s, turn, log, cmd)
	if err != nil {
		t.Fatalf("apply investigate error: %v", err)
	}
	if got := len(log.Private[0]); got != 1 {
		t.Fatalf("private log count: got %d want %d", got, 1)
	}
}

func TestApplyAccuseCorrectAndWrong(t *testing.T) {
	t.Run("correct guess discards", func(t *testing.T) {
		s := makeState(
			state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Colonel)}, Coins: 2},
			state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 2},
		)
		turn := &state.TurnState{}
		log := &state.TurnLog{}
		flow := actions.NewActionFlow()
		cmd := &models.PlayerAction{ID: models.Accuse, SourceIndex: 0, Payload: models.AccusePayload{TargetIndex: 1, Guess: models.Thief}}
		if err := actions.ApplyAction(flow, s, turn, log, cmd); err != nil {
			t.Fatalf("apply accuse error: %v", err)
		}
		if got := len(s.Players[1].Hand); got != 0 {
			t.Fatalf("target hand size: got %d want %d", got, 0)
		}
	})

	t.Run("wrong guess grants coins", func(t *testing.T) {
		s := makeState(
			state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Colonel)}, Coins: 2},
			state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 2},
		)
		turn := &state.TurnState{}
		log := &state.TurnLog{}
		flow := actions.NewActionFlow()
		cmd := &models.PlayerAction{ID: models.Accuse, SourceIndex: 0, Payload: models.AccusePayload{TargetIndex: 1, Guess: models.Politician}}
		if err := actions.ApplyAction(flow, s, turn, log, cmd); err != nil {
			t.Fatalf("apply accuse error: %v", err)
		}
		if got := s.Players[1].Coins; got != 6 {
			t.Fatalf("target coins: got %d want %d", got, 6)
		}
	})
}

func TestApplyAssassinateRequestsDiscard(t *testing.T) {
	s := makeState(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Terrorist)}, Coins: 3},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief), makeCard(3, models.Colonel)}, Coins: 2},
	)
	turn := &state.TurnState{}
	log := &state.TurnLog{}
	cmd := &models.PlayerAction{ID: models.Assassinate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}
	err := applyWithResponse(t, s, turn, log, cmd, func(req state.StepRequest) {
		if req.ActionID != models.Assassinate || req.Actor != 1 || req.Count != 1 || req.Context != state.ContextDiscard {
			t.Fatalf("unexpected request: %+v", req)
		}
	}, []int{1})
	if err != nil {
		t.Fatalf("apply assassinate error: %v", err)
	}
	if got := len(s.Players[1].Hand); got != 1 {
		t.Fatalf("assassinate should discard: got %d want %d", got, 1)
	}
	if got := len(s.DrainDiscardEvents()); got != 1 {
		t.Fatalf("discard events: got %d want %d", got, 1)
	}
}

func TestCoupAndAssassinateTransferCoinsOnElimination(t *testing.T) {
	t.Run("coup transfers remaining coins", func(t *testing.T) {
		s := makeState(
			state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Businesswoman)}, Coins: 1},
			state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 4},
		)
		turn := &state.TurnState{}
		log := &state.TurnLog{}
		cmd := &models.PlayerAction{ID: models.Coup, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}
		err := applyWithResponse(t, s, turn, log, cmd, nil, []int{0})
		if err != nil {
			t.Fatalf("apply coup: %v", err)
		}
		if s.Players[0].Coins != 5 {
			t.Fatalf("source coins: got %d want 5", s.Players[0].Coins)
		}
		if s.Players[1].Coins != 0 {
			t.Fatalf("target coins not drained: got %d want 0", s.Players[1].Coins)
		}
	})

	t.Run("assassinate transfers remaining coins", func(t *testing.T) {
		s := makeState(
			state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Terrorist)}, Coins: 3},
			state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}, Coins: 6},
		)
		turn := &state.TurnState{}
		log := &state.TurnLog{}
		cmd := &models.PlayerAction{ID: models.Assassinate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}
		err := applyWithResponse(t, s, turn, log, cmd, nil, []int{0})
		if err != nil {
			t.Fatalf("apply assassinate: %v", err)
		}
		if s.Players[0].Coins != 9 {
			t.Fatalf("source coins: got %d want 9", s.Players[0].Coins)
		}
		if s.Players[1].Coins != 0 {
			t.Fatalf("target coins not drained: got %d want 0", s.Players[1].Coins)
		}
	})
}

func TestApplyStealTransfersTwoCoins(t *testing.T) {
	s := makeState(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Thief)}, Coins: 1},
		state.Player{ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Politician)}, Coins: 5},
	)
	turn := &state.TurnState{}
	log := &state.TurnLog{}
	flow := actions.NewActionFlow()
	cmd := &models.PlayerAction{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}

	if err := actions.ApplyAction(flow, s, turn, log, cmd); err != nil {
		t.Fatalf("apply steal error: %v", err)
	}
	if got := s.Players[0].Coins; got != 3 {
		t.Fatalf("source coins: got %d want %d", got, 3)
	}
	if got := s.Players[1].Coins; got != 3 {
		t.Fatalf("target coins: got %d want %d", got, 3)
	}
}

func TestApplyExchangeReturnsCards(t *testing.T) {
	s := makeState(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Politician), makeCard(2, models.Thief)}, Coins: 2},
	)
	s.Deck = state.Deck{makeCard(3, models.Colonel), makeCard(4, models.TaxCollector)}
	turn := &state.TurnState{}
	log := &state.TurnLog{}
	cmd := &models.PlayerAction{ID: models.Exchange, SourceIndex: 0}

	err := applyWithResponse(t, s, turn, log, cmd, func(req state.StepRequest) {
		if req.ActionID != models.Exchange || req.Actor != 0 || req.Count != 2 || req.Context != state.ContextExchange {
			t.Fatalf("unexpected request: %+v", req)
		}
	}, []int{0, 1})
	if err != nil {
		t.Fatalf("apply exchange error: %v", err)
	}
	if got := len(s.Players[0].Hand); got != 2 {
		t.Fatalf("hand size: got %d want %d", got, 2)
	}
	if got := len(s.Deck); got != 2 {
		t.Fatalf("deck size: got %d want %d", got, 2)
	}
}

func TestApplyEscape(t *testing.T) {
	s := makeState(state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Politician)}, Coins: 9})
	turn := &state.TurnState{}
	log := &state.TurnLog{}
	flow := actions.NewActionFlow()
	cmd := &models.PlayerAction{ID: models.Escape, SourceIndex: 0}

	if err := actions.ApplyAction(flow, s, turn, log, cmd); err != nil {
		t.Fatalf("apply escape error: %v", err)
	}
}

func TestActionFlowWithApplyAction(t *testing.T) {
	// Ensure ApplyAction works when driven via ActionFlow and Next loop.
	s := makeState(
		state.Player{ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.Politician), makeCard(2, models.Thief)}, Coins: 2},
	)
	s.Deck = state.Deck{makeCard(3, models.Colonel), makeCard(4, models.TaxCollector)}
	turn := &state.TurnState{}
	log := &state.TurnLog{}
	cmd := &models.PlayerAction{ID: models.Exchange, SourceIndex: 0}

	runner := actions.StartAction(context.Background(), func(f *actions.ActionFlow) error {
		return actions.ApplyAction(f, s, turn, log, cmd)
	})
	req := <-runner.Flow.Requests
	if req.ActionID != models.Exchange || req.Count != 2 {
		t.Fatalf("unexpected request: %+v", req)
	}
	if _, err := runner.Next([]int{0, 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
