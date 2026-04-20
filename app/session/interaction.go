package session

type InteractionKind string

const (
	InteractionMainAction InteractionKind = "main_action"
	InteractionChallenge  InteractionKind = "challenge"
	InteractionCounter    InteractionKind = "counter"
	InteractionCardSelect InteractionKind = "card_selection"
)

type RecipientSet struct {
	Players []int
}

type InteractionState struct {
	ID         string
	Kind       InteractionKind
	Recipients RecipientSet
	Payload    any
}

type ActionOption struct {
	ID             string
	Label          string
	Description    string
	TargetRequired bool
	GuessRequired  bool
}

type MainActionInteraction struct {
	ActorIndex  int
	TurnNumber  int
	Title       string
	Description string
	Actions     []ActionOption
}

type ChallengeInteraction struct {
	ActorIndex         int
	Title              string
	Description        string
	Action             string
	ActionLabel        string
	ClaimedRole        string
	AllowedChallengers []int
}

type CounterInteraction struct {
	Title           string
	Description     string
	MainAction      string
	MainActionLabel string
	AllowedPlayers  []int
	AllowedActions  []ActionOption
}

type CardOption struct {
	Index int
	Label string
}

type CardSelectionInteraction struct {
	ActorIndex  int
	Title       string
	Description string
	ActionID    string
	Count       int
	Cards       []CardOption
}
