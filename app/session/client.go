package session

import "context"

// Deprecated: transport bindings are moving into server/ws ConnectionRegistry.
//
// Client is the transport-facing live connection attached to a client session.
type Client interface {
	Session() *ClientSession
	Send(ctx context.Context, message any) error
	Close() error
}
