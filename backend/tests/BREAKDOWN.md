# Tests Package Breakdown: The Proving Ground of Truth

## The Core Philosophy: Tests Are the Foundation of Confidence

In a system where the engine claims to be deterministic, where state mutations claim to be guarded, where actions claim to follow rules—how do we know these claims are true? **Tests are the proof.** They are the experiments that verify our hypotheses about the code. Without tests, the architecture is just a beautiful theory. With tests, it becomes trustworthy engineering.

The tests package is not an afterthought or a chore. It is the **proving ground** where we demonstrate that the system works as designed. It is where we capture edge cases, document behavior, and provide confidence for refactoring. A tested system is a system you can change without fear.

### The Mental Model: The Scientific Laboratory

Imagine a pristine laboratory where scientists conduct experiments to verify the laws of physics. Each experiment is carefully designed: setup the initial conditions, apply a stimulus, observe the result, compare to prediction. If the result matches prediction, the law holds. If not, the law must be revised.

**The tests package is this laboratory.**

- **Test functions** are the experiments
- **Stub providers** are the controlled stimuli (we script exactly what "players" do)
- **Game state assertions** are the observations
- **Expected results** are the predictions from our architectural theory

When tests pass, we have empirical evidence that the system behaves as specified. When they fail, we have discovered a bug or a misunderstanding.

**Why this matters**: In science, a theory without experimental verification is speculation. In software, code without tests is unverified speculation. Tests transform code into reliable knowledge.

## The Architecture of Testing: Pure and Deterministic

### The StubProvider: Fake Reality

Testing the engine is only possible because of the `InputProvider` interface. In production, the `sessionProvider` bridges to real WebSockets. In tests, the `stubProvider` (or `scriptedInput`) returns predetermined values.

```go
type scriptedInput struct {
    actions    []*models.PlayerAction
    challenges [][]engine.ChallengeDecision
    counters   [][]engine.CounterDecision
    steps      []any
}
```

**How it works**:
1. Test sets up the stub with a script: "Player 0 will play Steal targeting Player 1"
2. Test calls `game.RunTurn(ctx, stub, clock, config)`
3. Engine calls `stub.RequestAction()`
4. Stub returns the scripted action immediately (no waiting)
5. Engine processes, may call `stub.ChallengeResponses()`
6. Stub returns scripted challenges
7. Process continues deterministically

**The key insight**: We are not testing the network layer. We are testing the **pure game logic**. The stub lets us write deterministic scenarios:

```go
// Scenario: Player 0 steals, Player 1 challenges, challenge fails
provider := &scriptedInput{
    actions:    []*models.PlayerAction{{ID: models.Steal, SourceIndex: 0, TargetIndex: 1}},
    challenges: [][]engine.ChallengeDecision{{{ChallengerIndex: 1}}},
}
result, _ := game.RunTurn(ctx, provider, clock, config)
// Assert: Player 1 loses a card (challenge was wrong), steal succeeds
```

**Why this is powerful**: 
- **Deterministic**: Same test, same result, every time
- **Fast**: No network, no waiting, milliseconds per test
- **Readable**: The scenario is explicit in code
- **Exhaustive**: Can test rare edge cases (what if everyone challenges simultaneously?)

### The ManualClock: Controlled Time

Time is the enemy of determinism. `time.Now()` returns different values on each call. But the engine needs timeouts for challenges and counters.

The `manualClock` (or similar test implementations) solves this:

```go
type manualClock struct {
    timers []chan time.Time
}

func (c *manualClock) After(d time.Duration) <-chan time.Time {
    ch := make(chan time.Time, 1)
    c.timers = append(c.timers, ch)
    return ch
}

func (c *manualClock) FireNext() {
    if len(c.timers) > 0 {
        c.timers[0] <- time.Now()
        c.timers = c.timers[1:]
    }
}
```

**How it works**:
1. Engine calls `clock.After(timeout)`, expects a channel that fires after the duration
2. ManualClock captures the channel but does not fire it yet
3. Test can choose when time "passes" by calling `FireNext()`
4. Or test can choose to never fire, simulating "timeout expired"

**Why this matters**: We can test timeout scenarios precisely:
```go
// Scenario: Challenge window opens, no one challenges, timeout expires
provider := &scriptedInput{actions: [...], challenges: [[]]}  // empty challenges
clock := newManualClock()

// Run turn in background
resultChan := make(chan TurnResult)
go func() {
    result, _ := game.RunTurn(ctx, provider, clock, config)
    resultChan <- result
}()

// Wait for timer to be created
clock.WaitNextTimer()

// Fire the timer (simulating timeout)
clock.FireNext()

// Now the turn completes
result := <-resultChan
// Assert: Action executed (no challenges), no one lost cards
```

