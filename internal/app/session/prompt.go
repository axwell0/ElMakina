//nolint:err113 // Prompt constructors return descriptive runtime errors; conversion to static sentinels is deferred.
package session

import (
	"fmt"
	"slices"
	"strings"

	coreactions "github.com/axwell0/elmakina/engine/core/actions"
	corestate "github.com/axwell0/elmakina/engine/core/state"
	coretypes "github.com/axwell0/elmakina/engine/core/types"
	runtimeturn "github.com/axwell0/elmakina/engine/runtime/turn"
)

type PromptKind string

const (
	PromptMainAction PromptKind = "main_action"
	PromptChallenge  PromptKind = "challenge"
	PromptCounter    PromptKind = "counter"
	PromptCardSelect PromptKind = "card_selection"
)

type Prompt struct {
	ID         string     `json:"id"`
	Kind       PromptKind `json:"kind"`
	Recipients []int      `json:"recipients"`
	Payload    any        `json:"payload"`
}

type ActionOption struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Description    string `json:"description,omitempty"`
	TargetRequired bool   `json:"targetRequired,omitempty"`
	GuessRequired  bool   `json:"guessRequired,omitempty"`
}

type MainActionPrompt struct {
	ActorIndex  int            `json:"actorIndex"`
	TurnNumber  int            `json:"turnNumber"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Actions     []ActionOption `json:"actions"`
}

type ChallengePrompt struct {
	ActorIndex         int            `json:"actorIndex"`
	Title              string         `json:"title"`
	Description        string         `json:"description,omitempty"`
	Action             string         `json:"action"`
	ActionLabel        string         `json:"actionLabel"`
	ClaimedRole        coretypes.Role `json:"claimedRole"`
	AllowedChallengers []int          `json:"allowedChallengers"`
}

type CounterPrompt struct {
	Title                  string                 `json:"title"`
	Description            string                 `json:"description,omitempty"`
	MainAction             string                 `json:"mainAction"`
	MainActionLabel        string                 `json:"mainActionLabel"`
	AllowedPlayers         []int                  `json:"allowedPlayers"`
	AllowedActions         []ActionOption         `json:"allowedActions"`
	AllowedActionsByPlayer map[int][]ActionOption `json:"allowedActionsByPlayer,omitempty"`
}

type CardSelectionPrompt struct {
	ActorIndex  int                      `json:"actorIndex"`
	Title       string                   `json:"title"`
	Description string                   `json:"description,omitempty"`
	ActionID    string                   `json:"actionId"`
	Count       int                      `json:"count"`
	Cards       []coreactions.CardOption `json:"cards"`
}

type StepResponse struct {
	Kind          coreactions.StepKind               `json:"kind"`
	CardSelection *coreactions.CardSelectionResponse `json:"card_selection,omitempty"`
}

const (
	mainActionTitle       = "Choose an action"
	mainActionDescription = "Select your move for this turn."

	challengeTitle       = "Challenge opportunity"
	challengeDescription = "Choose whether to challenge this claim."

	counterTitle       = "Counter opportunity"
	counterDescription = "Choose whether to counter this action."

	cardSelectionTitle       = "Select cards"
	cardSelectionDescription = "Choose the cards to keep."
)

func mainActionPrompt(gs *corestate.GameState, actor int) Prompt {
	allowed := coreactions.AllowedMainActions(gs, actor)
	options := make([]ActionOption, 0, len(allowed))
	for _, actionID := range allowed {
		options = append(options, actionOptionFromActionID(actionID))
	}
	return Prompt{
		Kind:       PromptMainAction,
		Recipients: []int{actor},
		Payload: MainActionPrompt{
			ActorIndex:  actor,
			TurnNumber:  gs.TurnNumber,
			Title:       mainActionTitle,
			Description: mainActionDescription,
			Actions:     options,
		},
	}
}

func challengePrompt(window runtimeturn.ChallengeWindow) (Prompt, error) {
	if window.Action == nil {
		return Prompt{}, fmt.Errorf("challenge prompt is missing action")
	}
	return Prompt{
		Kind:       PromptChallenge,
		Recipients: slices.Clone(window.AllowedChallengers),
		Payload: ChallengePrompt{
			ActorIndex:         window.ActorIndex,
			Title:              challengeTitle,
			Description:        challengeDescription,
			Action:             string(window.Action.Action),
			ActionLabel:        coreactions.ActionLabel(window.Action.Action),
			ClaimedRole:        window.ClaimedRole,
			AllowedChallengers: slices.Clone(window.AllowedChallengers),
		},
	}, nil
}

func counterPrompt(window runtimeturn.CounterWindow) (Prompt, error) {
	if window.MainAction == nil {
		return Prompt{}, fmt.Errorf("counter prompt is missing main action")
	}
	allowed := make([]ActionOption, 0, len(window.AllowedActions))
	for _, actionID := range window.AllowedActions {
		allowed = append(allowed, actionOptionFromActionID(actionID))
	}
	allowedByPlayer := make(map[int][]ActionOption, len(window.AllowedActionsByPlayer))
	for playerIndex, actionIDs := range window.AllowedActionsByPlayer {
		options := make([]ActionOption, 0, len(actionIDs))
		for _, actionID := range actionIDs {
			options = append(options, actionOptionFromActionID(actionID))
		}
		allowedByPlayer[playerIndex] = options
	}
	return Prompt{
		Kind:       PromptCounter,
		Recipients: slices.Clone(window.AllowedPlayers),
		Payload: CounterPrompt{
			Title:                  counterTitle,
			Description:            counterDescription,
			MainAction:             string(window.MainAction.Action),
			MainActionLabel:        coreactions.ActionLabel(window.MainAction.Action),
			AllowedPlayers:         slices.Clone(window.AllowedPlayers),
			AllowedActions:         allowed,
			AllowedActionsByPlayer: allowedByPlayer,
		},
	}, nil
}

func cardSelectionPrompt(req coreactions.StepRequest) (Prompt, error) {
	if req.Kind != coreactions.StepCardSelection {
		return Prompt{}, fmt.Errorf("unsupported step kind %q", req.Kind)
	}
	if req.CardSelection == nil {
		return Prompt{}, fmt.Errorf("card selection step is missing card options")
	}
	options := slices.Clone(req.CardSelection.Cards)
	title := cardSelectionTitle
	if strings.TrimSpace(req.Title) != "" {
		title = req.Title
	}
	description := cardSelectionDescription
	if strings.TrimSpace(req.Description) != "" {
		description = req.Description
	}
	return Prompt{
		Kind:       PromptCardSelect,
		Recipients: []int{req.ActorIndex},
		Payload: CardSelectionPrompt{
			ActorIndex:  req.ActorIndex,
			Title:       title,
			Description: description,
			ActionID:    string(req.Action),
			Count:       req.CardSelection.Count,
			Cards:       options,
		},
	}, nil
}

func actionOptionFromActionID(id coretypes.ActionID) ActionOption {
	option := ActionOption{
		ID: string(id),
	}
	if presentation, ok := coreactions.ActionPresentationByID(id); ok {
		option.Label = presentation.Label
		option.Description = presentation.Description
		switch presentation.Payload.(type) {
		case coretypes.TargetPayload:
			option.TargetRequired = true
		case coretypes.AccusePayload:
			option.TargetRequired = true
			option.GuessRequired = true
		}
	}
	return option
}
