# Package: engine

## Knowledge Context
Prerequisites:
- **`backend/models`**: Understand the Action/Role definitions (defs.go:6, roles.go:12).
- **`backend/engine/state`**: Know the GameState structure (state/gamestate.go:47).
- **State Machines**: Familiarity with turn-based state progressions.

## Core Responsibility
The `engine` package is the "Game Kernel". It manages the state machine of a single game instance. It is responsible for enforcing rules, managing the turn lifecycle (Action -> Challenge -> Counter -> Resolution), and mutating the game state (`GameState`) safely. It is **transport-agnostic**; it doesn't know about WebSockets or HTTP.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                   ENGINE STRUCTURE                       │
└─────────────────────────────────────────────────────────┘

Game (game.go:19)
├─ CurrentState (state.GameState)  ◀── "One Source of Truth"
├─ TurnState (state.TurnState)     ◀── Per-turn scratchpad
├─ History ([]GameState)           ◀── Replay data
├─ ActiveFlow (*actions.FlowRunner)◀── Multi-step actions
└─ rng (*rand.Rand)                ◀── Deterministic or random

InputProvider (orchestrator_types.go:24)
├─ RequestAction()    ◀── Get player's main action
├─ ChallengeResponses() ◀── Poll for challenges
├─ CounterResponses() ◀── Poll for counters
└─ RequestStep()      ◀── Multi-step input

RunTurn (orchestrator_turn.go:29)
├─ 1. Solicit Action ◀── calls InputProvider.RequestAction
├─ 2. Validate ◀── actions.ValidatePlayerAction (validators.go:9)
├─ 3. Pay Cost ◀── actions.PayActionCost (validators.go:77)
├─ 4. Challenge Window ◀── runChallengeWindow (orchestrator_challenge.go)
├─ 5. Counter Window ◀── collectCounters (orchestrator_counters.go:32)
└─ 6. Execute ◀── actions.ApplyAction (apply.go:101)
```

## Inner Workings and Logic
The engine operates on a request/response cycle driven by the `RunTurn` function.
1.  **State Container**: `Game` holds the current `GameState` and the `History`.
    - **See**: `game.go:19` for struct definition
    - **See**: `game.go:41` for NewGame constructor
    - **History**: Array of GameState snapshots for replay
2.  **Turn Orchestration** (`RunTurn` at orchestrator_turn.go:29):
    - **Step 1: Solicitation**: Asks the current player for an `PlayerAction` via `InputProvider`.
      - **Location**: orchestrator_turn.go:54
      - **Interface**: InputProvider.RequestAction (orchestrator_types.go:24)
    - **Step 2: Validation**: Checks basic validity (funds, turn order) using `actions.ValidatePlayerAction`.
      - **Location**: orchestrator_turn.go:67
      - **Function**: validators.go:9
    - **Step 3: Cost**: Deducts coins immediately (refunded later if action is canceled).
      - **Location**: orchestrator_turn.go:71
      - **Function**: validators.go:77 (PayActionCost)
      - **Refund**: Line 89-90 if action canceled
    - **Step 4: Challenge Window**: If the action claims a role, all other players are polled for a Challenge.
      - **Location**: orchestrator_turn.go:81
      - **Function**: runChallengeWindow (orchestrator_challenge.go)
      - **Logic**: If challenger wins, actor loses card; if actor wins, challenger loses card
      - **Refund**: Action cost refunded if challenge successful (line 88-90)
    - **Step 5: Counter Window**: If the action is offensive, targeted players are polled for Counters.
      - **Location**: orchestrator_turn.go:98
      - **Function**: collectCounters (orchestrator_counters.go:32)
      - **Logic**: Counters can be challenged (line 124), each has its own mini-challenge window
      - **Post-counters**: Applied after main action (line 166-170)
    - **Step 6: Application**: If surviving, the action is executed via `actions.ApplyAction`.
      - **Location**: orchestrator_turn.go:163 (main), line 167 (counters)
      - **Function**: actions/apply.go:101
      - **Result**: TurnResult with logs and state changes
3.  **Turn Resolution**: Result returned, Game.History updated
    - **Location**: orchestrator_turn.go:185
    - **Logging**: Public logs (line 176), private logs (line 178), hand info (line 182)

## Key Architectural Patterns
- **Orchestrator Pattern**: `RunTurn` coordinates complex multi-step interactions (challenges, responses) into a single atomic "Turn".
  - **Challenge/Counter Recursion**: Counters can be challenged, creating nested challenge windows
- **Dependency Injection**: `InputProvider` is an interface. The engine doesn't know *how* inputs arrive (CLI, Test Mock, WebSocket), only that they do.
  - **Implementations**:
    - Test: `stubProvider` in orchestrator_turn_test.go:12
    - Production: `sessionProvider` in server/ws/session_provider.go
- **Functional State Mutation**: While not purely functional, the design encourages passing `GameState` snapshot copies to inputs to prevent accidental mutation outside the engine.
  - **Snapshot**: state/gamestate.go:66
  - **History**: game.go:22

## Critical Components

### Game (game.go:19)
The root object for a match.
```go
type Game struct {
    CurrentState state.GameState     // Current game state
    TurnState    state.TurnState     // Per-turn scratchpad
    History      []state.GameState   // Replay history
    ActiveFlow   *actions.FlowRunner // Multi-step action coordinator
    rng          *rand.Rand          // Random number generator
}
```

**Key Methods**:
- `NewGame()` (game.go:30) - Constructor with optional RNG
- `CheckGameOver()` (game.go:83) - Returns winner if only one player has cards
- `IncrementTurn()` (game.go:112) - Advances to next active player

### InputProvider (orchestrator_types.go:24)
The bridge to the outside world. The "sensors" for the Engine.
```go
type InputProvider interface {
    RequestAction(ctx context.Context, actor int, gs *state.GameState) (*models.PlayerAction, error)
    ChallengeResponses(ctx context.Context, window ChallengeWindow) (<-chan ChallengeDecision, error)
    CounterResponses(ctx context.Context, window CounterWindow) (<-chan CounterDecision, error)
    RequestStep(ctx context.Context, req state.StepRequest) (any, error)
}
```

**Design Note**: GameState passed to RequestAction is read-only. Implementations must not mutate it.

**Implementations**:
- `stubProvider` (orchestrator_turn_test.go:12) - Test mock
- `sessionProvider` (server/ws/session_provider.go) - WebSocket bridge

### RunTurn (orchestrator_turn.go:29)
The blocking function that runs exactly one turn.
```go
func (g *Game) RunTurn(ctx context.Context, input InputProvider, clock Clock, cfg TurnConfig) (*TurnResult, error)
```

**Returns**: TurnResult with:
- MainApplied bool - Whether main action executed
- MainCanceled bool - Whether main action was blocked
- ChallengeResults []ChallengeResult - Challenge outcomes
- CounterResults []CounterResult - Counter outcomes
- Logs []string - Public game log
- PrivateLogs map[int][]string - Per-player private info

### GameState (state/gamestate.go:47)
The struct holding players, coins, hands, and deck.
**See**: engine/state/README.md for full details.

### TurnResult (orchestrator_types.go - implicitly defined by usage)
A detailed report of what occurred during the turn, including public logs and private revelations.

**Structure** (inferred from orchestrator_turn.go):
```go
type TurnResult struct {
    Main            *models.PlayerAction
    MainApplied     bool
    MainCanceled    bool
    ChallengeResults []ChallengeResult
    CounterResults   []CounterResult
    Logs            []string
    PrivateLogs     map[int][]string
}
```

## Supporting Components

### Clock (orchestrator_types.go:43)
Abstraction for time (timeouts).
```go
type Clock interface {
    After(d time.Duration) <-chan time.Time
}
```

**Implementations**:
- `RealClock` (orchestrator_types.go:47) - Uses time.After
- `testClock` (orchestrator_turn_test.go:52) - Never fires (deterministic)

**Usage**: Challenge and counter timeouts in RunTurn

### ChallengeWindow / CounterWindow (orchestrator_types.go:69,79)
Context for challenge/counter phases.
```go
type ChallengeWindow struct {
    Kind               ChallengeKind
    Action             *models.PlayerAction
    ActorIndex         int
    ClaimedRole        models.Role
    AllowedChallengers []int
}

