package engine

import (
	"context"
	"fmt"
	"strings"

	"ElMakina/backend/actions"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

// RunTurn is the "Brain" of the game loop. It handles the complex dance between players.
//
// If Game is the Kernel, RunTurn is the Process Scheduler. It coordinates a single
// atomic "Turn" which, in this game, is actually a multi-phase conversation:
//
//  1. SOLICITATION: We ask the actor "What do you want to do?". This blocks until we get an Input.
//  2. VALIDATION & COST: We check if they have the coins. If so, they PAY IMMEDIATELY.
//     (Refunds happen later if the action is canceled).
//  3. CHALLENGE WINDOW: If the action involves a role (like taxing), everyone else
//     gets a chance to call "Liar!". If they lose the challenge, the action is dead.
//  4. COUNTER WINDOW: If the action is offensive (like stealing), the target can
//     try to block it. Blocks themselves can be challenged!
//  5. APPLICATION: Only if the action survives the gauntlet above do we actually
//     apply its effects to the GameState.
//
// Architecture Note: InputProvider gets the live GameState and must treat it as read-only.
//
// See: orchestrator_types.go:24 for InputProvider interface definition.
// See: actions/validators.go:9 for validation logic.
// See: actions/apply.go:101 for action execution.
// See: orchestrator_counters.go:32 for counter collection.
// See: server/ws/session_runner.go:241 for WebSocket integration.
func (g *Game) RunTurn(ctx context.Context, input InputProvider, clock Clock, cfg TurnConfig) (*TurnResult, error) {
	if g == nil {
		return nil, fmt.Errorf("game is nil")
	}
	if input == nil {
		return nil, fmt.Errorf("input provider is nil")
	}
	if clock == nil {
		return nil, fmt.Errorf("clock is nil")
	}

	gs := &g.CurrentState
	actorIndex := gs.CurrentPlayerIndex
	log := &state.TurnLog{}
	result := &TurnResult{
		PrivateLogs: make(map[int][]string),
	}
	result.Logs = append(result.Logs, fmt.Sprintf("turn %d begins: %s", gs.TurnNumber, gs.Players[actorIndex].Name))
	for _, player := range gs.Players {
		if len(player.Hand) == 0 {
			continue
		}
		result.Logs = append(result.Logs, fmt.Sprintf("%s has %d coins (%d card(s))", player.Name, player.Coins, len(player.Hand)))
	}

	mainCmd, err := input.RequestAction(ctx, actorIndex, gs)
	if err != nil {
		return result, err
	}
	if mainCmd == nil {
		return result, fmt.Errorf("main action is nil")
	}
	if mainCmd.SourceIndex != actorIndex {
		return result, fmt.Errorf("main action source %d does not match current player %d", mainCmd.SourceIndex, actorIndex)
	}
	result.Main = mainCmd
	result.Logs = append(result.Logs, fmt.Sprintf("%s declares %s", gs.Players[actorIndex].Name, mainCmd.ID))

	if err := actions.ValidatePlayerAction(gs, mainCmd); err != nil {
		return result, err
	}
	mainCost := actions.ActionCost(mainCmd.ID)
	if err := actions.PayActionCost(gs, mainCmd); err != nil {
		return result, err
	}
	if mainCost > 0 {
		result.Logs = append(result.Logs, fmt.Sprintf("%s pays %d coins for %s", gs.Players[actorIndex].Name, mainCost, mainCmd.ID))
	}

	mainAllowed := true
	if role := actionRole(mainCmd.ID); role != "" {
		result.Logs = append(result.Logs, fmt.Sprintf("challenge window opened for %s (%s)", mainCmd.ID, role))
		challengeRes, err := g.runChallengeWindow(ctx, input, clock, cfg.ChallengeTimeout, ChallengeMain, mainCmd, role)
		result.ChallengeResults = append(result.ChallengeResults, challengeRes)
		result.Logs = append(result.Logs, challengeRes.Logs...)
		if err != nil {
			return result, err
		}
		mainAllowed = challengeRes.ActionAllowed
		if !mainAllowed && mainCost > 0 {
			_, _ = gs.AddCoins(mainCmd.SourceIndex, mainCost)
			result.Logs = append(result.Logs, fmt.Sprintf("%s refunded %d coins for canceled %s", gs.Players[actorIndex].Name, mainCost, mainCmd.ID))
		}
	}

	var counters []*models.PlayerAction
	if mainAllowed {
		result.Logs = append(result.Logs, fmt.Sprintf("counter window opened for %s", mainCmd.ID))
		var counterLogs []string
		counters, counterLogs, err = g.collectCounters(ctx, input, clock, cfg.CounterTimeout, mainCmd)
		if err != nil {
			return result, err
		}
		result.Logs = append(result.Logs, counterLogs...)
	}

	mainCanceled := false
	type pendingPostCounter struct {
		cmd   *models.PlayerAction
		index int
	}
	var postCounters []pendingPostCounter

	for _, counter := range counters {
		counterResult := CounterResult{Action: counter}
		counterCost := actions.ActionCost(counter.ID)
		if err := actions.PayActionCost(gs, counter); err != nil {
			return result, err
		}
		if counterCost > 0 {
			result.Logs = append(result.Logs, fmt.Sprintf("%s pays %d coins for %s", gs.Players[counter.SourceIndex].Name, counterCost, counter.ID))
		}

		if role := actionRole(counter.ID); role != "" {
			result.Logs = append(result.Logs, fmt.Sprintf("challenge window opened for %s (%s)", counter.ID, role))
			challengeRes, err := g.runChallengeWindow(ctx, input, clock, cfg.ChallengeTimeout, ChallengeCounter, counter, role)
			result.ChallengeResults = append(result.ChallengeResults, challengeRes)
			result.Logs = append(result.Logs, challengeRes.Logs...)
			if err != nil {
				return result, err
			}
			counterResult.Challenge = &challengeRes
			if !challengeRes.ActionAllowed {
				if counterCost > 0 {
					_, _ = gs.AddCoins(counter.SourceIndex, counterCost)
					result.Logs = append(result.Logs, fmt.Sprintf("%s refunded %d coins for canceled %s", gs.Players[counter.SourceIndex].Name, counterCost, counter.ID))
				}
				counterResult.Canceled = true
				result.Logs = append(result.Logs, fmt.Sprintf("%s's counter %s was challenged and canceled", gs.Players[counter.SourceIndex].Name, counter.ID))
				result.CounterResults = append(result.CounterResults, counterResult)
				continue
			}
		}

		if counterCancelsMain(counter) {
			result.Logs = append(result.Logs, fmt.Sprintf("%s counters with %s", gs.Players[counter.SourceIndex].Name, counter.ID))
			if err := g.applyActionWithSteps(ctx, input, counter, log); err != nil {
				return result, err
			}
			counterResult.Applied = true
			mainCanceled = true
			result.CounterResults = append(result.CounterResults, counterResult)
			result.Logs = append(result.Logs, fmt.Sprintf("main action %s is canceled by %s", mainCmd.ID, counter.ID))
			break
		}
		result.Logs = append(result.Logs, fmt.Sprintf("%s counters with %s", gs.Players[counter.SourceIndex].Name, counter.ID))
		result.CounterResults = append(result.CounterResults, counterResult)
		postCounters = append(postCounters, pendingPostCounter{
			cmd:   counter,
			index: len(result.CounterResults) - 1,
		})
	}

	if mainAllowed && !mainCanceled {
		if err := g.applyActionWithSteps(ctx, input, mainCmd, log); err != nil {
			return result, err
		}
		for _, counter := range postCounters {
			if err := g.applyActionWithSteps(ctx, input, counter.cmd, log); err != nil {
				return result, err
			}
			result.CounterResults[counter.index].Applied = true
		}
		result.MainApplied = true
	}

	result.MainCanceled = !result.MainApplied
	result.Logs = append(result.Logs, log.Public...)
	if log.Private != nil {
		for index, entries := range log.Private {
			result.PrivateLogs[index] = append(result.PrivateLogs[index], entries...)
		}
	}
	appendHandLogs(result.PrivateLogs, gs)
	result.Logs = append(result.Logs, fmt.Sprintf("turn %d ends", gs.TurnNumber))

	return result, nil
}

func appendHandLogs(private map[int][]string, gs *state.GameState) {
	if gs == nil {
		return
	}
	for index, player := range gs.Players {
		if len(player.Hand) == 0 {
			private[index] = append(private[index], "Your hand: (none)")
			continue
		}
		roles := make([]string, 0, len(player.Hand))
		for _, card := range player.Hand {
			roles = append(roles, string(card.Role))
		}
		private[index] = append(private[index], fmt.Sprintf("Your hand: %s", strings.Join(roles, ", ")))
	}
}