**Controlled time** makes async testing synchronous and deterministic.

## Test Categories: What We Prove

### 1. Orchestrator Tests: The Turn Flow

**File**: `orchestrator_test.go`

These tests verify the core turn loop: action solicitation, validation, challenge window, counter window, application.

**What they prove**:
- Actions proceed through phases in correct order
- Challenges happen at the right time
- Counters block actions correctly
- Timeouts work (via manual clock)
- State changes correctly after each phase

**Example pattern**:
```go
func TestTurn_Flow(t *testing.T) {
    // Setup: 2 players, Player 0 has Thief, Player 1 has 2 coins
    gs := state.NewGameState(2, deck)
    gs.Players[0].Hand = []state.Card{{Role: models.Thief}}
    gs.Players[1].Coins = 2
    
    // Script: Player 0 steals, no challenges, no counters
    provider := &scriptedInput{
        actions: []*models.PlayerAction{{
            ID: models.Steal, SourceIndex: 0, TargetIndex: 1,
        }},
        challenges: [[]],
        counters: [[]],
    }
    
    // Execute
    result, err := engine.RunTurn(ctx, provider, clock, config)
    
    // Verify
    assert.NoError(t, err)
    assert.Equal(t, 2, gs.Players[0].Coins)  // gained 2
    assert.Equal(t, 0, gs.Players[1].Coins)  // lost 2
}
```

### 2. Actions Tests: Individual Behaviors

**File**: `actions_test.go`

These tests verify specific action handlers and validators in isolation.

**What they prove**:
- Steal transfers exactly 2 coins
- Coup costs exactly 7 coins
- Exchange swaps correct number of cards
- Validation rejects invalid targets
- Validation rejects insufficient coins

**Why isolate**: When a test fails, you know exactly which action is broken. No need to trace through the full turn flow.

### 3. Flow Tests: Complex Sequences

**File**: `flow_test.go`

These tests verify multi-turn scenarios and complex interactions.

**What they prove**:
- Games can complete from start to finish
- Win conditions are detected correctly
- Player elimination works
- Multiple turns in sequence maintain state correctly

**Why they matter**: Individual unit tests do not catch integration issues. Flow tests verify the system works as a whole.

### 4. Session Tests: The Reality Bridge

**File**: `session_test.go`

These tests verify the server's session management: connections, disconnections, reconnections, grace periods.

**What they prove**:
- Players can connect and join games
- Disconnect starts grace period
- Reconnect within grace period resumes game
- Forfeit after grace period
- State synchronization on reconnect

**Pattern**: These use the real `sessionProvider` (or a test variant), not just stubs. They verify the translation between sync and async worlds.

### 5. Lobby Tests: Social Infrastructure

**File**: `lobby_test.go`

These tests verify lobby creation, joining, leaving, and game start.

**What they prove**:
- Lobbies can be created
- Players can join open lobbies
- Leaders can start games
- Players receive lobby updates

### 6. Store Tests: Persistence

**File**: `store_test.go`

These tests verify storage and retrieval of game state.

**What they prove**:
- GameState serializes correctly
- GameState deserializes correctly
- Snapshots can be saved and loaded
- History maintains integrity

### 7. WebSocket Configuration Tests

**File**: `ws_config_test.go`

These verify WebSocket behavior, message formatting, connection handling.

### 8. Forfeit Tests: Graceful Degradation

**File**: `forfeit_test.go`

These verify player abandonment handling.

## Common Pitfalls: When Tests Lie

### Pitfall 1: "Testing the Stub"

**The mistake**: Writing tests that only verify the stub works, not the real code.

```go
// WRONG: this tests that addition works, not the game
func TestSteal(t *testing.T) {
    game := &MockGame{}
    game.On("TransferCoins", 0, 1, 2).Return(nil)
    
    err := Steal(game, 0, 1)
    
    assert.NoError(t, err)
    game.AssertExpectations(t)  // Yay, mock was called!
}
```

**Why it destroys everything**:
- Tests pass but real game is broken
- Mocks hide integration issues
- False confidence

**The right way**: Test with real GameState, real actions, real engine. Use stubs only for InputProvider (the boundary), not for internal state.

### Pitfall 2: "Order-Dependent Tests"

**The mistake**: Tests that pass when run alone but fail when run with others.

```go
var globalCounter = 0

func TestA(t *testing.T) {
    globalCounter++
    assert.Equal(t, 1, globalCounter)
}

func TestB(t *testing.T) {
    globalCounter++
    assert.Equal(t, 1, globalCounter)  // Fails if TestA ran first!
}
```

