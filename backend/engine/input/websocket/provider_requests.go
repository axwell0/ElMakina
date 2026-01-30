package websocket

import (
	"context"
	"fmt"

	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

// RequestAction asks the current player "What do you want to do?" and waits for an answer.
//
// This is the entry point for the main phase of a turn. The engine calls this when it's
// a player's turn to act. We send a message to that player's WebSocket and block until
// they respond with their chosen action.
//
// THE REQUEST/RESPONSE PATTERN:
//   1. Generate a unique requestID for this specific ask
//  2. Register a "waiter" (a channel that will receive the response)
//  3. Send the request to the player's WebSocket
//  4. Block on the waiter channel until response arrives OR context is cancelled
//  5. Clean up the waiter registration
//
// WHY THIS COMPLEXITY?
// WebSocket connections are long-lived and multiplexed. Multiple requests might be
// in flight (though in practice, the game is sequential). The requestID lets us
// match responses to their original requests, even if they arrive out of order
// (though they shouldn't in a well-behaved client).
//
// TIMEOUT HANDLING:
// The context passed in has a deadline. If the player doesn't respond in time,
// ctx.Done() fires and we return an error. The orchestrator will handle this
// as a forfeit or pass, depending on configuration.
func (p *Provider) RequestAction(ctx context.Context, actor int, _ *state.GameState) (*models.PlayerAction, error) {
	reqID := p.nextRequestID()
	waiter := p.registerWaiter(reqID)
	defer p.removeWaiter(reqID)

	if err := p.sendTo(actor, Envelope{
		Type:      msgRequestAction,
		RequestID: reqID,
		Payload:   mustMarshal(requestActionPayload{ActorIndex: actor}),
	}); err != nil {
		return nil, err
	}

	errCh := p.Errors()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			if err != nil {
				return nil, err
			}
		case resp := <-waiter.ch:
			if resp.sender != actor {
				return nil, fmt.Errorf("action response from unexpected player %d", resp.sender)
			}
			if resp.env.Type != msgAction {
				return nil, fmt.Errorf("unexpected response type %q", resp.env.Type)
			}
			return parseActionPayload(resp.env.Payload)
		}
	}
}

// ChallengeResponses opens the "challenge window" for all eligible players.
//
// When a player declares an action with a role claim (like "I'm the Thief"), everyone
// else gets a chance to call "Liar!". This function broadcasts the challenge opportunity
// and returns a channel that streams responses as they arrive.
//
// WHY A STREAMING CHANNEL?
// Unlike RequestAction (one player, one response), challenges involve multiple players
// responding at roughly the same time. We need to collect all "Pass" responses AND
// be ready to handle a "Challenge" at any moment. The channel lets the orchestrator
// decide when to stop listening (usually on first challenge or when everyone passes).
//
// THE GOROUTINE LIFECYCLE:
// We spawn a goroutine that:
//   - Listens for responses on the internal waiter channel
//   - Validates each response (is it from an allowed player?)
//   - Parses the challenge decision and sends it on the output channel
//   - Exits when the context is cancelled (timeout or resolution)
//
// The goroutine cleans up its waiter registration when done to prevent memory leaks.
func (p *Provider) ChallengeResponses(ctx context.Context, window engine.ChallengeWindow) (<-chan engine.ChallengeDecision, error) {
	reqID := p.nextRequestID()
	waiter := p.registerWaiter(reqID)

	for _, player := range window.AllowedChallengers {
		if err := p.sendTo(player, Envelope{
			Type:      msgChallengeWindow,
			RequestID: reqID,
			Payload: mustMarshal(challengeWindowPayload{
				ActorIndex:  window.ActorIndex,
				ActionID:    window.Action.ID,
				ClaimedRole: string(window.ClaimedRole),
				Kind:        string(window.Kind),
				TargetIndex: targetIndexFromAction(window.Action),
			}),
		}); err != nil {
			p.removeWaiter(reqID)
			return nil, err
		}
	}

	out := make(chan engine.ChallengeDecision)
	go func() {
		defer close(out)
		defer p.removeWaiter(reqID)

		errCh := p.Errors()
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errCh:
				if err != nil {
					return
				}
			case resp := <-waiter.ch:
				if resp.env.Type != msgChallenge {
					p.notifyError(fmt.Errorf("unexpected challenge response type %q", resp.env.Type))
					return
				}
				decision, err := parseChallengePayload(resp.env.Payload)
				if err != nil {
					p.notifyError(err)
					return
				}
				if !isAllowedPlayer(decision.ChallengerIndex, window.AllowedChallengers) {
					p.notifyError(fmt.Errorf("challenger %d not allowed", decision.ChallengerIndex))
					return
				}
				if decision.ChallengerIndex != resp.sender {
					p.notifyError(fmt.Errorf("challenge sender %d does not match payload %d", resp.sender, decision.ChallengerIndex))
					return
				}
				out <- decision
			}
		}
	}()

	return out, nil
}

