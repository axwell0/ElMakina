package ws

import (
	"encoding/json"
	"testing"

	api "github.com/axwell0/elmakina/internal/api"
	sessionapp "github.com/axwell0/elmakina/internal/app/session"
	"github.com/stretchr/testify/require"
)

// The serialization tests ensure browser-facing websocket envelopes keep a stable JSON shape.
func TestPromptEnvelopeJSONShape(t *testing.T) {
	envelope := PromptEnvelope{
		Type: "prompt",
		Prompt: &api.PromptView{
			ID:         "ix-1",
			Kind:       "challenge",
			Recipients: []int{1},
			Payload: map[string]any{
				"actorIndex":  0,
				"action":      "assassinate",
				"claimedRole": "Colonel",
			},
		},
	}

	payload, err := json.Marshal(envelope)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, "prompt", decoded["type"])
}

func TestCommandEnvelopeJSONShape(t *testing.T) {
	command := CommandEnvelopeMessage{
		Type:     "command",
		PromptID: "ix-2",
		Kind:     "challenge_decision",
		Payload: map[string]any{
			"challengerIndex": 1,
			"pass":            false,
		},
	}

	payload, err := json.Marshal(command)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, "command", decoded["type"])
	require.Equal(t, "ix-2", decoded["promptId"])
}

func TestDecodeCommandEnvelopeReturnsConstrainedPayload(t *testing.T) {
	command, err := DecodeCommandEnvelope("session-1", CommandEnvelopeMessage{
		Type:     api.TypeCommand,
		PromptID: "ix-3",
		Kind:     string(sessionapp.CommandCardSelection),
		Payload: map[string]any{
			"selectedIndices": []any{0, 2},
		},
	})
	require.NoError(t, err)

	require.Equal(t, sessionapp.CommandCardSelection, command.Payload.CommandKind())
	require.Equal(t, sessionapp.CardSelectionCommand{SelectedIndices: []int{0, 2}}, command.Payload)
}
