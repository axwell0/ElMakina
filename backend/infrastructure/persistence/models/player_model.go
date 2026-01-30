package models

import "time"

// PlayerModel is the GORM model for the players table.
type PlayerModel struct {
	ID        string    `gorm:"primaryKey;size:64"`
	Nick      string    `gorm:"size:32;not null"`
	Token     string    `gorm:"size:64;not null;uniqueIndex"`
	Avatar    string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName returns the table name for player records.
func (PlayerModel) TableName() string {
	return "players"
}
