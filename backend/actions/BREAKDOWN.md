# Actions Package Breakdown: The Rulebook of Reality

## The Core Philosophy: Rules as Data, Not Dogma

Every game is a contract between players—a shared fiction that only exists because everyone agrees on the rules. The moment the rules become unclear or inconsistent, the game collapses. The actions package exists to prevent this collapse by treating rules not as scattered code, but as **centralized, inspectable data**.

This is a radical departure from how most games are implemented. Typically, game rules are implicit—buried in switch statements, scattered across handlers, encoded in the shape of the code itself. Want to know if the Thief can block a steal? You must grep through ten files and hope you found all the branches. Want to change the cost of Coup from 7 to 8? You must find every place that checks the cost, hoping you do not miss one.

We reject this approach. In ElMakina, the rules are **explicit**. They live in the ActionRegistry, a single map where every action's properties are declared for all to see. The code reads the rules; it does not embody them. This separation between rule and mechanism is the foundation of maintainability.

### The Mental Model: The Library of Alexandria

Imagine a vast library where every scroll contains the complete definition of one action. Each scroll answers every question about that action: What does it cost? Who can play it? What happens when it resolves? Can it be challenged? Can it be blocked?

The ActionRegistry is this library. When the engine needs to know something about an action, it does not guess or infer—it consults the library. The registry is the **single source of truth** for game rules.

**Why this matters**: The library can be inspected, modified, even generated. Game designers could theoretically edit the rules without touching code. Balance patches become data changes. The rules become visible, tangible, changeable.

## The Architecture of Action: Validation and Execution

Every action moves through two distinct phases: the question of permission, and the fact of consequence.

### ASCII Flow: Action Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│                    ACTION LIFECYCLE FLOW                         │
└─────────────────────────────────────────────────────────────────┘

Player declares action
         │
         ▼
┌─────────────────┐
│   VALIDATION    │◄──┐
│   (Gatekeeper)  │   │
│                 │   │
│ • Check cost    │   │ Invalid
│ • Check target  │───┴───▶ Reject (no state change)
│ • Check rules   │
└────────┬────────┘
         │ Valid
         ▼
┌─────────────────┐     ┌──────────────────┐
│  CHALLENGE      │     │    COUNTER       │
│   WINDOW        │     │    WINDOW        │
│                 │     │                  │
│ Others can say  │     │ Targets can say  │
│ "You're lying"  │     │ "I block you"    │
└────────┬────────┘     └────────┬─────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌──────────────────┐
│ Challenge       │     │ Counter         │
│ successful?     │     │ successful?     │
└────────┬────────┘     └────────┬─────────┘
         │                       │
    Yes  │                       │ Yes
         ▼                       ▼
┌─────────────────┐     ┌──────────────────┐
│ Action CANCELED │     │ Action CANCELED  │
│ Actor loses     │     │ by counter       │
│ card            │     └────────┬─────────┘
└────────┬────────┘              │
         │                       │
         │ No                    │ No
         └───────────┬───────────┘
                     │
                     ▼
            ┌─────────────────┐
            │   EXECUTION     │
            │   (Handler)     │
            │                 │
            │ • Transfer coins│
            │ • Move cards    │
            │ • Update state  │
            └────────┬────────┘
                     │
                     ▼
            ┌─────────────────┐
            │  TURN COMPLETE  │
            └─────────────────┘
