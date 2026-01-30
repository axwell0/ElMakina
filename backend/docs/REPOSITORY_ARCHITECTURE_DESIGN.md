# ElMakina Repository Architecture Design Document

## Phase 1: Clean Architecture Migration

### Executive Summary

This document outlines the design for migrating the ElMakina backend from a JSON blob persistence anti-pattern to a production-grade Clean Architecture with proper repository patterns, normalized database schema, and transaction boundaries.

---

## 1. Directory Structure

```
backend/
├── domain/                          # Pure business logic (no external deps)
│   ├── entities/                    # Core domain models
│   │   ├── player.go               # Player entity
│   │   ├── lobby.go                # Lobby entity with business rules
│   │   └── errors.go               # Domain-specific errors
│   ├── repositories/               # Repository interfaces (ports)
│   │   ├── player_repository.go
│   │   ├── lobby_repository.go
│   │   └── unit_of_work.go         # Transaction abstraction
│   └── services/                   # Domain services (complex operations)
│       └── lobby_service.go        # Orchestration logic
│
├── infrastructure/                  # External concerns
│   └── persistence/
│       ├── models/                 # GORM models (internal only)
│       │   ├── player_model.go
│       │   ├── lobby_model.go
│       │   └── lobby_player_model.go  # Junction table
│       ├── repositories/           # Repository implementations
│       │   ├── postgres_player_repository.go
│       │   ├── postgres_lobby_repository.go
│       │   └── postgres_unit_of_work.go
│       ├── transactions/           # Transaction management
│       │   └── gorm_unit_of_work.go
│       └── migrations/             # Database migrations
│           ├── 001_create_players_table.sql
│           ├── 002_create_lobbies_table.sql
│           └── 003_create_lobby_players_table.sql
│
├── application/                     # Use cases / Application layer
│   ├── dto/                        # Data transfer objects
│   │   ├── player_dto.go
│   │   └── lobby_dto.go
│   └── services/                   # Application services
│       └── lobby_application_service.go
│
└── server/                         # Interface adapters (existing)
    ├── ws/                         # WebSocket handlers
    ├── http/                       # HTTP handlers
    └── ...                         # Existing code
```

---

## 2. Domain Entities

### 2.1 Player Entity

```go
// domain/entities/player.go
package entities

import (
	"fmt"
	"time"
)

// PlayerID is a strongly-typed identifier for type safety.
type PlayerID string

func (id PlayerID) String() string { return string(id) }

// Player represents a human player in the system.
// This is the canonical domain model - no GORM tags, no JSON serialization concerns.
type Player struct {
	id        PlayerID
	nick      string
	token     string // Secret reconnection token
	avatar    string
	createdAt time.Time
	updatedAt time.Time
}

// NewPlayer creates a validated Player entity.
func NewPlayer(id PlayerID, nick, token string, createdAt time.Time) (*Player, error) {
	if id == "" {
		return nil, fmt.Errorf("player id is required")
	}
	if nick == "" {
		return nil, fmt.Errorf("player nick is required")
	}
	if token == "" {
		return nil, fmt.Errorf("player token is required")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return &Player{
		id:        id,
		nick:      nick,
		token:     token,
		createdAt: createdAt,
		updatedAt: createdAt,
	}, nil
}

// ReconstitutePlayer recreates a Player from persisted data.
// Use this only in repositories when loading from database.
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

// --- Getters ---

func (p *Player) ID() PlayerID        { return p.id }
func (p *Player) Nick() string        { return p.nick }
func (p *Player) Token() string       { return p.token }
func (p *Player) Avatar() string      { return p.avatar }
func (p *Player) CreatedAt() time.Time { return p.createdAt }
func (p *Player) UpdatedAt() time.Time { return p.updatedAt }

// --- Domain Behavior ---

// SetAvatar updates the player's avatar.
func (p *Player) SetAvatar(avatar string) error {
	if avatar == "" {
		return fmt.Errorf("avatar cannot be empty")
	}
	p.avatar = avatar
	p.updatedAt = time.Now().UTC()
	return nil
}

// ChangeNick updates the player's nickname.
func (p *Player) ChangeNick(newNick string) error {
	if newNick == "" {
		return fmt.Errorf("nick cannot be empty")
	}
	p.nick = newNick
	p.updatedAt = time.Now().UTC()
	return nil
}

// ValidateToken checks if the provided token matches.
func (p *Player) ValidateToken(token string) bool {
	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(p.token), []byte(token)) == 1
}
```

### 2.2 Lobby Entity

