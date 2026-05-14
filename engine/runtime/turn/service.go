// Package turn contains the Service that orchestrates one complete game turn.
//
// # Turn Pipeline
//
// Every turn flows through the same pipeline, in order:
//
//	declare main action
//	    └─► open challenge window  (only if the action claims a role — e.g. Steal claims Thief)
//	              └─► collect counters  (only if the main action survived the challenge)
//	                        └─► for each counter: open challenge window for that counter
//	                                  └─► apply all surviving actions
//
// The pipeline is encoded in Service.Run() as a sequence of method calls, each returning
// whether the current phase succeeded before the next phase starts.
//
// # Partial Results
//
// If anything fails mid-turn (context canceled, bad input, etc.), Run() still returns a
// TurnResult built from whatever was recorded up to that point. This means the WebSocket
// layer always gets something to broadcast — clients won't see a silent failure.
//
// # Deferred Counters
//
// Not all counters cancel the main action. TaxBusinessWoman reduces the main action's
// payout instead of canceling it — so the main action still runs. These non-canceling
// counters are collected in a "deferred" list and applied after the main action completes.
// Canceling counters (BlockSteal, BlockTerrorist, etc.) fire immediately and skip the rest.
package turn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/axwell0/elmakina/engine/core/actions"
	"github.com/axwell0/elmakina/engine/core/state"
	"github.com/axwell0/elmakina/engine/core/types"
	"github.com/axwell0/elmakina/engine/turn"
)

type Service struct {
	State *state.GameState // The live state object that this turn will mutate.
}

// NewService wraps a game state in the turn orchestrator.
func NewService(gs *state.GameState) *Service {
	return &Service{State: gs}
}

// buildProjectedResult turns whatever the service has recorded so far into a
// transport-friendly result. We call it on both success and failure paths so
// partial turn logs still reach the websocket layer.
func buildProjectedResult(gs *state.GameState, main *types.PlayerAction, mainApplied, mainCanceled bool, logs turn.TurnLog, counterSummaries []turn.CounterSummary, challengeSummaries []turn.ChallengeSummary) *turn.TurnResult {
	appendHandSummaries(&logs, gs)
	return turn.BuildResult(main, mainApplied, mainCanceled, logs, counterSummaries, challengeSummaries)
}

// counterPhaseResult collects everything the counter phase produces.
// mainCanceled tells the next phase whether to skip applying the main action.
// deferred holds non-canceling counters that should run after the main action.
type counterPhaseResult struct {
	summaries    []turn.CounterSummary
	challenges   []turn.ChallengeSummary
	mainCanceled bool
	deferred     []deferredCounter
}

// deferredCounter is a counter action that survived challenge but doesn't cancel the
// main action. summaryIndex links it back to its entry in counterPhaseResult.summaries
// so we can mark it "applied" once it runs.
type deferredCounter struct {
	declaration  declaredAction
	summaryIndex int
}

// Run executes one complete turn. It asks for the main action, opens any
// challenge or counter windows, applies the surviving actions, and returns a
// structured result for the websocket layer.
func (s *Service) Run(ctx context.Context, input InputProvider, clock Clock, cfg Config) (*turn.TurnResult, error) {
	if s == nil || s.State == nil {
		return nil, errServiceStateNil
	}
	if input == nil {
		return nil, errInputProviderNil
	}
	if clock == nil {
		return nil, errClockNil
	}

	gs := s.State
	actorIndex := gs.CurrentPlayerIndex
	logs := turn.TurnLog{}
	var challengeSummaries []turn.ChallengeSummary
	var counterSummaries []turn.CounterSummary
	var main *types.PlayerAction
	var mainApplied bool
	var mainCanceled bool

	logs.Publicf("turn %d begins: %s", gs.TurnNumber, playerName(gs, actorIndex))

	mainDecl, err := s.declareMainAction(ctx, input, actorIndex, &logs)
	main = mainDecl.Command
	if err != nil {
		return buildProjectedResult(gs, main, mainApplied, mainCanceled, logs, counterSummaries, challengeSummaries), err
	}

	mainAllowed, mainChallengeSummaries, err := s.resolveMainChallenge(ctx, input, clock, cfg.ChallengeTimeout, mainDecl, &logs)
	challengeSummaries = append(challengeSummaries, mainChallengeSummaries...)
	if err != nil {
		return buildProjectedResult(gs, main, mainApplied, mainCanceled, logs, counterSummaries, challengeSummaries), err
	}

	counterPhase, err := s.handleCounters(ctx, input, clock, cfg, mainDecl, mainAllowed, &logs)
	counterSummaries = append(counterSummaries, counterPhase.summaries...)
	challengeSummaries = append(challengeSummaries, counterPhase.challenges...)
	if err != nil {
		return buildProjectedResult(gs, main, mainApplied, mainCanceled, logs, counterSummaries, challengeSummaries), err
	}

	if err := s.applySurvivingActions(ctx, input, mainDecl, mainAllowed, counterPhase.mainCanceled, counterPhase.deferred, counterSummaries, &logs); err != nil {
		return buildProjectedResult(gs, main, mainApplied, mainCanceled, logs, counterSummaries, challengeSummaries), err
	}

	mainApplied = mainAllowed && !counterPhase.mainCanceled
	mainCanceled = !mainApplied
	logs.Publicf("turn %d ends", gs.TurnNumber)

	return buildProjectedResult(gs, main, mainApplied, mainCanceled, logs, counterSummaries, challengeSummaries), nil
}

