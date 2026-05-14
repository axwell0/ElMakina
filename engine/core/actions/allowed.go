package actions

import (
	"github.com/axwell0/elmakina/engine/core/state"
	"github.com/axwell0/elmakina/engine/core/types"
)

// isEligible enforces that every non-nil criterion passes.
func isEligible(ctx EligibilityContext, criteria []EligibilityCriterion) bool {
	for _, criterion := range criteria {
		if criterion == nil {
			continue
		}
		if !criterion.IsEligible(ctx) {
			return false
		}
	}
	return true
}

// AllowedMainActions returns currently offerable main actions for actorIndex.
func AllowedMainActions(gs *state.GameState, actorIndex int) []types.ActionID {
	if gs == nil || actorIndex < 0 || actorIndex >= len(gs.players) {
		return nil
	}
	ids := MainActions()
	allowed := make([]types.ActionID, 0, len(ids))
	ctx := EligibilityContext{GameState: gs, ActorIndex: actorIndex}
	for _, id := range ids {
		def, ok := MainAction(id)
		if !ok {
			continue
		}
		if !isEligible(ctx, def.Eligibility) {
			continue
		}
		allowed = append(allowed, id)
	}
	return allowed
}

// AllowedCounterActions returns counters the responder may currently declare
// against the pending main.
func AllowedCounterActions(gs *state.GameState, main *types.PlayerAction, responderIndex int) []types.ActionID {
	if gs == nil || main == nil || responderIndex < 0 || responderIndex >= len(gs.players) {
		return nil
	}
	if responderIndex == main.ActorIndex {
		return nil
	}
	ctx := EligibilityContext{
		GameState:  gs,
		ActorIndex: responderIndex,
		Counter: &CounterEligibilityFacet{
			MainAction:      main,
			MainActorIndex:  main.ActorIndex,
			MainTargetIndex: targetIndexFromCounterMain(main),
		},
	}
	candidates := CountersForMain(main.Action)
	allowed := make([]types.ActionID, 0, len(candidates))
	for _, def := range candidates {
		if !isEligible(ctx, def.Eligibility) {
			continue
		}
		allowed = append(allowed, def.ID)
	}
	return allowed
}

func targetIndexFromCounterMain(action *types.PlayerAction) *int {
	if action == nil {
		return nil
	}
	switch payload := action.Payload.(type) {
	case types.TargetPayload:
		return &payload.TargetIndex
	case types.AccusePayload:
		return &payload.TargetIndex
	default:
		return nil
	}
}
