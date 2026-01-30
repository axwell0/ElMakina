# Repository Architecture Design Document

## Phase 1: Clean Architecture for ElMakina Backend

**Version:** 1.0  
**Date:** 2026-01-30  
**Status:** Design Complete  

---

## Executive Summary

This document outlines a production-grade repository architecture for the ElMakina backend, transitioning from a JSON blob persistence anti-pattern to a normalized, transaction-safe schema following Clean Architecture principles.

### Key Goals

1. **Eliminate JSON blob anti-pattern** - Replace `datatypes.JSON` with normalized tables
2. **Add transaction boundaries** - Ensure atomic operations across multiple entities
3. **Establish repository pattern** - Clear separation between domain and persistence
4. **Enable optimistic locking** - Prevent race conditions in concurrent lobby operations
5. **Maintain backward compatibility** - Zero-downtime migration strategy

---

## 1. Directory Structure

```
backend/
├── domain/                          # Pure business logic (no external deps)
│   ├── entities/                    # Domain models (GORM-free)
│   │   ├── player.go
│   │   ├── lobby.go
│   │   └── errors.go
│   ├── repositories/                # Repository interfaces (ports)
│   │   ├── player_repository.go
│   │   ├── lobby_repository.go
│   │   └── unit_of_work.go
│   └── services/                    # Domain services (optional)
│       └── lobby_service.go
│
├── infrastructure/                  # External concerns (adapters)
│   └── persistence/
│       ├── models/                  # GORM models (internal only)
│       │   ├── player_model.go
│       │   ├── lobby_model.go
│       │   └── lobby_player_model.go
│       ├── repositories/            # Repository implementations
│       │   ├── postgres_player_repository.go
│       │   ├── postgres_lobby_repository.go
│       │   └── postgres_unit_of_work.go
│       ├── transactions/            # Transaction management
│       │   └── unit_of_work.go
│       └── mappers/                 # Domain <-> Model conversions
│           ├── player_mapper.go
│           └── lobby_mapper.go
│
├── application/                     # Use cases (optional layer)
│   └── dto/                         # Data transfer objects
│       ├── lobby_dto.go
│       └── player_dto.go
│
└── server/                          # Existing code (adapted)
    ├── lobby.go                     # Refactored to use repositories
    ├── lobby_manager.go             # New: orchestrates domain operations
    └── store.go                     # Deprecated (kept for compatibility)
```

### Package Dependencies (Enforced)

```
┌─────────────────────────────────────────────────────────────┐
│  domain/entities        domain/repositories                 │
│       │                        │                            │
│       │    ◄───────────────────┘                            │
│       │                                                     │
│       ▼                                                     │
│  infrastructure/persistence/                                │
│       ├── models/                                           │
│       ├── mappers/                                          │
│       └── repositories/                                     │
│                                                             │
│  Dependency Rule: Inner layers know nothing of outer layers │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Domain Entities

### 2.1 Player Entity

```go
// File: backend/domain/entities/player.go

package entities

import (
    "context"
    "fmt"
    "time"
)

// PlayerID is a type-safe identifier for players.
type PlayerID string

func (id PlayerID) String() string { return string(id) }

// Player represents a human participant in the game.
// This is a pure domain entity with no persistence concerns.
type Player struct {
    id        PlayerID
    nick      string
    token     string    // Secret reconnection token
    avatar    string
    createdAt time.Time
    updatedAt time.Time
}

// NewPlayer creates a new player with validation.
func NewPlayer(id PlayerID, nick, token string) (*Player, error) {
    if id == "" {
        return nil, fmt.Errorf("player id is required: %w", ErrInvalidEntity)
    }
    if nick == "" {
        return nil, fmt.Errorf("player nick is required: %w", ErrInvalidEntity)
    }
    if token == "" {
        return nil, fmt.Errorf("player token is required: %w", ErrInvalidEntity)
    }
    if len(nick) > 32 {
        return nil, fmt.Errorf("player nick too long (max 32): %w", ErrInvalidEntity)
    }

    now := time.Now().UTC()
    return &Player{
        id:        id,
        nick:      nick,
        token:     token,
        createdAt: now,
        updatedAt: now,
    }, nil
}

// ReconstitutePlayer reconstructs a player from persistence (no validation).
// Use only within repository mappers.
func ReconstitutePlayer(id PlayerID, nick, token, avatar string, createdAt, updatedAt time.Time) *Player {
    return &Player{
        id:        id,
        nick:      nick,
        token:     token,
        avatar:    avatar,
        createdAt: createdAt,
        updatedAt: updatedAt,
    }
}

// ID returns the player's unique identifier.
func (p *Player) ID() PlayerID { return p.id }

// Nick returns the player's display name.
func (p *Player) Nick() string { return p.nick }

// Token returns the player's secret reconnection token.
func (p *Player) Token() string { return p.token }

// Avatar returns the player's avatar identifier.
func (p *Player) Avatar() string { return p.avatar }

// CreatedAt returns the creation timestamp.
func (p *Player) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns the last update timestamp.
func (p *Player) UpdatedAt() time.Time { return p.updatedAt }

// SetAvatar updates the player's avatar.
func (p *Player) SetAvatar(avatar string) {
    p.avatar = avatar
    p.updatedAt = time.Now().UTC()
}

// CanAuthenticate verifies if the provided token matches.
func (p *Player) CanAuthenticate(token string) bool {
    return p.token == token
}
```

### 2.2 Lobby Entity

```go
// File: backend/domain/entities/lobby.go

package entities

import (
    "fmt"
    "time"
)

// LobbyID is a type-safe identifier for lobbies.
type LobbyID string

func (id LobbyID) String() string { return string(id) }

// LobbyStatus represents the lifecycle state of a lobby.
type LobbyStatus string

