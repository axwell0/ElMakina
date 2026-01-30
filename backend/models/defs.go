package models

// ActionID is a string enum for all possible moves.
//
// See: actions/registry.go:102 for ActionRegistry mapping.
// See: orchestrator_turn.go:54 for processing.
type ActionID string

const (
	Business         ActionID = "businesswoman"
	BlockForeignAid  ActionID = "block_foreign_aid"
	Income           ActionID = "income"
	ForeignAid       ActionID = "foreign_aid"
	Coup             ActionID = "coup"
	Tax              ActionID = "tax"
	TaxBusinessWoman ActionID = "tax_business_woman"
	Investigate      ActionID = "investigate"
	BlockPolice      ActionID = "block_investigate"
	Accuse           ActionID = "accuse"
	BlockTerrorist   ActionID = "block_terrorist"
	Assassinate      ActionID = "assassinate"
	Steal            ActionID = "steal"
	BlockSteal       ActionID = "block_steal"
	Exchange         ActionID = "exchange"
	Escape           ActionID = "escape"
	Pass             ActionID = "pass" // Pass action, no operation done.
)

// PlayerAction is the message structure for player moves. Flows from
// WebSocket handler → orchestrator → engine. The Payload field uses `any` to support
// different action-specific data structures (see payloads.go).
type PlayerAction struct {
	ID          ActionID // Action to execute
	SourceIndex int      // Player index in GameState.Players slice
	Payload     any      // Action-specific data (TargetPayload, AccusePayload, or nil, see payloads.go)
	MainAction  ActionID // Original action being countered (preserved through counter chain)
}

// ActionKind distinguishes between turn actions and counter-actions. The engine
// uses this to determine valid moves to be sent to the frontend at each stage of the turn flow.
type ActionKind string

const (
	MainAction    ActionKind = "main"
	CounterAction ActionKind = "counter"
)
