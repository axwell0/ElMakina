package turn

import (
	"fmt"

	"github.com/axwell0/elmakina/engine/core/state"
	"github.com/axwell0/elmakina/engine/core/types"
	"github.com/axwell0/elmakina/engine/turn"
)

// declaredAction wraps a raw PlayerAction with turn-lifecycle bookkeeping.
//
// A PlayerAction is just the player's choice ("I want to Steal from player 2").
// declaredAction adds state that only matters during turn execution:
//   - Cost: how many coins were deducted at declaration time
//
// This separation keeps PlayerAction as a plain data type while the orchestration
// layer tracks mutable turn state here.
type declaredAction struct {
	Command *types.PlayerAction
	Cost    int
}

func (d *declaredAction) payCost(gs *state.GameState, logs *turn.TurnLog) error {
	if d == nil || d.Command == nil {
		return nil
	}
	if d.Cost > 0 {
		// Deduct coins from the actor's total. (pay)
		if _, err := gs.AddCoins(d.Command.ActorIndex, -d.Cost); err != nil {
			return fmt.Errorf("pay action cost: %w", err)
		}
		logs.Publicf("%s pays %d coins for %s", playerName(gs, d.Command.ActorIndex), d.Cost, d.Command.Action)
	}
	return nil
}

func (d *declaredAction) refundCost(gs *state.GameState, logs *turn.TurnLog) {
	if d == nil || gs == nil || logs == nil || d.Command == nil || d.Cost <= 0 {
		return
	}
	// Add back the coins to the actor's totalw
	if _, err := gs.AddCoins(d.Command.ActorIndex, d.Cost); err != nil {
		logs.Publicf("refund failed for %s: %s", playerName(gs, d.Command.ActorIndex), err)
		return
	}
	logs.Publicf("%s refunded %d coins for canceled %s", playerName(gs, d.Command.ActorIndex), d.Cost, d.Command.Action)
}
