package turn

import (
	"context"
	"fmt"
	"slices"
	"time"

	coreactions "github.com/axwell0/elmakina/engine/core/actions"
	corestate "github.com/axwell0/elmakina/engine/core/state"
	coretypes "github.com/axwell0/elmakina/engine/core/types"
	turnmodel "github.com/axwell0/elmakina/engine/turn"
)

// runChallengeWindow asks all eligible players whether they want to challenge the actor's
// claimed role. The window stays open until one of three things happens:
//  1. All allowed challengers explicitly pass → action proceeds unchallenged
//  2. One player commits a challenge → resolution happens immediately (proof or bluff)
//  3. The timeout expires (or ctx is canceled) → same as everyone passing
//
// A "claimed role" means the actor declared an action that requires a specific role card
// (e.g. Steal claims the Thief role). Any other player can challenge: "I don't believe
// you have that card." If correct, the actor is bluffing and loses a card. If wrong,
// the challenger pays the penalty instead.
//
//nolint:gocognit,cyclop // The challenge window is a compact event loop; splitting it would scatter timeout/channel semantics.
func (s *Service) runChallengeWindow(ctx context.Context, input InputProvider, clock Clock, timeout time.Duration, kind ChallengeKind, action *coretypes.PlayerAction, role coretypes.Role, logs *turnmodel.TurnLog) (turnmodel.ChallengeSummary, error) {
	window := ChallengeWindow{
		Kind:        kind,
		Action:      action,
		ActorIndex:  action.ActorIndex,
		ClaimedRole: role,
	}
	for i, player := range s.State.players {
		if i != action.ActorIndex && len(player.Hand) > 0 {
			window.AllowedChallengers = append(window.AllowedChallengers, i)
		}
	}
	windowCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, err := input.ChallengeResponses(windowCtx, window)
	if err != nil {
		return turnmodel.ChallengeSummary{}, fmt.Errorf("open challenge responses: %w", err)
	}
	var timeoutCh <-chan time.Time
	if timeout > 0 {
		timeoutCh = clock.After(timeout)
	}
	var errCh <-chan error
	type errorReporter interface {
		Errors() <-chan error
	}
	if reporter, ok := input.(errorReporter); ok {
		errCh = reporter.Errors()
	}
	passed := make(map[int]bool, len(window.AllowedChallengers))

	for {
		select {
		case <-ctx.Done():
			return turnmodel.ChallengeSummary{}, fmt.Errorf("challenge window canceled: %w", ctx.Err())
		case err := <-errCh:
			if err != nil {
				return turnmodel.ChallengeSummary{}, fmt.Errorf("challenge response stream: %w", err)
			}
		case <-timeoutCh:
			// Window expired with no committed challenge.
			return turnmodel.ChallengeSummary{ActionID: action.Action, ActorIndex: action.ActorIndex, ChallengerIndex: turnmodel.NoPlayer, ClaimedRole: role, ActionAllowed: true, Outcome: turnmodel.OutcomeNoChallenge}, nil
		case decision, ok := <-ch:
			if !ok {
				// Provider closed channel, treat as no-challenge outcome.
				return turnmodel.ChallengeSummary{ActionID: action.Action, ActorIndex: action.ActorIndex, ChallengerIndex: turnmodel.NoPlayer, ClaimedRole: role, ActionAllowed: true, Outcome: turnmodel.OutcomeNoChallenge}, nil
			}
			if !slices.Contains(window.AllowedChallengers, decision.ChallengerIndex) {
				return turnmodel.ChallengeSummary{}, fmt.Errorf("%w: %d", errChallengerNotAllowed, decision.ChallengerIndex)
			}
			if decision.Pass {
				// Track explicit passes so we can close window once all responders passed.
				passed[decision.ChallengerIndex] = true
				logs.Publicf("%s passes challenge", playerName(s.State, decision.ChallengerIndex))
				if len(passed) == len(window.AllowedChallengers) {
					return turnmodel.ChallengeSummary{
						ActionID:        action.Action,
						ActorIndex:      action.ActorIndex,
						ChallengerIndex: turnmodel.NoPlayer,
						ClaimedRole:     role,
						ActionAllowed:   true,
						Outcome:         turnmodel.OutcomeNoChallenge,
					}, nil
				}
				continue
			}
			// A committed challenge needs discard/proof card selections first.
			if err := s.populateChallengeSelections(ctx, input, action, role, &decision); err != nil {
				return turnmodel.ChallengeSummary{}, fmt.Errorf("populate challenge selections: %w", err)
			}
			return s.ResolveChallenge(logs, action.Action, action.ActorIndex, role, decision)
		}
	}
}

