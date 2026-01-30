package websocket

import (
	"encoding/json"
	"fmt"

	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

func mustMarshal(v any) RawMessage {
	raw, _ := Marshal(v)
	return raw
}

func targetIndexFromAction(cmd *models.PlayerAction) *int {
	if cmd == nil {
		return nil
	}
	switch payload := cmd.Payload.(type) {
	case models.TargetPayload:
		return &payload.TargetIndex
	case *models.TargetPayload:
		if payload == nil {
			return nil
		}
		return &payload.TargetIndex
	case models.AccusePayload:
		return &payload.TargetIndex
	case *models.AccusePayload:
		if payload == nil {
			return nil
		}
		return &payload.TargetIndex
	default:
		return nil
	}
}

func parseActionPayload(raw RawMessage) (*models.PlayerAction, error) {
	var payload actionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if payload.Pass {
		return nil, fmt.Errorf("pass not allowed for actions")
	}
	if payload.ID == "" {
		return nil, fmt.Errorf("missing action id")
	}
	if payload.SourceIndex == nil {
		return nil, fmt.Errorf("missing source_index")
	}
	if payload.Guess != nil && payload.TargetIndex == nil {
		return nil, fmt.Errorf("guess requires target_index")
	}
	cmd := &models.PlayerAction{
		ID:          payload.ID,
		SourceIndex: *payload.SourceIndex,
	}
	if payload.MainAction != nil {
		cmd.MainAction = *payload.MainAction
	}
	if payload.TargetIndex != nil && payload.Guess != nil {
		role, err := models.ParseRole(*payload.Guess)
		if err != nil {
			return nil, err
		}
		cmd.Payload = models.AccusePayload{TargetIndex: *payload.TargetIndex, Guess: role}
	} else if payload.TargetIndex != nil {
		cmd.Payload = models.TargetPayload{TargetIndex: *payload.TargetIndex}
	}
	return cmd, nil
}

func parseCounterPayload(raw RawMessage) (engine.CounterDecision, error) {
	var payload actionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return engine.CounterDecision{}, err
	}
	if payload.Pass {
		return engine.CounterDecision{Pass: true}, nil
	}
	if payload.ID == "" {
		return engine.CounterDecision{}, fmt.Errorf("missing action id")
	}
	if payload.SourceIndex == nil {
		return engine.CounterDecision{}, fmt.Errorf("missing source_index")
	}
	if payload.Guess != nil && payload.TargetIndex == nil {
		return engine.CounterDecision{}, fmt.Errorf("guess requires target_index")
	}
	cmd := &models.PlayerAction{
		ID:          payload.ID,
		SourceIndex: *payload.SourceIndex,
	}
	if payload.MainAction != nil {
		cmd.MainAction = *payload.MainAction
	}
	if payload.TargetIndex != nil && payload.Guess != nil {
		role, err := models.ParseRole(*payload.Guess)
		if err != nil {
			return engine.CounterDecision{}, err
		}
		cmd.Payload = models.AccusePayload{TargetIndex: *payload.TargetIndex, Guess: role}
	} else if payload.TargetIndex != nil {
		cmd.Payload = models.TargetPayload{TargetIndex: *payload.TargetIndex}
	}
	return engine.CounterDecision{Command: cmd}, nil
}

func parseChallengePayload(raw RawMessage) (engine.ChallengeDecision, error) {
	var payload challengePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return engine.ChallengeDecision{}, err
	}
	if payload.ChallengerIndex == nil {
		return engine.ChallengeDecision{}, fmt.Errorf("missing challenger_index")
	}
	return engine.ChallengeDecision{
		ChallengerIndex:        *payload.ChallengerIndex,
		ChallengerDiscardIndex: payload.ChallengerDiscardIndex,
		ActorDiscardIndex:      payload.ActorDiscardIndex,
		ActorProvingCardIndex:  payload.ActorProvingCardIndex,
		Pass:                   payload.Pass,
	}, nil
}

func parseStepPayload(req state.StepRequest, raw RawMessage) (any, error) {
	if req.Kind == state.StepCardSelect {
		var indices []int
		if err := json.Unmarshal(raw, &indices); err != nil {
			return nil, err
		}
		return indices, nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func isAllowedPlayer(player int, allowed []int) bool {
	for _, p := range allowed {
		if p == player {
			return true
		}
	}
	return false
}
