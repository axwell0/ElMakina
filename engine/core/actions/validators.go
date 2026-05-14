package actions

import (
	"fmt"
	"slices"

	game "github.com/axwell0/elmakina/engine/core/state"
	models "github.com/axwell0/elmakina/engine/core/types"
)

// validators.go is the declaration gate for action commands.
//
// These checks must remain side-effect free: they decide whether execution is
// allowed, but never mutate GameState.

// ValidateMainAction runs generic and action-specific checks for a declared main action.
func ValidateMainAction(s *game.GameState, cmd *models.PlayerAction) error {
	def, err := mainActionDefForCommand(cmd)
	if err != nil {
		return err
	}
	return validatePlayerActionWithDef(s, cmd, def)
}

// ValidateCounterAction runs generic and action-specific checks for a counter
// in the context of the main action it is answering.
func ValidateCounterAction(s *game.GameState, cmd *models.PlayerAction, mainAction models.ActionID) error {
	def, err := counterActionDefForCommand(cmd)
	if err != nil {
		return err
	}
	if !slices.Contains(def.Counter.AllowedAgainst, mainAction) {
		return fmt.Errorf("%w: %q cannot counter %q", errCounterNotAllowed, cmd.Action, mainAction)
	}
	return validatePlayerActionWithDef(s, cmd, def)
}

// ValidatePlayerAction runs generic and action-specific checks before execution.
func ValidatePlayerAction(s *game.GameState, cmd *models.PlayerAction) error {
	if cmd == nil {
		return errCommandNil
	}
	def, ok := Action(cmd.Action)
	if !ok {
		return fmt.Errorf("%w: %s", errUnknownAction, cmd.Action)
	}
	return validatePlayerActionWithDef(s, cmd, def)
}

func mainActionDefForCommand(cmd *models.PlayerAction) (ActionDefinition, error) {
	if cmd == nil {
		return ActionDefinition{}, errCommandNil
	}
	def, ok := Action(cmd.Action)
	if !ok {
		return ActionDefinition{}, fmt.Errorf("%w: %s", errUnknownAction, cmd.Action)
	}
	if def.IsCounter() {
		return ActionDefinition{}, fmt.Errorf("%w: %q is not a main action", errWrongActionKind, cmd.Action)
	}
	return def, nil
}

func counterActionDefForCommand(cmd *models.PlayerAction) (ActionDefinition, error) {
	if cmd == nil {
		return ActionDefinition{}, errCommandNil
	}
	def, ok := Action(cmd.Action)
	if !ok {
		return ActionDefinition{}, fmt.Errorf("%w: %s", errUnknownAction, cmd.Action)
	}
	if !def.IsCounter() {
		return ActionDefinition{}, fmt.Errorf("%w: %q is not a counter action", errWrongActionKind, cmd.Action)
	}
	return def, nil
}

func validatePlayerActionWithDef(s *game.GameState, cmd *models.PlayerAction, def ActionDefinition) error {
	if cmd == nil {
		return errCommandNil
	}
	if err := validatePlayerIndex(s, cmd.ActorIndex); err != nil {
		return fmt.Errorf("validate actor: %w", err)
	}

	if s.players[cmd.ActorIndex].Coins < def.Cost {
		return fmt.Errorf("%w: have %d, need %d", errNotEnoughCoins, s.players[cmd.ActorIndex].Coins, def.Cost)
	}

	if err := validatePayloadShape(cmd.Payload, def.Payload); err != nil {
		return err
	}

	ctx := CommandContext{
		GameState: s,
		Command:   cmd,
		Def:       def,
	}
	for _, criterion := range def.Command {
		if criterion == nil {
			continue
		}
		if err := criterion.ValidateCommand(ctx); err != nil {
			return fmt.Errorf("validate command criterion: %w", err)
		}
	}

	return nil
}

