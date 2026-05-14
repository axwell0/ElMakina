// Package actions apply.go contains the concrete Action implementations.
//
// # Builder Functions (Functional Pattern)
//
// Most actions here are constructed with builder functions like gainCoinsAction(4).
// These return an Action interface implemented by a closure that captures the parameter.
// This is idiomatic Go functional style — it avoids a new struct type for every variant:
//
//	gainCoinsAction(1)  // Income
//	gainCoinsAction(2)  // ForeignAid
//	gainCoinsAction(4)  // Business
//
// All three share the same logic, just with a different captured `amount`.
//
// # Two-Phase Actions
//
// runExchange and runDiscardAction are two-phase (see runtime.go for the full explanation).
// You can spot them by the `if ctx.Step == nil { ... }` pattern at the top of the function:
//   - Step is nil → first call, set up the prompt
//   - Step is non-nil → second call, apply the player's response
package actions

import (
	"fmt"
	"math/rand"

	"github.com/axwell0/elmakina/engine/core/state"
	"github.com/axwell0/elmakina/engine/core/types"
)

// randIntn is a variable so tests can replace it with a deterministic function.
// This is a common Go testability pattern: instead of calling rand.Intn directly
// (which would produce unpredictable results in tests), we swap this out in tests
// to control which card gets selected during Investigate.
var randIntn = rand.Intn

// completeApplied is a helper that runs `run`, then marks the action as applied in the
// record and the context. It's a higher-order function: it takes a function and wraps it.
// Calling it is equivalent to: run the logic, then call AppendActionApplied + Complete.
func completeApplied(ctx *ActionContext, run func() error) error {
	if err := run(); err != nil {
		return err
	}
	ctx.AppendActionApplied()
	ctx.Complete()
	return nil
}

type actionFunc func(*ActionContext) error

func (a actionFunc) Run(ctx *ActionContext) error {
	return a(ctx)
}

func gainCoinsAction(amount int) ActionRunner {
	return actionFunc(func(ctx *ActionContext) error {
		return completeApplied(ctx, func() error {
			return ctx.AddCoins(ctx.Command.ActorIndex, amount)
		})
	})
}

func noOpAction() ActionRunner {
	return actionFunc(func(ctx *ActionContext) error {
		return completeApplied(ctx, func() error { return nil })
	})
}

type ExchangeAction struct{}

func (ExchangeAction) Run(ctx *ActionContext) error {
	return runExchange(ctx)
}

type DiscardAction struct {
	label string
}

func (a DiscardAction) Run(ctx *ActionContext) error {
	return runDiscardAction(ctx, a.label)
}

func discardAction(label string) ActionRunner {
	return DiscardAction{label: label}
}

func runExchange(ctx *ActionContext) error {
	if ctx.Step == nil {
		source := &ctx.State.players[ctx.Command.ActorIndex]
		handCount := len(source.Hand)
		if handCount == 0 {
			return errNoCardsToExchange
		}
		_, err := ctx.DrawCards(ctx.Command.ActorIndex, handCount)
		if err != nil {
			return fmt.Errorf("draw exchange cards: %w", err)
		}
		req := StepRequest{
			Kind:        StepCardSelection,
			Action:      ctx.Command.Action,
			ActorIndex:  ctx.Command.ActorIndex,
			Title:       "Choose your cards",
			Description: "Select the cards you want to keep after drawing.",
			CardSelection: &CardSelectionStep{
				Count: handCount,
				Cards: cardOptions(source.Hand),
			},
		}
		ctx.Need(req, exchangeState{
			ActorIndex:  ctx.Command.ActorIndex,
			KeepCount:   handCount,
			RequestData: req,
		})
		return nil
	}

	stepState, err := RequireStepState[exchangeState](ctx, "exchange")
	if err != nil {
		return fmt.Errorf("read exchange step: %w", err)
	}
	selection, err := RequireResponse[CardSelectionResponse](ctx)
	if err != nil {
		return fmt.Errorf("read exchange response: %w", err)
	}
	if err := validateCardSelection(selection, stepState.KeepCount, len(ctx.State.players[ctx.Command.ActorIndex].Hand), "exchange"); err != nil {
		return fmt.Errorf("validate exchange selection: %w", err)
	}
	if _, err := ctx.ReturnCardsToDeck(ctx.Command.ActorIndex, selection.SelectedCardIndices); err != nil {
		return fmt.Errorf("return exchange cards: %w", err)
	}
	ctx.AppendActionApplied()
	ctx.Complete()
	return nil
}

