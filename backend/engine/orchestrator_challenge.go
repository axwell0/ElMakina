package engine

import (
	"context"
	"fmt"
	"time"

	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

// runChallengeWindow orchestrates the "Liar!" phase of a turn.
//
// When a player declares an action with a role claim (like "I'm the Thief, I'm stealing from you"),
// the game enters a challenge window. Every other player gets a chance to call them out.
// This function manages that window: collecting responses, handling timeouts, and resolving
// the challenge if someone pulls the trigger.
//
// THE CHALLENGE FLOW:
//  1. OPEN WINDOW: We broadcast to all eligible players "Someone claimed [Role]. Challenge?"
//  2. COLLECT RESPONSES: Players either Pass (let it happen) or Challenge (call "Liar!")
//  3. FIRST CHALLENGE WINS: If multiple players challenge, the first one in the channel wins
//  4. RESOLVE: We call ResolveChallenge to determine who was bluffing and who loses a card
//  5. TIMEOUT: If everyone passes or time runs out, the action proceeds unchallenged
//
// WHY THE COMPLEXITY?
// Challenges are the core social mechanic of Coup. Players need time to read the situation,
// and multiple people might want to challenge. We need to:
//   - Handle simultaneous responses (race conditions)
//   - Allow timeouts (AFK players shouldn't block the game forever)
//   - Track who has passed (if everyone passes, we can close early)
//   - Support context cancellation (player disconnects, game ends)
//
// The function returns a ChallengeResult that tells the orchestrator whether the action
// is allowed to proceed or was canceled due to a successful challenge.
func (g *Game) runChallengeWindow(ctx context.Context, input InputProvider, clock Clock, timeout time.Duration, kind ChallengeKind, action *models.PlayerAction, role models.Role) (ChallengeResult, error) {
	window := ChallengeWindow{
		Kind:               kind,
		Action:             action,
		ActorIndex:         action.SourceIndex,
		ClaimedRole:        role,
		AllowedChallengers: allowedPlayers(&g.CurrentState, action.SourceIndex),
	}
	windowCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, err := input.ChallengeResponses(windowCtx, window)
	if err != nil {
		return ChallengeResult{}, err
	}
	timeoutCh := timerChan(clock, timeout)
	errCh := inputErrors(input)
	passed := make(map[int]bool, len(window.AllowedChallengers))

	for {
		select {
		case <-ctx.Done():
			return ChallengeResult{}, ctx.Err()
		case err := <-errCh:
			if err != nil {
				return ChallengeResult{}, err
			}
		case <-timeoutCh:
			return ChallengeResult{
				ActorIndex:      action.SourceIndex,
				ChallengerIndex: NoPlayer,
				ClaimedRole:     role,
				ActionAllowed:   true,
				Outcome:         OutcomeNoChallenge,
			}, nil
		case decision, ok := <-ch:
			if !ok {
				return ChallengeResult{
					ActorIndex:      action.SourceIndex,
					ChallengerIndex: NoPlayer,
					ClaimedRole:     role,
					ActionAllowed:   true,
					Outcome:         OutcomeNoChallenge,
				}, nil
			}
			if !isAllowedPlayer(decision.ChallengerIndex, window.AllowedChallengers) {
				return ChallengeResult{}, fmt.Errorf("challenger %d not allowed", decision.ChallengerIndex)
			}
			if decision.Pass {
				passed[decision.ChallengerIndex] = true
				passLog := fmt.Sprintf("%s passes challenge", g.CurrentState.Players[decision.ChallengerIndex].Name)
				if len(passed) == len(window.AllowedChallengers) {
					return ChallengeResult{
						ActorIndex:      action.SourceIndex,
						ChallengerIndex: NoPlayer,
						ClaimedRole:     role,
						ActionAllowed:   true,
						Outcome:         OutcomeNoChallenge,
						Logs:            []string{passLog},
					}, nil
				}
				continue
			}
			if err := g.populateChallengeSelections(ctx, input, action, role, &decision); err != nil {
				return ChallengeResult{}, err
			}
			return g.ResolveChallenge(action.SourceIndex, role, decision)
		}
	}
}

// populateChallengeSelections gathers the card indices needed to resolve a challenge.
//
// When a challenge happens, we need to know:
//   - If the actor HAS the role: Which card proves it? (ActorProvingCardIndex)
//   - If the actor DOESN'T have the role: Which card do they discard? (ActorDiscardIndex)
//   - The challenger always discards a card: Which one? (ChallengerDiscardIndex)
//
// This function figures out what information we need and requests it from the players
// via the InputProvider. It's essentially the "setup" before ResolveChallenge does
// the actual state mutation.
//
// The logic branch here is important:
//   - Actor has role → We auto-select their proving card (first matching role)
//     but ask the challenger which card to discard
//   - Actor bluffing → We ask the actor which card to discard (they lose one)
//
// This separation exists because when the actor has the role, we KNOW which card
// they need to reveal (the one matching the claimed role). But when they're bluffing,
// they get to choose which influence to lose.
func (g *Game) populateChallengeSelections(ctx context.Context, input InputProvider, action *models.PlayerAction, role models.Role, decision *ChallengeDecision) error {
	if g == nil || input == nil || action == nil || decision == nil {
		return fmt.Errorf("invalid challenge selection context")
	}
	gs := &g.CurrentState
	actor := &gs.Players[action.SourceIndex]
	challenger := &gs.Players[decision.ChallengerIndex]

	if g.playerHasRole(actor, role) {
		index := firstRoleIndex(actor.Hand, role)
		if index < 0 {
			return fmt.Errorf("actor does not have claimed role")
		}
		decision.ActorProvingCardIndex = index

		index, err := requestCardIndex(ctx, input, action.ID, decision.ChallengerIndex, len(challenger.Hand), state.ContextDiscard, cardOptions(challenger.Hand))
		if err != nil {
			return err
		}
		decision.ChallengerDiscardIndex = index
		return nil
	}

	index, err := requestCardIndex(ctx, input, action.ID, action.SourceIndex, len(actor.Hand), state.ContextDiscard, cardOptions(actor.Hand))
	if err != nil {
		return err
	}
	decision.ActorDiscardIndex = index
	return nil
}

// requestCardIndex asks a player to select a specific card from their hand.
//
// This is a helper for the challenge resolution flow. When someone loses a challenge,
// they need to discard an influence card. This function sends a StepRequest to the
// player asking them to pick which card to lose.
//
// The StepKind is always StepCardSelect because we're asking for card indices.
// The Context (discard vs exchange) tells the UI what label to show.
func requestCardIndex(ctx context.Context, input InputProvider, actionID models.ActionID, actor int, max int, context state.StepContext, options []string) (int, error) {
	resp, err := input.RequestStep(ctx, state.StepRequest{
		ActionID: actionID,
		Kind:     state.StepCardSelect,
		Actor:    actor,
		Count:    1,
		Context:  context,
		Options:  options,
	})
	if err != nil {
		return 0, err
	}
	indices, ok := resp.([]int)
	if !ok {
		return 0, fmt.Errorf("invalid response type: expected []int, got %T", resp)
	}
	if len(indices) != 1 {
		return 0, fmt.Errorf("invalid response: expected 1 card index")
	}
	index := indices[0]
	if index < 0 || index >= max {
		return 0, fmt.Errorf("invalid card index")
	}
	return index, nil
}

// cardOptions converts a player's hand into display labels for the UI.
//
// When asking a player to select a card, we send these labels so the frontend
// can show meaningful buttons like "Colonel" or "Thief" instead of just "Card 1".
//
// The fallback to "Card N" is defensive programming—in theory all cards should
// have roles, but if something goes wrong, we still show something usable.
func cardOptions(hand []state.Card) []string {
	options := make([]string, 0, len(hand))
	for i, card := range hand {
		label := card.Role
		if label == "" {
			label = models.Role(fmt.Sprintf("Card %d", i+1))
		}
		options = append(options, string(label))
	}
	return options
}

func firstRoleIndex(hand []state.Card, role models.Role) int {
	for i, card := range hand {
		if card.Role == role {
			return i
		}
	}
	return -1
}