func (s *Service) declareMainAction(ctx context.Context, input InputProvider, actorIndex int, logs *turn.TurnLog) (declaredAction, error) {
	mainCmd, err := input.RequestAction(ctx, actorIndex, s.State)
	if err != nil {
		return declaredAction{}, fmt.Errorf("request main action: %w", err)
	}
	if mainCmd == nil {
		return declaredAction{}, errMainActionNil
	}
	if mainCmd.ActorIndex != actorIndex {
		return declaredAction{}, fmt.Errorf("%w: source %d, current %d", errActorMismatch, mainCmd.ActorIndex, actorIndex)
	}

	mainDecl := declaredAction{
		Command: mainCmd,
		Cost:    actions.ActionCost(mainCmd.Action),
	}
	appendActionDeclaration(s.State, logs, mainCmd)

	if err := actions.ValidateMainAction(s.State, mainCmd); err != nil {
		return declaredAction{}, fmt.Errorf("validate main action: %w", err)
	}
	if err := mainDecl.payCost(s.State, logs); err != nil {
		return declaredAction{}, fmt.Errorf("pay main action cost: %w", err)
	}

	return mainDecl, nil
}

func (s *Service) resolveMainChallenge(ctx context.Context, input InputProvider, clock Clock, timeout time.Duration, main declaredAction, logs *turn.TurnLog) (bool, []turn.ChallengeSummary, error) {
	if _, ok := actions.ActionPresentationByID(main.Command.Action); !ok {
		return true, nil, nil
	}
	role := actionRole(main.Command.Action)
	if role == "" {
		return true, []turn.ChallengeSummary{{
			ActionID:        main.Command.Action,
			ActorIndex:      main.Command.ActorIndex,
			ChallengerIndex: turn.NoPlayer,
			ActionAllowed:   true,
			Outcome:         turn.OutcomeNoChallenge,
		}}, nil
	}

	logs.Publicf("challenge window opened for %s (%s)", main.Command.Action, role)
	challengeRes, err := s.runChallengeWindow(ctx, input, clock, timeout, ChallengeMain, main.Command, role, logs)
	if err != nil {
		return false, []turn.ChallengeSummary{challengeRes}, err
	}
	if !challengeRes.ActionAllowed {
		main.refundCost(s.State, logs)
	}
	return challengeRes.ActionAllowed, []turn.ChallengeSummary{challengeRes}, nil
}

func (s *Service) handleCounters(ctx context.Context, input InputProvider, clock Clock, cfg Config, main declaredAction, mainAllowed bool, logs *turn.TurnLog) (counterPhaseResult, error) {
	if !mainAllowed {
		return counterPhaseResult{}, nil
	}

	logs.Publicf("counter window opened for %s", main.Command.Action)
	counters, err := s.collectCounters(ctx, input, clock, cfg.CounterTimeout, main.Command, logs)
	if err != nil {
		return counterPhaseResult{}, err
	}

	var phase counterPhaseResult
	for _, counterCmd := range counters {
		counterDecl, counterSummary, challengeSummary, survives, err := s.resolveCounterDeclaration(ctx, input, clock, cfg.ChallengeTimeout, counterCmd, logs)
		if challengeSummary != nil {
			phase.challenges = append(phase.challenges, *challengeSummary)
		}
		if err != nil {
			return phase, err
		}

		phase.summaries = append(phase.summaries, counterSummary)
		summaryIndex := len(phase.summaries) - 1
		if !survives {
			continue
		}

		counterCancelsMain := true
		if def, ok := actions.CounterAction(counterDecl.Command.Action); ok {
			counterCancelsMain = def.Counter.CancelsMain
		}
		if counterCancelsMain {
			if err := s.applyActionWithSteps(ctx, input, counterDecl.Command, logs); err != nil {
				return phase, err
			}
			phase.summaries[summaryIndex].Applied = true
			phase.mainCanceled = true
			logs.Publicf("main action %s is canceled by %s", main.Command.Action, counterDecl.Command.Action)
			break
		}

		phase.deferred = append(phase.deferred, deferredCounter{
			declaration:  counterDecl,
			summaryIndex: summaryIndex,
		})
	}

	return phase, nil
}

