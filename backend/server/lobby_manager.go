package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
	"ElMakina/backend/domain/services"
)

// LobbyManager coordinates lobby and player operations.
// It provides a unified interface for the WebSocket server.
type LobbyManager struct {
	playerService services.PlayerService
	lobbyRepo     repositories.LobbyRepository
	unitOfWork    repositories.UnitOfWork
	minPlayers    int
	maxPlayers    int
	mu            sync.RWMutex
	lobbyCache    map[entities.LobbyID]*entities.Lobby
}

// NewLobbyManager creates a new lobby manager with all dependencies.
func NewLobbyManager(
	minPlayers, maxPlayers int,
	playerService services.PlayerService,
	lobbyRepo repositories.LobbyRepository,
	unitOfWork repositories.UnitOfWork,
) *LobbyManager {
	return &LobbyManager{
		playerService: playerService,
		lobbyRepo:     lobbyRepo,
		unitOfWork:    unitOfWork,
		minPlayers:    minPlayers,
		maxPlayers:    maxPlayers,
		lobbyCache:    make(map[entities.LobbyID]*entities.Lobby),
	}
}

// RegisterPlayer creates a new player with the given nickname.
func (m *LobbyManager) RegisterPlayer(ctx context.Context, nick string) (*entities.Player, error) {
	return m.playerService.RegisterPlayer(ctx, nick)
}

// GetPlayerByID retrieves a player by their ID.
func (m *LobbyManager) GetPlayerByID(ctx context.Context, id entities.PlayerID) (*entities.Player, error) {
	return m.playerService.GetPlayerByID(ctx, id)
}

// ReconnectPlayer retrieves a player by their reconnect token.
func (m *LobbyManager) ReconnectPlayer(ctx context.Context, token string) (*entities.Player, error) {
	return m.playerService.ReconnectPlayer(ctx, token)
}

// SetPlayerAvatar updates a player's avatar.
func (m *LobbyManager) SetPlayerAvatar(ctx context.Context, id entities.PlayerID, avatar string) error {
	return m.playerService.SetPlayerAvatar(ctx, id, avatar)
}

// UnregisterPlayer removes a player from the system.
func (m *LobbyManager) UnregisterPlayer(ctx context.Context, id entities.PlayerID) error {
	return m.playerService.UnregisterPlayer(ctx, id)
}

// CreateLobby creates a new lobby with the given leader.
func (m *LobbyManager) CreateLobby(ctx context.Context, leaderID entities.PlayerID) (*entities.Lobby, error) {
	// Verify leader exists
	leader, err := m.playerService.GetPlayerByID(ctx, leaderID)
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
	m.lobbyCache[lobby.ID()] = lobby
	m.mu.Unlock()

	return lobby, nil
}

// JoinLobby adds a player to a lobby atomically.
func (m *LobbyManager) JoinLobby(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) error {
	return m.unitOfWork.Execute(ctx, func(ctx context.Context) error {
		// Get lobby and player within transaction
		lobby, err := m.lobbyRepo.GetByID(ctx, lobbyID)
		if err != nil {
			return err
		}

		player, err := m.playerService.GetPlayerByID(ctx, playerID)
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
func (m *LobbyManager) LeaveLobby(ctx context.Context, lobbyID entities.LobbyID, playerID entities.PlayerID) (bool, error) {
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
func (m *LobbyManager) StartLobby(ctx context.Context, lobbyID entities.LobbyID, leaderID entities.PlayerID) error {
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
func (m *LobbyManager) GetLobby(ctx context.Context, id entities.LobbyID) (*entities.Lobby, error) {
	// Check cache first
	m.mu.RLock()
	if lobby, ok := m.lobbyCache[id]; ok {
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
	m.lobbyCache[id] = lobby
	m.mu.Unlock()

	return lobby, nil
}

// ListLobbies returns all open lobbies.
func (m *LobbyManager) ListLobbies(ctx context.Context) ([]*entities.Lobby, error) {
	status := entities.LobbyOpen
	return m.lobbyRepo.List(ctx, repositories.LobbyListFilter{
		Status: &status,
	})
}

// GetLobbyByPlayerID finds the lobby containing the given player.
func (m *LobbyManager) GetLobbyByPlayerID(ctx context.Context, playerID entities.PlayerID) (*entities.Lobby, error) {
	return m.lobbyRepo.GetByPlayerID(ctx, playerID)
}

// PruneEmptyLobbies removes lobbies that have been empty for longer than the given duration.
func (m *LobbyManager) PruneEmptyLobbies(ctx context.Context, olderThan time.Duration) ([]entities.LobbyID, error) {
	// Get all open lobbies
	status := entities.LobbyOpen
	lobbies, err := m.lobbyRepo.List(ctx, repositories.LobbyListFilter{
		Status: &status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list lobbies: %w", err)
	}

	var removed []entities.LobbyID
	cutoff := time.Now().UTC().Add(-olderThan)

	for _, lobby := range lobbies {
		// Check if lobby is empty and old enough
		if lobby.PlayerCount() == 0 && lobby.CreatedAt().Before(cutoff) {
			if err := m.lobbyRepo.Delete(ctx, lobby.ID()); err != nil {
				// Log error but continue
				continue
			}
			removed = append(removed, lobby.ID())
		}
	}

	return removed, nil
}

// generateID creates a unique identifier.
func generateID() string {
	// Simple implementation - in production use UUID
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
