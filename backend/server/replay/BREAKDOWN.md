# Replay Package Breakdown: Time Travel as a Service

## The Core Philosophy: History Must Be Preserved and Replayable

A game is not a fleeting moment—it is a narrative that unfolds over time. Players want to review their victories, analyze their defeats, and share their most dramatic moments. Regulators may need to audit games for fairness. Developers need to debug issues that happened hours ago. All of these require one thing: **the complete, accurate preservation of game history**.

The replay package exists to make time travel possible. It records every event, every state change, every decision. It stores them durably. It serves them back on demand, allowing anyone to relive the game exactly as it happened.

This is not just logging—it is **historical preservation**. The replay system treats each game as a precious artifact to be archived with care.

### The Mental Model: The Great Library of Games

Imagine an infinite library where every game ever played is meticulously documented. Each game has:
- A **catalog entry** (who played, when, what rules)
- A **chronicle of events** (every action, challenge, counter)
- A **series of snapshots** (the game state after each turn)
- **Access controls** (who can see what)

When you want to revisit a game, you visit the library. You request the catalog entry to find the game. You receive the chronicle to understand the narrative. You examine the snapshots to see the precise state at any moment.

**The replay package is this library.**

- **Match** = The catalog entry
- **MatchEvent** = The chronicle entries
- **MatchSnapshot** = The preserved states
- **Visibility** = The access controls (public vs private)

**Why this matters**: Games are social experiences worth preserving. The replay system honors that value by treating history as sacred.

## The Architecture: Recording, Storing, Serving

### The Recorder Interface: Contract for Preservation

```go
type Recorder interface {
    StartMatch(ctx context.Context, input MatchStartInput) error
    RecordEvent(ctx context.Context, input EventInput) error
    Snapshot(ctx context.Context, input SnapshotInput) error
    EndMatch(ctx context.Context, input MatchEndInput) error
    LoadReplay(ctx context.Context, matchID string, viewerPlayerID string) (*ReplayPayload, error)
}
```

This interface defines the lifecycle:
1. **StartMatch**: Create the catalog entry
2. **RecordEvent**: Chronicle what happens
3. **Snapshot**: Preserve periodic states
4. **EndMatch**: Mark completion
5. **LoadReplay**: Serve the history back

**Why an interface?** Implementations vary:
- **PostgresStore**: Durable, queryable, scalable
- **FileStore**: Simple, local, good for development
- **HybridStore**: Events in Postgres, snapshots in S3
- **NoOpStore**: No recording (for testing, privacy modes)

The interface lets the server use recording without caring how it works.

### The Data Model: Structured History

#### Match: The Catalog Entry

```go
type Match struct {
    ID              string    // Unique game identifier
    LobbyID         string    // Which lobby this came from
    RulesetVersion  string    // Game rules version
    ProtocolVersion string    // API version
    RNGSeed         string    // Random seed for reproducibility
    CreatedAt       time.Time // When game started
    EndedAt         *time.Time // When game ended (nil if ongoing)
}
```

**Critical insight**: The RNG seed is stored. This means any game can be **reproduced exactly** by replaying with the same seed and actions. Determinism meets persistence.

#### MatchParticipant: The Cast

```go
type MatchParticipant struct {
    MatchID     string // Which game
    PlayerID    string // Who played
    PlayerIndex int    // Their seat number
    Nick        string // Their display name
    Avatar      string // Their avatar choice
}
```

Participants are recorded separately from the match because:
- Players can change names between games
- We preserve who played in this specific game
- Enables "games by player" queries

#### MatchEvent: The Chronicle

```go
type MatchEvent struct {
    MatchID    string         // Which game
    Seq        int64          // Order in sequence
    Type       string         // Event type (action, challenge, etc.)
    Visibility string         // Who can see this (public/private)
    PlayerID   *string        // Who triggered this (optional)
    Payload    datatypes.JSON // Event details (flexible schema)
}
```

**Why JSON for payload?** Events vary:
- Action events have action details
- Challenge events have challenger and outcome
- System events have no player

JSON provides flexibility without schema migrations for every new event type.

**Visibility matters**: Not all events are public. A player's private cards should not appear in the public replay. The visibility field enables filtering.

#### MatchSnapshot: The Preserved State

```go
type MatchSnapshot struct {
    MatchID string         // Which game
    Seq     int64          // When in the sequence
    Payload datatypes.JSON // Serialized GameState
}
```

Snapshots are periodic captures of the complete game state. They enable:
- **Reconstruction**: Build game state at any point without replaying from start
- **Debugging**: See exactly what the game looked like when a bug occurred
- **Verification**: Check that event replay produces correct states

**How many snapshots?** One per turn is typical. A 20-turn game has 20 snapshots. Storage is cheap; completeness is valuable.

### Recording Flow: Capture in Motion

When a game runs, the session runner calls the recorder:

