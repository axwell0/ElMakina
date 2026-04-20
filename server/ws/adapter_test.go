// File: server/ws/adapter_test.go
// Purpose: Protects the adapter layer that turns engine state into websocket messages.
package ws

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sessionapp "example.com/elmakina/app/session"
	corestate "example.com/elmakina/engine/core/state"
	coretypes "example.com/elmakina/engine/core/types"
	runtimeturn "example.com/elmakina/engine/runtime/turn"
)

func TestBuildPromptEnvelopeUsesSessionInteractionState(t *testing.T) {
	interaction := &sessionapp.InteractionState{
		ID:   "ix-1",
		Kind: sessionapp.InteractionChallenge,
		Recipients: sessionapp.RecipientSet{
			Players: []int{1},
		},
		Payload: sessionapp.ChallengeInteraction{
			ActorIndex:  0,
			Action:      "assassinate",
			ClaimedRole: "Colonel",
		},
	}

	envelope, err := BuildPromptEnvelope(interaction)
	require.NoError(t, err)
	require.Equal(t, MessageType("prompt"), envelope.Type)
	require.Equal(t, "challenge", envelope.Interaction.Kind)
}

func TestBuildTurnResultMessageIncludesViewerPrivateData(t *testing.T) {
	session := newTestSession(t)
	session.State.TurnNumber = 4

	result := &runtimeturn.TurnResult{
		Logs:         []string{"public log"},
		PrivateLogs:  map[int][]string{0: {"private log"}},
		Main:         corestate.NewNoPayloadAction(coretypes.Income, 0),
		MainApplied:  true,
		MainCanceled: false,
	}

	message, err := BuildTurnResultMessage(session, 0, result)
	require.NoError(t, err)
	require.Equal(t, TypeTurnResult, message.Type)
	require.Equal(t, session.ID, message.RoomID)
	require.Equal(t, 4, message.TurnNumber)
	require.Equal(t, []string{"public log"}, message.Logs)
	require.Equal(t, []string{"private log"}, message.PrivateLogs)
	require.Len(t, message.Hand, len(session.State.Players[0].Hand))
	require.Equal(t, "income", message.Main.ID)
}

func TestBuildRoomSyncMessageNestsConnectedPlayersInRoomState(t *testing.T) {
	session, message := buildLobbyRoomSyncMessage(t, func(session *sessionapp.GameSession) {
		require.NoError(t, session.MarkReady(0, true))
	})
	require.Equal(t, TypeRoomSync, message.Type)
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
	require.Len(t, message.Hand, len(session.State.Players[0].Hand))
}

func TestBuildRoomSyncMessageIncludesLeaderAndLifecycleFlags(t *testing.T) {
	_, message := buildLobbyRoomSyncMessage(t, nil)
	require.Equal(t, TypeRoomSync, message.Type)
	require.Equal(t, 0, message.Room.LeaderPlayerIndex)
	require.True(t, message.Room.CanRestart)
	require.True(t, message.Room.CanDelete)
}

func TestDecodeSubmitMessages(t *testing.T) {
	targetIndex := 1
	command, err := DecodeSubmitAction(SubmitActionMessage{
		Type:   TypeSubmitAction,
		RoomID: "room-1",
		Action: ActionMessage{
			ID:         string(coretypes.Accuse),
			ActorIndex: 0,
			Payload: ActionPayloadMessage{
				TargetIndex: &targetIndex,
				Guess:       string(coretypes.Thief),
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, coretypes.Accuse, command.ID)

	accuse, ok := command.Payload.(coretypes.AccusePayload)
	require.True(t, ok)
	require.Equal(t, 1, accuse.TargetIndex)
	require.Equal(t, coretypes.Thief, accuse.Guess)

	counter, err := DecodeSubmitCounter(SubmitCounterMessage{
		Type:   TypeSubmitCounter,
		RoomID: "room-1",
		Decision: CounterDecisionMessage{
			PlayerIndex: 1,
			Command: &ActionMessage{
				ID:         string(coretypes.BlockSteal),
				ActorIndex: 1,
				MainAction: string(coretypes.Steal),
			},
		},
	})
	require.NoError(t, err)
	require.False(t, counter.Pass)
	require.NotNil(t, counter.Command)
	require.Equal(t, coretypes.BlockSteal, counter.Command.ID)

	challenge := DecodeSubmitChallenge(SubmitChallengeMessage{
		Type:   TypeSubmitChallenge,
		RoomID: "room-1",
		Decision: ChallengeDecisionMessage{
			ChallengerIndex:        1,
			ActorDiscardIndex:      intPtr(0),
			ActorProvingCardIndex:  intPtr(1),
			ChallengerDiscardIndex: intPtr(0),
		},
	})
	require.Equal(t, 1, challenge.ChallengerIndex)

	step, err := DecodeSubmitStep(&corestate.StepRequest{
		ActionID: coretypes.Exchange,
		Actor:    0,
		Kind:     corestate.StepCardSelection,
		CardSelection: &corestate.CardSelectionStep{
			Count: 2,
		},
	}, SubmitStepMessage{
		Type:   TypeSubmitStep,
		RoomID: "room-1",
		Response: StepResponseMessage{
			Kind: string(corestate.StepCardSelection),
			CardSelection: &CardSelectionResponseMessage{
				SelectedIndices: []int{0, 2},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, step.CardSelection)
	require.Equal(t, []int{0, 2}, step.CardSelection.SelectedIndices)
}

func TestDecodeSubmitActionRejectsMalformedTargetPayloads(t *testing.T) {
	_, err := DecodeSubmitAction(SubmitActionMessage{
		Type:   TypeSubmitAction,
		RoomID: "room-1",
		Action: ActionMessage{
			ID:         string(coretypes.Investigate),
			ActorIndex: 0,
		},
	})
	require.EqualError(t, err, `action "investigate" requires a target`)

	targetIndex := 1
	_, err = DecodeSubmitAction(SubmitActionMessage{
		Type:   TypeSubmitAction,
		RoomID: "room-1",
		Action: ActionMessage{
			ID:         string(coretypes.Accuse),
			ActorIndex: 0,
			Payload: ActionPayloadMessage{
				TargetIndex: &targetIndex,
			},
		},
	})
	require.EqualError(t, err, `action "accuse" requires a guess`)
}

func TestDecodeSubmitCounterRejectsMissingCommandWhenPassingFalse(t *testing.T) {
	_, err := DecodeSubmitCounter(SubmitCounterMessage{
		Type:   TypeSubmitCounter,
		RoomID: "room-1",
		Decision: CounterDecisionMessage{
			PlayerIndex: 1,
			Pass:        false,
		},
	})
	require.EqualError(t, err, "counter command is required when pass is false")
}

func newTestSession(t *testing.T) *sessionapp.GameSession {
	t.Helper()

	session, err := sessionapp.NewGameSession("room-1", "A", 2, rand.New(rand.NewPCG(1, 2)))
	require.NoError(t, err)

	return session
}

func buildLobbyRoomSyncMessage(t *testing.T, mutate func(*sessionapp.GameSession)) (*sessionapp.GameSession, RoomSyncMessage) {
	t.Helper()

	session := newTestSession(t)
	playerSession, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	playerSession.MarkSeen(playerSession.LastSeenAt)
	if mutate != nil {
		mutate(session)
	}

	message, err := BuildRoomSyncMessage(session, 0)
	require.NoError(t, err)
	return session, message
}

func intPtr(value int) *int {
	return &value
}
