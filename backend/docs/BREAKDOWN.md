# System Breakdown: Architecture, Mental Models & Design Philosophy

## The Core Philosophy: Purity vs. Reality

When building any interactive system, you face a fundamental tension that most developers ignore until it destroys them. On one side is **purity**: the desire for code that is deterministic, testable, and behaves identically every time. On the other side is **reality**: networks fail, users behave unpredictably, timing is never guaranteed, and the world is chaos.

Most systems fail because they let reality infect their purity. They sprinkle network error handling throughout business logic. They mix database timeouts with game rules. They create code where you cannot tell if a bug is in the game logic or the error recovery.

ElMakina rejects this approach entirely. We maintain a **hard boundary** between purity and reality. The engine is pure. The server handles reality. They communicate through a narrow, well-defined interface. This is not just architecture—it is a **philosophy of separation**.

### The Mental Model: The Game Master in a Sealed Room

Imagine a grandmaster chess player sitting in a perfectly sealed, white room. The grandmaster has total knowledge of the chess board—every piece, every position, every rule. But the room has no windows, no doors, no cameras. The only opening is a mail slot in the wall.

To make a move, you write your move on a slip of paper and slide it through the slot. Eventually, a response slides back out showing the new board state. The grandmaster does not know who you are, how old you are, or how the message traveled. They do not know if you are a grandmaster yourself or a child learning the game. They only know the move and the rules.

**This is exactly the mental model of the ElMakina engine.**

The engine is the grandmaster in the white room. The InputProvider interface is the mail slot. The server is everything outside—the people writing messages, the paper, the delivery, the chaos of the outside world.

When you read engine code, you must adopt this mental model. There is no "network" in the engine. There is no "disconnected player." Those concepts literally do not exist. When the engine calls `input.RequestAction()`, it is sliding a message through the mail slot saying "What does the current player want to do?" The engine then waits, patient and timeless, for a response to slide back through.

**Why this matters**: When debugging, you must know which universe a bug lives in. If coins are not transferring correctly, look in the engine or actions package—the game master's logic is wrong. If players are not receiving turn notifications, look in the server—the mail slot mechanism is broken. You never have to wonder.

## The Three Laws of Engine Purity

The engine follows three strict laws that make it trustworthy:

### Law 1: Determinism (Same Input, Same Output)

Given the exact same sequence of actions, the engine will always produce the exact same sequence of game states. Always. On every machine. At any time.

This is not accidental—it is **intentional and enforced**. The engine has no `time.Now()` calls (except through the Clock interface which can be mocked). It has no random number generation (except through the RNG which can be seeded). It has no external dependencies that could vary.

**Mental model**: Think of the engine like a mathematical function. `f(actions) = game_states`. Pure. Stateless. Timeless.

**Why this matters**: You can replay any game exactly. You can write tests that verify exact game states. You can debug by replaying the exact sequence of events. You can run the same test a thousand times and get the same result.

### Law 2: Total Isolation (No Side Effects)

The engine never writes to a database. It never sends a network request. It never reads a file. It never touches anything outside its own memory.

The only output is the return value. The only input is the parameters. This is **total isolation**.

**Mental model**: The white room is soundproof and sealed. Nothing gets in or out except through the mail slot.

**Why this matters**: You can test the engine in a vacuum. No test databases. No mock servers. No network setup. Just create a Game, feed it actions, check the state. Tests run in milliseconds. They are deterministic. They are reliable.

### Law 3: Synchronous Simplicity (Block and Wait)

The engine is entirely synchronous. When it needs input, it calls a function and waits for the return value. It does not use callbacks. It does not use event handlers. It does not use async/await. It simply blocks.

**Mental model**: The game master reads your message, thinks, and writes a response. There is no "event loop" in the white room. Just direct, linear, human-scale thinking.

**Why this matters**: Synchronous code is linear and readable. You can follow the logic step by step. There are no "gotchas" about what happens in what order. The control flow is obvious.

## The Server's Compassion: Graceful Degradation in a Chaotic World

While the engine is pure and ruthless, the server lives in reality and must be **compassionate**. Real players have spotty WiFi, phones that go to sleep, browsers that crash, and lives that interrupt games. The server handles all of this with grace.

### The Mental Model: The Translator Between Worlds

The server is a **real-time translator** between two incompatible worlds:

