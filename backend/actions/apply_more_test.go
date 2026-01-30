package actions

import (
	"testing"

	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

func TestApplyIncomeAndForeignAid(t *testing.T) {
	gs := &state.GameState{
		Players: []state.Player{{ID: 0, Name: "A", Coins: 0, Hand: []state.Card{{ID: 1, Role: models.TaxCollector}}}},
	}
	log := &state.TurnLog{}
	turn := &state.TurnState{}

	if err := applyIncome(nil, gs, turn, log, &models.PlayerAction{ID: models.Income, SourceIndex: 0}); err != nil {
		t.Fatalf("applyIncome: %v", err)
	}
	if gs.Players[0].Coins != 1 {
		t.Fatalf("expected 1 coin, got %d", gs.Players[0].Coins)
	}

	if err := applyForeignAid(nil, gs, turn, log, &models.PlayerAction{ID: models.ForeignAid, SourceIndex: 0}); err != nil {
		t.Fatalf("applyForeignAid: %v", err)
	}
	if gs.Players[0].Coins != 3 {
		t.Fatalf("expected 3 coins, got %d", gs.Players[0].Coins)
	}
}

func TestApplyTaxBusinessWomanLimitAndNormal(t *testing.T) {
	gs := &state.GameState{
		Players: []state.Player{
			{ID: 0, Name: "A", Coins: 7, Hand: []state.Card{{ID: 1, Role: models.TaxCollector}}},
			{ID: 1, Name: "B", Coins: 8, Hand: []state.Card{{ID: 2, Role: models.TaxCollector}}},
			{ID: 2, Name: "C", Coins: 6, Hand: []state.Card{{ID: 3, Role: models.TaxCollector}}},
		},
		CurrentPlayerIndex: 0,
	}
	turn := &state.TurnState{TaxBusinessWomanCount: 4}
	log := &state.TurnLog{}

	// TaxBusinessWoman at limit should no-op.
	if err := applyTaxBusinessWoman(nil, gs, turn, log, &models.PlayerAction{ID: models.TaxBusinessWoman, SourceIndex: 0}); err != nil {
		t.Fatalf("applyTaxBusinessWoman: %v", err)
	}
	if gs.Players[0].Coins != 7 || gs.Players[1].Coins != 8 {
		t.Fatalf("unexpected coins after businesswoman tax")
	}

	// Normal tax should charge 7+ coin players.
	turn.TaxBusinessWomanCount = 0
	if err := applyTax(nil, gs, turn, log, &models.PlayerAction{ID: models.Tax, SourceIndex: 0}); err != nil {
		t.Fatalf("applyTax normal: %v", err)
	}
	if gs.Players[0].Coins != 6 || gs.Players[1].Coins != 7 {
		t.Fatalf("expected tax to reduce coins for 7+ players")
	}
	if gs.Players[2].Coins != 6 {
		t.Fatalf("player with <7 coins should not be taxed")
	}
}

func TestApplyCoupAndAssassinate(t *testing.T) {
	gs := &state.GameState{
		Players: []state.Player{
			{ID: 0, Name: "A", Coins: 7, Hand: []state.Card{{ID: 1, Role: models.TaxCollector}}},
			{ID: 1, Name: "B", Coins: 2, Hand: []state.Card{{ID: 2, Role: models.Thief}, {ID: 3, Role: models.Policewoman}}},
		},
		Deck: state.Deck{{ID: 4, Role: models.Colonel}},
	}
	log := &state.TurnLog{}
	turn := &state.TurnState{}

	if err := runWithCardSelection(func(flow *ActionFlow) error {
		return applyCoup(flow, gs, turn, log, &models.PlayerAction{ID: models.Coup, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}})
	}, []int{0}); err != nil {
		t.Fatalf("applyCoup: %v", err)
	}
	if len(gs.Players[1].Hand) != 1 {
		t.Fatalf("expected 1 card after coup, got %d", len(gs.Players[1].Hand))
	}

	if err := runWithCardSelection(func(flow *ActionFlow) error {
		return applyAssassinate(flow, gs, turn, log, &models.PlayerAction{ID: models.Assassinate, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}})
	}, []int{0}); err != nil {
		t.Fatalf("applyAssassinate: %v", err)
	}
	if len(gs.Players[1].Hand) != 0 {
		t.Fatalf("expected 0 cards after assassinate, got %d", len(gs.Players[1].Hand))
	}
}

