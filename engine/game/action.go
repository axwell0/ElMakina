package game

// Action represents one of the player moves in El Makina.
// Each action has a cost, may require claiming a role, and may target another player.
// Actions are defined here as constants and their behaviors are registered in engine/rules/catalog.go.
type Action string

const (
	Income           Action = "income"
	ForeignAid       Action = "foreign_aid"
	Tax              Action = "tax"
	Business         Action = "business"
	Steal            Action = "steal"
	Assassinate      Action = "assassinate"
	Coup             Action = "coup"
	Exchange         Action = "exchange"
	Investigate      Action = "investigate"
	Accuse           Action = "accuse"
	Escape           Action = "escape"
	BlockSteal       Action = "block_steal"
	BlockTerrorist   Action = "block_terrorist"
	BlockPolice      Action = "block_investigate"
	BlockForeignAid  Action = "block_foreign_aid"
	TaxBusinessWoman Action = "business_tax"
	Pass             Action = "pass"
)

// Command is the engine's command: "player X wants to do Y".
//
// This is the lowest-level command representation in the codebase.
// It is created by the protocol/parse layer from the player's raw HTTP/WebSocket input
// and passed into the turn/rules system via SubmitTurnInput.
// TargetIndex and Guess carry extra data only for actions that need them;
// they are empty for untargeted actions like Income or Tax.
type Command struct {
	Action      Action `json:"action"`                // which action to execute (one of the Action constants above)
	ActorIndex  int    `json:"actorIndex"`            // zero-based index into the Players slice in game.State
	TargetIndex *int   `json:"targetIndex,omitempty"` // set for targeted actions.
	Guess       Role   `json:"guess,omitempty"`       // only set for Accuse
}
