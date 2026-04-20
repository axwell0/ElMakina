package session

type InteractionKind string

const (
	InteractionMainAction InteractionKind = "main_action"
	InteractionChallenge  InteractionKind = "challenge"
	InteractionCounter    InteractionKind = "counter"
	InteractionCardSelect InteractionKind = "card_selection"
)

type RecipientSet struct {
	Players []int `json:"players"`
}

type InteractionState struct {
	ID         string          `json:"id"`
	Kind       InteractionKind `json:"kind"`
	Recipients RecipientSet    `json:"recipients"`
	Payload    any             `json:"payload"`
}

type ActionOption struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Description    string `json:"description,omitempty"`
	TargetRequired bool   `json:"targetRequired,omitempty"`
	GuessRequired  bool   `json:"guessRequired,omitempty"`
}

type MainActionInteraction struct {
	ActorIndex  int            `json:"actorIndex"`
	TurnNumber  int            `json:"turnNumber"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Actions     []ActionOption `json:"actions"`
}

type ChallengeInteraction struct {
	ActorIndex         int    `json:"actorIndex"`
	Title              string `json:"title"`
	Description        string `json:"description,omitempty"`
	Action             string `json:"action"`
	ActionLabel        string `json:"actionLabel"`
	ClaimedRole        string `json:"claimedRole"`
	AllowedChallengers []int  `json:"allowedChallengers"`
}

type CounterInteraction struct {
	Title           string         `json:"title"`
	Description     string         `json:"description,omitempty"`
	MainAction      string         `json:"mainAction"`
	MainActionLabel string         `json:"mainActionLabel"`
	AllowedPlayers  []int          `json:"allowedPlayers"`
	AllowedActions  []ActionOption `json:"allowedActions"`
}

type CardOption struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}

type CardSelectionInteraction struct {
	ActorIndex  int          `json:"actorIndex"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	ActionID    string       `json:"actionId"`
	Count       int          `json:"count"`
	Cards       []CardOption `json:"cards"`
}
