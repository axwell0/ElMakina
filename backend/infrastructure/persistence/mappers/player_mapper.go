package mappers

import (
	"ElMakina/backend/domain/entities"
	"ElMakina/backend/infrastructure/persistence/models"
)

// PlayerToModel converts a domain Player to a GORM model.
func PlayerToModel(player *entities.Player) *models.PlayerModel {
	return &models.PlayerModel{
		ID:           player.ID().String(),
		Nick:         player.Nick(),
		Token:        player.Token(),
		Avatar:       player.Avatar(),
		CreatedAt:    player.CreatedAt(),
		UpdatedAt:    player.UpdatedAt(),
		LastActiveAt: player.LastActiveAt(),
	}
}

// PlayerToEntity converts a GORM model to a domain Player.
func PlayerToEntity(model *models.PlayerModel) *entities.Player {
	return entities.ReconstitutePlayer(
		entities.PlayerID(model.ID),
		model.Nick,
		model.Token,
		model.Avatar,
		model.CreatedAt,
		model.UpdatedAt,
		model.LastActiveAt,
	)
}

// PlayersToEntities converts a slice of GORM models to domain Players.
func PlayersToEntities(models []*models.PlayerModel) []*entities.Player {
	players := make([]*entities.Player, len(models))
	for i, model := range models {
		players[i] = PlayerToEntity(model)
	}
	return players
}
