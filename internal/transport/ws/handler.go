//nolint:err113,wrapcheck // Upgrade wrapper intentionally forwards websocket accept errors verbatim.
package ws

import (
	"fmt"
	"net/http"

	"github.com/coder/websocket"

	sessionapp "github.com/axwell0/elmakina/internal/app/session"
)

// UpgradeConnection accepts the websocket handshake and wraps the socket in the room transport.
func UpgradeConnection(w http.ResponseWriter, r *http.Request, session *sessionapp.ClientSession, subprotocol string) (*Connection, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}
	// Only advertise a subprotocol when the client actually negotiated one.
	var opts *websocket.AcceptOptions
	if subprotocol != "" {
		// Echo the exact negotiated subprotocol back to the browser during the upgrade.
		opts = &websocket.AcceptOptions{Subprotocols: []string{subprotocol}}
	}
	// Accept upgrades the HTTP request to a websocket and returns the live socket.
	socket, err := websocket.Accept(w, r, opts)
	if err != nil {
		return nil, err
	}
	return NewConnection(session, socket), nil
}