- **The Engine World**: Synchronous, blocking, pure, deterministic
- **The Network World**: Asynchronous, event-driven, chaotic, unreliable

When the engine blocks waiting for `RequestAction()`, the server translates that into: "Send a WebSocket message to the player and set a timeout." When the WebSocket message arrives, the server translates that back into: "Return from RequestAction() with this value."

This translation is the **session runner's** primary job. It is the most complex code in the server because it bridges the unbridgeable gap between sync and async.

### The Grace Period: Compassion for Disconnection

When a player disconnects, we do not punish them immediately. We start a **grace period timer**. If they reconnect within the grace period, they resume exactly where they left off—same game, same cards, same turn (if it was their turn). If they do not reconnect, they forfeit.

**Mental model**: We assume good intent. Maybe their WiFi dropped. Maybe they got a phone call. We give them a chance to return. Only after the grace period do we conclude they have abandoned the game.

**Why this matters**: This makes the game playable on real-world networks. Without this compassion, every WiFi blip would end the game. With it, brief interruptions are survivable.

## The State: Snapshots of Time

Game state is constantly changing. How do we think about this without going mad?

### The Mental Model: Photographs, Not Video

Imagine you are watching a game and you take a photograph every time something important happens. Each photograph captures the complete game state at that exact moment—every card, every coin, whose turn it is. You end up with an album of photographs.

This is how ElMakina thinks about game state. Each **snapshot** is a complete, self-contained record of the game at one instant. The `History` is the album of all snapshots.

**Why snapshots instead of deltas?** A delta ("player 0 gained 2 coins") requires you to replay all previous changes to understand the current state. A snapshot is immediately understandable. You can look at any snapshot in isolation and know exactly what the game looked like.

### Immutability and Mutation Guards

While the current GameState changes, we treat it with respect. You cannot simply modify `game.Players[0].Coins = 100`. You must use `game.AddCoins(0, 100)`. 

**Mental model**: The game state is a vault. The methods are the guards. They check ID badges (valid indices), verify permissions (sufficient coins), and enforce rules (no negative balances).

**Why this matters**: It prevents bugs. When you see `AddCoins`, you know it enforces the MaxCoins limit. When you see direct assignment, you have no guarantees.

## Actions: Rules as Data, Not Code

Game rules change. We might want to adjust costs, add new actions, or modify effects. How do we make this maintainable?

### The Mental Model: The Rulebook is a Database

Imagine a physical rulebook for a board game. It has a section for each action: "Steal costs 0 coins, claims the Thief role, and transfers 2 coins." If you want to change the cost, you edit that one line in the rulebook. You do not rewrite the entire game.

The `ActionRegistry` is our rulebook. It is a **data structure** (a map) that contains the rules for every action. When we want to change a rule, we change one field in one entry.

**Why this matters**: Rules become configurable. We could theoretically load the ActionRegistry from a JSON file or database at startup. Game designers could tweak balance without touching code.

### Validation vs. Execution: The Two-Phase Commit

Every action has two distinct phases:

1. **Validation**: "Is this legal?" (Check coins, check targets, check rules)
2. **Execution**: "What happens?" (Transfer coins, discard cards, update state)

These phases are strictly separated. Validation happens first. If it fails, execution never happens. If it passes, execution definitely happens (unless blocked by challenges/counters).

**Mental model**: This is like a two-phase commit in databases. First, we verify everyone is ready. Then, we actually do it. There is no partial failure.

**Why this matters**: You never get half an action. You never spend coins and then fail to get the effect. The separation ensures atomicity.

## Challenges and Counters: Truth and Interruption

The social deduction aspect of the game—challenging bluffs and countering actions—is the most complex part of the rules.

### The Mental Model: Claims and Verification

When a player plays an action that claims a role ("I have the Thief, so I steal"), they are making a **claim about reality**—specifically, about what cards are in their hand.

A challenge is a **request for proof**. "Prove you have the Thief."

The engine checks the truth: does the claimed role exist in the actor's hand? This is a **factual verification**, not a moral judgment. The result is binary: truth or lie.

Consequences follow from the fact:
- Truth: The challenger was wrong, so they lose a card.
- Lie: The actor was bluffing, so they lose their action and a card.

**Why this matters**: The engine is a truth engine, not a social engine. It does not care about psychology or tells or human behavior. It just checks facts and applies consequences.

### The Mental Model: Interruption and Replacement

