# Package: tests

## What This Is

Integration-style tests that exercise core flows across modules (engine, server, ws, and CLI paths). This is your **TDD Playground** - write tests first, then implement.

## Core Responsibility

- Test action semantics and step flows
- Test lobby lifecycle and session behavior
- Test orchestrator turn sequencing and timeouts
- Test WebSocket protocol and server behaviors
- Provide test helpers and patterns for other packages

## How to Run

From repo root:

```bash
# Run all tests
go test ./...

# Run with race detection (recommended for concurrent code)
go test -race ./...

# Run specific package
go test ./backend/tests/...

# Run with coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Run specific test
go test -run TestApplySteal ./backend/tests/
```

## TDD Philosophy

### Test First Workflow

```
1. Write test that fails (RED)
   └─▶ Define expected behavior

2. Write minimal code to pass (GREEN)
   └─▶ Make test pass

3. Refactor (REFACTOR)
   └─▶ Clean up while keeping tests green

4. Repeat
```

### Testing Principles

1. **Test behavior, not implementation** - Test what the code does, not how it does it
2. **One concept per test** - Each test should verify one thing
3. **Descriptive names** - Test names should explain the scenario
4. **Arrange-Act-Assert** - Clear structure in each test
5. **Table-driven** - Use test tables for multiple scenarios

## Test Organization

```
tests/
├── actions_test.go          # Action handler tests
├── flow_test.go             # ActionFlow tests
├── lobby_test.go            # Lobby lifecycle tests
├── orchestrator_test.go     # Turn orchestration tests
├── session_test.go          # Session tests
├── store_test.go            # Storage tests
├── swap_card_test.go        # Card swap tests
├── forfeit_test.go          # Forfeit behavior tests
└── ws_config_test.go        # WebSocket config tests
```

## Critical Components

### Test Helpers

**makeCard** (actions_test.go:11):
```go
func makeCard(id int, role models.Role) state.Card {
    return state.Card{ID: id, Role: role}
}
```

**makeState** (actions_test.go:15):
```go
func makeState(players ...state.Player) *state.GameState {
    return &state.GameState{
        TurnNumber:         1,
        Players:            players,
        Deck:               state.Deck{},
        CurrentPlayerIndex: 0,
    }
}
```

**applyWithResponse** (actions_test.go:24):
```go
// Helper for multi-step actions (Exchange, Coup, etc.)
func applyWithResponse(t *testing.T, s *state.GameState, turn *state.TurnState, 
    log *state.TurnLog, cmd *models.PlayerAction, 
    assertReq func(state.StepRequest), response any) error
```

### Table-Driven Tests

**Pattern** (actions_test.go:40):
```go
func TestValidatePlayerAction(t *testing.T) {
    t.Parallel()
    
    tests := []struct {
        name    string
        state   *state.GameState
        cmd     *models.PlayerAction
        wantErr bool
    }{
        {name: "nil command", state: baseState, cmd: nil, wantErr: true},
        {name: "source out of range", state: baseState, cmd: &models.PlayerAction{ID: models.Income, SourceIndex: 99}, wantErr: true},
        // ... more cases
    }
    
    for _, tt := range tests {
        tt := tt  // Capture loop variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            err := actions.ValidatePlayerAction(tt.state, tt.cmd)
            if tt.wantErr && err == nil {
                t.Fatalf("expected error")
            }
            if !tt.wantErr && err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
        })
    }
}
```

**Key points**:
- Define test cases as structs
- Use `t.Run()` for subtests
- Use `t.Parallel()` for speed
- Capture loop variable to avoid race

### StubProvider (orchestrator_turn_test.go:12)

Implements `InputProvider` for deterministic testing:

