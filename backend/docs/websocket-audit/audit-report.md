# WebSocket Code Audit Report

**Date:** 2026-01-30  
**Scope:** `backend/server/ws/` and `backend/engine/input/websocket/`  
**Guidelines:** AGENTS.md - Balanced Defensiveness & Locking Best Practices  
**Total Issues Found:** 25

---

## Severity Summary

| Severity | Count | Description |
|----------|-------|-------------|
| 🔴 **Critical** | 4 | Locks held during I/O operations |
| 🟡 **Moderate** | 3 | Defensive copies of non-escaping data |
| 🟢 **Minor** | 15 | Over-defensive nil checks in trusted boundaries |
| 💡 **Style** | 3 | Unnecessary defer usage |

---

## 🔴 Critical Issues: Locks Held During I/O

### Issue 1: `broadcastLobbyState` holds lock during WebSocket writes

**File:** `backend/server/ws/server.go`  
**Lines:** 474-483

**Original Code:**
```go
s.mu.Lock()
defer s.mu.Unlock()
for _, id := range lobby.PlayerIDs {
    client := s.clients[id]
    if client == nil {
        continue
    }
    _ = s.send(client, Envelope{Type: MsgLobbyState, Payload: payload})  // I/O inside lock!
}
```

**Violation:** Holds `s.mu` lock while calling `s.send()` which performs WebSocket writes (`WriteJSON`). This blocks all other operations on the server (client registration, session attachment) while network I/O completes. Under high load or with slow clients, this creates a bottleneck and potential cascading delays.

**Solution:**
```go
// Collect clients while locked
s.mu.Lock()
clientsToNotify := make([]*clientConn, 0, len(lobby.PlayerIDs))
for _, id := range lobby.PlayerIDs {
    if client := s.clients[id]; client != nil {
        clientsToNotify = append(clientsToNotify, client)
    }
}
s.mu.Unlock()

// Send after releasing lock
for _, client := range clientsToNotify {
    _ = s.send(client, Envelope{Type: MsgLobbyState, Payload: payload})
}
```

---

### Issue 2: `send` method holds lock during I/O with defer

**File:** `backend/server/ws/server.go`  
**Lines:** 360-364

**Original Code:**
```go
func (s *Server) send(client *clientConn, env Envelope) error {
    client.sendMu.Lock()
    defer client.sendMu.Unlock()
    return client.conn.WriteJSON(env)  // I/O while holding lock
}
```

**Violation:** The `sendMu` lock is held during `WriteJSON` (network I/O). Using `defer` extends the lock duration until function return, preventing concurrent sends to the same client and potentially causing delays.

**Solution:**
```go
func (s *Server) send(client *clientConn, env Envelope) error {
    client.sendMu.Lock()
    err := client.conn.WriteJSON(env)
    client.sendMu.Unlock()
    return err
}
```

---

### Issue 3: `broadcast` holds RLock AND sendMu during I/O

**File:** `backend/server/ws/session_provider.go`  
**Lines:** 689-700

**Original Code:**
```go
func (p *sessionProvider) broadcast(env Envelope) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    for _, client := range p.players {
        if client == nil {
            continue
        }
        client.sendMu.Lock()
        _ = client.conn.WriteJSON(env)  // I/O while holding both locks!
        client.sendMu.Unlock()
    }
}
```

**Violation:** This is a **double violation** - it holds the `RWMutex` read lock while iterating AND the `sendMu` during WebSocket writes. This blocks all session state modifications (player joins, disconnects) while broadcasting to all clients. The `defer` on the RLock makes it worse.

**Solution:**
```go
func (p *sessionProvider) broadcast(env Envelope) {
    p.mu.RLock()
    clients := make([]*clientConn, 0, len(p.players))
    for _, client := range p.players {
        if client != nil {
            clients = append(clients, client)
        }
    }
    p.mu.RUnlock()

    for _, client := range clients {
        client.sendMu.Lock()
        _ = client.conn.WriteJSON(env)
        client.sendMu.Unlock()
    }
}
```

---

### Issue 4: `sendTo` holds lock during I/O with defer

**File:** `backend/engine/input/websocket/provider_transport.go`  
**Lines:** 43-53

**Original Code:**
```go
func (p *Provider) sendTo(player int, env Envelope) error {
    p.mu.RLock()
    state := p.conns[player]
    p.mu.RUnlock()
    if state == nil {
        return fmt.Errorf("player %d not connected", player)
    }
    state.sendMu.Lock()
    defer state.sendMu.Unlock()
    return state.conn.WriteJSON(env)  // I/O with defer holding lock
}
```

**Violation:** The `sendMu` is held during `WriteJSON` and `defer` delays unlock until function return. This pattern appears in the critical path of message delivery.

