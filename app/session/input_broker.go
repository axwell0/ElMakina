package session

import (
	"context"
	"fmt"
	"slices"
	"sync"

	corestate "example.com/elmakina/engine/core/state"
	coretypes "example.com/elmakina/engine/core/types"
	ports "example.com/elmakina/engine/ports"
)

// InputBroker is runtime-facing plumbing only. InteractionCoordinator owns the
// application-facing live interaction state.
//
// InputBroker owns the currently open turn input window and forwards validated
// submissions back to the runtime through the ports.InputProvider contract.
type InputBroker struct {
	mu       sync.RWMutex
	open     *OpenInput
	revision int64
	events   chan OpenInput

	mainAction chan *coretypes.PlayerAction
	challenge  chan ports.ChallengeDecision
	counter    chan ports.CounterDecision
	step       chan corestate.StepResponse
	errs       chan error
}

// NewInputBroker creates a new broker with independent channels per input kind.
func NewInputBroker() *InputBroker {
	return &InputBroker{
		events:     make(chan OpenInput, 8),
		mainAction: make(chan *coretypes.PlayerAction),
		challenge:  make(chan ports.ChallengeDecision),
		counter:    make(chan ports.CounterDecision),
		step:       make(chan corestate.StepResponse),
		errs:       make(chan error, 1),
	}
}

// Events emits snapshots whenever a new input window opens.
func (b *InputBroker) Events() <-chan OpenInput {
	return b.events
}

// Current returns a defensive copy of the currently open input window.
func (b *InputBroker) Current() *OpenInput {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return cloneOpenInput(b.open)
}

// Errors exposes asynchronous provider errors for runtime helpers.
func (b *InputBroker) Errors() <-chan error {
	return b.errs
}

// RequestAction opens a main-action window and waits for a matching submission.
func (b *InputBroker) RequestAction(ctx context.Context, actor int, _ *corestate.GameState) (*coretypes.PlayerAction, error) {
	rev := b.openWindow(OpenInput{
		Kind:  InputMainAction,
		Actor: actor,
	})
	defer b.clearWindowIfRevision(rev)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case action := <-b.mainAction:
		return action, nil
	}
}

// ChallengeResponses opens a challenge window and returns the decision stream.
func (b *InputBroker) ChallengeResponses(ctx context.Context, window ports.ChallengeWindow) (<-chan ports.ChallengeDecision, error) {
	rev := b.openWindow(OpenInput{
		Kind:      InputChallenge,
		Actor:     window.ActorIndex,
		Challenge: cloneChallengeWindow(&window),
	})
	go func() {
		<-ctx.Done()
		b.clearWindowIfRevision(rev)
	}()
	return b.challenge, nil
}

// CounterResponses opens a counter window and returns the decision stream.
func (b *InputBroker) CounterResponses(ctx context.Context, window ports.CounterWindow) (<-chan ports.CounterDecision, error) {
	rev := b.openWindow(OpenInput{
		Kind:    InputCounter,
		Actor:   -1,
		Counter: cloneCounterWindow(&window),
	})
	go func() {
		<-ctx.Done()
		b.clearWindowIfRevision(rev)
	}()
	return b.counter, nil
}

// RequestCardSelection opens a step window and waits for a step response.
func (b *InputBroker) RequestCardSelection(ctx context.Context, req corestate.StepRequest) (corestate.CardSelectionResponse, error) {
	rev := b.openWindow(OpenInput{
		Kind:  InputStep,
		Actor: req.Actor,
		Step:  cloneStepRequest(&req),
	})
	defer b.clearWindowIfRevision(rev)

	select {
	case <-ctx.Done():
		return corestate.CardSelectionResponse{}, ctx.Err()
	case response := <-b.step:
		if response.CardSelection == nil {
			return corestate.CardSelectionResponse{}, fmt.Errorf("step response missing card selection payload")
		}
		return *response.CardSelection, nil
	}
}

// SubmitMainAction validates and forwards a main-action submission.
func (b *InputBroker) SubmitMainAction(_ string, actor int, action *coretypes.PlayerAction) error {
	current := b.Current()
	if current == nil || current.Kind != InputMainAction {
		return errInputNotOpen(InputMainAction)
	}
	if current.Actor != actor {
		return fmt.Errorf("player %d is not allowed to answer %s", actor, InputMainAction)
	}
	if action == nil {
		return fmt.Errorf("main action is nil")
	}
	if action.ActorIndex != actor {
		return fmt.Errorf("action actor %d does not match expected player %d", action.ActorIndex, actor)
	}
	b.mainAction <- action
	return nil
}