const (
    LobbyOpen    LobbyStatus = "open"
    LobbyInGame  LobbyStatus = "in_game"
    LobbyClosed  LobbyStatus = "closed"
)

// Valid returns true if the status is a known value.
func (s LobbyStatus) Valid() bool {
    switch s {
    case LobbyOpen, LobbyInGame, LobbyClosed:
        return true
    }
    return false
}

// LobbyPlayer represents a player's membership in a lobby.
type LobbyPlayer struct {
    PlayerID PlayerID
    JoinedAt time.Time
}

// Lobby is a waiting room for players before a game starts.
// It enforces business invariants and maintains consistency.
type Lobby struct {
    id        LobbyID
    leaderID  PlayerID
    players   []LobbyPlayer // Ordered list maintains join order
    status    LobbyStatus
    version   int64         // Optimistic locking version
    createdAt time.Time
    updatedAt time.Time
}

// NewLobby creates a new lobby with the given leader.
func NewLobby(id LobbyID, leaderID PlayerID) (*Lobby, error) {
    if id == "" {
        return nil, fmt.Errorf("lobby id is required: %w", ErrInvalidEntity)
    }
    if leaderID == "" {
        return nil, fmt.Errorf("leader id is required: %w", ErrInvalidEntity)
    }

    now := time.Now().UTC()
    return &Lobby{
        id:        id,
        leaderID:  leaderID,
        players:   []LobbyPlayer{{PlayerID: leaderID, JoinedAt: now}},
        status:    LobbyOpen,
        version:   1,
        createdAt: now,
        updatedAt: now,
    }, nil
}

// ReconstituteLobby reconstructs a lobby from persistence.
// Use only within repository mappers.
func ReconstituteLobby(
    id LobbyID,
    leaderID PlayerID,
    players []LobbyPlayer,
    status LobbyStatus,
    version int64,
    createdAt, updatedAt time.Time,
) *Lobby {
    // Defensive copy of players slice
    playersCopy := make([]LobbyPlayer, len(players))
    copy(playersCopy, players)

    return &Lobby{
        id:        id,
        leaderID:  leaderID,
        players:   playersCopy,
        status:    status,
        version:   version,
        createdAt: createdAt,
        updatedAt: updatedAt,
    }
}

// ID returns the lobby's unique identifier.
func (l *Lobby) ID() LobbyID { return l.id }

// LeaderID returns the current leader's player ID.
func (l *Lobby) LeaderID() PlayerID { return l.leaderID }

// Status returns the lobby's current status.
func (l *Lobby) Status() LobbyStatus { return l.status }

// Version returns the optimistic locking version.
func (l *Lobby) Version() int64 { return l.version }

// CreatedAt returns the creation timestamp.
func (l *Lobby) CreatedAt() time.Time { return l.createdAt }

// UpdatedAt returns the last update timestamp.
func (l *Lobby) UpdatedAt() time.Time { return l.updatedAt }

// PlayerIDs returns a slice of all player IDs in join order.
func (l *Lobby) PlayerIDs() []PlayerID {
    ids := make([]PlayerID, len(l.players))
    for i, p := range l.players {
        ids[i] = p.PlayerID
    }
    return ids
}

// PlayerCount returns the number of players in the lobby.
func (l *Lobby) PlayerCount() int { return len(l.players) }

// HasPlayer returns true if the player is in the lobby.
func (l *Lobby) HasPlayer(playerID PlayerID) bool {
    for _, p := range l.players {
        if p.PlayerID == playerID {
            return true
        }
    }
    return false
}

// IsOpen returns true if the lobby is accepting new players.
func (l *Lobby) IsOpen() bool { return l.status == LobbyOpen }

// IsInGame returns true if the lobby has an active game.
func (l *Lobby) IsInGame() bool { return l.status == LobbyInGame }

// AddPlayer adds a player to the lobby if it's open and not full.
func (l *Lobby) AddPlayer(playerID PlayerID, maxPlayers int) error {
    if !l.IsOpen() {
        return fmt.Errorf("lobby is not open: %w", ErrLobbyNotOpen)
    }
    if l.HasPlayer(playerID) {
        return fmt.Errorf("player already in lobby: %w", ErrPlayerAlreadyInLobby)
    }
    if len(l.players) >= maxPlayers {
        return fmt.Errorf("lobby is full: %w", ErrLobbyFull)
    }

    l.players = append(l.players, LobbyPlayer{
        PlayerID: playerID,
        JoinedAt: time.Now().UTC(),
    })
    l.version++
    l.updatedAt = time.Now().UTC()
    return nil
}

// RemovePlayer removes a player from the lobby.
// If the leader leaves, the next player becomes leader.
// Returns true if the lobby should be closed (empty).
func (l *Lobby) RemovePlayer(playerID PlayerID) (shouldClose bool, err error) {
    if !l.IsOpen() {
        return false, fmt.Errorf("lobby is not open: %w", ErrLobbyNotOpen)
    }

    idx := -1
    for i, p := range l.players {
        if p.PlayerID == playerID {
            idx = i
            break
        }
    }
    if idx == -1 {
        return false, fmt.Errorf("player not in lobby: %w", ErrPlayerNotInLobby)
    }

    // Remove player while preserving order
    l.players = append(l.players[:idx], l.players[idx+1:]...)

    // If lobby is now empty, mark for closure
    if len(l.players) == 0 {
        l.status = LobbyClosed
        l.version++
        l.updatedAt = time.Now().UTC()
        return true, nil
    }

    // If leader left, promote next player
    if l.leaderID == playerID {
        l.leaderID = l.players[0].PlayerID
    }

    l.version++
    l.updatedAt = time.Now().UTC()
    return false, nil
}

