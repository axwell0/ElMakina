package actions

import (
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"context"
	"fmt"
)

// ActionFlow manages the communication between the action goroutine and the engine.
//
// This is the mechanism that allows actions to pause mid-execution
// and wait for player input. Actions like Exchange need to:
//  1. Draw cards from deck
//  2. [PAUSE] Ask player which cards to return
//  3. Return selected cards
//
// Without ActionFlow, we'd need to split this into multiple functions and manage
// state manually. With ActionFlow, the action code reads linearly and naturally.
//
// The flow uses three channels:
//   - Requests: Action sends StepRequest when it needs input
//   - Responses: Engine sends the player's response
//   - Errs: Action sends errors if it fails
//   - Done: Closed when action completes (success or failure)
type ActionFlow struct {
	Requests  chan state.StepRequest // Outbound: action requests input
	Responses chan any               // Inbound: engine provides response
	Errs      chan error             // Outbound: action reports errors
	Done      chan struct{}          // Closed when action finishes
}

// NewActionFlow creates a new ActionFlow with unbuffered channels.
// Unbuffered channels ensure synchronous handoff between action and engine.
func NewActionFlow() *ActionFlow {
	return &ActionFlow{
		Requests:  make(chan state.StepRequest),
		Responses: make(chan any),
		Errs:      make(chan error, 1),
		Done:      make(chan struct{}),
	}
}

// Yield pauses the action and waits for input from the engine.
//
// This is the blocking call that actions make when they need player input.
// The action goroutine blocks here until the engine calls FlowRunner.Next()
// with a response.
//
// INTERWORKING WITH Next():
//   1. Action calls Yield(req) → sends req to Requests channel, blocks on Responses
//   2. Engine calls Next(input) → sends input to Responses channel, blocks on Requests
//   3. Action receives input from Responses, unblocks and continues
//   4. Engine receives next req from Requests, unblocks and returns it
//
// This creates a synchronous handoff: the action and engine take turns blocking
// on each other's channels, ensuring they never race.
//
// Example usage in applyExchange:
//
//	indices := flow.RequestCardSelection(...)
//	// Action blocks here until engine provides card indices

func (f *ActionFlow) Yield(req state.StepRequest) any {
	f.Requests <- req
	resp := <-f.Responses
	return resp
}

// RequestCardSelection is a type-safe helper for requesting card indices from a player.
//
// Wraps Yield with type assertion to ensure we get []int back.
// Used by actions like Exchange (select cards to return) and Coup/Assassinate
// (select card to discard).
func (f *ActionFlow) RequestCardSelection(actionID models.ActionID, actor int, count int, context state.StepContext, options []string) ([]int, error) {
	resp := f.Yield(state.StepRequest{
		ActionID: actionID,
		Kind:     state.StepCardSelect,
		Actor:    actor,
		Count:    count,
		Context:  context,
		Options:  options,
	})

	indices, ok := resp.([]int)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected []int, got %T", resp)
	}
	return indices, nil
}

// Result allows the action to terminate with an error or nil.
//
// Called by the action function when it completes (successfully or with error).
// If err is non-nil, it's sent to the Errs channel. The Done channel is always
// closed to signal completion.
func (f *ActionFlow) Result(err error) {
	if err != nil {
		f.Errs <- err
	}
	close(f.Done)
}

// FlowRunner manages the lifecycle of an ActionFlow.
//
// Created by StartAction, this struct gives the engine control over the
// running action goroutine. The engine can:
//   - Call Next() to advance the action with input
//   - Call Cancel() to abort the action
//   - Call GetResult() to check for errors
type FlowRunner struct {
	Flow   *ActionFlow
	Cancel context.CancelFunc // Cancels the action goroutine
}

// StartAction launches an action in a goroutine and returns a FlowRunner.
//
// The action function receives an ActionFlow and can call Yield() to pause
// and request input. The goroutine runs until the action returns or is canceled.
//
// Example:
//
//	runner := StartAction(ctx, func(f *ActionFlow) error {
//	    return actions.ApplyAction(f, state, turn, log, cmd)
//	})
//	// Later: runner.Next(input) to advance the action
func StartAction(ctx context.Context, action func(f *ActionFlow) error) *FlowRunner {
	flow := NewActionFlow()
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer close(flow.Requests)
		err := action(flow)
		flow.Result(err)
	}()

	return &FlowRunner{
		Flow:   flow,
		Cancel: cancel,
	}
}

// Next advances the flow with a new result.
//
// Called by the engine to provide input to a paused action. Returns:
//   - (*StepRequest, nil): Action yielded and is waiting for more input
//   - (nil, error): Action failed
//   - (nil, nil): Action completed successfully
//
// This method handles the channel dance:
//  1. Send input to Responses channel
//  2. Wait for action to either yield again or finish
//  3. Return the next request or completion status
func (r *FlowRunner) Next(input any) (*state.StepRequest, error) {
	select {
	case r.Flow.Responses <- input:
		// Wait for next yield or completion
		select {
		case req, ok := <-r.Flow.Requests:
			if !ok {
				// Action finished
				return nil, r.GetResult()
			}
			return &req, nil
		case <-r.Flow.Done:
			return nil, r.GetResult()
		}
	case <-r.Flow.Done:
		return nil, r.GetResult()
	}
}

// GetResult returns the action's error if it failed, nil otherwise.
//
// Non-blocking: returns nil immediately if no error was sent.
// Called after Next() returns nil to check if the action succeeded.
func (r *FlowRunner) GetResult() error {
	select {
	case err := <-r.Flow.Errs:
		return err
	default:
		return nil
	}
}