```
Game Start
    ↓
StartMatch() - Create catalog entry
    ↓
Turn 1
    ↓
RecordEvent(action) - Player played Income
    ↓
Snapshot(state) - State after turn 1
    ↓
Turn 2
    ↓
RecordEvent(action) - Player played Steal
RecordEvent(challenge) - Player 1 challenged
RecordEvent(challenge_resolved) - Challenge failed
    ↓
Snapshot(state) - State after turn 2
    ↓
...
    ↓
Game End
    ↓
EndMatch() - Mark as complete
```

**Performance**: Recording is asynchronous. The session runner does not wait for database writes. Events are buffered and flushed.

### Loading and Masking: Privacy as First-Class

When loading a replay, we must respect privacy:

```go
func (s *Store) LoadReplay(ctx context.Context, matchID string, viewerPlayerID string) (*ReplayPayload, error) {
    // Load match and participants (public)
    // Load events filtered by visibility:
    //   - If viewer is participant: show public + their private events
    //   - If viewer is spectator: show only public events
    // Load snapshots and mask private information:
    //   - Mask other players' hands
    //   - Show only what viewer could see at that moment
}
```

**The maskGameState function** creates a view of state appropriate for the viewer:
- Viewer sees their own hand clearly
- Viewer sees other players' hands as empty (hidden)
- Public information (coins, visible cards) is shown

**Why this matters**: Replays respect information asymmetry. You cannot use replays to see what other players held. The replay shows you only what you could have known during the game.

## Common Pitfalls: When History Fails

### Pitfall 1: "Forgetting the RNG Seed"

**The mistake**: Recording events but not the random seed.

**Why it destroys everything**:
- Cannot reproduce game exactly
- Debugging requires guesswork
- Replays may differ from original

**The right way**: Store the RNG seed in Match. Use the same seed to reproduce exact card draws.

### Pitfall 2: "Recording After the Fact"

**The mistake**: Buffering events in memory, only writing to database at game end.

**Why it destroys everything**:
- Crash loses entire game history
- Cannot view in-progress games
- Memory pressure for long games

**The right way**: Record incrementally. Write events as they happen. Use transactions for atomicity, not delayed writes.

### Pitfall 3: "Inconsistent Sequences"

**The mistake**: Events without sequence numbers, or with unreliable ordering.

**Why it destroys everything**:
- Replay shows events out of order
- Debugging based on incorrect timeline
- State reconstruction fails

**The right way**: Strict monotonic sequence numbers (Seq). Database index on (match_id, seq). Events ordered by sequence, not timestamp.

### Pitfall 4: "Over-Recording"

**The mistake**: Recording every possible detail, creating massive records.

**Example**: Recording full GameState for every event (not just snapshots).

**Why it hurts**:
- Database bloat
- Slow queries
- Expensive storage

**The right way**: 
- Events = what happened (compact)
- Snapshots = periodic full states
- Balance: enough to reconstruct, not so much to overwhelm

### Pitfall 5: "No Retention Policy"

**The mistake**: Keeping all replays forever.

**Why it hurts**:
- Infinite storage growth
- Performance degradation over time
- Legal/compliance issues (GDPR right to be forgotten)

**The right way**: Implement retention:
- Auto-delete after N days
- Mark for deletion when lobby deleted
- Anonymize rather than delete (remove PII, keep statistics)

## Design Trade-offs: Why We Chose This Way

### Trade-off: JSON vs. Structured Columns

**Why we chose JSON for event payloads**:
- Flexible schema (different event types have different fields)
- No migrations for new event types
- Easy to add fields without breaking old records

**What we gave up**:
- Cannot query inside payload ("find all games where Player 0 Coup'd Player 1")
- No type safety at database level
- Storage overhead of JSON text

**Mitigation**: Index on (match_id, type) to find specific event types. Use application-level filtering for payload content.

### Trade-off: Separate Tables vs. Single Table

**Why we chose separate tables** (Match, MatchParticipant, MatchEvent, MatchSnapshot):
- Normalized schema (no duplication)
- Efficient queries (load only what you need)
- Type safety per table

**What we gave up**:
- Joins required for full replay
- More complex queries
- Transaction needed to create complete record

**Why this is correct**: Replays are read-heavy, write-once. Normalization optimizes for reading specific parts.

### Trade-off: Database vs. File Storage

**Why we chose Postgres as default**:
- ACID guarantees
- Concurrent access
- Query capabilities
- Operational maturity

**What we gave up**:
- Operational complexity (need Postgres)
- Cost (hosted database vs local disk)
- Speed (network round-trip per write)

**Mitigation**: Async writes buffer the latency. For high throughput, batch inserts.

### Trade-off: Real-time vs. Post-Game

**Why we chose post-game replays**:
- Simpler architecture
- Complete data available
- No live streaming complexity

**What we gave up**:
- Live spectator mode
- Real-time debugging
- "Watch live games" feature

**Future**: Could add live events (WebSocket broadcast) separate from replay recording.

## Performance Considerations: Scale of History

### Write Volume: Event-Driven

Typical game:
- 20 turns
- 3 events per turn (action, maybe challenge, resolution)
- 60 events per game
- 6 snapshots per game (one every few turns)

