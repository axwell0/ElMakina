package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlayer(t *testing.T) {
	tests := []struct {
		name    string
		id      PlayerID
		nick    string
		token   string
		wantErr error
	}{
		{
			name:    "valid player",
			id:      "player-1",
			nick:    "Alice",
			token:   "secret-token-1",
			wantErr: nil,
		},
		{
			name:    "empty id",
			id:      "",
			nick:    "Alice",
			token:   "secret-token-1",
			wantErr: ErrInvalidEntity,
		},
		{
			name:    "empty nick",
			id:      "player-1",
			nick:    "",
			token:   "secret-token-1",
			wantErr: ErrInvalidEntity,
		},
		{
			name:    "empty token",
			id:      "player-1",
			nick:    "Alice",
			token:   "",
			wantErr: ErrInvalidEntity,
		},
		{
			name:    "nick too long",
			id:      "player-1",
			nick:    "this is a very long nick that exceeds the maximum length of 32 characters",
			token:   "secret-token-1",
			wantErr: ErrInvalidEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player, err := NewPlayer(tt.id, tt.nick, tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, player)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.id, player.ID())
				assert.Equal(t, tt.nick, player.Nick())
				assert.Equal(t, tt.token, player.Token())
				assert.NotZero(t, player.CreatedAt())
				assert.NotZero(t, player.UpdatedAt())
			}
		})
	}
}

func TestPlayer_SetAvatar(t *testing.T) {
	player, err := NewPlayer("player-1", "Alice", "token-1")
	require.NoError(t, err)

	// Set avatar
	player.SetAvatar("avatar-url-1")

	assert.Equal(t, "avatar-url-1", player.Avatar())
}

func TestPlayer_CanAuthenticate(t *testing.T) {
	player, err := NewPlayer("player-1", "Alice", "correct-token")
	require.NoError(t, err)

	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "correct token",
			token:    "correct-token",
			expected: true,
		},
		{
			name:     "incorrect token",
			token:    "wrong-token",
			expected: false,
		},
		{
			name:     "empty token",
			token:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := player.CanAuthenticate(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReconstitutePlayer(t *testing.T) {
	id := PlayerID("player-1")
	nick := "Alice"
	token := "secret-token"
	avatar := "avatar-url"
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	player := ReconstitutePlayer(id, nick, token, avatar, createdAt, updatedAt)

	assert.Equal(t, id, player.ID())
	assert.Equal(t, nick, player.Nick())
	assert.Equal(t, token, player.Token())
	assert.Equal(t, avatar, player.Avatar())
	assert.Equal(t, createdAt, player.CreatedAt())
	assert.Equal(t, updatedAt, player.UpdatedAt())
}
