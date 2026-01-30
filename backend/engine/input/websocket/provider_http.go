package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ServeHTTP upgrades the request to a WebSocket and performs the hello handshake.
func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := p.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	var hello Envelope
	if err := conn.ReadJSON(&hello); err != nil {
		_ = conn.Close()
		return
	}
	if hello.Type != msgHello {
		_ = conn.Close()
		p.notifyError(fmt.Errorf("expected hello message"))
		return
	}

	var payload helloPayload
	if err := json.Unmarshal(hello.Payload, &payload); err != nil {
		_ = conn.Close()
		p.notifyError(fmt.Errorf("invalid hello payload: %w", err))
		return
	}
	if payload.PlayerIndex < 0 {
		_ = conn.Close()
		p.notifyError(fmt.Errorf("invalid player index"))
		return
	}

	if err := p.registerConn(payload.PlayerIndex, conn); err != nil {
		_ = conn.Close()
		p.notifyError(err)
		return
	}
}
