package transactions

import (
	"context"

	"ElMakina/backend/domain/repositories"
	"gorm.io/gorm"
)

// GormUnitOfWork implements UnitOfWork using GORM transactions.
type GormUnitOfWork struct {
	db *gorm.DB
}

// NewGormUnitOfWork creates a new UnitOfWork bound to a GORM database.
func NewGormUnitOfWork(db *gorm.DB) repositories.UnitOfWork {
	return &GormUnitOfWork{db: db}
}

// Execute implements the UnitOfWork interface.
func (u *GormUnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create a context with the transaction
		txCtx := ContextWithDB(ctx, tx)
		return fn(txCtx)
	})
}

// contextKey is a private type for context keys to avoid collisions.
type contextKey struct{ name string }

var dbContextKey = &contextKey{"db"}

// ContextWithDB injects a GORM DB into the context.
func ContextWithDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbContextKey, db)
}

// DBFromContext extracts a GORM DB from the context.
// Returns the default DB if no transaction is in progress.
func DBFromContext(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if db, ok := ctx.Value(dbContextKey).(*gorm.DB); ok {
		return db
	}
	return defaultDB
}