```go
// domain/entities/lobby.go
package entities

import (
	"fmt"
	"time"
)

// LobbyID is a strongly-typed identifier.
type LobbyID string

func (id LobbyID) String() string { return string(id) }

// LobbyStatus represents the lifecycle state of a lobby.
type LobbyStatus string

const (
	LobbyStatusOpen    LobbyStatus = "open"
	LobbyStatusInGame  LobbyStatus = "in_game"
	LobbyStatusClosed  LobbyStatus = "closed"
)

// LobbyPlayer represents a player's membership in a lobby.
// This is a value object within the Lobby aggregate.
type LobbyPlayer struct {
	playerID PlayerID
	joinedAt time.Time
}

func (lp LobbyPlayer) PlayerID() PlayerID { return lp.playerID }
func (lp LobbyPlayer) JoinedAt() time.Time { return lp.joinedAt }

// Lobby is the aggregate root for the lobby domain.
// It enforces all invariants related to lobby membership and lifecycle.
type Lobby struct {
	id       LobbyID
	leaderID PlayerID
	status   LobbyStatus
	players  []LobbyPlayer // Ordered list of players
	version  int64         // Optimistic locking version
	createdAt time.Time
	updatedAt time.Time
}

// NewLobby creates a new lobby with the given leader.
func NewLobby(id LobbyID, leaderID PlayerID, createdAt time.Time) (*Lobby, error) {
	if id == "" {
		return nil, fmt.Errorf("lobby id is required")
	}
	if leaderID == "" {
		return nil, fmt.Errorf("leader id is required")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return &Lobby{
		id:        id,
		leaderID:  leaderID,
		status:    LobbyStatusOpen,
		players:   []LobbyPlayer{{playerID: leaderID, joinedAt: createdAt}},
		version:   1,
		createdAt: createdAt,
		updatedAt: createdAt,
	}, nil
}

// ReconstituteLobby recreates a Lobby from persisted data.
// Use only in repositories.
func ReconstituteLobby(
	id LobbyID,
	leaderID PlayerID,
	status LobbyStatus,
	players []LobbyPlayer,
	version int64,
	createdAt, updatedAt time.Time,
) *Lobby {
	return &Lobby{
		id:        id,
		leaderID:  leaderID,
		status:    status,
		players:   players,
		version:   version,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// --- Getters ---

func (l *Lobby) ID() LobbyID              { return l.id }
func (l *Lobby) LeaderID() PlayerID       { return l.leaderID }
func (l *Lobby) Status() LobbyStatus      { return l.status }
func (l *Lobby) Version() int64           { return l.version }
func (l *Lobby) CreatedAt() time.Time     { return l.createdAt }
func (l *Lobby) UpdatedAt() time.Time     { return l.updatedAt }

// Players returns a copy of the player list to prevent external mutation.
func (l *Lobby) Players() []LobbyPlayer {
	result := make([]LobbyPlayer, len(l.players))
	copy(result, l.players)
	return result
}

// PlayerIDs returns just the player IDs for convenience.
func (l *Lobby) PlayerIDs() []PlayerID {
	result := make([]PlayerID, len(l.players))
	for i, p := range l.players {
		result[i] = p.playerID
	}
	return result
}

// PlayerCount returns the number of players in the lobby.
func (l *Lobby) PlayerCount() int {
	return len(l.players)
}

// HasPlayer checks if a player is in the lobby.
func (l *Lobby) HasPlayer(playerID PlayerID) bool {
	for _, p := range l.players {
		if p.playerID == playerID {
			return true
		}
	}
	return false
}

// IsLeader checks if the given player is the lobby leader.
func (l *Lobby) IsLeader(playerID PlayerID) bool {
	return l.leaderID == playerID
}

// --- Domain Behavior ---

// AddPlayer adds a player to the lobby.
// Returns error if lobby is not open, player already present, or lobby is full.
func (l *Lobby) AddPlayer(playerID PlayerID, maxPlayers int) error {
	if l.status != LobbyStatusOpen {
		return ErrLobbyNotOpen
	}
	if l.HasPlayer(playerID) {
		return ErrPlayerAlreadyInLobby
	}
	if len(l.players) >= maxPlayers {
		return ErrLobbyFull
	}

	l.players = append(l.players, LobbyPlayer{
		playerID: playerID,
		joinedAt: time.Now().UTC(),
	})
	l.touch()
	return nil
}

// RemovePlayer removes a player from the lobby.
// If the leader leaves, the next player becomes leader.
// Returns true if the lobby should be closed (empty).
func (l *Lobby) RemovePlayer(playerID PlayerID) (shouldClose bool, err error) {
	if l.status != LobbyStatusOpen {
		return false, ErrLobbyNotOpen
	}
	if !l.HasPlayer(playerID) {
		return false, ErrPlayerNotInLobby
	}

	// Find and remove player
	newPlayers := make([]LobbyPlayer, 0, len(l.players)-1)
	for _, p := range l.players {
		if p.playerID != playerID {
			newPlayers = append(newPlayers, p)
		}
	}
	l.players = newPlayers

	// Handle leader departure
	if l.leaderID == playerID && len(l.players) > 0 {
		l.leaderID = l.players[0].playerID
	}

	l.touch()
	return len(l.players) == 0, nil
}

// Start transitions the lobby to in-game status.
// Only the leader can start, and minimum players must be met.
func (l *Lobby) Start(leaderID PlayerID, minPlayers int) error {
	if l.status != LobbyStatusOpen {
		return ErrLobbyNotOpen
	}
	if l.leaderID != leaderID {
		return ErrNotLobbyLeader
	}
	if len(l.players) < minPlayers {
		return ErrNotEnoughPlayers
	}

	l.status = LobbyStatusInGame
	l.touch()
	return nil
}

// Close marks the lobby as closed.
func (l *Lobby) Close() error {
	if l.status == LobbyStatusClosed {
		return ErrLobbyAlreadyClosed
	}
	l.status = LobbyStatusClosed
	l.touch()
	return nil
}

// ResetToOpen transitions an in-game lobby back to open.
// Used after server restart when sessions are not recoverable.
func (l *Lobby) ResetToOpen() error {
	if l.status != LobbyStatusInGame {
		return ErrLobbyNotInGame
	}
	l.status = LobbyStatusOpen
	l.touch()
	return nil
}

// touch updates the updatedAt timestamp and increments version.
func (l *Lobby) touch() {
	l.updatedAt = time.Now().UTC()
	l.version++
}
```