**Why it destroys everything**:
- Flaky CI builds
- Developers waste time debugging
- Cannot parallelize tests

**The right way**: Tests must be independent. No shared mutable state. Each test creates its own GameState, its own stubs, its own everything.

### Pitfall 3: "Testing Implementation, Not Behavior"

**The mistake**: Tests that break when implementation changes, even though behavior is correct.

```go
// WRONG: tests internal structure
func TestGame(t *testing.T) {
    g := NewGame()
    g.runPhase1()  // internal method
    g.runPhase2()  // internal method
    
    assert.Equal(t, 2, len(g.internalSlice))
}
```

**Why it hurts**:
- Refactoring becomes impossible
- Tests are brittle
- Tests do not document intended behavior

**The right way**: Test public behavior. Call `RunTurn()`, not internal phases. Assert on final state, not internal structure.

### Pitfall 4: "Silent Failures"

**The mistake**: Tests that do not actually check anything meaningful.

```go
// WRONG: what is this proving?
func TestSomething(t *testing.T) {
    game := NewGame()
    result, _ := game.DoSomething()
    assert.NotNil(t, result)  // Always passes if function returns something!
}
```

**Why it destroys everything**:
- Tests give false confidence
- Bugs slip through
- Code coverage metrics lie

**The right way**: Assert specific, meaningful properties:
```go
assert.Equal(t, expectedCoins, player.Coins)
assert.True(t, player.HasLost)  // specific condition
assert.ErrorIs(t, err, ErrInsufficientCoins)
```

### Pitfall 5: "Missing Edge Cases"

**The mistake**: Only testing the happy path.

```go
// WRONG: only tests success
func TestSteal(t *testing.T) {
    game := NewGameWithCoins(2)
    err := game.Steal(0, 1)
    assert.NoError(t, err)
}
```

**Why it hurts**:
- Edge case bugs survive
- Production crashes on unusual inputs

**The right way**: Test failure modes too:
```go
func TestSteal_InsufficientCoins(t *testing.T) {
    game := NewGameWithCoins(0)  // target has 0 coins
    err := game.Steal(0, 1)
    assert.Error(t, err)
}

func TestSteal_InvalidTarget(t *testing.T) {
    game := NewGameWith2Players()
    err := game.Steal(0, 5)  // player 5 does not exist
    assert.Error(t, err)
}

func TestSteal_SelfTarget(t *testing.T) {
    game := NewGame()
    err := game.Steal(0, 0)  // cannot steal from self
    assert.Error(t, err)
}
```

## Design Trade-offs: How We Test

### Trade-off: Unit vs. Integration Tests

**Our balance**:
- **Unit tests** for actions (validators, handlers)
- **Integration tests** for turn flow (orchestrator)
- **System tests** for session/lobby (full stack)

**Why**: Unit tests give precise failure location. Integration tests catch interaction bugs. System tests verify real behavior.

**Ratio**: ~70% unit, 20% integration, 10% system. Unit tests are fast and precise. System tests are slow but necessary for reality-handling code.

### Trade-off: Table-Driven vs. Individual Tests

**Why we use table-driven tests**:
- Compact (many cases in one function)
- Easy to add new cases
- Clear pattern: setup → execute → assert

**Example**:
```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name      string
        action    ActionID
        state     GameState
        wantError bool
    }{
        {"valid steal", Steal, stateWithCoins(2), false},
        {"steal from broke", Steal, stateWithCoins(0), true},
        {"valid coup", Coup, stateWithCoins(7), false},
        {"coup too poor", Coup, stateWithCoins(6), true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.action, tt.state)
            assert.Equal(t, tt.wantError, err != nil)
        })
    }
}
```

**What we give up**: Individual test functions have more descriptive names and can have different setup. We accept this trade-off for the compactness of tables.

### Trade-off: Stubbing vs. Real Dependencies

**When we stub**:
- InputProvider (boundary to async world)
- Clock (boundary to time)
- RNG (when testing specific random scenarios)

**When we use real**:
- GameState (state is the subject under test)
- Action handlers (the logic we are verifying)
- Engine orchestration (the flow we are testing)

**Why**: Stub boundaries, test internals. The purity of the engine makes this possible.

## Performance Considerations: Fast Feedback

### Test Speed: The Virtue of Purity

Engine tests run in **milliseconds** because:
- No network I/O
- No database queries
- No real-time delays (manual clock)
- Deterministic execution (no waiting for goroutines)

