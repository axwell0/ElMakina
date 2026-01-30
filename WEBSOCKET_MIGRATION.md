# WebSocket Server Migration Guide

## Overview

This document outlines the migration path for updating the WebSocket server to use the new repository-based architecture. The WebSocket server currently uses the old `server.Player` type and method signatures that don't match the new domain-driven design.

## Current State

### What's Been Completed ✅

1. **Domain Layer** (`backend/domain/`)
   - `entities.Player` - Domain entity with validation
   - `entities.Lobby` - Domain entity with business rules
   - `services.PlayerService` - Player lifecycle management
   - `repositories.PlayerRepository` - Data access interface
   - `repositories.LobbyRepository` - Data access interface
   - `repositories.UnitOfWork` - Transaction boundaries

2. **Infrastructure Layer** (`backend/infrastructure/persistence/`)
   - PostgreSQL repository implementations
   - GORM models and mappers
   - Transaction management
   - Migration support (needs switch to goose)

3. **Application Layer** (`backend/server/`)
   - `LobbyManager` - Uses PlayerService and repositories
   - Removed old JSON blob storage

### What's Broken ❌

The WebSocket server (`backend/server/ws/server.go`) has compilation errors:

```
backend/server/ws/server.go:51:22: undefined: server.Player
backend/server/ws/server.go:62:39: not enough arguments in call to server.NewLobbyManager
backend/server/ws/server.go:116:21: undefined: server.Player
backend/server/ws/server.go:118:43: not enough arguments in call to s.manager.ReconnectPlayer
backend/server/ws/server.go:129:42: not enough arguments in call to s.manager.RegisterPlayer
backend/server/ws/server.go:194:17: s.manager.RemovePlayer undefined
```

## Migration Plan

### Phase 1: Update Type References

**File:** `backend/server/ws/server.go`

#### 1.1 Update clientConn struct

**Current:**
```go
type clientConn struct {
    player      *server.Player  // ❌ Old type
    conn        wsConn
    sendMu      sync.Mutex
    lobbyID     string
    session     *sessionRunner
    playerIndex int
}
```

**New:**
```go
type clientConn struct {
    playerID    entities.PlayerID  // ✅ Use ID instead of full object
    playerNick  string             // Cache nick for display
    conn        wsConn
    sendMu      sync.Mutex
    lobbyID     entities.LobbyID   // ✅ Type-safe lobby ID
    session     *sessionRunner
    playerIndex int
}
```

#### 1.2 Update NewServer function

**Current:**
```go
func NewServer(manager *server.LobbyManager, clock engine.Clock, cfg engine.TurnConfig, grace time.Duration) *Server {
    if manager == nil {
        manager = server.NewLobbyManager(2, 9)  // ❌ Old signature
    }
    // ...
}
```

**New:**
```go
func NewServer(manager *server.LobbyManager, clock engine.Clock, cfg engine.TurnConfig, grace time.Duration) *Server {
    // Manager is now required - no default creation
    if manager == nil {
        panic("LobbyManager is required")
    }
    // ...
}
```

### Phase 2: Update Method Calls

#### 2.1 Player Registration

**Current:**
```go
player, err := s.manager.RegisterPlayer(nick)
```

**New:**
```go
ctx := context.Background()
player, err := s.manager.RegisterPlayer(ctx, nick)
if err != nil {
    return fmt.Errorf("registration failed: %w", err)
}
// Store player ID, not full object
client.playerID = player.ID()
client.playerNick = player.Nick()
```

#### 2.2 Player Reconnection

**Current:**
```go
player, err := s.manager.ReconnectPlayer(token)
```

**New:**
```go
ctx := context.Background()
player, err := s.manager.ReconnectPlayer(ctx, token)
if err != nil {
    return fmt.Errorf("reconnection failed: %w", err)
}
client.playerID = player.ID()
client.playerNick = player.Nick()
```

#### 2.3 Avatar Update

**Current:**
```go
err := s.manager.SetPlayerAvatar(playerID, avatar)
```

**New:**
```go
ctx := context.Background()
playerID := entities.PlayerID(client.playerID)  // Convert string to type
err := s.manager.SetPlayerAvatar(ctx, playerID, avatar)
```

#### 2.4 Lobby Creation

**Current:**
```go
lobby, err := s.manager.CreateLobby(playerID)
```

**New:**
```go
ctx := context.Background()
leaderID := entities.PlayerID(client.playerID)
lobby, err := s.manager.CreateLobby(ctx, leaderID)
if err != nil {
    return fmt.Errorf("lobby creation failed: %w", err)
}
client.lobbyID = lobby.ID()  // Now entities.LobbyID
```

