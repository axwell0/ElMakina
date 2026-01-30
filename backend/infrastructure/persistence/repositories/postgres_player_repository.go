package repositories

import (
	"context"
	"fmt"
	"time"

	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
	"ElMakina/backend/infrastructure/persistence/mappers"
	"ElMakina/backend/infrastructure/persistence/models"
	"ElMakina/backend/infrastructure/persistence/transactions"
	"gorm.io/gorm"
)

// postgresPlayerRepository implements repositories.PlayerRepository.
type postgresPlayerRepository struct {
	db *gorm.DB
}

// NewPostgresPlayerRepository creates a new PostgreSQL player repository.
func NewPostgresPlayerRepository(db *gorm.DB) repositories.PlayerRepository {
	return &postgresPlayerRepository{db: db}
}

// getDB returns the appropriate DB instance (transaction or default).
func (r *postgresPlayerRepository) getDB(ctx context.Context) *gorm.DB {
	return transactions.DBFromContext(ctx, r.db)
}

// GetByID retrieves a player by their unique ID.
func (r *postgresPlayerRepository) GetByID(ctx context.Context, id entities.PlayerID) (*entities.Player, error) {
	var model models.PlayerModel
	result := r.getDB(ctx).First(&model, "id = ?", id.String())
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("player not found: %w", entities.ErrPlayerNotFound)
		}
		return nil, fmt.Errorf("failed to get player: %w", result.Error)
	}
	return mappers.PlayerToEntity(&model), nil
}

// GetByToken retrieves a player by their secret reconnection token.
func (r *postgresPlayerRepository) GetByToken(ctx context.Context, token string) (*entities.Player, error) {
	var model models.PlayerModel
	result := r.getDB(ctx).First(&model, "token = ?", token)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("player not found: %w", entities.ErrPlayerNotFound)
		}
		return nil, fmt.Errorf("failed to get player by token: %w", result.Error)
	}
	return mappers.PlayerToEntity(&model), nil
}

// GetByIDs retrieves multiple players by their IDs.
func (r *postgresPlayerRepository) GetByIDs(ctx context.Context, ids []entities.PlayerID) ([]*entities.Player, error) {
	if len(ids) == 0 {
		return []*entities.Player{}, nil
	}

	// Convert IDs to strings
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	var models_list []models.PlayerModel
	result := r.getDB(ctx).Where("id IN ?", idStrings).Find(&models_list)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get players: %w", result.Error)
	}

	// Convert to entities
	players := make([]*entities.Player, len(models_list))
	for i, model := range models_list {
		players[i] = mappers.PlayerToEntity(&model)
	}
	return players, nil
}

// Save persists a player (insert or update).
func (r *postgresPlayerRepository) Save(ctx context.Context, player *entities.Player) error {
	model := mappers.PlayerToModel(player)

	// Use upsert (ON CONFLICT DO UPDATE)
	result := r.getDB(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to save player: %w", result.Error)
	}

	return nil
}

// Delete removes a player permanently.
func (r *postgresPlayerRepository) Delete(ctx context.Context, id entities.PlayerID) error {
	result := r.getDB(ctx).Delete(&models.PlayerModel{}, "id = ?", id.String())
	if result.Error != nil {
		return fmt.Errorf("failed to delete player: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("player not found: %w", entities.ErrPlayerNotFound)
	}
	return nil
}

// Exists checks if a player with the given ID exists.
func (r *postgresPlayerRepository) Exists(ctx context.Context, id entities.PlayerID) (bool, error) {
	var count int64
	result := r.getDB(ctx).Model(&models.PlayerModel{}).Where("id = ?", id.String()).Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("failed to check player existence: %w", result.Error)
	}
	return count > 0, nil
}

// ListInactive retrieves players who have been inactive since the given time.
func (r *postgresPlayerRepository) ListInactive(ctx context.Context, since time.Time) ([]*entities.Player, error) {
	var modelsList []models.PlayerModel
	result := r.getDB(ctx).Where("last_active_at < ?", since).Find(&modelsList)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list inactive players: %w", result.Error)
	}

	players := make([]*entities.Player, len(modelsList))
	for i, model := range modelsList {
		players[i] = mappers.PlayerToEntity(&model)
	}
	return players, nil
}

// DeleteBatch removes multiple players in a single operation.
func (r *postgresPlayerRepository) DeleteBatch(ctx context.Context, ids []entities.PlayerID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	result := r.getDB(ctx).Where("id IN ?", idStrings).Delete(&models.PlayerModel{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete players: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}
