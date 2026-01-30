# Models Package Breakdown: The Vocabulary of Shared Understanding

## The Core Philosophy: Language Precedes Action

Before two people can cooperate, they must agree on words. If one calls it a "User" and another calls it a "Player," communication fragments. If one sends a message with field "target" and another expects "targetIndex," integration fails. Language is the invisible infrastructure that enables all other work.

The models package is this language. It contains no logic—no algorithms, no decisions, no state changes. It only contains **definitions**: what an Action is, what a Role is, how messages are structured. It is the dictionary that the entire codebase uses to communicate.

This package is intentionally the most boring code in the system. Boring is the goal. When code here changes, it should be because the game design changed (we added a new action), not because of technical churn. Stability is the supreme virtue.

### The Mental Model: The Rosetta Stone

Imagine an ancient stone tablet inscribed with the same text in three languages. Scholars from different civilizations can gather around it and agree: "This symbol means the same thing in all our tongues."

The models package is this Rosetta Stone. When the server sends an ActionID, the engine receives the same ActionID. When the engine returns a Role, the client recognizes the same Role. The types ensure that all packages speak the same language.

**Why this matters**: Without shared definitions, the system fragments into dialects. The server speaks Server-ish, the engine speaks Engine-ish, and translators between them introduce bugs. With shared models, there is one language, one truth, one understanding.

## The Architecture of Types: Strings as Self-Documentation

### Why Strings Instead of Integers?

Many systems use integer enums for identifiers: `ActionSteal = 1`, `ActionCoup = 2`. We deliberately chose strings: `Steal ActionID = "steal"`, `Coup ActionID = "coup"`.

**The case for integers**:
- More compact in memory and transmission
- Faster comparison (integer equality vs string comparison)
- Type safety with enums

**Why we rejected it**:

1. **Debugging**: When you see `"steal"` in a log, you know exactly what happened. When you see `7`, you must look up the mapping. Strings are **self-documenting** in the most important context—when things go wrong.

2. **Forward compatibility**: With integers, adding a new action requires choosing the next available number. Removing an action leaves a gap. With strings, you just add a new string. No gaps, no ordering, no confusion.

3. **Human-readable protocols**: WebSocket messages contain strings. Developers can read them without decoding tools. Testing with curl or websocat becomes possible.

**Performance note**: The cost is negligible. ActionIDs are compared only a few times per turn. Network latency dominates by orders of magnitude. Readability is worth more than nanoseconds.

### Constants: The Compromise

To get the benefits of type safety while keeping readability, we define constants:

```go
const (
    Steal  ActionID = "steal"
    Coup   ActionID = "coup"
    Income ActionID = "income"
)
```

This gives us:
- **Compiler protection**: Typo `Steal` and the compiler catches it
- **IDE support**: Autocomplete shows available actions
- **Refactoring**: Rename a constant, IDE updates all usages
- **Readability**: Logs still show `"steal"`, not `1`

Best of both worlds.

## The Main Types: Building Blocks of Meaning

### ActionID: The Universe of Possibility

ActionID represents every possible move a player can make. It is the **vocabulary of agency**—all the ways a player can affect the game.

The complete list includes:
- **Base actions**: Income (1 coin), Foreign Aid (2 coins, blockable), Coup (7 coins, unblockable)
- **Role actions**: Tax (claim Tax Collector, 3 coins), Steal (claim Thief, steal 2), Assassinate (claim Terrorist, force discard)
- **Counter actions**: Block Steal, Block Terrorist, Block Foreign Aid, Escape
- **Special actions**: Exchange (swap cards), Accuse (guess role), Pass (do nothing), GiveUp (forfeit)

**Design principle**: The list is comprehensive. If a player can do it, it is in this list. If it is not in this list, players cannot do it.

**Extending**: Add new constants here. The rest of the system adapts automatically.

### Role: The Character of Identity

Role represents the seven character types in the game. Each role is both an identity and a set of capabilities.

- **Businesswoman**: Accumulate wealth quickly (4 coins), vulnerable to taxation
- **Tax Collector**: Control economy (block aid, tax business), information broker
- **Policewoman**: Reveal secrets (investigate), protect herself from investigation
- **Colonel**: Defensive powerhouse (block assassinations), accuser of roles
- **Terrorist**: Direct attacker (assassinate for 3 coins), feared presence
- **Thief**: Economic disruptor (steal 2 coins), protected from theft
- **Politician**: Adaptive survivor (exchange cards), flexible identity

**The social contract**: When you play a role action, you claim to have that role. Other players can challenge—"Prove it." The roles enable the social deduction layer of the game.

### PlayerAction: The Universal Message Format

PlayerAction is the envelope that carries all player intentions. Whether the message comes from a WebSocket, a test case, or a future CLI, it becomes a PlayerAction before reaching the engine.