**Solution:**
```go
func (p *Provider) sendTo(player int, env Envelope) error {
    p.mu.RLock()
    state := p.conns[player]
    p.mu.RUnlock()
    if state == nil {
        return fmt.Errorf("player %d not connected", player)
    }
    state.sendMu.Lock()
    err := state.conn.WriteJSON(env)
    state.sendMu.Unlock()
    return err
}
```

---

## 🟡 Moderate Issues: Defensive Copies of Non-Escaping Data

### Issue 5: `setActionWindow` makes unnecessary defensive copy

**File:** `backend/server/ws/session_provider.go`  
**Lines:** 94-98

**Original Code:**
```go
func (p *sessionProvider) setActionWindow(reqID string, actor int, allowed []models.ActionID) {
    p.promptMu.Lock()
    p.action = &actionWindow{requestID: reqID, actor: actor, allowed: append([]models.ActionID{}, allowed...)}
    p.promptMu.Unlock()
}
```

**Violation:** The `append([]models.ActionID{}, allowed...)` creates a defensive copy of the slice. However, the data doesn't escape the provider (it's stored in `p.action` and only read later by the same goroutine). If the caller guarantees not to modify the slice after the call, this copy is unnecessary overhead.

**Solution:**
```go
func (p *sessionProvider) setActionWindow(reqID string, actor int, allowed []models.ActionID) {
    p.promptMu.Lock()
    p.action = &actionWindow{requestID: reqID, actor: actor, allowed: allowed}
    p.promptMu.Unlock()
}
```

---

### Issue 6: `setChallengeWindow` makes defensive copy of challengers

**File:** `backend/server/ws/session_provider.go`  
**Lines:** 108-121

**Original Code:**
```go
func (p *sessionProvider) setChallengeWindow(reqID string, window engine.ChallengeWindow, timeout time.Duration) {
    p.promptMu.Lock()
    p.chal = &challengeWindowState{
        requestID:          reqID,
        actorIndex:         window.ActorIndex,
        actionID:           window.Action.ID,
        claimedRole:        window.ClaimedRole,
        kind:               window.Kind,
        targetIndex:        targetIndexFromAction(window.Action),
        allowedChallengers: append([]int{}, window.AllowedChallengers...),  // Defensive copy
        timeoutMs:          timeout.Milliseconds(),
    }
    p.promptMu.Unlock()
}
```

**Violation:** Same pattern as Issue 5 - defensive copy of `AllowedChallengers` slice that doesn't escape the provider's ownership.

**Solution:**
```go
func (p *sessionProvider) setChallengeWindow(reqID string, window engine.ChallengeWindow, timeout time.Duration) {
    p.promptMu.Lock()
    p.chal = &challengeWindowState{
        requestID:          reqID,
        actorIndex:         window.ActorIndex,
        actionID:           window.Action.ID,
        claimedRole:        window.ClaimedRole,
        kind:               window.Kind,
        targetIndex:        targetIndexFromAction(window.Action),
        allowedChallengers: window.AllowedChallengers,
        timeoutMs:          timeout.Milliseconds(),
    }
    p.promptMu.Unlock()
}
```

---

### Issue 7: `setCounterWindow` makes two defensive copies

**File:** `backend/server/ws/session_provider.go`  
**Lines:** 131-143

**Original Code:**
```go
func (p *sessionProvider) setCounterWindow(reqID string, window engine.CounterWindow, timeout time.Duration) {
    p.promptMu.Lock()
    p.counter = &counterWindowState{
        requestID:      reqID,
        actorIndex:     window.MainAction.SourceIndex,
        actionID:       window.MainAction.ID,
        allowedActions: append([]models.ActionID{}, window.AllowedActions...),  // Defensive copy
        allowedPlayers: append([]int{}, window.AllowedPlayers...),              // Defensive copy
        targetIndex:    targetIndexFromAction(window.MainAction),
        timeoutMs:      timeout.Milliseconds(),
    }
    p.promptMu.Unlock()
}
```

**Violation:** Two defensive copies of slices that don't escape the provider. Creates unnecessary GC pressure.

**Solution:**
```go
func (p *sessionProvider) setCounterWindow(reqID string, window engine.CounterWindow, timeout time.Duration) {
    p.promptMu.Lock()
    p.counter = &counterWindowState{
        requestID:      reqID,
        actorIndex:     window.MainAction.SourceIndex,
        actionID:       window.MainAction.ID,
        allowedActions: window.AllowedActions,
        allowedPlayers: window.AllowedPlayers,
        targetIndex:    targetIndexFromAction(window.MainAction),
        timeoutMs:      timeout.Milliseconds(),
    }
    p.promptMu.Unlock()
}
```

---

