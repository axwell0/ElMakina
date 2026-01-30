# Package: server/ws

## Knowledge Context
Prerequisites:
- **WebSockets**: Understanding of full-duplex communication, frames, and closing handshakes.
- **`backend/server`**: The lobby/session logic this package exposes (server/lobby.go:67).
- **JSON**: The wire format.
- **`backend/engine`**: How RunTurn and InputProvider work (engine/orchestrator_turn.go:29, orchestrator_types.go:24).

## Core Responsibility
This package is the **Transport Layer**. It is the only place in the backend that speaks "Network". It:
1.  Accepts WebSocket connections.
2.  Deserializes incoming JSON messages.
3.  Routes messages to the appropriate Lobby or Game Session.
4.  Serializes Engine updates and pushes them to clients.
5.  Handles disconnections, reconnections, and timeouts.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                  WEBSOCKET ARCHITECTURE                  │
└─────────────────────────────────────────────────────────┘

HTTP Upgrade
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│ Server (server.go:30)                                    │
│ ├─ Manages client connections (map[string]*clientConn)   │
│ ├─ Handles lobby lifecycle                               │
│ ├─ Routes messages                                       │
│ └─ Coordinates game sessions                             │
└─────────────────────────────────────────────────────────┘
    │
    ├─ Lobby Mode ──▶ handleLobbyMessage() ──▶ LobbyManager
    │                       (server.go:223)
    │
    └─ Game Mode ───▶ sessionRunner ──▶ Game Engine
                          (session_runner.go:201)

SessionRunner (session_runner.go:201)
    │
    ├─ Owns Game instance (engine/game.go:19)
    ├─ Implements InputProvider (engine/orchestrator_types.go:24)
    ├─ Runs game loop: RunTurn() in loop
    ├─ Broadcasts results to all clients
    └─ Handles disconnect/reconnect

SessionProvider (session_provider.go)
    │
    ├─ Implements InputProvider interface
    ├─ Adapts async WebSocket to sync engine calls
    ├─ RequestAction() waits for player message
    ├─ ChallengeResponses() polls all players
    └─ CounterResponses() polls eligible players
```

## Inner Workings and Logic

### Connection Lifecycle

**ServeHTTP** (server.go:88) - HTTP upgrade to WebSocket:
1. Upgrade HTTP to WebSocket (gorilla/websocket)
2. Read "hello" message (player registration or reconnect)
3. Create clientConn with player identity
4. Start readLoop in goroutine (server.go:205)
5. Send initial state (lobby list, player token)

**readLoop** (server.go:205) - Message handling:
```go
for {
    Read JSON message
    │
    ├─ IF in game session:
    │    └─▶ Enqueue to sessionRunner (session_runner.go:241)
    │
    └─ IF in lobby:
         └─▶ handleLobbyMessage() (server.go:223)
}
```

**unregisterClient** (server.go:177) - Disconnection:
- IF in game: Detach from session, schedule forfeit
- IF in lobby: Remove from lobby
- Keep player registration for reconnect

### Message Routing

**Lobby Messages** (server.go:223):
```go
MsgHello        - Ignored (already handled in handshake)
MsgLobbyCreate  - Creates new lobby
MsgLobbyJoin    - Joins existing lobby
MsgLobbyStart   - Starts game (transitions to Game Mode)
MsgLobbyList    - Returns list of available lobbies
```

**Game Messages** (session_runner.go - via provider):
```go
// Sent to sessionRunner.EnqueueClientEnvelope()
// Provider routes to appropriate handler based on game phase

MsgAction       - Player's action choice
MsgChallenge    - Challenge decision (pass/challenge)
MsgCounter      - Counter action
MsgStepResponse - Multi-step response (card selection)
MsgChat         - Chat message (side channel)
MsgForfeit      - Surrender
MsgPause        - Request pause/resume
```

### SessionRunner - The Game Loop

**sessionRunner** (session_runner.go:201) - Drives the engine:
```go
func (r *sessionRunner) runGameLoop() {
    for {
        // 1. Run one turn
        result, err := r.game.RunTurn(ctx, r.provider, r.clock, r.cfg)
        
        // 2. Handle result
        if err != nil {
            // Handle error (disconnect, etc.)
        }
        
        // 3. Broadcast to all players
        r.broadcastTurnResult(result)
        
        // 4. Check game over
        if winner := r.game.CheckGameOver(); winner != nil {
            r.broadcastGameOver(winner)
            break
        }
        
        // 5. Increment turn
        r.game.IncrementTurn()
    }
}
```

**Key Method**: `run()` (session_runner.go:241) - Command processing loop:
```go
for cmd := range r.cmdCh {
    switch cmd.Type {
    case cmdClientEnvelope:
        // Route to provider or handle directly
    case cmdOffline:
        // Player disconnected
    case cmdOnline:
        // Player reconnected
    }
}
```

### SessionProvider - InputProvider Implementation

**Adapts async WebSocket to sync engine** (session_provider.go):

Implements `engine.InputProvider` (orchestrator_types.go:24):
```go
type sessionProvider struct {
    // ... fields
}

