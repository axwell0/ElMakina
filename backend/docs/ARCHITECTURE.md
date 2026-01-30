# ElMakina Backend Architecture

This is your navigation hub. Start here to understand the system.

## 5-Minute Overview

ElMakina is a multiplayer card game server (similar to Coup) handling:
- **Lobby Management**: Players create/join rooms
- **Game Engine**: Turn-based mechanics with challenges/counters
- **WebSocket Transport**: Real-time communication
- **Replay System**: Game recording and playback

The codebase prioritizes **testability** through interface-based design and clear separation between pure game logic and I/O.

---

## System Architecture (High Level)

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                              │
│                     (Browser/WebSocket)                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ WebSocket Upgrade
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     SERVER LAYER                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │   server.go  │  │session_runner│  │   lobby.go   │            │
│  │   :88        │  │   :201       │  │   :67        │            │
│  └──────────────┘  └──────────────┘  └──────────────┘            │
│       │                   │               │                      │
│       │                   │               │                      │
│       ▼                   ▼               ▼                      │
│  ┌──────────────────────────────────────────────────────┐       │
│  │              InputProvider Interface                  │       │
│  │          (orchestrator_types.go:24)                   │       │
│  └──────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Implements
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     ENGINE LAYER                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │   game.go    │  │orchestrator  │  │ orchestrator │            │
│  │   :19        │  │   _turn.go   │  │ _counters.go │            │
│  │              │  │   :29        │  │   :32        │            │
│  └──────────────┘  └──────────────┘  └──────────────┘            │
│       │                   │               │                      │
│       │                   │               │                      │
│       ▼                   ▼               ▼                      │
│  ┌──────────────────────────────────────────────────────┐       │
│  │              GameState (gamestate.go:47)              │       │
│  │              TurnState (turn_state.go)                │       │
│  └──────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Mutates
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     ACTIONS LAYER                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │ registry.go  │  │ validators.go│  │   apply.go   │            │
│  │   :102       │  │   :9         │  │   :101       │            │
│  │ ActionDefs   │  │ Validation   │  │ Execution    │            │
│  └──────────────┘  └──────────────┘  └──────────────┘            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Uses
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     MODELS LAYER                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │  defs.go     │  │   roles.go   │  │ payloads.go  │            │
│  │   :6         │  │   :12        │  │              │            │
│  │ ActionID     │  │    Role      │  │  Payloads    │            │
│  └──────────────┘  └──────────────┘  └──────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

---

## Turn Flow Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                      TURN LIFECYCLE                             │
└────────────────────────────────────────────────────────────────┘

  ┌──────────────┐
  │ Player Turn  │
  │   Begins     │
  └──────┬───────┘
         │
         ▼
  ┌─────────────────────────────────────┐
  │ 1. SOLICIT ACTION                   │
  │    RequestAction()                  │
  │    orchestrator_turn.go:54          │
  └──────┬──────────────────────────────┘
         │
         ▼
  ┌─────────────────────────────────────┐
  │ 2. VALIDATE & PAY COST              │
  │    ValidatePlayerAction()          │
  │    validators.go:9                  │
  │    actions/apply.go:116             │
  └──────┬──────────────────────────────┘
         │
         ▼
  ┌─────────────────────────────────────┐
  │ 3. CHALLENGE WINDOW                 │
  │    (If action has Role)             │
  │    runChallengeWindow()             │
  │    orchestrator_turn.go:81          │
  │                                     │
  │    ┌───────┐     ┌───────┐         │
  │    │Chall. │────▶│Lose   │         │
  │    │Called │     │Card   │         │
  │    └───────┘     └───────┘         │
  │       │                             │
  │       ▼                             │
  │    ┌───────┐     ┌───────┐         │
  │    │Chall. │────▶│Action │         │
  │    │Failed │     │Allowed│         │
  │    └───────┘     └───────┘         │
  └──────┬──────────────────────────────┘
         │
         ▼
  ┌─────────────────────────────────────┐
  │ 4. COUNTER WINDOW                   │
  │    (If action is offensive)         │
  │    collectCounters()                │
  │    orchestrator_counters.go:32      │
  │                                     │
  │    ┌──────────────┐                 │
  │    │BlockSteal    │                 │
  │    │BlockTerrorist│                 │
  │    │TaxBusiness   │                 │
  │    └──────────────┘                 │
  │                                     │
  │    Each counter triggers its own    │
  │    challenge window (recursion!)    │
  └──────┬──────────────────────────────┘
         │
         ▼
  ┌─────────────────────────────────────┐
  │ 5. RESOLUTION                       │
  │    ┌────────────┐ ┌────────────┐   │
  │    │Main Action │ │Counters    │   │
  │    │Applied     │ │Applied     │   │
  │    │apply.go:101│ │(if any)    │   │
  │    └────────────┘ └────────────┘   │
  └──────┬──────────────────────────────┘
         │
         ▼
  ┌──────────────┐
  │ Turn Ends    │
  │ Advance to   │
  │ Next Player  │
  └──────────────┘
