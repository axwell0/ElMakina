package actions

import (
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"strings"
	"testing"
)

func TestApplyInvestigateRevealsRandomCard(t *testing.T) {
	prev := randIntn
	t.Cleanup(func() { randIntn = prev })
	randIntn = func(n int) int {
		if n != 2 {
			t.Fatalf("expected target hand size 2, got %d", n)
		}
		return 1
	}

	gs := &state.GameState{
		Players: []state.Player{
			{ID: 0, Name: "A", Hand: []state.Card{{ID: 1, Role: models.Businesswoman}}, Coins: 2},
			{ID: 1, Name: "B", Hand: []state.Card{
				{ID: 2, Role: models.Thief},
				{ID: 3, Role: models.Policewoman},
			}, Coins: 2},
		},
	}
	log := &state.TurnLog{}
	turn := &state.TurnState{}
	cmd := &models.PlayerAction{
		ID:          models.Investigate,
		SourceIndex: 0,
		Payload:     models.TargetPayload{TargetIndex: 1},
	}

	if err := applyInvestigate(nil, gs, turn, log, cmd); err != nil {
		t.Fatalf("applyInvestigate: %v", err)
	}
	if len(log.Private[0]) != 1 {
		t.Fatalf("expected 1 private log, got %d", len(log.Private[0]))
	}
	if !strings.Contains(log.Private[0][0], "Policewoman") {
		t.Fatalf("expected reveal of Policewoman, got %q", log.Private[0][0])
	}
	if len(log.Public) != 1 {
		t.Fatalf("expected 1 public log, got %d", len(log.Public))
	}
	if !strings.Contains(log.Public[0], "investigates") {
		t.Fatalf("expected investigate public log, got %q", log.Public[0])
	}
}