// StartGame transitions the lobby to in-game status.
func (l *Lobby) StartGame(minPlayers int) error {
    if !l.IsOpen() {
        return fmt.Errorf("lobby is not open: %w", ErrLobbyNotOpen)
    }
    if len(l.players) < minPlayers {
        return fmt.Errorf("not enough players (need %d, have %d): %w",
            minPlayers, len(l.players), ErrNotEnoughPlayers)
    }

    l.status = LobbyInGame
    l.version++
    l.updatedAt = time.Now().UTC()
    return nil
}

// ResetToOpen transitions an in-game lobby back to open.
// Used after server restart when sessions are lost.
func (l *Lobby) ResetToOpen() error {
    if l.status != LobbyInGame {
        return fmt.Errorf("lobby is not in game: %w", ErrInvalidStatusTransition)
    }

    l.status = LobbyOpen
    l.version++
    l.updatedAt = time.Now().UTC()
    return nil
}

// CanStart returns true if the lobby can be started by the given player.
func (l *Lobby) CanStart(playerID PlayerID, minPlayers int) error {
    if l.leaderID != playerID {
        return fmt.Errorf("only leader can start: %w", ErrNotLeader)
    }
    if !l.IsOpen() {
        return fmt.Errorf("lobby is not open: %w", ErrLobbyNotOpen)
    }
    if len(l.players) < minPlayers {
        return fmt.Errorf("not enough players: %w", ErrNotEnoughPlayers)
    }
    return nil
}
```

### 2.3 Domain Errors

```go
// File: backend/domain/entities/errors.go

package entities

import "errors"

// Domain errors for entity validation and business rules.
var (
    // Entity validation errors
    ErrInvalidEntity = errors.New("invalid entity")

    // Lobby operation errors
    ErrLobbyNotFound          = errors.New("lobby not found")
    ErrLobbyNotOpen           = errors.New("lobby is not open")
    ErrLobbyFull              = errors.New("lobby is full")
    ErrLobbyClosed            = errors.New("lobby is closed")
    ErrPlayerAlreadyInLobby   = errors.New("player already in lobby")
    ErrPlayerNotInLobby       = errors.New("player not in lobby")
    ErrNotEnoughPlayers       = errors.New("not enough players")
    ErrNotLeader              = errors.New("only leader can perform this action")
    ErrInvalidStatusTransition = errors.New("invalid status transition")
    ErrConcurrentModification = errors.New("concurrent modification detected")

    // Player operation errors
    ErrPlayerNotFound = errors.New("player not found")
    ErrInvalidToken   = errors.New("invalid token")
    ErrPlayerInLobby  = errors.New("player is in a lobby")
)

// IsNotFound returns true if the error is a "not found" error.
func IsNotFound(err error) bool {
    if err == nil {
        return false
    }
    return errors.Is(err, ErrLobbyNotFound) || errors.Is(err, ErrPlayerNotFound)
}

// IsConflict returns true if the error is a conflict error.
func IsConflict(err error) bool {
    if err == nil {
        return false
    }
    return errors.Is(err, ErrConcurrentModification) ||
        errors.Is(err, ErrPlayerAlreadyInLobby) ||
        errors.Is(err, ErrLobbyFull)
}
```

---

## 3. Repository Interfaces

### 3.1 Player Repository

```go
// File: backend/domain/repositories/player_repository.go

package repositories

import (
    "context"

    "ElMakina/backend/domain/entities"
)

// PlayerRepository defines the contract for player persistence.
// Implementations must handle context cancellation and transaction boundaries.
type PlayerRepository interface {
    // GetByID retrieves a player by their unique ID.
    // Returns ErrPlayerNotFound if the player doesn't exist.
    GetByID(ctx context.Context, id entities.PlayerID) (*entities.Player, error)

    // GetByToken retrieves a player by their secret reconnection token.
    // Returns ErrPlayerNotFound if the token is invalid.
    GetByToken(ctx context.Context, token string) (*entities.Player, error)

    // GetByIDs retrieves multiple players by their IDs.
    // Returns partial results; missing players are silently skipped.
    GetByIDs(ctx context.Context, ids []entities.PlayerID) ([]*entities.Player, error)

    // Save persists a player (insert or update).
    // For updates, the version field is used for optimistic locking.
    Save(ctx context.Context, player *entities.Player) error

    // Delete removes a player permanently.
    // Returns ErrPlayerNotFound if the player doesn't exist.
    Delete(ctx context.Context, id entities.PlayerID) error

    // Exists checks if a player with the given ID exists.
    Exists(ctx context.Context, id entities.PlayerID) (bool, error)
}
```

### 3.2 Lobby Repository

```go
// File: backend/domain/repositories/lobby_repository.go

package repositories

import (
    "context"

    "ElMakina/backend/domain/entities"
)

// LobbyListFilter provides filtering options for lobby queries.
type LobbyListFilter struct {
    Status    *entities.LobbyStatus // Filter by status (nil = all)
    MinPlayers int                  // Minimum player count
    MaxPlayers int                  // Maximum player count
    Limit      int                  // Max results (0 = no limit)
    Offset     int                  // Pagination offset
}

// LobbyWithPlayers bundles a lobby with its full player details.
type LobbyWithPlayers struct {
    Lobby   *entities.Lobby
    Players []*entities.Player
}

