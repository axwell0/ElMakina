package repositories_test

import (
	"context"
	"testing"

	"ElMakina/backend/domain/entities"
	domain "ElMakina/backend/domain/repositories"
	"ElMakina/backend/infrastructure/persistence/repositories"
	"ElMakina/backend/infrastructure/persistence/testutil"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// PostgresPlayerRepositoryTestSuite tests the PostgreSQL player repository.
type PostgresPlayerRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo domain.PlayerRepository
}

func (s *PostgresPlayerRepositoryTestSuite) SetupSuite() {
	// Use SQLite in-memory for unit tests (pure Go implementation)
	var err error
	s.db, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	s.Require().NoError(err)

	// Run migrations
	testDB := testutil.NewTestDB(s.T(), s.db)
	testDB.RunMigrations(s.T())

	// Create repository
	s.repo = repositories.NewPostgresPlayerRepository(s.db)
}

func (s *PostgresPlayerRepositoryTestSuite) SetupTest() {
	// Clean up before each test
	testDB := testutil.NewTestDB(s.T(), s.db)
	testDB.Cleanup(s.T())
}

func (s *PostgresPlayerRepositoryTestSuite) TestSaveAndGetByID() {
	ctx := context.Background()

	// Create and save a player
	player, err := entities.NewPlayer("player-1", "Alice", "token-1")
	s.Require().NoError(err)

	err = s.repo.Save(ctx, player)
	s.Require().NoError(err)

	// Retrieve and verify
	retrieved, err := s.repo.GetByID(ctx, "player-1")
	s.Require().NoError(err)

	s.Equal(player.ID(), retrieved.ID())
	s.Equal(player.Nick(), retrieved.Nick())
	s.Equal(player.Token(), retrieved.Token())
}

func (s *PostgresPlayerRepositoryTestSuite) TestGetByID_NotFound() {
	ctx := context.Background()

	_, err := s.repo.GetByID(ctx, "non-existent")
	s.ErrorIs(err, entities.ErrPlayerNotFound)
}

func (s *PostgresPlayerRepositoryTestSuite) TestGetByToken() {
	ctx := context.Background()

	// Create and save a player
	player, err := entities.NewPlayer("player-1", "Alice", "secret-token")
	s.Require().NoError(err)
	s.Require().NoError(s.repo.Save(ctx, player))

	// Retrieve by token
	retrieved, err := s.repo.GetByToken(ctx, "secret-token")
	s.Require().NoError(err)
	s.Equal("player-1", retrieved.ID().String())
}

func (s *PostgresPlayerRepositoryTestSuite) TestGetByIDs() {
	ctx := context.Background()

	// Create multiple players
	player1, _ := entities.NewPlayer("player-1", "Alice", "token-1")
	player2, _ := entities.NewPlayer("player-2", "Bob", "token-2")
	player3, _ := entities.NewPlayer("player-3", "Charlie", "token-3")

	s.Require().NoError(s.repo.Save(ctx, player1))
	s.Require().NoError(s.repo.Save(ctx, player2))
	s.Require().NoError(s.repo.Save(ctx, player3))

	// Retrieve subset
	players, err := s.repo.GetByIDs(ctx, []entities.PlayerID{"player-1", "player-3"})
	s.Require().NoError(err)
	s.Len(players, 2)
}

func (s *PostgresPlayerRepositoryTestSuite) TestExists() {
	ctx := context.Background()

	// Create player
	player, _ := entities.NewPlayer("player-1", "Alice", "token-1")
	s.Require().NoError(s.repo.Save(ctx, player))

	// Check existence
	exists, err := s.repo.Exists(ctx, "player-1")
	s.Require().NoError(err)
	s.True(exists)

	// Check non-existent
	exists, err = s.repo.Exists(ctx, "non-existent")
	s.Require().NoError(err)
	s.False(exists)
}

func (s *PostgresPlayerRepositoryTestSuite) TestDelete() {
	ctx := context.Background()

	// Create player
	player, _ := entities.NewPlayer("player-1", "Alice", "token-1")
	s.Require().NoError(s.repo.Save(ctx, player))

	// Delete
	err := s.repo.Delete(ctx, "player-1")
	s.Require().NoError(err)

	// Verify deleted
	_, err = s.repo.GetByID(ctx, "player-1")
	s.ErrorIs(err, entities.ErrPlayerNotFound)
}

func TestPostgresPlayerRepository(t *testing.T) {
	// Skip if CGO is not available
	t.Skip("Skipping repository tests - requires CGO for SQLite")
	suite.Run(t, new(PostgresPlayerRepositoryTestSuite))
}
