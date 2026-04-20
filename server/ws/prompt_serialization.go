package ws

import (
	"encoding/json"
	"fmt"

	sessionapp "example.com/elmakina/app/session"
)

func BuildPromptEnvelope(interaction *sessionapp.InteractionState) (PromptEnvelope, error) {
	if interaction == nil {
		return PromptEnvelope{}, fmt.Errorf("interaction is nil")
	}

	payload, err := payloadToMap(interaction.Payload)
	if err != nil {
		return PromptEnvelope{}, err
	}

	return PromptEnvelope{
		Type: TypePrompt,
		Interaction: InteractionWireState{
			ID:         interaction.ID,
			Kind:       string(interaction.Kind),
			Recipients: append([]int(nil), interaction.Recipients.Players...),
			Payload:    payload,
		},
	}, nil
}

func payloadToMap(payload any) (map[string]any, error) {
	if payload == nil {
		return map[string]any{}, nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode interaction payload: %w", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("decode interaction payload: %w", err)
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}