Counters are **interruptions**. When you counter a steal, you are saying: "Stop. That action does not resolve. Instead, I block it (or something else happens)."

The engine handles this with a simple boolean: was the main action canceled? If yes, the main action's effects never apply. The counter's effects (if any) do apply.

**Why this matters**: This is how the engine thinks about all interruptions. A counter is just a conditional that sets `mainCanceled = true`.

## Testing: The Proof of Purity

The engine's purity enables testing that would be impossible in a traditional architecture.

### The Mental Model: Time Travel Debugging

Because the engine is deterministic and stateless, you can **replay any game exactly**. Feed in the same actions, get out the same states. This means:

- You can write tests that verify exact game states.
- You can debug by replaying the exact sequence that caused a bug.
- You can verify fixes by replaying the bug scenario and confirming it no longer occurs.

**Why this matters**: Debugging becomes scientific. You can reproduce bugs 100% of the time. You can verify fixes with certainty.

### The StubProvider: Fake Reality

In tests, we use a `StubProvider` that implements `InputProvider` but returns predetermined values. Instead of waiting for a WebSocket message, it immediately returns the action we told it to return.

**Mental model**: We are sliding fake messages through the mail slot. The game master processes them as if they were real.

**Why this matters**: Tests run instantly. No network setup. No timing issues. Just pure logic testing.

## Debugging: Knowing Where to Look

When something breaks, your mental model tells you where to look:

| Symptom | Where to Look | Why |
|---------|--------------|-----|
| Coins wrong after steal | `actions/apply.go` | Handler logic |
| Can steal with 0 coins | `actions/validators.go` | Validation missing |
| Challenge happens at wrong time | `engine/orchestrator_turn.go` | Turn flow |
| Negative coins allowed | `engine/state/gamestate.go` | Method missing check |
| Messages not arriving | `server/ws/*.go` | WebSocket code |
| Cannot join lobby | `server/lobby.go` | Lobby logic |

The architecture guides you. The separation of concerns means you never have to search the entire codebase.

## Extending the System: Where Changes Belong

When adding features, ask: **"Does this change the game rules, or how players interact with the game?"**

- **Game rules** → `actions/` or `engine/` (new actions, modified turn flow)
- **Player interaction** → `server/` (new lobby features, chat improvements)
- **Both** → Split the change across the boundary (rare)

**Mental model**: Respect the wall of the white room. Do not let reality leak in. Do not let game logic leak out.

## Common Pitfalls: What Breaks the Architecture

Even with clear principles, developers often accidentally blur the boundary between purity and reality. Here are the most common mistakes and why they destroy the system's guarantees.

### Pitfall 1: "Just Add a Database Call to the Engine"

**The mistake**: "I need to save the game state, so I will add a database.Save() call inside RunTurn()."

**Why it destroys everything**:
- The engine is no longer deterministic (database might fail or be slow)
- Tests now require a database setup
- You cannot replay games without the database
- The white room has a window—chaos enters

**The right way**: The engine returns state changes. The server saves them. The engine stays pure.

### Pitfall 2: "Use a Global Variable for the Current Time"

**The mistake**: "I need to check timeouts, so I will use time.Now() inside the engine."

**Why it destroys everything**:
- Tests become non-deterministic (time passes differently on each run)
- You cannot replay games at different speeds
- Time-based bugs become impossible to reproduce

**The right way**: Use the `Clock` interface. Pass a `RealClock` in production, a `TestClock` in tests.

### Pitfall 3: "Let the InputProvider Mutate the GameState"

**The mistake**: "The InputProvider needs to check the game state, so I will pass a pointer and let it modify things."

**Why it destroys everything**:
- The InputProvider (outside the white room) can corrupt the game
- Bugs become impossible to trace (who changed what?)
- The boundary is breached

**The right way**: Pass a read-only snapshot to the InputProvider. All mutations go through the engine's controlled channels.

### Pitfall 4: "Async/Await Would Be Cleaner"

**The mistake**: "All this blocking code is old-fashioned. I will rewrite it with async/await."

**Why it destroys everything**:
- Callbacks fragment the linear turn flow across functions
- Error handling becomes scattered
- The code is harder to reason about (what runs when?)

**The right way**: Keep it synchronous and blocking. The linear flow is a feature, not a bug.

## Design Trade-offs: Why We Chose This Way