func TestApplyStealAndAccuse(t *testing.T) {
	gs := &state.GameState{
		Players: []state.Player{
			{ID: 0, Name: "A", Coins: 2, Hand: []state.Card{{ID: 1, Role: models.Thief}}},
			{ID: 1, Name: "B", Coins: 2, Hand: []state.Card{{ID: 2, Role: models.Colonel}, {ID: 3, Role: models.TaxCollector}}},
		},
	}
	log := &state.TurnLog{}
	turn := &state.TurnState{}

	if err := applySteal(nil, gs, turn, log, &models.PlayerAction{ID: models.Steal, SourceIndex: 0, Payload: models.TargetPayload{TargetIndex: 1}}); err != nil {
		t.Fatalf("applySteal: %v", err)
	}
	if gs.Players[0].Coins != 4 || gs.Players[1].Coins != 0 {
		t.Fatalf("unexpected coins after steal")
	}

	// Correct accuse discards a matching card.
	if err := applyAccuse(nil, gs, turn, log, &models.PlayerAction{ID: models.Accuse, SourceIndex: 0, Payload: models.AccusePayload{TargetIndex: 1, Guess: models.Colonel}}); err != nil {
		t.Fatalf("applyAccuse correct: %v", err)
	}
	if len(gs.Players[1].Hand) != 1 {
		t.Fatalf("expected 1 card after correct accuse, got %d", len(gs.Players[1].Hand))
	}

	// Wrong accuse gives target coins.
	if err := applyAccuse(nil, gs, turn, log, &models.PlayerAction{ID: models.Accuse, SourceIndex: 0, Payload: models.AccusePayload{TargetIndex: 1, Guess: models.Thief}}); err != nil {
		t.Fatalf("applyAccuse wrong: %v", err)
	}
	if gs.Players[1].Coins != 4 {
		t.Fatalf("expected 4 coins after wrong accuse, got %d", gs.Players[1].Coins)
	}
}

func TestApplyExchangeEscapePassBlock(t *testing.T) {
	gs := &state.GameState{
		Players: []state.Player{{ID: 0, Name: "A", Coins: 10, Hand: []state.Card{{ID: 1, Role: models.TaxCollector}, {ID: 2, Role: models.Thief}}}},
		Deck:    state.Deck{{ID: 3, Role: models.Policewoman}, {ID: 4, Role: models.Colonel}},
	}
	log := &state.TurnLog{}
	turn := &state.TurnState{}

	if err := runWithCardSelection(func(flow *ActionFlow) error {
		return applyExchange(flow, gs, turn, log, &models.PlayerAction{ID: models.Exchange, SourceIndex: 0})
	}, []int{0, 1}); err != nil {
		t.Fatalf("applyExchange: %v", err)
	}
	if len(gs.Players[0].Hand) == 0 {
		t.Fatalf("expected hand to retain cards after exchange")
	}

	if err := applyEscape(nil, gs, turn, log, &models.PlayerAction{ID: models.Escape, SourceIndex: 0}); err != nil {
		t.Fatalf("applyEscape: %v", err)
	}
	if err := applyPass(nil, gs, turn, log, &models.PlayerAction{ID: models.Pass, SourceIndex: 0}); err != nil {
		t.Fatalf("applyPass: %v", err)
	}
	if err := applyBlock(nil, gs, turn, log, &models.PlayerAction{ID: models.BlockSteal, SourceIndex: 0, MainAction: models.Steal}); err != nil {
		t.Fatalf("applyBlock: %v", err)
	}
}

func runWithCardSelection(fn func(flow *ActionFlow) error, indices []int) error {
	flow := NewActionFlow()
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn(flow)
	}()

	req, ok := <-flow.Requests
	if ok {
		_ = req
		flow.Responses <- indices
	}
	return <-errCh
}
