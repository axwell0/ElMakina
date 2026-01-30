package engine

import (
	"context"

	"ElMakina/backend/actions"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

func (g *Game) applyActionWithSteps(ctx context.Context, input InputProvider, cmd *models.PlayerAction, log *state.TurnLog) error {
	runner := actions.StartAction(ctx, func(f *actions.ActionFlow) error {
		return actions.ApplyAction(f, &g.CurrentState, &g.TurnState, log, cmd)
	})

	req, ok := <-runner.Flow.Requests
	if !ok {
		return runner.GetResult()
	}

	for {
		resp, err := input.RequestStep(ctx, req)
		if err != nil {
			runner.Cancel()
			return err
		}
		nextReq, err := runner.Next(resp)
		if err != nil {
			return err
		}
		if nextReq == nil {
			return nil
		}
		req = *nextReq
	}
}

func actionRole(id models.ActionID) models.Role {
	if def, ok := actions.ActionRegistry[id]; ok {
		return def.Role
	}
	return ""
}