```

### Phase 1: Validation — The Gatekeeper

Before an action can affect the world, it must pass through the validator—a function that asks: "Is this action legal in this moment?"

The validator is a **pure function**. It receives the current game state and the proposed action, and returns either permission (nil error) or denial (an error explaining why). It does not modify state. It does not apply effects. It only judges.

**Mental model**: The validator is a customs official inspecting a traveler's documents. The official checks: Do you have a valid passport? Do you have the required visa? Do you meet the entry requirements? The official does not care where you will go once admitted, only whether you may enter.

**Common validation checks**:
- Does the actor have enough coins to pay the cost?
- Is the target player index valid (within bounds, still alive)?
- Does the target have the required resources (coins to steal, cards to discard)?
- Is it the actor's turn?
- Are there special restrictions (e.g., cannot target yourself)?

**Critical insight**: Validation happens **before** any state changes. This ensures atomicity—an invalid action leaves no trace. If you attempt a Coup with 6 coins, the validator rejects you before you spend anything. You keep your coins, the game continues, and no one knows you tried (except perhaps in logs).

### Phase 2: Execution — The Agent of Change

Once validation passes, the handler applies the action's effects. This is where the game state transforms—coins transfer, cards move, players lose influence.

The handler is **not a pure function** in the mathematical sense—it mutates the game state. But it is pure in intent: given the same starting state and same action, it always produces the same ending state. There are no random surprises in execution (randomness, if any, is controlled by the engine's RNG).

**Mental model**: The handler is a skilled artisan performing a craft. The validator said "yes, you may build." Now the artisan builds, following the blueprint (the action definition) precisely.

**Why separation matters**:
- **Security**: Invalid actions cannot accidentally change state
- **Clarity**: You can read the validator to understand constraints, the handler to understand effects
- **Testing**: You can test validation separately from execution
- **Debugging**: You know exactly where to look—validation errors mean check the validator, wrong effects mean check the handler

## The ActionRegistry: Centralization as Wisdom

The registry is the heart of the actions package. Understanding its design reveals the philosophy of the entire system.

### ASCII Diagram: The Registry as Library

```
┌─────────────────────────────────────────────────────────────────┐
│                  ACTION REGISTRY (The Library)                   │
└─────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────┐
│                     ActionRegistry Map                          │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  "steal"    │    │   "coup"    │    │  "exchange" │         │
│  │             │    │             │    │             │         │
│  │ ActionDef   │    │ ActionDef   │    │ ActionDef   │         │
│  │ ┌─────────┐ │    │ ┌─────────┐ │    │ ┌─────────┐ │         │
│  │ │Cost: 0  │ │    │ │Cost: 7  │ │    │ │Cost: 0  │ │         │
│  │ │Role:    │ │    │ │Role:    │ │    │ │Role:    │ │         │
│  │ │ Thief   │ │    │ │  nil    │ │    │ │Politician│         │
│  │ │Handler  │ │    │ │Handler  │ │    │ │Handler  │ │         │
│  │ │Validator│ │    │ │Validator│ │    │ │Validator│ │         │
│  │ └─────────┘ │    │ └─────────┘ │    │ └─────────┘ │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
└────────────────────────────────────────────────────────────────┘

         ▲                    ▲                    ▲
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │
                              │ Lookup
                              │
         ┌────────────────────┴────────────────────┐
         │                                         │
         ▼                                         ▼
┌──────────────────┐                  ┌──────────────────────┐
│  Engine asks:    │                  │   Engine receives:   │
│ "What is steal?" │                  │   ActionDef struct   │
└──────────────────┘                  │   with all metadata  │
                                      └──────────────────────┘
                                              │
                                              │ Uses
                                              ▼
                              ┌───────────────────────────────┐
                              │   Validation: call Validator  │
                              │   Execution:  call Handler    │
                              └───────────────────────────────┘
```

### Why a Map, Not Methods?

### Why a Map, Not Methods?

We could have implemented actions as methods on a Game struct:

```go
// Alternative we rejected
func (g *Game) DoSteal(target int) error { ... }
func (g *Game) DoCoup(target int) error { ... }
```

We chose a registry instead because:

1. **Uniformity**: Every action is processed through the same code path. The engine does not need special cases for different actions.

2. **Introspection**: We can ask the registry questions like "What actions claim the Thief role?" or "What counters can block Steal?" This enables dynamic behavior (e.g., showing players only the actions they can afford).

3. **Configurability**: The registry could be loaded from a JSON file. Game balance could be tuned without recompiling.

4. **Extensibility**: Adding a new action requires no changes to the engine—just a new entry in the registry.

**Mental model**: The registry is a dictionary. The engine looks up words (action IDs) to find their definitions. It does not need to know the words in advance—it just needs to know how to read the dictionary.

### The Anatomy of ActionDef

Each entry in the registry is an ActionDef—a struct that completely describes one action:

- **ID**: The action's identifier (redundant but convenient)
- **Kind**: Main or Counter (determines when it can be played)
- **RoleClaim**: Which role this action claims (empty if none)
- **BlockedActions**: For counters—which actions does this block?
- **AllowMultiple**: Can multiple players play this counter simultaneously?
- **Cost**: Coins required to play
- **Handler**: Function that executes the action
- **Validator**: Function that checks legality

This is **declarative programming**. We declare what the action is, and the system figures out how to use it.

## Main Actions vs Counter Actions: The Asymmetry of Agency

Game actions divide into two categories with fundamentally different temporal properties.

### Main Actions: The Protagonists

Main actions are played on your turn. They drive the narrative forward. They are the **protagonists** of the game story—each turn, a main action takes center stage.

Main actions claim the spotlight: "I am happening now. Respond to me."

They open challenge windows (if they claim a role) and counter windows (if they can be blocked). They demand attention.

**Examples**: Income, Foreign Aid, Coup, Steal, Assassinate, Exchange

### Counter Actions: The Antagonists

Counter actions are played in response to main actions. They are **reactive**, not proactive. They exist to interrupt, to block, to say "no."

Counter actions steal the spotlight momentarily, then return it. They do not advance the turn—they halt or redirect it.

**Examples**: Block Steal, Block Terrorist, Block Foreign Aid, Escape

### Why the Distinction Matters

The engine treats these differently:
- Main actions trigger the full turn flow (validation → challenge window → counter window → execution)
- Counter actions are collected during the counter window and evaluated as a group
- Main actions are chosen by the current player; counters can be played by anyone eligible
- Main actions claiming roles can be challenged; counters claiming roles can also be challenged

Understanding this asymmetry is crucial when adding new actions. Ask: Is this something a player does on their turn, or something they do in response to others?

## Multi-Step Actions: The Illusion of Simplicity

Some actions appear simple but require mid-execution decisions. Exchange is the canonical example: draw two cards, choose which to keep, return the rest. This seems like one action, but it requires a pause for player input.

### ASCII Diagram: ActionFlow Pause and Resume

```
┌─────────────────────────────────────────────────────────────────┐
│              MULTI-STEP ACTION FLOW (ActionFlow)                 │
└─────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────┐
│                         Exchange Action                          │
└──────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│  1. VALIDATION  │
│  (has Politician│
│   role? etc)    │
└────────┬────────┘
         │ Valid
         ▼
