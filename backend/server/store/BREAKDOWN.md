# Store Package Breakdown: Persistence as Boundary

## The Core Philosophy: The Boundary Between Memory and Disk

The engine is pure—it exists only in memory, untouched by time, indifferent to crashes. But the real world is messy: servers restart, processes die, hardware fails. The store package exists at the boundary between the engine's timeless purity and the persistent reality of disks and databases.

This package follows a critical architectural principle: **persistence is a server concern, not an engine concern**. The engine produces snapshots; the server decides when and how to save them. The engine knows nothing of files or databases. The store knows nothing of game rules. They meet at the interface.

### The Mental Model: The Archivist

Imagine a diligent archivist working in a library. The archivist does not write the books (the engine does that). The archivist does not read the books for content (the server does that). The archivist's sole purpose is to **preserve** and **retrieve**.

When a book is complete, the archivist carefully places it in the archive. When someone needs a book, the archivist retrieves it from the archive. The archivist ensures the book is safe from fire, flood, and time. But the archivist never edits the book's content.

**The store is this archivist.**

- **LobbySnapshot** is the book to be preserved
- **Save()** places it in the archive
- **Load()** retrieves it from the archive
- The archivist ensures atomicity, durability, and consistency

**Why this matters**: The separation of concerns keeps the engine pure. The engine just produces state. Whether that state lives in memory for 5 minutes or on disk for 5 years is not the engine's concern. The store handles reality so the engine can remain timeless.

## The Interface: Contract Over Implementation

```go
type LobbyStore interface {
    Load(ctx context.Context) (*LobbySnapshot, error)
    Save(ctx context.Context, snapshot LobbySnapshot) error
}
```

This tiny interface is the entire contract. It says:
- You can save a snapshot
- You can load it back
- Operations respect context (cancellation, timeouts)
- Errors are explicit

**Why an interface?** Because implementations may vary:
- **FileStore**: JSON on local disk (development, single-node)
- **RedisStore**: In-memory with persistence (production, distributed)
- **PostgresStore**: Relational database (enterprise, analytics)
- **MockStore**: In-memory only, for testing

The interface lets us swap implementations without changing callers. The server uses `LobbyStore`; it does not know or care which implementation.

## The FileStore: Simplicity as Virtue

The FileStore is the reference implementation—simple, clear, sufficient for many deployments.

### The Storage Format: JSON

Snapshots are serialized as JSON. Human-readable, debuggable, portable.

```json
{
  "lobbies": {
    "lobby_abc123": {
      "id": "lobby_abc123",
      "leader_id": "player_1",
      "players": [...],
      "status": "open"
    }
  },
  "players": {
    "player_1": {
      "id": "player_1",
      "nickname": "Alice",
      "token": "..."
    }
  },
  "by_token": {
    "token_xyz789": "player_1"
  },
  "next_lobby_id": 42
}
```

**Why JSON**: 
- Human readable (debug with `cat` or `jq`)
- Universal support (every language can parse it)
- Self-describing (field names included)
- Good enough performance for our scale

**Trade-off**: JSON is larger than binary formats (msgpack, protobuf). For lobby state (kilobytes, not megabytes), this does not matter.

### Atomic Writes: Safety in Simplicity

The FileStore writes atomically using the "write to temp, then rename" pattern:

```go
tmp := s.Path + ".tmp"
if err := os.WriteFile(tmp, data, 0o644); err != nil {
    return err
}
return os.Rename(tmp, s.Path)
```

**Why this matters**:
- If the process crashes mid-write, the temp file is incomplete, original file is untouched
- Readers never see partial writes
- Rename is atomic on POSIX systems (and Windows with caveats)
- No locks needed for readers (they see consistent state)

**The guarantee**: At any moment, the file is either the old snapshot or the new snapshot, never a corrupted mix.

### Concurrency: Mutex Over Complexity

```go
type FileStore struct {
    Path string
    mu   sync.Mutex
}
```

A simple mutex protects the file. Only one Save() at a time.

**Why not more complex?**:
- Lobby saves are infrequent (lobby changes are user-paced, not high-frequency)
- Read-Write locks would add complexity for minimal gain
- If contention becomes an issue, switch to a different implementation (Redis), not more complex locking