Every architecture involves trade-offs. Understanding why we chose one path over another helps you make consistent decisions when extending the system.

### Trade-off: Synchronous vs. Asynchronous

**Why we chose synchronous (blocking)**:
- Linear code is readable and debuggable
- Error handling is simple (just return errors up the stack)
- No callback hell or promise chains

**What we gave up**:
- The calling goroutine is blocked (but we run it in its own goroutine, so this is fine)
- Cannot easily cancel mid-turn (we handle this with context cancellation)

**Why this is correct for us**: Game turns are naturally sequential. The complexity of async does not buy us anything.

### Trade-off: Snapshots vs. Deltas

**Why we chose snapshots (full state copies)**:
- Any state is understandable in isolation
- Replays are trivial (just show the snapshots)
- Debugging is easy (look at any point in time)

**What we gave up**:
- More memory usage (but game states are small)
- Slightly more CPU for copying (negligible compared to network latency)

**Why this is correct for us**: Developer time is more valuable than the small cost of snapshots.

### Trade-off: Centralized Registry vs. Scattered Rules

**Why we chose the ActionRegistry (centralized)**:
- All rules visible in one place
- Easy to modify (change one field)
- Could theoretically load from JSON/config

**What we gave up**:
- Slightly more indirection (lookup in map)
- Cannot use switch statements (but we do not want to)

**Why this is correct for us**: Game design changes frequently. Centralized rules make balance changes easy.

### Trade-off: Interface vs. Concrete Types

**Why we chose the InputProvider interface**:
- Engine is decoupled from implementation
- Can test with stubs
- Could swap WebSocket for TCP or CLI without touching engine

**What we gave up**:
- Cannot see implementation details from engine (this is actually good)
- Slightly more complex call stack

**Why this is correct for us**: Testing is crucial. The interface enables testability.

## Performance Considerations: When to Care

Most of the time, the engine's performance does not matter—network latency dominates. But there are specific places where performance does matter.

### Hot Path: RunTurn Loop

`RunTurn` is called for every turn of every game. It needs to be efficient:
- **Avoid allocations** in the turn loop (reuse buffers)
- **Minimize lock contention** (the engine is single-threaded per game, so this is natural)
- **Do not block on I/O** (the InputProvider handles this, not the engine)

**What does not matter**: Micro-optimizing the validation logic. It runs once per action. Network latency is 1000x slower.

### Hot Path: Snapshot()

`Snapshot` creates deep copies of the game state. This happens after every turn:
- **Keep GameState small** (do not add huge fields)
- **Use efficient copy methods** (built-in copy() for slices)
- **Avoid deep reflection** (use explicit copy code)

**Current state**: Snapshots are fast enough. A typical game state is < 10KB. Copying 10KB is negligible.

### Not Hot: Validation

Validating an action (checking coins, targets, etc.) happens once per action. It involves a few map lookups and bounds checks. This is not worth optimizing.

**Do not**: Add caching for validation. It adds complexity for no gain.

### Not Hot: ActionRegistry Lookup

Looking up an action in the registry is a map access. Maps in Go are fast. The registry is small (< 50 entries). Do not optimize this.

## Evolution: How to Extend Without Breaking

The system is designed to evolve. Here is how to add features without violating the architecture.

### Adding a New Action

