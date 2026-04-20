package session

import "testing"

func TestInteractionCoordinatorPublishesLatestInteraction(t *testing.T) {
	coordinator := NewInteractionCoordinator()

	state := InteractionState{
		ID:   "ix-1",
		Kind: InteractionChallenge,
		Recipients: RecipientSet{
			Players: []int{1},
		},
		Payload: ChallengeInteraction{
			ActorIndex: 0,
			Action:     "assassinate",
		},
	}

	coordinator.Open(state)

	current := coordinator.Current()
	if current == nil || current.ID != "ix-1" {
		t.Fatalf("expected current interaction to be published")
	}
}

func TestInteractionCoordinatorClearsOnlyMatchingInteraction(t *testing.T) {
	coordinator := NewInteractionCoordinator()
	coordinator.Open(InteractionState{ID: "ix-1", Kind: InteractionMainAction})
	coordinator.Open(InteractionState{ID: "ix-2", Kind: InteractionChallenge})

	coordinator.Clear("ix-1")

	current := coordinator.Current()
	if current == nil || current.ID != "ix-2" {
		t.Fatalf("expected stale clear to leave newer interaction intact")
	}
}
