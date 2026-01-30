# Package: actions

## What This Is

The **game rules engine**. Every action in the game—from Income to Assassinate—is defined, validated, and executed here.

Think of this as the "Rulebook in Code". While the engine handles the flow of turns, this package handles what each move actually does.

## Core Responsibility

- Define all possible actions (ActionRegistry)
- Validate actions before execution (validators)
- Execute action effects (apply handlers)
- Handle multi-step actions (ActionFlow)

## Inner Workings and Logic

```
┌─────────────────────────────────────────────────────────┐
│                  ACTIONS ARCHITECTURE                    │
└─────────────────────────────────────────────────────────┘

ActionRegistry (registry.go:102)
    │
    ├─▶ ActionDef (registry.go:41)
    │    ├─ ID: ActionID (models/defs.go:6)
    │    ├─ Kind: MainAction/CounterAction (defs.go:38)
    │    ├─ Role: models.Role (roles.go:12)
    │    ├─ Cost: int
    │    ├─ Handler: ActionHandler
    │    └─ Validator: ActionValidator
    │
    └─▶ Maps ActionID → ActionDef

Validation (validators.go:9)
    │
    ├─▶ ValidatePlayerAction()
    │    ├─ Check action exists
    │    ├─ Check source index valid
    │    ├─ Check payload type correct
    │    └─ Run action-specific validator
    │
    └─▶ Action-specific validators
         ├─ validateCoup (needs 7 coins)
         ├─ validateSteal (needs 2 coins, target has coins)
         ├─ validateAccuse (needs 4 coins)
         └─ etc.

Execution (apply.go)
    │
    ├─▶ ApplyAction()
    │    ├─ Get ActionDef from registry
    │    ├─ Call Handler function
    │    └─ Handle multi-step via ActionFlow
    │
    └─▶ Handler functions
         ├─ applyIncome (simple)
         ├─ applySteal (transfers coins)
         ├─ applyCoup (forces discard via StepRequest)
         ├─ applyExchange (draws cards, StepRequest for selection)
         └─ etc.

ActionFlow (flow.go)
    │
    ├─▶ Manages multi-step actions
    │    ├─ Exchange: draw cards → select which to keep
    │    ├─ Coup: target loses card → select which to discard
    │    └─ Assassinate: same as Coup
    │
    └─▶ Pauses/resumes action execution
```

## Critical Components

### ActionDef (registry.go:41)

The blueprint for a single action:

```go
type ActionDef struct {
    ID             models.ActionID      // e.g., "steal"
    Kind           models.ActionKind    // MainAction or CounterAction
    Role           models.Role          // Required role (or empty)
    AllowedAgainst []models.ActionID    // For counters: what they block
    MultiCounter   bool                 // Multiple players can counter?
    Cost           int                  // Coins to pay
    Handler        ActionHandler        // Effect function
    Validator      ActionValidator      // Validation function
}
```

**Key insight**: Separation of Validator (can we?) and Handler (what happens?)

### ActionRegistry (registry.go:102)

The rulebook - every possible move:

```go
var ActionRegistry = map[models.ActionID]ActionDef{
    models.Income: {
        ID:        models.Income,
        Kind:      models.MainAction,
        Handler:   applyIncome,
        Validator: validateNoPayload,
    },
    models.Steal: {
        ID:        models.Steal,
        Kind:      models.MainAction,
        Role:      models.Thief,
        Handler:   applySteal,
        Validator: validateSteal,
    },
    models.BlockSteal: {
        ID:             models.BlockSteal,
        Kind:           models.CounterAction,
        Role:           models.Thief,
        AllowedAgainst: []models.ActionID{models.Steal},
        Handler:        applyBlock,
        Validator:      validateNoPayload,
    },
    // ... all actions defined here
}
```

**Organization**: Grouped by game concept, not alphabetically
1. Base actions (Income, ForeignAid)
2. Role actions (Tax, Steal, etc.)
3. Counters (BlockSteal, BlockTerrorist, etc.)
4. Special (Coup, Exchange, Escape)

### Validation (validators.go)

**ValidatePlayerAction** (validators.go:9):
```go
func ValidatePlayerAction(s *state.GameState, cmd *models.PlayerAction) error
```

Checks:
1. Command not nil
2. Source index in bounds
3. ActionID exists in registry
4. Payload type matches action requirements
5. Action-specific validation (coins, targets, etc.)

**Action-Specific Validators**:
- `validateCoup` - 7 coins required, valid target
- `validateSteal` - 2 coins required, target has coins
- `validateAccuse` - 4 coins required, valid target
- `validateTax` - Someone has 7+ coins (taxable)
- `validateTaxBusinessWoman` - MainAction is Business
- `validateNoPayload` - No payload allowed (Income, etc.)

**See**: validators.go for full list

### Execution (apply.go)

**ApplyAction** (apply.go:101):
```go
func ApplyAction(flow *ActionFlow, s *state.GameState, 
    turn *state.TurnState, log *state.TurnLog, 
    cmd *models.PlayerAction) error
```

