package session

import (
	"fmt"
	"strings"
	"unicode"

	coreactions "example.com/elmakina/engine/core/actions"
	corestate "example.com/elmakina/engine/core/state"
	coretypes "example.com/elmakina/engine/core/types"
)

func interactionFromOpenInput(session *GameSession, input OpenInput) (InteractionState, error) {
	if session == nil {
		return InteractionState{}, fmt.Errorf("session is nil")
	}

	state := InteractionState{
		ID: fmt.Sprintf("%s:%s:%d", session.ID, input.Kind, input.Revision),
	}

	switch input.Kind {
	case InputMainAction:
		allowed := coreactions.AllowedMainActions(&session.State, input.Actor)
		options := make([]ActionOption, 0, len(allowed))
		for _, actionID := range allowed {
			options = append(options, actionOptionFromActionID(actionID))
		}
		state.Kind = InteractionMainAction
		state.Recipients = RecipientSet{Players: []int{input.Actor}}
		state.Payload = MainActionInteraction{
			ActorIndex:  input.Actor,
			TurnNumber:  session.State.TurnNumber,
			Title:       "Choose an action",
			Description: "Select your move for this turn.",
			Actions:     options,
		}
	case InputChallenge:
		if input.Challenge == nil || input.Challenge.Action == nil {
			return InteractionState{}, fmt.Errorf("challenge prompt is missing action")
		}
		state.Kind = InteractionChallenge
		state.Recipients = RecipientSet{Players: append([]int(nil), input.Challenge.AllowedChallengers...)}
		state.Payload = ChallengeInteraction{
			ActorIndex:         input.Challenge.ActorIndex,
			Title:              "Challenge opportunity",
			Description:        "Choose whether to challenge this claim.",
			Action:             string(input.Challenge.Action.ID),
			ActionLabel:        actionLabel(input.Challenge.Action.ID),
			ClaimedRole:        string(input.Challenge.ClaimedRole),
			AllowedChallengers: append([]int(nil), input.Challenge.AllowedChallengers...),
		}
	case InputCounter:
		if input.Counter == nil || input.Counter.MainAction == nil {
			return InteractionState{}, fmt.Errorf("counter prompt is missing main action")
		}
		allowed := make([]ActionOption, 0, len(input.Counter.AllowedActions))
		for _, actionID := range input.Counter.AllowedActions {
			allowed = append(allowed, actionOptionFromActionID(actionID))
		}
		state.Kind = InteractionCounter
		state.Recipients = RecipientSet{Players: append([]int(nil), input.Counter.AllowedPlayers...)}
		state.Payload = CounterInteraction{
			Title:           "Counter opportunity",
			Description:     "Choose whether to counter this action.",
			MainAction:      string(input.Counter.MainAction.ID),
			MainActionLabel: actionLabel(input.Counter.MainAction.ID),
			AllowedPlayers:  append([]int(nil), input.Counter.AllowedPlayers...),
			AllowedActions:  allowed,
		}
	case InputStep:
		if input.Step == nil {
			return InteractionState{}, fmt.Errorf("step prompt is missing step request")
		}
		if input.Step.Kind != corestate.StepCardSelection {
			return InteractionState{}, fmt.Errorf("unsupported step kind %q", input.Step.Kind)
		}
		if input.Step.CardSelection == nil {
			return InteractionState{}, fmt.Errorf("card selection step is missing card options")
		}
		options := make([]CardOption, 0, len(input.Step.CardSelection.Cards))
		for _, card := range input.Step.CardSelection.Cards {
			options = append(options, CardOption{Index: card.Index, Label: card.Label})
		}
		title := "Select cards"
		if strings.TrimSpace(input.Step.Title) != "" {
			title = input.Step.Title
		}
		description := "Choose the cards to keep."
		if strings.TrimSpace(input.Step.Description) != "" {
			description = input.Step.Description
		}
		state.Kind = InteractionCardSelect
		state.Recipients = RecipientSet{Players: []int{input.Step.Actor}}
		state.Payload = CardSelectionInteraction{
			ActorIndex:  input.Step.Actor,
			Title:       title,
			Description: description,
			ActionID:    string(input.Step.ActionID),
			Count:       input.Step.CardSelection.Count,
			Cards:       options,
		}
	default:
		return InteractionState{}, fmt.Errorf("unsupported input kind %q", input.Kind)
	}

	return state, nil
}

func actionOptionFromActionID(id coretypes.ActionID) ActionOption {
	option := ActionOption{
		ID:    string(id),
		Label: actionLabel(id),
	}
	switch id {
	case coretypes.Coup, coretypes.Investigate, coretypes.Assassinate, coretypes.Steal, coretypes.Accuse:
		option.TargetRequired = true
	}
	if id == coretypes.Accuse {
		option.GuessRequired = true
	}
	return option
}

func actionLabel(id coretypes.ActionID) string {
	switch id {
	case coretypes.Income:
		return "Income"
	case coretypes.Tax:
		return "Tax"
	case coretypes.Exchange:
		return "Exchange"
	case coretypes.Coup:
		return "Coup"
	case coretypes.Investigate:
		return "Investigate"
	case coretypes.Assassinate:
		return "Assassinate"
	case coretypes.Steal:
		return "Steal"
	case coretypes.Accuse:
		return "Accuse"
	case coretypes.BlockSteal:
		return "Block Steal"
	case coretypes.BlockForeignAid:
		return "Block Foreign Aid"
	case coretypes.BlockPolice:
		return "Block Investigate"
	case coretypes.BlockTerrorist:
		return "Block Terrorist"
	default:
		text := strings.ReplaceAll(string(id), "_", " ")
		return titleCaseWords(text)
	}
}

func titleCaseWords(text string) string {
	runes := []rune(strings.ToLower(text))
	capNext := true
	for i, r := range runes {
		if unicode.IsSpace(r) || r == '-' {
			capNext = true
			continue
		}
		if capNext {
			runes[i] = unicode.ToUpper(r)
			capNext = false
		}
	}
	return string(runes)
}