type CounterWindow struct {
    MainAction     *models.PlayerAction
    AllowedPlayers []int
    AllowedActions []models.ActionID
}
```

**Passed to**: InputProvider.ChallengeResponses / CounterResponses

### ChallengeDecision / CounterDecision
Results from challenge/counter polling.
```go
type ChallengeDecision struct {
    ChallengerIndex int
    Pass            bool  // true = pass (no challenge), false = challenge
}

type CounterDecision struct {
    PlayerIndex int
    Command     *models.PlayerAction
}
```

## File Structure

```
engine/
├── game.go                    # Game struct, NewGame, CheckGameOver
├── orchestrator_turn.go       # RunTurn - main turn orchestration
├── orchestrator_challenge.go  # Challenge logic
├── orchestrator_counters.go   # Counter collection logic
├── orchestrator_helpers.go    # Helper functions
├── orchestrator_types.go      # Interfaces and types (InputProvider, etc.)
├── challenge.go               # Challenge resolution logic
├── turn.go                    # Turn-related helpers
└── input/                     # Input implementations
    └── websocket/             # WebSocket InputProvider (see server/ws/)
```

## Turn Flow Detailed

```
START
  │
  ├─▶ Save state to History (snapshot)
  │
  ├─▶ InputProvider.RequestAction() ──▶ Wait for player input
  │    │
  │    └─▶ Returns PlayerAction
  │
  ├─▶ ValidatePlayerAction() ──▶ Check valid
  │    │
  │    └─▶ Error? Return early
  │
  ├─▶ PayActionCost() ──▶ Deduct coins
  │
  ├─▶ IF action has Role:
  │    │
  │    ├─▶ runChallengeWindow()
  │    │    │
  │    │    ├─▶ InputProvider.ChallengeResponses()
  │    │    │
  │    │    ├─▶ IF challenge received:
  │    │    │    │
  │    │    │    ├─▶ Check actor has claimed role
  │    │    │    │
  │    │    │    ├─▶ IF actor has role:
  │    │    │    │    ├─▶ Challenger loses card
  │    │    │    │    └─▶ Action allowed to continue
  │    │    │    │
  │    │    │    └─▶ IF actor bluffing:
  │    │    │         ├─▶ Actor loses card
           │         ├─▶ Refund cost
  │    │    │         └─▶ Action canceled
  │    │    │
  │    │    └─▶ Return ChallengeResult
  │    │
  │    └─▶ IF challenge successful: Skip to end
  │
  ├─▶ collectCounters()
  │    │
  │    ├─▶ InputProvider.CounterResponses()
  │    │
  │    ├─▶ FOR each counter:
  │    │    │
  │    │    ├─▶ IF counter has Role:
  │    │    │    │
  │    │    │    └─▶ runChallengeWindow() (recursive!)
  │    │    │         Same logic as above
  │    │    │
  │    │    ├─▶ IF counter not canceled:
  │    │    │    │
  │    │    │    ├─▶ IF counter cancels main:
  │    │    │    │    ├─▶ Apply counter effect
  │    │    │    │    ├─▶ Main action canceled
  │    │    │    │    └─▶ Break (stop collecting)
  │    │    │    │
  │    │    │    └─▶ IF counter is post-counter:
  │    │    │         └─▶ Queue for later execution
  │    │    │
  │    │    └─▶ IF counter canceled by challenge:
  │    │         ├─▶ Refund cost
  │    │         └─▶ Continue to next counter
  │    │
  │    └─▶ Return counters and logs
  │
  ├─▶ IF main action allowed AND not canceled:
  │    │
  │    ├─▶ ApplyAction() ──▶ Execute main action
  │    │
  │    └─▶ FOR each queued post-counter:
  │         └─▶ ApplyAction() ──▶ Execute counter
  │
  ├─▶ Compile TurnResult
  │    ├─▶ Public logs
  │    ├─▶ Private logs
  │    └─▶ Hand info for each player
  │
  └─▶ RETURN TurnResult
