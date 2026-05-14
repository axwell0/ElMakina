// Package room implements the Actor Model for safe concurrent room mutations.
//
// # Actor Model
//
// The problem: a room is shared state (player list, game session, connection registry)
// that multiple goroutines touch concurrently — the WebSocket reader loop, the turn
// loop, reconnect handlers, etc. The naive solution is to scatter mutexes everywhere,
// but that's error-prone and easy to deadlock.
//
// The Actor Model solution: instead of protecting state with locks, one single goroutine
// owns all room state. Every other goroutine that wants to mutate the room sends it a
// function to execute. The owner goroutine runs those functions one at a time — which
// means mutations are automatically serialized with no locks needed on the data itself.
//
// Think of it like a single cashier serving a queue: customers (goroutines) hand the
// cashier (actor) their request, and the cashier handles them one at a time.
//
//nolint:contextcheck,wrapcheck // Actor queues intentionally accept root contexts and bubble cancellation directly.
package room

import (
	"context"
	"errors"
)

var ErrClosed = errors.New("room actor closed")

// Actor is the serializing goroutine that owns a room's mutable state.
//
// The key field is commands: it's a channel of functions (not data). Callers package
// their mutation as a closure and send it here. The single run() goroutine pulls
// closures off the channel and executes them one by one — serialization guaranteed.
type Actor struct {
	commands chan func()   // mutations arrive as closures; executed one at a time by run()
	done     chan struct{} // closed by run() when it exits; Close() waits on this
	closed   chan struct{} // closed by Close() to signal run() to stop
}

// NewActor creates an Actor and starts its background goroutine immediately.
// buffer controls how many pending commands can queue up before senders block.
func NewActor(buffer int) *Actor {
	if buffer <= 0 {
		buffer = 1
	}
	actor := &Actor{
		commands: make(chan func(), buffer),
		done:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
	go actor.run()
	return actor
}

// run is the single goroutine that processes all commands.
// This is the heart of the actor model — only this goroutine executes mutations,
// so callers never need to lock the data they mutate inside their commands.
func (a *Actor) run() {
	defer close(a.done) // signal Close() that we've fully stopped
	for {
		select {
		case command := <-a.commands:
			if command != nil {
				command()
			}
		case <-a.closed:
			return
		}
	}
}

// Do sends a function to the actor and waits for it to finish, returning its error.
// Use this when the caller needs to know the mutation succeeded before proceeding
// (e.g. "add player, then send them the room snapshot").
//
// How it works: Do wraps fn in a closure that sends the result back on a reply channel,
// then blocks until either the reply arrives or the context is canceled.
func (a *Actor) Do(ctx context.Context, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	// Buffer of 1 so the actor goroutine never blocks when writing the result back —
	// even if the caller's ctx has already been canceled and nobody is reading.
	reply := make(chan error, 1)
	if err := a.Enqueue(ctx, func() {
		reply <- fn()
	}); err != nil {
		return err
	}
	select {
	case err := <-reply:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DoBool is like Do but for functions that return a bool instead of an error.
// Used for read-queries that need the actor's serialization guarantee
// (e.g. "is the room ready to start?").
func (a *Actor) DoBool(ctx context.Context, fn func() bool) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	reply := make(chan bool, 1)
	if err := a.Enqueue(ctx, func() {
		reply <- fn()
	}); err != nil {
		return false
	}
	select {
	case ok := <-reply:
		return ok
	case <-ctx.Done():
		return false
	}
}

// Enqueue sends a function to the actor without waiting for it to finish.
// Use this for fire-and-forget mutations where the caller doesn't need a result
// (e.g. "broadcast this message to all clients").
//
// The double-select pattern here first does a non-blocking check for closure
// (to avoid queuing work into a stopped actor), then does a blocking send
// that also watches for closure and context cancellation.
func (a *Actor) Enqueue(ctx context.Context, fn func()) error {
	if a == nil {
		return ErrClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Fast non-blocking check: is the actor already closed?
	// This avoids queuing a command that will never run.
	select {
	case <-a.closed:
		return ErrClosed
	default:
	}
	// Blocking send: wait until the command channel has space,
	// the actor closes, or the caller's context expires.
	select {
	case a.commands <- fn:
		return nil
	case <-a.closed:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the actor and waits for the background goroutine to fully exit.
// After Close returns, no more commands will be executed. Safe to call multiple times.
//
// The select-on-closed pattern is Go's idiomatic "close once" guard:
// closing an already-closed channel panics, so we check first.
func (a *Actor) Close() {
	if a == nil {
		return
	}
	select {
	case <-a.closed:
		return // already closed, nothing to do
	default:
		close(a.closed) // signal run() to stop
	}
	<-a.done // wait for run() to fully exit before returning
}
