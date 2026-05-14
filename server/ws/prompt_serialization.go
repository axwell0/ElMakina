// prompt_serialization.go converts engine-layer prompts into WebSocket payloads.
//
// # What is a "prompt"?
//
// During a turn the engine may pause and ask one or more players to make a
// decision (e.g. "which card do you want to keep after Exchange?").  This is
// represented internally as a session Prompt.
//
// When a prompt opens, the room calls BuildPromptEnvelope() to
// create a PromptEnvelope and sends it to the eligible players.  The browser
// renders a UI widget based on Prompt.Kind and waits for the player to submit
// a CommandEnvelopeMessage.
//
// When the engine moves on (prompt closes), the room calls
// BuildClearedPromptEnvelope() so the browser can hide the widget.
//
// DecodeCommandEnvelope is the reverse path: it takes the browser's raw command
// map and decodes it into the concrete Go struct the engine expects.
//
//nolint:err113 // Prompt codec currently returns descriptive runtime errors for invalid client payloads.
package ws

import (
	"encoding/json"
	"fmt"
	"slices"

	api "github.com/axwell0/elmakina/internal/api"
	sessionapp "github.com/axwell0/elmakina/internal/app/session"
)

// BuildPromptEnvelope turns the active prompt into the websocket prompt payload.
func BuildPromptEnvelope(prompt *sessionapp.Prompt) (PromptEnvelope, error) {
	if prompt == nil {
		return BuildClearedPromptEnvelope("")
	}

	return PromptEnvelope{
		Type: api.TypePrompt,
		Prompt: &api.PromptView{
			ID:         prompt.ID,
			Kind:       string(prompt.Kind),
			Recipients: slices.Clone(prompt.Recipients),
			Payload:    prompt.Payload,
		},
	}, nil
}

// BuildClearedPromptEnvelope tells the browser that the previous prompt is no longer active.
func BuildClearedPromptEnvelope(promptID string) (PromptEnvelope, error) {
	return PromptEnvelope{
		Type:            api.TypePrompt,
		Prompt:          nil,
		ClearedPromptID: promptID,
	}, nil
}

// DecodeCommandEnvelope converts a browser command into the engine's command envelope.
func DecodeCommandEnvelope(sessionID string, message CommandEnvelopeMessage) (sessionapp.CommandEnvelope, error) {
	if message.Type != api.TypeCommand {
		return sessionapp.CommandEnvelope{}, fmt.Errorf("unsupported command envelope type %q", message.Type)
	}
	// Start with the common command metadata that every command kind needs.
	command := sessionapp.CommandEnvelope{
		PromptID:  message.PromptID,
		Kind:      sessionapp.CommandKind(message.Kind),
		SessionID: sessionID,
	}
	switch command.Kind {
	case sessionapp.CommandMainAction:
		// Main actions are the primary turn decision, so they carry the action ID and optional target/guess.
		payload, err := decodePayload[sessionapp.MainActionCommand](message.Payload)
		if err != nil {
			return sessionapp.CommandEnvelope{}, err
		}
		command.Payload = payload
	case sessionapp.CommandChallengeDecision:
		// Challenge decisions record whether the challenger passed or escalated.
		payload, err := decodePayload[sessionapp.ChallengeDecisionCommand](message.Payload)
		if err != nil {
			return sessionapp.CommandEnvelope{}, err
		}
		command.Payload = payload
	case sessionapp.CommandCounterDecision:
		// Counter decisions mirror the challenge shape but apply to the counter window.
		payload, err := decodePayload[sessionapp.CounterDecisionCommand](message.Payload)
		if err != nil {
			return sessionapp.CommandEnvelope{}, err
		}
		command.Payload = payload
	case sessionapp.CommandCardSelection:
		// Card selection is used when the engine asks a player to choose one or more cards.
		payload, err := decodePayload[sessionapp.CardSelectionCommand](message.Payload)
		if err != nil {
			return sessionapp.CommandEnvelope{}, err
		}
		command.Payload = payload
	default:
		return sessionapp.CommandEnvelope{}, fmt.Errorf("unsupported command kind %q", command.Kind)
	}
	return command, nil
}

// decodePayload round-trips the generic map through JSON into the target command type.
func decodePayload[T sessionapp.CommandPayload](payload map[string]any) (T, error) {
	var target T
	// Marshal first so the generic browser payload can be decoded into the concrete Go command struct.
	encoded, err := json.Marshal(payload)
	if err != nil {
		return target, fmt.Errorf("encode command payload: %w", err)
	}
	if err := json.Unmarshal(encoded, &target); err != nil {
		return target, fmt.Errorf("decode command payload: %w", err)
	}
	return target, nil
}