#### 2.5 Join Lobby

**Current:**
```go
err := s.manager.JoinLobby(lobbyID, playerID)
```

**New:**
```go
ctx := context.Background()
lobbyID := entities.LobbyID(lobbyIDStr)
playerID := entities.PlayerID(client.playerID)
err := s.manager.JoinLobby(ctx, lobbyID, playerID)
```

#### 2.6 Leave Lobby

**Current:**
```go
err := s.manager.RemovePlayer(lobbyID, playerID)  // ❌ Method doesn't exist
```

**New:**
```go
ctx := context.Background()
lobbyID := entities.LobbyID(client.lobbyID)
playerID := entities.PlayerID(client.playerID)
shouldClose, err := s.manager.LeaveLobby(ctx, lobbyID, playerID)
if shouldClose {
    // Handle lobby closure
}
```

### Phase 3: Context Propagation

All manager methods now require a `context.Context` parameter. You need to:

1. **Add context to message handlers:**
```go
func (s *Server) handleMessage(ctx context.Context, client *clientConn, msg Envelope) error {
    // Pass ctx to all manager calls
}
```

2. **Create context from HTTP request:**
```go
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // Use request context
    // ...
}
```

3. **Add timeouts:**
```go
ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
defer cancel()
```

### Phase 4: Error Handling

Update error handling to use domain errors:

```go
player, err := s.manager.RegisterPlayer(ctx, nick)
if err != nil {
    switch {
    case errors.Is(err, entities.ErrNicknameEmpty):
        return fmt.Errorf("nickname cannot be empty")
    case errors.Is(err, entities.ErrNicknameTooLong):
        return fmt.Errorf("nickname too long (max 32 characters)")
    case errors.Is(err, entities.ErrPlayerAlreadyInLobby):
        return fmt.Errorf("player already in a lobby")
    default:
        return fmt.Errorf("registration failed: %w", err)
    }
}
```

### Phase 5: Update Message Types

Update WebSocket message types to use new ID types:

**Current:**
```go
type JoinLobbyMessage struct {
    LobbyID string `json:"lobbyId"`
}
```

**New:**
```go
type JoinLobbyMessage struct {
    LobbyID string `json:"lobbyId"`  // Keep as string for JSON
}

// Convert when using:
lobbyID := entities.LobbyID(msg.LobbyID)
```

### Phase 6: Testing

1. **Update unit tests:**
   - Mock PlayerService instead of old LobbyManager
   - Use context in all test calls
   - Test error cases with domain errors

2. **Integration tests:**
   - Test full player lifecycle
   - Test lobby operations
   - Test concurrent operations

3. **Manual testing:**
   - Player registration
   - Reconnection with token
   - Avatar upload
   - Lobby creation/joining/leaving
   - Game start

## Implementation Checklist

### Files to Modify

- [ ] `backend/server/ws/server.go` - Main server logic
- [ ] `backend/server/ws/messages.go` - Message types (if ID fields need updating)
- [ ] `backend/server/ws/server_test.go` - Update tests
- [ ] `backend/server/ws/config.go` - Already updated ✅

### Type Conversions Needed

| Old | New | Conversion |
|-----|-----|------------|
| `string` (player ID) | `entities.PlayerID` | `entities.PlayerID(idStr)` |
| `string` (lobby ID) | `entities.LobbyID` | `entities.LobbyID(idStr)` |
| `*server.Player` | `entities.PlayerID` | Store ID, fetch when needed |
| `server.Player` | `entities.Player` | Use domain entity |

### Method Signature Changes

| Method | Old Signature | New Signature |
|--------|--------------|---------------|
| `RegisterPlayer` | `(nick string) (*Player, error)` | `(ctx context.Context, nick string) (*entities.Player, error)` |
| `ReconnectPlayer` | `(token string) (*Player, error)` | `(ctx context.Context, token string) (*entities.Player, error)` |
| `SetPlayerAvatar` | `(playerID, avatar string) error` | `(ctx context.Context, playerID entities.PlayerID, avatar string) error` |
| `UnregisterPlayer` | `(playerID string) error` | `(ctx context.Context, playerID entities.PlayerID) error` |
| `CreateLobby` | `(leaderID string) (*Lobby, error)` | `(ctx context.Context, leaderID entities.PlayerID) (*entities.Lobby, error)` |
| `JoinLobby` | `(lobbyID, playerID string) error` | `(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) error` |
| `LeaveLobby` | N/A (was RemovePlayer) | `(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) (bool, error)` |
| `StartLobby` | `(lobbyID, leaderID string) error` | `(ctx context.Context, lobbyID entities.LobbyID, leaderID entities.PlayerID) error` |