// LobbyRepository defines the contract for lobby persistence.
type LobbyRepository interface {
    // GetByID retrieves a lobby by ID without player details.
    // Returns ErrLobbyNotFound if the lobby doesn't exist.
    GetByID(ctx context.Context, id entities.LobbyID) (*entities.Lobby, error)

    // GetByIDWithPlayers retrieves a lobby with all player details populated.
    // Returns ErrLobbyNotFound if the lobby doesn't exist.
    GetByIDWithPlayers(ctx context.Context, id entities.LobbyID) (*LobbyWithPlayers, error)

    // GetByPlayerID finds the lobby containing the given player.
    // Returns ErrLobbyNotFound if the player is not in any lobby.
    GetByPlayerID(ctx context.Context, playerID entities.PlayerID) (*entities.Lobby, error)

    // List retrieves lobbies matching the filter criteria.
    // Returns empty slice (not error) if no lobbies match.
    List(ctx context.Context, filter LobbyListFilter) ([]*entities.Lobby, error)

    // ListWithPlayerCounts retrieves lobbies with player counts for listing.
    // More efficient than full player details for lobby browser.
    ListWithPlayerCounts(ctx context.Context, filter LobbyListFilter) ([]LobbySummary, error)

    // Save persists a lobby (insert or update).
    // Uses optimistic locking: increments version, checks for conflicts.
    Save(ctx context.Context, lobby *entities.Lobby) error

    // Delete removes a lobby and all its player associations.
    // Returns ErrLobbyNotFound if the lobby doesn't exist.
    Delete(ctx context.Context, id entities.LobbyID) error

    // Exists checks if a lobby with the given ID exists.
    Exists(ctx context.Context, id entities.LobbyID) (bool, error)
}

// LobbySummary is a lightweight view for lobby listing.
type LobbySummary struct {
    ID           entities.LobbyID
    LeaderID     entities.PlayerID
    LeaderNick   string
    PlayerCount  int
    PlayerNicks  []string
    Status       entities.LobbyStatus
    CreatedAt    int64 // Unix timestamp for JSON compatibility
}
```

### 3.3 Unit of Work (Transaction Boundary)

```go
// File: backend/domain/repositories/unit_of_work.go

package repositories

import (
    "context"

    "ElMakina/backend/domain/entities"
)

// UnitOfWork provides atomic operations across multiple repositories.
// It ensures that complex operations (like join lobby + update player)
// are persisted atomically.
type UnitOfWork interface {
    // Execute runs the given function within a transaction.
    // If the function returns an error, the transaction is rolled back.
    // If the function returns nil, the transaction is committed.
    Execute(ctx context.Context, fn func(ctx context.Context) error) error
}

// AtomicLobbyOperations defines atomic operations that span multiple entities.
// These are implemented using UnitOfWork to ensure consistency.
type AtomicLobbyOperations interface {
    // JoinLobbyAtomically adds a player to a lobby in a single transaction.
    // Validates lobby is open, not full, and player is not already present.
    JoinLobbyAtomically(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID, maxPlayers int) error

    // LeaveLobbyAtomically removes a player from a lobby.
    // Handles leader transfer and lobby closure if empty.
    LeaveLobbyAtomically(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) (lobbyClosed bool, err error)

    // StartLobbyAtomically transitions lobby to in-game and creates session metadata.
    // Returns session data needed to initialize the game.
    StartLobbyAtomically(ctx context.Context, lobbyID entities.LobbyID, leaderID entities.PlayerID, minPlayers int) (*SessionInitData, error)

    // RegisterPlayerAtomically creates a player and optionally adds them to a lobby.
    RegisterPlayerAtomically(ctx context.Context, nick, token string, lobbyID *entities.LobbyID) (*entities.Player, error)
}

// SessionInitData contains all information needed to start a game session.
type SessionInitData struct {
    MatchID       string
    LobbyID       entities.LobbyID
    PlayerIDs     []entities.PlayerID
    PlayerNicks   []string
    PlayerAvatars []string
    RNGSeed       string
}

// UnitOfWorkProvider creates UnitOfWork instances bound to a persistence backend.
type UnitOfWorkProvider interface {
    // NewUnitOfWork creates a new UnitOfWork for atomic operations.
    NewUnitOfWork() UnitOfWork

    // AtomicLobby returns the atomic operations interface.
    AtomicLobby() AtomicLobbyOperations
}
```

---

## 4. Transaction Strategy

### 4.1 Unit of Work Implementation

```go
// File: backend/infrastructure/persistence/transactions/unit_of_work.go

package transactions

import (
    "context"
    "fmt"

    "ElMakina/backend/domain/repositories"
    "gorm.io/gorm"
)

// GormUnitOfWork implements UnitOfWork using GORM transactions.
type GormUnitOfWork struct {
    db *gorm.DB
}

// NewGormUnitOfWork creates a new UnitOfWork bound to a GORM database.
func NewGormUnitOfWork(db *gorm.DB) *GormUnitOfWork {
    return &GormUnitOfWork{db: db}
}

// Execute implements the UnitOfWork interface.
func (u *GormUnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
    return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // Create a context with the transaction
        txCtx := ContextWithDB(ctx, tx)
        return fn(txCtx)
    })
}

// contextKey is a private type for context keys to avoid collisions.
type contextKey struct{ name string }

var dbContextKey = &contextKey{"db"}

// ContextWithDB injects a GORM DB into the context.
func ContextWithDB(ctx context.Context, db *gorm.DB) context.Context {
    return context.WithValue(ctx, dbContextKey, db)
}

// DBFromContext extracts a GORM DB from the context.
// Returns the default DB if no transaction is in progress.
func DBFromContext(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
    if db, ok := ctx.Value(dbContextKey).(*gorm.DB); ok {
        return db
    }
    return defaultDB
}
```

### 4.2 Repository Transaction Support

```go
// File: backend/infrastructure/persistence/repositories/postgres_lobby_repository.go (excerpt)

// postgresLobbyRepository implements repositories.LobbyRepository.
type postgresLobbyRepository struct {
    db *gorm.DB
}

// NewPostgresLobbyRepository creates a new PostgreSQL lobby repository.
func NewPostgresLobbyRepository(db *gorm.DB) repositories.LobbyRepository {
    return &postgresLobbyRepository{db: db}
}

