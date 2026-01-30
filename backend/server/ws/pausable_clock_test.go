package ws

import (
	"testing"
	"time"
)

type fakeClock struct {
	called  chan struct{}
	timerCh chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{called: make(chan struct{})}
}

func (f *fakeClock) After(d time.Duration) <-chan time.Time {
	f.timerCh = make(chan time.Time, 1)
	select {
	case <-f.called:
	default:
		close(f.called)
	}
	return f.timerCh
}

func TestPausableClockPausesUntilResume(t *testing.T) {
	base := newFakeClock()
	clock := newPausableClock(base)

	clock.Pause()
	out := clock.After(10 * time.Millisecond)

	select {
	case <-base.called:
		t.Fatalf("expected base clock not to be called while paused")
	default:
	}

	clock.Resume()
	select {
	case <-base.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected base clock After to be called after resume")
	}

	base.timerCh <- time.Now()
	select {
	case <-out:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected timer to fire after resume")
	}
}