func (p *sessionProvider) RequestAction(...) (*models.PlayerAction, error) {
    // Send prompt to player
    // Block until response received
    // Return PlayerAction
}

func (p *sessionProvider) ChallengeResponses(...) (<-chan ChallengeDecision, error) {
    // Send challenge prompt to all eligible players
    // Collect responses with timeout
    // Return channel of decisions
}

func (p *sessionProvider) CounterResponses(...) (<-chan CounterDecision, error) {
    // Similar to ChallengeResponses
}

func (p *sessionProvider) RequestStep(...) (any, error) {
    // Request multi-step input
    // Block until response
}
```

**Design Pattern**: This is the Adapter Pattern - adapting async WS events to the sync interface the engine expects.

## Key Architectural Patterns

- **Actor Model-ish**: Each connection is effectively an actor with inbox/outbox handling (readLoop/writeLoop).
- **Adapter Pattern**: `SessionInputProvider` adapts the asynchronous, event-driven WebSocket stream into the synchronous, call-response interface required by the `engine`.
- **Event Loop**: The `SessionRunner` effectively acts as the game loop, ticking the engine forward.
- **Graceful Degradation**: On disconnect, player is scheduled for forfeit; can reconnect within grace period.

## Critical Components

### Server (server.go:30)
Manages the set of active connections.
```go
type Server struct {
    upgrader gws.Upgrader
    manager  *server.LobbyManager      // lobby.go:67
    clock    engine.Clock              // orchestrator_types.go:43
    cfg      engine.TurnConfig         // orchestrator_types.go:54
    clients  map[string]*clientConn    // playerID -> connection
    sessions map[string]*sessionRunner // lobbyID -> session
    forfeit  *ForfeitScheduler         // forfeit.go
}
```

**Entry Point**: `ServeHTTP` (server.go:88)

### SessionRunner (session_runner.go:201)
The "main loop" for a running match.
```go
type sessionRunner struct {
    session     *server.GameSession    // lobby.go
    game        *engine.Game           // engine/game.go:19
    provider    *sessionProvider       // session_provider.go
    cmdCh       chan command           // Command queue
    // ... other fields
}
```

**Key Methods**:
- `Start()` - Begins game loop in goroutine
- `runGameLoop()` - Main game loop (calls RunTurn repeatedly)
- `run()` - Command processor (session_runner.go:241)
- `EnqueueClientEnvelope()` - Queue message from client
- `Forfeit()` - Handle player forfeit

### SessionProvider (session_provider.go)
The bridge from WS → Engine (implements InputProvider).
```go
type sessionProvider struct {
    responses   map[int]chan *models.PlayerAction
    challenges  map[int]chan ChallengeDecision
    counters    map[int]chan CounterDecision
    steps       map[int]chan any
}
```

**Key Methods** (implementing InputProvider):
- `RequestAction()` - Wait for player action
- `ChallengeResponses()` - Collect challenges
- `CounterResponses()` - Collect counters
- `RequestStep()` - Multi-step input
- `attach()` / `detach()` - Player connection management

### ForfeitScheduler (forfeit.go)
Logic to handle "rage quits" or disconnects.
```go
type ForfeitScheduler struct {
    clock Clock
    grace time.Duration  // Grace period for reconnect
}
```

**Usage**: On disconnect, schedule forfeit; on reconnect, cancel.

### ClientConn (server.go:50)
Represents a single WebSocket connection.
```go
type clientConn struct {
    player      *server.Player  // lobby.go
    conn        wsConn          // WebSocket connection
    lobbyID     string          // Current lobby
    session     *sessionRunner  // Active game session
    playerIndex int             // Position in game
}
```

## File Structure

```
server/ws/
├── server.go           # WebSocket server, connection handling
├── session_runner.go   # Game loop, session lifecycle
├── session_provider.go # InputProvider implementation
├── messages.go         # WebSocket message types
├── forfeit.go          # Disconnect/forfeit logic
├── pause.go            # Game pause handling
├── config.go           # Game configuration messages
├── replay_helpers.go   # Replay-related utilities
├── request_action_test.go  # Test examples
├── server_test.go      # Server tests
├── session_runner_*_test.go # Session tests
└── *_test.go           # Other tests
```

## Message Protocol

### Client → Server (Incoming)
```go
type Envelope struct {
    Type    string          // Message type
    Payload json.RawMessage // Type-specific data
}
```

**Types**:
- `hello` - Initial connection (nickname, avatar, reconnect token)
- `lobby.create` - Create lobby
- `lobby.join` - Join lobby
- `lobby.start` - Start game (leader only)
- `lobby.list` - List available lobbies
- `action` - Game action (during play)
- `challenge` - Challenge decision
- `counter` - Counter action
- `step_response` - Multi-step response
- `chat` - Chat message
- `forfeit` - Surrender

### Server → Client (Outgoing)
```go
type Envelope struct {
    Type    string
    Payload json.RawMessage
    RequestID string // For correlation (optional)
}
```

**Types**:
- `hello_ack` - Connection accepted
- `hello_error` - Connection rejected
- `lobby.created` - Lobby created
- `lobby.joined` - Joined lobby
- `lobby.state` - Lobby state update
- `lobby.list` - List of lobbies
- `lobby.started` - Game started
- `game.config` - Game configuration
- `game.state` - Game state update
- `game.turn` - Turn result
- `game.over` - Game ended
- `game.paused` - Game paused
- `prompt.action` - Request action choice
- `prompt.challenge` - Request challenge decision
- `prompt.counter` - Request counter action
- `prompt.step` - Request multi-step input
- `chat` - Chat message
- `error` - Error message

**See**: `messages.go` for full type definitions

## Session Lifecycle

```
1. CONNECTION
   Client ──WebSocket──▶ Server.ServeHTTP (server.go:88)
   
