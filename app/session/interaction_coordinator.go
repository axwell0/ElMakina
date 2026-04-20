package session

import "sync"

type InteractionCoordinator struct {
	mu      sync.RWMutex
	current *InteractionState
	events  chan InteractionState
}

func NewInteractionCoordinator() *InteractionCoordinator {
	return &InteractionCoordinator{
		events: make(chan InteractionState, 8),
	}
}

func (c *InteractionCoordinator) Events() <-chan InteractionState {
	return c.events
}

func (c *InteractionCoordinator) Current() *InteractionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.current == nil {
		return nil
	}
	copy := *c.current
	return &copy
}

func (c *InteractionCoordinator) Open(next InteractionState) {
	c.mu.Lock()
	c.current = &next
	snapshot := *c.current
	c.mu.Unlock()

	select {
	case c.events <- snapshot:
	default:
	}
}

func (c *InteractionCoordinator) Clear(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.current != nil && c.current.ID == id {
		c.current = nil
	}
}
