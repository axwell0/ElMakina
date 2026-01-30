package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
)

// LobbyManagerV2 is a new implementation using the repository pattern.
// It provides transaction safety and optimistic locking.
type LobbyManagerV2 struct {
	playerRepo repositories.PlayerRepository
	lobbyRepo  repositories.LobbyRepository
	unitOfWork repositories.UnitOfWork
	minPlayers int
	maxPlayers int
	mu         sync.RWMutex
	cache      map[entities.LobbyID]*entities.Lobby
}

// NewLobbyManagerV2 creates a new lobby manager with repository dependencies.
func NewLobbyManagerV2(
	minPlayers, maxPlayers int,
	playerRepo repositories.PlayerRepository,
	lobbyRepo repositories.LobbyRepository,
	unitOfWork repositories.UnitOfWork,
) *LobbyManagerV2 {
	return &LobbyManagerV2{
		playerRepo: playerRepo,
		lobbyRepo:  lobbyRepo,
		unitOfWork: unitOfWork,
		minPlayers: minPlayers,
		maxPlayers: maxPlayers,
		cache:      make(map[entities.LobbyID]*entities.Lobby),
	}
}

// CreateLobby creates a new lobby with the given leader.
func (m *LobbyManagerV2) CreateLobby(ctx context.Context, leaderID entities.PlayerID) (*entities.Lobby, error) {
	// Verify leader exists
	leader, err := m.playerRepo.GetByID(ctx, leaderID)
	if err != nil {
		return nil, fmt.Errorf("leader not found: %w", err)
	}

	// Generate lobby ID
	lobbyID := entities.LobbyID(generateID())

	// Create lobby
	lobby, err := entities.NewLobby(lobbyID, leader.ID())
	if err != nil {
		return nil, err
	}

	// Save to repository
	if err := m.lobbyRepo.Save(ctx, lobby); err != nil {
		return nil, fmt.Errorf("failed to save lobby: %w", err)
	}

	// Update cache
	m.mu.Lock()
	m.cache[lobby.ID()] = lobby
	m.mu.Unlock()

	return lobby, nil
}

// JoinLobby adds a player to a lobby atomically.
func (m *LobbyManagerV2) JoinLobby(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) error {
	return m.unitOfWork.Execute(ctx, func(ctx context.Context) error {
		// Get lobby and player within transaction
		lobby, err := m.lobbyRepo.GetByID(ctx, lobbyID)
		if err != nil {
			return err
		}

		player, err := m.playerRepo.GetByID(ctx, playerID)
		if err != nil {
			return err
		}

		// Apply business logic
		if err := lobby.AddPlayer(player.ID(), m.maxPlayers); err != nil {
			return err
		}

		// Save changes
		return m.lobbyRepo.Save(ctx, lobby)
	})
}

// LeaveLobby removes a player from a lobby atomically.
func (m *LobbyManagerV2) LeaveLobby(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) (bool, error) {
	var shouldClose bool
	err := m.unitOfWork.Execute(ctx, func(ctx context.Context) error {
		// Get lobby within transaction
		lobby, err := m.lobbyRepo.GetByID(ctx, lobbyID)
		if err != nil {
			return err
		}

		// Apply business logic
		shouldClose, err = lobby.RemovePlayer(playerID)
		if err != nil {
			return err
		}

		// Save changes
		return m.lobbyRepo.Save(ctx, lobby)
	})

	return shouldClose, err
}

// StartLobby transitions a lobby to in-game status atomically.
func (m *LobbyManagerV2) StartLobby(ctx context.Context, lobbyID entities.LobbyID, leaderID entities.PlayerID) error {
	return m.unitOfWork.Execute(ctx, func(ctx context.Context) error {
		// Get lobby within transaction
		lobby, err := m.lobbyRepo.GetByID(ctx, lobbyID)
		if err != nil {
			return err
		}

		// Verify leader
		if err := lobby.CanStart(leaderID, m.minPlayers); err != nil {
			return err
		}

		// Start game
		if err := lobby.StartGame(m.minPlayers); err != nil {
			return err
		}

		// Save changes
		return m.lobbyRepo.Save(ctx, lobby)
	})
}

// GetLobby retrieves a lobby by ID.
func (m *LobbyManagerV2) GetLobby(ctx context.Context, id entities.LobbyID) (*entities.Lobby, error) {
	// Check cache first
	m.mu.RLock()
	if lobby, ok := m.cache[id]; ok {
		m.mu.RUnlock()
		return lobby, nil
	}
	m.mu.RUnlock()

	// Load from repository
	lobby, err := m.lobbyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update cache
	m.mu.Lock()
	m.cache[id] = lobby
	m.mu.Unlock()

	return lobby, nil
}

// ListLobbies returns all open lobbies.
func (m *LobbyManagerV2) ListLobbies(ctx context.Context) ([]*entities.Lobby, error) {
	status := entities.LobbyOpen
	return m.lobbyRepo.List(ctx, repositories.LobbyListFilter{
		Status: &status,
	})
}

// generateID creates a unique identifier.
func generateID() string {
	// Simple implementation - in production use UUID
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