// getDB returns the appropriate DB instance (transaction or default).
func (r *postgresLobbyRepository) getDB(ctx context.Context) *gorm.DB {
    return transactions.DBFromContext(ctx, r.db)
}

// Save persists a lobby with optimistic locking.
func (r *postgresLobbyRepository) Save(ctx context.Context, lobby *entities.Lobby) error {
    db := r.getDB(ctx)

    model := mappers.LobbyToModel(lobby)

    // Try update with optimistic locking
    result := db.Model(&models.LobbyModel{}).
        Where("id = ? AND version = ?", model.ID, lobby.Version()).
        Updates(map[string]any{
            "leader_id":  model.LeaderID,
            "status":     model.Status,
            "version":    lobby.Version() + 1,
            "updated_at": time.Now(),
        })

    if result.Error != nil {
        return fmt.Errorf("failed to save lobby: %w", result.Error)
    }

    if result.RowsAffected == 0 {
        return fmt.Errorf("lobby modified by another transaction: %w", entities.ErrConcurrentModification)
    }

    // Update player associations
    // ... (implementation details)

    return nil
}
```

---

## 5. Migration Strategy

### 5.1 Database Schema Migration

```sql
-- Migration: 001_create_normalized_lobby_schema.sql
-- Created: 2026-01-30
-- Description: Normalize lobby data from JSON blob to relational schema

-- Step 1: Create new normalized tables
CREATE TABLE players (
    id VARCHAR(64) PRIMARY KEY,
    nick VARCHAR(32) NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    avatar TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_players_token ON players(token);

CREATE TABLE lobbies (
    id VARCHAR(64) PRIMARY KEY,
    leader_id VARCHAR(64) NOT NULL REFERENCES players(id),
    status VARCHAR(16) NOT NULL CHECK (status IN ('open', 'in_game', 'closed')),
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lobbies_status ON lobbies(status);
CREATE INDEX idx_lobbies_leader ON lobbies(leader_id);

CREATE TABLE lobby_players (
    lobby_id VARCHAR(64) NOT NULL REFERENCES lobbies(id) ON DELETE CASCADE,
    player_id VARCHAR(64) NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (lobby_id, player_id)
);

CREATE INDEX idx_lobby_players_player ON lobby_players(player_id);

-- Step 2: Migration tracking table
CREATE TABLE _migration_status (
    key VARCHAR(64) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO _migration_status (key, value) VALUES ('lobby_migration_phase', 'schema_created');
```

### 5.2 Data Migration Plan

```go
// File: backend/infrastructure/persistence/migrations/lobby_migration.go

package migrations

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "ElMakina/backend/domain/entities"
    "ElMakina/backend/infrastructure/persistence/models"
    "ElMakina/backend/server"
    "gorm.io/datatypes"
    "gorm.io/gorm"
)

// LobbyMigration handles migrating JSON blob data to normalized schema.
type LobbyMigration struct {
    db *gorm.DB
}

// NewLobbyMigration creates a new migration handler.
func NewLobbyMigration(db *gorm.DB) *LobbyMigration {
    return &LobbyMigration{db: db}
}

// Phase represents the current migration state.
type Phase string

const (
    PhaseNotStarted    Phase = "not_started"
    PhaseSchemaCreated Phase = "schema_created"
    PhaseDataMigrated  Phase = "data_migrated"
    PhaseComplete      Phase = "complete"
)

// GetPhase returns the current migration phase.
func (m *LobbyMigration) GetPhase(ctx context.Context) (Phase, error) {
    var status models.MigrationStatus
    result := m.db.WithContext(ctx).First(&status, "key = ?", "lobby_migration_phase")
    if result.Error == gorm.ErrRecordNotFound {
        return PhaseNotStarted, nil
    }
    if result.Error != nil {
        return "", result.Error
    }
    return Phase(status.Value), nil
}

// setPhase updates the migration phase.
func (m *LobbyMigration) setPhase(ctx context.Context, phase Phase) error {
    return m.db.WithContext(ctx).Exec(`
        INSERT INTO _migration_status (key, value, updated_at)
        VALUES ('lobby_migration_phase', ?, ?)
        ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
    `, string(phase), time.Now()).Error
}

// MigrateData migrates existing JSON blob data to normalized tables.
func (m *LobbyMigration) MigrateData(ctx context.Context) error {
    phase, err := m.GetPhase(ctx)
    if err != nil {
        return fmt.Errorf("failed to get migration phase: %w", err)
    }
    if phase != PhaseSchemaCreated {
        return fmt.Errorf("migration not in correct phase: %s", phase)
    }

    // Load latest snapshot from legacy table
    var record struct {
        Data datatypes.JSON
    }
    result := m.db.WithContext(ctx).Table("lobby_snapshots").
        Order("updated_at desc").
        First(&record)

    if result.Error == gorm.ErrRecordNotFound {
        // No data to migrate
        return m.setPhase(ctx, PhaseDataMigrated)
    }
    if result.Error != nil {
        return fmt.Errorf("failed to load legacy snapshot: %w", result.Error)
    }

    var snapshot server.LobbySnapshot
    if err := json.Unmarshal(record.Data, &snapshot); err != nil {
        return fmt.Errorf("failed to unmarshal snapshot: %w", err)
    }

    // Begin transaction for data migration
    return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // Migrate players
        for id, player := range snapshot.Players {
            playerModel := models.PlayerModel{
                ID:        id,
                Nick:      player.Nick,
                Token:     player.Token,
                Avatar:    player.Avatar,
                CreatedAt: time.Now(),
                UpdatedAt: time.Now(),
            }
            if err := tx.Create(&playerModel).Error; err != nil {
                return fmt.Errorf("failed to migrate player %s: %w", id, err)
            }
        }

        // Migrate lobbies
        for id, lobby := range snapshot.Lobbies {
            lobbyModel := models.LobbyModel{
                ID:        id,
                LeaderID:  lobby.LeaderID,
                Status:    string(lobby.Status),
                Version:   1,
                CreatedAt: time.Now(),
                UpdatedAt: time.Now(),
            }
            if err := tx.Create(&lobbyModel).Error; err != nil {
                return fmt.Errorf("failed to migrate lobby %s: %w", id, err)
            }

            // Migrate lobby player associations
            for _, playerID := range lobby.PlayerIDs {
                lobbyPlayer := models.LobbyPlayerModel{
                    LobbyID:  id,
                    PlayerID: playerID,
                    JoinedAt: time.Now(),
                }
                if err := tx.Create(&lobbyPlayer).Error; err != nil {
                    return fmt.Errorf("failed to migrate lobby_player %s/%s: %w", id, playerID, err)
                }
            }
        }

        return nil
    })
}

