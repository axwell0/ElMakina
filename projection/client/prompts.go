package client

// Deprecated: live prompt projection is migrating to app/session InteractionState.

import (
	"fmt"

	sessionapp "example.com/elmakina/app/session"
	coreactions "example.com/elmakina/engine/core/actions"
	corestate "example.com/elmakina/engine/core/state"
	coretypes "example.com/elmakina/engine/core/types"
)

type ActionOption struct {
	Label          string
	Description    string
	ID             string
	TargetRequired bool
	GuessRequired  bool
}

type PromptMainAction struct {
	RoomID      string
	ActorIndex  int
	TurnNumber  int
	Title       string
	Description string
	Actions     []ActionOption
}

type PromptChallenge struct {
	RoomID             string
	ActorIndex         int
	Title              string
	Description        string
	Action             string
	ActionLabel        string
	ClaimedRole        string
	AllowedChallengers []int
}

type PromptCounter struct {
	RoomID          string
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

type CardSelection struct {
	Count int
	Cards []CardOption
}

type StepDescriptor struct {
	ActionID      string
	Kind          string
	CardSelection *CardSelection
}

type PromptStep struct {
	RoomID      string
	ActorIndex  int
	Title       string
	Description string
	Step        StepDescriptor
}

func BuildPrompt(session *sessionapp.GameSession) (any, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}
	openInput := session.OpenInput()
	if openInput == nil {
		return nil, fmt.Errorf("session has no open input")
	}

	switch openInput.Kind {
	case sessionapp.InputMainAction:
		return buildMainActionPrompt(session, openInput), nil
	case sessionapp.InputChallenge:
		return buildChallengePrompt(session, openInput)
	case sessionapp.InputCounter:
		return buildCounterPrompt(session, openInput)
	case sessionapp.InputStep:
		return buildStepPrompt(session, openInput)
	default:
		return nil, fmt.Errorf("unsupported input kind %q", openInput.Kind)
	}
}

func buildMainActionPrompt(session *sessionapp.GameSession, input *sessionapp.OpenInput) PromptMainAction {
	allowed := coreactions.AllowedMainActions(&session.State, input.Actor)
	title, description := MainActionPromptCopy()

	return PromptMainAction{
		RoomID:      session.ID,
		ActorIndex:  input.Actor,
		TurnNumber:  session.State.TurnNumber,
		Title:       title,
		Description: description,
		Actions:     buildActionOptions(allowed),
	}
}

func buildChallengePrompt(session *sessionapp.GameSession, input *sessionapp.OpenInput) (PromptChallenge, error) {
	if input.Challenge == nil || input.Challenge.Action == nil {
		return PromptChallenge{}, fmt.Errorf("challenge prompt is missing action")
	}
	title, description := ChallengePromptCopy(string(input.Challenge.Action.ID), string(input.Challenge.ClaimedRole))

	return PromptChallenge{
		RoomID:             session.ID,
		ActorIndex:         input.Challenge.ActorIndex,
		Title:              title,
		Description:        description,
		Action:             string(input.Challenge.Action.ID),
		ActionLabel:        ActionLabelByID(input.Challenge.Action.ID),
		ClaimedRole:        string(input.Challenge.ClaimedRole),
		AllowedChallengers: append([]int(nil), input.Challenge.AllowedChallengers...),
	}, nil
}

func buildCounterPrompt(session *sessionapp.GameSession, input *sessionapp.OpenInput) (PromptCounter, error) {
	if input.Counter == nil || input.Counter.MainAction == nil {
		return PromptCounter{}, fmt.Errorf("counter prompt is missing main action")
	}
	title, description := CounterPromptCopy(string(input.Counter.MainAction.ID))

	return PromptCounter{
		RoomID:          session.ID,
		Title:           title,
		Description:     description,
		MainAction:      string(input.Counter.MainAction.ID),
		MainActionLabel: ActionLabelByID(input.Counter.MainAction.ID),
		AllowedPlayers:  append([]int(nil), input.Counter.AllowedPlayers...),
		AllowedActions:  buildActionOptions(input.Counter.AllowedActions),
	}, nil
}

func buildStepPrompt(session *sessionapp.GameSession, input *sessionapp.OpenInput) (PromptStep, error) {
	if input.Step == nil {
		return PromptStep{}, fmt.Errorf("step prompt is missing step request")
	}
	title, description := StepPromptCopy(*input.Step)
	if input.Step.Title != "" {
		title = input.Step.Title
	}
	if input.Step.Description != "" {
		description = input.Step.Description
	}

	return PromptStep{
		RoomID:      session.ID,
		ActorIndex:  input.Step.Actor,
		Title:       title,
		Description: description,
		Step: StepDescriptor{
			ActionID:      string(input.Step.ActionID),
			Kind:          string(input.Step.Kind),
			CardSelection: cardSelectionDescriptor(input.Step.CardSelection),
		},
	}, nil
}

func cardSelectionDescriptor(step *corestate.CardSelectionStep) *CardSelection {
	if step == nil {
		return nil
	}
	cards := make([]CardOption, 0, len(step.Cards))
	for _, card := range step.Cards {
		cards = append(cards, CardOption{Index: card.Index, Label: card.Label})
	}
	return &CardSelection{Count: step.Count, Cards: cards}
}

func buildActionOptions(ids []coretypes.ActionID) []ActionOption {
	options := make([]ActionOption, 0, len(ids))
	for _, id := range ids {
		presentation, ok := ActionCopyByID(id)
		if !ok {
			presentation = ActionPresentation{Label: FallbackActionLabel(string(id))}
		}
		option := ActionOption{
			ID:          string(id),
			Label:       presentation.Label,
			Description: presentation.Description,
		}
		switch id {
		case coretypes.Coup, coretypes.Investigate, coretypes.Assassinate, coretypes.Steal:
			option.TargetRequired = true
		case coretypes.Accuse:
			option.TargetRequired = true
			option.GuessRequired = true
		}
		options = append(options, option)
	}
	return options
}
