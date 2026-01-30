package mappers

import (
	"ElMakina/backend/domain/entities"
	"ElMakina/backend/infrastructure/persistence/models"
)

// LobbyToModel converts a domain Lobby to a GORM model.
func LobbyToModel(lobby *entities.Lobby) *models.LobbyModel {
	return &models.LobbyModel{
		ID:        lobby.ID().String(),
		LeaderID:  lobby.LeaderID().String(),
		Status:    string(lobby.Status()),
		Version:   lobby.Version(),
		CreatedAt: lobby.CreatedAt(),
		UpdatedAt: lobby.UpdatedAt(),
	}
}

// LobbyToEntity converts a GORM model to a domain Lobby.
func LobbyToEntity(model *models.LobbyModel, playerModels []models.LobbyPlayerModel) *entities.Lobby {
	// Convert player associations
	players := make([]entities.LobbyPlayer, len(playerModels))
	for i, pm := range playerModels {
		players[i] = entities.LobbyPlayer{
			PlayerID: entities.PlayerID(pm.PlayerID),
			JoinedAt: pm.JoinedAt,
		}
	}

	return entities.ReconstituteLobby(
		entities.LobbyID(model.ID),
		entities.PlayerID(model.LeaderID),
		players,
		entities.LobbyStatus(model.Status),
		model.Version,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

// LobbyPlayersToModels converts player associations to GORM models.
func LobbyPlayersToModels(lobbyID entities.LobbyID, playerIDs []entities.PlayerID, joinedAt []interface{}) []models.LobbyPlayerModel {
	// Note: joinedAt is unused in this simplified version
	// In production, you'd track actual join times
	models_list := make([]models.LobbyPlayerModel, len(playerIDs))
	for i, pid := range playerIDs {
		models_list[i] = models.LobbyPlayerModel{
			LobbyID:  lobbyID.String(),
			PlayerID: pid.String(),
		}
	}
	return models_list
}