**Rate**: 1000 concurrent games = 60,000 events/minute = 1,000 events/second.

**Postgres can handle this easily.** With batching, even easier.

### Storage Size: Compact History

Per-game estimates:
- Match record: ~200 bytes
- Participants: ~500 bytes (6 players)
- Events: ~60 × 200 bytes = 12KB
- Snapshots: ~6 × 2KB = 12KB
- **Total**: ~25KB per game

**1 million games**: 25GB. Trivial for modern storage.

### Query Patterns: Replay Loading

Primary query: Load all data for one match:
```sql
SELECT * FROM matches WHERE id = ?
SELECT * FROM participants WHERE match_id = ? ORDER BY player_index
SELECT * FROM events WHERE match_id = ? AND (visibility = 'public' OR player_id = ?) ORDER BY seq
SELECT * FROM snapshots WHERE match_id = ? ORDER BY seq
```

**Indexed properly**: All queries use primary key or indexed foreign keys. Sub-millisecond.

### Hot Path: Recording

Events must record without blocking gameplay:

**Pattern**: Async with backpressure
```go
select {
case recorder.eventChan <- event:
    // buffered, will flush
default:
    // buffer full, drop event or block briefly
    // but never block game loop
}
```

**Alternative**: Synchronous with timeout (context.Background() with short deadline).

## Evolution: Extending the Archive

### Adding Event Search

To find games with specific actions:

1. Add GIN index on event payload (Postgres JSONB)
2. Query: `payload @> '{"action": "steal"}'`
3. Or: Add separate `action_id` column for fast filtering

### Adding Statistics

Aggregate data without scanning all events:

1. Create materialized view of match statistics
2. Refresh periodically
3. Query statistics table for dashboards

### Adding Export

Allow players to download replays:

```go
func (s *Store) ExportReplay(matchID string, format string) ([]byte, error) {
    payload, _ := s.LoadReplay(ctx, matchID, "")
    switch format {
    case "json":
        return json.Marshal(payload)
    case "video":
        return renderToVideo(payload)  // future
    }
}
```

### GDPR Compliance: Right to Erasure

Implement anonymization:

```go
func (s *Store) AnonymizePlayer(ctx context.Context, playerID string) error {
    // Replace nicks with "Player 1", "Player 2"
    // Remove player IDs from events
    // Keep game statistics, remove PII
}
```

Anonymize rather than delete to preserve aggregate statistics.

## Anti-Patterns: Code That Signals Trouble

### Anti-Pattern: "Recording Only Outcomes"

```go
// WRONG: only storing winner
func (s *Store) SaveGameResult(matchID string, winner int) {
    db.Exec("INSERT INTO results ...")
}
```

**What this means**: Cannot reconstruct what happened. No debugging possible.

**Fix**: Record events and snapshots, not just outcomes.

### Anti-Pattern: "Tight Coupling to Game Logic"

```go
// WRONG: recorder knows about specific actions
func (s *Store) RecordStealEvent(...) {
    // special handling for steal
}
```

**What this means**: Adding new actions requires recorder changes. Violates boundary.

**Fix**: Generic RecordEvent with type and JSON payload. Game logic stays in engine.

### Anti-Pattern: "Synchronous Recording on Hot Path"

```go
// WRONG: blocking game loop
func (s *Session) OnAction(action Action) {
    s.recorder.RecordEvent(ctx, event)  // blocks on DB write!
    s.engine.Process(action)
}
```

**What this means**: Database latency stalls gameplay. Unacceptable.

**Fix**: Async recording. Game loop continues; recording happens in background.

## Team Culture: Respecting History

### Code Review Questions

When reviewing replay changes:
- [ ] Are all events recorded (nothing lost)?
- [ ] Is sequence ordering maintained?
- [ ] Is privacy handled (masking, visibility)?
- [ ] Are errors handled without blocking gameplay?
- [ ] Is retention/compliance considered?

### Testing Recording

Test with:
1. Unit tests: Mock recorder, verify calls
2. Integration tests: Real database, verify data written correctly
3. Load tests: High event rate, verify no drops
4. Privacy tests: Verify masking works, private events stay private

### Onboarding: Understanding Archives

New developers should understand:
1. Replays are for players, debugging, and auditing
2. Privacy is paramount—mask correctly
3. Performance matters—do not block gameplay
4. History is forever—design for retention

## Summary: The Library of Experience

The replay package is the **library of experience**. It preserves every game as a story worth telling. It enables time travel—revisiting past moments, learning from them, sharing them.

Key principles:
1. **Complete recording**: Events + snapshots + metadata
2. **Privacy first**: Masking, visibility controls
3. **Performance**: Async, non-blocking recording
4. **Durability**: Database persistence with ACID guarantees
5. **Flexibility**: JSON payloads, interface-based implementations

Games are ephemeral experiences made permanent through recording. The replay package honors those experiences by preserving them with care.

The great library holds every story. Each replay is a world to revisit.
