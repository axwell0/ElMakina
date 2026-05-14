package turn

import (
	"context"
	"time"

	coreactions "github.com/axwell0/elmakina/engine/core/actions"
	corestate "github.com/axwell0/elmakina/engine/core/state"
	coretypes "github.com/axwell0/elmakina/engine/core/types"
)

type InputProvider interface {
	RequestAction(ctx context.Context, actor int, gs *corestate.GameState) (*coretypes.PlayerAction, error)
	ChallengeResponses(ctx context.Context, window ChallengeWindow) (<-chan ChallengeDecision, error)
	CounterResponses(ctx context.Context, window CounterWindow) (<-chan CounterDecision, error)
	RequestCardSelection(ctx context.Context, req coreactions.StepRequest) (coreactions.CardSelectionResponse, error)
}

type ChallengeDecision struct {
	ChallengerIndex        int
	ChallengerDiscardIndex int
	ActorDiscardIndex      int
	ActorProvingCardIndex  int
	Pass                   bool
}

type CounterDecision struct {
	PlayerIndex int
	Pass        bool
	Command     *coretypes.PlayerAction
}

type ChallengeKind string

const (
	ChallengeMain    ChallengeKind = "main"
	ChallengeCounter ChallengeKind = "counter"
)

type ChallengeWindow struct {
	Kind               ChallengeKind
	Action             *coretypes.PlayerAction
	ActorIndex         int
	ClaimedRole        coretypes.Role
	AllowedChallengers []int
}

type CounterWindow struct {
	MainAction             *coretypes.PlayerAction
	AllowedPlayers         []int
	AllowedActions         []coretypes.ActionID
	AllowedActionsByPlayer map[int][]coretypes.ActionID
}

type Clock interface {
	After(d time.Duration) <-chan time.Time
}

type RealClock struct{}

func (RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type Config struct {
	ChallengeTimeout time.Duration
	CounterTimeout   time.Duration
	TurnTimeout      time.Duration
}
