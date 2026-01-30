package actions

import (
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"fmt"
)

// ValidatePlayerAction checks if an PlayerAction is valid before execution.
//
// This is the first line of defense against invalid moves. It validates:
//   - Command is not nil
//   - Source player index is valid
//   - Action ID exists in ActionRegistry
//   - Player has enough coins to pay the cost
//   - Action-specific validator (if defined) passes
//
// Called by the orchestrator before paying costs or executing the action.
// If validation fails, the turn is rejected and the player must try again.
//
// See: orchestrator_turn.go:67 for call site in turn flow.
// See: registry.go:102 for ActionRegistry lookup.
// See: tests/actions_test.go:40 for test examples.
func ValidatePlayerAction(s *state.GameState, cmd *models.PlayerAction) error {
	if cmd == nil {
		return fmt.Errorf("command is nil")
	}

	if err := validatePlayerIndex(s, cmd.SourceIndex, "source"); err != nil {
		return err
	}

	def, ok := ActionRegistry[cmd.ID]
	if !ok {
		return fmt.Errorf("unknown action: %s", cmd.ID)
	}

	if s.Players[cmd.SourceIndex].Coins < def.Cost {
		return fmt.Errorf("not enough coins: have %d, need %d", s.Players[cmd.SourceIndex].Coins, def.Cost)
	}

	if def.Validator != nil {
		if err := def.Validator(s, cmd); err != nil {
			return err
		}
	}

	return nil
}

// validatePlayerIndex checks that a player index is within the valid range.
// Used by validators to ensure source and target indices reference actual players.
func validatePlayerIndex(s *state.GameState, index int, role string) error {
	if index < 0 || index >= len(s.Players) {
		return fmt.Errorf("%s index %d out of range", role, index)
	}
	return nil
}

// validateTargetAlive checks that the target player is still in the game.
// A player is eliminated when they have no cards left in their hand.
func validateTargetAlive(s *state.GameState, targetIndex int) error {
	if err := validatePlayerIndex(s, targetIndex, "target"); err != nil {
		return err
	}
	if len(s.Players[targetIndex].Hand) == 0 {
		return fmt.Errorf("target %d is eliminated", targetIndex)
	}
	return nil
}

// validateTargetHasCoins checks that the target player has at least the minimum coins.
// Used by Steal action to ensure there are coins to steal.
func validateTargetHasCoins(s *state.GameState, targetIndex int, minimum int) error {
	if s.Players[targetIndex].Coins < minimum {
		return fmt.Errorf("target %d does not have %d coins", targetIndex, minimum)
	}
	return nil
}

// PayActionCost deducts the action's cost from the source player's coins.
//
// Called by the orchestrator after validation but before execution.
// If the action is later canceled (by challenge or counter), the cost
// is refunded by the orchestrator.
func PayActionCost(s *state.GameState, cmd *models.PlayerAction) error {
	cost := ActionCost(cmd.ID)
	if cost > 0 {
		if _, err := s.AddCoins(cmd.SourceIndex, -cost); err != nil {
			return err
		}
	}
	return nil
}

// ActionCost returns the coin cost for an action.
// Looks up the cost in ActionRegistry. Returns 0 if action not found.
func ActionCost(id models.ActionID) int {
	if def, ok := ActionRegistry[id]; ok {
		return def.Cost
	}
	return 0
}

// ApplyAction executes an action's handler against the game state.
//
// This is the entry point for action execution. It:
//  1. Looks up the action definition in ActionRegistry
//  2. Calls the registered handler function
//  3. Returns any errors from the handler
//
// The handler may use the ActionFlow to pause and request player input
// (see flow.go for details on multi-step actions).
//
// See: orchestrator_turn.go:163 for call site in turn flow.
// See: registry.go:102 for ActionRegistry lookup.
// See: apply.go for handler implementations.
func ApplyAction(flow *ActionFlow, s *state.GameState, turn *state.TurnState, log *state.TurnLog, cmd *models.PlayerAction) error {
	def, ok := ActionRegistry[cmd.ID]
	if !ok {
		return fmt.Errorf("unknown action: %s", cmd.ID)
	}
	return def.Handler(flow, s, turn, log, cmd)
}

// targetPayload extracts and validates a TargetPayload from an PlayerAction.
// Returns error if payload is not a TargetPayload type.
func targetPayload(cmd *models.PlayerAction) (models.TargetPayload, error) {
	payload, ok := cmd.Payload.(models.TargetPayload)
	if !ok {
		return models.TargetPayload{}, fmt.Errorf("expected TargetPayload, got %T", cmd.Payload)
	}
	return payload, nil
}

// accusePayload extracts and validates an AccusePayload from an PlayerAction.
// Returns error if payload is not an AccusePayload type.
func accusePayload(cmd *models.PlayerAction) (models.AccusePayload, error) {
	payload, ok := cmd.Payload.(models.AccusePayload)
	if !ok {
		return models.AccusePayload{}, fmt.Errorf("expected AccusePayload, got %T", cmd.Payload)
	}
	return payload, nil
}

// Action-specific validators

func validateNoPayload(s *state.GameState, cmd *models.PlayerAction) error {
	if cmd.Payload != nil {
		return fmt.Errorf("payload is not allowed for this action")
	}
	return nil
}

func validateCoup(s *state.GameState, cmd *models.PlayerAction) error {
	payload, err := targetPayload(cmd)
	if err != nil {
		return err
	}
	return validateTargetAlive(s, payload.TargetIndex)
}

func validateTax(s *state.GameState, cmd *models.PlayerAction) error {
	if err := validateNoPayload(s, cmd); err != nil {
		return err
	}
	if cmd.MainAction != "" {
		return fmt.Errorf("tax is not a counter action")
	}
	if !hasTaxablePlayer(s) {
		return fmt.Errorf("no players have 7+ coins to tax")
	}
	return nil
}

func hasTaxablePlayer(gs *state.GameState) bool {
	if gs == nil {
		return false
	}
	for _, player := range gs.Players {
		if len(player.Hand) == 0 {
			continue
		}
		if player.Coins >= 7 {
			return true
		}
	}
	return false
}

func validateTaxBusinessWoman(s *state.GameState, cmd *models.PlayerAction) error {
	if err := validateNoPayload(s, cmd); err != nil {
		return err
	}
	if cmd.MainAction != models.Business {
		return fmt.Errorf("tax business woman must counter take4")
	}
	return nil
}

func validateInvestigate(s *state.GameState, cmd *models.PlayerAction) error {
	payload, err := targetPayload(cmd)
	if err != nil {
		return err
	}
	return validateTargetAlive(s, payload.TargetIndex)
}

func validateAccuse(s *state.GameState, cmd *models.PlayerAction) error {
	payload, err := accusePayload(cmd)
	if err != nil {
		return err
	}
	return validateTargetAlive(s, payload.TargetIndex)
}

func validateAssassinate(s *state.GameState, cmd *models.PlayerAction) error {
	payload, err := targetPayload(cmd)
	if err != nil {
		return err
	}
	return validateTargetAlive(s, payload.TargetIndex)
}

func validateSteal(s *state.GameState, cmd *models.PlayerAction) error {
	payload, err := targetPayload(cmd)
	if err != nil {
		return err
	}
	if err := validateTargetAlive(s, payload.TargetIndex); err != nil {
		return err
	}
	return validateTargetHasCoins(s, payload.TargetIndex, 2)
}
