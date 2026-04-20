package session

import "testing"

func TestInteractionStateCarriesKindRecipientsAndPayload(t *testing.T) {
	state := InteractionState{
		ID:   "ix-1",
		Kind: InteractionMainAction,
		Recipients: RecipientSet{
			Players: []int{0},
		},
		Payload: MainActionInteraction{
			ActorIndex: 0,
			TurnNumber: 3,
			Title:      "Choose an action",
		},
	}

	if state.ID != "ix-1" {
		t.Fatalf("expected stable interaction ID")
	}
	if state.Kind != InteractionMainAction {
		t.Fatalf("expected main action kind")
	}
	if len(state.Recipients.Players) != 1 || state.Recipients.Players[0] != 0 {
		t.Fatalf("expected single recipient player 0")
	}
}

func TestCommandEnvelopeCarriesInteractionIdentity(t *testing.T) {
	command := CommandEnvelope{
		InteractionID: "ix-2",
		Kind:          CommandChallengeDecision,
		SessionID:     "room-1-player-1",
		Payload: ChallengeDecisionCommand{
			ChallengerIndex: 1,
			Pass:            false,
		},
	}

	if command.InteractionID != "ix-2" {
		t.Fatalf("expected interaction identity to be preserved")
	}
	if command.Kind != CommandChallengeDecision {
		t.Fatalf("expected challenge command kind")
	}
}
