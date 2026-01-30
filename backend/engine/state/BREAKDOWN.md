# State Package Breakdown: The Vault and the Archive

## The Core Philosophy: State is Sacred, Guard it Fiercely

Game state is the only thing that matters. Everything else—the WebSocket connections, the session logic, the action handlers—exists only to transform state from one valid configuration to another. State is the **ground truth** of the game. When the server crashes and restarts, when players reconnect after hours, the state is what persists. Everything else is ephemeral.

The state package treats state with the reverence it deserves. It provides:
1. **Immutable snapshots**: Complete, self-contained records of moments in time
2. **Guarded mutation**: State changes only through validated methods
3. **Clear ownership**: No ambiguity about who can modify what

This package is the foundation. If state is corrupt, the game is meaningless. If state is unclear, debugging is impossible. If state is unprotected, bugs proliferate.

### The Mental Model: The Vault of Time

Imagine a vast underground vault containing a perfect record of every moment in history. Each shelf holds complete snapshots—photographs so detailed they capture everything: every coin, every card, every player's position. These snapshots are **immutable**. Once placed on the shelf, they never change. They are historical fact.

But there is also the **living present**—the current moment that has not yet been archived. This is where the game happens. This is where mutations occur. But even here, changes are not chaotic. They pass through **guardians**—methods that verify, validate, and enforce the laws of physics before allowing any change.

**The GameState is the living present.** The History is the archive of snapshots. The methods (AddCoins, DiscardFromHand, etc.) are the guardians.

**Why this matters**: When you debug, you can walk into the vault and inspect any moment. When you test, you can create hypothetical presents and see what happens. When you play, the guardians ensure the world remains consistent.

## The Architecture: Snapshots and Guards

### Snapshots: Photographs, Not Deltas

A naive approach to state history would record **deltas**: "Player 0 gained 2 coins." "Player 1 lost a card." This is compact but treacherous. To know the state at turn 10, you must replay all deltas from turn 0.

We choose **snapshots** instead: complete copies of the entire state at each significant moment.

**Why snapshots win**:
1. **Immediate understanding**: Look at any snapshot and know exactly what the game looked like
2. **Debugging simplicity**: No replay required to inspect a moment
3. **Replay simplicity**: To replay, just show the snapshots in sequence
4. **Testability**: Create a snapshot, feed it to the engine, observe the result

**The cost**: Memory. A snapshot copies everything—players, hands, deck, turn number.

**The reality**: Game states are small. A typical 6-player game with 2 cards each, plus deck, is under 10KB. Even with 100 turns, that's 1MB. Memory is cheap. Developer time is expensive.

### The Snapshot Method: Deep Copy Discipline

```go
func (g *GameState) Snapshot() *GameState {
    newState := *g  // Shallow copy of struct
    
    // Deep copy slices (they are reference types)
    newState.Players = make([]Player, len(g.Players))
    for i := range g.Players {
        newState.Players[i] = g.Players[i]
        newState.Players[i].Hand = append([]Card{}, g.Players[i].Hand...)
    }
    
    newState.Deck = make(Deck, len(g.Deck))
    copy(newState.Deck, g.Deck)
    
    // Transient fields are NOT copied
    newState.DiscardEvents = nil
    newState.PendingRequest = nil
    
    return &newState
}
```

**Critical details**:
- Slices must be explicitly copied (`append([]Card{}, ...)` creates a new slice with copied elements)
- Transient fields (DiscardEvents, PendingRequest) are cleared—they belong to the current turn, not history
- Pointers to external objects must be deep-copied (none currently, but be vigilant)

**Why deep copy matters**: Shallow copies share underlying arrays. Modifying the "copy" would modify the original. This destroys the immutability guarantee.

### Guarded Mutation: The Method Contract

You cannot simply write:
```go
game.Players[0].Coins = 100  // FORBIDDEN
```

You must use:
```go
game.AddCoins(0, 100)  // CORRECT
```

**The guardian pattern**:
1. **Validate inputs**: Check indices are in bounds, deltas are valid
2. **Check invariants**: Ensure no negative coins, respect MaxCoins limit
3. **Apply change**: Modify state
4. **Return result**: New value or error

**Why this matters**:
- **Consistency**: All state changes enforce the same rules
- **Discoverability**: Methods document what operations are possible
- **Testability**: Mock or intercept state changes easily
- **Debugging**: Stack traces show which guardian was called

