package tests

import (
	"ElMakina/backend/server"
	"testing"
)

func TestGameSessionForfeit(t *testing.T) {
	manager := server.NewLobbyManager(2, 9)

	p1, _ := manager.RegisterPlayer("p1")
	p2, _ := manager.RegisterPlayer("p2")

	lobby, err := manager.CreateLobby(p1.ID)
	if err != nil {
		t.Fatalf("create lobby: %v", err)
	}
	if err := manager.JoinLobby(lobby.ID, p2.ID); err != nil {
		t.Fatalf("join lobby: %v", err)
	}

	session, err := manager.StartLobby(lobby.ID, p1.ID)
	if err != nil {
		t.Fatalf("start lobby: %v", err)
	}

	if ok := session.Forfeit(p2.ID); !ok {
		t.Fatalf("expected forfeit to succeed")
	}
	if got := len(session.Game.CurrentState.Players[1].Hand); got != 0 {
		t.Fatalf("expected empty hand, got %d", got)
	}
}
