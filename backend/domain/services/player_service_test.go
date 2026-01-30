package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"ElMakina/backend/domain/entities"
	"ElMakina/backend/domain/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPlayerRepository is a test double for PlayerRepository.
type mockPlayerRepository struct {
	players          map[entities.PlayerID]*entities.Player
	byToken          map[string]*entities.Player
	saveCalls        int
	deleteCalls      int
	deleteBatchCalls int
}

func newMockPlayerRepository() *mockPlayerRepository {
	return &mockPlayerRepository{
		players: make(map[entities.PlayerID]*entities.Player),
		byToken: make(map[string]*entities.Player),
	}
}

func (m *mockPlayerRepository) GetByID(ctx context.Context, id entities.PlayerID) (*entities.Player, error) {
	if p, ok := m.players[id]; ok {
		return p, nil
	}
	return nil, entities.ErrPlayerNotFound
}

func (m *mockPlayerRepository) GetByToken(ctx context.Context, token string) (*entities.Player, error) {
	if p, ok := m.byToken[token]; ok {
		return p, nil
	}
	return nil, entities.ErrPlayerNotFound
}

func (m *mockPlayerRepository) GetByIDs(ctx context.Context, ids []entities.PlayerID) ([]*entities.Player, error) {
	var result []*entities.Player
	for _, id := range ids {
		if p, ok := m.players[id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPlayerRepository) Save(ctx context.Context, player *entities.Player) error {
	m.saveCalls++
	m.players[player.ID()] = player
	m.byToken[player.Token()] = player
	return nil
}

func (m *mockPlayerRepository) Delete(ctx context.Context, id entities.PlayerID) error {
	m.deleteCalls++
	if p, ok := m.players[id]; ok {
		delete(m.players, id)
		delete(m.byToken, p.Token())
		return nil
	}
	return entities.ErrPlayerNotFound
}

func (m *mockPlayerRepository) Exists(ctx context.Context, id entities.PlayerID) (bool, error) {
	_, ok := m.players[id]
	return ok, nil
}

func (m *mockPlayerRepository) ListInactive(ctx context.Context, since time.Time) ([]*entities.Player, error) {
	var result []*entities.Player
	for _, p := range m.players {
		if p.LastActiveAt().Before(since) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPlayerRepository) DeleteBatch(ctx context.Context, ids []entities.PlayerID) (int, error) {
	m.deleteBatchCalls++
	count := 0
	for _, id := range ids {
		if p, ok := m.players[id]; ok {
			delete(m.players, id)
			delete(m.byToken, p.Token())
			count++
		}
	}
	return count, nil
}

// mockLobbyRepository is a test double for LobbyRepository.
type mockLobbyRepository struct {
	playerLobbies map[entities.PlayerID]*entities.Lobby
}

func newMockLobbyRepository() *mockLobbyRepository {
	return &mockLobbyRepository{
		playerLobbies: make(map[entities.PlayerID]*entities.Lobby),
	}
}

func (m *mockLobbyRepository) GetByID(ctx context.Context, id entities.LobbyID) (*entities.Lobby, error) {
	return nil, entities.ErrLobbyNotFound
}

func (m *mockLobbyRepository) GetByIDWithPlayers(ctx context.Context, id entities.LobbyID) (*repositories.LobbyWithPlayers, error) {
	return nil, entities.ErrLobbyNotFound
}

func (m *mockLobbyRepository) GetByPlayerID(ctx context.Context, playerID entities.PlayerID) (*entities.Lobby, error) {
	if l, ok := m.playerLobbies[playerID]; ok {
		return l, nil
	}
	return nil, entities.ErrLobbyNotFound
}

func (m *mockLobbyRepository) List(ctx context.Context, filter repositories.LobbyListFilter) ([]*entities.Lobby, error) {
	return nil, nil
}

func (m *mockLobbyRepository) ListWithPlayerCounts(ctx context.Context, filter repositories.LobbyListFilter) ([]repositories.LobbySummary, error) {
	return nil, nil
}

func (m *mockLobbyRepository) Save(ctx context.Context, lobby *entities.Lobby) error {
	return nil
}

func (m *mockLobbyRepository) Delete(ctx context.Context, id entities.LobbyID) error {
	return nil
}

func (m *mockLobbyRepository) Exists(ctx context.Context, id entities.LobbyID) (bool, error) {
	return false, nil
}

func (m *mockLobbyRepository) SetPlayerLobby(playerID entities.PlayerID, lobby *entities.Lobby) {
	m.playerLobbies[playerID] = lobby
}

// mockUnitOfWork executes functions directly without transactions.
type mockUnitOfWork struct{}

func (m *mockUnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// staticTokenGenerator generates predictable tokens for testing.
type staticTokenGenerator struct {
	tokens []string
	index  int
}

func (g *staticTokenGenerator) GeneratePlayerToken() (string, error) {
	if g.index < len(g.tokens) {
		token := g.tokens[g.index]
		g.index++
		return token, nil
	}
	return "", errors.New("no more tokens")
}

// staticIDGenerator generates predictable IDs for testing.
type staticIDGenerator struct {
	ids   []entities.PlayerID
	index int
}

func (g *staticIDGenerator) GeneratePlayerID() entities.PlayerID {
	if g.index < len(g.ids) {
		id := g.ids[g.index]
		g.index++
		return id
	}
	return ""
}

func TestPlayerService_RegisterPlayer(t *testing.T) {
	testCases := []struct {
		name        string
		nick        string
		expectError bool
		errorIs     error
	}{
		{
			name:        "valid nickname",
			nick:        "Alice",
			expectError: false,
		},
		{
			name:        "empty nickname",
			nick:        "",
			expectError: true,
			errorIs:     entities.ErrNicknameEmpty,
		},
		{
			name:        "nickname too long",
			nick:        "thisnicknameiswaytoolongtobeallowed",
			expectError: true,
			errorIs:     entities.ErrNicknameTooLong,
		},
		{
			name:        "nickname at max length",
			nick:        "thisisexactlythirtytwocharslong!",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			playerRepo := newMockPlayerRepository()
			lobbyRepo := newMockLobbyRepository()
			unitOfWork := &mockUnitOfWork{}
			tokenGen := &staticTokenGenerator{tokens: []string{"token123"}}
			idGen := &staticIDGenerator{ids: []entities.PlayerID{"player-1"}}

			svc := NewPlayerService(playerRepo, lobbyRepo, unitOfWork, tokenGen, idGen)

			player, err := svc.RegisterPlayer(context.Background(), tc.nick)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, player)
			assert.Equal(t, tc.nick, player.Nick())
			assert.Equal(t, entities.PlayerID("player-1"), player.ID())
			assert.Equal(t, "token123", player.Token())
			assert.Equal(t, 1, playerRepo.saveCalls)
		})
	}
}

func TestPlayerService_ReconnectPlayer(t *testing.T) {
	testCases := []struct {
		name        string
		token       string
		setup       func(*mockPlayerRepository)
		expectError bool
		errorIs     error
	}{
		{
			name:  "valid token",
			token: "valid-token",
			setup: func(m *mockPlayerRepository) {
				player, _ := entities.NewPlayer("player-1", "Alice", "valid-token")
				m.players["player-1"] = player
				m.byToken["valid-token"] = player
			},
			expectError: false,
		},
		{
			name:        "empty token",
			token:       "",
			setup:       func(m *mockPlayerRepository) {},
			expectError: true,
			errorIs:     entities.ErrInvalidToken,
		},
		{
			name:        "invalid token",
			token:       "invalid-token",
			setup:       func(m *mockPlayerRepository) {},
			expectError: true,
			errorIs:     entities.ErrInvalidToken,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			playerRepo := newMockPlayerRepository()
			lobbyRepo := newMockLobbyRepository()
			unitOfWork := &mockUnitOfWork{}

			tc.setup(playerRepo)

			svc := NewPlayerService(playerRepo, lobbyRepo, unitOfWork, nil, nil)

			player, err := svc.ReconnectPlayer(context.Background(), tc.token)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, player)
		})
	}
}

