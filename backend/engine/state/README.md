# Package: engine/state

## Knowledge Context

This module owns the in-memory game state, deck behavior, and turn logs. It is the canonical data model for the engine.

## Core Responsibility

The `state` package is the **data layer** of the game engine. It contains the "one source of truth" for game state and all state mutation methods.

Think of this as the database of the game - it holds the data and enforces integrity constraints, but knows nothing about game rules or networking.

## Responsibilities

- Define `GameState`, `Player`, `Deck`, and turn log structures.
- Provide safe state mutation helpers (coin changes, discards, draws, swaps).
- Track transient turn data such as `PendingRequest` and discard events.

## Inner Workings and Logic

```
┌─────────────────────────────────────────────────────────┐
│                  STATE STRUCTURES                        │
└─────────────────────────────────────────────────────────┘

GameState (gamestate.go:47)          TurnState (turn_state.go)
├─ TurnNumber                        ├─ TaxUsed
├─ Players []Player                  ├─ ExchangeTaken
├─ Deck                              └─ [per-turn flags]
├─ CurrentPlayerIndex
├─ DiscardEvents        ──────▶ TurnLog (turn_log.go)
└─ PendingRequest                   ├─ Public []string
                                    └─ Private map[int][]string

Player (types.go)
├─ ID
├─ Name
├─ Hand []Card
└─ Coins

Card (types.go)
├─ ID
└─ Role (models/roles.go:12)

Deck ([]Card)
- Draw pile for the game
```

## Critical Components

### GameState (gamestate.go:47)

The **single source of truth** for a moment in time:

```go
type GameState struct {
    TurnNumber         int
    Players            []Player
    Deck               Deck
    CurrentPlayerIndex int
    
    // Transient (not serialized)
    DiscardEvents  []DiscardEvent
    PendingRequest *StepRequest
    seed           *rand.Rand
}
```

**Key Methods** (gamestate.go):
- `Snapshot()` (line 66) - Deep copy for replay/persistence
- `AddCoins()` (line 118) - Enforces non-negative and MaxCoins cap
- `DiscardFromHand()` (line 87) - Removes card, returns to deck
- `DrawCards()` (line 135) - Deals from deck
- `ReturnCardsToDeck()` (line 147) - Used by Exchange action
- `SwapCard()` (line 185) - Used by Investigate action
- `DrainDiscardEvents()` (line 106) - Clear and return events

**Mutation Rules**:
1. Always use methods - never mutate fields directly
2. Methods validate indices before modifying
3. Methods enforce game rules (coin limits, MaxCoins=12)
4. Errors return if constraints violated

**Invariants and Sharp Edges**:
- `MaxCoins` (gamestate.go:9) caps the coin total; callers should not bypass `AddCoins`
- `DiscardEvents` are per-turn and must be drained to avoid stale UI
- `PendingRequest` is transient and should not be serialized

### TurnState (turn_state.go)

Temporary scratchpad for current turn:

```go
type TurnState struct {
    TaxUsed        bool  // Tax taken this turn?
    ExchangeTaken  bool  // Exchange used this turn?
    // Additional per-turn flags
}
```

**Reset**: Called at turn end (`game.go:132`)

### TurnLog (turn_log.go)

Tracks what happened during a turn:

```go
type TurnLog struct {
    Public  []string            // Everyone sees
    Private map[int][]string    // Per-player secrets (e.g., investigation results)
}
```

**Usage**: Orchestrator populates, sent to clients at turn end. See: `orchestrator_turn.go:177`

### StepRequest (gamestate.go:26)

Request for multi-step action input:

```go
type StepRequest struct {
    ActionID models.ActionID  // Which action (models/defs.go:6)
    Kind     StepKind         // Type of input needed
    Actor    int              // Who must respond
    Count    int              // How many items to select
    Context  StepContext      // discard / exchange
    Options  []string         // Valid options
}
```

**Used by**: Exchange (card selection), Coup/Assassinate (discard selection)

**Flow**:
```
Action handler ──▶ Set PendingRequest ──▶ InputProvider.RequestStep()
                                              │
                                              ▼
                                    Player responds with selection
                                              │
                                              ▼
                                    Action resumes
```

**See**: `actions/flow.go` for ActionFlow mechanism

