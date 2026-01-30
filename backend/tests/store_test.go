package tests

import (
	"ElMakina/backend/server"
	"ElMakina/backend/server/store"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lobbies.json")
	fs := &store.FileStore{Path: path}

	snap := server.LobbySnapshot{
		Lobbies: map[string]server.Lobby{
			"lobby-1": {ID: "lobby-1", LeaderID: "player-1", PlayerIDs: []string{"player-1"}, Status: server.LobbyOpen},
		},
		Players: map[string]server.Player{
			"player-1": {ID: "player-1", Nick: "p1", Token: "t1"},
		},
		ByToken:     map[string]string{"t1": "player-1"},
		NextLobbyID: 1,
	}

	if err := fs.Save(context.Background(), snap); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := fs.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil || loaded.Players["player-1"].Nick != "p1" {
		t.Fatalf("loaded snapshot mismatch")
	}
}

func TestFileStoreMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")
	fs := &store.FileStore{Path: path}

	loaded, err := fs.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil snapshot for missing file")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("unexpected file created on load")
	}
}
