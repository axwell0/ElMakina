package websocket

import (
	"fmt"
	"sync/atomic"
)

func (p *Provider) registerConn(playerIndex int, conn wsConn) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.conns[playerIndex]; ok {
		return fmt.Errorf("player %d already connected", playerIndex)
	}
	state := &connState{index: playerIndex, conn: conn}
	p.conns[playerIndex] = state
	go p.readLoop(state)
	return nil
}

func (p *Provider) readLoop(state *connState) {
	for {
		var env Envelope
		if err := state.conn.ReadJSON(&env); err != nil {
			_ = state.conn.Close()
			p.unregisterConn(state.index, err)
			return
		}
		if env.Type == msgHello || env.RequestID == "" {
			continue
		}
		p.dispatch(state.index, env)
	}
}

func (p *Provider) unregisterConn(playerIndex int, err error) {
	p.mu.Lock()
	delete(p.conns, playerIndex)
	p.mu.Unlock()
	p.notifyError(fmt.Errorf("player %d disconnected: %w", playerIndex, err))
}

func (p *Provider) sendTo(player int, env Envelope) error {
	p.mu.RLock()
	state := p.conns[player]
	p.mu.RUnlock()
	if state == nil {
		return fmt.Errorf("player %d not connected", player)
	}
	state.sendMu.Lock()
	err := state.conn.WriteJSON(env)
	state.sendMu.Unlock()
	return err
}

func (p *Provider) dispatch(sender int, env Envelope) {
	p.mu.RLock()
	waiter := p.pending[env.RequestID]
	p.mu.RUnlock()
	if waiter == nil {
		// Protocol violation: clients must only respond to active request_id values.
		p.notifyError(fmt.Errorf("unknown request_id %q from player %d", env.RequestID, sender))
		return
	}
	select {
	case waiter.ch <- responseEnvelope{sender: sender, env: env}:
	default:
		p.notifyError(fmt.Errorf("response buffer full for request %s", env.RequestID))
	}
}

func (p *Provider) registerWaiter(reqID string) *waiter {
	p.mu.Lock()
	defer p.mu.Unlock()
	w := &waiter{ch: make(chan responseEnvelope, 16)}
	p.pending[reqID] = w
	return w
}

func (p *Provider) removeWaiter(reqID string) {
	p.mu.Lock()
	delete(p.pending, reqID)
	p.mu.Unlock()
}

func (p *Provider) nextRequestID() string {
	id := atomic.AddUint64(&p.nextID, 1)
	return fmt.Sprintf("req-%d", id)
}

func (p *Provider) notifyError(err error) {
	select {
	case p.errCh <- err:
	default:
	}
}
