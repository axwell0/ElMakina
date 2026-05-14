// File: server/ws/session_controller_test.go
// Purpose: Exercises the hosted room prompt flow and websocket guardrails.
//
//nolint:errcheck,testifylint // Concurrent scenario tests intentionally trade strict style for concise flow assertions.
package ws

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	coretypes "github.com/axwell0/elmakina/engine/core/types"
	runtimeturn "github.com/axwell0/elmakina/engine/runtime/turn"
	api "github.com/axwell0/elmakina/internal/api"
	sessionapp "github.com/axwell0/elmakina/internal/app/session"
	"github.com/axwell0/elmakina/internal/apperrors"
)

// The controller tests exercise the live websocket flow through a hosted room.
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
	require.Equal(t, "main_action", prompt.Prompt.Kind)

	clients[1].assertNoMessage(t)

	err := room.HandleCommand("room-1-player-0", CommandEnvelopeMessage{
		Type:     api.TypeCommand,
		PromptID: prompt.Prompt.ID,
		Kind:     string(sessionapp.CommandMainAction),
		Payload: map[string]any{
			"action":     string(coretypes.Income),
			"actorIndex": 0,
		},
	})
	require.NoError(t, err)
	require.NoError(t, <-resultCh)

	player0Result, ok := clients[0].awaitTurnResult(t)
	require.True(t, ok)
	player1Result, ok := clients[1].awaitTurnResult(t)
	require.True(t, ok)

	require.Equal(t, api.TypeTurnResult, player0Result.Type)
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
		Type:     api.TypeCommand,
		PromptID: prompt.Prompt.ID,
		Kind:     string(sessionapp.CommandMainAction),
		Payload: map[string]any{
			"action":     string(coretypes.Income),
			"actorIndex": 1,
		},
	})
	require.Error(t, err)
}

func TestRoomRejectsCommandForWrongPromptRecipient(t *testing.T) {
	room, clients := newTestRoom(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_, _ = room.RunTurn(ctx)
	}()

	message := clients[0].awaitMessage(t)
	prompt, ok := message.(PromptEnvelope)
	require.True(t, ok)
	require.Equal(t, "main_action", prompt.Prompt.Kind)

	err := room.HandleCommand("room-1-player-1", CommandEnvelopeMessage{
		Type:     "command",
		PromptID: prompt.Prompt.ID,
		Kind:     "main_action",
		Payload: map[string]any{
			"action":     "income",
			"actorIndex": 1,
		},
	})
	require.Error(t, err)
}

func TestRoomRuntimeSendsPromptOnlyToRecipients(t *testing.T) {
	room, clients := newTestRoom(t)

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

	err := room.dispatchPromptToPlayers(prompt)
	require.NoError(t, err)

	_, ok := clients[1].awaitMessage(t).(PromptEnvelope)
	require.True(t, ok)
	clients[0].assertNoMessage(t)
}

func TestRoomRuntimeClearsPromptForPreviousRecipients(t *testing.T) {
	room, clients := newTestRoom(t)

	prompt := &sessionapp.Prompt{
		ID:         "ix-1",
		Kind:       sessionapp.PromptChallenge,
		Recipients: []int{0, 1},
		Payload: sessionapp.ChallengePrompt{
			ActorIndex:  0,
			Action:      "assassinate",
			ClaimedRole: "Colonel",
		},
	}

	err := room.dispatchPromptToPlayers(prompt)
	require.NoError(t, err)
	for _, client := range clients {
		_, ok := client.awaitMessage(t).(PromptEnvelope)
		require.True(t, ok)
	}

	err = room.clearPrompt(prompt)
	require.NoError(t, err)
	for _, client := range clients {
		message, ok := client.awaitMessage(t).(PromptEnvelope)
		require.True(t, ok)
		require.Equal(t, api.TypePrompt, message.Type)
		require.Nil(t, message.Prompt)
		require.Equal(t, "ix-1", message.ClearedPromptID)
	}
}

func TestRoomActorSerializesCommands(t *testing.T) {
	room, _ := newTestRoom(t)
	var active int32
	var maxActive int32
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := room.do(context.Background(), func() error {
				current := atomic.AddInt32(&active, 1)
				for {
					maxActiveSeen := atomic.LoadInt32(&maxActive)
					if current <= maxActiveSeen || atomic.CompareAndSwapInt32(&maxActive, maxActiveSeen, current) {
						break
					}
				}
				time.Sleep(time.Millisecond)
				atomic.AddInt32(&active, -1)
				return nil
			})
			require.NoError(t, err)
		}()
	}

	wg.Wait()
	require.Equal(t, int32(1), maxActive)
}

