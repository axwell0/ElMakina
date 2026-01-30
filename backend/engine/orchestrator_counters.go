package engine

import (
	"context"
	"fmt"
	"time"

	"ElMakina/backend/actions"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

// collectCounters gathers counter-actions from players who want to block the main action.
//
// After a role action survives challenges, targets get a chance to block it.
// This is different from challenges:
//   - Challenges question "Do you have that role?"
//   - Counters claim "I have the role that blocks you"
//
// THE COUNTER FLOW:
//  1. DETERMINE WHO CAN COUNTER: Which counter actions are valid for this main action?
//  2. NOTIFY ELIGIBLE PLAYERS: Ask them if they want to counter
//  3. COLLECT RESPONSES: Players either Pass or play a Counter
//  4. FIRST VALID COUNTER WINS: Usually the first counter action ends the window
//  5. EXCEPTION: MultiCounter actions allow multiple counters (e.g., TaxBusinessWoman)
//
// SPECIAL CASES:
//   - Escape: Can be played by anyone being targeted by Coup or Assassinate
//   - BlockSteal: Only the target of Steal can play this
//   - TaxBusinessWoman: Multiple TaxCollectors can tax the same Businesswoman
//
// Returns a list of counter actions and logs of what happened during collection.
func (g *Game) collectCounters(ctx context.Context, input InputProvider, clock Clock, timeout time.Duration, main *models.PlayerAction) ([]*models.PlayerAction, []string, error) {
	allowed := allowedCounters(main.ID)
	if len(allowed) == 0 {
		return nil, nil, nil
	}
	window := CounterWindow{
		MainAction:     main,
		AllowedActions: allowed,
		AllowedPlayers: allowedPlayers(&g.CurrentState, main.SourceIndex),
	}
	windowCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, err := input.CounterResponses(windowCtx, window)
	if err != nil {
		return nil, nil, err
	}
	timeoutCh := timerChan(clock, timeout)
	errCh := inputErrors(input)

	var counters []*models.PlayerAction
	var logs []string
	passed := make(map[int]bool, len(window.AllowedPlayers))
	responded := make(map[int]bool, len(window.AllowedPlayers))
	for {
		select {
		case <-ctx.Done():
			return counters, logs, ctx.Err()
		case err := <-errCh:
			if err != nil {
				return counters, logs, err
			}
		case <-timeoutCh:
			return counters, logs, nil
		case decision, ok := <-ch:
			if !ok {
				return counters, logs, nil
			}
			if decision.Pass {
				if !isAllowedPlayer(decision.PlayerIndex, window.AllowedPlayers) {
					return counters, logs, fmt.Errorf("counter player %d not allowed", decision.PlayerIndex)
				}
				if responded[decision.PlayerIndex] {
					return counters, logs, fmt.Errorf("counter player %d already responded", decision.PlayerIndex)
				}
				passed[decision.PlayerIndex] = true
				responded[decision.PlayerIndex] = true
				logs = append(logs, fmt.Sprintf("%s passes counter", g.CurrentState.Players[decision.PlayerIndex].Name))
				if len(responded) == len(window.AllowedPlayers) {
					return counters, logs, nil
				}
				continue
			}
			cmd := decision.Command
			if cmd == nil {
				return counters, logs, fmt.Errorf("counter action is nil")
			}
			if !isAllowedPlayer(cmd.SourceIndex, window.AllowedPlayers) {
				return counters, logs, fmt.Errorf("counter player %d not allowed", cmd.SourceIndex)
			}
			if responded[cmd.SourceIndex] {
				return counters, logs, fmt.Errorf("counter player %d already responded", cmd.SourceIndex)
			}
			if !isAllowedCounter(cmd.ID, allowed) {
				return counters, logs, fmt.Errorf("counter action %q not allowed", cmd.ID)
			}
			cmd.MainAction = main.ID
			if err := actions.ValidatePlayerAction(&g.CurrentState, cmd); err != nil {
				return counters, logs, err
			}
			counters = append(counters, cmd)
			responded[cmd.SourceIndex] = true
			if def, ok := actions.ActionRegistry[cmd.ID]; ok && !def.MultiCounter {
				return counters, logs, nil
			}
			if len(responded) == len(window.AllowedPlayers) {
				return counters, logs, nil
			}
		}
	}
}

// allowedCounters finds all counter actions that can block a given main action.
//
// We scan the ActionRegistry looking for actions where AllowedAgainst includes
// the main action ID. For example, if main is "steal", we find "block_steal".
//
// This is how the system knows which counters to offer players during the
// counter window—it's data-driven from the registry, not hardcoded.
func allowedCounters(main models.ActionID) []models.ActionID {
	var allowed []models.ActionID
	for id, def := range actions.ActionRegistry {
		for _, against := range def.AllowedAgainst {
			if against == main {
				allowed = append(allowed, id)
				break
			}
		}
	}
	return allowed
}

func isAllowedCounter(id models.ActionID, allowed []models.ActionID) bool {
	for _, a := range allowed {
		if a == id {
			return true
		}
	}
	return false
}

// allowedPlayers returns all active players except the one taking the action.
//
// For challenges and counters, we need to know who can respond. This filters:
//   - Exclude the actor (you can't challenge your own action)
//   - Exclude eliminated players (no cards = no influence = can't participate)
//
// This is used to build the list of eligible challengers/counterers.
func allowedPlayers(gs *state.GameState, exclude int) []int {
	if gs == nil {
		return nil
	}
	players := make([]int, 0, len(gs.Players))
	for i, p := range gs.Players {
		if i == exclude {
			continue
		}
		if len(p.Hand) == 0 {
			continue
		}
		players = append(players, i)
	}
	return players
}

func isAllowedPlayer(player int, allowed []int) bool {
	for _, a := range allowed {
		if a == player {
			return true
		}
	}
	return false
}

// counterCancelsMain determines if a successful counter completely stops the main action.
//
// Most counters DO cancel the main action—if you block a steal, the steal doesn't happen.
// However, TaxBusinessWoman is special: it reduces the Business from 4 coins to 3 coins,
// but doesn't completely stop the action. The Businesswoman still gets 3 coins.
//
// This function helps the orchestrator decide whether to apply the main action's
// effects after processing counters.
func counterCancelsMain(counter *models.PlayerAction) bool {
	if counter == nil {
		return false
	}
	// Tax against Business reduces the main action but does not cancel it.
	if counter.ID == models.TaxBusinessWoman && counter.MainAction == models.Business {
		return false
	}
	return true
}
