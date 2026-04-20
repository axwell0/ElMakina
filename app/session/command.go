package session

type CommandKind string

const (
	CommandMainAction        CommandKind = "main_action"
	CommandChallengeDecision CommandKind = "challenge_decision"
	CommandCounterDecision   CommandKind = "counter_decision"
	CommandCardSelection     CommandKind = "card_selection"
)

type CommandEnvelope struct {
	InteractionID string      `json:"interactionId"`
	Kind          CommandKind `json:"kind"`
	SessionID     string      `json:"sessionId"`
	Payload       any         `json:"payload"`
}

type MainActionCommand struct {
	ActionID    string `json:"actionId"`
	ActorIndex  int    `json:"actorIndex"`
	MainAction  string `json:"mainAction,omitempty"`
	TargetIndex *int   `json:"targetIndex,omitempty"`
	Guess       string `json:"guess,omitempty"`
}

type ChallengeDecisionCommand struct {
	ChallengerIndex        int  `json:"challengerIndex"`
	ChallengerDiscardIndex *int `json:"challengerDiscardIndex,omitempty"`
	ActorDiscardIndex      *int `json:"actorDiscardIndex,omitempty"`
	ActorProvingCardIndex  *int `json:"actorProvingCardIndex,omitempty"`
	Pass                   bool `json:"pass"`
}

type CounterDecisionCommand struct {
	PlayerIndex int                `json:"playerIndex"`
	Pass        bool               `json:"pass"`
	Command     *MainActionCommand `json:"command,omitempty"`
}

type CardSelectionCommand struct {
	SelectedIndices []int `json:"selectedIndices"`
}