**The pattern**: Start simple. Optimize when measured.

## Common Pitfalls: When Persistence Fails

### Pitfall 1: "I'll Just Write Directly to the File"

**The mistake**:
```go
// WRONG: non-atomic, readers see partial writes
func (s *FileStore) Save(data []byte) error {
    return os.WriteFile(s.Path, data, 0o644)
}
```

**Why it destroys everything**:
- Crash during write leaves corrupted file
- Concurrent readers see partial JSON (parse errors)
- Data loss on power failure

**The right way**: Atomic rename pattern. Write temp, validate, then rename.

### Pitfall 2: "Ignoring Context Cancellation"

**The mistake**:
```go
// WRONG: does not respect timeout
func (s *FileStore) Save(ctx context.Context, snapshot LobbySnapshot) error {
    // ignores ctx entirely
    return s.saveInternal(snapshot)
}
```

**Why it hurts**:
- Server shutdown waits forever
- Timeouts do not work
- Resource leaks

**The right way**: Respect context even if implementation does not use it:
```go
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // proceed
}
```

(Note: FileStore currently ignores context—acceptable for local disk, but interface demands it for future implementations.)

### Pitfall 3: "No Validation on Load"

**The mistake**:
```go
// WRONG: trusts file contents
func (s *FileStore) Load() (*LobbySnapshot, error) {
    data, _ := os.ReadFile(s.Path)
    var snap LobbySnapshot
    json.Unmarshal(data, &snap)
    return &snap, nil  // Returns whatever garbage was in the file
}
```

**Why it destroys everything**:
- Corrupted files produce corrupted state
- Missing fields cause nil pointer panics later
- Silent data corruption

**The right way**: Validate after load:
```go
if err := json.Unmarshal(data, &snapshot); err != nil {
    return nil, fmt.Errorf("corrupted snapshot: %w", err)
}
// Validate required fields
if snapshot.Lobbies == nil {
    snapshot.Lobbies = make(map[string]Lobby)
}
```

### Pitfall 4: "Storing Engine State in LobbyStore"

**The mistake**: Trying to save GameState via LobbyStore.

**Why it is wrong**:
- LobbyStore and engine state are different concerns
- Different lifetimes (lobbies persist between games)
- Different storage needs (engine state has replay package)

**The right way**: Engine state belongs in the replay package's storage. Lobby state belongs here. Do not conflate them.

## Design Trade-offs: Why We Chose This Way

### Trade-off: Interface vs. Concrete Type

**Why we chose interface**:
- Swappable implementations
- Testability (mock store)
- Future flexibility (Redis, Postgres)

**What we gave up**:
- Cannot see implementation in IDE
- Slightly more complex (interface definition)
- Potential for nil interface panics

**Why this is correct**: Storage is infrastructure. Infrastructure varies by environment. Interface enables flexibility.

### Trade-off: File vs. Database

**Why we chose file as default**:
- Zero dependencies (no Redis, no Postgres)
- Works everywhere (dev, test, small production)
- Human readable (debug with text editor)
- Simple to backup (copy file)

**What we gave up**:
- Concurrent access from multiple servers
- Query capabilities (find all lobbies with status "open")
- Automatic replication

**Mitigation**: Interface allows swapping to database when needed. File is the default, not the only option.

### Trade-off: JSON vs. Binary

**Why we chose JSON**:
- Human readable
- Easy to debug
- Language agnostic
- Schema evolution (new fields do not break old code)

**What we gave up**:
- Size (JSON is larger than binary)
- Speed (parsing JSON is slower than msgpack)
- No schema enforcement (can have typos in field names)

**Why this is correct**: Lobby state is small. Readability is worth more than the microsecond parsing difference.

## Performance Considerations: Persistence at Scale

### Write Frequency: The Real Bottleneck

Lobby saves happen when:
- Player joins lobby
- Player leaves lobby
- Lobby leader changes
- Game starts (lobby → game transition)

These are **user-paced events** (seconds apart, not milliseconds). Write throughput is not a concern.

### File Size: Negligible

LobbySnapshot size:
- 10 lobbies × 100 bytes each = 1KB
- 100 players × 200 bytes each = 20KB
- Total: ~21KB

