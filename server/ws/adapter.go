package ws

import (
	"fmt"

	sessionapp "example.com/elmakina/app/session"
	corestate "example.com/elmakina/engine/core/state"
	coretypes "example.com/elmakina/engine/core/types"
	ports "example.com/elmakina/engine/ports"
	runtimeturn "example.com/elmakina/engine/runtime/turn"
	clientprojection "example.com/elmakina/projection/client"
)

// BuildTurnResultMessage converts the internal turn result into the wire format.
func BuildTurnResultMessage(session *sessionapp.GameSession, viewerIndex int, result *runtimeturn.TurnResult) (TurnResultMessage, error) {
	projected, err := clientprojection.BuildTurnResult(session, viewerIndex, result)
	if err != nil {
		return TurnResultMessage{}, err
	}

	message := TurnResultMessage{
		Type:             TypeTurnResult,
		RoomID:           projected.RoomID,
		TurnNumber:       projected.TurnNumber,
		Main:             actionMessageFromProjection(projected.Main),
		MainApplied:      projected.MainApplied,
		MainCanceled:     projected.MainCanceled,
		Logs:             append([]string(nil), projected.Logs...),
		PrivateLogs:      append([]string(nil), projected.PrivateLogs...),
		CounterResults:   make([]CounterResultMessage, 0, len(projected.CounterResults)),
		ChallengeResults: make([]ChallengeResultMessage, 0, len(projected.ChallengeResults)),
		Summary:          matchSummaryFromProjection(projected.Summary),
		Hand:             privateCardsFromProjection(projected.Hand),
	}
	for _, counter := range projected.CounterResults {
		message.CounterResults = append(message.CounterResults, counterResultFromProjection(counter))
	}
	for _, challenge := range projected.ChallengeResults {
		message.ChallengeResults = append(message.ChallengeResults, challengeResultFromProjection(challenge))
	}
	return message, nil
}

// BuildRoomSyncMessage converts the session snapshot into the per-player sync message.
func BuildRoomSyncMessage(session *sessionapp.GameSession, viewerIndex int) (RoomSyncMessage, error) {
	projected, err := clientprojection.BuildRoomSync(session, viewerIndex)
	if err != nil {
		return RoomSyncMessage{}, err
	}

	return RoomSyncMessage{
		Type:              TypeRoomSync,
		RoomID:            projected.RoomID,
		PlayerIndex:       projected.PlayerIndex,
		PlayerName:        projected.PlayerName,
		StatusTitle:       projected.StatusTitle,
		StatusDescription: projected.StatusDescription,
		Room:              roomViewFromProjection(projected.Room),
		Summary:           matchSummaryFromProjection(projected.Summary),
		Hand:              privateCardsFromProjection(projected.Hand),
	}, nil
}

// BuildGameOverMessage converts the end-of-match state into the browser payload.
func BuildGameOverMessage(session *sessionapp.GameSession, winnerIndex int) (GameOverMessage, error) {
	projected, err := clientprojection.BuildGameOver(session, winnerIndex)
	if err != nil {
		return GameOverMessage{}, err
	}
	return GameOverMessage{
		Type:        TypeGameOver,
		RoomID:      projected.RoomID,
		WinnerIndex: projected.WinnerIndex,
		WinnerName:  projected.WinnerName,
		Summary:     matchSummaryFromProjection(projected.Summary),
	}, nil
}

func DecodeSubmitAction(message SubmitActionMessage) (*coretypes.PlayerAction, error) {
	return decodeActionMessage(message.Action)
}

func DecodeSubmitChallenge(message SubmitChallengeMessage) ports.ChallengeDecision {
	decision := ports.ChallengeDecision{
		ChallengerIndex: message.Decision.ChallengerIndex,
		Pass:            message.Decision.Pass,
	}
	if message.Decision.ChallengerDiscardIndex != nil {
		decision.ChallengerDiscardIndex = *message.Decision.ChallengerDiscardIndex
	}
	if message.Decision.ActorDiscardIndex != nil {
		decision.ActorDiscardIndex = *message.Decision.ActorDiscardIndex
	}
	if message.Decision.ActorProvingCardIndex != nil {
		decision.ActorProvingCardIndex = *message.Decision.ActorProvingCardIndex
	}
	return decision
}

func DecodeSubmitCounter(message SubmitCounterMessage) (ports.CounterDecision, error) {
	decision := ports.CounterDecision{
		PlayerIndex: message.Decision.PlayerIndex,
		Pass:        message.Decision.Pass,
	}
	if message.Decision.Pass {
		return decision, nil
	}
	if message.Decision.Command == nil {
		return ports.CounterDecision{}, fmt.Errorf("counter command is required when pass is false")
	}
	command, err := decodeActionMessage(*message.Decision.Command)
	if err != nil {
		return ports.CounterDecision{}, err
	}
	decision.Command = command
	return decision, nil
}

func DecodeSubmitStep(step *corestate.StepRequest, message SubmitStepMessage) (corestate.StepResponse, error) {
	if step == nil {
		return corestate.StepResponse{}, fmt.Errorf("step request is nil")
	}

	response := corestate.StepResponse{Kind: step.Kind}
	switch step.Kind {
	case corestate.StepCardSelection:
		if message.Response.CardSelection == nil {
			return corestate.StepResponse{}, fmt.Errorf("card selection response is required")
		}
		response.CardSelection = &corestate.CardSelectionResponse{
			SelectedIndices: append([]int(nil), message.Response.CardSelection.SelectedIndices...),
		}
	default:
		return corestate.StepResponse{}, fmt.Errorf("unsupported step kind %q", step.Kind)
	}

	return response, nil
}

