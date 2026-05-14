package turn

import (
	"fmt"
	"maps"
	"slices"

	"github.com/axwell0/elmakina/engine/core/types"
)

const NoPlayer = -1

type ChallengeOutcome string

const (
	OutcomeActorProved  ChallengeOutcome = "actor_proved"
	OutcomeActorBluffed ChallengeOutcome = "actor_bluffed"
	OutcomeNoChallenge  ChallengeOutcome = "no_challenge"
)

// ChallengeSummary summarizes one challenge window.
type ChallengeSummary struct {
	ActionID        types.ActionID
	ActionAllowed   bool
	ActorIndex      int
	ChallengerIndex int
	ClaimedRole     types.Role
	Outcome         ChallengeOutcome
	Logs            []string
	LostCardRole    types.Role
	LostPlayerIndex *int
}

// CounterSummary captures the projected outcome of a single counter action.
type CounterSummary struct {
	Action    *types.PlayerAction
	Applied   bool
	Canceled  bool
	Challenge *ChallengeSummary
}

type TurnLog struct {
	Public  []string
	Private map[int][]string
}

func (l *TurnLog) Publicf(format string, args ...any) {
	if l == nil {
		return
	}
	l.Public = append(l.Public, fmt.Sprintf(format, args...))
}

func (l *TurnLog) Privatef(playerIndex int, format string, args ...any) {
	if l == nil {
		return
	}
	if l.Private == nil {
		l.Private = make(map[int][]string)
	}
	l.Private[playerIndex] = append(l.Private[playerIndex], fmt.Sprintf(format, args...))
}

// TurnResult is the transport projection of a completed turn.
type TurnResult struct {
	Logs               []string
	PrivateLogs        map[int][]string
	Main               *types.PlayerAction
	MainApplied        bool
	MainCanceled       bool
	CounterSummaries   []CounterSummary
	ChallengeSummaries []ChallengeSummary
}

func BuildResult(main *types.PlayerAction, mainApplied, mainCanceled bool, logs TurnLog, counterSummaries []CounterSummary, challengeSummaries []ChallengeSummary) *TurnResult {
	return &TurnResult{
		Logs:               slices.Clone(logs.Public),
		PrivateLogs:        clonePrivateLogs(logs.Private),
		Main:               main,
		MainApplied:        mainApplied,
		MainCanceled:       mainCanceled,
		CounterSummaries:   slices.Clone(counterSummaries),
		ChallengeSummaries: slices.Clone(challengeSummaries),
	}
}

func clonePrivateLogs(logs map[int][]string) map[int][]string {
	if logs == nil {
		return nil
	}
	clone := maps.Clone(logs)
	for playerIndex, lines := range logs {
		clone[playerIndex] = slices.Clone(lines)
	}
	return clone
}
