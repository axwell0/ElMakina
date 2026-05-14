# El Makina — Codebase Reading Plan

Work through these layers in dependency order. Each group builds on the one before it.

---

## Layer 1 — Domain Vocabulary (30 min)

Zero imports from the rest of the project. Read these first so every other file makes sense.

| File | What to understand |
|---|---|
| `engine/core/types/roles.go` | The role enum — Thief, Terrorist, etc. The "cards" players hold. |
| `engine/core/types/actions.go` | Every `ActionID` constant and the `PlayerAction` command struct. This is the engine's canonical input type. |
| `engine/core/types/payloads.go` | `TargetPayload` and `AccusePayload` — the polymorphic data carried by actions. A nil `Payload` means the action needs no extra data. |
| `engine/core/config/config.go` | `GameConfig` — starting coins, deck copies, card counts. The single source of tunable parameters. |

---

## Layer 2 — Pure Game State (45 min)

State that the engine mutates, isolated from networking.

| File | What to understand |
|---|---|
| `engine/core/state/player.go` | `Player` struct: `Hand`, `Coins`, elimination logic. |
| `engine/core/state/deck.go` | Deck construction and `Draw`. |
| `engine/core/state/state.go` | `GameState` — the root mutable object. `Snapshot()` produces a deep copy every time a turn starts (key to the copy-on-commit pattern). |
| `engine/core/state/state_test.go` | Walk the tests to see exactly what invariants the state enforces. |

---

## Layer 3 — Action Registry & Application (60 min)

The heart of game logic: what actions are legal, what they cost, and how they mutate state.

| File | What to understand |
|---|---|
| `engine/core/actions/eligibility.go` | `MainAction(id)` and `CounterAction(id)` — the registry. Each `ActionDef` carries `Role`, `RequiresTarget`, `CancelsMain`, etc. |
| `engine/core/actions/validators.go` | `ValidateMainAction` — pre-flight checks before coins are spent. |
| `engine/core/actions/apply.go` | `ApplyAction` — the actual state mutation for each action ID. Two-phase actions (`Exchange`, discard) use the `if ctx.Step == nil` pattern: first call sets up a prompt, second call applies the response. |
| `engine/core/actions/payload_helpers.go` | Helpers for extracting typed payloads safely. |
| `engine/core/actions/runtime.go` | The `StepRequest`/`StepResponse` mechanism — used when an action needs extra card-selection input mid-turn. |
| `engine/core/actions/actions_test.go` + `registry_test.go` | Property-style tests that enumerate all actions. Useful reference for expected behavior. |

---

## Layer 4 — Turn Orchestration (60 min)

How a complete turn flows, including challenge and counter windows.

| File | What to understand |
|---|---|
| `engine/turn/result.go` | `TurnResult`, `TurnLog`, `CounterSummary`, `ChallengeSummary` — the structured output the turn produces. |
| `engine/runtime/turn/input.go` | `InputProvider` interface — the seam between the engine and the session layer. The engine only calls these methods; it has no WebSocket knowledge. |
| `engine/runtime/turn/counters.go` | Counter window collection logic. |
| `engine/runtime/turn/challenge.go` | Challenge window: who can challenge, card reveal, discard logic. |
| `engine/runtime/turn/actions.go` | `declaredAction` — binds a command to its cost; `payCost`/`refundCost`. |
| **`engine/runtime/turn/service.go`** | **The turn pipeline.** Read the package-level doc comment first, then trace `Run` → `declareMainAction` → `resolveMainChallenge` → `handleCounters` → `applySurvivingActions`. |
| `engine/runtime/turn/service_test.go` + `scenario_test.go` | End-to-end scenario tests with a mock `InputProvider`. |

---

## Layer 5 — Application / Session Layer (60 min)

The stateful aggregate that bridges the engine with the WebSocket world.

| File | What to understand |
|---|---|
| `internal/apperrors/errors.go` | Sentinel errors (`ErrNoPrompt`, `ErrStalePrompt`, `ErrInvalidPhase`, …). |
| `internal/app/session/prompt.go` | `Prompt` struct and `PromptKind` — the typed questions the engine asks players. |
| `internal/app/session/client_session.go` | `ClientSession` — one player's durable seat: `DeviceID`, `ResumeToken`, `LastSeenAt`. |
| **`internal/app/session/game_session.go`** | **Core concurrency design.** `StartTurn` clones `GameState` via `Snapshot()`, runs the entire turn against the clone in a goroutine, and commits only on success. The typed channels (`mainAction`, `challenge`, `counter`, `step`) and `promptEvents` channel are the interface between engine and WebSocket handler. |
| `internal/app/session/lobby.go` | `Lobby` aggregate: lifecycle (`PhaseLobby → PhaseInTurn → PhaseGameOver`), player joining, readiness, leader tracking. All public methods are mutex-protected. |
| `internal/app/session/command.go` | `CommandEnvelope` and typed command payloads decoded from WebSocket messages. |
| `internal/app/session/*_test.go` | Lobby + game session tests cover the full join → start → turn → game-over path. |

