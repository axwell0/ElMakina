// File: server/ws/session_controller_test.go
// Purpose: Exercises the hosted room prompt flow and websocket guardrails.
package ws

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sessionapp "example.com/elmakina/app/session"
	coretypes "example.com/elmakina/engine/core/types"
	ports "example.com/elmakina/engine/ports"
)

func TestRoomRunsIncomeTurnEndToEnd(t *testing.T) {
	room, clients := newTestRoom(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := room.RunTurn(ctx)
		resultCh <- err
	}()

	message := clients[0].awaitMessage(t)
	prompt, ok := message.(PromptEnvelope)
	require.True(t, ok)
	require.Equal(t, "main_action", prompt.Interaction.Kind)

	clients[1].assertNoMessage(t)

	err := room.HandleCommand("room-1-player-0", CommandEnvelopeMessage{
		Type:          TypeCommand,
		InteractionID: prompt.Interaction.ID,
		Kind:          string(sessionapp.CommandMainAction),
		Payload: map[string]any{
			"actionId":   string(coretypes.Income),
			"actorIndex": 0,
		},
	})
	require.NoError(t, err)
	require.NoError(t, <-resultCh)

	player0Result, ok := clients[0].awaitMessage(t).(TurnResultMessage)
	require.True(t, ok)
	player1Result, ok := clients[1].awaitMessage(t).(TurnResultMessage)
	require.True(t, ok)

	require.Equal(t, TypeTurnResult, player0Result.Type)
	require.Equal(t, "income", player0Result.Main.ID)
	require.Equal(t, 3, player0Result.Summary.Players[0].Coins)
	require.Len(t, player0Result.Hand, 3)
	require.Len(t, player1Result.Hand, 3)
}

func TestRoomRejectsWrongPlayerSubmission(t *testing.T) {
	room, clients := newTestRoom(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_, _ = room.RunTurn(ctx)
	}()

	message := clients[0].awaitMessage(t)
	prompt, ok := message.(PromptEnvelope)
	require.True(t, ok)

	err := room.HandleCommand("room-1-player-1", CommandEnvelopeMessage{
		Type:          TypeCommand,
		InteractionID: prompt.Interaction.ID,
		Kind:          string(sessionapp.CommandMainAction),
		Payload: map[string]any{
			"actionId":   string(coretypes.Income),
			"actorIndex": 1,
		},
	})
	require.Error(t, err)
}

func TestRoomRejectsCommandForWrongInteractionRecipient(t *testing.T) {
	room, clients := newTestRoom(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_, _ = room.RunTurn(ctx)
	}()

	message := clients[0].awaitMessage(t)
	prompt, ok := message.(PromptEnvelope)
	require.True(t, ok)
	require.Equal(t, "main_action", prompt.Interaction.Kind)

	err := room.HandleCommand("room-1-player-1", CommandEnvelopeMessage{
		Type:          "command",
		InteractionID: prompt.Interaction.ID,
		Kind:          "main_action",
		Payload: map[string]any{
			"actionId":   "income",
			"actorIndex": 1,
		},
	})
	require.Error(t, err)
}

func newTestRoom(t *testing.T) (*Room, []*stubClient) {
	t.Helper()

	session, err := sessionapp.NewGameSession("room-1", "A", 2, rand.New(rand.NewPCG(3, 4)))
	require.NoError(t, err)
	first, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	second, err := session.JoinPlayer("B", time.Unix(0, 0).UTC())
	require.NoError(t, err)

	room, err := newRoom(session, ports.TurnConfig{}, nil)
	require.NoError(t, err)

	clients := []*stubClient{
		newStubClient(first),
		newStubClient(second),
	}
	for _, client := range clients {
		require.NoError(t, room.Register(client))
	}

	return room, clients
}

type stubClient struct {
	session  *sessionapp.ClientSession
	messages chan any
}

func newStubClient(session *sessionapp.ClientSession) *stubClient {
	return &stubClient{
		session:  session,
		messages: make(chan any, 8),
	}
}

func (c *stubClient) Session() *sessionapp.ClientSession {
	return c.session
}

func (c *stubClient) Send(_ context.Context, message any) error {
	c.messages <- message
	return nil
}

func (c *stubClient) Close() error {
	close(c.messages)
	return nil
}

func (c *stubClient) awaitMessage(t *testing.T) any {
	t.Helper()
	select {
	case message := <-c.messages:
		return message
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
		return nil
	}
}

func (c *stubClient) assertNoMessage(t *testing.T) {
	t.Helper()
	select {
	case message := <-c.messages:
		t.Fatalf("expected no message, got %#v", message)
	case <-time.After(150 * time.Millisecond):
	}
}
