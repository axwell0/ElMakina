package server

import (
	"context"
	"testing"
)

type memStore struct {
	loaded *LobbySnapshot
	saved  *LobbySnapshot
}

func (m *memStore) Load(ctx context.Context) (*LobbySnapshot, error) { return m.loaded, nil }
func (m *memStore) Save(ctx context.Context, snapshot LobbySnapshot) error {
	m.saved = &snapshot
	return nil
}

func TestLobbyManagerLifecycle(t *testing.T) {
	manager := NewLobbyManager(2, 4)
	leader, err := manager.RegisterPlayer("leader")
	if err != nil {
		t.Fatalf("register leader: %v", err)
	}
	p2, err := manager.RegisterPlayer("p2")
	if err != nil {
		t.Fatalf("register p2: %v", err)
	}

	lobby, err := manager.CreateLobby(leader.ID)
	if err != nil {
		t.Fatalf("create lobby: %v", err)
	}
	if lobby.LeaderID != leader.ID {
		t.Fatalf("expected leader set")
	}

	if err := manager.JoinLobby(lobby.ID, p2.ID); err != nil {
		t.Fatalf("join lobby: %v", err)
	}

	list := manager.ListLobbies()
	if len(list) != 1 || list[0].PlayerCount != 2 {
		t.Fatalf("expected lobby list to include 2 players")
	}

	session, err := manager.StartLobby(lobby.ID, leader.ID)
	if err != nil {
		t.Fatalf("start lobby: %v", err)
	}
	if session == nil || session.LobbyID != lobby.ID {
		t.Fatalf("expected session")
	}
}

func TestLobbyManagerReconnectAndAvatar(t *testing.T) {
	manager := NewLobbyManager(2, 4)
	player, err := manager.RegisterPlayer("player")
	if err != nil {
		t.Fatalf("register player: %v", err)
	}
	if err := manager.SetPlayerAvatar(player.ID, "avatar-1"); err != nil {
		t.Fatalf("set avatar: %v", err)
	}
	reconnected, err := manager.ReconnectPlayer(player.Token)
	if err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if reconnected.Avatar != "avatar-1" {
		t.Fatalf("expected avatar to persist")
	}

	if err := manager.UnregisterPlayer(player.ID); err != nil {
		t.Fatalf("unregister: %v", err)
	}
}

func TestLobbyManagerLoadSnapshot(t *testing.T) {
	store := &memStore{loaded: &LobbySnapshot{
		Lobbies:     map[string]Lobby{"l1": {ID: "l1", LeaderID: "p1", PlayerIDs: []string{"p1"}, Status: LobbyOpen}},
		Players:     map[string]Player{"p1": {ID: "p1", Nick: "p1", Token: "t1"}},
		ByToken:     map[string]string{"t1": "p1"},
		NextLobbyID: 2,
	}}
	manager, err := NewLobbyManagerWithStore(2, 4, store)
	if err != nil {
		t.Fatalf("new manager with store: %v", err)
	}

	player, err := manager.ReconnectPlayer("t1")
	if err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if player.ID != "p1" {
		t.Fatalf("expected loaded player")
	}

	if err := manager.SetPlayerAvatar(player.ID, "avatar"); err != nil {
		t.Fatalf("set avatar: %v", err)
	}
	if store.saved == nil {
		t.Fatalf("expected snapshot to be saved after changes")
	}
}
