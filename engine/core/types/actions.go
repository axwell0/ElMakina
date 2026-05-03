package types

// Action is the custom type of every allowed action in the game.
type Action string

const (
	Business         Action = "business"
	Exchange         Action = "exchange"
	Escape           Action = "escape"
	Income           Action = "income"
	ForeignAid       Action = "foreign_aid"
	Coup             Action = "coup"
	Tax              Action = "tax"
	Investigate      Action = "investigate"
	Accuse           Action = "accuse"
	Assassinate      Action = "assassinate"
	Steal            Action = "steal"
	BlockSteal       Action = "block_steal"
	BlockTerrorist   Action = "block_terrorist"
	BlockPolice      Action = "block_investigate"
	BlockForeignAid  Action = "block_foreign_aid"
	TaxBusinessWoman Action = "business_tax"
	Pass             Action = "pass" // Pass means "do nothing."
)

// Actions Represents all the actions available in the game engine.
var Actions = []Action{
	Business,
	Exchange,
	Escape,
	Income,
	ForeignAid,
	Coup,
	Tax,
	Investigate,
	Accuse,
	Assassinate,
	Steal,
	BlockSteal,
	BlockTerrorist,
	BlockPolice,
	BlockForeignAid,
	TaxBusinessWoman,
	Pass,
}

type PlayerAction struct {
	Action     Action        // Action to execute.
	ActorIndex int           // Player index in GameState.Players.
	Payload    ActionPayload // Action-specific data, if any.
}