// EnableDualWrite enables writing to both old and new schemas.
func (m *LobbyMigration) EnableDualWrite(ctx context.Context) error {
    // Set feature flag in migration status
    return m.db.WithContext(ctx).Exec(`
        INSERT INTO _migration_status (key, value, updated_at)
        VALUES ('lobby_dual_write', 'true', ?)
        ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
    `, time.Now()).Error
}

// IsDualWriteEnabled checks if dual write mode is active.
func (m *LobbyMigration) IsDualWriteEnabled(ctx context.Context) (bool, error) {
    var status models.MigrationStatus
    result := m.db.WithContext(ctx).First(&status, "key = ?", "lobby_dual_write")
    if result.Error == gorm.ErrRecordNotFound {
        return false, nil
    }
    if result.Error != nil {
        return false, result.Error
    }
    return status.Value == "true", nil
}

// CompleteMigration finalizes the migration and disables legacy writes.
func (m *LobbyMigration) CompleteMigration(ctx context.Context) error {
    // Verify all data is migrated
    var legacyCount, newCount int64
    m.db.WithContext(ctx).Table("lobby_snapshots").Count(&legacyCount)
    m.db.WithContext(ctx).Model(&models.LobbyModel{}).Count(&newCount)

    if newCount == 0 && legacyCount > 0 {
        return fmt.Errorf("migration incomplete: no lobbies in new schema")
    }

    // Disable dual write
    if err := m.db.WithContext(ctx).Exec(`
        UPDATE _migration_status SET value = 'false', updated_at = ? WHERE key = 'lobby_dual_write'
    `, time.Now()).Error; err != nil {
        return err
    }

    // Mark complete
    return m.setPhase(ctx, PhaseComplete)
}
```

### 5.3 Migration Rollback Plan

```go
// Rollback performs a rollback of the migration.
// WARNING: This will lose any data created after migration.
func (m *LobbyMigration) Rollback(ctx context.Context) error {
    return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // Drop new tables
        if err := tx.Migrator().DropTable("lobby_players", "lobbies", "players"); err != nil {
            return fmt.Errorf("failed to drop tables: %w", err)
        }

        // Reset migration status
        if err := tx.Delete(&models.MigrationStatus{}, "key LIKE 'lobby_%'").Error; err != nil {
            return fmt.Errorf("failed to reset migration status: %w", err)
        }

        return nil
    })
}
```

---

## 6. Implementation Priority

### Phase 1: Foundation (Week 1)

1. **Create domain entities** (`domain/entities/`)
   - Player entity with validation
   - Lobby entity with invariants
   - Domain errors

2. **Define repository interfaces** (`domain/repositories/`)
   - PlayerRepository interface
   - LobbyRepository interface
   - UnitOfWork interface

3. **Create GORM models** (`infrastructure/persistence/models/`)
   - PlayerModel
   - LobbyModel
   - LobbyPlayerModel (junction table)

### Phase 2: Repository Implementation (Week 2)

1. **Implement mappers** (`infrastructure/persistence/mappers/`)
   - Player mapper (domain <-> model)
   - Lobby mapper (domain <-> model)

2. **Implement repositories** (`infrastructure/persistence/repositories/`)
   - PostgresPlayerRepository
   - PostgresLobbyRepository

3. **Implement transactions** (`infrastructure/persistence/transactions/`)
   - GormUnitOfWork
   - Context propagation

### Phase 3: Migration & Dual Write (Week 3)

1. **Create migration scripts**
   - Schema migration SQL
   - Data migration code
   - Rollback procedures

2. **Implement dual-write mode**
   - Legacy store writes to both schemas
   - Verification scripts

3. **Test migration**
   - Unit tests for migration logic
   - Integration tests with real data

### Phase 4: Refactor Lobby Manager (Week 4)

1. **Create LobbyManager v2**
   - Use repository interfaces
   - Add transaction boundaries
   - Implement atomic operations

2. **Update WebSocket server**
   - Wire new LobbyManager
   - Maintain API compatibility

3. **Deprecate legacy code**
   - Mark old store as deprecated
   - Add migration warnings

### Phase 5: Cleanup (Week 5)

1. **Disable dual write**
   - Switch to new schema only
   - Monitor for issues

2. **Remove legacy code**
   - Delete old JSON blob store
   - Clean up migration tables

3. **Performance optimization**
   - Add missing indexes
   - Optimize queries

---

## 7. Testing Strategy

### 7.1 Unit Tests (Table-Driven)

```go
// File: backend/domain/entities/lobby_test.go

