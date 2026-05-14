package turn

import (
	"context"
	"fmt"

	"github.com/axwell0/elmakina/engine/core/actions"
	"github.com/axwell0/elmakina/engine/core/types"
	"github.com/axwell0/elmakina/engine/turn"
)

func (s *Service) applyActionWithSteps(ctx context.Context, inputProvider InputProvider, actionCommand *types.PlayerAction, logs *turn.TurnLog) error {
	result, err := actions.ApplyAction(s.State, logs, actionCommand)
	if err != nil {
		return fmt.Errorf("apply action: %w", err)
	}
	for result.Step != nil {
		var resp any
		var err error
		stepRequest := result.Step.Request
		switch stepRequest.Kind {
		case actions.StepCardSelection:
			resp, err = inputProvider.RequestCardSelection(ctx, stepRequest)
		default:
			return fmt.Errorf("%w: %q", errUnsupportedActionStep, stepRequest.Kind)
		}
		if err != nil {
			return fmt.Errorf("request card selection: %w", err)
		}
		result, err = actions.ResumeAction(s.State, logs, result.Step, resp, actionCommand)
		if err != nil {
			return fmt.Errorf("resume action: %w", err)
		}
	}
	if !result.Applied {
		return fmt.Errorf("%w: %s", errActionDidNotComplete, actionCommand.Action)
	}
	return nil
}