### 2.3 Domain Errors

```go
// domain/entities/errors.go
package entities

import "errors"

// Domain errors for lobby operations.
// These are business rule violations, not infrastructure errors.
var (
	ErrLobbyNotOpen       = errors.New("lobby is not open")
	ErrLobbyNotInGame     = errors.New("lobby is not in-game")
	ErrLobbyAlreadyClosed = errors.New("lobby is already closed")
	ErrLobbyFull          = errors.New("lobby is full")
	
	ErrPlayerAlreadyInLobby = errors.New("player is already in the lobby")
	ErrPlayerNotInLobby     = errors.New("player is not in the lobby")
	
	ErrNotLobbyLeader    = errors.New("only the lobby leader can perform this action")
	ErrNotEnoughPlayers  = errors.New("not enough players to start")
	
	ErrPlayerNotFound = errors.New("player not found")
	ErrLobbyNotFound  = errors.New("lobby not found")
)
```

---

## 3. Repository Interfaces

### 3.1 Player Repository

```go
// domain/repositories/player_repository.go
package repositories

import (
	"context"
	
	"ElMakina/backend/domain/entities"
)

// PlayerRepository defines the contract for player persistence.
// Implementations may use PostgreSQL, Redis, or any other storage.
type PlayerRepository interface {
	// GetByID retrieves a player by their unique ID.
	// Returns ErrPlayerNotFound if the player doesn't exist.
	GetByID(ctx context.Context, id entities.PlayerID) (*entities.Player, error)
	
	// GetByToken retrieves a player by their reconnection token.
	// Returns ErrPlayerNotFound if no player has this token.
	GetByToken(ctx context.Context, token string) (*entities.Player, error)
	
	// GetByIDs retrieves multiple players by their IDs.
	// Missing players are silently ignored (returned slice may be shorter than input).
	GetByIDs(ctx context.Context, ids []entities.PlayerID) ([]*entities.Player, error)
	
	// Save persists a player. Creates if new, updates if existing.
	Save(ctx context.Context, player *entities.Player) error
	
	// Delete removes a player permanently.
	Delete(ctx context.Context, id entities.PlayerID) error
	
	// Exists checks if a player with the given ID exists.
	Exists(ctx context.Context, id entities.PlayerID) (bool, error)
}
```

### 3.2 Lobby Repository

```go
// domain/repositories/lobby_repository.go
package repositories

import (
	"context"
	
	"ElMakina/backend/domain/entities"
)

// LobbyFilter provides filtering options for lobby queries.
type LobbyFilter struct {
	Status    *entities.LobbyStatus
	LeaderID  *entities.PlayerID
	PlayerID  *entities.PlayerID
	Limit     int
	Offset    int
}

// LobbyRepository defines the contract for lobby persistence.
type LobbyRepository interface {
	// GetByID retrieves a lobby by ID with all players populated.
	// Returns ErrLobbyNotFound if the lobby doesn't exist.
	GetByID(ctx context.Context, id entities.LobbyID) (*entities.Lobby, error)
	
	// GetByIDWithPlayers retrieves a lobby with full player entities populated.
	// This is useful when you need player details (nicks, avatars) for a lobby.
	GetByIDWithPlayers(ctx context.Context, id entities.LobbyID) (*entities.Lobby, []*entities.Player, error)
	
	// List retrieves lobbies matching the filter criteria.
	List(ctx context.Context, filter LobbyFilter) ([]*entities.Lobby, error)
	
	// ListOpen returns all open lobbies (convenience method).
	ListOpen(ctx context.Context) ([]*entities.Lobby, error)
	
	// Save persists a lobby. Creates if new, updates if existing.
	// Uses optimistic locking - returns ErrVersionConflict if version mismatch.
	Save(ctx context.Context, lobby *entities.Lobby) error
	
	// Delete removes a lobby permanently.
	Delete(ctx context.Context, id entities.LobbyID) error
	
	// Exists checks if a lobby with the given ID exists.
	Exists(ctx context.Context, id entities.LobbyID) (bool, error)
	
	// FindByPlayer returns the lobby ID containing the player, if any.
	FindByPlayer(ctx context.Context, playerID entities.PlayerID) (entities.LobbyID, bool, error)
}

// ErrVersionConflict is returned when optimistic locking fails.
var ErrVersionConflict = errors.New("version conflict: entity was modified by another process")
```

### 3.3 Unit of Work (Transaction Management)

