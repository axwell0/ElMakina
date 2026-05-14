package actions

import (
	game "github.com/axwell0/elmakina/engine/core/state"
	"github.com/axwell0/elmakina/engine/core/types"
)

// EligibilityCriterion decides whether an action should be offered to a player
// before they try to declare it. The same interface covers main and counter
// actions; counter-specific data lives in ctx.Counter.
type EligibilityCriterion interface {
	IsEligible(ctx EligibilityContext) bool
}

// EligibilityContext is the read-only snapshot passed to every criterion.
// ActorIndex is the player whose options are being evaluated — i.e. the
// active player for mains, the responder for counters.
type EligibilityContext struct {
	GameState  *game.GameState
	ActorIndex int

	// Counter is non-nil only when evaluating counter-action eligibility.
	Counter *CounterEligibilityFacet
}

// CounterEligibilityFacet adds the pending-main context that counter
// criteria need to reason about (target, attacker, etc.).
type CounterEligibilityFacet struct {
	MainAction      *types.PlayerAction
	MainActorIndex  int
	MainTargetIndex *int
}

// CoinEligibilityCriterion requires the acting player to have at least the
// requested number of coins.
type CoinEligibilityCriterion struct {
	MinCoins int
}

func (c CoinEligibilityCriterion) IsEligible(ctx EligibilityContext) bool {
	if ctx.GameState == nil || ctx.ActorIndex < 0 || ctx.ActorIndex >= len(ctx.GameState.players) {
		return false
	}
	return ctx.GameState.players[ctx.ActorIndex].Coins >= c.MinCoins
}

// OpponentEligibilityCriterion requires at least one other non-eliminated
// player to exist. MinCoins optionally raises the bar for targetable actions
// like Steal.
type OpponentEligibilityCriterion struct {
	MinCoins int
}

func (c OpponentEligibilityCriterion) IsEligible(ctx EligibilityContext) bool {
	if ctx.GameState == nil || ctx.ActorIndex < 0 || ctx.ActorIndex >= len(ctx.GameState.players) {
		return false
	}
	for i, player := range ctx.GameState.players {
		if i == ctx.ActorIndex || len(player.Hand) == 0 {
			continue
		}
		if player.Coins >= c.MinCoins {
			return true
		}
	}
	return false
}

// ResponderCoinEligibilityCriterion requires the responder (i.e. the actor in
// a counter evaluation) to have at least MinCoins. It is equivalent to
// CoinEligibilityCriterion semantically but documents counter-only intent.
type ResponderCoinEligibilityCriterion struct {
	MinCoins int
}

func (c ResponderCoinEligibilityCriterion) IsEligible(ctx EligibilityContext) bool {
	if ctx.GameState == nil || ctx.ActorIndex < 0 || ctx.ActorIndex >= len(ctx.GameState.players) {
		return false
	}
	return ctx.GameState.players[ctx.ActorIndex].Coins >= c.MinCoins
}
