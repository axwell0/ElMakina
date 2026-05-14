package session

type CommandKind string

const (
	CommandMainAction        CommandKind = "main_action"
	CommandChallengeDecision CommandKind = "challenge_decision"
	CommandCounterDecision   CommandKind = "counter_decision"
	CommandCardSelection     CommandKind = "card_selection"
)

type CommandPayload interface {
	CommandKind() CommandKind
}

type CommandEnvelope struct {
	PromptID  string         `json:"promptId"`
	Kind      CommandKind    `json:"kind"`
	SessionID string         `json:"sessionId"`
	Payload   CommandPayload `json:"payload"`
}

type MainActionCommand struct {
	Action      string `json:"action"`
	ActorIndex  int    `json:"actorIndex"`
	TargetIndex *int   `json:"targetIndex,omitempty"`
	Guess       string `json:"guess,omitempty"`
}

func (MainActionCommand) CommandKind() CommandKind { return CommandMainAction }

type ChallengeDecisionCommand struct {
	ChallengerIndex        int  `json:"challengerIndex"`
	ChallengerDiscardIndex *int `json:"challengerDiscardIndex,omitempty"`
	ActorDiscardIndex      *int `json:"actorDiscardIndex,omitempty"`
	ActorProvingCardIndex  *int `json:"actorProvingCardIndex,omitempty"`
	Pass                   bool `json:"pass"`
}

func (ChallengeDecisionCommand) CommandKind() CommandKind { return CommandChallengeDecision }

type CounterDecisionCommand struct {
	PlayerIndex int                   `json:"playerIndex"`
	Pass        bool                  `json:"pass"`
	Command     *CounterActionCommand `json:"command,omitempty"`
}

func (CounterDecisionCommand) CommandKind() CommandKind { return CommandCounterDecision }

type CounterActionCommand struct {
	Action     string `json:"action"`
	ActorIndex int    `json:"actorIndex"`
}

type CardSelectionCommand struct {
	SelectedIndices []int `json:"selectedIndices"`
}

func (CardSelectionCommand) CommandKind() CommandKind { return CommandCardSelection }