**The invariant enforcement**:
- `MaxCoins = 12`: AddCoins caps at this limit
- No negative coins: AddCoins returns error if result would be negative
- Valid indices: All methods check playerIndex and cardIndex bounds

## The Structure: What State Contains

### GameState: The Complete Picture

```go
type GameState struct {
    TurnNumber         int           // Current turn (0, 1, 2, ...)
    Players            []Player      // All players and their state
    Deck               Deck          // Draw pile
    CurrentPlayerIndex int           // Whose turn it is
    
    // Transient fields (not part of history)
    DiscardEvents  []DiscardEvent    // Cards discarded this turn
    PendingRequest *StepRequest      // Multi-step action waiting for input
    seed           *rand.Rand        // RNG (transient, not serialized)
}
```

**Why these fields**:
- **TurnNumber**: Human-readable turn counter
- **Players**: The actors in the game (their cards, coins, identity)
- **Deck**: The source of randomness and uncertainty
- **CurrentPlayerIndex**: The game's pointer—who acts now

**Why transient fields exist**:
- **DiscardEvents**: Track what happened this turn for logging. Cleared each turn.
- **PendingRequest**: Track multi-step actions (Exchange, Investigate). Not part of persistent state.
- **seed**: The RNG source. Deterministic per game, but not part of the state snapshot.

### Player: The Actor State

```go
type Player struct {
    ID    int       // Index in Players slice (0, 1, 2, ...)
    Name  string    // Display name ("Alice", "Bob")
    Hand  []Card    // Role cards held (typically 2, reduces as game progresses)
    Coins int       // Current wealth (0 to MaxCoins)
}
```

**The ID is the index**: This is a critical design choice. Players are referred to by their position in the Players slice, not a UUID or separate ID field. This makes arrays and lookups simple.

**Trade-off**: Player order is fixed for the game. You cannot reorder players without updating all references. But games do not need to reorder players mid-game.

### Card: The Role Identity

```go
type Card struct {
    ID   int         // Unique card identifier (for tracking individual cards)
    Role models.Role // The role on this card (Thief, Colonel, etc.)
}
```

**Why card ID**: Two cards can have the same role. The ID distinguishes them (useful for tracking specific cards in logs, though not currently exposed to players).

### Deck: The Random Source

