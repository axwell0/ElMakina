package engine

import (
	"context"
	"time"

	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

// InputProvider is the "Sensors" for the Engine.
//
// The Engine is like a brain in a jar—it has no eyes or ears. It only knows what's
// happening because the InputProvider tells it.
//
// This is an interface because different environments provide input differently:
// - In a real game, this is a bridge to WebSockets (the `server/ws` package).
// - In a test, this is a "Mock" that just returns whatever action we want to test.
// - In a CLI tool, this might be a `fmt.Scanln` prompt.
//
// By keeping this as an interface, the Engine remains "Pure". It doesn't care
// if a human clicked a button or a script sent a JSON packet; it just needs
// an PlayerAction to process.
//
// See: orchestrator_turn.go:29 for usage in turn flow.
// See: server/ws/session_provider.go for WebSocket implementation.
// See: orchestrator_turn_test.go:12 for test stub implementation.
type InputProvider interface {
	// RequestAction blocks until the current player chooses a main action.
	// Receives the live GameState; implementations must treat it as read-only.
	RequestAction(ctx context.Context, actor int, gs *state.GameState) (*models.PlayerAction, error)

	// ChallengeResponses returns a channel that yields challenge decisions.
	// Multiple players may challenge; first valid challenge wins.
	ChallengeResponses(ctx context.Context, window ChallengeWindow) (<-chan ChallengeDecision, error)

	// CounterResponses returns a channel that yields counter decisions.
	// Counters arrive in order; first valid counter may close the window.
	CounterResponses(ctx context.Context, window CounterWindow) (<-chan CounterDecision, error)

	// RequestStep blocks for multi-step action input (card selection, etc.).
	RequestStep(ctx context.Context, req state.StepRequest) (any, error)
}

// Clock abstracts time for timeout handling. RealClock uses time.After,
// test implementations can use deterministic fake clocks.
//
// See: RealClock implementation below.
// See: orchestrator_turn_test.go:52 for testClock implementation.
type Clock interface {
	After(d time.Duration) <-chan time.Time
}

type RealClock struct{}

func (RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// TurnConfig configures timeout durations for each phase of a turn.
type TurnConfig struct {
	ChallengeTimeout time.Duration // Max time for players to challenge
	CounterTimeout   time.Duration // Max time for players to counter
	TurnTimeout      time.Duration // Max time for current player to choose action
}

type ChallengeKind string

const (
	ChallengeMain    ChallengeKind = "main"    // Challenging the main action
	ChallengeCounter ChallengeKind = "counter" // Challenging a counter action
)

// ChallengeWindow describes the context for a challenge phase. Passed to
// InputProvider.ChallengeResponses so it knows who can challenge and what role.
type ChallengeWindow struct {
	Kind               ChallengeKind         // Main or counter challenge
	Action             *models.PlayerAction // Action being challenged
	ActorIndex         int                   // Player who claimed the role
	ClaimedRole        models.Role           // Role being challenged
	AllowedChallengers []int                 // Players eligible to challenge
}

// CounterWindow describes the context for a counter phase. Passed to
// InputProvider.CounterResponses so it knows who can counter and with what.
type CounterWindow struct {
	MainAction     *models.PlayerAction // Action being countered
	AllowedPlayers []int                 // Players eligible to counter
	AllowedActions []models.ActionID     // Valid counter action IDs
}
