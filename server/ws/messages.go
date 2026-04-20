package ws

// MessageType identifies the kind of websocket message being sent or received.
type MessageType string

const (
	TypeRoomSync         MessageType = "room.sync"
	TypePrompt           MessageType = "prompt"
	TypePromptMainAction MessageType = "prompt.main_action"
	TypePromptChallenge  MessageType = "prompt.challenge"
	TypePromptCounter    MessageType = "prompt.counter"
	TypePromptStep       MessageType = "prompt.step"
	TypeTurnResult       MessageType = "turn.result"
	TypeGameOver         MessageType = "game.over"

	TypeCommand         MessageType = "command"
	TypeSubmitAction    MessageType = "action.submit"
	TypeSubmitChallenge MessageType = "challenge.submit"
	TypeSubmitCounter   MessageType = "counter.submit"
	TypeSubmitStep      MessageType = "step.submit"
	TypeSubmitReady     MessageType = "room.ready"
	TypeSubmitStart     MessageType = "room.start"
)

// PromptEnvelope is the canonical live-prompt message family.
type PromptEnvelope struct {
	Type        MessageType          `json:"type"`
	Interaction InteractionWireState `json:"interaction"`
}

// InteractionWireState is the websocket-safe interaction shape.
type InteractionWireState struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Recipients []int          `json:"recipients"`
	Payload    map[string]any `json:"payload"`
}

// CommandEnvelopeMessage is the canonical inbound command message.
type CommandEnvelopeMessage struct {
	Type          MessageType     `json:"type"`
	InteractionID string          `json:"interactionId"`
	Kind          string          `json:"kind"`
	Payload       map[string]any  `json:"payload"`
}

// ActionOption is the client-facing description of a legal action choice.
type ActionOption struct {
	Label          string `json:"label"`
	Description    string `json:"description,omitempty"`
	ID             string `json:"id"`
	TargetRequired bool   `json:"targetRequired,omitempty"`
	GuessRequired  bool   `json:"guessRequired,omitempty"`
}

// ActionPayloadMessage mirrors the action payload sent from the browser.
type ActionPayloadMessage struct {
	TargetIndex *int   `json:"targetIndex,omitempty"`
	Guess       string `json:"guess,omitempty"`
}

// ActionMessage is the wire format for a player action.
type ActionMessage struct {
	ID         string               `json:"id"`
	ActorIndex int                  `json:"actorIndex"`
	MainAction string               `json:"mainAction,omitempty"`
	Payload    ActionPayloadMessage `json:"payload,omitempty"`
}

// PromptMainActionMessage is sent when the current player must choose a move.
type PromptMainActionMessage struct {
	Type        MessageType    `json:"type"`
	RoomID      string         `json:"roomId"`
	ActorIndex  int            `json:"actorIndex"`
	TurnNumber  int            `json:"turnNumber"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Actions     []ActionOption `json:"actions"`
}

// PromptChallengeMessage is sent to players who may challenge a claim.
type PromptChallengeMessage struct {
	Type               MessageType `json:"type"`
	RoomID             string      `json:"roomId"`
	ActorIndex         int         `json:"actorIndex"`
	Title              string      `json:"title"`
	Description        string      `json:"description,omitempty"`
	Action             string      `json:"action"`
	ActionLabel        string      `json:"actionLabel"`
	ClaimedRole        string      `json:"claimedRole"`
	AllowedChallengers []int       `json:"allowedChallengers"`
}

// PromptCounterMessage is sent to players who may block an action.
type PromptCounterMessage struct {
	Type            MessageType    `json:"type"`
	RoomID          string         `json:"roomId"`
	Title           string         `json:"title"`
	Description     string         `json:"description,omitempty"`
	MainAction      string         `json:"mainAction"`
	MainActionLabel string         `json:"mainActionLabel"`
	AllowedPlayers  []int          `json:"allowedPlayers"`
	AllowedActions  []ActionOption `json:"allowedActions"`
}

// StepDescriptor explains a follow-up selection that must be completed before
// a multi-step action can continue.
type StepDescriptor struct {
	ActionID string `json:"actionId"`
	Kind     string `json:"kind"`

	CardSelection *CardSelectionDescriptor `json:"cardSelection,omitempty"`
}

type CardSelectionDescriptor struct {
	Count int                 `json:"count"`
	Cards []CardOptionMessage `json:"cards,omitempty"`
}

type CardOptionMessage struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}

// PromptStepMessage is sent when the engine pauses for a card or option pick.
type PromptStepMessage struct {
	Type        MessageType    `json:"type"`
	RoomID      string         `json:"roomId"`
	ActorIndex  int            `json:"actorIndex"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Step        StepDescriptor `json:"step"`
}

// MatchSummaryMessage is the public match snapshot shared with every player.
type MatchSummaryMessage struct {
	TurnNumber         int                    `json:"turnNumber"`
	CurrentPlayerIndex int                    `json:"currentPlayerIndex"`
	Players            []PlayerSummaryMessage `json:"players"`
}

// PlayerSummaryMessage is the public row shown for each player in the room.
type PlayerSummaryMessage struct {
	PlayerIndex int    `json:"playerIndex"`
	Name        string `json:"name"`
	Coins       int    `json:"coins"`
	Cards       int    `json:"cards"`
	Eliminated  bool   `json:"eliminated"`
}

// PrivateCardMessage reveals a card only to its owner.
type PrivateCardMessage struct {
	ID   int    `json:"id"`
	Role string `json:"role"`
}

