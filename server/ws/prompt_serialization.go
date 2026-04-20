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

func DecodeCommandEnvelope(sessionID string, message CommandEnvelopeMessage) (sessionapp.CommandEnvelope, error) {
	if message.Type != TypeCommand {
		return sessionapp.CommandEnvelope{}, fmt.Errorf("unsupported command envelope type %q", message.Type)
	}
	command := sessionapp.CommandEnvelope{
		InteractionID: message.InteractionID,
		Kind:          sessionapp.CommandKind(message.Kind),
		SessionID:     sessionID,
	}
	switch command.Kind {
	case sessionapp.CommandMainAction:
		var payload sessionapp.MainActionCommand
		if err := decodePayload(message.Payload, &payload); err != nil {
			return sessionapp.CommandEnvelope{}, err
		}
		command.Payload = payload
	case sessionapp.CommandChallengeDecision:
		var payload sessionapp.ChallengeDecisionCommand
		if err := decodePayload(message.Payload, &payload); err != nil {
			return sessionapp.CommandEnvelope{}, err
		}
		command.Payload = payload
	case sessionapp.CommandCounterDecision:
		var payload sessionapp.CounterDecisionCommand
		if err := decodePayload(message.Payload, &payload); err != nil {
			return sessionapp.CommandEnvelope{}, err
		}
		command.Payload = payload
	case sessionapp.CommandCardSelection:
		var payload sessionapp.CardSelectionCommand
		if err := decodePayload(message.Payload, &payload); err != nil {
			return sessionapp.CommandEnvelope{}, err
		}
		command.Payload = payload
	default:
		return sessionapp.CommandEnvelope{}, fmt.Errorf("unsupported command kind %q", command.Kind)
	}
	return command, nil
}

func decodePayload(payload map[string]any, target any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode command payload: %w", err)
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		return fmt.Errorf("decode command payload: %w", err)
	}
	return nil
}