func TestRoomActorRejectsCommandAfterClose(t *testing.T) {
	room, _ := newTestRoom(t)
	room.CloseAllClients()

	err := room.HandleReady("room-1-player-0", SubmitReadyMessage{RoomID: "room-1"})
	require.Error(t, err)
}

func TestRoomRejectsStalePromptResponse(t *testing.T) {
	room, clients := newTestRoom(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		_, _ = room.RunTurn(ctx)
	}()

	message := clients[0].awaitMessage(t)
	prompt, ok := message.(PromptEnvelope)
	require.True(t, ok)

	err := room.HandleCommand("room-1-player-0", CommandEnvelopeMessage{
		Type:     api.TypeCommand,
		PromptID: prompt.Prompt.ID + "-old",
		Kind:     string(sessionapp.CommandMainAction),
		Payload: map[string]any{
			"action":     string(coretypes.Income),
			"actorIndex": 0,
		},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, apperrors.ErrStalePrompt)
}

func TestRoomReconnectDuringPromptReceivesSyncAndPrompt(t *testing.T) {
	room, clients := newTestRoom(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		_, _ = room.RunTurn(ctx)
	}()

	firstPrompt, ok := clients[0].awaitMessage(t).(PromptEnvelope)
	require.True(t, ok)
	require.Equal(t, "main_action", firstPrompt.Prompt.Kind)

	reconnected := newStubClient(room.lobby.Session("room-1-player-0"))
	require.True(t, room.Unregister("room-1-player-0", clients[0]))
	require.NoError(t, room.Register(reconnected))
	require.NoError(t, room.sendRoomSync(0))
	require.NoError(t, room.SendPromptToPlayer(0))

	_, ok = reconnected.awaitMessage(t).(api.RoomSyncView)
	require.True(t, ok)
	prompt, ok := reconnected.awaitMessage(t).(PromptEnvelope)
	require.True(t, ok)
	require.Equal(t, firstPrompt.Prompt.ID, prompt.Prompt.ID)
}

func TestRoomSecondSocketReplacesFirstSessionConnection(t *testing.T) {
	room, clients := newTestRoom(t)
	replacement := newStubClient(room.lobby.Session("room-1-player-0"))

	require.NoError(t, room.Register(replacement))
	require.True(t, clients[0].closed)
	require.True(t, room.IsCurrent("room-1-player-0", replacement))
	require.False(t, room.IsCurrent("room-1-player-0", clients[0]))
}

// newTestRoom builds a small hosted room with two registered websocket stubs.
func newTestRoom(t *testing.T) (*Room, []*stubClient) {
	t.Helper()

	lobby, err := sessionapp.NewLobby("room-1", "A", 2)
	require.NoError(t, err)
	first, err := lobby.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	second, err := lobby.JoinPlayer("B", time.Unix(0, 0).UTC())
	require.NoError(t, err)

	room, err := newRoom(lobby, runtimeturn.Config{}, nil)
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
	closed   bool
}

// newStubClient creates a buffered websocket stub that records every sent message.
func newStubClient(session *sessionapp.ClientSession) *stubClient {
	return &stubClient{
		session:  session,
		messages: make(chan any, 8),
	}
}

// Session exposes the underlying client session so the room can read player metadata.
func (c *stubClient) Session() *sessionapp.ClientSession {
	return c.session
}

// Send records a websocket payload for later assertions.
func (c *stubClient) Send(_ context.Context, message any) error {
	c.messages <- message
	return nil
}

// Close closes the message channel to mirror a disconnected websocket.
func (c *stubClient) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	close(c.messages)
	return nil
}

// awaitMessage blocks until the stub receives the next payload or times out.
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

// awaitTurnResult waits specifically for the turn result message in the message stream.
func (c *stubClient) awaitTurnResult(t *testing.T) (api.TurnResultView, bool) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case message := <-c.messages:
			result, ok := message.(api.TurnResultView)
			if ok {
				return result, true
			}
		case <-timeout:
			t.Fatal("timed out waiting for turn result")
			return api.TurnResultView{}, false
		}
	}
}

// assertNoMessage confirms that no payload arrives during a short timeout window.
func (c *stubClient) assertNoMessage(t *testing.T) {
	t.Helper()
	select {
	case message := <-c.messages:
		t.Fatalf("expected no message, got %#v", message)
	case <-time.After(150 * time.Millisecond):
	}
}