func TestPlayerService_UnregisterPlayer(t *testing.T) {
	testCases := []struct {
		name        string
		playerID    entities.PlayerID
		setup       func(*mockPlayerRepository, *mockLobbyRepository)
		expectError bool
		errorIs     error
	}{
		{
			name:     "player not in lobby",
			playerID: "player-1",
			setup: func(mpr *mockPlayerRepository, mlr *mockLobbyRepository) {
				player, _ := entities.NewPlayer("player-1", "Alice", "token")
				mpr.players["player-1"] = player
				mpr.byToken["token"] = player
			},
			expectError: false,
		},
		{
			name:     "player in lobby",
			playerID: "player-1",
			setup: func(mpr *mockPlayerRepository, mlr *mockLobbyRepository) {
				player, _ := entities.NewPlayer("player-1", "Alice", "token")
				mpr.players["player-1"] = player
				mpr.byToken["token"] = player

				lobby, _ := entities.NewLobby("lobby-1", "player-1")
				mlr.SetPlayerLobby("player-1", lobby)
			},
			expectError: true,
			errorIs:     entities.ErrPlayerInLobby,
		},
		{
			name:        "player not found",
			playerID:    "player-1",
			setup:       func(mpr *mockPlayerRepository, mlr *mockLobbyRepository) {},
			expectError: true,
			errorIs:     entities.ErrPlayerNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			playerRepo := newMockPlayerRepository()
			lobbyRepo := newMockLobbyRepository()
			unitOfWork := &mockUnitOfWork{}

			tc.setup(playerRepo, lobbyRepo)

			svc := NewPlayerService(playerRepo, lobbyRepo, unitOfWork, nil, nil)

			err := svc.UnregisterPlayer(context.Background(), tc.playerID)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 1, playerRepo.deleteCalls)
		})
	}
}

