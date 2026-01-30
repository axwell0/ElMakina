package tests

import (
	"ElMakina/backend/server"
	"testing"
)

func TestLobbyCreateJoinListStart(t *testing.T) {
	manager := server.NewLobbyManager(2, 9)

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
		t.Fatalf("leader mismatch")
	}

	if err := manager.JoinLobby(lobby.ID, p2.ID); err != nil {
		t.Fatalf("join lobby: %v", err)
	}

	list := manager.ListLobbies()
	if len(list) != 1 {
		t.Fatalf("expected 1 lobby, got %d", len(list))
	}
	if list[0].PlayerCount != 2 {
		t.Fatalf("player count: got %d want %d", list[0].PlayerCount, 2)
	}

	session, err := manager.StartLobby(lobby.ID, leader.ID)
	if err != nil {
		t.Fatalf("start lobby: %v", err)
	}
	if session.Game == nil {
		t.Fatalf("expected game session")
	}
	if got := session.IndexByPlayer[leader.ID]; got != 0 {
		t.Fatalf("leader index: got %d want %d", got, 0)
	}
	if got := session.IndexByPlayer[p2.ID]; got != 1 {
		t.Fatalf("p2 index: got %d want %d", got, 1)
	}
}

func TestLobbyConstraints(t *testing.T) {
	manager := server.NewLobbyManager(2, 2)

	p1, err := manager.RegisterPlayer("p1")
	if err != nil {
		t.Fatalf("register p1: %v", err)
	}
	p2, err := manager.RegisterPlayer("p2")
	if err != nil {
		t.Fatalf("register p2: %v", err)
	}
	p3, err := manager.RegisterPlayer("p3")
	if err != nil {
		t.Fatalf("register p3: %v", err)
	}

	lobby, err := manager.CreateLobby(p1.ID)
	if err != nil {
		t.Fatalf("create lobby: %v", err)
	}
	if err := manager.JoinLobby(lobby.ID, p2.ID); err != nil {
		t.Fatalf("join lobby: %v", err)
	}
	if err := manager.JoinLobby(lobby.ID, p3.ID); err == nil {
		t.Fatalf("expected lobby full error")
	}

	if _, err := manager.StartLobby(lobby.ID, p2.ID); err == nil {
		t.Fatalf("expected leader-only start error")
	}
}

func TestLobbyNotEnoughPlayers(t *testing.T) {
	manager := server.NewLobbyManager(2, 9)

	leader, _ := manager.RegisterPlayer("leader")
	lobby, err := manager.CreateLobby(leader.ID)
	if err != nil {
		t.Fatalf("create lobby: %v", err)
	}
	if _, err := manager.StartLobby(lobby.ID, leader.ID); err == nil {
		t.Fatalf("expected not enough players error")
	}
}

func TestRegisterPlayerDuplicateNick(t *testing.T) {
	manager := server.NewLobbyManager(2, 9)
	first, err := manager.RegisterPlayer("dup")
	if err != nil {
		t.Fatalf("register dup: %v", err)
	}
	second, err := manager.RegisterPlayer("dup")
	if err != nil {
		t.Fatalf("register dup again: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected unique player IDs for duplicate nicknames")
	}
}

func TestReconnectPlayer(t *testing.T) {
	manager := server.NewLobbyManager(2, 9)
	player, err := manager.RegisterPlayer("p1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	reconnected, err := manager.ReconnectPlayer(player.Token)
	if err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if reconnected.ID != player.ID {
		t.Fatalf("expected same player on reconnect")
	}
}

func TestRemovePlayerTransfersLeader(t *testing.T) {
	manager := server.NewLobbyManager(2, 9)
	leader, _ := manager.RegisterPlayer("leader")
	p2, _ := manager.RegisterPlayer("p2")

	lobby, _ := manager.CreateLobby(leader.ID)
	if err := manager.JoinLobby(lobby.ID, p2.ID); err != nil {
		t.Fatalf("join: %v", err)
	}
	if err := manager.RemovePlayer(lobby.ID, leader.ID); err != nil {
		t.Fatalf("remove leader: %v", err)
	}
	updated, err := manager.GetLobby(lobby.ID)
	if err != nil {
		t.Fatalf("get lobby: %v", err)
	}
	if updated.LeaderID != p2.ID {
		t.Fatalf("expected leader transfer to p2")
	}
}

func TestMultipleLobbiesConcurrent(t *testing.T) {
	manager := server.NewLobbyManager(2, 9)

	a1, _ := manager.RegisterPlayer("a1")
	a2, _ := manager.RegisterPlayer("a2")
	b1, _ := manager.RegisterPlayer("b1")
	b2, _ := manager.RegisterPlayer("b2")

	lobbyA, err := manager.CreateLobby(a1.ID)
	if err != nil {
		t.Fatalf("create lobbyA: %v", err)
	}
	lobbyB, err := manager.CreateLobby(b1.ID)
	if err != nil {
		t.Fatalf("create lobbyB: %v", err)
	}

	if err := manager.JoinLobby(lobbyA.ID, a2.ID); err != nil {
		t.Fatalf("join lobbyA: %v", err)
	}
	if err := manager.JoinLobby(lobbyB.ID, b2.ID); err != nil {
		t.Fatalf("join lobbyB: %v", err)
	}

	sessionA, err := manager.StartLobby(lobbyA.ID, a1.ID)
	if err != nil {
		t.Fatalf("start lobbyA: %v", err)
	}
	sessionB, err := manager.StartLobby(lobbyB.ID, b1.ID)
	if err != nil {
		t.Fatalf("start lobbyB: %v", err)
	}

	if sessionA.LobbyID == sessionB.LobbyID {
		t.Fatalf("expected distinct sessions")
	}
}