## 🟢 Minor Issues: Over-Defensive Nil Checks

These checks add noise and suggest uncertainty about object lifecycle invariants. They should be removed inside trusted boundaries.

### Issue 8: `sendRaw` checks for nil connection

**File:** `backend/server/ws/server.go`  
**Lines:** 366-371

**Original Code:**
```go
func (s *Server) sendRaw(conn wsConn, env Envelope) error {
    if conn == nil {
        return fmt.Errorf("nil connection")
    }
    return conn.WriteJSON(env)
}
```

**Violation:** This is inside a trusted boundary - `sendRaw` is only called from `ServeHTTP` with a connection just upgraded from HTTP. The nil check is defensive programming that suggests uncertainty about the caller's behavior. Inside trusted boundaries, we should rely on proper initialization rather than defensive checks.

**Solution:**
```go
func (s *Server) sendRaw(conn wsConn, env Envelope) error {
    return conn.WriteJSON(env)
}
```

---

### Issues 9-15: `eventRecorder` methods check `r == nil`

**File:** `backend/server/ws/session_runner.go`  
**Lines:** 82-84, 100-105, and similar patterns in other methods

**Original Code (record method):**
```go
func (r *eventRecorder) record(ctx context.Context, eventType string, visibility string, playerID *string, payload any) {
    if r == nil || r.recorder == nil || r.matchID == "" {
        return
    }
    // ... method body
}
```

**Violation:** Checking `r == nil` on a method receiver is defensive programming inside trusted boundaries. If the recorder is properly initialized (which it is in `newSessionRunner`), this check is unnecessary. The nil checks suggest the code doesn't trust its own initialization, which is a code smell. If the receiver is nil, that's a bug that should be caught early, not silently ignored.

**Solution:** Remove all nil checks on `eventRecorder` methods. Trust the initialization in `newSessionRunner`:

```go
func (r *eventRecorder) record(ctx context.Context, eventType string, visibility string, playerID *string, payload any) {
    if r.recorder == nil || r.matchID == "" {
        return
    }
    // ... rest of method
}
```

Similar changes needed for:
- `currentSeq()` (lines 100-105)
- `playerIDForIndex()` (lines 107-113)
- `startMatch()` (lines 115-123)
- `snapshot()` (lines 125-144)
- `endMatch()` (lines 146-164)

---

### Issues 16-18: `sessionRunner` Enqueue methods check `r == nil`

**File:** `backend/server/ws/session_runner.go`  
**Lines:** 303-311, 313-321, 323-331

**Original Code:**
```go
func (r *sessionRunner) EnqueueClientEnvelope(senderIndex int, env Envelope) {
    if r == nil {
        return
    }
    r.msgCh <- sessionMessage{kind: msgClientEnvelope, senderIndex: senderIndex, envelope: env}
}

func (r *sessionRunner) EnqueueOffline(playerID string) {
    if r == nil {
        return
    }
    r.msgCh <- sessionMessage{kind: msgOffline, playerID: playerID}
}

func (r *sessionRunner) EnqueueOnline(playerID string) {
    if r == nil {
        return
    }
    r.msgCh <- sessionMessage{kind: msgOnline, playerID: playerID}
}
```

**Violation:** These nil checks suggest the sessionRunner might not be initialized, but it's created in `attachSession` and stored in the client struct. If it's nil, that's a programming bug that should be caught and fixed, not silently ignored which could mask real issues.

**Solution:**
```go
func (r *sessionRunner) EnqueueClientEnvelope(senderIndex int, env Envelope) {
    r.msgCh <- sessionMessage{kind: msgClientEnvelope, senderIndex: senderIndex, envelope: env}
}

func (r *sessionRunner) EnqueueOffline(playerID string) {
    r.msgCh <- sessionMessage{kind: msgOffline, playerID: playerID}
}

func (r *sessionRunner) EnqueueOnline(playerID string) {
    r.msgCh <- sessionMessage{kind: msgOnline, playerID: playerID}
}
```

---

### Issues 19-22: Broadcast methods check `r == nil`

**File:** `backend/server/ws/session_runner.go`  
**Lines:** 559-564, 948-956, 958-972, 974-977

**Original Code (handleChatMessage):**
```go
func (r *sessionRunner) handleChatMessage(senderIndex int, env Envelope) {
    if r == nil || r.session == nil || r.session.Game == nil {
        return
    }
    // ... method body
}
```

**Violation:** Same pattern - checking if the sessionRunner itself is nil inside a method. This is over-defensive programming. The session check is reasonable, but the receiver check is not.

**Solution:**
```go
func (r *sessionRunner) handleChatMessage(senderIndex int, env Envelope) {
    if r.session == nil || r.session.Game == nil {
        return
    }
    // ... method body
}
```