## State Mutation Patterns

### Correct Way:
```go
// Enforces rules, returns error if invalid
coins, err := gs.AddCoins(playerIndex, delta)
if err != nil {
    return err
}
```

### Incorrect Way:
```go
// Bypasses validation!
gs.Players[playerIndex].Coins += delta
```

## Cross-Reference Guide

| Component | Definition | Mutated By | Read By |
|-----------|------------|------------|---------|
| GameState | gamestate.go:47 | apply.go handlers | orchestrator_turn.go |
| TurnState | turn_state.go | Reset each turn | orchestrator_turn.go |
| TurnLog | turn_log.go | apply.go | orchestrator_turn.go:177 |
| StepRequest | gamestate.go:26 | apply.go | InputProvider (orchestrator_types.go:38) |
| Player | types.go | GameState methods | All packages |
| Card | types.go | Deck methods | Hand management |
| Deck | deck.go | Draw/return methods | NewGame, Exchange |

## Key Files

- `gamestate.go`: `GameState` and mutation helpers
- `deck.go`: deck creation, shuffling, and draw logic
- `turn_log.go`: public/private logs for UI rendering
- `turn_state.go`: per-turn temporary state
- `types.go`: shared domain types (Player, Card)

## File Dependency Map

```
engine/state/
    │
    ├─ gamestate.go ──▶ GameState, state methods
    │       │
    │       ├─▶ types.go (Player, Card)
    │       └─▶ models/ (Role, ActionID from defs.go:6, roles.go:12)
    │
    ├─ turn_state.go ──▶ TurnState
    │
    ├─ turn_log.go ──▶ TurnLog
    │
    ├─ deck.go ──▶ Deck management
    │       └─▶ Uses models/Role for card roles
    │
    └─ types.go ──▶ Basic types

Called by:
    ├─ engine/game.go (NewGame at line 41)
    ├─ actions/apply.go (handlers like applySteal)
    └─ tests/ (test helpers like makeState)
```

## Snapshots and Replays

**Why Snapshots?**
```go
func (g *GameState) Snapshot() *GameState  // gamestate.go:66
```

1. **Replay System**: History of every turn stored (see: server/replay/)
2. **Undo Protection**: Can restore previous state
3. **Test Isolation**: Test can't accidentally mutate shared state
4. **Thread Safety**: Snapshots are immutable copies

**Implementation**:
- Deep copy of Players slice (and their Hands)
- Copy of Deck
- DiscardEvents and PendingRequest set to nil (transient)

**See**: `engine/game.go:22` - History []GameState

## Design Decisions

### Methods vs Direct Access

All mutations go through methods to:
- Validate bounds (prevent index panics)
- Enforce rules (no negative coins, MaxCoins=12 cap)
- Record events (DiscardEvents for logging)
- Return errors (fail fast)

### MaxCoins Cap

```go
const MaxCoins = 12  // gamestate.go:9
```

Coins capped at 12 to prevent runaway economy. Handled in `AddCoins()` method at line 118.

### Transient Fields

`DiscardEvents` and `PendingRequest` are not serialized:
- They're temporary per-turn state
- Replays don't need them
- They complicate JSON serialization
- `Snapshot()` clears them at line 79-80

## Testing

**Test Files**:
- `gamestate_test.go` - State method tests
- `turn_log` tests - Logging behavior
- `tests/actions_test.go` - Action effect tests using state

**Test Pattern**:
```go
// In tests/
s := makeState(
    state.Player{ID: 0, Name: "A", Hand: cards, Coins: 2},
    state.Player{ID: 1, Name: "B", Hand: cards, Coins: 2},
)

// Mutate and verify
s.AddCoins(0, 3)
if s.Players[0].Coins != 5 { t.Fatal(...) }
```

## Quick Reference

| Want to... | Look at |
|------------|---------|
| Add coins | gamestate.go:118 (AddCoins) |
| Discard card | gamestate.go:87 (DiscardFromHand) |
| Draw cards | gamestate.go:135 (DrawCards) |
| Create snapshot | gamestate.go:66 (Snapshot) |
| See player structure | types.go |
| See deck logic | deck.go |
| See turn log | turn_log.go |
| Debug state bug | Check mutation methods enforce rules |
| Test state | gamestate_test.go |