Flow:
1. Lookup ActionDef in registry
2. Call Handler with all context
3. Handler may use ActionFlow for multi-step
4. Return error if anything fails

**Handler Functions** (apply.go):
- `applyIncome` - Add 1 coin
- `applyForeignAid` - Add 2 coins
- `applyCoup` - Pay 7, force target discard
- `applySteal` - Transfer 2 coins
- `applyTax` - Add 3 coins
- `applyBusiness` - Add 4 coins
- `applyAssassinate` - Pay 3, force discard
- `applyInvestigate` - Peek at card, optionally swap
- `applyAccuse` - Guess role, discard if correct or pay if wrong
- `applyExchange` - Draw 2, keep 2 (multi-step)
- `applyBlock` - Counter action (cancels main)
- `applyTaxBusinessWoman` - Tax 1 coin from Business player
- `applyEscape` - Pay 9 to survive
- `applyPass` - No-op

### Multi-Step Actions (flow.go)

Some actions need player input mid-execution:

**Examples**:
- **Exchange**: Draw 2 cards → wait for selection → return rest
- **Coup**: Target must choose which card to discard
- **Assassinate**: Same as Coup

**ActionFlow** manages this:
```go
type ActionFlow struct {
    Requests  chan state.StepRequest  // Send requests
    Responses chan any                // Receive responses
    ctx       context.Context
}
```

**Pattern**:
```go
func applyExchange(flow *ActionFlow, ...) error {
    // 1. Draw cards
    s.DrawCards(actor, 2)
    
    // 2. Request player selection
    req := state.StepRequest{
        ActionID: cmd.ID,
        Kind:     state.StepCardSelect,
        Actor:    actor,
        Count:    2,  // Keep 2 cards
        Context:  state.ContextExchange,
    }
    flow.Requests <- req
    
    // 3. Pause and wait for response
    resp := <-flow.Responses
    
    // 4. Process response (indices to keep)
    keep := resp.([]int)
    s.ReturnCardsToDeck(actor, keep)
    
    return nil
}
```

**See**: `applyExchange` in apply.go for full implementation

## Adding a New Action

Follow this workflow to add a completely new action:

### Step 1: Define ActionID
```go
// models/defs.go:6
const (
    // ... existing actions ...
    MyNewAction ActionID = "my_new_action"
)
```

### Step 2: Add to Registry
```go
// actions/registry.go:102
var ActionRegistry = map[models.ActionID]ActionDef{
    // ... existing actions ...
    models.MyNewAction: {
        ID:        models.MyNewAction,
        Kind:      models.MainAction,  // or CounterAction
        Role:      models.SomeRole,     // or "" for no role
        Cost:      3,                   // or 0
        Handler:   applyMyNewAction,
        Validator: validateMyNewAction,
    },
}
```

### Step 3: Implement Handler
```go
// actions/apply.go

func applyMyNewAction(flow *ActionFlow, s *state.GameState, 
    turn *state.TurnState, log *state.TurnLog, 
    cmd *models.PlayerAction) error {
    
    // Simple action - no multi-step
    actor := cmd.SourceIndex
    
    // Do the thing
    _, err := s.AddCoins(actor, 3)
    if err != nil {
        return err
    }
    
    // Log it
    log.Public = append(log.Public, 
        fmt.Sprintf("Player %d gained 3 coins", actor))
    
    return nil
}
```

### Step 4: Implement Validator
```go
// actions/validators.go

func validateMyNewAction(s *state.GameState, cmd *models.PlayerAction) error {
    // Check generic requirements
    if err := requireTargetPayload(cmd); err != nil {
        return err
    }
    
    payload := cmd.Payload.(models.TargetPayload)
    
    // Check target valid
    if err := validateTarget(s, payload.TargetIndex); err != nil {
        return err
    }
    
    // Check action-specific rules
    if s.Players[payload.TargetIndex].Coins < 2 {
        return fmt.Errorf("target has no coins to steal")
    }
    
    return nil
}
```

### Step 5: Add Payload (if needed)
```go
// models/payloads.go

type MyNewPayload struct {
    TargetIndex int  `json:"target_index"`
    Amount      int  `json:"amount"`
}
```

Update validator to check payload type:
```go
func requireMyNewPayload(cmd *models.PlayerAction) error {
    if _, ok := cmd.Payload.(models.MyNewPayload); !ok {
        return fmt.Errorf("my_new_action requires MyNewPayload")
    }
    return nil
}
```

### Step 6: Test It
```go
// tests/actions_test.go

func TestApplyMyNewAction(t *testing.T) {
    s := makeState(
        state.Player{ID: 0, Name: "A", Coins: 2, Hand: cards},
        state.Player{ID: 1, Name: "B", Coins: 5, Hand: cards},
    )
    
    turn := &state.TurnState{}
    log := &state.TurnLog{}
    flow := actions.NewActionFlow()
    
    cmd := &models.PlayerAction{
        ID:          models.MyNewAction,
        SourceIndex: 0,
        Payload:     models.TargetPayload{TargetIndex: 1},
    }
    
    err := actions.ApplyAction(flow, s, turn, log, cmd)
    if err != nil {
        t.Fatalf("apply failed: %v", err)
    }
    
    // Verify effects
    if s.Players[0].Coins != 5 {
        t.Fatalf("expected 5 coins, got %d", s.Players[0].Coins)
    }
    if s.Players[1].Coins != 2 {
        t.Fatalf("expected 2 coins, got %d", s.Players[1].Coins)
    }
}
```