// RoomViewMessage is the public room-state payload displayed in the UI.
type RoomViewMessage struct {
	Phase             string `json:"phase"`
	LeaderPlayerIndex int    `json:"leaderPlayerIndex"`
	PlayerCount       int    `json:"playerCount"`
	JoinedPlayers     int    `json:"joinedPlayers"`
	ConnectedPlayers  []int  `json:"connectedPlayers"`
	ReadyPlayers      []int  `json:"readyPlayers"`
	CanStart          bool   `json:"canStart"`
	CanRestart        bool   `json:"canRestart"`
	CanDelete         bool   `json:"canDelete"`
}

// RoomSyncMessage is the full client snapshot sent whenever room state changes.
type RoomSyncMessage struct {
	Type              MessageType          `json:"type"`
	RoomID            string               `json:"roomId"`
	PlayerIndex       int                  `json:"playerIndex"`
	PlayerName        string               `json:"playerName"`
	StatusTitle       string               `json:"statusTitle,omitempty"`
	StatusDescription string               `json:"statusDescription,omitempty"`
	Room              RoomViewMessage      `json:"room"`
	Summary           MatchSummaryMessage  `json:"summary"`
	Hand              []PrivateCardMessage `json:"hand,omitempty"`
}

// ChallengeResultMessage reports the outcome of a challenge to the UI.
type ChallengeResultMessage struct {
	Success         bool     `json:"success"`
	ActionAllowed   bool     `json:"actionAllowed"`
	ActorIndex      int      `json:"actorIndex"`
	ChallengerIndex int      `json:"challengerIndex"`
	ClaimedRole     string   `json:"claimedRole"`
	Outcome         string   `json:"outcome"`
	Logs            []string `json:"logs,omitempty"`
	LostCardRole    string   `json:"lostCardRole,omitempty"`
	LostPlayerIndex *int     `json:"lostPlayerIndex,omitempty"`
}

// CounterResultMessage reports the outcome of a counter action.
type CounterResultMessage struct {
	Action    *ActionMessage          `json:"action,omitempty"`
	Applied   bool                    `json:"applied"`
	Canceled  bool                    `json:"canceled"`
	Challenge *ChallengeResultMessage `json:"challenge,omitempty"`
}

// TurnResultMessage is the full payload sent when a turn finishes.
type TurnResultMessage struct {
	Type             MessageType              `json:"type"`
	RoomID           string                   `json:"roomId"`
	TurnNumber       int                      `json:"turnNumber"`
	Main             *ActionMessage           `json:"main,omitempty"`
	MainApplied      bool                     `json:"mainApplied"`
	MainCanceled     bool                     `json:"mainCanceled"`
	Logs             []string                 `json:"logs,omitempty"`
	PrivateLogs      []string                 `json:"privateLogs,omitempty"`
	CounterResults   []CounterResultMessage   `json:"counterResults,omitempty"`
	ChallengeResults []ChallengeResultMessage `json:"challengeResults,omitempty"`
	Summary          MatchSummaryMessage      `json:"summary"`
	Hand             []PrivateCardMessage     `json:"hand,omitempty"`
}

// GameOverMessage tells clients who won the match.
type GameOverMessage struct {
	Type        MessageType         `json:"type"`
	RoomID      string              `json:"roomId"`
	WinnerIndex int                 `json:"winnerIndex"`
	WinnerName  string              `json:"winnerName"`
	Summary     MatchSummaryMessage `json:"summary"`
}

// SubmitActionMessage is the browser payload for a main action submission.
type SubmitActionMessage struct {
	Type   MessageType   `json:"type"`
	RoomID string        `json:"roomId"`
	Action ActionMessage `json:"action"`
}

// ChallengeDecisionMessage is the browser payload for a challenge response.
type ChallengeDecisionMessage struct {
	ChallengerIndex        int  `json:"challengerIndex"`
	ChallengerDiscardIndex *int `json:"challengerDiscardIndex,omitempty"`
	ActorDiscardIndex      *int `json:"actorDiscardIndex,omitempty"`
	ActorProvingCardIndex  *int `json:"actorProvingCardIndex,omitempty"`
	Pass                   bool `json:"pass"`
}

// SubmitChallengeMessage wraps a challenge decision with the room metadata.
type SubmitChallengeMessage struct {
	Type     MessageType              `json:"type"`
	RoomID   string                   `json:"roomId"`
	Decision ChallengeDecisionMessage `json:"decision"`
}

// CounterDecisionMessage is the browser payload for a counter response.
type CounterDecisionMessage struct {
	PlayerIndex int            `json:"playerIndex"`
	Pass        bool           `json:"pass"`
	Command     *ActionMessage `json:"command,omitempty"`
}

// SubmitCounterMessage wraps a counter decision with the room metadata.
type SubmitCounterMessage struct {
	Type     MessageType            `json:"type"`
	RoomID   string                 `json:"roomId"`
	Decision CounterDecisionMessage `json:"decision"`
}

// StepResponseMessage carries the typed response for a step prompt.
type StepResponseMessage struct {
	Kind string `json:"kind"`

	CardSelection *CardSelectionResponseMessage `json:"cardSelection,omitempty"`
}

type CardSelectionResponseMessage struct {
	SelectedIndices []int `json:"selectedIndices,omitempty"`
}

// SubmitStepMessage wraps a step response with the room metadata.
type SubmitStepMessage struct {
	Type     MessageType         `json:"type"`
	RoomID   string              `json:"roomId"`
	Response StepResponseMessage `json:"response"`
}

// SubmitReadyMessage marks a player as ready in the lobby.
type SubmitReadyMessage struct {
	Type   MessageType `json:"type"`
	RoomID string      `json:"roomId"`
}

// SubmitStartMessage asks the leader to start the room.
type SubmitStartMessage struct {
	Type   MessageType `json:"type"`
	RoomID string      `json:"roomId"`
}
