package tests

import (
	"ElMakina/backend/server"
	"ElMakina/backend/server/store"
	"ElMakina/backend/server/ws"
	"context"
	"path/filepath"
	"testing"
)

func TestNewLobbyManagerFromEnvLoadsSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lobbies.json")

	snapshot := server.LobbySnapshot{
		Lobbies: map[string]server.Lobby{
			"lobby-1": {
				ID:        "lobby-1",
				LeaderID:  "player-1",
				PlayerIDs: []string{"player-1"},
				Status:    server.LobbyOpen,
			},
		},
		Players: map[string]server.Player{
			"player-1": {ID: "player-1", Nick: "leader", Token: "token-1"},
		},
		ByToken:     map[string]string{"token-1": "player-1"},
		NextLobbyID: 1,
	}
	fileStore := &store.FileStore{Path: path}
	if err := fileStore.Save(context.Background(), snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	t.Setenv(ws.EnvLobbyStorePath, path)
	manager, err := ws.NewLobbyManagerFromEnv(2, 9)
	if err != nil {
		t.Fatalf("NewLobbyManagerFromEnv: %v", err)
	}

	lobby, err := manager.GetLobby("lobby-1")
	if err != nil {
		t.Fatalf("GetLobby: %v", err)
	}
	if lobby.LeaderID != "player-1" {
		t.Fatalf("expected leader player-1, got %s", lobby.LeaderID)
	}
}

func TestNewLobbyManagerFromEnvNoPath(t *testing.T) {
	t.Setenv(ws.EnvLobbyStorePath, "")
	manager, err := ws.NewLobbyManagerFromEnv(2, 9)
	if err != nil {
		t.Fatalf("NewLobbyManagerFromEnv: %v", err)
	}
	if manager == nil {
		t.Fatalf("expected non-nil manager")
	}
}
