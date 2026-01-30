package ws

import (
	"ElMakina/backend/engine"
	"sync"
	"time"
)

// pausableClock wraps an engine.Clock and halts timeouts while paused.
// When resumed, timers continue with their remaining duration.
type pausableClock struct {
	base engine.Clock

	mu       sync.Mutex
	paused   bool
	pauseCh  chan struct{}
	resumeCh chan struct{}
}

func newPausableClock(base engine.Clock) *pausableClock {
	if base == nil {
		base = engine.RealClock{}
	}
	return &pausableClock{
		base:    base,
		pauseCh: make(chan struct{}),
	}
}

func (c *pausableClock) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.paused {
		return
	}
	c.paused = true
	close(c.pauseCh)
	c.resumeCh = make(chan struct{})
}

func (c *pausableClock) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.paused {
		return
	}
	c.paused = false
	if c.resumeCh != nil {
		close(c.resumeCh)
		c.resumeCh = nil
	}
	c.pauseCh = make(chan struct{})
}

func (c *pausableClock) After(d time.Duration) <-chan time.Time {
	if d <= 0 {
		return nil
	}
	out := make(chan time.Time, 1)
	go func() {
		remaining := d
		for {
			if remaining <= 0 {
				out <- time.Now()
				return
			}
			c.mu.Lock()
			paused := c.paused
			pauseCh := c.pauseCh
			resumeCh := c.resumeCh
			c.mu.Unlock()

			if paused {
				if resumeCh != nil {
					<-resumeCh
				}
				continue
			}

			start := time.Now()
			timerCh := c.base.After(remaining)
			select {
			case <-timerCh:
				out <- time.Now()
				return
			case <-pauseCh:
				elapsed := time.Since(start)
				if elapsed < remaining {
					remaining -= elapsed
				} else {
					remaining = 0
				}
				c.mu.Lock()
				resumeCh = c.resumeCh
				c.mu.Unlock()
				if resumeCh != nil {
					<-resumeCh
				}
			}
		}
	}()
	return out
}
