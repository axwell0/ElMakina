package ws

import (
	sessionapp "example.com/elmakina/app/session"
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
