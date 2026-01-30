package models

import "time"

// LobbyPlayerModel is the GORM model for the lobby_players junction table.
type LobbyPlayerModel struct {
	LobbyID  string    `gorm:"primaryKey;size:64"`
	PlayerID string    `gorm:"primaryKey;size:64"`
	JoinedAt time.Time `gorm:"not null"`

	// Associations
	Lobby  LobbyModel  `gorm:"foreignKey:LobbyID;references:ID"`
	Player PlayerModel `gorm:"foreignKey:PlayerID;references:ID"`
}

// TableName returns the table name for lobby player records.
func (LobbyPlayerModel) TableName() string {
	return "lobby_players"
}