---

## Layer 6 — Store & Persistence (20 min)

| File | What to understand |
|---|---|
| `internal/store/store.go` | `RoomStore` and `SessionStore` interfaces + `Bundle` (the two together). |
| `internal/store/memory/store.go` | In-memory implementation — the only store wired up in production right now. |

---

## Layer 7 — WebSocket Server (60 min)

The transport layer. Read these roughly in order.

| File | What to understand |
|---|---|
| `server/ws/messages.go` | All outbound message types (`RoomSyncMessage`, `PromptEnvelope`, `TurnResultMessage`, `GameOverMessage`). |
| `server/ws/errors.go` | Typed transport errors (`roomNotFoundError`, `sessionNotFoundError`, etc.). |
| `server/ws/helpers.go` | Auth helpers: `bearerToken`, `websocketResumeAuth`. |
| `server/ws/connection_registry.go` | `ConnectionRegistry` — maps `sessionID → playerIndex → LiveClient`. The thread-safe registry of live sockets. |
| `server/ws/projection.go` | `BuildRoomSyncMessage` — translates a `LobbySnapshot` into the player-specific JSON view (hides other players' hands). |
| `server/ws/prompt_serialization.go` | Serializes `Prompt` objects into typed WebSocket envelopes per prompt kind. |
| **`server/ws/room.go`** | **The room runtime.** Serializes mutations through `Actor`, runs `startGameLoop`, and most importantly `RunTurn` — the `select` loop that reacts to `promptEvents` while the engine goroutine runs. |
| `server/ws/server.go` | HTTP handlers: `HandleWebSocket`, `HandleListResumableGames`, `HandleRestartRoom`, `HandleDeleteRoom`. This is where HTTP requests become rooms and sessions. |
| `server/ws/*_test.go` | Integration tests using `testHandler` — an in-process WebSocket round-trip harness. |

---

## Layer 8 — Infrastructure & Entry Points (20 min)

| File | What to understand |
|---|---|
| `internal/transport/http/server.go` | Route registration: wires the `ws.Server` into `net/http`. |
| `internal/transport/ws/connection.go` + `handler.go` | Low-level WebSocket upgrade and read/write loops. |
| `internal/app/room/actor.go` | `Actor` — the bounded channel queue that serializes all lobby mutations (fan-in pattern). |
| `internal/api/types.go` | Shared API response types between HTTP and WebSocket. |
| `server/logging/logger.go` | `slog`-based structured logger setup. |
| `server/cmd/server/main.go` | Entry point: reads `ELMAKINA_HTTP_ADDR`, builds `ws.NewServer()`, wraps it in `httpserver.NewHandler`. |

---

## Layer 9 — Frontend (15 min)

| File | What to understand |
|---|---|
| `web/src/App.tsx` | The single React component — currently a minimal shell. |
| `web/src/styles/tokens.css` | Design tokens. |
| `web/vite.config.ts` | Vite dev server with proxy to `:8080`. |

---

## Key Data Flow

```
Browser WebSocket message
        │
        ▼
server/ws/server.go  (HandleWebSocket)
        │  upgrades to WS, reads frames
        ▼
server/ws/room.go  (handleIncoming → HandleCommand / HandleReady / HandleStart)
        │  serialized through Actor
        ▼
internal/app/session/lobby.go  (SubmitCommand / MarkReady / StartGame)
        │
        ▼
internal/app/session/game_session.go  (submitCommandFromSession → channel send)
        │  typed Go channel
        ▼
engine/runtime/turn/service.go  (Run — blocks reading channels)
        │  calls InputProvider (game_session implements it)
        ▼
engine/core/actions/  (validate → apply → mutate GameState copy)
        │
        ▼
turn.TurnResult  →  broadcast back through room.broadcastTurnResult
```

---

## Key Invariants

**Copy-on-commit:** `StartTurn` clones `GameState` via `Snapshot()`, runs the entire turn against the clone in a background goroutine, and only writes it back to `s.state` if no error occurred. A failed turn leaves the authoritative state untouched.

**Prompt/channel pipeline:** The engine runs in a goroutine and blocks on typed Go channels (`mainAction`, `challenge`, `counter`, `step`). `Room.RunTurn` sits in a `select` loop reacting to `promptEvents` and routing WebSocket messages into those channels. This keeps the engine fully synchronous and testable while the WebSocket layer handles all async concerns.

**Actor serialization:** Every public method on `Room` that mutates lobby state funnels through `r.do()` / `r.doBool()`, which serialize work through a bounded channel queue (`Actor`). This means the lobby is always mutated by one goroutine at a time, even when multiple WebSocket handlers fire concurrently.
