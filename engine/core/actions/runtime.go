// Package actions runtime.go handles two-phase action execution.
//
// # Two-Phase Actions
//
// Most actions (Income, Steal, etc.) execute in a single step: the engine calls Run(),
// state changes, done. But some actions need a mid-execution player input:
//
//   - Exchange: "draw 2 cards, then pick which cards you want to keep"
//   - Coup/Assassinate: "you must discard a card — which one?"
//
// For these, Run() is called twice:
//  1. First call (ctx.Step == nil): set up the prompt, call ctx.Need() to pause.
//     The engine surfaces the prompt to the UI and waits for the player.
//  2. Second call (ctx.Step != nil): the player responded. ctx.Response holds their
//     answer. Apply the state change and call ctx.Complete().
//
// ActionContext is the mutable execution context passed to both calls. It carries
// the game state, turn logs, and (on the second call) the player's response.
package actions

import (
	"fmt"

	"github.com/axwell0/elmakina/engine/core/state"
	"github.com/axwell0/elmakina/engine/core/types"
	"github.com/axwell0/elmakina/engine/turn"
)

type StepKind string

const (
	StepCardSelection StepKind = "card_selection"
)

// StepRequest describes what the UI should ask the player during a mid-action pause.
// The application layer converts this into a card selection prompt and broadcasts it.
type StepRequest struct {
	Kind          StepKind           `json:"kind"`
	Action        types.ActionID     `json:"action"`
	ActorIndex    int                `json:"actor_index"`
	Title         string             `json:"title,omitempty"`
	Description   string             `json:"description,omitempty"`
	CardSelection *CardSelectionStep `json:"card_selection,omitempty"`
}

type CardSelectionStep struct {
	Count int          `json:"count"`
	Cards []CardOption `json:"cards,omitempty"`
}

type CardOption struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}

type CardSelectionResponse struct {
	SelectedCardIndices []int `json:"selected_card_indices,omitempty"`
}

// ActionContinuation is action-owned data needed to resume a paused action.
type ActionContinuation interface {
	ActionID() types.ActionID
	ResponderIndex() int
	StepRequest() StepRequest
}

// ActionStep is the mid-action pause point. It carries the UI request and the
// action-owned continuation needed to resume safely.
type ActionStep struct {
	Action       types.ActionID
	ActorIndex   int
	Request      StepRequest
	Continuation ActionContinuation
}

// ActionResult is returned by Run(). The two fields are mutually exclusive:
//   - Step non-nil: the action is paused, waiting for player input. Not yet applied.
//   - Applied true, Step nil: the action completed successfully.
//   - Applied false, Step nil: the action was a no-op (e.g. counter that does nothing).
type ActionResult struct {
	Applied bool
	Step    *ActionStep
}

// ActionContext is passed to every Action.Run() call. It provides:
//   - State: the live game state to mutate
//   - Logs: the public/private turn log sink
//   - Command: the player's original action command
//   - Step: non-nil on a resume call (second phase of a two-phase action)
//   - Response: the player's answer to the step prompt (cast to the expected type)
//
// The unexported fields (applied, nextStep) are written by ctx.Complete() and ctx.Need(),
// then read by runAction() to build the ActionResult.
type ActionContext struct {
	State    *state.GameState
	Logs     *turn.TurnLog
	Command  *types.PlayerAction
	Step     *ActionStep
	Response any
	applied  bool        // set by Complete(); tells runAction() the action finished
	nextStep *ActionStep // set by Need(); tells runAction() to pause and wait for input
}

// Need pauses the action and requests player input.
// Call this when your action needs the player to pick something (e.g. which cards to keep).
// The engine will surface the StepRequest as a UI prompt and call Run() again with the
// player's response in ctx.Response. The continuation carries anything your action needs to
// remember between the two calls.
func (ctx *ActionContext) Need(req StepRequest, continuation ActionContinuation) {
	ctx.nextStep = &ActionStep{
		Action:       ctx.Command.Action,
		ActorIndex:   req.ActorIndex,
		Request:      req,
		Continuation: continuation,
	}
}

// Complete marks the action as successfully applied.
// Call this at the end of a successful Run() after all state changes are done.
// Without calling Complete(), the engine treats the action as a no-op.
func (ctx *ActionContext) Complete() {
	ctx.applied = true
}

func (ctx *ActionContext) AppendActionApplied() {
	if ctx == nil || ctx.Logs == nil || ctx.Command == nil {
		return
	}
	switch ctx.Command.Action {
	case types.Pass:
		ctx.Logs.Publicf("%s passes", ctx.playerName(ctx.Command.ActorIndex))
	case types.Escape:
		ctx.Logs.Publicf("%s escapes", ctx.playerName(ctx.Command.ActorIndex))
	}
}

