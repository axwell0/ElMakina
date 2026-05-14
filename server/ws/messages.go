// messages.go defines the wire-format structs for WebSocket messages.
//
// Every struct here maps 1:1 to a JSON object the browser sends or receives.
// Each message has a Type field as a discriminator — the browser reads Type first
// to know which handler to route it to, then decodes the rest of the fields.
//
// # Flow overview
//
// Browser → Server (inbound):
//
//	incomingEnvelope (helpers.go)  — raw envelope; Type decoded first
//	    ├── CommandEnvelopeMessage  — player submits a game command
//	    ├── SubmitReadyMessage      — player marks themselves ready
//	    └── SubmitStartMessage      — leader starts the match
//
// Server → Browser (outbound):
//
//	PromptEnvelope                  — prompt opened or cleared
//	api.RoomSyncView                — full room snapshot (in internal/api)
//	api.TurnResultView              — completed turn summary (in internal/api)
//	api.GameOverView                — game finished (in internal/api)
//
// The split between this file and internal/api is intentional:
// structs here are WebSocket-specific; structs in internal/api are shared
// between WebSocket and HTTP responses.
package ws

import api "github.com/axwell0/elmakina/internal/api"

// PromptEnvelope carries the currently answerable prompt, if one is open.
// When Prompt is non-nil, the browser should render the prompt UI.
// When ClearedPromptID is set and Prompt is nil, the browser should hide any
// prompt with that ID (the engine moved on).
type PromptEnvelope struct {
	Type            api.MessageType `json:"type"`
	Prompt          *api.PromptView `json:"prompt"`
	ClearedPromptID string          `json:"clearedPromptId,omitempty"`
}

// CommandEnvelopeMessage is the inbound command message sent by the browser when the
// player submits a response to a prompt (e.g. chooses an action, picks a card).
//
// PromptID links this response back to the specific prompt that triggered it —
// the server uses it to reject stale responses from a previous prompt.
//
// Payload is map[string]any because different commands have different shapes.
// The server decodes it into the concrete type based on Kind.
type CommandEnvelopeMessage struct {
	Type     api.MessageType `json:"type"`
	PromptID string          `json:"promptId"`
	Kind     string          `json:"kind"`
	Payload  map[string]any  `json:"payload"`
}

// SubmitReadyMessage marks a player as ready in the lobby.
type SubmitReadyMessage struct {
	Type   api.MessageType `json:"type"`
	RoomID string          `json:"roomId"`
}

// SubmitStartMessage asks the leader to start the room.
type SubmitStartMessage struct {
	Type   api.MessageType `json:"type"`
	RoomID string          `json:"roomId"`
}