**Structure**:
- **ID**: What action (ActionID)
- **SourceIndex**: Who is acting (player index)
- **Payload**: Action-specific data (varies by action type)
- **MainAction**: For counters—which action is being blocked?

**The any type**: Payload uses `any` (interface{}) because different actions need different data. Steal needs a target index. Exchange needs card selections. Income needs nothing.

**Trade-off**: We lose compile-time type safety on payloads. A handler must type-assert: `payload := cmd.Payload.(TargetPayload)`. If wrong, panic at runtime.

**Mitigation**: 
- Validators check payload type immediately
- Tests catch type mismatches quickly
- The pattern is consistent (always check at entry point)

**Alternative considered**: Use a struct with optional fields for all possible payloads. Rejected because it couples all actions together—adding an action with a new payload type requires modifying the shared struct.

### Payload Types: The Details of Intent

Payloads carry action-specific information:

- **TargetPayload**: Which player is targeted (used by Steal, Coup, Assassinate, Investigate)
- **AccusePayload**: Who to accuse and what role to guess (used by Accuse)
- **CardSelection**: Which cards to keep/return (used by Exchange)

**Design principle**: Payloads are **minimal**. They contain only what the action needs, no more. This reduces coupling and makes testing easier.

### ActionKind: The Classification of Agency

ActionKind distinguishes Main actions from Counter actions. This is a **temporal distinction**—when can this action be played?

- **MainKind**: Played on your turn, proactive
- **CounterKind**: Played in response, reactive

**Why a separate field**: Actions could theoretically be both (though currently none are). The separation lets the engine treat them differently in the turn flow without hardcoding specific action IDs.

## The Isolation of Models: Why No Dependencies?

The models package imports nothing from the rest of the codebase. It is a **leaf node** in the dependency graph.

**Why this matters**:

1. **Circular import prevention**: Engine imports models. Server imports models. Actions imports models. Models imports none of them. No cycles possible.

2. **Stability**: Changes to engine logic do not affect models. Changes to server handling do not affect models. The vocabulary remains constant while implementation evolves.

3. **Universal importability**: Any package can import models without fear. It will not pull in heavy dependencies or cause cycles.

**The cost**: Models cannot use types defined elsewhere. If two packages need to share a type, it must be defined in models (or a shared dependency). This is intentional—it forces us to keep shared types minimal and stable.

## Common Pitfalls: When the Language Fractures

### Pitfall 1: "I'll Just Add This Field for Convenience"

**The mistake**: Adding a field to a struct because one consumer needs it, even though it is not part of the shared concept.

**Example**: Adding `ConnectionID` to Player because the server wants it there. But the engine does not know about connections.

**Why it destroys everything**:
- Models becomes coupled to specific consumers
- Other consumers get fields they do not need
- The shared vocabulary becomes polluted

**The right way**: Keep models minimal. If the server needs to associate a ConnectionID with a player, it maintains a separate map: `map[playerID]connectionID`. Do not force all consumers to carry your baggage.

### Pitfall 2: "Integers Are More Efficient"

**The mistake**: Changing ActionID from string to int for "performance."

**Why it destroys everything**:
- Logs become unreadable (what is action 7?)
- Debugging becomes harder
- The change ripples through the entire system
- The performance gain is negligible (measured in nanoseconds)

**The right way**: Keep strings. Profile first. Only optimize what matters.

### Pitfall 3: "I'll Use a Pointer for Optional Fields"

**The mistake**: Using `*string` instead of `string` to represent "optional" or "unset."

**Why it hurts**:
- Nil checks everywhere
- Risk of nil pointer dereference
- More complex serialization

**The right way**: Use the zero value (empty string `""`) to mean "unset." Add a boolean `HasX` field if you need to distinguish "unset" from "empty." Or use a separate `Optional[T]` type if you really need it.

### Pitfall 4: "JSON Tags Are Implementation Details"

**The mistake**: Treating JSON struct tags as "just for serialization" and changing them casually.

**Why it destroys everything**:
- Breaks backward compatibility with clients
- Other services consuming the API break
- The wire format is a **public contract**

**The right way**: JSON tags are **public API**. Changing them requires the same care as changing function signatures. Version your API if you must change them.

## Design Trade-offs: Why We Chose This Way

### Trade-off: any vs. Discriminated Union

**Why we chose any for Payload**:
- Go does not have native discriminated unions
- Type switches work well enough
- Simple and flexible

**What we gave up**:
- Compile-time exhaustiveness checking
- Type safety without runtime checks

**Alternative considered**: Define a Payload interface with a marker method. Each payload type implements it. Rejected because it adds boilerplate (empty methods) for little gain.

**Why this is correct**: The validator immediately checks payload type. Tests catch mismatches. The risk is contained.

