package actions

import (
	"fmt"

	models "github.com/axwell0/elmakina/engine/core/types"
)

// payload_helpers centralizes payload-shape assertions so handlers and
// validators can stay focused on domain behavior.

// targetPayloadFromAction returns a typed target payload or a descriptive error
// when the action carries a different payload shape.
func targetPayloadFromAction(cmd *models.PlayerAction) (models.TargetPayload, error) {
	if cmd == nil {
		return models.TargetPayload{}, errCommandNil
	}
	payload, ok := cmd.Payload.(models.TargetPayload)
	if !ok {
		return models.TargetPayload{}, fmt.Errorf("%w: expected TargetPayload, got %T", errUnsupportedPayloadShape, cmd.Payload)
	}
	return payload, nil
}

// accusePayloadFromAction returns a typed accuse payload or a descriptive error
// when the action carries a different payload shape.
func accusePayloadFromAction(cmd *models.PlayerAction) (models.AccusePayload, error) {
	if cmd == nil {
		return models.AccusePayload{}, errCommandNil
	}
	payload, ok := cmd.Payload.(models.AccusePayload)
	if !ok {
		return models.AccusePayload{}, fmt.Errorf("%w: expected AccusePayload, got %T", errUnsupportedPayloadShape, cmd.Payload)
	}
	return payload, nil
}