```go
// domain/repositories/unit_of_work.go
package repositories

import (
	"context"
	"errors"
)

// UnitOfWork defines a transactional boundary.
// Multiple operations can be performed within a single transaction,
// and either all succeed or all fail together.
//
// Usage pattern:
//
//	uow := unitOfWorkFactory.Begin(ctx)
//	defer uow.Rollback() // Safe to call even after Commit
//	
//	player, err := uow.Players().GetByID(ctx, playerID)
//	if err != nil { return err }
//	
//	lobby, err := uow.Lobbies().GetByID(ctx, lobbyID)
//	if err != nil { return err }
//	
//	if err := lobby.AddPlayer(player.ID(), maxPlayers); err != nil {
//	    return err
//	}
//	
//	if err := uow.Lobbies().Save(ctx, lobby); err != nil {
//	    return err
//	}
//	
//	return uow.Commit()
//
type UnitOfWork interface {
	// Players returns the player repository within this transaction.
	Players() PlayerRepository
	
	// Lobbies returns the lobby repository within this transaction.
	Lobbies() LobbyRepository
	
	// Commit saves all changes made within this unit of work.
	// Returns an error if the transaction cannot be committed.
	Commit() error
	
	// Rollback discards all changes made within this unit of work.
	// Safe to call multiple times; nil if already committed.
	Rollback() error
}

// UnitOfWorkFactory creates new units of work.
type UnitOfWorkFactory interface {
	// Begin starts a new unit of work.
	Begin(ctx context.Context) UnitOfWork
}

// ErrTransactionAlreadyCommitted is returned when attempting to rollback
// a transaction that has already been committed.
var ErrTransactionAlreadyCommitted = errors.New("transaction already committed")

// ErrTransactionAlreadyRolledBack is returned when attempting to commit
// a transaction that has already been rolled back.
var ErrTransactionAlreadyRolledBack = errors.New("transaction already rolled back")
```

---

## 4. Transaction Strategy

### 4.1 Why Unit of Work?

The current system has race conditions because operations like "join lobby" involve:
1. Checking if player exists
2. Checking if lobby exists and is open
3. Adding player to lobby
4. Saving the updated state

Without transactions, two concurrent join requests could both pass the checks and corrupt state.

### 4.2 Implementation Design

```go
// infrastructure/persistence/transactions/gorm_unit_of_work.go
package transactions

import (
	"context"
	"sync"
	
	"gorm.io/gorm"
	
	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
	"ElMakina/backend/infrastructure/persistence/repositories"
)

// GormUnitOfWork implements UnitOfWork using GORM transactions.
type GormUnitOfWork struct {
	db       *gorm.DB
	tx       *gorm.DB
	committed bool
	rolledBack bool
	mu       sync.Mutex
	
	// Repositories within this transaction
	playerRepo repositories.PlayerRepository
	lobbyRepo  repositories.LobbyRepository
}

// NewGormUnitOfWork creates a new unit of work.
func NewGormUnitOfWork(db *gorm.DB) *GormUnitOfWork {
	return &GormUnitOfWork{db: db}
}

// Begin starts the transaction.
func (u *GormUnitOfWork) Begin(ctx context.Context) repositories.UnitOfWork {
	u.tx = u.db.WithContext(ctx).Begin()
	
	// Create repositories that operate within this transaction
	u.playerRepo = repositories.NewPostgresPlayerRepository(u.tx)
	u.lobbyRepo = repositories.NewPostgresLobbyRepository(u.tx)
	
	return u
}

func (u *GormUnitOfWork) Players() repositories.PlayerRepository {
	return u.playerRepo
}

func (u *GormUnitOfWork) Lobbies() repositories.LobbyRepository {
	return u.lobbyRepo
}

func (u *GormUnitOfWork) Commit() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	if u.committed {
		return repositories.ErrTransactionAlreadyCommitted
	}
	if u.rolledBack {
		return repositories.ErrTransactionAlreadyRolledBack
	}
	
	if err := u.tx.Commit().Error; err != nil {
		return err
	}
	
	u.committed = true
	return nil
}

func (u *GormUnitOfWork) Rollback() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	if u.committed {
		return repositories.ErrTransactionAlreadyCommitted
	}
	if u.rolledBack {
		return nil // Idempotent
	}
	
	if err := u.tx.Rollback().Error; err != nil {
		return err
	}
	
	u.rolledBack = true
	return nil
}

// Ensure GormUnitOfWork implements the interface
var _ repositories.UnitOfWork = (*GormUnitOfWork)(nil)
```

### 4.3 Usage in Domain Services