### Trade-off: Flat vs. Namespaced Constants

**Why we chose flat constants** (`Steal`, `Coup`) over namespaced (`Action.Steal`, `Action.Coup`):
- Go does not have enums or namespaces
- Flat constants are idiomatic in Go
- Shorter to type and read

**What we gave up**:
- Organization (all ActionIDs mixed with all Roles)
- Potential for naming collisions

**Mitigation**: Use clear, unique names: `Steal` (action) vs `Thief` (role). If collisions occur, prefix: `ActionSteal`, `RoleThief`.

### Trade-off: Structs vs. Interfaces

**Why we chose structs for models**:
- Simple, concrete, serializable
- JSON marshaling works automatically
- No hidden behavior

**What we gave up**:
- Polymorphism (cannot have multiple Player implementations)
- Cannot enforce methods on data types

**Why this is correct**: Models are **data transfer objects**. They should be dumb containers. Behavior belongs in packages that consume them.

## Evolution: Extending the Vocabulary

### Adding a New Action

1. Add constant to models/: `const MyAction ActionID = "my_action"`
2. If needed, add payload type: `type MyActionPayload struct { ... }`
3. Register in actions/ ActionRegistry
4. Update any documentation

**Compatibility**: Old clients will not recognize the new action ID. They should ignore unknown actions gracefully. The server should handle this (not send unknown actions to old clients, or handle their confusion).

### Adding a New Role

1. Add constant to models/: `const MyRole Role = "MyRole"`
2. Add to Roles slice so the deck knows to include it
3. Update actions that interact with the role (counters, etc.)

**Balance impact**: New roles change the game fundamentally. This is a design decision, not just a code change.

### Modifying Existing Types

**Adding fields**: Safe if the zero value is sensible. Old code will see zero values for new fields.

**Removing fields**: Breaking change. Old code will have compile errors. Old data in databases will have extra fields (harmless in most cases).

**Changing field types**: Breaking change. Requires migration.

**Renaming**: Use new JSON tag, keep old field as deprecated alias. Remove in next major version.

## Anti-Patterns: Code That Signals Trouble

### Anti-Pattern: "Importing Business Logic Packages"

```go
// WRONG: models importing engine
package models
import "ElMakina/backend/engine"
```

**What this means**: Circular import incoming. Models becomes coupled to implementation.

**Fix**: Models must remain a leaf. If engine types are needed in models, the dependency is inverted incorrectly. Move shared types to models, or use interfaces.

### Anti-Pattern: "Methods on Model Types"

```go
// WRONG: logic in models
func (p Player) CanAfford(cost int) bool {
    return p.Coins >= cost
}
```

**What this means**: Models is no longer just vocabulary—it has opinions. Multiple packages may need different "CanAfford" logic (e.g., with discounts, penalties).

**Fix**: Keep models dumb. Put logic in engine/ or actions/.

### Anti-Pattern: "JSON Tags That Lie"

```go
// WRONG: tag doesn't match field name
type PlayerAction struct {
    ID     ActionID `json:"action"`  // field is ID, JSON is action
    Source int      `json:"player"`  // field is Source, JSON is player
}
```

**What this means**: Confusion between Go code and wire format. Debugging requires mental translation.

**Fix**: Match JSON tags to field names where possible. If they must differ, document why clearly.

## Team Culture: Preserving the Language

### Code Review Checklist

When reviewing model changes:
- [ ] Does this belong in models/ (shared) or a specific package?
- [ ] Are field names clear and consistent?
- [ ] Are JSON tags correct and stable?
- [ ] Is the zero value sensible?
- [ ] Does this change break backward compatibility?

### Documentation as Dictionary

Models are self-documenting to some extent (field names), but complex types need explanation:
- Document payload structures
- Document valid values for enums (even though they are strings)
- Document relationships between types

### Onboarding: Learning the Vocabulary

New developers should:
1. Read models/ first—understand the vocabulary
2. See how the same types flow through server → engine → actions
3. Understand that changing models affects everyone

**The rule**: Models changes are high-impact. Treat them with care.

## Summary: The Sacred Vocabulary

The models package is the **sacred vocabulary** of the system. It enables communication between packages. It defines the boundaries of what is possible. It remains stable while implementation evolves.

Key principles:
1. **No logic**: Models are definitions, not behavior
2. **No dependencies**: Models import nothing, everything imports models
3. **Strings over ints**: Readability over micro-optimization
4. **Constants for safety**: Use typed constants, not raw strings
5. **Stability**: Change slowly, change carefully

When the vocabulary is clear, implementation is easy. When the vocabulary is muddled, confusion spreads. Preserve the sacred vocabulary. Keep it clean, minimal, and stable.

The Rosetta Stone must remain legible to all.
