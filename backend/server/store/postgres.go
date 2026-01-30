package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ElMakina/backend/server"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LobbyRecord is the database model for lobby snapshots.
type LobbyRecord struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Data      datatypes.JSON `gorm:"type:jsonb;not null"`
}

// TableName returns the table name for lobby records.
func (LobbyRecord) TableName() string {
	return "lobby_snapshots"
}

// PostgresLobbyStore implements server.LobbyStore using PostgreSQL.
type PostgresLobbyStore struct {
	db *gorm.DB
}

// NewPostgresLobbyStore creates a new PostgreSQL-backed lobby store.
func NewPostgresLobbyStore(db *gorm.DB) *PostgresLobbyStore {
	return &PostgresLobbyStore{db: db}
}

// AutoMigrate creates the necessary database tables.
func (s *PostgresLobbyStore) AutoMigrate(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(&LobbyRecord{})
}

// Load retrieves the latest lobby snapshot from the database.
func (s *PostgresLobbyStore) Load(ctx context.Context) (*server.LobbySnapshot, error) {
	var record LobbyRecord
	result := s.db.WithContext(ctx).Order("updated_at desc").First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load lobby snapshot: %w", result.Error)
	}

	var snapshot server.LobbySnapshot
	if err := json.Unmarshal(record.Data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lobby snapshot: %w", err)
	}

	return &snapshot, nil
}

// Save persists the lobby snapshot to the database.
func (s *PostgresLobbyStore) Save(ctx context.Context, snapshot server.LobbySnapshot) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal lobby snapshot: %w", err)
	}

	record := LobbyRecord{
		Data: data,
	}

	// Try to update existing record, or create new one
	result := s.db.WithContext(ctx).Order("updated_at desc").First(&LobbyRecord{})
	if result.Error == nil {
		// Update existing
		result = s.db.WithContext(ctx).Model(&LobbyRecord{}).Order("updated_at desc").Update("data", data)
	} else {
		// Create new
		result = s.db.WithContext(ctx).Create(&record)
	}

	if result.Error != nil {
		return fmt.Errorf("failed to save lobby snapshot: %w", result.Error)
	}

	return nil
}