```go
// domain/services/lobby_service.go
package services

import (
	"context"
	"fmt"
	
	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
)

// LobbyService orchestrates complex lobby operations.
type LobbyService struct {
	uowFactory repositories.UnitOfWorkFactory
	minPlayers int
	maxPlayers int
}

// NewLobbyService creates a new lobby service.
func NewLobbyService(uowFactory repositories.UnitOfWorkFactory, minPlayers, maxPlayers int) *LobbyService {
	return &LobbyService{
		uowFactory: uowFactory,
		minPlayers: minPlayers,
		maxPlayers: maxPlayers,
	}
}

// JoinLobby adds a player to a lobby atomically.
func (s *LobbyService) JoinLobby(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) error {
	uow := s.uowFactory.Begin(ctx)
	defer uow.Rollback() // Safe - no-op if committed
	
	// Verify player exists
	player, err := uow.Players().GetByID(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get player: %w", err)
	}
	
	// Load lobby
	lobby, err := uow.Lobbies().GetByID(ctx, lobbyID)
	if err != nil {
		return fmt.Errorf("get lobby: %w", err)
	}
	
	// Check if player is already in any lobby
	existingLobbyID, inLobby, err := uow.Lobbies().FindByPlayer(ctx, playerID)
	if err != nil {
		return fmt.Errorf("check existing lobby: %w", err)
	}
	if inLobby && existingLobbyID != lobbyID {
		return fmt.Errorf("player already in different lobby")
	}
	
	// Apply domain logic
	if err := lobby.AddPlayer(player.ID(), s.maxPlayers); err != nil {
		return err // Domain error (e.g., lobby full)
	}
	
	// Save changes
	if err := uow.Lobbies().Save(ctx, lobby); err != nil {
		return fmt.Errorf("save lobby: %w", err)
	}
	
	return uow.Commit()
}

// StartLobby transitions a lobby to in-game status.
func (s *LobbyService) StartLobby(ctx context.Context, lobbyID entities.LobbyID, leaderID entities.PlayerID) (*GameSessionInfo, error) {
	uow := s.uowFactory.Begin(ctx)
	defer uow.Rollback()
	
	lobby, err := uow.Lobbies().GetByIDWithPlayers(ctx, lobbyID)
	if err != nil {
		return nil, fmt.Errorf("get lobby: %w", err)
	}
	
	if err := lobby.Start(leaderID, s.minPlayers); err != nil {
		return nil, err
	}
	
	if err := uow.Lobbies().Save(ctx, lobby); err != nil {
		return nil, fmt.Errorf("save lobby: %w", err)
	}
	
	// Build session info from loaded data
	players := lobby.Players()
	playerIDs := make([]entities.PlayerID, len(players))
	for i, p := range players {
		playerIDs[i] = p.PlayerID()
	}
	
	if err := uow.Commit(); err != nil {
		return nil, err
	}
	
	return &GameSessionInfo{
		LobbyID:   lobbyID,
		PlayerIDs: playerIDs,
	}, nil
}

// LeaveLobby removes a player from a lobby.
func (s *LobbyService) LeaveLobby(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) (lobbyClosed bool, err error) {
	uow := s.uowFactory.Begin(ctx)
	defer uow.Rollback()
	
	lobby, err := uow.Lobbies().GetByID(ctx, lobbyID)
	if err != nil {
		return false, fmt.Errorf("get lobby: %w", err)
	}
	
	shouldClose, err := lobby.RemovePlayer(playerID)
	if err != nil {
		return false, err
	}
	
	if shouldClose {
		if err := uow.Lobbies().Delete(ctx, lobbyID); err != nil {
			return false, fmt.Errorf("delete lobby: %w", err)
		}
	} else {
		if err := uow.Lobbies().Save(ctx, lobby); err != nil {
			return false, fmt.Errorf("save lobby: %w", err)
		}
	}
	
	return shouldClose, uow.Commit()
}

// GameSessionInfo contains the data needed to start a game session.
type GameSessionInfo struct {
	LobbyID   entities.LobbyID
	PlayerIDs []entities.PlayerID
}
```

---

## 5. Migration Strategy

### 5.1 Database Schema Migration

```sql
-- migrations/001_create_players_table.sql
CREATE TABLE players (
    id VARCHAR(64) PRIMARY KEY,
    nick VARCHAR(128) NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    avatar TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_players_token ON players(token);

-- migrations/002_create_lobbies_table.sql
CREATE TABLE lobbies (
    id VARCHAR(64) PRIMARY KEY,
    leader_id VARCHAR(64) NOT NULL REFERENCES players(id),
    status VARCHAR(16) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_game', 'closed')),
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lobbies_status ON lobbies(status);
CREATE INDEX idx_lobbies_leader ON lobbies(leader_id);

-- migrations/003_create_lobby_players_table.sql
CREATE TABLE lobby_players (
    lobby_id VARCHAR(64) NOT NULL REFERENCES lobbies(id) ON DELETE CASCADE,
    player_id VARCHAR(64) NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    position INT NOT NULL DEFAULT 0,
    PRIMARY KEY (lobby_id, player_id)
);

CREATE INDEX idx_lobby_players_player ON lobby_players(player_id);

-- migrations/004_migrate_from_snapshot.sql
-- This migration converts existing JSON blob data to normalized schema

-- Step 1: Create temporary function to extract data from JSON
CREATE OR REPLACE FUNCTION migrate_lobby_snapshots()
RETURNS void AS $$
DECLARE
    snapshot_record RECORD;
    lobby_data JSONB;
    player_data JSONB;
    lobby_key TEXT;
    player_key TEXT;
    token_key TEXT;
BEGIN
    -- Get the latest snapshot
    SELECT data INTO lobby_data
    FROM lobby_snapshots
    ORDER BY updated_at DESC
    LIMIT 1;
    
    IF lobby_data IS NULL THEN
        RETURN;
    END IF;
    
    -- Migrate players
    FOR player_key, player_data IN SELECT * FROM jsonb_each(lobby_data->'Players')
    LOOP
        INSERT INTO players (id, nick, token, avatar, created_at, updated_at)
        VALUES (
            player_data->>'ID',
            player_data->>'Nick',
            player_data->>'Token',
            COALESCE(player_data->>'Avatar', ''),
            NOW(),
            NOW()
        )
        ON CONFLICT (id) DO UPDATE SET
            nick = EXCLUDED.nick,
            token = EXCLUDED.token,
            avatar = EXCLUDED.avatar,
            updated_at = NOW();
    END LOOP;
    
    -- Migrate lobbies
    FOR lobby_key, lobby_data IN SELECT * FROM jsonb_each(lobby_data->'Lobbies')
    LOOP
        INSERT INTO lobbies (id, leader_id, status, version, created_at, updated_at)
        VALUES (
            lobby_data->>'ID',
            lobby_data->>'LeaderID',
            COALESCE(lobby_data->>'Status', 'open'),
            1,
            NOW(),
            NOW()
        )
        ON CONFLICT (id) DO UPDATE SET
            leader_id = EXCLUDED.leader_id,
            status = EXCLUDED.status,
            updated_at = NOW();
        
        -- Migrate lobby players
        FOR i IN 0 .. jsonb_array_length(lobby_data->'PlayerIDs') - 1
        LOOP
            INSERT INTO lobby_players (lobby_id, player_id, joined_at, position)
            VALUES (
                lobby_data->>'ID',
                jsonb_array_element_text(lobby_data->'PlayerIDs', i),
                NOW(),
                i
            )
            ON CONFLICT (lobby_id, player_id) DO NOTHING;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Execute migration
SELECT migrate_lobby_snapshots();

-- Drop temporary function
DROP FUNCTION migrate_lobby_snapshots();

-- Keep old table for rollback capability (can be dropped later)
-- ALTER TABLE lobby_snapshots RENAME TO lobby_snapshots_backup;
```

