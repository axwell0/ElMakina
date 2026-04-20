package session

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	corestate "example.com/elmakina/engine/core/state"
	coretypes "example.com/elmakina/engine/core/types"
	ports "example.com/elmakina/engine/ports"
	"github.com/stretchr/testify/require"
)

func TestGameSessionRestartResetsState(t *testing.T) {
	session := newHostedTestSession(t)
	session.MarkGameOver()
	session.State.Players[0].Coins = 9
	require.NoError(t, session.MarkReady(0, true))

	require.NoError(t, session.RestartGame())
	require.Equal(t, PhaseLobby, session.Phase())
	require.Equal(t, 2, session.State.Players[0].Coins)
	require.Empty(t, session.ReadyPlayerIndices())
	require.Nil(t, session.OpenInput())
}

func TestGameSessionStartTurnAndSubmitMainAction(t *testing.T) {
	session := newHostedTestSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, session.StartTurn(ctx, ports.TurnConfig{}))

	waitForSessionOpenInput(t, session, InputMainAction, func(input *OpenInput) bool {
		return input.Actor == 0
	})

	err := session.SubmitMainAction("session-0", &coretypes.PlayerAction{
		ID:         coretypes.Income,
		ActorIndex: 0,
		Payload:    coretypes.NoPayload{},
	})
	require.NoError(t, err)

	result, err := session.AwaitTurn(ctx)
	require.NoError(t, err)
	require.True(t, result.MainApplied)
	require.Equal(t, 3, session.State.Players[0].Coins)
	require.Nil(t, session.OpenInput())
}

func TestGameSessionOpenInputProgressesChallengeToCounter(t *testing.T) {
	session := newHostedTestSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, session.StartTurn(ctx, ports.TurnConfig{}))

	waitForSessionOpenInput(t, session, InputMainAction, func(input *OpenInput) bool {
		return input.Actor == 0
	})

	require.NoError(t, session.SubmitMainAction("session-0", &coretypes.PlayerAction{
		ID:         coretypes.Investigate,
		ActorIndex: 0,
		Payload:    coretypes.TargetPayload{TargetIndex: 1},
	}))

	waitForSessionOpenInput(t, session, InputChallenge, func(input *OpenInput) bool {
		return input.Challenge != nil
	})

	require.NoError(t, session.SubmitChallengeDecision("session-1", ports.ChallengeDecision{
		ChallengerIndex: 1,
		Pass:            true,
	}))

	waitForSessionOpenInput(t, session, InputCounter, func(input *OpenInput) bool {
		return input.Counter != nil &&
			input.Counter.MainAction != nil &&
			input.Counter.MainAction.ID == coretypes.Investigate
	})

	require.NoError(t, session.SubmitCounterDecision("session-1", ports.CounterDecision{
		PlayerIndex: 1,
		Pass:        true,
	}))

	_, err := session.AwaitTurn(ctx)
	require.NoError(t, err)
}

func TestGameSessionOpenInputShowsExchangeStep(t *testing.T) {
	session := newHostedTestSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, session.StartTurn(ctx, ports.TurnConfig{}))

	waitForSessionOpenInput(t, session, InputMainAction, func(input *OpenInput) bool {
		return input.Actor == 0
	})

	require.NoError(t, session.SubmitMainAction("session-0", &coretypes.PlayerAction{
		ID:         coretypes.Exchange,
		ActorIndex: 0,
		Payload:    coretypes.NoPayload{},
	}))

	waitForSessionOpenInput(t, session, InputChallenge, func(input *OpenInput) bool {
		return input.Challenge != nil
	})

	require.NoError(t, session.SubmitChallengeDecision("session-1", ports.ChallengeDecision{
		ChallengerIndex: 1,
		Pass:            true,
	}))

	waitForSessionOpenInput(t, session, InputStep, func(input *OpenInput) bool {
		return input.Step != nil
	})

	require.NoError(t, session.SubmitStepResponse("session-0", corestate.StepResponse{
		Kind: corestate.StepCardSelection,
		CardSelection: &corestate.CardSelectionResponse{
			SelectedIndices: []int{0, 1},
		},
	}))

	_, err := session.AwaitTurn(ctx)
	require.NoError(t, err)
}

func TestGameSessionPublishesInteractionStateDuringTurn(t *testing.T) {
	session := newHostedTestSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := session.StartTurn(ctx, ports.TurnConfig{})
	require.NoError(t, err)

	interaction := waitForSessionInteraction(t, session, InteractionMainAction)
	require.Equal(t, InteractionMainAction, interaction.Kind)
}

func newHostedTestSession(t *testing.T) *GameSession {
	t.Helper()

	session, err := NewGameSession("room-1", "A", 2, rand.New(rand.NewPCG(1, 2)))
	require.NoError(t, err)
	first, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	second, err := session.JoinPlayer("B", time.Unix(0, 0).UTC())
	require.NoError(t, err)

	session.State.Players[0].Hand = []corestate.Card{
		{ID: 1, Role: coretypes.Policewoman},
		{ID: 2, Role: coretypes.TaxCollector},
	}
	session.State.Players[1].Hand = []corestate.Card{
		{ID: 3, Role: coretypes.Policewoman},
		{ID: 4, Role: coretypes.Thief},
	}

	require.NoError(t, session.AttachClient(&attachableClient{
		session: first,
	}))
	require.NoError(t, session.AttachClient(&attachableClient{
		session: second,
	}))

	return session
}

func waitForSessionOpenInput(t *testing.T, session *GameSession, kind InputKind, extra func(*OpenInput) bool) *OpenInput {
	t.Helper()

	var input *OpenInput
	require.Eventually(t, func() bool {
		input = session.OpenInput()
		if input == nil || input.Kind != kind {
			return false
		}
		if extra != nil && !extra(input) {
			return false
		}
		return true
	}, 200*time.Millisecond, 10*time.Millisecond)
	return input
}

func waitForSessionInteraction(t *testing.T, session *GameSession, kind InteractionKind) *InteractionState {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case interaction := <-session.InteractionEvents():
			if interaction.Kind == kind {
				return &interaction
			}
		case <-timeout:
			t.Fatal("timed out waiting for session interaction")
			return nil
		}
	}
}
