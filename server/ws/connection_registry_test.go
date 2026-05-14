package ws

import (
	"context"
	"testing"

	sessionapp "github.com/axwell0/elmakina/internal/app/session"
	"github.com/stretchr/testify/require"
)

func TestConnectionRegistryReplacesExistingSessionConnection(t *testing.T) {
	registry := NewConnectionRegistry()
	first := newNamedTestConnection("first")
	second := newNamedTestConnection("second")

	require.NoError(t, registry.Attach("session-1", 0, first))
	require.NoError(t, registry.Attach("session-1", 0, second))

	current, ok := registry.Connection("session-1")
	require.True(t, ok)
	require.Equal(t, second, current)
	require.True(t, first.closed)
	require.False(t, second.closed)
}

func TestConnectionRegistryReplacesExistingPlayerConnection(t *testing.T) {
	registry := NewConnectionRegistry()
	first := newNamedTestConnection("first")
	second := newNamedTestConnection("second")

	require.NoError(t, registry.Attach("session-1", 0, first))
	require.NoError(t, registry.Attach("session-2", 0, second))

	_, ok := registry.Connection("session-1")
	require.False(t, ok)
	current, ok := registry.Connection("session-2")
	require.True(t, ok)
	require.Equal(t, second, current)
	require.True(t, first.closed)
	require.False(t, second.closed)
}

func TestConnectionRegistryDetachIfCurrentProtectsReplacement(t *testing.T) {
	registry := NewConnectionRegistry()
	first := newNamedTestConnection("first")
	second := newNamedTestConnection("second")

	require.NoError(t, registry.Attach("session-1", 0, first))
	require.NoError(t, registry.Attach("session-1", 0, second))

	require.False(t, registry.DetachIfCurrent("session-1", 0, first))
	require.True(t, registry.HasSession("session-1"))
	require.True(t, registry.DetachIfCurrent("session-1", 0, second))
	require.False(t, registry.HasSession("session-1"))
}

func TestConnectionRegistryConnectedPlayersAreSorted(t *testing.T) {
	registry := NewConnectionRegistry()

	require.NoError(t, registry.Attach("session-2", 2, newNamedTestConnection("two")))
	require.NoError(t, registry.Attach("session-0", 0, newNamedTestConnection("zero")))
	require.NoError(t, registry.Attach("session-1", 1, newNamedTestConnection("one")))

	require.Equal(t, []int{0, 1, 2}, registry.ConnectedPlayers())
}

func TestConnectionRegistryCloseAllClearsAndReturnsFirstError(t *testing.T) {
	registry := NewConnectionRegistry()
	first := newNamedTestConnection("first")
	first.closeErr = context.Canceled
	second := newNamedTestConnection("second")

	require.NoError(t, registry.Attach("session-1", 0, first))
	require.NoError(t, registry.Attach("session-2", 1, second))

	err := registry.CloseAll()
	require.ErrorIs(t, err, context.Canceled)
	require.True(t, first.closed)
	require.True(t, second.closed)
	require.Empty(t, registry.ConnectedPlayers())
}

type namedTestConnection struct {
	name     string
	closed   bool
	closeErr error
}

func newNamedTestConnection(name string) *namedTestConnection {
	return &namedTestConnection{name: name}
}

func (c *namedTestConnection) Send(_ context.Context, _ any) error {
	return nil
}

func (c *namedTestConnection) Session() *sessionapp.ClientSession {
	return &sessionapp.ClientSession{ClientSessionID: c.name}
}

func (c *namedTestConnection) Close() error {
	c.closed = true
	return c.closeErr
}