The Deck is a slice of Cards. It provides:
- `Draw(n)`: Remove and return n cards from the top
- `Shuffle()`: Randomize order (uses the GameState's seeded RNG)
- `Append()`: Return cards to bottom

**The deck is the only source of randomness** in the game. All randomness flows through it.

## Mutation Methods: The Guardians in Detail

### AddCoins: Wealth Management

```go
func (g *GameState) AddCoins(playerIndex int, delta int) (int, error)
```

**Behavior**:
- Validates playerIndex is in bounds
- Calculates newCoins = current + delta
- If newCoins < 0: returns error (insufficient funds)
- If newCoins > MaxCoins (12): caps at MaxCoins
- Updates player.Coins
- Returns new coin count

**Why cap instead of error**: The game rules say you cannot hold more than 12 coins. If an action would give you more, the excess is lost. This is a game rule, not an error condition.

### DiscardFromHand: Card Removal

```go
func (g *GameState) DiscardFromHand(playerIndex, cardIndex int) (Card, error)
```

**Behavior**:
- Validates playerIndex and cardIndex
- Removes card from player's hand
- Returns card to bottom of deck
- Records a DiscardEvent (for logging)
- Returns the discarded card

**Why return to deck**: Cards are never destroyed, only recycled. This maintains card counts and enables Exchange to work correctly.

**DiscardEvents**: These are transient—they accumulate during a turn, then are drained and logged. They provide a record of what happened without modifying persistent state.

### DrawCards: Replenishment

```go
func (g *GameState) DrawCards(playerIndex, count int) error
```

**Behavior**:
- Validates playerIndex
- Draws count cards from deck
- Appends to player's hand

**Why separate method**: Drawing is distinct from other operations. It has different validation (does not check MaxHandSize because the game has no max hand size—you keep what you draw).

### ReturnCardsToDeck: The Exchange Pattern

```go
func (g *GameState) ReturnCardsToDeck(playerIndex int, cardIndices []int) error
```

**Behavior**:
- Validates all indices before modifying anything
- Separates hand into "keep" and "return" piles
- Updates hand to "keep"
- Appends "return" to deck

**Critical**: All-or-nothing validation. If any index is invalid, nothing happens. This prevents partial state corruption.

**Why used**: Exchange action draws 2, keeps some, returns some. This method handles the returning atomically.

### SwapCard: The Politician's Power

```go
func (g *GameState) SwapCard(playerIndex, cardIndex int) error
```

**Behavior**:
- Removes card from hand, returns to deck
- Draws one new card
- Adds to hand

**Used by**: Politician's Exchange action to swap specific cards with the deck.

## Common Pitfalls: When State Goes Wrong

### Pitfall 1: "Direct Field Mutation"

**The mistake**:
```go
game.Players[0].Coins += 5  // Bypasses MaxCoins check!
game.Players[1].Hand = append(game.Players[1].Hand, newCard)  // No validation!
```

**Why it destroys everything**:
- Violates MaxCoins invariant
- Can create invalid card counts
- No error handling
- Bypasses logging/auditing

**The right way**: Always use guardian methods. If the method does not exist, add it.

### Pitfall 2: "Shallow Copy Bugs"

**The mistake**:
```go
snapshot := *gameState  // Shallow copy
// Later: modify snapshot.Players[0].Hand
// Original is also modified!
```

**Why it destroys everything**:
- Snapshots are not immutable
- History becomes corrupted
- Replays show wrong state
- Tests become flaky

**The right way**: Use the official `Snapshot()` method. It knows which fields need deep copying.

### Pitfall 3: "Forgetting Transient Fields"

**The mistake**:
```go
// In Snapshot()
newState.DiscardEvents = g.DiscardEvents  // Shares slice!
```

**Why it destroys everything**:
- History snapshots retain references to mutable state
- Subsequent mutations affect archived snapshots
- Replays become nondeterministic

**The right way**: Set transient fields to nil in snapshots. They do not belong in history.

### Pitfall 4: "State Leaking Between Games"

**The mistake**:
```go
game1 := NewGame(...)
game2 := game1  // Reference copy, not value copy!
```

**Why it destroys everything**:
- Two games share the same state
- Actions in one affect the other
- Impossible to debug

**The right way**: Always create fresh GameState for each game. Use Snapshot() to copy state intentionally, never accidental reference sharing.

## Design Trade-offs: Why We Chose This Way

### Trade-off: Methods vs. Direct Access

**Why we chose methods (guardians)**:
- Invariant enforcement at compile time (must call method)
- Centralized validation logic
- Clear API surface (what operations are valid)
- Easy to add logging, metrics, hooks

**What we gave up**:
- Verbosity (more code than direct access)
- Cannot use struct initialization syntax for complex state
- Slight performance overhead of function calls

**Why this is correct**: State corruption is the worst bug. The verbosity is worth the safety.

### Trade-off: Slice Indices vs. Object References

**Why we chose slice indices**:
- Simple, efficient (integers not pointers)
- Works well with Go's slice operations
- Easy to serialize
- No GC pressure from object references

**What we gave up**:
- Type safety (any int could be a player index)
- Self-documenting code (p.ID vs just an int)
- Must validate bounds everywhere

**Mitigation**: Use the `PlayerIndex` type alias (if added) or validate immediately in methods.

### Trade-off: Mutable Current State vs. Immutable State

**Why we chose mutable current state + immutable snapshots**:
- Performance (no allocation on every coin change)
- Simplicity (normal Go patterns)
- Snapshots provide immutability when needed

**What we gave up**:
- Cannot share current state safely (must snapshot first)
- Must remember to snapshot before passing to other goroutines
- Potential for accidental mutation

**Why this is correct**: The current state is private to the engine goroutine. It is never shared. Snapshots are for crossing boundaries (persistence, broadcasting, testing).

## Performance Considerations: State in Motion

### Snapshot: The Hot Path

`Snapshot()` is called after every turn. It must be efficient:

**Current implementation**:
- One allocation for new GameState struct
- One allocation for Players slice
- One allocation per player's Hand
- One allocation for Deck

**Total**: ~10 small allocations per turn. Negligible.

**Optimization opportunities** (only if profiling shows need):
- Object pool for GameState structs
- Pre-allocated hand slices
- Copy-on-write for unchanged data

**Do not optimize prematurely**: Current code is clear and correct. Optimize only when measured.

### State Size: The Scaling Limit

**Current size estimate** (6 players, 15 cards in deck):
- GameState struct: ~64 bytes
- Players slice (6 * Player): ~6 * 48 bytes = 288 bytes
- Hands (6 * 2 cards * 16 bytes): ~192 bytes
- Deck (15 * 16 bytes): ~240 bytes
- **Total**: ~1KB per snapshot

**100 turns**: 100KB of history. Trivial.

**Scaling to 1000 players**: Not our use case. This is a social game for 2-6 players.

### Memory Layout: Cache Friendliness

Current layout is reasonably cache-friendly:
- Players stored contiguously in slice
- Each player's hand stored contiguously
- Sequential access patterns in most operations

**Not a concern for our scale.** Do not micro-optimize memory layout.

## Evolution: Modifying State

### Adding a New State Field

1. Add field to GameState struct
2. Update Snapshot() to copy the field (if it should be in history)
3. Update any methods that should use it
4. Add tests
5. Consider: should this be transient or persistent?

**Backward compatibility**: Old snapshots without the field will have zero values, which is usually correct.

### Adding a New Mutation Method

1. Define the method on *GameState
2. Validate all inputs
3. Enforce invariants
4. Return error for invalid operations
5. Add unit tests

**Pattern to follow**: Look at AddCoins or DiscardFromHand as templates.

### Changing Invariants

If game rules change (e.g., MaxCoins becomes 15):

1. Update the constant
2. Update guardian methods
3. Update tests
4. Consider: do old snapshots need migration?

**Usually safe**: Old snapshots with 12 coins are still valid with 15 max. The new limit only affects new operations.

## Anti-Patterns: Code That Signals Trouble

### Anti-Pattern: "Public Mutable Fields"

```go
// WRONG: anyone can modify
type GameState struct {
    Players []Player  // exported, mutable
}
// Elsewhere:
game.Players[0].Coins = 999  // no validation!
```

**What this means**: State is unprotected. Invalid states are possible.

**Fix**: Unexport fields (players instead of Players) or provide only getter methods. Force all mutation through guardians.

### Anti-Pattern: "Validation in the Wrong Place"

```go
// WRONG: validator checking state invariants
func ValidateSteal(gs *GameState, cmd ActionCommand) error {
    if gs.Players[target].Coins < 0 {
        return error  // This should be impossible!
    }
}
```

**What this means**: Invariants are not being enforced by state methods. State corruption has occurred.

**Fix**: Guardians should enforce invariants. Validators check action prerequisites. If an invariant is violated, it is a bug in the guardian, not the validator's concern.

### Anti-Pattern: "Leaking Internal References"

```go
// WRONG: returning mutable internal state
func (g *GameState) GetPlayerHand(playerIndex int) []Card {
    return g.Players[playerIndex].Hand  // returns reference to internal slice!
}
// Caller can modify:
hand := game.GetPlayerHand(0)
hand[0] = Card{Role: "Thief"}  // Corrupts game state!
```

**What this means**: Internal mutable state escapes. Caller can corrupt game.

**Fix**: Return copies:
```go
func (g *GameState) GetPlayerHand(playerIndex int) []Card {
    hand := g.Players[playerIndex].Hand
    return append([]Card(nil), hand...)  // copy
}
```

## Team Culture: Respecting the State

### Code Review Questions

When reviewing state-related changes:
- [ ] Are all mutations through guardian methods?
- [ ] Does Snapshot() properly deep-copy new fields?
- [ ] Are invariants enforced (bounds checks, no negatives)?
- [ ] Are transient fields properly excluded from snapshots?
- [ ] Are errors returned, not panics?

### Documentation in Code

GameState and its methods should be extensively documented:
- What invariants does this method enforce?
- What are the valid inputs?
- What error conditions can occur?
- Is this field transient or persistent?

### Onboarding: Understanding the Vault

New developers should understand:
1. State is the ground truth—everything else serves it
2. Never mutate directly—always use methods
3. Snapshots are immutable records—respect deep copy discipline
4. Invariants are sacred—never allow invalid states

## Summary: The Sacred Ground

The state package is the **sacred ground** of the system. It holds the truth of the game. It guards that truth fiercely. It records history faithfully.

Key principles:
1. **State is truth**: Everything exists to serve the state
2. **Guardians enforce invariants**: All mutation goes through methods
3. **Snapshots are immutable**: History cannot be altered
4. **Deep copy discipline**: Never let references leak between snapshots
5. **Validation at boundaries**: Check inputs, enforce invariants

When state is clean, the game is trustworthy. When state is corrupt, nothing matters. Preserve the sacred ground. Guard it well.

The vault of time holds the only record that matters.