```

---

## Package Dependency Map

```
                    ┌─────────────────────┐
                    │   backend/models    │
                    │   (Foundation)      │
                    └─────────────────────┘
                           ▲      ▲
                           │      │
        ┌──────────────────┘      └──────────────────┐
        │                                            │
┌───────────────────┐                      ┌───────────────────┐
│ backend/engine/   │                      │ backend/actions/  │
│ state/            │                      │                   │
│ (Data layer)      │                      │ (Game logic)      │
└───────────────────┘                      └───────────────────┘
        ▲                                            ▲
        │                                            │
        └──────────────────┬─────────────────────────┘
                           │
                    ┌─────────────────────┐
                    │  backend/engine/    │
                    │  (Game engine)      │
                    └─────────────────────┘
                           ▲
                           │
                    ┌─────────────────────┐
                    │ backend/server/ws/  │
                    │ (Network layer)     │
                    └─────────────────────┘
```

**Dependency Rules:**
- `models` has NO dependencies (foundation types)
- `engine/state` depends only on `models`
- `actions` depends on `models` and `engine/state`
- `engine` depends on `models`, `engine/state`, and `actions`
- `server/ws` depends on all of the above

---

## File Cross-Reference Matrix

### Core Entry Points

| File | Line | What It Is | Used By | See Also |
|------|------|------------|---------|----------|
| `models/defs.go` | 6 | ActionID enum | All packages | registry.go:102 |
| `models/roles.go` | 12 | Role enum | registry.go, deck.go | orchestrator_turn.go:79 |
| `engine/game.go` | 19 | Game struct | session_runner.go | orchestrator_turn.go:29 |
| `engine/state/gamestate.go` | 47 | GameState | game.go, apply.go | Snapshot() method |
| `engine/orchestrator_types.go` | 24 | InputProvider | session_runner.go, tests | Challenge/Counter windows |
| `engine/orchestrator_turn.go` | 29 | RunTurn | session_runner.go:241 | Turn flow logic |
| `actions/registry.go` | 102 | ActionRegistry | orchestrator_turn.go | Action definitions |
| `actions/validators.go` | 9 | ValidatePlayerAction | orchestrator_turn.go:67 | Action validation |
| `actions/apply.go` | 101 | ApplyAction | orchestrator_turn.go | Action execution |
| `server/ws/server.go` | 88 | ServeHTTP | main.go | WebSocket upgrade |
| `server/ws/session_runner.go` | 201 | sessionRunner | server.go | Game loop bridge |
| `server/lobby.go` | 67 | LobbyManager | server.go | Lobby lifecycle |

### State Management

| File | Line | Purpose | Related Files |
|------|------|---------|---------------|
| `engine/state/gamestate.go` | 47 | Game state container | game.go, apply.go |
| `engine/state/turn_state.go` | - | Per-turn scratchpad | orchestrator_turn.go |
| `engine/state/deck.go` | - | Card deck management | game.go (NewGame) |
| `engine/state/types.go` | - | Card, Player structs | gamestate.go |
| `engine/state/turn_log.go` | - | Logging structures | orchestrator_turn.go |

### Action System

| File | Line | Purpose | Related Files |
|------|------|---------|---------------|
| `actions/registry.go` | 102 | Action definitions | defs.go |
| `actions/validators.go` | 9 | Input validation | orchestrator_turn.go:67 |
| `actions/apply.go` | 101 | Effect execution | registry.go |
| `actions/flow.go` | - | Multi-step actions | apply.go (Exchange) |
| `actions/apply_more_test.go` | - | Additional tests | actions_test.go |

---

## Common Tasks

### "I need to add a new role/action"

```
Step 1: Define the ActionID
    └─▶ models/defs.go:6
        Add to ActionID const list

Step 2: Add Role (if new character)
    └─▶ models/roles.go:12
        Add to Role const and Roles slice

Step 3: Define Payload (if needed)
    └─▶ models/payloads.go
        Add new Payload type

