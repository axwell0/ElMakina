package models

import "time"

// LobbyModel is the GORM model for the lobbies table.
type LobbyModel struct {
	ID        string    `gorm:"primaryKey;size:64"`
	LeaderID  string    `gorm:"size:64;not null;index"`
	Status    string    `gorm:"size:16;not null;index"`
	Version   int64     `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`

	// Associations
	Players []LobbyPlayerModel `gorm:"foreignKey:LobbyID;references:ID"`
}

// TableName returns the table name for lobby records.
func (LobbyModel) TableName() string {
	return "lobbies"
}
