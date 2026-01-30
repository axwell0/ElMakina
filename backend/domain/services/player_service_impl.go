package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
	"github.com/google/uuid"
)

// playerService implements PlayerService with transaction safety.
type playerService struct {
	playerRepo     repositories.PlayerRepository
	lobbyRepo      repositories.LobbyRepository
	unitOfWork     repositories.UnitOfWork
	tokenGenerator TokenGenerator
	idGenerator    IDGenerator
}

// NewPlayerService creates a new player service with the given dependencies.
func NewPlayerService(
	playerRepo repositories.PlayerRepository,
	lobbyRepo repositories.LobbyRepository,
	unitOfWork repositories.UnitOfWork,
	tokenGenerator TokenGenerator,
	idGenerator IDGenerator,
) PlayerService {
	// Use default implementations if not provided
	if tokenGenerator == nil {
		tokenGenerator = &cryptoTokenGenerator{}
	}
	if idGenerator == nil {
		idGenerator = &uuidGenerator{}
	}

	return &playerService{
		playerRepo:     playerRepo,
		lobbyRepo:      lobbyRepo,
		unitOfWork:     unitOfWork,
		tokenGenerator: tokenGenerator,
		idGenerator:    idGenerator,
	}
}

// RegisterPlayer creates a new player with the given nickname.
func (s *playerService) RegisterPlayer(ctx context.Context, nick string) (*entities.Player, error) {
	// Validate nickname
	if err := validateNickname(nick); err != nil {
		return nil, err
	}

	// Generate token and ID
	token, err := s.tokenGenerator.GeneratePlayerToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	id := s.idGenerator.GeneratePlayerID()

	// Create player entity
	player, err := entities.NewPlayer(id, nick, token)
	if err != nil {
		return nil, err
	}

	// Persist player
	if err := s.playerRepo.Save(ctx, player); err != nil {
		return nil, fmt.Errorf("failed to save player: %w", err)
	}

	return player, nil
}

// SetPlayerAvatar updates a player's avatar.
func (s *playerService) SetPlayerAvatar(ctx context.Context, playerID entities.PlayerID, avatar string) error {
	// Validate avatar (optional field, just check length)
	if len(avatar) > 255 {
		return fmt.Errorf("avatar too long (max 255): %w", entities.ErrInvalidEntity)
	}

	// Get player
	player, err := s.playerRepo.GetByID(ctx, playerID)
	if err != nil {
		return err
	}

	// Update avatar
	player.SetAvatar(avatar)

	// Save changes
	if err := s.playerRepo.Save(ctx, player); err != nil {
		return fmt.Errorf("failed to save player: %w", err)
	}

	return nil
}

// ReconnectPlayer authenticates a player using their reconnection token.
func (s *playerService) ReconnectPlayer(ctx context.Context, token string) (*entities.Player, error) {
	if token == "" {
		return nil, entities.ErrInvalidToken
	}

	// Look up player by token
	player, err := s.playerRepo.GetByToken(ctx, token)
	if err != nil {
		if entities.IsNotFound(err) {
			return nil, entities.ErrInvalidToken
		}
		return nil, err
	}

	// Update activity timestamp
	player.TouchActivity()

	// Save changes
	if err := s.playerRepo.Save(ctx, player); err != nil {
		return nil, fmt.Errorf("failed to update player activity: %w", err)
	}

	return player, nil
}

// UnregisterPlayer removes a player from the system.
func (s *playerService) UnregisterPlayer(ctx context.Context, playerID entities.PlayerID) error {
	return s.unitOfWork.Execute(ctx, func(ctx context.Context) error {
		// Check if player exists
		exists, err := s.playerRepo.Exists(ctx, playerID)
		if err != nil {
			return err
		}
		if !exists {
			return entities.ErrPlayerNotFound
		}

		// Check if player is in any lobby
		_, err = s.lobbyRepo.GetByPlayerID(ctx, playerID)
		if err == nil {
			// Player is in a lobby
			return entities.ErrPlayerInLobby
		}
		if !entities.IsNotFound(err) {
			// Unexpected error
			return err
		}

		// Safe to delete
		if err := s.playerRepo.Delete(ctx, playerID); err != nil {
			return fmt.Errorf("failed to delete player: %w", err)
		}

		return nil
	})
}

// GetPlayerByID retrieves a player by their unique ID.
func (s *playerService) GetPlayerByID(ctx context.Context, playerID entities.PlayerID) (*entities.Player, error) {
	return s.playerRepo.GetByID(ctx, playerID)
}

// PruneInactivePlayers removes players who have been inactive longer than the given duration.
func (s *playerService) PruneInactivePlayers(ctx context.Context, olderThan time.Duration) ([]entities.PlayerID, error) {
	cutoffTime := time.Now().UTC().Add(-olderThan)

	// Get inactive players
	inactivePlayers, err := s.playerRepo.ListInactive(ctx, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to list inactive players: %w", err)
	}

	if len(inactivePlayers) == 0 {
		return []entities.PlayerID{}, nil
	}

	// Filter out players who are in lobbies
	var playersToDelete []entities.PlayerID
	for _, player := range inactivePlayers {
		_, err := s.lobbyRepo.GetByPlayerID(ctx, player.ID())
		if entities.IsNotFound(err) {
			// Player is not in a lobby, safe to delete
			playersToDelete = append(playersToDelete, player.ID())
		}
		// If player is in a lobby or there's an error, skip them
	}

	if len(playersToDelete) == 0 {
		return []entities.PlayerID{}, nil
	}

	// Delete in batch
	_, err = s.playerRepo.DeleteBatch(ctx, playersToDelete)
	if err != nil {
		return nil, fmt.Errorf("failed to delete inactive players: %w", err)
	}

	return playersToDelete, nil
}

// TouchPlayerActivity updates the last active timestamp for a player.
func (s *playerService) TouchPlayerActivity(ctx context.Context, playerID entities.PlayerID) error {
	player, err := s.playerRepo.GetByID(ctx, playerID)
	if err != nil {
		return err
	}

	player.TouchActivity()

	if err := s.playerRepo.Save(ctx, player); err != nil {
		return fmt.Errorf("failed to update player activity: %w", err)
	}

	return nil
}

// validateNickname checks if a nickname is valid.
func validateNickname(nick string) error {
	if nick == "" {
		return entities.ErrNicknameEmpty
	}
	if len(nick) > 32 {
		return entities.ErrNicknameTooLong
	}
	return nil
}

// cryptoTokenGenerator generates cryptographically secure random tokens.
type cryptoTokenGenerator struct{}

// GeneratePlayerToken creates a 32-byte hex-encoded random token.
func (g *cryptoTokenGenerator) GeneratePlayerToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// uuidGenerator generates UUID-based player IDs.
type uuidGenerator struct{}

// GeneratePlayerID creates a new UUID-based player ID.
func (g *uuidGenerator) GeneratePlayerID() entities.PlayerID {
	return entities.PlayerID(uuid.New().String())
}
