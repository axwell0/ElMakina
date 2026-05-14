// File: engine/runtime/turn/counters.go
// Purpose: Runs counter windows and applies the results back into the turn.
package turn

import (
	"context"
	"fmt"
	"slices"
	"time"

	coreactions "github.com/axwell0/elmakina/engine/core/actions"
	coretypes "github.com/axwell0/elmakina/engine/core/types"
	turnmodel "github.com/axwell0/elmakina/engine/turn"
)

// collectCounters opens the counter window and gathers every counter response
// that is allowed to land before the window closes.
//
//nolint:gocognit,cyclop // Counter collection is a single select loop that keeps ordering, timeout, and cap rules together.
func (s *Service) collectCounters(ctx context.Context, input InputProvider, clock Clock, timeout time.Duration, main *coretypes.PlayerAction, logs *turnmodel.TurnLog) ([]*coretypes.PlayerAction, error) {
	var allowedPlayers []int
	for i, player := range s.State.players {
		if i != main.ActorIndex && len(player.Hand) > 0 {
			allowedPlayers = append(allowedPlayers, i)
		}
	}
	allowedByPlayer := make(map[int][]coretypes.ActionID, len(allowedPlayers))
	allowedSet := make(map[coretypes.ActionID]struct{})
	for _, playerIndex := range allowedPlayers {
		playerAllowed := coreactions.AllowedCounterActions(s.State, main, playerIndex)
		if len(playerAllowed) == 0 {
			continue
		}
		allowedByPlayer[playerIndex] = playerAllowed
		for _, id := range playerAllowed {
			allowedSet[id] = struct{}{}
		}
	}
	allowed := make([]coretypes.ActionID, 0, len(allowedSet))
	for _, id := range coreactions.CounterActions() {
		if _, ok := allowedSet[id]; ok {
			allowed = append(allowed, id)
		}
	}
	if len(allowed) == 0 {
		// Action has no registered counters; skip window entirely.
		return nil, nil
	}
	window := CounterWindow{
		MainAction:             main,
		AllowedActions:         allowed,
		AllowedActionsByPlayer: allowedByPlayer,
		AllowedPlayers:         allowedPlayers,
	}
	windowCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, err := input.CounterResponses(windowCtx, window)
	if err != nil {
		return nil, fmt.Errorf("open counter responses: %w", err)
	}
	var timeoutCh <-chan time.Time
	if timeout > 0 {
		timeoutCh = clock.After(timeout)
	}
	var errCh <-chan error
	type errorReporter interface {
		Errors() <-chan error
	}
	if reporter, ok := input.(errorReporter); ok {
		errCh = reporter.Errors()
	}

	var counters []*coretypes.PlayerAction
	responded := make(map[int]bool, len(window.AllowedPlayers))
	actionCounts := make(map[coretypes.ActionID]int)
	for {
		select {
		case <-ctx.Done():
			return counters, fmt.Errorf("counter window canceled: %w", ctx.Err())
		case err := <-errCh:
			if err != nil {
				return counters, fmt.Errorf("counter response stream: %w", err)
			}
		case <-timeoutCh:
			// Window timeout ends counter collection with whatever we have.
			return counters, nil
		case decision, ok := <-ch:
			if !ok {
				return counters, nil
			}
			if decision.Pass {
				if !slices.Contains(window.AllowedPlayers, decision.PlayerIndex) {
					return counters, fmt.Errorf("%w: %d", errCounterPlayerNotFound, decision.PlayerIndex)
				}
				if responded[decision.PlayerIndex] {
					return counters, fmt.Errorf("%w: %d", errCounterPlayerDuplicate, decision.PlayerIndex)
				}
				responded[decision.PlayerIndex] = true
				logs.Publicf("%s passes counter", playerName(s.State, decision.PlayerIndex))
				if len(responded) == len(window.AllowedPlayers) {
					// Everyone responded; close window early.
					return counters, nil
				}
				continue
			}

			cmd := decision.Command
			if cmd == nil {
				return counters, errCounterActionNil
			}
			if !slices.Contains(window.AllowedPlayers, cmd.ActorIndex) {
				return counters, fmt.Errorf("%w: %d", errCounterPlayerNotFound, cmd.ActorIndex)
			}
			if responded[cmd.ActorIndex] {
				return counters, fmt.Errorf("%w: %d", errCounterPlayerDuplicate, cmd.ActorIndex)
			}
			if !slices.Contains(allowedByPlayer[cmd.ActorIndex], cmd.Action) {
				return counters, fmt.Errorf("%w: %q", errCounterActionNotFound, cmd.Action)
			}
			// Validate counter payload and actor legality in current state.
			if err := coreactions.ValidateCounterAction(s.State, cmd, main.Action); err != nil {
				return counters, fmt.Errorf("validate counter action: %w", err)
			}
			counters = append(counters, cmd)
			responded[cmd.ActorIndex] = true
			actionCounts[cmd.Action]++

			if def, ok := coreactions.CounterAction(cmd.Action); ok {
				maxDeclarations := def.Counter.MaxCounterDeclarations
				if maxDeclarations == 0 {
					maxDeclarations = 1
				}
				if maxDeclarations > 0 && actionCounts[cmd.Action] >= maxDeclarations {
					// Reached configured per-counter cap.
					return counters, nil
				}
				if maxDeclarations == 1 {
					// Single-declaration counters close the window immediately.
					return counters, nil
				}
			}
			if len(responded) == len(window.AllowedPlayers) {
				return counters, nil
			}
		}
	}
}