// CounterResponses opens the "counter window" for players who can block an action.
//
// After a role action survives challenges, targets get a chance to block it.
// For example, when someone tries to Steal, the victim can claim "I'm the Thief too,
// I block your steal." This function notifies eligible players and collects responses.
//
// HOW IT DIFFERS FROM CHALLENGES:
//   - Counters are defensive (blocking an action against you)
//   - Players can "Pass" to let the action through
//   - Multiple players might be able to counter (e.g., TaxBusinessWoman)
//   - The first valid counter usually wins (unlike challenges where first wins)
//
// PASS HANDLING:
// When a player explicitly passes, we note their player index. This helps the
// orchestrator track who has responded vs who is still thinking.
func (p *Provider) CounterResponses(ctx context.Context, window engine.CounterWindow) (<-chan engine.CounterDecision, error) {
	reqID := p.nextRequestID()
	waiter := p.registerWaiter(reqID)

	for _, player := range window.AllowedPlayers {
		if err := p.sendTo(player, Envelope{
			Type:      msgCounterWindow,
			RequestID: reqID,
			Payload: mustMarshal(counterWindowPayload{
				ActorIndex:    window.MainAction.SourceIndex,
				ActionID:      window.MainAction.ID,
				AllowedAction: window.AllowedActions,
				TargetIndex:   targetIndexFromAction(window.MainAction),
			}),
		}); err != nil {
			p.removeWaiter(reqID)
			return nil, err
		}
	}

	out := make(chan engine.CounterDecision)
	go func() {
		defer close(out)
		defer p.removeWaiter(reqID)

		errCh := p.Errors()
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errCh:
				if err != nil {
					return
				}
			case resp := <-waiter.ch:
				if resp.env.Type != msgCounter {
					p.notifyError(fmt.Errorf("unexpected counter response type %q", resp.env.Type))
					return
				}
				decision, err := parseCounterPayload(resp.env.Payload)
				if err != nil {
					p.notifyError(err)
					return
				}
				if decision.Pass {
					decision.PlayerIndex = resp.sender
					out <- decision
					continue
				}
				if decision.Command != nil && decision.Command.SourceIndex != resp.sender {
					p.notifyError(fmt.Errorf("counter sender %d does not match payload %d", resp.sender, decision.Command.SourceIndex))
					return
				}
				out <- decision
			}
		}
	}()

	return out, nil
}

// RequestStep asks for mid-action input (like "which cards do you want to keep?").
//
// Some actions aren't atomic—they need player input partway through. The Exchange
// action is the classic example: draw 2 cards, THEN decide which to keep. This
// function handles that "then" part.
//
// WHEN IS THIS USED?
//   - Exchange: Selecting which cards to return to deck
//   - Coup/Assassinate: Target selecting which card to discard
//   - Challenge resolution: Loser selecting which card to lose
//
// The StepRequest contains all context needed (action ID, player index, how many
// cards to select, etc.). We send this to the player and block until they respond.
//
// WHY "any" RETURN TYPE?
// Different steps return different data types. Card selection returns []int,
// but future step types might return other things. Using "any" lets us extend
// the system without changing the interface.
func (p *Provider) RequestStep(ctx context.Context, req state.StepRequest) (any, error) {
	reqID := p.nextRequestID()
	waiter := p.registerWaiter(reqID)
	defer p.removeWaiter(reqID)

	if err := p.sendTo(req.Actor, Envelope{
		Type:      msgRequestStep,
		RequestID: reqID,
		Payload:   mustMarshal(requestStepPayload{Step: req}),
	}); err != nil {
		return nil, err
	}

	errCh := p.Errors()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			if err != nil {
				return nil, err
			}
		case resp := <-waiter.ch:
			if resp.sender != req.Actor {
				return nil, fmt.Errorf("step response from unexpected player %d", resp.sender)
			}
			if resp.env.Type != msgStepResult {
				return nil, fmt.Errorf("unexpected step response type %q", resp.env.Type)
			}
			return parseStepPayload(req, resp.env.Payload)
		}
	}
}