```

## External Dependencies
- **`backend/models`**: For types (defs.go:6, roles.go:12).
- **`backend/engine/state`**: For GameState (gamestate.go:47).
- **`backend/actions`**: Logic for applying specific effects (registry.go:102, validators.go:9, apply.go:101).
  - Coupling here is intentional: engine drives the flow, actions update the state.

## Cross-Reference Guide

| Component | Location | Used By | Purpose |
|-----------|----------|---------|---------|
| Game | game.go:19 | session_runner.go | Root game object |
| NewGame | game.go:30 | lobby.go | Constructor |
| CheckGameOver | game.go:83 | session_runner.go | Win condition |
| IncrementTurn | game.go:112 | RunTurn | Turn advancement |
| InputProvider | orchestrator_types.go:24 | RunTurn, tests | Input abstraction |
| RequestAction | orchestrator_types.go:27 | RunTurn:54 | Get player action |
| ChallengeResponses | orchestrator_types.go:31 | runChallengeWindow | Poll challenges |
| CounterResponses | orchestrator_types.go:35 | collectCounters | Poll counters |
| RequestStep | orchestrator_types.go:38 | ApplyAction | Multi-step input |
| RunTurn | orchestrator_turn.go:29 | session_runner.go:241 | Main orchestrator |
| collectCounters | orchestrator_counters.go:32 | RunTurn:98 | Counter collection |
| runChallengeWindow | orchestrator_challenge.go | RunTurn:81 | Challenge logic |
| TurnConfig | orchestrator_types.go:54 | RunTurn | Timeout durations |
| Clock | orchestrator_types.go:43 | RunTurn | Time abstraction |

## Testing

**Pattern**: Use `stubProvider` to mock InputProvider
```go
// orchestrator_turn_test.go:12
type stubProvider struct {
    action     *models.PlayerAction
    challenges []ChallengeDecision
    counters   []CounterDecision
    steps      []any
}
```

**Test Types**:
- `TestRunTurnIncome` - Simple action
- `TestRunTurnChallengeBluffCancels` - Challenge successful
- `TestRunTurnChallengeProvedAllowsAction` - Challenge failed
- `TestRunTurnCounterCancelsMain` - Counter blocks action
- `TestRunTurnExchangeStep` - Multi-step action

**See**: orchestrator_turn_test.go for full test suite

## Quick Reference

| Want to... | Look at |
|------------|---------|
| Understand turn flow | orchestrator_turn.go:29 |
| Add new action | actions/README.md |
| Fix challenge bug | orchestrator_challenge.go |
| Fix counter bug | orchestrator_counters.go:32 |
| Understand InputProvider | orchestrator_types.go:24 |
| See game lifecycle | game.go:19 |
| Test turn logic | orchestrator_turn_test.go |
| See WebSocket integration | server/ws/session_runner.go:201 |
| Replay/history | game.go:22 (History field) |