func (s *Service) resolveCounterDeclaration(ctx context.Context, input InputProvider, clock Clock, timeout time.Duration, counterCmd *types.PlayerAction, logs *turn.TurnLog) (declaredAction, turn.CounterSummary, *turn.ChallengeSummary, bool, error) {
	counterDecl := declaredAction{
		Command: counterCmd,
		Cost:    actions.ActionCost(counterCmd.Action),
	}
	counterSummary := turn.CounterSummary{Action: counterCmd}

	logs.Publicf("%s counters with %s", playerName(s.State, counterDecl.Command.ActorIndex), counterDecl.Command.Action)
	if err := counterDecl.payCost(s.State, logs); err != nil {
		return declaredAction{}, turn.CounterSummary{}, nil, false, fmt.Errorf("pay counter action cost: %w", err)
	}
	// get the action from the registry
	_, ok := actions.ActionPresentationByID(counterCmd.Action)
	if !ok {
		return counterDecl, counterSummary, nil, true, nil
	}
	role := actionRole(counterCmd.Action)
	if role == "" {
		return counterDecl, counterSummary, nil, true, nil
	}

	logs.Publicf("challenge window opened for %s (%s)", counterCmd.Action, role)
	challengeRes, err := s.runChallengeWindow(ctx, input, clock, timeout, ChallengeCounter, counterCmd, role, logs)
	counterSummary.Challenge = &challengeRes
	if err != nil {
		return counterDecl, counterSummary, &challengeRes, false, fmt.Errorf("run counter challenge: %w", err)
	}
	if !challengeRes.ActionAllowed {
		counterDecl.refundCost(s.State, logs)
		counterSummary.Canceled = true
		return counterDecl, counterSummary, &challengeRes, false, nil
	}

	return counterDecl, counterSummary, &challengeRes, true, nil
}

func actionRole(id types.ActionID) types.Role {
	if def, ok := actions.MainAction(id); ok {
		return def.Role
	}
	if def, ok := actions.CounterAction(id); ok {
		return def.Role
	}
	return ""
}

func (s *Service) applySurvivingActions(ctx context.Context, input InputProvider, main declaredAction, mainAllowed, mainCanceled bool, deferred []deferredCounter, counterSummaries []turn.CounterSummary, logs *turn.TurnLog) error {
	if !mainAllowed || mainCanceled {
		return nil
	}

	if err := s.applyActionWithSteps(ctx, input, main.Command, logs); err != nil {
		return err
	}
	for _, counter := range deferred {
		if err := s.applyActionWithSteps(ctx, input, counter.declaration.Command, logs); err != nil {
			return err
		}
		counterSummaries[counter.summaryIndex].Applied = true
	}
	return nil
}

func appendActionDeclaration(gs *state.GameState, logs *turn.TurnLog, action *types.PlayerAction) {
	if logs == nil || action == nil {
		return
	}
	switch payload := action.Payload.(type) {
	case types.TargetPayload:
		switch action.Action {
		case types.Coup:
			logs.Publicf("%s coups %s", playerName(gs, action.ActorIndex), playerName(gs, payload.TargetIndex))
		case types.Investigate:
			logs.Publicf("%s investigates %s", playerName(gs, action.ActorIndex), playerName(gs, payload.TargetIndex))
		case types.Assassinate:
			logs.Publicf("%s assassinates %s", playerName(gs, action.ActorIndex), playerName(gs, payload.TargetIndex))
		case types.Steal:
			logs.Publicf("%s steals 2 from %s", playerName(gs, action.ActorIndex), playerName(gs, payload.TargetIndex))
		default:
			logs.Publicf("%s declares %s targeting %s", playerName(gs, action.ActorIndex), action.Action, playerName(gs, payload.TargetIndex))
		}
	case types.AccusePayload:
		logs.Publicf("%s accuses %s of being %s", playerName(gs, action.ActorIndex), playerName(gs, payload.TargetIndex), payload.Guess)
	default:
		logs.Publicf("%s declares %s", playerName(gs, action.ActorIndex), action.Action)
	}
}

func appendHandSummaries(logs *turn.TurnLog, gs *state.GameState) {
	if logs == nil || gs == nil {
		return
	}
	for index, player := range gs.players {
		if len(player.Hand) == 0 {
			logs.Privatef(index, "Your hand: (none)")
			continue
		}
		roles := make([]string, 0, len(player.Hand))
		for _, card := range player.Hand {
			roles = append(roles, string(card.Role))
		}
		logs.Privatef(index, "Your hand: %s", strings.Join(roles, ", "))
	}
}

func playerName(gs *state.GameState, index int) string {
	if gs == nil || index < 0 || index >= len(gs.players) {
		return fmt.Sprintf("player-%d", index)
	}
	return gs.players[index].Name
}
