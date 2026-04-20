package ws

import (
	"encoding/json"
	"testing"

	sessionapp "example.com/elmakina/app/session"
	"github.com/stretchr/testify/require"
)

func TestPromptEnvelopeJSONShape(t *testing.T) {
	envelope := PromptEnvelope{
		Type: "prompt",
		Interaction: InteractionWireState{
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
		Type:          "command",
		InteractionID: "ix-2",
		Kind:          "challenge_decision",
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
	require.Equal(t, "ix-2", decoded["interactionId"])
	_ = sessionapp.InteractionState{}
}
