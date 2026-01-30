# Package: models

## Knowledge Context
Prerequisites:
- Basic understanding of the "El Makina" (Coup variant) game rules.
- Familiarity with Go structs and JSON tagging.

## Core Responsibility
The `models` package serves as the canonical domain language dictionary for the entire system. It defines the shared vocabulary—Actions, Roles, and Payloads—that strictly decouples the `engine` (rules), `server` (networking), and `client` (UI). It contains **no logic**, only type definitions and constant values.

## Inner Workings and Logic
This package is purely declarative.
- **Action Registry**: A static map (`ActionRegistry`) acts as the single source of truth for all metadata about actions (e.g., is it a main action? is it a counter? what role does it claim? what is the cost? which handler/validator keys apply?).
  - **Location**: Actually defined in `actions/registry.go:102` (not in this package)
  - **Uses**: The ActionIDs defined here in `defs.go:6`
- **Role Parsing**: String-to-Role conversion logic ensures that invalid role strings are caught at the boundary.
  - **Function**: `ParseRole()` in `roles.go:36`
- **Payloads**: Usage of `interface{}`/`any` in `PlayerAction` requires concrete types like `TargetPayload` to be defined here for safe casting in the engine.
  - **Types**: `TargetPayload`, `AccusePayload` in `payloads.go`

## Key Architectural Patterns
- **Shared Type Definition**: This package is imported by almost every other package (`engine`, `server`, `actions`, `client`). It prevents cyclic dependencies by remaining a leaf node in the dependency graph.
- **Registry Pattern**: `ActionRegistry` centralizes game rule metadata, allowing valid moves to be queried without instantiating a game state.

## Critical Components

### ActionID Enum (defs.go:6)
String constants for all possible moves (e.g., "tax", "coup").

```
┌─────────────────────────────────────────────────────────┐
│                   ACTION IDs                             │
└─────────────────────────────────────────────────────────┘

Base Actions (no role required):
  Income           - Take 1 coin
  ForeignAid       - Take 2 coins (blockable)
  Coup             - Pay 7 coins, force discard

Role Actions (can be challenged):
  Tax              - Take 3 coins (TaxCollector)
  Business         - Take 4 coins (Businesswoman)
  Investigate      - Peek at card (Policewoman)
  Accuse           - Guess role (Colonel)
  Assassinate      - Pay 3 coins, force discard (Terrorist)
  Steal            - Take 2 coins from target (Thief)
  Exchange         - Swap cards with deck (Politician)

Counter Actions:
  BlockForeignAid  - Block ForeignAid (TaxCollector)
  BlockSteal       - Block Steal (Thief)
  BlockTerrorist   - Block Assassinate (Colonel)
  BlockPolice      - Block Investigate (Policewoman)
  TaxBusinessWoman - Tax Businesswoman (TaxCollector)
  Escape           - Pay 9 coins to survive Coup/Assassinate
  Pass             - No operation
```

**See also**: `actions/registry.go:102` where ActionIDs are mapped to definitions.

### Role Enum (roles.go:12)
String constants for player roles (e.g., "Colonel", "Thief").

```
Businesswoman - Can take 4 coins (vulnerable to Tax)
TaxCollector  - Can block ForeignAid, Tax Businesswoman
Policewoman   - Can investigate, block investigations
Colonel       - Can accuse, block assassinations
Terrorist     - Can assassinate
Thief         - Can steal, block steals
Politician    - Can exchange cards
```

**Challenge Mechanic**: When claiming a role (e.g., "I use TaxCollector"), others can challenge. Must reveal that role or lose a card.

**See also**: `engine/orchestrator_turn.go:79` for challenge validation.

### PlayerAction (defs.go:29)
The uniform envelope for any player move:

```go
type PlayerAction struct {
    ID          ActionID  // What action (defs.go:6)
    SourceIndex int       // Who is acting (index in Players slice)
    Payload     any       // Action-specific data (payloads.go)
    MainAction  ActionID  // Original action (for counters)
}
```

**Flow**:
```
WebSocket ──▶ server/ws/server.go:88 ──▶ Parse JSON
                │
                ▼
    PlayerAction ──▶ session_runner.go:201
                │
                ▼
    engine.RunTurn() ──▶ Validate → Challenge → Counter → Execute
```

### Payload Types (payloads.go)

**TargetPayload** - Used by: Coup, Steal, Assassinate, Investigate
```go
type TargetPayload struct {
    TargetIndex int  // Which player to target
}
```
**Validation**: `actions/validators.go:119` checks target bounds.

**AccusePayload** - Used by: Accuse action
```go
type AccusePayload struct {
    TargetIndex int   // Who to accuse
    Guess       Role  // Which role (roles.go:12)
}
```

**NoPayload** - Used by: Income, ForeignAid, Tax, Exchange, etc.

### ActionKind (defs.go:38)
Distinguishes main actions from counters:
```go
MainAction    - Your turn action
CounterAction - Response to main action
```

**Used by**: `actions/registry.go:46` to set ActionDef.Kind.

## External Dependencies
- **None**: This package relies only on the Go standard library.
- **Used by**: Every other package (engine, actions, server, tests)

## Cross-Reference Guide

| Type | Definition | Primary Usage | Related Files |
|------|------------|---------------|---------------|
| ActionID | defs.go:6 | registry.go:102 | orchestrator_turn.go:54 |
| Role | roles.go:12 | registry.go, deck.go | orchestrator_turn.go:79 |
| PlayerAction | defs.go:29 | server/ws/, engine/ | validators.go:119 |
| TargetPayload | payloads.go | apply.go | validators.go |
| AccusePayload | payloads.go | apply.go | validators.go |
| ActionKind | defs.go:38 | registry.go | orchestrator_turn.go:67 |

## Adding New Types

### Add ActionID:
```go
// defs.go:6
const (
    MyNewAction ActionID = "my_new_action"
)
```

### Add Role:
```go
// roles.go:12
const (
    MyNewRole Role = "MyNewRole"
)
// Add to Roles slice in roles.go:24
```

### Add Payload:
```go
// payloads.go
type MyNewPayload struct {
    FieldName string `json:"field_name"`
}
```

**Next Steps**: See `actions/README.md#AddingNewActions` for complete workflow.

## Design Notes

### Why Strings for Enums?
- Human-readable JSON over WebSocket
- Debugging - logs show "steal" not "7"
- Forward compatibility

### Why `any` for Payload?
- Flexibility - different actions need different data
- Type safety enforced by validators (validators.go:119)
- JSON unmarshaling at WebSocket boundary

## Quick Reference

| Want to... | Look at |
|------------|---------|
| See all moves | defs.go:6 |
| See all roles | roles.go:12 |
| Understand action data | payloads.go |
| Add new type | This section above |
| Debug action issues | actions/README.md |
