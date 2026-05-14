// File: server/ws/adapter_test.go
// Purpose: Protects the adapter layer that turns engine state into websocket messages.
package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	coretypes "github.com/axwell0/elmakina/engine/core/types"
	turnmodel "github.com/axwell0/elmakina/engine/turn"
	api "github.com/axwell0/elmakina/internal/api"
	sessionapp "github.com/axwell0/elmakina/internal/app/session"
)

// The adapter tests verify that engine snapshots become the exact websocket payloads the browser expects.
func TestBuildPromptEnvelopeUsesSessionPrompt(t *testing.T) {
	prompt := &sessionapp.Prompt{
		ID:         "ix-1",
		Kind:       sessionapp.PromptChallenge,
		Recipients: []int{1},
		Payload: sessionapp.ChallengePrompt{
			ActorIndex:  0,
			Action:      "assassinate",
			ClaimedRole: "Colonel",
		},
	}

	envelope, err := BuildPromptEnvelope(prompt)
	require.NoError(t, err)
	require.Equal(t, api.MessageType("prompt"), envelope.Type)
	require.Equal(t, "challenge", envelope.Prompt.Kind)
}

func TestPromptPathNoLongerDependsOnProjectionClient(t *testing.T) {
	prompt := &sessionapp.Prompt{
		ID:         "ix-1",
		Kind:       sessionapp.PromptMainAction,
		Recipients: []int{0},
		Payload: sessionapp.MainActionPrompt{
			ActorIndex: 0,
			TurnNumber: 1,
			Title:      "Choose",
		},
	}

	envelope, err := BuildPromptEnvelope(prompt)
	require.NoError(t, err)
	require.Equal(t, "main_action", envelope.Prompt.Kind)
}

func TestBuildPromptEnvelopeAllowsClearedPrompt(t *testing.T) {
	envelope, err := BuildClearedPromptEnvelope("ix-1")
	require.NoError(t, err)
	require.Equal(t, api.TypePrompt, envelope.Type)
	require.Nil(t, envelope.Prompt)
	require.Equal(t, "ix-1", envelope.ClearedPromptID)
}

func TestBuildTurnResultMessageIncludesViewerPrivateData(t *testing.T) {
	lobby := newTestLobby(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, lobby.Game().IncrementTurn())
	}

	result := &turnmodel.TurnResult{
		Logs:        []string{"public log"},
		PrivateLogs: map[int][]string{0: {"private log"}},
		Main: &coretypes.PlayerAction{
			Action:     coretypes.Income,
			ActorIndex: 0,
		},
		MainApplied:  true,
		MainCanceled: false,
	}

	message, err := BuildTurnResultMessage(lobby, 0, result)
	require.NoError(t, err)
	require.Equal(t, api.TypeTurnResult, message.Type)
	require.Equal(t, lobby.ID(), message.RoomID)
	require.Equal(t, 4, message.TurnNumber)
	require.Equal(t, []string{"public log"}, message.Logs)
	require.Equal(t, []string{"private log"}, message.PrivateLogs)
	require.Len(t, message.Hand, len(lobby.Game().Snapshot().players[0].Hand))
	require.Equal(t, "income", message.Main.ID)
}

func TestBuildRoomSyncMessageNestsConnectedPlayersInRoomState(t *testing.T) {
	lobby, message := buildLobbyRoomSyncMessage(t, func(lobby *sessionapp.Lobby) {
		require.NoError(t, lobby.MarkReady(0, true))
	})
	require.Equal(t, api.TypeRoomSync, message.Type)
	require.Equal(t, "lobby", message.Room.Phase)
	require.Equal(t, 0, message.Room.LeaderPlayerIndex)
	require.Equal(t, 2, message.Room.PlayerCount)
	require.Equal(t, 1, message.Room.JoinedPlayers)
	require.Equal(t, []int{0}, message.Room.ConnectedPlayers)
	require.Equal(t, []int{0}, message.Room.ReadyPlayers)
	require.False(t, message.Room.CanStart)
	require.True(t, message.Room.CanRestart)
	require.True(t, message.Room.CanDelete)
	require.Equal(t, "A", message.PlayerName)
	require.Len(t, message.Hand, len(lobby.Game().Snapshot().players[0].Hand))
}

func TestBuildRoomSyncMessageIncludesLeaderAndLifecycleFlags(t *testing.T) {
	_, message := buildLobbyRoomSyncMessage(t, nil)
	require.Equal(t, api.TypeRoomSync, message.Type)
	require.Equal(t, 0, message.Room.LeaderPlayerIndex)
	require.True(t, message.Room.CanRestart)
	require.True(t, message.Room.CanDelete)
}

// newTestSession creates a deterministic room state that the adapter tests can reuse.
func newTestLobby(t *testing.T) *sessionapp.Lobby {
	t.Helper()

	lobby, err := sessionapp.NewLobby("room-1", "A", 2)
	require.NoError(t, err)

	return lobby
}

// buildLobbyRoomSyncMessage joins the leader into a lobby and lets the caller tweak the session before projection.
func buildLobbyRoomSyncMessage(t *testing.T, mutate func(*sessionapp.Lobby)) (*sessionapp.Lobby, api.RoomSyncView) {
	t.Helper()

	lobby := newTestLobby(t)
	_, err := lobby.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	if mutate != nil {
		mutate(lobby)
	}

	message, err := BuildRoomSyncMessage(lobby, 0, []int{0})
	require.NoError(t, err)
	return lobby, message
}