## Migration Strategy

### Option 1: Big Bang (Recommended for this case)

Update all WebSocket code in one commit. Since the server is the only consumer of the lobby manager, this is feasible.

**Pros:**
- Atomic change
- No intermediate state
- Easier to test

**Cons:**
- Large commit
- Higher risk

### Option 2: Adapter Pattern (If you need gradual migration)

Create an adapter that wraps the new manager with the old interface:

```go
type LobbyManagerAdapter struct {
    inner *server.LobbyManager
}

func (a *LobbyManagerAdapter) RegisterPlayer(nick string) (*server.Player, error) {
    ctx := context.Background()
    player, err := a.inner.RegisterPlayer(ctx, nick)
    if err != nil {
        return nil, err
    }
    // Convert entities.Player to server.Player
    return &server.Player{
        ID: player.ID().String(),
        Nick: player.Nick(),
        Token: player.Token(),
        Avatar: player.Avatar(),
    }, nil
}
```

**Pros:**
- Gradual migration
- Lower risk

**Cons:**
- Temporary complexity
- More code to maintain

## Testing Strategy

### Before Migration

1. Ensure all existing tests pass
2. Document current behavior
3. Create test scenarios for:
   - Player registration flow
   - Reconnection flow
   - Lobby lifecycle
   - Concurrent operations

### During Migration

1. Update tests alongside code
2. Run tests frequently
3. Use table-driven tests for error cases

### After Migration

1. Run full integration tests
2. Manual testing of all flows
3. Load testing for concurrent operations

## Common Pitfalls

### 1. Forgetting Context

**Wrong:**
```go
player, err := s.manager.RegisterPlayer(nick)  // Missing ctx
```

**Right:**
```go
ctx := r.Context()
player, err := s.manager.RegisterPlayer(ctx, nick)
```

### 2. String vs Typed IDs

**Wrong:**
```go
lobbyID := "lobby-123"  // Plain string
err := s.manager.JoinLobby(ctx, lobbyID, playerID)
```

**Right:**
```go
lobbyID := entities.LobbyID("lobby-123")  // Typed
err := s.manager.JoinLobby(ctx, lobbyID, playerID)
```

### 3. Storing Full Objects

**Wrong:**
```go
client.player = player  // Storing *entities.Player
```

**Right:**
```go
client.playerID = player.ID()  // Store ID only
client.playerNick = player.Nick()  // Cache display data
```

### 4. Ignoring Context Cancellation

**Wrong:**
```go
ctx := context.Background()  // Never cancelled
```

**Right:**
```go
ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
defer cancel()
```

## Example: Complete Handler Update

**Before:**
```go
func (s *Server) handleRegister(client *clientConn, msg RegisterMessage) error {
    player, err := s.manager.RegisterPlayer(msg.Nick)
    if err != nil {
        return err
    }
    client.player = player
    return s.send(client, Envelope{Type: "registered", Payload: player})
}
```

**After:**
```go
func (s *Server) handleRegister(ctx context.Context, client *clientConn, msg RegisterMessage) error {
    player, err := s.manager.RegisterPlayer(ctx, msg.Nick)
    if err != nil {
        return fmt.Errorf("registration failed: %w", err)
    }
    
    client.playerID = player.ID()
    client.playerNick = player.Nick()
    
    response := RegisterResponse{
        ID:    player.ID().String(),
        Nick:  player.Nick(),
        Token: player.Token(),
    }
    
    return s.send(client, Envelope{Type: "registered", Payload: response})
}
```

## Timeline Estimate

- **Phase 1-2** (Type updates, method calls): 2-3 hours
- **Phase 3** (Context propagation): 1-2 hours
- **Phase 4** (Error handling): 1 hour
- **Phase 5** (Message types): 30 minutes
- **Phase 6** (Testing): 2-3 hours

**Total: 6-10 hours of focused work**

## Next Steps

1. Create worktree for WebSocket migration
2. Update `clientConn` struct
3. Update all manager method calls
4. Add context propagation
5. Update error handling
6. Run tests
7. Manual testing
8. Merge to experiment branch

## Questions?

If you encounter issues during migration:

1. Check domain entity methods match your usage
2. Ensure context is passed through the entire call chain
3. Verify type conversions at API boundaries
4. Test error cases explicitly

---

**Status:** Ready for implementation  
**Priority:** High (blocks WebSocket functionality)  
**Risk:** Medium (well-defined scope, comprehensive tests exist)