### 5.2 Application Migration Strategy

#### Phase 1: Dual-Write (Week 1)
- Implement new repository layer alongside existing code
- Write to both old (JSON blob) and new (normalized) schemas
- Read from old schema (source of truth)
- Monitor for discrepancies

#### Phase 2: Validation (Week 2)
- Compare reads from both schemas
- Fix any migration issues
- Add metrics/logging for migration health

#### Phase 3: Switchover (Week 3)
- Flip read path to new schema
- Keep dual-write for rollback safety
- Monitor closely

#### Phase 4: Cleanup (Week 4)
- Remove old schema writes
- Archive/drop old table
- Clean up migration code

### 5.3 Backward Compatibility

```go
// infrastructure/persistence/migration_adapter.go
package persistence

import (
	"context"
	
	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
)

// DualWriteLobbyRepository writes to both old and new schemas.
// Used during migration to ensure data consistency.
type DualWriteLobbyRepository struct {
	oldStore server.LobbyStore           // JSON blob store
	newRepo  repositories.LobbyRepository // Normalized repository
}

func (r *DualWriteLobbyRepository) Save(ctx context.Context, lobby *entities.Lobby) error {
	// Always write to new schema
	if err := r.newRepo.Save(ctx, lobby); err != nil {
		return err
	}
	
	// Best-effort write to old schema (for rollback capability)
	// Don't fail if old store fails
	snapshot := convertToSnapshot(lobby)
	_ = r.oldStore.Save(ctx, snapshot)
	
	return nil
}

// Similar pattern for PlayerRepository...
```

---

## 6. Implementation Priority

### Week 1: Foundation
1. **Domain Entities** (`domain/entities/`)
   - Player entity with tests
   - Lobby entity with tests
   - Domain errors

2. **Repository Interfaces** (`domain/repositories/`)
   - PlayerRepository interface
   - LobbyRepository interface
   - UnitOfWork interface

3. **GORM Models** (`infrastructure/persistence/models/`)
   - Player model
   - Lobby model
   - LobbyPlayer junction model

### Week 2: PostgreSQL Implementation
4. **Repository Implementations** (`infrastructure/persistence/repositories/`)
   - PostgresPlayerRepository
   - PostgresLobbyRepository
   - Unit tests with testcontainers

5. **Transaction Management** (`infrastructure/persistence/transactions/`)
   - GormUnitOfWork implementation

6. **Database Migrations**
   - Migration scripts
   - Migration runner

### Week 3: Domain Services
7. **Domain Services** (`domain/services/`)
   - LobbyService with atomic operations
   - Comprehensive integration tests

8. **Application Layer** (`application/`)
   - DTOs
   - Application services

### Week 4: Integration & Migration
9. **Integration**
   - Wire up new repositories to existing LobbyManager
   - Feature flags for gradual rollout

10. **Data Migration**
    - Run migration scripts
    - Validate data integrity
    - Monitor in production

---

## 7. Error Handling Strategy

### 7.1 Error Hierarchy

```
error
├── domain/entities.Err*          # Business rule violations
├── repositories.Err*             # Data access errors
│   ├── ErrNotFound
│   ├── ErrVersionConflict
│   └── ErrDuplicate
└── infrastructure.Err*           # Infrastructure errors
    ├── ErrConnection
    └── ErrTimeout
```

### 7.2 Error Wrapping

```go
// Wrap infrastructure errors with context
if err := db.Create(&model).Error; err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("%w: player with id %s", repositories.ErrNotFound, id)
    }
    return nil, fmt.Errorf("database error creating player: %w", err)
}
```

### 7.3 Error Handling in Handlers

```go
// server/ws/handlers.go
func (s *Server) handleJoinLobby(ctx context.Context, msg JoinLobbyMessage) error {
	err := s.lobbyService.JoinLobby(ctx, entities.LobbyID(msg.LobbyID), entities.PlayerID(msg.PlayerID))
	if err != nil {
		switch {
		case errors.Is(err, entities.ErrLobbyNotFound):
			return s.sendError(ctx, msg.PlayerID, "lobby_not_found", "Lobby does not exist")
		case errors.Is(err, entities.ErrPlayerNotFound):
			return s.sendError(ctx, msg.PlayerID, "player_not_found", "Player not registered")
		case errors.Is(err, entities.ErrLobbyFull):
			return s.sendError(ctx, msg.PlayerID, "lobby_full", "Lobby is at capacity")
		case errors.Is(err, entities.ErrPlayerAlreadyInLobby):
			return s.sendError(ctx, msg.PlayerID, "already_joined", "Already in this lobby")
		default:
			s.logger.Error("failed to join lobby", zap.Error(err))
			return s.sendError(ctx, msg.PlayerID, "internal_error", "Failed to join lobby")
		}
	}
	return s.broadcastLobbyUpdate(ctx, msg.LobbyID)
}
```

