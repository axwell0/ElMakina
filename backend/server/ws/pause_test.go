package ws

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/server"
	"testing"
	"time"
)

func TestSessionProviderPauseBlocksDispatch(t *testing.T) {
	session := &server.GameSession{}
	provider := newSessionProvider(session, engine.TurnConfig{}, engine.RealClock{}, nil)
	reqID, ch := provider.register()

	provider.Pause()

	done := make(chan struct{})
	go func() {
		provider.dispatch(0, Envelope{RequestID: reqID})
		close(done)
	}()

	select {
	case _, ok := <-done:
		if !ok {
			t.Fatalf("dispatch completed while paused")
		}
	case <-time.After(50 * time.Millisecond):
	}

	provider.Resume()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("dispatch did not resume")
	}

	select {
	case <-ch:
	default:
		t.Fatalf("expected response to be delivered after resume")
	}
}