package entities

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLobby_AddPlayer(t *testing.T) {
    type args struct {
        playerID   PlayerID
        maxPlayers int
    }

    tests := []struct {
        name      string
        setup     func() *Lobby
        args      args
        wantErr   error
        wantCount int
    }{
        {
            name: "add player to open lobby",
            setup: func() *Lobby {
                l, _ := NewLobby("lobby-1", "leader-1")
                return l
            },
            args:      args{playerID: "player-2", maxPlayers: 4},
            wantErr:   nil,
            wantCount: 2,
        },
        {
            name: "reject duplicate player",
            setup: func() *Lobby {
                l, _ := NewLobby("lobby-1", "leader-1")
                _ = l.AddPlayer("player-2", 4)
                return l
            },
            args:      args{playerID: "player-2", maxPlayers: 4},
            wantErr:   ErrPlayerAlreadyInLobby,
            wantCount: 2,
        },
        {
            name: "reject when lobby full",
            setup: func() *Lobby {
                l, _ := NewLobby("lobby-1", "leader-1")
                _ = l.AddPlayer("p2", 2)
                return l
            },
            args:      args{playerID: "p3", maxPlayers: 2},
            wantErr:   ErrLobbyFull,
            wantCount: 2,
        },
        {
            name: "reject when lobby not open",
            setup: func() *Lobby {
                l, _ := NewLobby("lobby-1", "leader-1")
                _ = l.AddPlayer("p2", 4)
                _ = l.StartGame(2)
                return l
            },
            args:      args{playerID: "p3", maxPlayers: 4},
            wantErr:   ErrLobbyNotOpen,
            wantCount: 2,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := tt.setup()
            err := l.AddPlayer(tt.args.playerID, tt.args.maxPlayers)

            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
            } else {
                require.NoError(t, err)
            }
            assert.Len(t, l.players, tt.wantCount)
        })
    }
}

func TestLobby_RemovePlayer(t *testing.T) {
    tests := []struct {
        name          string
        setup         func() *Lobby
        playerID      PlayerID
        wantClose     bool
        wantNewLeader PlayerID
        wantErr       error
    }{
        {
            name: "remove non-leader",
            setup: func() *Lobby {
                l, _ := NewLobby("lobby-1", "leader-1")
                _ = l.AddPlayer("player-2", 4)
                return l
            },
            playerID:      "player-2",
            wantClose:     false,
            wantNewLeader: "leader-1",
            wantErr:       nil,
        },
        {
            name: "remove leader promotes next player",
            setup: func() *Lobby {
                l, _ := NewLobby("lobby-1", "leader-1")
                _ = l.AddPlayer("player-2", 4)
                return l
            },
            playerID:      "leader-1",
            wantClose:     false,
            wantNewLeader: "player-2",
            wantErr:       nil,
        },
        {
            name: "remove last player closes lobby",
            setup: func() *Lobby {
                l, _ := NewLobby("lobby-1", "leader-1")
                return l
            },
            playerID:  "leader-1",
            wantClose: true,
            wantErr:   nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := tt.setup()
            shouldClose, err := l.RemovePlayer(tt.playerID)

            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.wantClose, shouldClose)
            if !tt.wantClose {
                assert.Equal(t, tt.wantNewLeader, l.LeaderID())
            }
        })
    }
}
```

### 7.2 Repository Integration Tests

```go
// File: backend/infrastructure/persistence/repositories/postgres_lobby_repository_test.go

package repositories

import (
    "context"
    "testing"

    "ElMakina/backend/domain/entities"
    "ElMakina/backend/infrastructure/persistence/testutil"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
)

type PostgresLobbyRepositoryTestSuite struct {
    suite.Suite
    db         *testutil.TestDB
    repo       repositories.LobbyRepository
    playerRepo repositories.PlayerRepository
}

func (s *PostgresLobbyRepositoryTestSuite) SetupSuite() {
    s.db = testutil.NewTestDB(s.T())
    s.repo = NewPostgresLobbyRepository(s.db.DB)
    s.playerRepo = NewPostgresPlayerRepository(s.db.DB)
}

func (s *PostgresLobbyRepositoryTestSuite) TearDownTest() {
    s.db.Cleanup()
}

func (s *PostgresLobbyRepositoryTestSuite) TestSaveAndGet() {
    ctx := context.Background()

    // Create and save a player (leader)
    leader, err := entities.NewPlayer("player-1", "Leader", "token-1")
    require.NoError(s.T(), err)
    require.NoError(s.T(), s.playerRepo.Save(ctx, leader))

    // Create and save a lobby
    lobby, err := entities.NewLobby("lobby-1", leader.ID())
    require.NoError(s.T(), err)
    require.NoError(s.T(), s.repo.Save(ctx, lobby))

    // Retrieve and verify
    retrieved, err := s.repo.GetByID(ctx, "lobby-1")
    require.NoError(s.T(), err)

    assert.Equal(s.T(), lobby.ID(), retrieved.ID())
    assert.Equal(s.T(), lobby.LeaderID(), retrieved.LeaderID())
    assert.Equal(s.T(), lobby.Version(), retrieved.Version())
}

func (s *PostgresLobbyRepositoryTestSuite) TestOptimisticLocking() {
    ctx := context.Background()

    // Setup
    leader, _ := entities.NewPlayer("player-1", "Leader", "token-1")
    s.playerRepo.Save(ctx, leader)
    lobby, _ := entities.NewLobby("lobby-1", leader.ID())
    s.repo.Save(ctx, lobby)

    // Simulate concurrent modification
    lobby2, _ := s.repo.GetByID(ctx, "lobby-1")

    // First update succeeds
    lobby.AddPlayer("player-2", 4)
    require.NoError(s.T(), s.repo.Save(ctx, lobby))

    // Second update fails due to version mismatch
    lobby2.AddPlayer("player-3", 4)
    err := s.repo.Save(ctx, lobby2)
    assert.ErrorIs(s.T(), err, entities.ErrConcurrentModification)
}