---

## 8. Testing Strategy

### 8.1 Unit Tests (Table-Driven)

```go
// domain/entities/lobby_test.go
package entities

import (
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLobby_AddPlayer(t *testing.T) {
	tests := []struct {
		name       string
		lobby      *Lobby
		playerID   PlayerID
		maxPlayers int
		wantErr    error
		wantCount  int
	}{
		{
			name:       "add player to open lobby",
			lobby:      mustCreateLobby("lobby-1", "leader-1"),
			playerID:   "player-2",
			maxPlayers: 4,
			wantErr:    nil,
			wantCount:  2,
		},
		{
			name:       "reject duplicate player",
			lobby:      mustCreateLobbyWithPlayers("lobby-1", "leader-1", []PlayerID{"leader-1", "player-2"}),
			playerID:   "player-2",
			maxPlayers: 4,
			wantErr:    ErrPlayerAlreadyInLobby,
			wantCount:  2,
		},
		{
			name:       "reject when lobby full",
			lobby:      mustCreateLobbyWithPlayers("lobby-1", "leader-1", []PlayerID{"leader-1", "p2", "p3", "p4"}),
			playerID:   "player-5",
			maxPlayers: 4,
			wantErr:    ErrLobbyFull,
			wantCount:  4,
		},
		{
			name:       "reject when lobby in-game",
			lobby:      mustCreateInGameLobby("lobby-1", "leader-1"),
			playerID:   "player-2",
			maxPlayers: 4,
			wantErr:    ErrLobbyNotOpen,
			wantCount:  1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lobby.AddPlayer(tt.playerID, tt.maxPlayers)
			
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			
			assert.Len(t, tt.lobby.Players(), tt.wantCount)
		})
	}
}

func TestLobby_Start(t *testing.T) {
	tests := []struct {
		name       string
		lobby      *Lobby
		leaderID   PlayerID
		minPlayers int
		wantErr    error
		wantStatus LobbyStatus
	}{
		{
			name:       "leader starts with enough players",
			lobby:      mustCreateLobbyWithPlayers("lobby-1", "leader-1", []PlayerID{"leader-1", "p2", "p3"}),
			leaderID:   "leader-1",
			minPlayers: 2,
			wantErr:    nil,
			wantStatus: LobbyStatusInGame,
		},
		{
			name:       "non-leader cannot start",
			lobby:      mustCreateLobbyWithPlayers("lobby-1", "leader-1", []PlayerID{"leader-1", "p2", "p3"}),
			leaderID:   "p2",
			minPlayers: 2,
			wantErr:    ErrNotLobbyLeader,
			wantStatus: LobbyStatusOpen,
		},
		{
			name:       "cannot start with insufficient players",
			lobby:      mustCreateLobby("lobby-1", "leader-1"),
			leaderID:   "leader-1",
			minPlayers: 2,
			wantErr:    ErrNotEnoughPlayers,
			wantStatus: LobbyStatusOpen,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lobby.Start(tt.leaderID, tt.minPlayers)
			
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			
			assert.Equal(t, tt.wantStatus, tt.lobby.Status())
		})
	}
}

// Helper functions
func mustCreateLobby(id, leaderID string) *Lobby {
	l, err := NewLobby(LobbyID(id), PlayerID(leaderID), time.Now())
	if err != nil {
		panic(err)
	}
	return l
}

func mustCreateLobbyWithPlayers(id, leaderID string, players []PlayerID) *Lobby {
	l := mustCreateLobby(id, leaderID)
	for i, p := range players {
		if i == 0 {
			continue // Leader already added
		}
		if err := l.AddPlayer(p, 10); err != nil {
			panic(err)
		}
	}
	return l
}

func mustCreateInGameLobby(id, leaderID string) *Lobby {
	l := mustCreateLobbyWithPlayers(id, leaderID, []PlayerID{leaderID, "p2", "p3"})
	if err := l.Start(PlayerID(leaderID), 2); err != nil {
		panic(err)
	}
	return l
}
```

### 8.2 Repository Integration Tests