func validatePlayerIndex(s *game.GameState, playerIndex int) error {
	if s == nil {
		return errStateNil
	}
	if playerIndex < 0 || playerIndex >= len(s.players) {
		return fmt.Errorf("%w: index %d out of range", errInvalidPlayerIndex, playerIndex)
	}
	return nil
}

// validateTargetAlive ensures the target exists and is not eliminated.
func validateTargetAlive(s *game.GameState, targetIndex int) error {
	if err := validatePlayerIndex(s, targetIndex); err != nil {
		return fmt.Errorf("validate target: %w", err)
	}
	if len(s.players[targetIndex].Hand) == 0 {
		return fmt.Errorf("%w: target %d", errTargetEliminated, targetIndex)
	}
	return nil
}

// validateTargetHasCoins enforces minimum stealable coins for target actions.
func validateTargetHasCoins(s *game.GameState, targetIndex, minimum int) error {
	if s.players[targetIndex].Coins < minimum {
		return fmt.Errorf("%w: target %d does not have %d coins", errTargetHasTooFewCoins, targetIndex, minimum)
	}
	return nil
}

// ActionCost returns an action's configured cost, or 0 if unknown.
func ActionCost(id models.ActionID) int {
	if def, ok := Action(id); ok {
		return def.Cost
	}
	return 0
}

func validatePayloadShape(actual, expected models.ActionPayload) error {
	switch expected.(type) {
	case nil:
		if actual != nil {
			return errPayloadNotAllowed
		}
	case models.TargetPayload:
		if _, ok := actual.(models.TargetPayload); !ok {
			return fmt.Errorf("%w: expected TargetPayload, got %T", errUnsupportedPayloadShape, actual)
		}
	case models.AccusePayload:
		if _, ok := actual.(models.AccusePayload); !ok {
			return fmt.Errorf("%w: expected AccusePayload, got %T", errUnsupportedPayloadShape, actual)
		}
	default:
		return fmt.Errorf("%w: unsupported template %T", errUnsupportedPayloadShape, expected)
	}
	return nil
}

// hasTaxableOpponent checks whether any live opponent currently has 7+ coins.
func hasTaxableOpponent(gs *game.GameState, actorIndex int) bool {
	if gs == nil {
		return false
	}
	for i, player := range gs.players {
		if i == actorIndex {
			continue
		}
		if len(player.Hand) == 0 {
			continue
		}
		if player.Coins >= 7 {
			return true
		}
	}
	return false
}

type CommandContext struct {
	GameState *game.GameState
	Command   *models.PlayerAction
	Def       ActionDefinition
}

type CommandCriterion interface {
	ValidateCommand(ctx CommandContext) error
}

type TargetAliveCriterion struct{}

func (TargetAliveCriterion) ValidateCommand(ctx CommandContext) error {
	targetIndex, err := targetIndexFromCommand(ctx.Command)
	if err != nil {
		return err
	}
	return validateTargetAlive(ctx.GameState, targetIndex)
}

type TargetCoinCriterion struct {
	MinCoins int
}

func (c TargetCoinCriterion) ValidateCommand(ctx CommandContext) error {
	targetIndex, err := targetIndexFromCommand(ctx.Command)
	if err != nil {
		return err
	}
	return validateTargetHasCoins(ctx.GameState, targetIndex, c.MinCoins)
}

type TaxableOpponentCriterion struct{}

func (TaxableOpponentCriterion) ValidateCommand(ctx CommandContext) error {
	if !hasTaxableOpponent(ctx.GameState, ctx.Command.ActorIndex) {
		return errNoTaxableOpponent
	}
	return nil
}

func targetIndexFromCommand(cmd *models.PlayerAction) (int, error) {
	switch payload := cmd.Payload.(type) {
	case models.TargetPayload:
		return payload.TargetIndex, nil
	case models.AccusePayload:
		return payload.TargetIndex, nil
	default:
		return 0, fmt.Errorf("%w: expected target payload, got %T", errUnsupportedPayloadShape, cmd.Payload)
	}
}
