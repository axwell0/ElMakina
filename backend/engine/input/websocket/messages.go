package websocket

import (
	"encoding/json"

	"ElMakina/backend/models"
)

// RawMessage aliases json.RawMessage for easier reuse in tests and payloads.
type RawMessage = json.RawMessage

// Envelope is the wire-level message container for the WS input protocol.
//
// Contract:
//   - Every request sent by the server includes a request_id.
//   - Clients must echo request_id in their response.
//   - Payload is decoded based on Type.
type Envelope struct {
	Type      string     `json:"type"`
	RequestID string     `json:"request_id,omitempty"`
	Payload   RawMessage `json:"payload,omitempty"`
}

const (
	msgHello           = "hello"
	msgRequestAction   = "request_action"
	msgRequestStep     = "request_step"
	msgChallengeWindow = "challenge_window"
	msgCounterWindow   = "counter_window"
	msgAction          = "action"
	msgChallenge       = "challenge"
	msgCounter         = "counter"
	msgStepResult      = "step_result"
)

type helloPayload struct {
	PlayerIndex int `json:"player_index"`
}

type actionPayload struct {
	ID          models.ActionID  `json:"id"`
	SourceIndex *int             `json:"source_index"`
	TargetIndex *int             `json:"target_index,omitempty"`
	Guess       *string          `json:"guess,omitempty"`
	MainAction  *models.ActionID `json:"main_action,omitempty"`
	Pass        bool             `json:"pass,omitempty"`
}

type challengePayload struct {
	ChallengerIndex        *int `json:"challenger_index"`
	ChallengerDiscardIndex int  `json:"challenger_discard_index"`
	ActorDiscardIndex      int  `json:"actor_discard_index"`
	ActorProvingCardIndex  int  `json:"actor_proving_card_index"`
	Pass                   bool `json:"pass,omitempty"`
}

type requestActionPayload struct {
	ActorIndex int `json:"actor_index"`
}

type requestStepPayload struct {
	Step any `json:"step"`
}

type challengeWindowPayload struct {
	ActorIndex  int             `json:"actor_index"`
	ActionID    models.ActionID `json:"action_id"`
	ClaimedRole string          `json:"claimed_role"`
	Kind        string          `json:"kind"`
	TargetIndex *int            `json:"target_index,omitempty"`
}

type counterWindowPayload struct {
	ActorIndex    int               `json:"actor_index"`
	ActionID      models.ActionID   `json:"action_id"`
	AllowedAction []models.ActionID `json:"allowed_actions,omitempty"`
	TargetIndex   *int              `json:"target_index,omitempty"`
}

// Marshal serializes a payload into raw JSON for the Envelope.
func Marshal(v any) (RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return RawMessage(b), nil
}