// populateChallengeSelections asks the relevant player to choose the card
// they will reveal or lose if the challenge is committed.
func (s *Service) populateChallengeSelections(ctx context.Context, input InputProvider, action *coretypes.PlayerAction, role coretypes.Role, decision *ChallengeDecision) error {
	if s == nil || s.State == nil || input == nil || action == nil || decision == nil {
		return errChallengeSelection
	}
	actor := &s.State.players[action.ActorIndex]
	challenger := &s.State.players[decision.ChallengerIndex]

	if playerHasRole(actor, role) {
		// PROOF BRANCH: actor has the claimed role card.
		// The actor reveals it (ActorProvingCardIndex), then draws a replacement so their
		// hand stays secret. The challenger must choose which of their own cards to lose.
		// SwapAndRevealCard (called in ResolveChallenge) handles the reveal+replace.
		index := firstRoleIndex(actor.Hand, role)
		if index < 0 {
			return errInvalidChallengeRole
		}
		decision.ActorProvingCardIndex = index

		// Ask the challenger which of their own cards to sacrifice.
		index, err := requestCardIndex(ctx, input, action.Action, decision.ChallengerIndex, cardOptions(challenger.Hand))
		if err != nil {
			return fmt.Errorf("request challenger discard: %w", err)
		}
		decision.ChallengerDiscardIndex = index
		return nil
	}

	// BLUFF BRANCH: actor does not have the claimed role card.
	// The actor must choose which of their own cards to discard as the bluff penalty.
	// ActorDiscardIndex is set (ActorProvingCardIndex stays zero, unused).
	index, err := requestCardIndex(ctx, input, action.Action, action.ActorIndex, cardOptions(actor.Hand))
	if err != nil {
		return fmt.Errorf("request actor discard: %w", err)
	}
	decision.ActorDiscardIndex = index
	return nil
}

// ResolveChallenge applies the consequences of a challenge after all required
// selections have been collected.
//
//nolint:gocognit,nestif,cyclop // The proof/bluff branches mirror the game rules and share one summary object.
func (s *Service) ResolveChallenge(logs *turnmodel.TurnLog, actionID coretypes.ActionID, actorIndex int, claimedRole coretypes.Role, decision ChallengeDecision) (turnmodel.ChallengeSummary, error) {
	res := turnmodel.ChallengeSummary{
		ActionID:        actionID,
		ActorIndex:      actorIndex,
		ChallengerIndex: decision.ChallengerIndex,
		ClaimedRole:     claimedRole,
	}
	if s == nil || s.State == nil {
		return res, errServiceStateNil
	}
	if decision.ChallengerIndex == turnmodel.NoPlayer {
		res.ActionAllowed = true
		res.Outcome = turnmodel.OutcomeNoChallenge
		return res, nil
	}

	if actorIndex < 0 || actorIndex >= len(s.State.players) {
		return res, fmt.Errorf("%w: actor %d", errInvalidCardIndex, actorIndex)
	}
	if decision.ChallengerIndex < 0 || decision.ChallengerIndex >= len(s.State.players) {
		return res, fmt.Errorf("%w: challenger %d", errInvalidCardIndex, decision.ChallengerIndex)
	}
	if decision.ChallengerIndex == actorIndex {
		return res, errInvalidChallengeTarget
	}

	actor := &s.State.players[actorIndex]
	challenger := &s.State.players[decision.ChallengerIndex]
	logs.Publicf("%s challenges %s claiming %s", playerName(s.State, decision.ChallengerIndex), playerName(s.State, actorIndex), claimedRole)
	res.Logs = append(res.Logs, fmt.Sprintf("%s challenges %s claiming %s", playerName(s.State, decision.ChallengerIndex), playerName(s.State, actorIndex), claimedRole))

	if playerHasRole(actor, claimedRole) {
		// Successful proof branch: claim stands and challenger pays discard penalty.
		if decision.ActorProvingCardIndex < 0 || decision.ActorProvingCardIndex >= len(actor.Hand) {
			return res, fmt.Errorf("%w: proving card", errInvalidCardIndex)
		}
		provingCard := actor.Hand[decision.ActorProvingCardIndex]
		if provingCard.Role != claimedRole {
			return res, errInvalidChallengeRole
		}
		if decision.ChallengerDiscardIndex < 0 || decision.ChallengerDiscardIndex >= len(challenger.Hand) {
			res.ActionAllowed = true
			return res, fmt.Errorf("%w: challenger discard", errInvalidCardIndex)
		}
		_, err := s.State.SwapAndRevealCard(actorIndex, decision.ActorProvingCardIndex)
		if err != nil {
			return res, fmt.Errorf("swap proving card: %w", err)
		}
		challengerLostDetail, err := s.State.DiscardFromHand(decision.ChallengerIndex, decision.ChallengerDiscardIndex)
		if err != nil {
			res.ActionAllowed = true
			return res, fmt.Errorf("discard challenger card: %w", err)
		}
		challengerLost := challengerLostDetail.Card
		logs.Publicf("%s reveals %s and swaps", playerName(s.State, actorIndex), claimedRole)
		logs.Publicf("%s finished exchange", playerName(s.State, actorIndex))
		logs.Privatef(actorIndex, "%s draws %d cards to exchange", playerName(s.State, actorIndex), 1)
		logs.Publicf("%s discards %s", playerName(s.State, decision.ChallengerIndex), challengerLost.Role)
		res.Logs = append(res.Logs,
			fmt.Sprintf("%s reveals %s and swaps", playerName(s.State, actorIndex), claimedRole),
			fmt.Sprintf("%s discards %s", playerName(s.State, decision.ChallengerIndex), challengerLost.Role),
		)
		if len(challenger.Hand) == 0 {
			logs.Publicf("%s is eliminated", playerName(s.State, decision.ChallengerIndex))
		}
		res.ActionAllowed = true
		res.Outcome = turnmodel.OutcomeActorProved
		res.LostCardRole = challengerLost.Role
		lostPlayerIndex := decision.ChallengerIndex
		res.LostPlayerIndex = &lostPlayerIndex
		return res, nil
	}

	// Failed proof branch: action is canceled and actor discards.
	lostDetail, err := s.State.DiscardFromHand(actorIndex, decision.ActorDiscardIndex)
	if err != nil {
		return res, fmt.Errorf("discard bluffing actor card: %w", err)
	}
	lost := lostDetail.Card
	logs.Publicf("%s was bluffing; the action is canceled", playerName(s.State, actorIndex))
	logs.Publicf("%s discards %s", playerName(s.State, actorIndex), lost.Role)
	res.Logs = append(res.Logs,
		fmt.Sprintf("%s was bluffing; the action is canceled", playerName(s.State, actorIndex)),
		fmt.Sprintf("%s discards %s", playerName(s.State, actorIndex), lost.Role),
	)
	if len(actor.Hand) == 0 {
		logs.Publicf("%s is eliminated", playerName(s.State, actorIndex))
	}
	res.ActionAllowed = false
	res.Outcome = turnmodel.OutcomeActorBluffed
	res.LostCardRole = lost.Role
	lostPlayerIndex := actorIndex
	res.LostPlayerIndex = &lostPlayerIndex
	return res, nil
}