### Step 7: Integration Test
```go
// engine/orchestrator_turn_test.go

func TestRunTurnMyNewAction(t *testing.T) {
    game := &Game{CurrentState: state.GameState{
        TurnNumber:         1,
        CurrentPlayerIndex: 0,
        Players: []state.Player{
            {ID: 0, Name: "A", Coins: 2, Hand: cards},
            {ID: 1, Name: "B", Coins: 5, Hand: cards},
        },
    }}
    
    cmd := &models.PlayerAction{
        ID:          models.MyNewAction,
        SourceIndex: 0,
        Payload:     models.TargetPayload{TargetIndex: 1},
    }
    
    provider := &stubProvider{action: cmd}
    res, err := game.RunTurn(context.Background(), provider, testClock{}, TurnConfig{})
    
    if err != nil {
        t.Fatalf("RunTurn: %v", err)
    }
    if !res.MainApplied {
        t.Fatal("expected main action applied")
    }
}
```

## Cross-Reference Guide

| Component | Location | Used By | Purpose |
|-----------|----------|---------|---------|
| ActionDef | registry.go:41 | ActionRegistry | Blueprint for actions |
| ActionRegistry | registry.go:102 | orchestrator_turn.go:67 | Look up action definitions |
| ValidatePlayerAction | validators.go:9 | orchestrator_turn.go:67 | Entry validation |
| ApplyAction | apply.go:101 | orchestrator_turn.go:163 | Entry execution |
| ActionFlow | flow.go | apply.go | Multi-step coordination |
| PayActionCost | validators.go:77 | orchestrator_turn.go:71 | Deduct coins |
| ActionCost | validators.go:92 | PayActionCost | Get cost from registry |

## Handler Pattern Examples

### Simple Action (Income)
```go
func applyIncome(_ *ActionFlow, s *state.GameState, ... cmd *models.PlayerAction) error {
    // No multi-step, ignore flow parameter
    _, err := s.AddCoins(cmd.SourceIndex, 1)
    return err
}
```

### Multi-Step Action (Exchange)
```go
func applyExchange(flow *ActionFlow, s *state.GameState, ... cmd *models.PlayerAction) error {
    actor := cmd.SourceIndex
    
    // Draw cards
    if err := s.DrawCards(actor, 2); err != nil {
        return err
    }
    
    // Request selection
    flow.Requests <- state.StepRequest{
        ActionID: cmd.ID,
        Kind:     state.StepCardSelect,
        Actor:    actor,
        Count:    len(s.Players[actor].Hand) - 2, // Keep all but 2
        Context:  state.ContextExchange,
    }
    
    // Wait for response
    resp := <-flow.Responses
    returnIndices := resp.([]int)
    
    // Return unselected cards
    return s.ReturnCardsToDeck(actor, returnIndices)
}
```

### Action with Target (Steal)
```go
func applySteal(_ *ActionFlow, s *state.GameState, ... cmd *models.PlayerAction) error {
    payload := cmd.Payload.(models.TargetPayload)
    actor, target := cmd.SourceIndex, payload.TargetIndex
    
    // Transfer 2 coins (or whatever target has)
    stealAmount := min(2, s.Players[target].Coins)
    
    if _, err := s.AddCoins(target, -stealAmount); err != nil {
        return err
    }
    if _, err := s.AddCoins(actor, stealAmount); err != nil {
        return err
    }
    
    return nil
}
```

## Design Principles

1. **Separation of Concerns**: Validation before execution
2. **Registry Pattern**: All metadata in one place
3. **Handler Functions**: Effect logic isolated and testable
4. **ActionFlow**: Linear code for multi-step actions (not callbacks)
5. **Fail Fast**: Validate everything before state mutation

## Testing

**Unit Tests** (tests/actions_test.go):
- Test each handler independently
- Use makeState() to create game state
- Verify state changes

**Integration Tests** (orchestrator_turn_test.go):
- Test full turn flow with stubProvider
- Verify challenge/counter behavior
- Test multi-step actions

**Patterns**:
```go
// Table-driven
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        s := makeState(tt.players...)
        err := actions.ValidatePlayerAction(s, tt.cmd)
        // Assert
    })
}
```

## Quick Reference

| Want to... | Look at |
|------------|---------|
| See all actions | registry.go:102 |
| Add new action | This README above |
| Understand validation | validators.go:9 |
| See action effects | apply.go handlers |
| Multi-step actions | flow.go, applyExchange |
| Test actions | tests/actions_test.go |
| Fix action bug | Check both validator and handler |
| Change action cost | registry.go (Cost field) |
| Change action effect | apply.go (handler function) |