func (ctx *ActionContext) AddCoins(playerIndex, delta int) error {
	if _, err := ctx.State.AddCoins(playerIndex, delta); err != nil {
		return fmt.Errorf("add coins for player %d: %w", playerIndex, err)
	}
	if delta > 0 {
		ctx.Logs.Publicf("%s gains %d coins", ctx.playerName(playerIndex), delta)
	} else if delta < 0 {
		ctx.Logs.Publicf("%s loses %d coins", ctx.playerName(playerIndex), -delta)
	}
	return nil
}

func (ctx *ActionContext) DrawCards(playerIndex, count int) (state.DrawDetail, error) {
	detail, err := ctx.State.DrawCards(playerIndex, count)
	if err != nil {
		return state.DrawDetail{}, fmt.Errorf("draw cards for player %d: %w", playerIndex, err)
	}
	ctx.Logs.Publicf("%s draws %d cards to exchange", ctx.playerName(detail.PlayerIndex), len(detail.Cards))
	return detail, nil
}

func (ctx *ActionContext) ReturnCardsToDeck(playerIndex int, cardIndices []int) (state.ReturnToDeckDetail, error) {
	detail, err := ctx.State.ReturnCardsToDeck(playerIndex, cardIndices)
	if err != nil {
		return state.ReturnToDeckDetail{}, fmt.Errorf("return cards for player %d: %w", playerIndex, err)
	}
	ctx.Logs.Publicf("%s finished exchange", ctx.playerName(detail.PlayerIndex))
	return detail, nil
}

func (ctx *ActionContext) DiscardFromHand(playerIndex, cardIndex int) (state.DiscardDetail, error) {
	detail, err := ctx.State.DiscardFromHand(playerIndex, cardIndex)
	if err != nil {
		return state.DiscardDetail{}, fmt.Errorf("discard card for player %d: %w", playerIndex, err)
	}
	ctx.Logs.Publicf("%s discards %s", ctx.playerName(detail.PlayerIndex), detail.Card.Role)
	return detail, nil
}

func (ctx *ActionContext) AppendPlayerEliminated(playerIndex int) {
	ctx.Logs.Publicf("%s is eliminated", ctx.playerName(playerIndex))
}

func (ctx *ActionContext) ObservePrivateCard(viewerIndex, ownerIndex int, card state.Card) {
	ctx.Logs.Privatef(viewerIndex, "You see %s has %s", ctx.playerName(ownerIndex), card.Role)
}

func (ctx *ActionContext) TransferCoins(fromPlayerIndex, toPlayerIndex, amount int) (int, error) {
	if err := validatePlayerIndex(ctx.State, fromPlayerIndex); err != nil {
		return 0, fmt.Errorf("validate transfer source: %w", err)
	}
	if err := validatePlayerIndex(ctx.State, toPlayerIndex); err != nil {
		return 0, fmt.Errorf("validate transfer destination: %w", err)
	}
	receiverCoins := ctx.State.players[toPlayerIndex].Coins
	if _, err := ctx.State.AddCoins(fromPlayerIndex, -amount); err != nil {
		return 0, fmt.Errorf("subtract transferred coins: %w", err)
	}
	newReceiverCoins, err := ctx.State.AddCoins(toPlayerIndex, amount)
	if err != nil {
		return 0, fmt.Errorf("add transferred coins: %w", err)
	}
	gained := newReceiverCoins - receiverCoins
	ctx.Logs.Publicf("%s takes %d coin(s) from %s", ctx.playerName(toPlayerIndex), gained, ctx.playerName(fromPlayerIndex))
	return gained, nil
}

func (ctx *ActionContext) playerName(index int) string {
	if ctx == nil || ctx.State == nil || index < 0 || index >= len(ctx.State.players) {
		return fmt.Sprintf("player-%d", index)
	}
	return ctx.State.players[index].Name
}

func (ctx *ActionContext) TargetPayload() (types.TargetPayload, error) {
	if ctx == nil || ctx.Command == nil {
		return types.TargetPayload{}, errActionContextNil
	}
	return targetPayloadFromAction(ctx.Command)
}

func (ctx *ActionContext) AccusePayload() (types.AccusePayload, error) {
	if ctx == nil {
		return types.AccusePayload{}, errActionContextNil
	}
	return accusePayloadFromAction(ctx.Command)
}