Similar changes for:
- `broadcastGameState()` (lines 948-956)
- `broadcastHandState()` (lines 958-972)
- `broadcastEliminations()` (lines 974-977)

---

### Issues 23-24: Timer broadcast methods check session fields

**File:** `backend/server/ws/session_runner.go`  
**Lines:** 701-704, 710-713

**Original Code:**
```go
func (r *sessionRunner) broadcastTurnTimerPause() {
    if r.session == nil || r.session.Game == nil {
        return
    }
    // ... method body
}
```

**Violation:** Checking session fields is slightly more reasonable than checking the receiver, but still defensive. These methods are called from trusted contexts where session should exist.

**Note:** These are borderline - if the session can legitimately be nil in some edge cases, keep the check. But don't check `r == nil`.

**Solution:**
```go
func (r *sessionRunner) broadcastTurnTimerPause() {
    if r.session == nil || r.session.Game == nil {
        return
    }
    // ... method body
}
```
(Remove only `r == nil` check if present)

---

## 💡 Style Issues: Unnecessary Defer Usage

### Issue 25: `isPaused` could use atomic instead of mutex

**File:** `backend/server/ws/session_provider.go`  
**Lines:** 749-753

**Original Code:**
```go
func (p *sessionProvider) isPaused() bool {
    p.pauseMu.Lock()
    defer p.pauseMu.Unlock()
    return p.paused
}
```

**Violation:** This is a simple boolean read that could use `atomic.Bool` instead of a mutex, avoiding lock overhead entirely. While not strictly wrong per AGENTS.md (short critical section), it's an opportunity for optimization.

**Solution:** Change the field type and method:

In struct definition:
```go
type sessionProvider struct {
    // ... other fields
    paused atomic.Bool  // Replace bool + pauseMu for this field
    // Remove: pauseMu sync.Mutex
}
```

In method:
```go
func (p *sessionProvider) isPaused() bool {
    return p.paused.Load()
}
```

Also update `Pause()` and `Resume()` methods to use `atomic.Store()`.

---

## Additional Minor Style Issues

### `markKicked` uses defer for simple operation

**File:** `backend/server/ws/session_runner.go`  
**Lines:** 625-633

**Original Code:**
```go
func (r *sessionRunner) markKicked(playerID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    if _, ok := r.kicked[playerID]; ok {
        return true
    }
    r.kicked[playerID] = struct{}{}
    return false
}
```

**Note:** This is a short critical section with no I/O, so `defer` is acceptable per AGENTS.md guidelines. However, explicit unlock would be slightly more efficient. This is optional.

**Optional Solution:**
```go
func (r *sessionRunner) markKicked(playerID string) bool {
    r.mu.Lock()
    _, ok := r.kicked[playerID]
    if !ok {
        r.kicked[playerID] = struct{}{}
    }
    r.mu.Unlock()
    return ok
}
```

---

### `registerWaiter` uses defer

**File:** `backend/engine/input/websocket/provider_transport.go`  
**Lines:** 71-77

**Original Code:**
```go
func (p *Provider) registerWaiter(reqID string) *waiter {
    p.mu.Lock()
    defer p.mu.Unlock()
    w := &waiter{ch: make(chan responseEnvelope, 16)}
    p.pending[reqID] = w
    return w
}
```

**Note:** Short critical section, no I/O. Acceptable per AGENTS.md but explicit unlock would be clearer.

**Optional Solution:**
```go
func (p *Provider) registerWaiter(reqID string) *waiter {
    p.mu.Lock()
    w := &waiter{ch: make(chan responseEnvelope, 16)}
    p.pending[reqID] = w
    p.mu.Unlock()
    return w
}
```

---

## Implementation Priority

1. **Critical (Issues 1-4):** Fix immediately - these affect performance and can cause deadlocks
2. **Moderate (Issues 5-7):** Fix next - reduces GC pressure
3. **Minor (Issues 8-24):** Fix when convenient - improves code clarity
4. **Style (Issue 25):** Optional optimization

---

## Files Modified

- `backend/server/ws/server.go` (Issues 1, 2, 8)
- `backend/server/ws/session_provider.go` (Issues 3, 5, 6, 7, 25)
- `backend/server/ws/session_runner.go` (Issues 9-24)
- `backend/engine/input/websocket/provider_transport.go` (Issues 4, optional)

---

## Verification

After implementing fixes:
1. Run `go test ./...` from `backend/` directory
2. Run `go build ./...` to ensure no compilation errors
3. Check `gofmt -l .` for formatting issues
4. Review with `git diff` to ensure changes are minimal and focused

---

*Report generated following AGENTS.md guidelines for balanced defensiveness and locking best practices.*