1. **Define in models/**: Add ActionID constant
2. **Register in actions/**: Add to ActionRegistry with validator and handler
3. **Test**: Write unit tests for handler, integration tests for flow
4. **Done**: No changes to engine or server needed

**Why this works**: The engine processes actions generically. New actions are just new data in the registry.

### Adding a New Game Mode

1. **Extend Game struct**: Add mode field if needed
2. **Modify orchestration**: Update RunTurn to check mode-specific rules
3. **Pass through server**: Server just passes the mode when creating Game

**Why this works**: The engine owns game logic. Adding modes is changing that logic.

### Adding Real-time Features (Spectators, Chat)

1. **Keep out of engine**: These do not affect game state
2. **Add to server/**: Handle in WebSocket layer
3. **Broadcast separately**: Chat messages go through server, not engine

**Why this works**: Real-time features are reality-world concerns. They stay in the server.

### Adding Persistence

1. **Snapshot after each turn**: Save GameState.History
2. **Save in server**: Server calls Save() after engine returns
3. **Load on reconnect**: Server loads history, replays through engine

**Why this works**: Persistence is a reality concern. The engine just produces states; the server saves them.

## Migration Patterns: Changing the Architecture

Sometimes you need to change something fundamental. Here is how to do it safely.

### Migration: Changing the State Schema

**Scenario**: You need to add a new field to GameState.

**Approach**:
1. Add field to GameState struct
2. Update Snapshot() to copy the new field
3. Update any methods that need to use it
4. Add tests
5. Deploy (old games without the field will use zero values, which is fine)

**Safety**: Go's zero values make schema changes safe. New fields default to empty/nil/false.

### Migration: Changing the InputProvider Interface

**Scenario**: You need to add a new method to InputProvider.

**Approach**:
1. Add method to interface
2. Implement in all providers (stub, session, etc.)
3. Update engine to use the new method
4. Test thoroughly

**Safety**: The compiler will force you to update all implementations. You cannot forget one.

### Migration: Changing an Action

**Scenario**: You want to change how Steal works.

**Approach**:
1. Modify the handler in actions/apply.go
2. Update the validator if needed
3. Update tests
4. Deploy

**Safety**: Old games in progress will use the new logic immediately (they call the handler). If this is not desired, you need versioning.

## Anti-Patterns: Code That Signals Trouble

Certain code patterns indicate that someone is fighting the architecture. Learn to recognize them.

### Anti-Pattern: "Engine Imports Server Package"

```go
// WRONG: engine should not import server
import "ElMakina/backend/server"
```

**What this means**: The boundary is breached. The engine knows about reality.

**Fix**: Move the code to the server, or define an interface in the engine that the server implements.

### Anti-Pattern: "Direct State Mutation"

```go
// WRONG: bypassing the guards
game.Players[0].Coins = 100
```

**What this means**: Someone is bypassing the validation guards. Invalid states become possible.

**Fix**: Always use methods like AddCoins(). If the method does not exist, add it with proper validation.

### Anti-Pattern: "Time.Now() in Engine"

```go
// WRONG: time in the white room
if time.Now().After(deadline) { ... }
```

**What this means**: The engine is no longer deterministic.

**Fix**: Use the Clock interface. Pass time through from the server.

### Anti-Pattern: "Global Config in Engine"

```go
// WRONG: global state
var GlobalTimeout = 30 * time.Second
```

**What this means**: The engine has hidden dependencies. Tests cannot control the value.

**Fix**: Pass config through parameters. Make dependencies explicit.

## The Human Element: Code Reviews and Team Culture

Architecture is not just about code—it is about **people**. How do you maintain these principles as the team grows?

### Code Review Checklist

When reviewing changes, ask:
- [ ] Does this respect the purity boundary? (Engine does not import server)
- [ ] Are state mutations guarded? (Methods, not direct assignment)
- [ ] Is time handled correctly? (Clock interface, not time.Now())
- [ ] Are errors handled at the right layer? (Server handles disconnects, engine handles invalid actions)
- [ ] Are tests deterministic? (No unseeded random, no real time)

### Documentation as Contract

The architecture is documented in two places:
1. **This file (BREAKDOWN.md)**: The philosophy and mental models
2. **The code itself**: Interfaces, function signatures, tests

When the code and the docs disagree, **the code is right** (it is what actually runs). But we should update the docs when we intentionally change the architecture.

### Onboarding New Developers

When a new developer joins:
1. Have them read this document first
2. Walk through a simple test (show them the stubProvider pattern)
3. Have them make a small change (add a log message, fix a typo)
4. Review their first real change with extra care (watch for boundary violations)

**The goal**: They should understand the "white room" mental model before they write any significant code.

## Summary: Architecture as Worldview

ElMakina's architecture is not just a technical choice—it is a **worldview** about how to manage complexity:

1. **Separate purity from reality** — Let each be what it is without compromise
2. **Make the core deterministic** — Timeless logic that can be tested and replayed
3. **Add compassion at the edges** — Handle real-world messiness without infecting the core
4. **Centralize rules as data** — Make the game configurable and maintainable
5. **Guard state with methods** — Prevent invalid states through enforcement, not convention
6. **Document the philosophy** — Help developers understand the "why," not just the "what"

When you work on this codebase, you are not just writing code. You are maintaining a **philosophical boundary** between order and chaos. Respect that boundary, and the system remains clean, testable, and maintainable. Blur that boundary, and you invite the complexity that destroys projects.

The white room must remain white.
