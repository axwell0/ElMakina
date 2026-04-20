package ws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectionRegistryReplacesExistingSessionConnection(t *testing.T) {
	registry := NewConnectionRegistry()
	first := newNamedTestConnection("first")
	second := newNamedTestConnection("second")

	registry.Attach("session-1", 0, first)
	registry.Attach("session-1", 0, second)

	current, ok := registry.Connection("session-1")
	require.True(t, ok)
	require.Equal(t, second, current)
}

type namedTestConnection struct {
	name string
}

func newNamedTestConnection(name string) *namedTestConnection {
	return &namedTestConnection{name: name}
}

func (c *namedTestConnection) Send(_ context.Context, _ any) error {
	return nil
}

func (c *namedTestConnection) Close() error {
	return nil
}
