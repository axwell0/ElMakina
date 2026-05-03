package types

type ActionPayload interface {
	isActionPayload()
}

// TargetPayload is used by actions that need a single target player index.
type TargetPayload struct {
	TargetIndex int // 0-indexed player position in GameState.Players.
}

// isActionPayload marks TargetPayload as a valid action payload variant.
func (tp TargetPayload) isActionPayload() {}

// AccusePayload is used by the Accuse action. It needs a target player and a
// role guess so the engine can compare the guess against the target's hand.
type AccusePayload struct {
	TargetIndex int  // 0-indexed player position in GameState.Players.
	Guess       Role // Role to check in target's hand
}

// isActionPayload marks AccusePayload as a valid action payload variant.
func (ap AccusePayload) isActionPayload() {}

// NoPayload is used by actions that don't require any additional data.
type NoPayload struct{}

// isActionPayload marks NoPayload as a valid action payload variant.
func (np NoPayload) isActionPayload() {}
