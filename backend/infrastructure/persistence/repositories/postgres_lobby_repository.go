package repositories

import (
	"context"
	"fmt"

	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
	"ElMakina/backend/infrastructure/persistence/mappers"
	"ElMakina/backend/infrastructure/persistence/models"
	"ElMakina/backend/infrastructure/persistence/transactions"
	"gorm.io/gorm"
)

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

// GetByID retrieves a lobby by ID without player details.
func (r *postgresLobbyRepository) GetByID(ctx context.Context, id entities.LobbyID) (*entities.Lobby, error) {
	var model models.LobbyModel
	result := r.getDB(ctx).First(&model, "id = ?", id.String())
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("lobby not found: %w", entities.ErrLobbyNotFound)
		}
		return nil, fmt.Errorf("failed to get lobby: %w", result.Error)
	}

	// Load player associations
	var playerModels []models.LobbyPlayerModel
	r.getDB(ctx).Where("lobby_id = ?", model.ID).Order("joined_at ASC").Find(&playerModels)

	return mappers.LobbyToEntity(&model, playerModels), nil
}

// GetByIDWithPlayers retrieves a lobby with all player details populated.
func (r *postgresLobbyRepository) GetByIDWithPlayers(ctx context.Context, id entities.LobbyID) (*repositories.LobbyWithPlayers, error) {
	var model models.LobbyModel
	result := r.getDB(ctx).Preload("Players").Preload("Players.Player").First(&model, "id = ?", id.String())
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("lobby not found: %w", entities.ErrLobbyNotFound)
		}
		return nil, fmt.Errorf("failed to get lobby: %w", result.Error)
	}

	// Convert lobby
	lobby := mappers.LobbyToEntity(&model, model.Players)

	// Convert players
	players := make([]*entities.Player, len(model.Players))
	for i, pm := range model.Players {
		players[i] = mappers.PlayerToEntity(&pm.Player)
	}

	return &repositories.LobbyWithPlayers{
		Lobby:   lobby,
		Players: players,
	}, nil
}

// GetByPlayerID finds the lobby containing the given player.
func (r *postgresLobbyRepository) GetByPlayerID(ctx context.Context, playerID entities.PlayerID) (*entities.Lobby, error) {
	var lobbyPlayer models.LobbyPlayerModel
	result := r.getDB(ctx).Where("player_id = ?", playerID.String()).First(&lobbyPlayer)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("player not in any lobby: %w", entities.ErrLobbyNotFound)
		}
		return nil, fmt.Errorf("failed to find lobby: %w", result.Error)
	}

	return r.GetByID(ctx, entities.LobbyID(lobbyPlayer.LobbyID))
}

// List retrieves lobbies matching the filter criteria.
func (r *postgresLobbyRepository) List(ctx context.Context, filter repositories.LobbyListFilter) ([]*entities.Lobby, error) {
	db := r.getDB(ctx).Model(&models.LobbyModel{})

	// Apply filters
	if filter.Status != nil {
		db = db.Where("status = ?", string(*filter.Status))
	}

	// Apply pagination
	if filter.Limit > 0 {
		db = db.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		db = db.Offset(filter.Offset)
	}

	var models_list []models.LobbyModel
	result := db.Order("created_at DESC").Find(&models_list)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list lobbies: %w", result.Error)
	}

	// Convert to entities
	lobbies := make([]*entities.Lobby, len(models_list))
	for i, model := range models_list {
		// Load player associations
		var playerModels []models.LobbyPlayerModel
		r.getDB(ctx).Where("lobby_id = ?", model.ID).Order("joined_at ASC").Find(&playerModels)
		lobbies[i] = mappers.LobbyToEntity(&model, playerModels)
	}

	return lobbies, nil
}