┌─────────────────┐
│ 2. DRAW CARDS   │
│                 │
│ Player draws 2  │
│ from deck       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────────────┐
│ 3. PLACE        │     │     ENGINE SENDS         │
│    BOOKMARK     │────▶│     StepRequest:         │
│                 │     │     "Choose 2 cards      │
│ Create          │     │      to keep"            │
│ ActionFlow      │     └───────────┬──────────────┘
│ channel         │                 │
└────────┬────────┘                 │
         │                          │
         │          PAUSE           │
         │◄─────────────────────────┘
         │    (Engine waits for
         │     player response)
         │
         │ Resume
         ▼
┌─────────────────┐
│ 4. RECEIVE      │
│    CHOICE       │
│                 │
│ Player chose    │
│ cards [0, 2]    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 5. RETURN       │
│    CARDS        │
│                 │
│ Return unchosen │
│ to deck         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 6. COMPLETE     │
│    ACTION       │
└─────────────────┘

┌────────────────────────────────────────────────────────────────┐
│  CODE PERSPECTIVE (Linear, despite pause):                     │
│                                                                │
│  func applyExchange(flow *ActionFlow, ...) error {             │
│      // Step 2: Draw cards                                     │
│      s.DrawCards(actor, 2)                                     │
│                                                                │
│      // Step 3: Place bookmark (PAUSE HAPPENS HERE)            │
│      flow.Requests <- StepRequest{...}                         │
│      response := <-flow.Responses   // ← BLOCKS/WAITS         │
│                                                                │
│      // Step 5: Return cards (RESUME HERE)                     │
│      s.ReturnCards(actor, response)                            │
│      return nil                                                │
│  }                                                             │
└────────────────────────────────────────────────────────────────┘
```

### The Mental Model: The Bookmark

Imagine reading a book and reaching a page that says "Choose your own adventure: turn to page 45 or page 82." You place a bookmark, go to the chosen page, then eventually return and continue reading.

Multi-step actions work similarly. The handler:
1. Performs the first part (draw cards)
2. Sends a StepRequest to the InputProvider (places a bookmark)
3. **Pauses** (the engine waits)
4. Receives the player's choice (turns to the chosen page)
5. Completes the action (continues reading)

From the action's perspective, this is linear code: draw, request, receive, return. The complexity of the pause/resume is handled by the ActionFlow machinery in the engine.

**Why this design**: It lets us write multi-step actions as if they were single-step. The linear flow is readable. The complexity is abstracted away.

## Common Pitfalls: When Actions Go Wrong

Even with clear architecture, developers often make mistakes when working with actions. Here are the most dangerous patterns.

### Pitfall 1: "I'll Just Add a Special Case in the Engine"

**The mistake**: "This action needs special handling, so I'll add an if-statement in RunTurn."

**Why it destroys everything**:
- The engine becomes coupled to specific actions
- Adding new actions requires engine changes
- The registry becomes a lie (actions exist outside the registry)
- Testing becomes harder (engine tests must know about specific actions)

**The right way**: Extend the ActionDef to include the special behavior as data. If Steal needs special targeting logic, add a TargetingMode field to ActionDef. Keep the engine generic.

### Pitfall 2: "Mutation in Validation"

**The mistake**: "I need to check if the target has coins, so I'll deduct them in the validator and refund if invalid."

**Why it destroys everything**:
- Invalid actions now have side effects
- State changes even when the action should be rejected
- Debugging becomes a nightmare (why did coins disappear?)

**The right way**: Validation must be **pure**. Check state, do not change it. If you need complex pre-checks, add a helper method to GameState that returns information without modification.

### Pitfall 3: "Inconsistent Error Messages"

**The mistake**: Each validator returns different error formats. Some return errors.New("not enough coins"), others return fmt.Errorf("player %d has %d coins but needs %d", player, have, need).

**Why it hurts**:
- The server cannot parse errors consistently
- Client error messages are unpredictable
- Internationalization becomes impossible

**The right way**: Use structured error types. Define validation error codes in models/. Return typed errors that the server can inspect and translate.

### Pitfall 4: "Bypassing the Registry"

**The mistake**: Writing code that checks action IDs directly instead of consulting the registry:

```go
// WRONG: hardcoded action knowledge
if action.ID == "steal" {
    // special steal logic
}
```

**Why it hurts**:
- Adding new steal-like actions requires code changes
- The registry's metadata is ignored
- Logic scatters across the codebase

**The right way**: Ask the registry: "Does this action claim a role?" "Can this action be countered?" Let the ActionDef drive the logic.

## Design Trade-offs: Why We Chose This Way

### Trade-off: Centralized Registry vs. Distributed Methods

**Why we chose centralized (registry)**:
- Single place to see all rules
- Engine remains generic (processes any registered action)
- Rules can be loaded from external sources
- Adding actions requires no engine changes

**What we gave up**:
- Slightly more indirection (map lookup)
- Cannot use IDE "go to definition" as easily
- More verbose than direct method calls

**Why this is correct**: Game design changes frequently. Centralized rules make balance changes trivial. The engine's genericity is worth the small cost of indirection.

### Trade-off: String IDs vs. Enum Constants

**Why we chose strings ("steal", "coup")**:
- Self-documenting in logs and JSON
- No ordering constraints (can add/remove without renumbering)
- Forward compatible (old clients can ignore unknown actions)

**What we gave up**:
- Type safety (can typo a string)
- Slightly more memory than integers

**Mitigation**: We define constants in models/: `const Steal ActionID = "steal"`. The compiler catches typos in code, while logs remain readable.

### Trade-off: Validation Functions vs. Validation Structs

**Why we chose functions**:
- Arbitrary validation logic is expressible
- Can check complex conditions ("target must have coins AND not be immune")
- Go's type system makes functions first-class

**What we gave up**:
- Cannot serialize validation rules to JSON
- Harder to generate validators automatically

**Why this is correct**: Validation often requires game-specific logic. Functions are the right tool for arbitrary computation.

## Performance Considerations: The Cost of Rules

### Registry Lookup: Negligible

Looking up an action in the registry is a map access. Go maps are O(1) and highly optimized. With fewer than 50 actions, the registry fits in cache. **Do not optimize this.**

### Validation: Hot Path, But Fast

Validation runs once per action. It involves:
- Map lookup (ActionDef)
- Function call (validator)
- A few bounds checks and arithmetic operations

This is **not a bottleneck**. Network latency is 1000x slower. **Do not add caching** for validation—it adds complexity for no gain.

### Handler Execution: The Real Work

Handlers modify game state: moving coins, transferring cards, updating logs. This is the real work of the action. But even here, the cost is negligible—a few slice operations and integer updates.

**The only expensive operation**: Multi-step actions that wait for player input. The "cost" here is wall-clock time, not CPU time. Players take seconds to decide; the computer waits patiently.

## Evolution: Adding and Modifying Actions

### Adding a New Action

The registry makes this straightforward:

1. **Define the ID** in models/: `const MyAction ActionID = "my_action"`
2. **Write the validator**: Check prerequisites, return error if invalid
3. **Write the handler**: Apply effects, write to log
4. **Register in ActionRegistry**: Add entry with metadata
5. **Test**: Unit test the validator and handler; integration test the flow

**No engine changes required.** The engine processes registered actions generically.

### Modifying an Existing Action

Change the registry entry:
- Adjust Cost: Change the Cost field
- Change targeting: Update the validator
- New effects: Modify the handler
- Add role claim: Set RoleClaim field

**Risk**: In-progress games will use the new logic immediately. If this is undesirable (e.g., changing core mechanics), you need versioning.

### Versioning Actions

If you must support multiple versions of rules:

1. Add a Version field to GameState
2. In the handler, check version and branch:
   ```go
   if gs.Version < 2 {
       oldBehavior()
   } else {
       newBehavior()
   }
   ```
3. Default new games to the latest version

**Better approach**: Version the entire ActionRegistry. Load different registries for different game versions. This keeps version checks out of individual handlers.

## Anti-Patterns: Code That Signals Trouble

### Anti-Pattern: "Action-Specific Engine Logic"

```go
// WRONG: engine knows about specific actions
if action.ID == models.Steal {
    // special handling for steal
} else {
    // generic handling
}
```

**What this means**: The engine is no longer generic. Adding actions requires engine changes.

**Fix**: Move the special behavior into the ActionDef. Add fields that describe the behavior (e.g., `RequiresTarget`, `CanBeChallenged`). The engine reads these fields and acts generically.

### Anti-Pattern: "Validation by Exception"

```go
// WRONG: using panic/recover for validation
try {
    gs.DeductCoins(player, cost)
} catch (InsufficientCoins) {
    return error
}
```

**What this means**: Validation is implicit, scattered, and exception-based. Hard to reason about.

**Fix**: Explicit validation functions that return errors. Validate first, then execute.

### Anti-Pattern: "Silent Failures in Handlers"

```go
// WRONG: ignoring errors
func handleSteal(gs *GameState, cmd ActionCommand) {
    target := cmd.Payload.(TargetPayload).TargetIndex
    gs.TransferCoins(actor, target, 2) // ignore error
}
```

**What this means**: Errors are swallowed. Invalid states become possible.

**Fix**: Always check and return errors. If an error "should never happen" (because validation already checked), still handle it—assertions can fail, bugs exist.

### Anti-Pattern: "Leaking Action Details"

```go
// WRONG: server inspecting action internals
if action.ID == "exchange" {
    // server knows Exchange has special payload structure
    cards := action.Payload.([]Card)
    // ...
}
```

**What this means**: The server is coupled to action implementation details. Changing the action requires server changes.

**Fix**: The server should treat actions as opaque. Parse JSON into models.PlayerAction, pass to engine. Let the actions package handle the details.

## Team Culture: Maintaining the Rulebook

### Code Review Questions

When reviewing action changes, ask:
- [ ] Is this registered in the ActionRegistry?
- [ ] Does the validator check all prerequisites?
- [ ] Does the handler handle all error cases?
- [ ] Are there tests for both success and failure paths?
- [ ] Does this duplicate existing functionality (DRY violation)?
- [ ] Are the costs/balances documented?

### Documentation as Rulebook

The registry is code-as-documentation, but some things need prose:
- **Balance rationale**: Why does Coup cost 7? Why not 6 or 8?
- **Interaction rules**: How do multiple counters resolve?
- **Timing rules**: When exactly can challenges be played?

Document these in comments near the registry, or in a separate game-design.md.

### Onboarding: Understanding Actions

When a new developer needs to add an action:
1. Have them read this document
2. Show them a simple existing action (Income, Coup)
3. Show them a complex existing action (Exchange, with multi-step flow)
4. Have them copy-paste-modify a simple action first
5. Review with extra care (watch for registry bypassing)

## Summary: The Rulebook Must Be Sacred

The actions package is the **rulebook of the game**. When it is clean and central, the game is maintainable. When it is scattered and implicit, the game becomes chaos.

Key principles:
1. **Rules as data**: The ActionRegistry is the source of truth
2. **Separation of concerns**: Validation asks permission; handlers execute
3. **Generic engine**: The engine processes actions, does not know them
4. **Atomicity**: Invalid actions leave no trace
5. **Extensibility**: New actions require no engine changes

Respect the registry. Keep validation pure. Let the engine stay generic. The rulebook must be sacred—clear, explicit, and central.

When players argue about the rules, the registry settles the dispute. When designers want to tweak balance, the registry makes it easy. When developers add features, the registry guides them.

The rulebook is the foundation. Keep it strong.