func TestPostgresLobbyRepository(t *testing.T) {
    suite.Run(t, new(PostgresLobbyRepositoryTestSuite))
}
```

### 7.3 Transaction Tests

```go
// File: backend/infrastructure/persistence/transactions/unit_of_work_test.go

package transactions

import (
    "context"
    "errors"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGormUnitOfWork_Execute(t *testing.T) {
    db := testutil.NewTestDB(t)
    uow := NewGormUnitOfWork(db.DB)

    t.Run("commits on success", func(t *testing.T) {
        ctx := context.Background()
        var committed bool

        err := uow.Execute(ctx, func(ctx context.Context) error {
            committed = true
            return nil
        })

        require.NoError(t, err)
        assert.True(t, committed)
    })

    t.Run("rolls back on error", func(t *testing.T) {
        ctx := context.Background()
        testErr := errors.New("test error")

        err := uow.Execute(ctx, func(ctx context.Context) error {
            return testErr
        })

        assert.ErrorIs(t, err, testErr)
    })

    t.Run("propagates context cancellation", func(t *testing.T) {
        ctx, cancel := context.WithCancel(context.Background())
        cancel() // Cancel immediately

        err := uow.Execute(ctx, func(ctx context.Context) error {
            return nil
        })

        assert.ErrorIs(t, err, context.Canceled)
    })
}
```

---

## 8. Error Handling Strategy

### 8.1 Error Hierarchy

```
errors/
├── domain/
│   ├── entity_errors.go      # Validation, invariants
│   └── repository_errors.go  # Not found, conflict
├── infrastructure/
│   └── persistence_errors.go # DB connection, timeout
└── application/
    └── service_errors.go     # Business logic failures
```

### 8.2 Error Wrapping Convention

```go
// Repository layer wraps infrastructure errors
db.First(&model).Error // gorm.ErrRecordNotFound
return fmt.Errorf("failed to get player by id %s: %w", id, ErrPlayerNotFound)

// Service layer wraps with context
player, err := r.playerRepo.GetByID(ctx, id)
if err != nil {
    return fmt.Errorf("cannot start game: %w", err)
}
```

### 8.3 HTTP Status Code Mapping

```go
// In HTTP handlers
func mapErrorToStatus(err error) int {
    switch {
    case entities.IsNotFound(err):
        return http.StatusNotFound
    case entities.IsConflict(err):
        return http.StatusConflict
    case errors.Is(err, context.DeadlineExceeded):
        return http.StatusGatewayTimeout
    default:
        return http.StatusInternalServerError
    }
}
```

---

## 9. Performance Considerations

### 9.1 Query Optimization

```go
// Eager loading for lobby with players
func (r *postgresLobbyRepository) GetByIDWithPlayers(ctx context.Context, id entities.LobbyID) (*repositories.LobbyWithPlayers, error) {
    var lobbyModel models.LobbyModel
    result := r.getDB(ctx).
        Preload("Players", func(db *gorm.DB) *gorm.DB {
            return db.Order("lobby_players.joined_at ASC")
        }).
        Preload("Players.Player").
        First(&lobbyModel, "id = ?", id)

    // ... mapping
}
```

### 9.2 Connection Pooling

```go
// In database initialization
sqlDB, err := db.DB()
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxLifetime(5 * time.Minute)
```

### 9.3 Caching Strategy (Future)

```go
// Decorator pattern for caching
type CachedLobbyRepository struct {
    inner  repositories.LobbyRepository
    cache  *redis.Client
    ttl    time.Duration
}

func (r *CachedLobbyRepository) GetByID(ctx context.Context, id entities.LobbyID) (*entities.Lobby, error) {
    // Check cache first
    // Fall through to inner on miss
}
```

---

## 10. Appendix: Full Type Reference

### Domain Entities

| Type | File | Description |
|------|------|-------------|
| `Player` | `domain/entities/player.go` | Player domain entity |
| `PlayerID` | `domain/entities/player.go` | Type-safe player identifier |
| `Lobby` | `domain/entities/lobby.go` | Lobby domain entity |
| `LobbyID` | `domain/entities/lobby.go` | Type-safe lobby identifier |
| `LobbyStatus` | `domain/entities/lobby.go` | Lobby lifecycle states |
| `LobbyPlayer` | `domain/entities/lobby.go` | Player membership in lobby |

### Repository Interfaces

| Interface | File | Methods |
|-----------|------|---------|
| `PlayerRepository` | `domain/repositories/player_repository.go` | GetByID, GetByToken, GetByIDs, Save, Delete, Exists |
| `LobbyRepository` | `domain/repositories/lobby_repository.go` | GetByID, GetByIDWithPlayers, GetByPlayerID, List, ListWithPlayerCounts, Save, Delete, Exists |
| `UnitOfWork` | `domain/repositories/unit_of_work.go` | Execute |
| `AtomicLobbyOperations` | `domain/repositories/unit_of_work.go` | JoinLobbyAtomically, LeaveLobbyAtomically, StartLobbyAtomically, RegisterPlayerAtomically |

### Infrastructure Models

| Model | File | Description |
|-------|------|-------------|
| `PlayerModel` | `infrastructure/persistence/models/player_model.go` | GORM model for players table |
| `LobbyModel` | `infrastructure/persistence/models/lobby_model.go` | GORM model for lobbies table |
| `LobbyPlayerModel` | `infrastructure/persistence/models/lobby_player_model.go` | GORM model for lobby_players junction table |

---

## Summary

This design provides:

1. **Clean separation** between domain logic and persistence
2. **Type safety** with dedicated ID types
3. **Transaction safety** via Unit of Work pattern
4. **Optimistic locking** to prevent race conditions
5. **Zero-downtime migration** with dual-write capability
6. **Comprehensive testing** strategy with table-driven tests
7. **Clear error handling** with domain-specific error types

The architecture is ready for implementation following the phased approach outlined in Section 6.