// runTax implements the Tax Collector action: take 1 coin from every opponent who has
// 7+ coins AND still has cards (eliminated players aren't taxed). The 7-coin threshold
// is a game rule — Tax only targets players who are wealthy enough that they'd otherwise
// be forced to coup on their next turn, making it a political/timing action.
func runTax(ctx *ActionContext) error {
	return completeApplied(ctx, func() error {
		for i := range ctx.State.players {
			if i == ctx.Command.ActorIndex {
				continue
			}
			if ctx.State.players[i].Coins >= 7 && len(ctx.State.players[i].Hand) > 0 {
				if err := ctx.AddCoins(i, -1); err != nil {
					return fmt.Errorf("tax player %d: %w", i, err)
				}
			}
		}
		return nil
	})
}

func runTaxBusinessWoman(ctx *ActionContext) error {
	return completeApplied(ctx, func() error {
		if err := validatePlayerIndex(ctx.State, ctx.State.CurrentPlayerIndex); err != nil {
			return fmt.Errorf("validate business tax target: %w", err)
		}
		if err := ctx.AddCoins(ctx.State.CurrentPlayerIndex, -1); err != nil {
			return fmt.Errorf("collect business tax: %w", err)
		}
		if err := ctx.AddCoins(ctx.Command.ActorIndex, 1); err != nil {
			return fmt.Errorf("pay business tax collector: %w", err)
		}
		return nil
	})
}

func runInvestigate(ctx *ActionContext) error {
	return completeApplied(ctx, func() error {
		payload, err := ctx.TargetPayload()
		if err != nil {
			return fmt.Errorf("read investigate payload: %w", err)
		}
		if err := validatePlayerIndex(ctx.State, payload.TargetIndex); err != nil {
			return fmt.Errorf("validate investigate target: %w", err)
		}
		target := &ctx.State.players[payload.TargetIndex]
		if len(target.Hand) == 0 {
			return errTargetHasNoCards
		}
		cardIndex := randIntn(len(target.Hand))
		card := target.Hand[cardIndex]
		ctx.ObservePrivateCard(ctx.Command.ActorIndex, target.ID, card)
		return nil
	})
}

func runAccuse(ctx *ActionContext) error {
	return completeApplied(ctx, func() error {
		payload, err := ctx.AccusePayload()
		if err != nil {
			return fmt.Errorf("read accuse payload: %w", err)
		}
		if err := validatePlayerIndex(ctx.State, payload.TargetIndex); err != nil {
			return fmt.Errorf("validate accuse target: %w", err)
		}
		target := &ctx.State.players[payload.TargetIndex]
		for i, card := range target.Hand {
			if card.Role != payload.Guess {
				continue
			}
			if _, err := ctx.DiscardFromHand(payload.TargetIndex, i); err != nil {
				return fmt.Errorf("discard accused card: %w", err)
			}
			if len(ctx.State.players[payload.TargetIndex].Hand) == 0 {
				ctx.AppendPlayerEliminated(payload.TargetIndex)
			}
			return nil
		}
		if err := ctx.AddCoins(payload.TargetIndex, 4); err != nil {
			return fmt.Errorf("pay false accusation penalty: %w", err)
		}
		return nil
	})
}

func runSteal(ctx *ActionContext) error {
	return completeApplied(ctx, func() error {
		payload, err := ctx.TargetPayload()
		if err != nil {
			return fmt.Errorf("read steal payload: %w", err)
		}
		if err := validatePlayerIndex(ctx.State, payload.TargetIndex); err != nil {
			return fmt.Errorf("validate steal target: %w", err)
		}
		if ctx.State.players[payload.TargetIndex].Coins < 2 {
			return errTargetHasTooFewCoins
		}
		if err := ctx.AddCoins(payload.TargetIndex, -2); err != nil {
			return fmt.Errorf("remove stolen coins: %w", err)
		}
		if err := ctx.AddCoins(ctx.Command.ActorIndex, 2); err != nil {
			return fmt.Errorf("add stolen coins: %w", err)
		}
		return nil
	})
}

type exchangeState struct {
	ActorIndex  int
	KeepCount   int
	RequestData StepRequest
}

func (s exchangeState) ActionID() types.ActionID {
	return types.Exchange
}

func (s exchangeState) ResponderIndex() int {
	return s.ActorIndex
}