// SubmitChallenge validates and forwards a challenge decision.
func (b *InputBroker) SubmitChallenge(_ string, playerIndex int, decision ports.ChallengeDecision) error {
	current := b.Current()
	if current == nil || current.Kind != InputChallenge || current.Challenge == nil {
		return errInputNotOpen(InputChallenge)
	}
	if !slices.Contains(current.Challenge.AllowedChallengers, playerIndex) {
		return fmt.Errorf("player %d is not allowed to answer %s", playerIndex, InputChallenge)
	}
	if decision.ChallengerIndex != playerIndex {
		return fmt.Errorf("challenge player %d does not match expected player %d", decision.ChallengerIndex, playerIndex)
	}
	b.challenge <- decision
	return nil
}

// SubmitCounter validates and forwards a counter decision.
func (b *InputBroker) SubmitCounter(_ string, playerIndex int, decision ports.CounterDecision) error {
	current := b.Current()
	if current == nil || current.Kind != InputCounter || current.Counter == nil {
		return errInputNotOpen(InputCounter)
	}
	if !slices.Contains(current.Counter.AllowedPlayers, playerIndex) {
		return fmt.Errorf("player %d is not allowed to answer %s", playerIndex, InputCounter)
	}
	if decision.PlayerIndex != playerIndex {
		return fmt.Errorf("counter player %d does not match expected player %d", decision.PlayerIndex, playerIndex)
	}
	b.counter <- decision
	return nil
}

// SubmitStep validates and forwards a step response.
func (b *InputBroker) SubmitStep(_ string, actor int, response corestate.StepResponse) error {
	current := b.Current()
	if current == nil || current.Kind != InputStep || current.Step == nil {
		return errInputNotOpen(InputStep)
	}
	if current.Actor != actor {
		return fmt.Errorf("player %d is not allowed to answer %s", actor, InputStep)
	}
	if response.Kind != current.Step.Kind {
		return fmt.Errorf("step kind %q does not match expected kind %q", response.Kind, current.Step.Kind)
	}
	b.step <- response
	return nil
}

func (b *InputBroker) openWindow(next OpenInput) int64 {
	b.mu.Lock()
	b.revision++
	next.Revision = b.revision
	b.open = cloneOpenInput(&next)
	snapshot := *b.open
	b.mu.Unlock()
	b.publish(snapshot)
	return next.Revision
}

func (b *InputBroker) clearWindowIfRevision(revision int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.open != nil && b.open.Revision == revision {
		b.open = nil
	}
}

func (b *InputBroker) publish(snapshot OpenInput) {
	select {
	case b.events <- snapshot:
	default:
		// Keep the freshest snapshot if consumers lag behind.
		select {
		case <-b.events:
		default:
		}
		select {
		case b.events <- snapshot:
		default:
		}
	}
}

func errInputNotOpen(kind InputKind) error {
	return fmt.Errorf("input %q is not currently open", kind)
}

func cloneOpenInput(input *OpenInput) *OpenInput {
	if input == nil {
		return nil
	}
	copy := *input
	copy.Step = cloneStepRequest(input.Step)
	copy.Challenge = cloneChallengeWindow(input.Challenge)
	copy.Counter = cloneCounterWindow(input.Counter)
	return &copy
}

func cloneStepRequest(input *corestate.StepRequest) *corestate.StepRequest {
	if input == nil {
		return nil
	}
	copy := *input
	if input.CardSelection != nil {
		selection := *input.CardSelection
		selection.Cards = append([]corestate.CardOption(nil), input.CardSelection.Cards...)
		copy.CardSelection = &selection
	}
	return &copy
}

func cloneChallengeWindow(input *ports.ChallengeWindow) *ports.ChallengeWindow {
	if input == nil {
		return nil
	}
	copy := *input
	copy.AllowedChallengers = append([]int(nil), input.AllowedChallengers...)
	if input.Action != nil {
		action := *input.Action
		copy.Action = &action
	}
	return &copy
}

func cloneCounterWindow(input *ports.CounterWindow) *ports.CounterWindow {
	if input == nil {
		return nil
	}
	cloned := *input
	cloned.AllowedPlayers = append([]int(nil), input.AllowedPlayers...)
	cloned.AllowedActions = append([]coretypes.ActionID(nil), input.AllowedActions...)
	if input.MainAction != nil {
		action := *input.MainAction
		cloned.MainAction = &action
	}
	return &cloned
}

var _ ports.InputProvider = (*InputBroker)(nil)