```go
// infrastructure/persistence/repositories/postgres_lobby_repository_test.go
package repositories

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	
	"ElMakina/backend/domain/entities"
)

type PostgresLobbyRepositoryTestSuite struct {
	suite.Suite
	ctx        context.Context
	db         *gorm.DB
	container  *postgres.PostgresContainer
	repository *PostgresLobbyRepository
}

func (s *PostgresLobbyRepositoryTestSuite) SetupSuite() {
	s.ctx = context.Background()
	
	// Start PostgreSQL container
	container, err := postgres.Run(s.ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30),
		),
	)
	require.NoError(s.T(), err)
	s.container = container
	
	// Connect to database
	host, err := container.Host(s.ctx)
	require.NoError(s.T(), err)
	port, err := container.MappedPort(s.ctx, "5432")
	require.NoError(s.T(), err)
	
	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=test sslmode=disable", host, port.Port())
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(s.T(), err)
	s.db = db
	
	// Run migrations
	s.db.AutoMigrate(&PlayerModel{}, &LobbyModel{}, &LobbyPlayerModel{})
	
	s.repository = NewPostgresLobbyRepository(s.db)
}

func (s *PostgresLobbyRepositoryTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Terminate(s.ctx)
	}
}

func (s *PostgresLobbyRepositoryTestSuite) SetupTest() {
	// Clean tables before each test
	s.db.Exec("TRUNCATE lobby_players, lobbies, players CASCADE")
}

func (s *PostgresLobbyRepositoryTestSuite) TestSaveAndGet() {
	// Create test player first
	playerRepo := NewPostgresPlayerRepository(s.db)
	player, err := entities.NewPlayer("player-1", "TestPlayer", "token-123", time.Now())
	require.NoError(s.T(), err)
	require.NoError(s.T(), playerRepo.Save(s.ctx, player))
	
	// Create and save lobby
	lobby, err := entities.NewLobby("lobby-1", "player-1", time.Now())
	require.NoError(s.T(), err)
	
	err = s.repository.Save(s.ctx, lobby)
	require.NoError(s.T(), err)
	
	// Retrieve and verify
	loaded, err := s.repository.GetByID(s.ctx, "lobby-1")
	require.NoError(s.T(), err)
	
	assert.Equal(s.T(), lobby.ID(), loaded.ID())
	assert.Equal(s.T(), lobby.LeaderID(), loaded.LeaderID())
	assert.Equal(s.T(), lobby.Status(), loaded.Status())
	assert.Equal(s.T(), 1, loaded.PlayerCount())
}

func (s *PostgresLobbyRepositoryTestSuite) TestOptimisticLocking() {
	// Setup
	playerRepo := NewPostgresPlayerRepository(s.db)
	player, _ := entities.NewPlayer("player-1", "TestPlayer", "token-123", time.Now())
	playerRepo.Save(s.ctx, player)
	
	lobby, _ := entities.NewLobby("lobby-1", "player-1", time.Now())
	s.repository.Save(s.ctx, lobby)
	
	// Simulate concurrent modification
	loaded1, _ := s.repository.GetByID(s.ctx, "lobby-1")
	loaded2, _ := s.repository.GetByID(s.ctx, "lobby-1")
	
	// First save succeeds
	loaded1.AddPlayer("player-2", 4)
	err := s.repository.Save(s.ctx, loaded1)
	require.NoError(s.T(), err)
	
	// Second save fails with version conflict
	loaded2.AddPlayer("player-3", 4)
	err = s.repository.Save(s.ctx, loaded2)
	assert.ErrorIs(s.T(), err, repositories.ErrVersionConflict)
}

func TestPostgresLobbyRepository(t *testing.T) {
	suite.Run(t, new(PostgresLobbyRepositoryTestSuite))
}
```

---

## 9. Example Usage

### 9.1 Current vs New Architecture

**Current (JSON Blob):**
```go
// Direct store access, no transactions
manager := NewLobbyManager(2, 4, postgresStore)
manager.JoinLobby(lobbyID, playerID) // Race condition possible!
```

**New (Repository Pattern):**
```go
// Proper transaction boundaries
uowFactory := transactions.NewGormUnitOfWorkFactory(db)
lobbyService := services.NewLobbyService(uowFactory, 2, 4)

// Atomic operation with proper error handling
err := lobbyService.JoinLobby(ctx, lobbyID, playerID)
if errors.Is(err, entities.ErrLobbyFull) {
    // Handle full lobby
}
```

### 9.2 WebSocket Handler Integration

```go
// server/ws/server.go

// Current approach (simplified)
func (s *Server) handleJoin(msg JoinMessage) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    err := s.lobbyManager.JoinLobby(msg.LobbyID, msg.PlayerID)
    if err != nil {
        s.sendError(msg.PlayerID, err)
        return
    }
    s.broadcastUpdate(msg.LobbyID)
}

// New approach with repository layer
func (s *Server) handleJoin(msg JoinMessage) {
    ctx := context.Background()
    
    err := s.lobbyService.JoinLobby(ctx, 
        entities.LobbyID(msg.LobbyID), 
        entities.PlayerID(msg.PlayerID),
    )
    if err != nil {
        s.handleDomainError(msg.PlayerID, err)
        return
    }
    
    s.broadcastUpdate(ctx, msg.LobbyID)
}

func (s *Server) handleDomainError(playerID string, err error) {
    switch {
    case errors.Is(err, entities.ErrLobbyNotFound):
        s.sendError(playerID, "LOBBY_NOT_FOUND", "Lobby does not exist")
    case errors.Is(err, entities.ErrLobbyFull):
        s.sendError(playerID, "LOBBY_FULL", "Lobby is at capacity")
    case errors.Is(err, entities.ErrPlayerAlreadyInLobby):
        s.sendError(playerID, "ALREADY_JOINED", "You are already in this lobby")
    default:
        s.logger.Error("unexpected error", zap.Error(err))
        s.sendError(playerID, "INTERNAL_ERROR", "An unexpected error occurred")
    }
}
```

---

## 10. Summary

This design provides:

1. **Clean Domain Layer**: Entities with rich behavior, no external dependencies
2. **Repository Pattern**: Abstraction over persistence, swappable implementations
3. **Transaction Boundaries**: Unit of Work pattern for atomic operations
4. **Optimistic Locking**: Version field prevents lost updates
5. **Type Safety**: Strongly-typed IDs prevent mixing up identifiers
6. **Testability**: Interfaces enable mocking, table-driven tests for domain logic
7. **Migration Path**: Dual-write strategy for safe transition
8. **Error Handling**: Clear error hierarchy, proper wrapping

The architecture follows Clean Architecture principles:
- Dependencies point inward
- Domain knows nothing about GORM, HTTP, or WebSockets
- Infrastructure details are isolated
- Business rules are testable in isolation
