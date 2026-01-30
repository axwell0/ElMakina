package actions

import (
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

// ActionHandler is the signature for functions that "do the thing" an action promises.
//
// Think of these as the "Effect" functions. When a player says "I want to Coup Bob",
// the validator first checks "can they afford it?" (validateCoup), then if that passes,
// this handler actually moves the coins and triggers the card discard.
//
// The flow parameter is only used for multi-step actions (like Exchange where the
// player needs to select which cards to keep). Simple actions like Income ignore it.
//
// See: apply.go for handler implementations (applySteal, applyCoup, etc.).
// See: registry.go:102 for ActionRegistry mapping.
type ActionHandler func(flow *ActionFlow, s *state.GameState, turn *state.TurnState, log *state.TurnLog, cmd *models.PlayerAction) error

// ActionValidator is the signature for functions that check "is this legal right now?"
//
// Validators are the gatekeepers. They check things like:
//   - Does the player have enough coins for a Coup?
//   - Is the target index valid?
//   - Can they actually use this action in the current game state?
//
// Validators run BEFORE any coins are spent or effects applied. If validation fails,
// the action is rejected immediately without side effects.
//
// See: validators.go for validator implementations.
// See: orchestrator_turn.go:67 for validation call site.
type ActionValidator func(s *state.GameState, cmd *models.PlayerAction) error

// ActionDef is the "Blueprint" for a single game action.
//
// This struct is the Rosetta Stone for understanding what an action does, what it costs,
// and when it can be used. Every possible move in the game is defined here with enough
// metadata for the engine to enforce rules without hardcoding action-specific logic.
//
// The separation of Handler and Validator is intentional:
//   - Validator answers: "CAN we do this?" (preconditions)
//   - Handler answers: "WHAT happens when we do?" (effects)
//
// This lets us fail fast (validation) before committing to potentially irreversible
// state changes (handler execution).
type ActionDef struct {
	// ID is the unique identifier for this action (e.g., "coup", "steal", "block_steal")
	ID models.ActionID

	// Kind distinguishes between main actions (your turn) and counter actions (blocking)
	Kind models.ActionKind

	// Role is the claimed role for this action (e.g., "Thief" for Steal).
	// Empty string means no role claim (anyone can do it, like Income).
	// Non-empty means the action can be CHALLENGED—other players can call "liar!"
	Role models.Role

	// AllowedAgainst lists which main actions this counter can block.
	// Only relevant for counter actions (Kind == CounterAction).
	// Example: BlockSteal can only block Steal.
	AllowedAgainst []models.ActionID

	// MultiCounter allows multiple players to use this counter simultaneously.
	// This is used for TaxBusinessWoman where multiple TaxCollectors can tax
	// the same Businesswoman in response to her Business action.
	MultiCounter bool

	// Cost is the coin price to perform this action.
	// Paid immediately when the action is declared, refunded if the action
	// is canceled by a challenge or counter.
	Cost int

	// Handler is the function that actually executes the action's game effects.
	// Called only after validation passes and the action survives challenges/counters.
	Handler ActionHandler

	// Validator checks if this action is legal in the current game state.
	// Called immediately when the action is declared, before any costs are paid.
	Validator ActionValidator
}

// ActionRegistry is the "Rulebook in Code"—every possible move defined in one place.
//
// This map is the authoritative source for "what can players do and what does it cost?"
// The engine consults this registry to:
//   - Validate that an action exists (is this a real move?)
//   - Check if an action can be challenged (does it have a Role?)
//   - Find counters for an action (what can block this?)
//   - Execute the action's effects (call the Handler)
//
// ORGANIZATIONAL PHILOSOPHY:
// Actions are grouped by their game concept, not alphabetically. This makes it easier
// to reason about related mechanics:
//
// 1. BASE ACTIONS (Income, ForeignAid): Simple economy moves anyone can do
// 2. ROLE ACTIONS (Tax, Steal, etc.): Actions claiming a specific role (can be challenged)
// 3. COUNTERS (BlockSteal, BlockTerrorist): Defensive responses to role actions
// 4. SPECIAL (Coup, Exchange): Actions with unique mechanics
//
// WHY THIS MATTERS:
// Adding a new role? You add 1-2 entries here (main action + counter).
// Changing balance? Modify the Cost field.
// Adding a new action? Follow the existing pattern: ID, Kind, Role, Handler, Validator.
//
// The engine doesn't know about "Tax" being special—it just looks up "tax" in this
// map and follows the instructions. This decouples game design from engine logic.
var ActionRegistry = map[models.ActionID]ActionDef{
	// Business (Businesswoman): Take 4 coins, claim Businesswoman role
	// High reward (4 coins) but vulnerable to Tax counter
	models.Business: {
		ID:        models.Business,
		Kind:      models.MainAction,
		Role:      models.Businesswoman,
		Handler:   applyBusiness,
		Validator: validateNoPayload,
	},

	models.Income: {
		ID:        models.Income,
		Kind:      models.MainAction,
		Handler:   applyIncome,
		Validator: validateNoPayload,
	},
	models.ForeignAid: {
		ID:        models.ForeignAid,
		Kind:      models.MainAction,
		Handler:   applyForeignAid,
		Validator: validateNoPayload,
	},
	models.Coup: {
		ID:        models.Coup,
		Kind:      models.MainAction,
		Cost:      7,
		Handler:   applyCoup,
		Validator: validateCoup,
	},
	models.Tax: {
		ID:        models.Tax,
		Kind:      models.MainAction,
		Role:      models.TaxCollector,
		Handler:   applyTax,
		Validator: validateTax,
	},
	models.TaxBusinessWoman: {
		ID:             models.TaxBusinessWoman,
		Kind:           models.CounterAction,
		Role:           models.TaxCollector,
		AllowedAgainst: []models.ActionID{models.Business},
		MultiCounter:   true,
		Handler:        applyTaxBusinessWoman,
		Validator:      validateTaxBusinessWoman,
	},

	models.Investigate: {
		ID:        models.Investigate,
		Kind:      models.MainAction,
		Role:      models.Policewoman,
		Handler:   applyInvestigate,
		Validator: validateInvestigate,
	},

	models.Accuse: {
		ID:        models.Accuse,
		Kind:      models.MainAction,
		Role:      models.Colonel,
		Cost:      4,
		Handler:   applyAccuse,
		Validator: validateAccuse,
	},

	models.Assassinate: {
		ID:        models.Assassinate,
		Kind:      models.MainAction,
		Role:      models.Terrorist,
		Cost:      3,
		Handler:   applyAssassinate,
		Validator: validateAssassinate,
	},

	models.Steal: {
		ID:        models.Steal,
		Kind:      models.MainAction,
		Role:      models.Thief,
		Handler:   applySteal,
		Validator: validateSteal,
	},
	models.BlockSteal: {
		ID:             models.BlockSteal,
		Kind:           models.CounterAction,
		AllowedAgainst: []models.ActionID{models.Steal},
		Role:           models.Thief,
		Handler:        applyBlock,
		Validator:      validateNoPayload,
	},
	models.BlockTerrorist: {
		ID:             models.BlockTerrorist,
		Kind:           models.CounterAction,
		AllowedAgainst: []models.ActionID{models.Assassinate},
		Role:           models.Colonel,
		Handler:        applyBlock,
		Validator:      validateNoPayload,
	},
	models.BlockPolice: {
		ID:             models.BlockPolice,
		Kind:           models.CounterAction,
		AllowedAgainst: []models.ActionID{models.Investigate},
		Role:           models.Policewoman,
		Handler:        applyBlock,
		Validator:      validateNoPayload,
	},
	models.BlockForeignAid: {
		ID:             models.BlockForeignAid,
		Kind:           models.CounterAction,
		AllowedAgainst: []models.ActionID{models.ForeignAid},
		Role:           models.TaxCollector,
		Handler:        applyBlock,
		Validator:      validateNoPayload,
	},

	models.Exchange: {
		ID:        models.Exchange,
		Kind:      models.MainAction,
		Role:      models.Politician,
		Handler:   applyExchange,
		Validator: validateNoPayload,
	},

	models.Escape: {
		ID:             models.Escape,
		Kind:           models.CounterAction,
		AllowedAgainst: []models.ActionID{models.Coup, models.Assassinate},
		Cost:           9,
		Handler:        applyEscape,
		Validator:      validateNoPayload,
	},
	models.Pass: {
		ID:        models.Pass,
		Kind:      models.MainAction,
		Handler:   applyPass,
		Validator: validateNoPayload,
	},
}
