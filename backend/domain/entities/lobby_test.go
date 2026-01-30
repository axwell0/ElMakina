package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLobby(t *testing.T) {
	tests := []struct {
		name     string
		id       LobbyID
		leaderID PlayerID
		wantErr  error
	}{
		{
			name:     "valid lobby",
			id:       "lobby-1",
			leaderID: "player-1",
			wantErr:  nil,
		},
		{
			name:     "empty id",
			id:       "",
			leaderID: "player-1",
			wantErr:  ErrInvalidEntity,
		},
		{
			name:     "empty leader id",
			id:       "lobby-1",
			leaderID: "",
			wantErr:  ErrInvalidEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lobby, err := NewLobby(tt.id, tt.leaderID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, lobby)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.id, lobby.ID())
				assert.Equal(t, tt.leaderID, lobby.LeaderID())
				assert.Equal(t, LobbyOpen, lobby.Status())
				assert.Equal(t, int64(1), lobby.Version())
				assert.Len(t, lobby.PlayerIDs(), 1)
				assert.Equal(t, tt.leaderID, lobby.PlayerIDs()[0])
			}
		})
	}
}

func TestLobby_AddPlayer(t *testing.T) {
	type args struct {
		playerID   PlayerID
		maxPlayers int
	}

	tests := []struct {
		name      string
		setup     func() *Lobby
		args      args
		wantErr   error
		wantCount int
	}{
		{
			name: "add player to open lobby",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				return l
			},
			args:      args{playerID: "player-2", maxPlayers: 4},
			wantErr:   nil,
			wantCount: 2,
		},
		{
			name: "reject duplicate player",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("player-2", 4)
				return l
			},
			args:      args{playerID: "player-2", maxPlayers: 4},
			wantErr:   ErrPlayerAlreadyInLobby,
			wantCount: 2,
		},
		{
			name: "reject when lobby full",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("p2", 2)
				return l
			},
			args:      args{playerID: "p3", maxPlayers: 2},
			wantErr:   ErrLobbyFull,
			wantCount: 2,
		},
		{
			name: "reject when lobby not open",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("p2", 4)
				_ = l.StartGame(2)
				return l
			},
			args:      args{playerID: "p3", maxPlayers: 4},
			wantErr:   ErrLobbyNotOpen,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.setup()
			err := l.AddPlayer(tt.args.playerID, tt.args.maxPlayers)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, l.PlayerIDs(), tt.wantCount)
		})
	}
}

func TestLobby_RemovePlayer(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() *Lobby
		playerID      PlayerID
		wantClose     bool
		wantNewLeader PlayerID
		wantErr       error
	}{
		{
			name: "remove non-leader",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("player-2", 4)
				return l
			},
			playerID:      "player-2",
			wantClose:     false,
			wantNewLeader: "leader-1",
			wantErr:       nil,
		},
		{
			name: "remove leader promotes next player",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("player-2", 4)
				return l
			},
			playerID:      "leader-1",
			wantClose:     false,
			wantNewLeader: "player-2",
			wantErr:       nil,
		},
		{
			name: "remove last player closes lobby",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				return l
			},
			playerID:  "leader-1",
			wantClose: true,
			wantErr:   nil,
		},
		{
			name: "reject removing non-existent player",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				return l
			},
			playerID:  "non-existent",
			wantClose: false,
			wantErr:   ErrPlayerNotInLobby,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.setup()
			shouldClose, err := l.RemovePlayer(tt.playerID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantClose, shouldClose)
			if !tt.wantClose {
				assert.Equal(t, tt.wantNewLeader, l.LeaderID())
			}
		})
	}
}

func TestLobby_StartGame(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() *Lobby
		minPlayers int
		wantErr    error
		wantStatus LobbyStatus
	}{
		{
			name: "start game with enough players",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("p2", 4)
				return l
			},
			minPlayers: 2,
			wantErr:    nil,
			wantStatus: LobbyInGame,
		},
		{
			name: "reject start with not enough players",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				return l
			},
			minPlayers: 2,
			wantErr:    ErrNotEnoughPlayers,
			wantStatus: LobbyOpen,
		},
		{
			name: "reject start when not open",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("p2", 4)
				_ = l.StartGame(2)
				return l
			},
			minPlayers: 2,
			wantErr:    ErrLobbyNotOpen,
			wantStatus: LobbyInGame,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.setup()
			err := l.StartGame(tt.minPlayers)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantStatus, l.Status())
		})
	}
}

func TestLobby_CanStart(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() *Lobby
		playerID   PlayerID
		minPlayers int
		wantErr    error
	}{
		{
			name: "leader can start",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("p2", 4)
				return l
			},
			playerID:   "leader-1",
			minPlayers: 2,
			wantErr:    nil,
		},
		{
			name: "non-leader cannot start",
			setup: func() *Lobby {
				l, _ := NewLobby("lobby-1", "leader-1")
				_ = l.AddPlayer("p2", 4)
				return l
			},
			playerID:   "p2",
			minPlayers: 2,
			wantErr:    ErrNotLeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.setup()
			err := l.CanStart(tt.playerID, tt.minPlayers)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLobby_HasPlayer(t *testing.T) {
	l, _ := NewLobby("lobby-1", "leader-1")
	_ = l.AddPlayer("player-2", 4)

	assert.True(t, l.HasPlayer("leader-1"))
	assert.True(t, l.HasPlayer("player-2"))
	assert.False(t, l.HasPlayer("player-3"))
}

func TestLobby_PlayerCount(t *testing.T) {
	l, _ := NewLobby("lobby-1", "leader-1")
	assert.Equal(t, 1, l.PlayerCount())

	_ = l.AddPlayer("p2", 4)
	assert.Equal(t, 2, l.PlayerCount())
}

func TestLobbyStatus_Valid(t *testing.T) {
	tests := []struct {
		status LobbyStatus
		valid  bool
	}{
		{LobbyOpen, true},
		{LobbyInGame, true},
		{LobbyClosed, true},
		{LobbyStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.Valid())
		})
	}
}
