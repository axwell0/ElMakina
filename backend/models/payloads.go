package models

// Payload types for PlayerAction.Payload field. Types implement the Payload
// interface via marker methods for compile-time type safety.
// Player indices reference positions in GameState.Players slice.

// TargetPayload is used by actions that require a target player (Coup, Steal,
// Assassinate, Investigate). The engine validates the index bounds and that
// the target is active and not the source player.
type TargetPayload struct {
	TargetIndex int // 0-indexed player position in GameState.Players
}

func (TargetPayload) isPayload() {}

// AccusePayload is used by the Accuse action. Requires both a target player
// and a role guess. The engine checks if the target actually has the guessed
// role in their hand to determine the accusation outcome.
type AccusePayload struct {
	TargetIndex int  // 0-indexed player position in GameState.Players
	Guess       Role // Role to check in target's hand
}

func (AccusePayload) isPayload() {}

// NoPayload is used by actions that don't require any additional data (Income,
// ForeignAid, Tax, Exchange, etc).
type NoPayload struct{}

func (NoPayload) isPayload() {}