Writing 21KB to SSD: ~0.1ms. Negligible.

### Concurrency: Single-Node Only

FileStore works only when:
- Single server process
- No horizontal scaling
- Shared nothing architecture

**For multi-node**: Use RedisStore or similar. The interface supports this; just swap implementations.

## Evolution: Extending Storage

### Adding a Redis Implementation

1. Create `store/redis.go`
2. Implement `LobbyStore` interface
3. Use Redis hashes for lobbies, sets for players
4. Handle serialization (JSON or Redis hashes)
5. Wire up in server initialization based on config

**Benefits**: Works across multiple servers, faster than file, TTL support for automatic cleanup.

### Adding Schema Versioning

If LobbySnapshot format changes:

1. Add `Version int` field
2. On Load(), check version
3. If old version, migrate:
   ```go
   if snapshot.Version == 1 {
       snapshot = migrateV1ToV2(snapshot)
   }
   ```
4. Always save as latest version

**Why**: Backward compatibility. Old snapshots load and upgrade automatically.

### Encryption at Rest

For production security:

```go
type EncryptedStore struct {
    Inner  LobbyStore
    Cipher encryption.Cipher
}

func (e *EncryptedStore) Save(ctx context.Context, snapshot LobbySnapshot) error {
    encrypted := e.Cipher.Encrypt(snapshot)
    return e.Inner.Save(ctx, encrypted)
}
```

Wrap any store with encryption. Interface enables this composition.

## Anti-Patterns: Code That Signals Trouble

### Anti-Pattern: "Business Logic in Store"

```go
// WRONG: store deciding what to save
func (s *FileStore) SaveIfNotFull(lobby Lobby) error {
    if len(lobby.Players) < 6 {
        return s.Save(lobby)
    }
    return nil
}
```

**What this means**: Store is making decisions. Violates single responsibility.

**Fix**: Store saves what it is given. Caller decides when and what to save.

### Anti-Pattern: "Silent Failures"

```go
// WRONG: ignoring errors
func (s *FileStore) Save(snapshot LobbySnapshot) {
    data, _ := json.Marshal(snapshot)  // ignore error
    os.WriteFile(s.Path, data, 0o644)  // ignore error
}
```

**What this means**: Store pretends to work even when it fails. Data loss happens silently.

**Fix**: Return all errors. Let caller decide how to handle (log, retry, alert).

### Anti-Pattern: "Store as Cache"

```go
// WRONG: store being used for caching
func (s *FileStore) GetLobby(id string) (*Lobby, error) {
    // reads from file every time!
}
```

**What this means**: Store is being used for hot-path reads. File I/O on every request.

**Fix**: Server keeps lobbies in memory. Store is for durability, not runtime access. Load on startup, save on change.

## Team Culture: Respecting the Boundary

### Code Review Questions

When reviewing store changes:
- [ ] Does this respect the interface contract?
- [ ] Are errors handled, not swallowed?
- [ ] Is context cancellation respected?
- [ ] Are writes atomic?
- [ ] Is corruption handled gracefully?
- [ ] Does this belong in store/ or elsewhere?

### Testing Storage

Stores are tested with:
1. Unit tests: Mock time, filesystem
2. Integration tests: Real temp files
3. Concurrency tests: Parallel saves/loads

**Pattern**: Table-driven tests with golden files (expected JSON output).

### Onboarding: Understanding Persistence

New developers should understand:
1. Store is a boundary, not business logic
2. The interface enables swappable implementations
3. Atomic writes prevent corruption
4. Context matters for future distributed implementations

## Summary: The Diligent Archivist

The store package is the **diligent archivist** of the system. It does not write the content (engine does). It does not interpret the content (server does). It preserves the content faithfully, durably, safely.

Key principles:
1. **Interface over implementation**: Swappable, testable, flexible
2. **Atomic writes**: No corruption, no partial states
3. **Simple first**: File is sufficient until proven otherwise
4. **Context awareness**: Interface supports future distributed needs
5. **Error handling**: Failures are explicit, not silent

The store handles the messy reality of persistence so the engine can remain pure. It is the bridge between memory and disk, between the timeless and the durable.

The archivist preserves the record for future generations.
