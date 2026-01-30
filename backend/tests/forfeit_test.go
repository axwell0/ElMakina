package tests

import (
	"ElMakina/backend/server/ws"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu    sync.Mutex
	chans []chan time.Time
}

func (f *fakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	f.mu.Lock()
	f.chans = append(f.chans, ch)
	f.mu.Unlock()
	return ch
}

func (f *fakeClock) Fire() {
	f.mu.Lock()
	if len(f.chans) == 0 {
		f.mu.Unlock()
		return
	}
	ch := f.chans[0]
	f.chans = f.chans[1:]
	f.mu.Unlock()
	ch <- time.Now()
}

func TestForfeitSchedulerFiresAfterGrace(t *testing.T) {
	clock := &fakeClock{}
	scheduler := ws.NewForfeitScheduler(clock, time.Second)

	called := make(chan string, 1)
	scheduler.Schedule("p1", func() { called <- "p1" })
	clock.Fire()

	select {
	case player := <-called:
		if player != "p1" {
			t.Fatalf("expected p1, got %s", player)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected forfeit to fire")
	}
}

func TestForfeitSchedulerCancel(t *testing.T) {
	clock := &fakeClock{}
	scheduler := ws.NewForfeitScheduler(clock, time.Second)

	called := make(chan string, 1)
	scheduler.Schedule("p1", func() { called <- "p1" })
	scheduler.Cancel("p1")
	clock.Fire()

	select {
	case <-called:
		t.Fatalf("did not expect forfeit after cancel")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestForfeitSchedulerImmediate(t *testing.T) {
	scheduler := ws.NewForfeitScheduler(nil, 0)
	called := make(chan string, 1)
	scheduler.Schedule("p1", func() { called <- "p1" })
	select {
	case <-called:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("expected immediate forfeit")
	}
}
