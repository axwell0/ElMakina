//nolint:err113,wrapcheck // Transport adapter keeps direct passthrough errors from websocket library.
package ws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/axwell0/elmakina/internal/app/session"
)

// Connection wraps a websocket socket with the session it serves and the locks
// needed to keep concurrent writes and double-closes safe.
type Connection struct {
	clientSession *session.ClientSession
	socket        *websocket.Conn
	writeMu       sync.Mutex
	closeMu       sync.Mutex
	closed        bool
}

// NewConnection binds the accepted websocket to the active client session.
func NewConnection(clientSession *session.ClientSession, socket *websocket.Conn) *Connection {
	// Store the session and socket together so later send/read calls can stay thin.
	return &Connection{clientSession: clientSession, socket: socket}
}

// Session exposes the owning client session so higher layers can recover player metadata.
func (c *Connection) Session() *session.ClientSession {
	return c.clientSession
}

// Send serializes one payload to the websocket while holding the write lock.
func (c *Connection) Send(ctx context.Context, message any) error {
	if c == nil || c.socket == nil {
		return fmt.Errorf("connection socket is nil")
	}
	// Serialize writes so concurrent broadcasts do not interleave on the websocket.
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return wsjson.Write(ctx, c.socket, message)
}

// Read decodes the next websocket message and refreshes the session's last-seen time.
func (c *Connection) Read(ctx context.Context, value any) error {
	if c == nil || c.socket == nil {
		return fmt.Errorf("connection socket is nil")
	}
	if c.clientSession != nil {
		// Reading from the socket counts as activity for the reconnect/resume flow.
		c.clientSession.MarkSeen(time.Now().UTC())
	}
	return wsjson.Read(ctx, c.socket, value)
}

// Close shuts the websocket down once, even if multiple callers race to close it.
func (c *Connection) Close() error {
	if c == nil || c.socket == nil {
		return nil
	}
	// Guard shutdown so duplicate cleanup paths do not call Close more than once.
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.socket.Close(websocket.StatusNormalClosure, "")
}
