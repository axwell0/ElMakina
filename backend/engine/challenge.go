package engine

import (
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"fmt"
)

// ResolveChallenge is the "Courtroom" where challenges are judged.
//
// When someone calls "Liar!" on a role claim, this function determines who was right
// and applies the consequences. It's the authoritative source for challenge logic.
//
// THE TWO OUTCOMES:
//
// 1. ACTOR HAS THE ROLE (Challenger Loses):
//    - Actor reveals their matching card to prove they weren't bluffing
//    - Actor swaps that card with a new one from the deck (to prevent card counting)
//    - Challenger must discard one of their influence cards
//    - Action proceeds as declared
//
// 2. ACTOR WAS BLUFFING (Actor Loses):
//    - Actor doesn't have the claimed role
//    - Actor must discard one of their influence cards
//    - Action is CANCELED (coins refunded if applicable)
//    - Challenger faces no penalty
//
// WHY THE CARD SWAP?
// When the actor proves they have the role, we don't just show the card—we swap it.
// This prevents card counting: if Alice reveals she has the Thief, Bob shouldn't be
// able to track that card forever. The swap keeps hidden information hidden.
//
// The function returns a ChallengeResult that includes:
//   - Success: Was the challenge successful (did the actor bluff)?
//   - ActionAllowed: Can the action proceed after resolution?
//   - Outcome: Detailed enum describing what happened
//   - Logs: Human-readable description of events for the game log
func (g *Game) ResolveChallenge(actorIndex int, claimedRole models.Role, decision ChallengeDecision) (ChallengeResult, error) {

	// Initialize result struct with challenge context
	res := ChallengeResult{
		ActorIndex:      actorIndex,
		ChallengerIndex: decision.ChallengerIndex,
		ClaimedRole:     claimedRole,
		LostPlayerIndex: -1,
	}
	if g == nil {
		return res, fmt.Errorf("game is nil")
	}

	// No challenge - action proceeds
	if decision.ChallengerIndex == NoPlayer {
		res.ActionAllowed = true
		res.Outcome = OutcomeNoChallenge
		return res, nil
	}

	// Validate player indices
	if err := g.validatePlayerIndex(actorIndex, "actor"); err != nil {
		return res, err
	}
	if err := g.validatePlayerIndex(decision.ChallengerIndex, "challenger"); err != nil {
		return res, err
	}
	if decision.ChallengerIndex == actorIndex {
		return res, fmt.Errorf("challenger cannot be the actor")
	}

	gs := &g.CurrentState
	actor := &gs.Players[actorIndex]
	challenger := &gs.Players[decision.ChallengerIndex]
	res.Logs = append(res.Logs, fmt.Sprintf("%s challenges %s claiming %s", challenger.Name, actor.Name, claimedRole))

	// Actor has the role - challenger loses a card
	if g.playerHasRole(actor, claimedRole) {
		if decision.ActorProvingCardIndex < 0 || decision.ActorProvingCardIndex >= len(actor.Hand) {
			return res, fmt.Errorf("invalid proving card index")
		}
		provingCard := actor.Hand[decision.ActorProvingCardIndex]
		if provingCard.Role != claimedRole {
			return res, fmt.Errorf("selected card does not match claimed role")
		}

		if err := gs.SwapCard(actorIndex, decision.ActorProvingCardIndex); err != nil {
			return res, err
		}

		if decision.ChallengerDiscardIndex < 0 || decision.ChallengerDiscardIndex >= len(challenger.Hand) {
			res.ActionAllowed = true
			return res, fmt.Errorf("invalid challenger discard index")
		}
		challengerLost, err := gs.DiscardFromHand(decision.ChallengerIndex, decision.ChallengerDiscardIndex)
		if err != nil {
			res.ActionAllowed = true
			return res, err
		}

		res.Logs = append(res.Logs, fmt.Sprintf("%s discards %s", challenger.Name, challengerLost.Role))
		if len(challenger.Hand) == 0 {
			res.Logs = append(res.Logs, fmt.Sprintf("%s is eliminated", challenger.Name))
		}
		res.Success = false
		res.ActionAllowed = true
		res.Outcome = OutcomeActorProved
		res.LostCard = &challengerLost
		res.LostPlayerIndex = decision.ChallengerIndex
		res.Logs = append(res.Logs, fmt.Sprintf("%s reveals %s and swaps", actor.Name, claimedRole))
		return res, nil
	}

	// Actor was bluffing - actor loses a card, action canceled
	lost, err := gs.DiscardFromHand(actorIndex, decision.ActorDiscardIndex)
	if err != nil {
		return res, err
	}
	res.Logs = append(res.Logs, fmt.Sprintf("%s discards %s", actor.Name, lost.Role))
	if len(actor.Hand) == 0 {
		res.Logs = append(res.Logs, fmt.Sprintf("%s is eliminated", actor.Name))
	}
	res.Success = true
	res.ActionAllowed = false
	res.Outcome = OutcomeActorBluffed
	res.LostCard = &lost
	res.LostPlayerIndex = actorIndex
	res.Logs = append(res.Logs, fmt.Sprintf("%s was bluffing; the action is canceled", actor.Name))
	return res, nil
}

// playerHasRole is a simple helper that answers "does this player actually have [Role]?"
//
// Used during challenge resolution to determine if the actor was bluffing.
// We iterate through the player's hand looking for a card with the claimed role.
//
// Note: This only checks the current hand. Cards that have been discarded
// are no longer in the hand slice, so they don't count.
func (g *Game) playerHasRole(player *state.Player, role models.Role) bool {
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
