package models

import "time"

// MigrationStatus tracks the state of database migrations.
type MigrationStatus struct {
	Key       string    `gorm:"primaryKey;size:64"`
	Value     string    `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName returns the table name for migration status.
func (MigrationStatus) TableName() string {
	return "_migration_status"
}
