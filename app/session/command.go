package session

type CommandKind string

const (
	CommandMainAction        CommandKind = "main_action"
	CommandChallengeDecision CommandKind = "challenge_decision"
	CommandCounterDecision   CommandKind = "counter_decision"
	CommandCardSelection     CommandKind = "card_selection"
)

type CommandEnvelope struct {
	InteractionID string
	Kind          CommandKind
	SessionID     string
	Payload       any
}

type MainActionCommand struct {
	ActionID    string
	ActorIndex  int
	TargetIndex *int
	Guess       string
}

type ChallengeDecisionCommand struct {
	ChallengerIndex        int
	ChallengerDiscardIndex *int
	ActorDiscardIndex      *int
	ActorProvingCardIndex  *int
	Pass                   bool
}

type CounterDecisionCommand struct {
	PlayerIndex int
	Pass        bool
	Command     *MainActionCommand
}

type CardSelectionCommand struct {
	SelectedIndices []int
}