2. HANDSHAKE
   Server ──hello_ack──▶ Client
   Server ──game.config─▶ Client
   
3. LOBBY PHASE
   Client ──lobby.create──▶ Server ──lobby.created──▶ Client
   or
   Client ──lobby.join────▶ Server ──lobby.joined───▶ Client
   
   Server broadcasts lobby.state to all lobby members
   
4. GAME START
   Leader ──lobby.start──▶ Server
   Server creates sessionRunner (session_runner.go:201)
   Server attaches all players
   Server ──lobby.started──▶ All clients
   Server ──game.state────▶ All clients
   
5. GAME LOOP
   sessionRunner.runGameLoop() begins
   For each turn:
     - Engine requests action via provider.RequestAction()
     - Server ──prompt.action──▶ Current player
     - Player ──action─────────▶ Server
     - Engine processes (challenge/counter/execute)
     - Server ──game.turn──────▶ All clients
   
6. GAME END
   Engine detects winner (CheckGameOver at game.go:83)
   Server ──game.over──▶ All clients
   Session cleanup
   
7. DISCONNECT/RECONNECT
   Disconnect:
     - Server schedules forfeit (forfeit.go)
     - Grace period begins
   Reconnect (within grace period):
     - Server cancels forfeit
     - Server ──game.state──▶ Reconnected client
     - Game continues
```

## Cross-Reference Guide

| Component | Location | Used By | Purpose |
|-----------|----------|---------|---------|
| Server | server.go:30 | main.go | WebSocket entry point |
| ServeHTTP | server.go:88 | HTTP router | Connection upgrade |
| clientConn | server.go:50 | Server | Connection state |
| handleLobbyMessage | server.go:223 | readLoop | Lobby routing |
| sessionRunner | session_runner.go:201 | Server | Game session |
| runGameLoop | session_runner.go | Start() | Main game loop |
| run | session_runner.go:241 | runGameLoop | Command processor |
| sessionProvider | session_provider.go | sessionRunner | InputProvider impl |
| EnqueueClientEnvelope | session_runner.go | readLoop | Message queue |
| Forfeit | session_runner.go | forfeit scheduler | Player surrender |
| ForfeitScheduler | forfeit.go | Server | Disconnect handling |
| messages.go | - | All | Protocol definitions |

## External Dependencies
- **`github.com/gorilla/websocket`**: The low-level WS library.
- **`backend/engine`**: The game logic being driven (orchestrator_turn.go:29).
- **`backend/server`**: The lobby state being exposed (lobby.go:67).
- **`backend/models`**: Message types (defs.go:6).
- **`backend/actions`**: Action validation (validators.go:9).

## Testing

**Test Types**:
- `server_test.go` - Connection handling
- `session_runner_test.go` - Game loop
- `*_test.go` - Specific features (chat, pause, etc.)

**Pattern**: Use httptest server + WebSocket client
```go
// Create test server
server := NewServer(...)
httptestServer := httptest.NewServer(server)

// Connect WebSocket
ws, _, err := websocket.DefaultDialer.Dial(httptestServer.URL, nil)

// Send/Receive messages
ws.WriteJSON(envelope)
ws.ReadJSON(&response)
```

**See**: `server_test.go` for examples

## Quick Reference

| Want to... | Look at |
|------------|---------|
| Understand WS protocol | messages.go |
| Add new message type | messages.go + server.go |
| Fix connection bug | server.go:88, session_runner.go |
| Fix game loop bug | session_runner.go:241 |
| Fix disconnect handling | forfeit.go |
| Fix lobby bug | server.go:223, server/lobby.go |
| Test WebSocket | *_test.go files |
| See InputProvider impl | session_provider.go |
| See full architecture | backend/docs/ARCHITECTURE.md |