**Comparison**: A typical integration test in a web app might take 100ms-1s. Our engine tests take 0.1ms-1ms.

**Why this matters**: Developers run tests frequently. Fast tests get run. Slow tests get skipped.

### Parallel Testing: Safe by Design

Tests can run in parallel (`t.Parallel()`) because:
- Each test creates its own GameState
- No shared mutable state
- No global variables
- Stubs are per-test

**Current state**: We do not currently use `t.Parallel()` because tests are already fast. But we could.

### Coverage: Meaningful Metrics

**Target**: High coverage of engine and actions. Lower coverage of server (integration tests).

**What counts**:
- All validators have success and failure cases
- All handlers have effect verification
- All turn phases have flow tests
- All error paths are exercised

**What does not count**: Coverage of `main()` or wiring code. Focus on logic.

## Evolution: Maintaining the Test Suite

### Adding Tests for New Actions

When adding a new action:
1. Unit test the validator (valid cases, invalid cases)
2. Unit test the handler (correct effects, correct logging)
3. Integration test in turn flow (action proceeds through phases)
4. Edge case tests (what if challenged? what if countered?)

**Pattern**: Copy tests from similar existing actions, modify for new behavior.

### Updating Tests for Refactoring

When refactoring:
1. Tests should guide the refactoring (they define behavior)
2. If tests break, either:
   - The refactoring changed behavior (bug—revert)
   - The tests were testing implementation (rewrite tests)
3. Use tests to verify refactoring preserved behavior

**Golden rule**: Tests are the specification. Behavior that tests verify is the contract.

### Regression Tests: Capturing Bugs

When a bug is found in production:
1. Write a test that reproduces the bug
2. Verify the test fails
3. Fix the code
4. Verify the test passes
5. Keep the test forever (regression prevention)

**Why**: Bugs tend to recur. A regression test ensures the specific failure mode never happens again.

## Anti-Patterns: Code That Signals Trouble

### Anti-Pattern: "Tests Without Assertions"

```go
// WRONG: does not verify anything
func TestGameRuns(t *testing.T) {
    game := NewGame()
    game.Run()  // Does it work? Who knows!
}
```

**What this means**: Test exists for coverage but proves nothing.

**Fix**: Assert specific outcomes:
```go
result, err := game.Run()
assert.NoError(t, err)
assert.True(t, result.GameOver)
assert.Equal(t, 0, result.Winner)
```

### Anti-Pattern: "Commented-Out Tests"

```go
// TODO: fix this test
// func TestBroken(t *testing.T) { ... }
```

**What this means**: Broken tests are ignored. Technical debt accumulates.

**Fix**: Either fix the test immediately, or delete it. Commented code rots.

### Anti-Pattern: "Test-Only Production Code"

```go
// In production code, only used by tests
func (g *Game) SetInternalState(state GameState) {
    g.state = state
}
```

**What this means**: Production code is being bent to accommodate tests.

**Fix**: Tests should use public APIs. If you cannot test without internal access, the API may need redesign, or the test approach is wrong.

## Team Culture: Testing as Craft

### Code Review Questions

When reviewing tests:
- [ ] Does this test meaningful behavior?
- [ ] Are there both success and failure cases?
- [ ] Are edge cases covered?
- [ ] Are tests independent (no shared state)?
- [ ] Do assertions check specific values, not just "not nil"?
- [ ] Is the test readable (clear setup, clear assertions)?

### The Red-Green-Refactor Cycle

Encourage TDD (Test-Driven Development):
1. **Red**: Write a failing test for the desired behavior
2. **Green**: Write minimal code to make test pass
3. **Refactor**: Clean up code while tests stay green

**Why**: Tests guide design. You write testable code. You have coverage from the start.

### Onboarding: Learning to Test

New developers should:
1. Read existing tests to understand patterns
2. Write tests for a small bug fix first
3. Experience the confidence of a green test suite
4. Understand that tests are production code (maintain them!)

## Summary: The Foundation of Trust

The tests package is the **foundation of trust**. It proves the engine is deterministic. It proves state mutations are correct. It proves actions follow rules. Without tests, we have only hope. With tests, we have confidence.

Key principles:
1. **Tests are experiments**: They verify hypotheses about the code
2. **Deterministic**: Same test, same result, always
3. **Fast**: No I/O, no waiting, pure logic
4. **Comprehensive**: Happy paths, error paths, edge cases
5. **Independent**: No shared state between tests
6. **Behavior-focused**: Test what, not how

When the test suite is green, the system is trustworthy. When a test fails, we have learned something. Tests transform code from fragile hope into reliable engineering.

The proving ground separates working code from broken theory.