```go
type stubProvider struct {
    action     *models.PlayerAction
    challenges []ChallengeDecision
    counters   []CounterDecision
    steps      []any
}

func (s *stubProvider) RequestAction(...) (*models.PlayerAction, error) {
    return s.action, nil
}

func (s *stubProvider) ChallengeResponses(...) (<-chan ChallengeDecision, error) {
    ch := make(chan ChallengeDecision, 1)
    if len(s.challenges) > 0 {
        ch <- s.challenges[0]
        s.challenges = s.challenges[1:]
    }
    close(ch)
    return ch, nil
}
// ... implement other methods
```

**Usage**:
```go
provider := &stubProvider{
    action:     &models.PlayerAction{ID: models.Income, SourceIndex: 0},
    challenges: []ChallengeDecision{{ChallengerIndex: 1, Pass: false}},
}
res, err := game.RunTurn(context.Background(), provider, testClock{}, TurnConfig{})
```

### testClock (orchestrator_turn_test.go:52)

Deterministic clock that never fires (for timeout testing):
```go
type testClock struct{}

func (testClock) After(d time.Duration) <-chan time.Time {
    ch := make(chan time.Time)
    return ch  // Never sends, so timeouts never trigger
}
```

## Test Patterns by Layer

### Action Tests (actions_test.go)

Test individual action effects:

```go
func TestApplyStealTransfersTwoCoins(t *testing.T) {
    // ARRANGE
    s := makeState(
        state.Player{ID: 0, Name: "A", Hand: cards, Coins: 1},
        state.Player{ID: 1, Name: "B", Hand: cards, Coins: 5},
    )
    turn := &state.TurnState{}
    log := &state.TurnLog{}
    flow := actions.NewActionFlow()
    cmd := &models.PlayerAction{
        ID:          models.Steal,
        SourceIndex: 0,
        Payload:     models.TargetPayload{TargetIndex: 1},
    }
    
    // ACT
    err := actions.ApplyAction(flow, s, turn, log, cmd)
    
    // ASSERT
    if err != nil {
        t.Fatalf("apply steal error: %v", err)
    }
    if s.Players[0].Coins != 3 {
        t.Fatalf("source coins: got %d want 3", s.Players[0].Coins)
    }
    if s.Players[1].Coins != 3 {
        t.Fatalf("target coins: got %d want 3", s.Players[1].Coins)
    }
}
```

### Turn Flow Tests (orchestrator_turn_test.go)

Test full turn orchestration:

```go
func TestRunTurnChallengeBluffCancels(t *testing.T) {
    // Setup game with bluffing player
    game := &Game{CurrentState: state.GameState{
        Players: []state.Player{
            {ID: 0, Name: "A", Hand: []state.Card{makeCard(1, models.TaxCollector)}}, // Claiming Thief but has TaxCollector
            {ID: 1, Name: "B", Hand: []state.Card{makeCard(2, models.Thief)}},
        },
    }}
    
    // Player 0 tries to steal (claims Thief role)
    cmd := &models.PlayerAction{
        ID:          models.Steal,
        SourceIndex: 0,
        Payload:     models.TargetPayload{TargetIndex: 1},
    }
    
    // Player 1 challenges
    provider := &stubProvider{
        action:     cmd,
        challenges: []ChallengeDecision{{ChallengerIndex: 1, Pass: false}},
        steps:      []any{[]int{0}}, // Discard first card
    }
    
    // Run turn
    res, err := game.RunTurn(context.Background(), provider, testClock{}, TurnConfig{})
    
    // Assert action canceled, player 0 lost card
    if res.MainApplied {
        t.Fatal("expected main action canceled")
    }
    if len(game.CurrentState.Players[0].Hand) != 0 {
        t.Fatal("expected actor to lose a card")
    }
}
```

### Multi-Step Action Tests

Test actions requiring player input:

