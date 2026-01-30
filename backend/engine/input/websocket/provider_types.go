package websocket

import (
	"net/http"
	"sync"

	ws "github.com/gorilla/websocket"
)

// Provider is the "Interpreter" between WebSocket clients and the game engine.
//
// The engine speaks in game terms: "RequestAction", "ChallengeResponses", etc.
// Players speak in WebSocket messages: JSON envelopes with types like "action", "challenge".
// This struct translates between those two worlds.
//
// ARCHITECTURAL ROLE:
// Think of this as the "Adapter Pattern" in action. The engine defines an interface
// (InputProvider) that describes WHAT it needs: "ask player X for an action".
// This Provider implements HOW to do that: send a WebSocket message, wait for reply.
//
// KEY DESIGN DECISIONS:
//
// 1. PULL-BASED, NOT PUSH-BASED
//    The engine asks for input when it needs it. We don't broadcast game state
//    constantly and hope players respond appropriately. The engine drives the flow.
//
// 2. REQUEST CORRELATION (request_id)
//    Every request gets a unique ID. Responses must include this ID so we know
//    which question they're answering. This handles race conditions where a slow
//    response to an old request arrives after we've moved on.
//
// 3. WAITER PATTERN
//    When we send a request, we register a "waiter" (a channel). The goroutine
//    handling WebSocket responses finds the right waiter and delivers the message.
//    This decouples the WebSocket read loop from the engine's request loop.
//
// 4. DISCONNECT HANDLING
//    WebSocket disconnects are terminal. If a player drops mid-turn, we signal
//    an error on errCh. The orchestrator decides how to handle this (forfeit, AI, etc.)
//
// CONCURRENCY SAFETY:
//   - mu (RWMutex) protects conns and pending maps
//   - Each connection has its own sendMu for serialized writes
//   - The WebSocket read loop runs in a separate goroutine per connection
type Provider struct {
	upgrader ws.Upgrader // Gorilla WebSocket upgrader with CORS settings

	mu      sync.RWMutex         // Guards the maps below
	conns   map[int]*connState   // player_index -> WebSocket connection
	pending map[string]*waiter   // request_id -> channel waiting for response
	errCh   chan error           // Terminal errors (disconnects) flow here
	nextID  uint64               // Atomic counter for generating request IDs
}

// connState tracks a single player's WebSocket connection.
//
// Each player has one WebSocket connection. This struct holds:
//   - index: The player's index in the game (0, 1, 2, etc.)
//   - conn: The actual WebSocket connection
//   - sendMu: Mutex for serializing writes (gorilla/websocket requires this)
//
// WHY SERIALIZE WRITES?
// The gorilla WebSocket library is not safe for concurrent writes. If multiple
// goroutines try to WriteJSON at the same time, Bad Things Happen. The sendMu
// ensures only one write happens at a time per connection.
type connState struct {
	index  int
	conn   wsConn
	sendMu sync.Mutex
}

type wsConn interface {
	ReadJSON(v any) error
	WriteJSON(v any) error
	Close() error
}

// responseEnvelope wraps a WebSocket message with its sender's identity.
//
// When the read loop receives a message, we need to know:
//   - Who sent it? (sender player index)
//   - What did they send? (the Envelope payload)
//
// This wrapper lets us pass both pieces of information through channels.
type responseEnvelope struct {
	sender int      // Player index who sent this
	env    Envelope // The parsed WebSocket message
}

// waiter represents a pending request waiting for its response.
//
// When the engine asks "What action do you want?", we create a waiter
// with a buffered channel. The WebSocket read loop will send the player's
// response on this channel, and the request goroutine blocks receiving from it.
//
// WHY A CHANNEL?
// This decouples the WebSocket read loop from the engine's request flow.
// The read loop just routes messages to waiters; it doesn't need to know
// what game logic is waiting for what.
type waiter struct {
	ch chan responseEnvelope // Channel where the response will be delivered
}

// NewProvider constructs a WebSocket input provider with permissive origin checks.
func NewProvider() *Provider {
	return &Provider{
		upgrader: ws.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		conns:   make(map[int]*connState),
		pending: make(map[string]*waiter),
		errCh:   make(chan error, 4),
	}
}

// Errors exposes terminal transport errors (e.g., disconnects).
func (p *Provider) Errors() <-chan error {
	return p.errCh
}
