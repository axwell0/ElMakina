package session

import (
	corestate "example.com/elmakina/engine/core/state"
	ports "example.com/elmakina/engine/ports"
)

// InputKind identifies which kind of turn input is currently open.
type InputKind string

const (
	InputMainAction InputKind = "main_action"
	InputChallenge  InputKind = "challenge"
	InputCounter    InputKind = "counter"
	InputStep       InputKind = "step"
)

// Deprecated: superseded by InteractionState during prompt-path redesign.
//
// OpenInput is a snapshot of the active input window exposed by InputBroker.
type OpenInput struct {
	Kind      InputKind
	Actor     int
	Step      *corestate.StepRequest
	Challenge *ports.ChallengeWindow
	Counter   *ports.CounterWindow
	Revision  int64
}