func RequireStepState[T any](ctx *ActionContext, label string) (T, error) {
	var zero T
	if ctx == nil || ctx.Step == nil {
		return zero, fmt.Errorf("%w: %s", errMissingContinuation, label)
	}
	value, ok := ctx.Step.Continuation.(T)
	if !ok {
		return zero, fmt.Errorf("%w: %s has %T", errInvalidContinuation, label, ctx.Step.Continuation)
	}
	return value, nil
}

func RequireResponse[T any](ctx *ActionContext) (T, error) {
	var zero T
	if ctx == nil {
		return zero, errActionContextNil
	}
	value, ok := ctx.Response.(T)
	if !ok {
		return zero, fmt.Errorf("%w: expected %T, got %T", errInvalidResponse, zero, ctx.Response)
	}
	return value, nil
}

// ApplyAction starts the first phase of an action.
// If the action needs player input, it returns a non-nil Step; call ResumeAction next.
// If the action completes immediately, ActionResult.Applied is true.
func ApplyAction(gs *state.GameState, logs *turn.TurnLog, cmd *types.PlayerAction) (ActionResult, error) {
	return runAction(&ActionContext{
		State:   gs,
		Logs:    logs,
		Command: cmd,
	})
}

// ResumeAction handles the second phase of a two-phase action.
// step is the ActionStep returned by the first ApplyAction call.
// resp is the player's response to the step prompt (e.g. CardSelectionResponse).
func ResumeAction(gs *state.GameState, logs *turn.TurnLog, step *ActionStep, resp any, cmd *types.PlayerAction) (ActionResult, error) {
	if step == nil {
		return ActionResult{}, errStepNil
	}
	if cmd == nil {
		return ActionResult{}, errCommandNil
	}
	if step.Action != cmd.Action {
		return ActionResult{}, fmt.Errorf("%w: resume step %s, command %s", errInvalidContinuation, step.Action, cmd.Action)
	}
	if step.Request.Action != cmd.Action {
		return ActionResult{}, fmt.Errorf("%w: request %s, command %s", errInvalidContinuation, step.Request.Action, cmd.Action)
	}
	if step.ActorIndex != step.Request.ActorIndex {
		return ActionResult{}, fmt.Errorf("%w: step actor %d, request actor %d", errInvalidContinuation, step.ActorIndex, step.Request.ActorIndex)
	}
	if step.Continuation == nil {
		return ActionResult{}, errStepContinuationNil
	}
	if step.Continuation.ActionID() != cmd.Action {
		return ActionResult{}, fmt.Errorf("%w: continuation %s, command %s", errInvalidContinuation, step.Continuation.ActionID(), cmd.Action)
	}
	if step.Continuation.ResponderIndex() != step.Request.ActorIndex {
		return ActionResult{}, fmt.Errorf("%w: continuation responder %d, request actor %d", errInvalidContinuation, step.Continuation.ResponderIndex(), step.Request.ActorIndex)
	}
	if err := validatePlayerIndex(gs, step.Request.ActorIndex); err != nil {
		return ActionResult{}, fmt.Errorf("validate step actor: %w", err)
	}
	if err := validateStepResponse(step.Request, resp); err != nil {
		return ActionResult{}, fmt.Errorf("validate step response: %w", err)
	}
	return runAction(&ActionContext{
		State:    gs,
		Logs:     logs,
		Command:  cmd,
		Step:     step,
		Response: resp,
	})
}

func validateStepResponse(req StepRequest, resp any) error {
	switch req.Kind {
	case StepCardSelection:
		if req.CardSelection == nil {
			return errCardSelectionOptions
		}
		if _, ok := resp.(CardSelectionResponse); !ok {
			return fmt.Errorf("%w: expected CardSelectionResponse, got %T", errInvalidResponse, resp)
		}
		return nil
	default:
		return fmt.Errorf("%w: %s", errUnknownStepKind, req.Kind)
	}
}

func runAction(actionCtx *ActionContext) (ActionResult, error) {
	if actionCtx.Command == nil {
		return ActionResult{}, errCommandNil
	}
	if err := validatePlayerIndex(actionCtx.State, actionCtx.Command.ActorIndex); err != nil {
		return ActionResult{}, fmt.Errorf("validate actor: %w", err)
	}
	def, ok := Action(actionCtx.Command.Action)
	if !ok {
		return ActionResult{}, fmt.Errorf("%w: %s", errUnknownAction, actionCtx.Command.Action)
	}
	if err := def.Runner.Run(actionCtx); err != nil {
		return ActionResult{}, fmt.Errorf("run action %s: %w", actionCtx.Command.Action, err)
	}
	return ActionResult{Applied: actionCtx.applied, Step: actionCtx.nextStep}, nil
}