Step 4: Register the Action
    └─▶ actions/registry.go:102
        Add to ActionRegistry map
        Set: ID, Kind, Role, Handler, Validator, Cost

Step 5: Implement Handler
    └─▶ actions/apply.go
        Add applyXxx() function
        See existing handlers for patterns

Step 6: Implement Validator
    └─▶ actions/validators.go
        Add validateXxx() function
        See validateCoup, validateSteal for examples

Step 7: Test It
    └─▶ tests/actions_test.go
        Add table-driven test case
        See TestApplyCoupDiscardsCard for pattern
```

### "I need to fix a game rule bug"

1. Identify the action: Check `models/defs.go:6` for ActionID
2. Find the handler: Look in `actions/registry.go:102` for the Handler field
3. Check validation: Look in `actions/validators.go` for validation logic
4. Check orchestration: If bug involves timing (challenge/counter), check `orchestrator_turn.go:29`
5. Write test: Add regression test in appropriate test file

### "I need to understand the WebSocket flow"

```
Entry: server/ws/server.go:88 (ServeHTTP)
  │
  ├─▶ Upgrades HTTP to WebSocket
  ├─▶ Reads hello message (player registration)
  ├─▶ Creates clientConn
  ├─▶ Starts readLoop in goroutine
  │
  └─▶ Lobby messages: handleLobbyMessage()
      └─▶ Create, Join, Start lobby
          └─▶ StartLobby calls session_runner.go:201
              └─▶ sessionRunner.runGameLoop()
                  └─▶ Calls game.RunTurn() for each turn
                      └─▶ Implements InputProvider interface
                          ├─▶ RequestAction (waits for player input)
                          ├─▶ ChallengeResponses (challenge window)
                          └─▶ CounterResponses (counter window)
```

### "I need to write a test"

```
Unit Test (Action Logic):
  └─▶ tests/actions_test.go
      Pattern: Table-driven with t.Parallel()
      Use: makeState(), makeCard() helpers
      See: TestApplyStealTransfersTwoCoins

Integration Test (Turn Flow):
  └─▶ backend/engine/orchestrator_turn_test.go
      Pattern: stubProvider implements InputProvider
      Use: testClock for deterministic timing
      See: TestRunTurnIncome

WebSocket Test:
  └─▶ backend/server/ws/*_test.go
      Pattern: httptest server + WebSocket dial
      See: server_test.go for examples
```

---

## Testing Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    TEST PYRAMID                              │
└─────────────────────────────────────────────────────────────┘

     ┌─────────┐
     │  E2E    │  WebSocket integration
     │  Tests  │  backend/server/ws/*_test.go
     └────┬────┘
          │
     ┌────▼────┐
     │Turn Flow│  Full turn orchestration
     │ Tests   │  orchestrator_turn_test.go
     └────┬────┘
          │
     ┌────▼────┐
     │ Action  │  Individual action handlers
     │ Tests   │  tests/actions_test.go
     └────┬────┘
          │
     ┌────▼────┐
     │ State   │  GameState methods
     │ Tests   │  engine/state/gamestate_test.go
     └─────────┘
```

**Test Patterns:**
- **Table-driven**: Define test cases as structs, iterate with t.Run()
- **stubProvider**: Implements InputProvider for deterministic testing
- **makeState()**: Helper to create GameState for tests
- **t.Parallel()**: Run tests in parallel (safe with loop var capture)
- **Race detection**: `go test -race ./...` catches concurrency bugs

---

## Design Principles

1. **Interface-Based Testing**: InputProvider interface allows testing without WebSockets
2. **Pure Functions Where Possible**: Actions are deterministic given state
3. **Immutable Snapshots**: GameState.Snapshot() for replay/undo
4. **Fail Fast**: Validation before mutation
5. **Small Interfaces**: InputProvider has only 4 methods

---

## Quick Navigation

| Want to... | Start Here |
|------------|------------|
| Add a new action | actions/README.md#AddingNewActions |
| Fix a bug | Check ARCHITECTURE.md File Matrix above |
| Understand turn flow | orchestrator_turn.go:29 |
| Write a test | tests/README.md#TestPatterns |
| Connect WebSocket | server/ws/README.md#Protocol |
| Understand types | models/README.md |
| Debug state issues | state/README.md#MutationRules |

---

## ASCII Diagram Maintenance

**Note**: Line numbers in cross-references may shift as code evolves. The general pattern remains stable. If you find a broken reference:

1. Find the function/struct name
2. Use `grep -rn "func RunTurn" backend/` to find new location
3. Update the reference

**Last Updated**: [Generated automatically on commit]