```go
func TestApplyExchangeReturnsCards(t *testing.T) {
    s := makeState(
        state.Player{ID: 0, Name: "A", Hand: []state.Card{
            makeCard(1, models.Politician), 
            makeCard(2, models.Thief),
        }},
    )
    s.Deck = state.Deck{
        makeCard(3, models.Colonel), 
        makeCard(4, models.TaxCollector),
    }
    
    cmd := &models.PlayerAction{ID: models.Exchange, SourceIndex: 0}
    
    // Use applyWithResponse helper
    err := applyWithResponse(t, s, turn, log, cmd, 
        func(req state.StepRequest) {
            // Assert request is correct
            if req.ActionID != models.Exchange {
                t.Fatalf("unexpected request: %+v", req)
            }
        },
        []int{0, 1}, // Response: keep cards at indices 0 and 1
    )
    
    if err != nil {
        t.Fatalf("apply exchange error: %v", err)
    }
    // Assert final state
}
```

## Coverage Focus

### Action Semantics
- Each action handler tested (apply.go)
- Validation rules tested (validators.go)
- Edge cases (insufficient funds, invalid targets)

### Step Flows
- Multi-step actions (Exchange, Coup, Assassinate)
- ActionFlow coordination (flow.go)
- Request/response handling

### Lobby Lifecycle
- Create, join, start lobbies
- Player disconnect/reconnect
- Leader changes
- Lobby state transitions

### Orchestrator
- Turn sequencing
- Challenge windows
- Counter windows
- Timeout handling

### WebSocket
- Protocol message handling
- Connection lifecycle
- Error scenarios

## Testing Best Practices

### DO
- Use `t.Parallel()` for independent tests
- Capture loop variable: `tt := tt`
- Use `t.Helper()` in test helpers
- Test edge cases (nil inputs, boundary conditions)
- Use descriptive test names: `TestApplyStealTransfersTwoCoins`
- Check both success and error cases

### DON'T
- Test implementation details (private functions)
- Use sleeps or time.After in tests
- Share mutable state between tests
- Skip error checking: `if err != nil { t.Fatal(err) }`

### Assertions
```go
// Good - clear failure message
if got != want {
    t.Fatalf("coins: got %d want %d", got, want)
}

// Good - table-driven with name
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test
    })
}

// Good - parallel tests
func TestXxx(t *testing.T) {
    t.Parallel()
    // test
}
```

## Cross-Reference Guide

| Test Type | Location | Tests | Helpers |
|-----------|----------|-------|---------|
| Action handlers | actions_test.go | ApplyAction | makeState, makeCard |
| Action validation | actions_test.go | ValidatePlayerAction | makeState |
| Turn flow | orchestrator_turn_test.go | RunTurn | stubProvider, testClock |
| Lobby | lobby_test.go | Lobby lifecycle | - |
| Session | session_test.go | Session management | - |
| WebSocket | server/ws/*_test.go | Protocol | httptest server |
| State | engine/state/*_test.go | GameState methods | - |
| Flow | flow_test.go | ActionFlow | - |

## Running Tests in Practice

```bash
# Quick check during development
go test ./backend/tests/... -v

# Before committing - full suite with race detection
go test -race ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks (if any)
go test -bench=. ./...

# Specific failing test with verbose output
go test -v -run TestApplySteal ./backend/tests/
```

## Integration with CI

Recommended CI commands:
```bash
go test -race ./...           # Catch race conditions
go vet ./...                  # Static analysis
go fmt ./...                  # Format check
go test -cover ./...          # Coverage check
```

## Notes

- Tests assume deterministic RNG or seeded clocks where needed
- Replay storage tests require database configuration when enabled
- Use `stubProvider` for deterministic InputProvider testing
- Use `testClock` for timeout testing (never fires)
- Each test should be independent (no shared state)

## Quick Reference

| Want to... | Look at |
|------------|---------|
| Write action test | actions_test.go pattern |
| Write turn flow test | orchestrator_turn_test.go |
| Write WebSocket test | server/ws/*_test.go |
| Mock InputProvider | stubProvider in orchestrator_turn_test.go |
| Create game state | makeState helper |
| Run tests | `go test ./...` |
| Run with race check | `go test -race ./...` |
| Fix flaky test | Check for shared state or timing issues |
| Add test for new action | actions_test.go examples |