// requestCardIndex asks the input provider for a single card selection and
// validates the returned index before the engine uses it.
func requestCardIndex(ctx context.Context, input InputProvider, actionID coretypes.ActionID, actor int, options []coreactions.CardOption) (int, error) {
	resp, err := input.RequestCardSelection(ctx, coreactions.StepRequest{
		Action:      actionID,
		Kind:        coreactions.StepCardSelection,
		ActorIndex:  actor,
		Title:       "Choose a card to lose",
		Description: "Select the card that should be discarded.",
		CardSelection: &coreactions.CardSelectionStep{
			Count: 1,
			Cards: options,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("request card selection: %w", err)
	}
	if len(resp.SelectedCardIndices) != 1 {
		return 0, fmt.Errorf("%w: expected 1 card index", errInvalidResponse)
	}
	index := resp.SelectedCardIndices[0]
	// Validate index against the options shown to the player.
	if index < 0 || index >= len(options) {
		return 0, errInvalidCardIndex
	}
	return index, nil
}

// cardOptions turns a hand into a list of labels that the UI can display.
func cardOptions(hand []corestate.Card) []coreactions.CardOption {
	options := make([]coreactions.CardOption, 0, len(hand))
	for i, card := range hand {
		label := card.Role
		if label == "" {
			label = coretypes.Role(fmt.Sprintf("Card %d", i+1))
		}
		options = append(options, coreactions.CardOption{
			Index: i,
			Label: string(label),
		})
	}
	return options
}

// firstRoleIndex finds the first card in a hand with the given role.
func firstRoleIndex(hand []corestate.Card, role coretypes.Role) int {
	for i, card := range hand {
		if card.Role == role {
			return i
		}
	}
	return -1
}

// playerHasRole is the fast check used before a challenge decides whether the
// actor proves the claim or loses a card.
func playerHasRole(player *corestate.Player, role coretypes.Role) bool {
	if player == nil {
		return false
	}
	for _, card := range player.Hand {
		if card.Role == role {
			return true
		}
	}
	return false
}
