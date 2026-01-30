package ws

import (
	"ElMakina/backend/engine"
	"sync"
	"time"
)

// ForfeitScheduler delays forfeits to allow brief reconnect windows.
//
// Intent:
//
//	Keep game flow responsive while tolerating short-lived disconnects.
//	Scheduling is keyed by player ID; the latest schedule wins.
type ForfeitScheduler struct {
	clock  engine.Clock
	grace  time.Duration
	mu     sync.Mutex
	tokens map[string]uint64
}

// NewForfeitScheduler builds a scheduler with the provided clock and grace duration.
func NewForfeitScheduler(clock engine.Clock, grace time.Duration) *ForfeitScheduler {
	if clock == nil {
		clock = engine.RealClock{}
	}
	return &ForfeitScheduler{
		clock:  clock,
		grace:  grace,
		tokens: make(map[string]uint64),
	}
}

// Schedule triggers a forfeit after the grace period unless canceled.
func (s *ForfeitScheduler) Schedule(playerID string, forfeit func()) {
	s.mu.Lock()
	s.tokens[playerID]++
	token := s.tokens[playerID]
	s.mu.Unlock()

	if s.grace <= 0 {
		forfeit()
		return
	}

	wait := s.clock.After(s.grace)
	go func(expected uint64) {
		<-wait
		s.mu.Lock()
		current := s.tokens[playerID]
		s.mu.Unlock()
		if current != expected {
			return
		}
		forfeit()
	}(token)
}

// Cancel invalidates any pending forfeit for the player.
func (s *ForfeitScheduler) Cancel(playerID string) {
	s.mu.Lock()
	s.tokens[playerID]++
	s.mu.Unlock()
}
