package ws

import (
	"testing"
	"time"

	"ElMakina/backend/engine"
	"ElMakina/backend/server"
)

type stubConn struct{}

func (s *stubConn) ReadJSON(v any) error  { return nil }
func (s *stubConn) WriteJSON(v any) error { return nil }
func (s *stubConn) Close() error          { return nil }

func TestKickVoteRequiresUnanimousEligible(t *testing.T) {
	game, err := engine.NewGame([]string{"Ada", "Ben", "Cara"}, nil)
	if err != nil {
		t.Fatalf("new game: %v", err)
	}
	session := &server.GameSession{
		LobbyID:   "l1",
		Game:      game,
		PlayerIDs: []string{"p1", "p2", "p3"},
		IndexByPlayer: map[string]int{
			"p1": 0,
			"p2": 1,
			"p3": 2,
		},
	}
	runner := newSessionRunner(session, engine.RealClock{}, engine.TurnConfig{}, 60*time.Second, nil)
	// Player index 1 is the disconnected target. Attach only 0 and 2 as eligible voters.
	runner.provider.attach(0, &clientConn{conn: &stubConn{}})
	runner.provider.attach(2, &clientConn{conn: &stubConn{}})

	runner.markOffline("p2")
	if runner.paused == nil || !runner.paused.active {
		t.Fatalf("expected pause to be active")
	}

	runner.handleKickVote(0, KickVotePayload{TargetIndex: 1})
	if len(game.CurrentState.Players[1].Hand) == 0 {
		t.Fatalf("expected player to remain until unanimous vote")
	}

	// Ineligible vote from the disconnected player should not count.
	runner.handleKickVote(1, KickVotePayload{TargetIndex: 1})
	if len(game.CurrentState.Players[1].Hand) == 0 {
		t.Fatalf("expected ineligible vote to be ignored")
	}

	runner.handleKickVote(2, KickVotePayload{TargetIndex: 1})
	if len(game.CurrentState.Players[1].Hand) != 0 {
		t.Fatalf("expected player to be kicked after unanimous vote")
	}
	if runner.paused != nil && runner.paused.active {
		t.Fatalf("expected pause to be cleared after kick")
	}
}
