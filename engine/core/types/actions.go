package types

type ActionID string

const (
	Income           ActionID = "income"
	ForeignAid       ActionID = "foreign_aid"
	Tax              ActionID = "tax"
	Business         ActionID = "business"
	Steal            ActionID = "steal"
	Assassinate      ActionID = "assassinate"
	Coup             ActionID = "coup"
	Exchange         ActionID = "exchange"
	Investigate      ActionID = "investigate"
	Accuse           ActionID = "accuse"
	Escape           ActionID = "escape"
	BlockSteal       ActionID = "block_steal"
	BlockTerrorist   ActionID = "block_terrorist"
	BlockPolice      ActionID = "block_investigate"
	BlockForeignAid  ActionID = "block_foreign_aid"
	TaxBusinessWoman ActionID = "business_tax"
	Pass             ActionID = "pass"
)

// PlayerAction is the engine's canonical command: "player X wants to do Y".
//
// It is created by the input layer (session/app) from the browser's raw command
// and passed into ApplyAction / ResumeAction.  Payload carries extra data only
// for actions that need a target or guess; it is nil for untargeted actions like
// Income or Tax.
type PlayerAction struct {
	Action     ActionID      // which action to execute (see ActionID constants above)
	ActorIndex int           // zero-based index into GameState.Players
	Payload    ActionPayload // nil for untargeted actions; TargetPayload or AccusePayload otherwise
}
