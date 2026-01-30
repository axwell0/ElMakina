package ws

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/server"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatMessageBroadcastsToSession(t *testing.T) {
	manager := server.NewLobbyManager(2, 9)
	leader, err := manager.RegisterPlayer("leader")
	require.NoError(t, err)
	p2, err := manager.RegisterPlayer("p2")
	require.NoError(t, err)

	lobby, err := manager.CreateLobby(leader.ID)
	require.NoError(t, err)
	require.NoError(t, manager.JoinLobby(lobby.ID, p2.ID))

	session, err := manager.StartLobby(lobby.ID, leader.ID)
	require.NoError(t, err)

	runner := newSessionRunner(session, engine.RealClock{}, engine.TurnConfig{}, 0, nil)

	leaderConn := &fakeConn{}
	p2Conn := &fakeConn{}

	leaderClient := &clientConn{player: leader, conn: leaderConn}
	p2Client := &clientConn{player: p2, conn: p2Conn}

	runner.provider.attach(session.IndexByPlayer[leader.ID], leaderClient)
	runner.provider.attach(session.IndexByPlayer[p2.ID], p2Client)

	env := Envelope{Type: MsgChatMessage, Payload: mustJSON(chatIncomingPayload{Text: "hi"})}
	handled := runner.HandleClientMessage(session.IndexByPlayer[leader.ID], env)
	require.True(t, handled)

	require.True(t, containsType(leaderConn.writes, MsgChatMessage))
	require.True(t, containsType(p2Conn.writes, MsgChatMessage))
}