func TestPlayerService_PruneInactivePlayers(t *testing.T) {
	ctx := context.Background()
	playerRepo := newMockPlayerRepository()
	lobbyRepo := newMockLobbyRepository()
	unitOfWork := &mockUnitOfWork{}

	// Create players with different activity times
	activePlayer, _ := entities.NewPlayer("player-1", "Active", "token1")
	activePlayer.TouchActivity() // Just updated

	// Create an "inactive" player by reconstituting with old timestamps
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	inactivePlayer := entities.ReconstitutePlayer(
		"player-2", "Inactive", "token2", "", oldTime, oldTime, oldTime,
	)

	playerRepo.Save(ctx, activePlayer)
	playerRepo.Save(ctx, inactivePlayer)

	svc := NewPlayerService(playerRepo, lobbyRepo, unitOfWork, nil, nil)

	// Prune players inactive for more than 1 hour
	removedIDs, err := svc.PruneInactivePlayers(ctx, time.Hour)

	require.NoError(t, err)
	assert.Len(t, removedIDs, 1)
	assert.Contains(t, removedIDs, entities.PlayerID("player-2"))
}

func TestPlayerService_SetPlayerAvatar(t *testing.T) {
	ctx := context.Background()
	playerRepo := newMockPlayerRepository()
	lobbyRepo := newMockLobbyRepository()
	unitOfWork := &mockUnitOfWork{}

	player, _ := entities.NewPlayer("player-1", "Alice", "token")
	playerRepo.Save(ctx, player)

	svc := NewPlayerService(playerRepo, lobbyRepo, unitOfWork, nil, nil)

	err := svc.SetPlayerAvatar(ctx, "player-1", "avatar.png")
	require.NoError(t, err)

	// Verify avatar was set
	updated, _ := playerRepo.GetByID(ctx, "player-1")
	assert.Equal(t, "avatar.png", updated.Avatar())
}

func TestPlayerService_TouchPlayerActivity(t *testing.T) {
	ctx := context.Background()
	playerRepo := newMockPlayerRepository()
	lobbyRepo := newMockLobbyRepository()
	unitOfWork := &mockUnitOfWork{}

	player, _ := entities.NewPlayer("player-1", "Alice", "token")
	originalActivity := player.LastActiveAt()
	playerRepo.Save(ctx, player)

	// Wait a tiny bit to ensure time changes
	time.Sleep(10 * time.Millisecond)

	svc := NewPlayerService(playerRepo, lobbyRepo, unitOfWork, nil, nil)

	err := svc.TouchPlayerActivity(ctx, "player-1")
	require.NoError(t, err)

	// Verify activity was updated
	updated, _ := playerRepo.GetByID(ctx, "player-1")
	assert.True(t, updated.LastActiveAt().After(originalActivity))
}