func decodeActionMessage(message ActionMessage) (*coretypes.PlayerAction, error) {
	if message.ID == "" {
		return nil, fmt.Errorf("action id is required")
	}
	if actionRequiresTarget(coretypes.ActionID(message.ID)) && message.Payload.TargetIndex == nil {
		return nil, fmt.Errorf("action %q requires a target", message.ID)
	}
	if actionRequiresGuess(coretypes.ActionID(message.ID)) && message.Payload.Guess == "" {
		return nil, fmt.Errorf("action %q requires a guess", message.ID)
	}

	action := &coretypes.PlayerAction{
		ID:         coretypes.ActionID(message.ID),
		ActorIndex: message.ActorIndex,
		MainAction: coretypes.ActionID(message.MainAction),
	}

	if message.Payload.TargetIndex != nil && message.Payload.Guess != "" {
		role, err := coretypes.ParseRole(message.Payload.Guess)
		if err != nil {
			return nil, err
		}
		action.Payload = coretypes.AccusePayload{
			TargetIndex: *message.Payload.TargetIndex,
			Guess:       role,
		}
		return action, nil
	}
	if message.Payload.TargetIndex != nil {
		action.Payload = coretypes.TargetPayload{TargetIndex: *message.Payload.TargetIndex}
		return action, nil
	}
	action.Payload = coretypes.NoPayload{}
	return action, nil
}

func actionRequiresTarget(id coretypes.ActionID) bool {
	switch id {
	case coretypes.Coup, coretypes.Investigate, coretypes.Assassinate, coretypes.Steal, coretypes.Accuse:
		return true
	default:
		return false
	}
}

func actionRequiresGuess(id coretypes.ActionID) bool {
	return id == coretypes.Accuse
}

func buildMatchSummaryMessage(session *sessionapp.GameSession) MatchSummaryMessage {
	return matchSummaryFromProjection(clientprojection.BuildMatchSummary(session))
}

func buildRoomViewMessage(session *sessionapp.GameSession) RoomViewMessage {
	return roomViewFromProjection(clientprojection.BuildRoomView(session))
}

func matchSummaryFromProjection(summary clientprojection.MatchSummary) MatchSummaryMessage {
	players := make([]PlayerSummaryMessage, 0, len(summary.Players))
	for _, player := range summary.Players {
		players = append(players, PlayerSummaryMessage{
			PlayerIndex: player.PlayerIndex,
			Name:        player.Name,
			Coins:       player.Coins,
			Cards:       player.Cards,
			Eliminated:  player.Eliminated,
		})
	}
	return MatchSummaryMessage{
		TurnNumber:         summary.TurnNumber,
		CurrentPlayerIndex: summary.CurrentPlayerIndex,
		Players:            players,
	}
}

func privateCardsFromProjection(cards []clientprojection.PrivateCard) []PrivateCardMessage {
	result := make([]PrivateCardMessage, 0, len(cards))
	for _, card := range cards {
		result = append(result, PrivateCardMessage{ID: card.ID, Role: card.Role})
	}
	return result
}

func roomViewFromProjection(room clientprojection.RoomView) RoomViewMessage {
	return RoomViewMessage{
		Phase:             room.Phase,
		LeaderPlayerIndex: room.LeaderPlayerIndex,
		PlayerCount:       room.PlayerCount,
		JoinedPlayers:     room.JoinedPlayers,
		ConnectedPlayers:  append([]int(nil), room.ConnectedPlayers...),
		ReadyPlayers:      append([]int(nil), room.ReadyPlayers...),
		CanStart:          room.CanStart,
		CanRestart:        room.CanRestart,
		CanDelete:         room.CanDelete,
	}
}

func actionMessageFromProjection(action *clientprojection.Action) *ActionMessage {
	if action == nil {
		return nil
	}
	return &ActionMessage{
		ID:         action.ID,
		ActorIndex: action.ActorIndex,
		MainAction: action.MainAction,
		Payload: ActionPayloadMessage{
			TargetIndex: action.Payload.TargetIndex,
			Guess:       action.Payload.Guess,
		},
	}
}

func challengeResultFromProjection(result clientprojection.ChallengeResult) ChallengeResultMessage {
	return ChallengeResultMessage{
		Success:         result.Success,
		ActionAllowed:   result.ActionAllowed,
		ActorIndex:      result.ActorIndex,
		ChallengerIndex: result.ChallengerIndex,
		ClaimedRole:     result.ClaimedRole,
		Outcome:         result.Outcome,
		Logs:            append([]string(nil), result.Logs...),
		LostCardRole:    result.LostCardRole,
		LostPlayerIndex: result.LostPlayerIndex,
	}
}

func counterResultFromProjection(result clientprojection.CounterResult) CounterResultMessage {
	message := CounterResultMessage{
		Action:   actionMessageFromProjection(result.Action),
		Applied:  result.Applied,
		Canceled: result.Canceled,
	}
	if result.Challenge != nil {
		challenge := challengeResultFromProjection(*result.Challenge)
		message.Challenge = &challenge
	}
	return message
}