// ListWithPlayerCounts retrieves lobbies with player counts for listing.
func (r *postgresLobbyRepository) ListWithPlayerCounts(ctx context.Context, filter repositories.LobbyListFilter) ([]repositories.LobbySummary, error) {
	db := r.getDB(ctx).Model(&models.LobbyModel{})

	// Apply filters
	if filter.Status != nil {
		db = db.Where("status = ?", string(*filter.Status))
	}

	// Apply pagination
	if filter.Limit > 0 {
		db = db.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		db = db.Offset(filter.Offset)
	}

	var models_list []models.LobbyModel
	result := db.Order("created_at DESC").Find(&models_list)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list lobbies: %w", result.Error)
	}

	// Build summaries
	summaries := make([]repositories.LobbySummary, len(models_list))
	for i, model := range models_list {
		// Load player associations
		var playerModels []models.LobbyPlayerModel
		r.getDB(ctx).Where("lobby_id = ?", model.ID).Order("joined_at ASC").Find(&playerModels)

		// Load leader nick
		var leader models.PlayerModel
		r.getDB(ctx).First(&leader, "id = ?", model.LeaderID)

		playerNicks := make([]string, len(playerModels))
		for j, pm := range playerModels {
			var player models.PlayerModel
			r.getDB(ctx).First(&player, "id = ?", pm.PlayerID)
			playerNicks[j] = player.Nick
		}

		summaries[i] = repositories.LobbySummary{
			ID:          entities.LobbyID(model.ID),
			LeaderID:    entities.PlayerID(model.LeaderID),
			LeaderNick:  leader.Nick,
			PlayerCount: len(playerModels),
			PlayerNicks: playerNicks,
			Status:      entities.LobbyStatus(model.Status),
			CreatedAt:   model.CreatedAt.Unix(),
		}
	}

	return summaries, nil
}

// Save persists a lobby with optimistic locking.
func (r *postgresLobbyRepository) Save(ctx context.Context, lobby *entities.Lobby) error {
	db := r.getDB(ctx)

	// Check if lobby exists
	var existing models.LobbyModel
	result := db.First(&existing, "id = ?", lobby.ID().String())

	if result.Error == gorm.ErrRecordNotFound {
		// Insert new lobby
		model := mappers.LobbyToModel(lobby)
		if err := db.Create(model).Error; err != nil {
			return fmt.Errorf("failed to create lobby: %w", err)
		}

		// Insert player associations
		for _, pid := range lobby.PlayerIDs() {
			lobbyPlayer := models.LobbyPlayerModel{
				LobbyID:  lobby.ID().String(),
				PlayerID: pid.String(),
			}
			if err := db.Create(&lobbyPlayer).Error; err != nil {
				return fmt.Errorf("failed to create lobby player: %w", err)
			}
		}
		return nil
	}

	if result.Error != nil {
		return fmt.Errorf("failed to check lobby existence: %w", result.Error)
	}

	// Update with optimistic locking
	result = db.Model(&models.LobbyModel{}).
		Where("id = ? AND version = ?", lobby.ID().String(), lobby.Version()).
		Updates(map[string]any{
			"leader_id":  lobby.LeaderID().String(),
			"status":     string(lobby.Status()),
			"version":    lobby.Version() + 1,
			"updated_at": lobby.UpdatedAt(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update lobby: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("lobby modified by another transaction: %w", entities.ErrConcurrentModification)
	}

	// Update player associations (delete and re-insert for simplicity)
	db.Where("lobby_id = ?", lobby.ID().String()).Delete(&models.LobbyPlayerModel{})
	for _, pid := range lobby.PlayerIDs() {
		lobbyPlayer := models.LobbyPlayerModel{
			LobbyID:  lobby.ID().String(),
			PlayerID: pid.String(),
		}
		if err := db.Create(&lobbyPlayer).Error; err != nil {
			return fmt.Errorf("failed to create lobby player: %w", err)
		}
	}

	return nil
}

// Delete removes a lobby and all its player associations.
func (r *postgresLobbyRepository) Delete(ctx context.Context, id entities.LobbyID) error {
	// Cascading delete will handle lobby_players due to foreign key constraint
	result := r.getDB(ctx).Delete(&models.LobbyModel{}, "id = ?", id.String())
	if result.Error != nil {
		return fmt.Errorf("failed to delete lobby: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("lobby not found: %w", entities.ErrLobbyNotFound)
	}
	return nil
}

// Exists checks if a lobby with the given ID exists.
func (r *postgresLobbyRepository) Exists(ctx context.Context, id entities.LobbyID) (bool, error) {
	var count int64
	result := r.getDB(ctx).Model(&models.LobbyModel{}).Where("id = ?", id.String()).Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("failed to check lobby existence: %w", result.Error)
	}
	return count > 0, nil
}