func (s exchangeState) StepRequest() StepRequest {
	return s.RequestData
}

type discardState struct {
	Action      types.ActionID
	TargetIndex int
	Label       string
	RequestData StepRequest
}

func (s discardState) ActionID() types.ActionID {
	return s.Action
}

func (s discardState) ResponderIndex() int {
	return s.TargetIndex
}

func (s discardState) StepRequest() StepRequest {
	return s.RequestData
}

func runDiscardAction(ctx *ActionContext, actionLabel string) error {
	if ctx.Step == nil {
		payload, err := ctx.TargetPayload()
		if err != nil {
			return fmt.Errorf("read discard payload: %w", err)
		}
		if err := validatePlayerIndex(ctx.State, payload.TargetIndex); err != nil {
			return fmt.Errorf("validate discard target: %w", err)
		}
		// IMPORTANT: the ActorIndex here is payload.TargetIndex (the victim), NOT the attacker.
		// The target is the one who must choose which of their own cards to discard.
		// This is by game design — the attacker decides to attack, the victim decides which
		// influence to lose.
		req := StepRequest{
			Kind:        StepCardSelection,
			Action:      ctx.Command.Action,
			ActorIndex:  payload.TargetIndex,
			Title:       "Choose a card to lose",
			Description: "Select the card that should be discarded.",
			CardSelection: &CardSelectionStep{
				Count: 1,
				Cards: cardOptions(ctx.State.players[payload.TargetIndex].Hand),
			},
		}
		ctx.Need(req, discardState{
			Action:      ctx.Command.Action,
			TargetIndex: payload.TargetIndex,
			Label:       actionLabel,
			RequestData: req,
		})
		return nil
	}

	stepState, err := RequireStepState[discardState](ctx, actionLabel)
	if err != nil {
		return fmt.Errorf("read discard step: %w", err)
	}
	selection, err := RequireResponse[CardSelectionResponse](ctx)
	if err != nil {
		return fmt.Errorf("read discard response: %w", err)
	}
	if err := validateCardSelection(selection, 1, len(ctx.State.players[stepState.TargetIndex].Hand), stepState.Label); err != nil {
		return fmt.Errorf("validate discard selection: %w", err)
	}
	if err := discardSelectedCard(ctx, stepState.TargetIndex, selection.SelectedCardIndices[0]); err != nil {
		return fmt.Errorf("discard selected card: %w", err)
	}
	ctx.AppendActionApplied()
	ctx.Complete()
	return nil
}

func validateCardSelection(selection CardSelectionResponse, expectedCount, optionCount int, label string) error {
	if len(selection.SelectedCardIndices) != expectedCount {
		return fmt.Errorf("%w to %s: expected %d card index(es), got %d", errInvalidResponse, label, expectedCount, len(selection.SelectedCardIndices))
	}
	selected := make(map[int]struct{}, len(selection.SelectedCardIndices))
	for _, idx := range selection.SelectedCardIndices {
		if idx < 0 || idx >= optionCount {
			return fmt.Errorf("%w: %d", errInvalidCardIndex, idx)
		}
		if _, exists := selected[idx]; exists {
			return fmt.Errorf("%w: %d", errDuplicateCardIndex, idx)
		}
		selected[idx] = struct{}{}
	}
	return nil
}

func discardSelectedCard(ctx *ActionContext, targetIndex, selectedIndex int) error {
	if _, err := ctx.DiscardFromHand(targetIndex, selectedIndex); err != nil {
		return fmt.Errorf("discard from hand: %w", err)
	}
	if len(ctx.State.players[targetIndex].Hand) != 0 {
		return nil
	}
	ctx.AppendPlayerEliminated(targetIndex)
	if ctx.Command.Action != types.Coup && ctx.Command.Action != types.Assassinate {
		return nil
	}
	coins := ctx.State.players[targetIndex].Coins
	if coins <= 0 {
		return nil
	}
	if _, err := ctx.TransferCoins(targetIndex, ctx.Command.ActorIndex, coins); err != nil {
		return fmt.Errorf("transfer eliminated player coins: %w", err)
	}
	return nil
}

func cardOptions(hand []state.Card) []CardOption {
	options := make([]CardOption, 0, len(hand))
	for i, card := range hand {
		label := card.Role
		if label == "" {
			label = types.Role(fmt.Sprintf("Card %d", i+1))
		}
		options = append(options, CardOption{Index: i, Label: string(label)})
	}
	return options
}
