package state

// TurnState is the "Scratchpad" for the current turn.
//
// While GameState is the official record (what's on the board right now),
// TurnState is temporary working memory—like scribbling notes on a napkin
// during a conversation. It's not persisted to replays because it's only
// relevant while a turn is actively happening.
//
// Why this separation matters:
//   - GameState needs to be snapshotted for replays and history
//   - TurnState tracks things that only matter within the current turn's
//     execution context, like "how many times has the Businesswoman been taxed?"
//   - When the turn ends, we throw away the napkin (call Reset())
//
// Currently, the only thing we track here is Businesswoman taxation limits,
// but this pattern allows us to add per-turn counters without polluting the
// core game state serialization.
type TurnState struct {
	// TaxBusinessWomanCount tracks how many times the Businesswoman has been
	// taxed during this turn. The Tax action has a limit (4 times) to prevent
	// infinite coin generation scenarios.
	TaxBusinessWomanCount int
}

// Reset clears all per-turn counters, preparing for a fresh turn.
//
// This is called by IncrementTurn() in game.go after advancing to the next
// player. Think of it as crumpling up the napkin and grabbing a fresh one.
func (t *TurnState) Reset() {
	if t == nil {
		return
	}
	t.TaxBusinessWomanCount = 0
}
