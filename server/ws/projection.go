// projection.go translates internal engine types into client-facing JSON messages.
//
// # Projection Pattern
//
// The engine works with rich internal types (GameState, TurnResult, Lobby).
// The browser works with flat JSON objects. This file is the translation layer —
// it "projects" internal state into the API shape each client sees.
//
// Crucially, different players see different data. Each player knows their own cards
// (Hand), but not other players' cards. So BuildTurnResultMessage is called once per
// player, producing a slightly different message for each (PrivateLogs differs per player,
// Hand always shows the recipient's own cards).
//
// Think of it like a sports broadcast: everyone sees the public score, but each player
// in a card game only sees their own hand.
//
//nolint:err113,gocritic // Projection copy text and response shaping are legacy and will be normalized later.
package ws

import (
	"fmt"
	"slices"

	corestate "github.com/axwell0/elmakina/engine/core/state"
	coretypes "github.com/axwell0/elmakina/engine/core/types"
	turnmodel "github.com/axwell0/elmakina/engine/turn"
	api "github.com/axwell0/elmakina/internal/api"
	sessionapp "github.com/axwell0/elmakina/internal/app/session"
)

// BuildRoomSyncMessage produces the full room snapshot for one specific player.
// It is called both on initial WebSocket connect and after any state change
// that all clients need to see (join, ready, start, turn complete, disconnect).
// viewerIndex determines which player's private hand is included in the response.
func BuildRoomSyncMessage(lobby *sessionapp.Lobby, viewerIndex int, connectedPlayers []int) (api.RoomSyncView, error) {
	state := lobby.Game().Snapshot()
	if viewerIndex < 0 || viewerIndex >= len(state.players) {
		return api.RoomSyncView{}, fmt.Errorf("player index %d out of range", viewerIndex)
	}

	player := state.players[viewerIndex]
	summary := buildMatchSummaryMessage(lobby)
	room := buildRoomViewMessage(lobby, connectedPlayers)
	statusTitle, statusDescription := roomStatusCopy(lobby, summary, room, viewerIndex)

	return api.RoomSyncView{
		Type:              api.TypeRoomSync,
		RoomID:            lobby.ID(),
		PlayerIndex:       viewerIndex,
		PlayerName:        player.Name,
		StatusTitle:       statusTitle,
		StatusDescription: statusDescription,
		Room:              room,
		Summary:           summary,
		Hand:              privateCardMessages(player.Hand),
	}, nil
}

func BuildGameOverMessage(lobby *sessionapp.Lobby, winnerIndex int) (api.GameOverView, error) {
	state := lobby.Game().Snapshot()
	if winnerIndex < 0 || winnerIndex >= len(state.players) {
		return api.GameOverView{}, fmt.Errorf("winner index %d out of range", winnerIndex)
	}

	return api.GameOverView{
		Type:        api.TypeGameOver,
		RoomID:      lobby.ID(),
		WinnerIndex: winnerIndex,
		WinnerName:  state.players[winnerIndex].Name,
		Summary:     buildMatchSummary(state),
	}, nil
}

func BuildTurnResultMessage(lobby *sessionapp.Lobby, viewerIndex int, result *turnmodel.TurnResult) (api.TurnResultView, error) {
	if result == nil {
		return api.TurnResultView{}, fmt.Errorf("turn result is nil")
	}
	state := lobby.Game().Snapshot()
	if viewerIndex < 0 || viewerIndex >= len(state.players) {
		return api.TurnResultView{}, fmt.Errorf("player index %d out of range", viewerIndex)
	}

	message := api.TurnResultView{
		Type:             api.TypeTurnResult,
		RoomID:           lobby.ID(),
		TurnNumber:       state.TurnNumber,
		Main:             actionMessageFromCore(result.Main),
		MainApplied:      result.MainApplied,
		MainCanceled:     result.MainCanceled,
		Logs:             slices.Clone(result.Logs),
		PrivateLogs:      slices.Clone(result.PrivateLogs[viewerIndex]), // viewer-specific logs
		CounterResults:   make([]api.CounterResultView, 0, len(result.CounterSummaries)),
		ChallengeResults: make([]api.ChallengeResultView, 0, len(result.ChallengeSummaries)),
		Summary:          buildMatchSummary(state),
		Hand:             privateCardMessages(state.players[viewerIndex].Hand),
	}

	for _, counter := range result.CounterSummaries {
		message.CounterResults = append(message.CounterResults, counterResultMessage(counter))
	}
	for _, challenge := range result.ChallengeSummaries {
		message.ChallengeResults = append(message.ChallengeResults, challengeResultMessage(challenge))
	}

	return message, nil
}

func buildMatchSummaryMessage(lobby *sessionapp.Lobby) api.MatchSnapshot {
	return buildMatchSummary(lobby.Game().Snapshot())
}

// buildMatchSummary converts internal GameState into the public-facing summary
// that all clients can see: turn number, current player, and each player's
// public info (name, coin count, card count, eliminated status).
// Note: card *count* is public; the card *roles* are private (only in Hand).
func buildMatchSummary(state corestate.GameState) api.MatchSnapshot {
	players := make([]api.PlayerSnapshot, 0, len(state.players))
	for i, player := range state.players {
		players = append(players, api.PlayerSnapshot{
			PlayerIndex: i,
			Name:        player.Name,
			Coins:       player.Coins,
			Cards:       len(player.Hand),
			Eliminated:  len(player.Hand) == 0,
		})
	}

	return api.MatchSnapshot{
		TurnNumber:         state.TurnNumber,
		CurrentPlayerIndex: state.CurrentPlayerIndex,
		Players:            players,
	}
}

func buildRoomViewMessage(lobby *sessionapp.Lobby, connectedPlayers []int) api.RoomView {
	connectedPlayers = slices.Clone(connectedPlayers)
	gameConfig := lobby.Game().Config()
	return api.RoomView{
		Phase:             string(lobby.Phase()),
		LeaderPlayerIndex: lobby.Leader(),
		PlayerCount:       lobby.PlayerCount(),
		JoinedPlayers:     lobby.JoinedPlayerCount(),
		ConnectedPlayers:  connectedPlayers,
		ReadyPlayers:      slices.Clone(lobby.ReadyPlayerIndices()),
		CanStart:          lobby.CanStart() && allJoinedPlayersConnected(lobby, connectedPlayers),
		CanRestart:        lobby.CanRestart(),
		CanDelete:         lobby.CanDelete(),
		GameConfig:        &gameConfig,
	}
}

func allJoinedPlayersConnected(lobby *sessionapp.Lobby, connectedPlayers []int) bool {
	connected := make(map[int]struct{}, len(connectedPlayers))
	for _, playerIndex := range connectedPlayers {
		connected[playerIndex] = struct{}{}
	}
	state := lobby.Game().Snapshot()
	for playerIndex := range state.players {
		if lobby.SessionForPlayer(playerIndex) == nil {
			continue
		}
		if _, ok := connected[playerIndex]; !ok {
			return false
		}
	}
	return lobby.JoinedPlayerCount() == lobby.PlayerCount()
}

// privateCardMessages converts internal Card structs into the PrivateCardView
// that the browser uses to render a player's own hand.  It is only ever
// included for the *viewer* — other players' hands are never sent over the wire.
func privateCardMessages(cards []corestate.Card) []api.PrivateCardView {
	result := make([]api.PrivateCardView, 0, len(cards))
	for _, card := range cards {
		result = append(result, api.PrivateCardView{
			ID:   card.ID,
			Role: string(card.Role),
		})
	}
	return result
}

func actionMessageFromCore(action *coretypes.PlayerAction) *api.ActionView {
	if action == nil {
		return nil
	}

	message := &api.ActionView{
		ID:         string(action.Action),
		ActorIndex: action.ActorIndex,
	}

	switch payload := action.Payload.(type) {
	case coretypes.TargetPayload:
		message.Payload.TargetIndex = &payload.TargetIndex
	case coretypes.AccusePayload:
		message.Payload.TargetIndex = &payload.TargetIndex
		message.Payload.Guess = string(payload.Guess)
	case nil:
	default:
	}

	return message
}

func counterResultMessage(result turnmodel.CounterSummary) api.CounterResultView {
	message := api.CounterResultView{
		Action:   actionMessageFromCore(result.Action),
		Applied:  result.Applied,
		Canceled: result.Canceled,
	}
	if result.Challenge != nil {
		challenge := challengeResultMessage(*result.Challenge)
		message.Challenge = &challenge
	}
	return message
}

func challengeResultMessage(result turnmodel.ChallengeSummary) api.ChallengeResultView {
	message := api.ChallengeResultView{
		Success:         result.Outcome == turnmodel.OutcomeActorBluffed,
		ActionAllowed:   result.ActionAllowed,
		ActorIndex:      result.ActorIndex,
		ChallengerIndex: result.ChallengerIndex,
		ClaimedRole:     string(result.ClaimedRole),
		Outcome:         string(result.Outcome),
		Logs:            slices.Clone(result.Logs),
		LostCardRole:    string(result.LostCardRole),
	}
	if result.LostPlayerIndex != nil {
		lostPlayerIndex := *result.LostPlayerIndex
		message.LostPlayerIndex = &lostPlayerIndex
	}
	return message
}

func roomStatusCopy(lobby *sessionapp.Lobby, summary api.MatchSnapshot, view api.RoomView, viewerIndex int) (string, string) {
	switch lobby.Phase() {
	case sessionapp.PhaseLobby:
		return lobbyWaitingCopy(slices.Contains(view.ReadyPlayers, viewerIndex), view.CanStart, view.LeaderPlayerIndex == viewerIndex)
	case sessionapp.PhaseGameOver:
		return gameOverWaitingCopy(view.LeaderPlayerIndex == viewerIndex)
	case sessionapp.PhaseInTurn:
		currentPlayerName := ""
		if summary.CurrentPlayerIndex >= 0 && summary.CurrentPlayerIndex < len(summary.Players) {
			currentPlayerName = summary.Players[summary.CurrentPlayerIndex].Name
		}
		return inTurnWaitingCopy(summary.CurrentPlayerIndex == viewerIndex, currentPlayerName)
	default:
		return "Stand by", "Waiting for the room state to update."
	}
}

func lobbyWaitingCopy(isReady, canStart, isLeader bool) (string, string) {
	switch {
	case !isReady:
		return "Lobby", "You are in the lobby. Mark ready when you are set."
	case canStart:
		if isLeader {
			return "Ready to start", "Both players are ready. Start the match when you are ready."
		}
		return "Ready to start", "Both players are ready. Waiting for the match to begin."
	case isLeader:
		return "Lobby", "You are ready. Waiting for the other player so you can start the match."
	default:
		return "Lobby", "You are ready. Waiting for the other player to get set."
	}
}

func inTurnWaitingCopy(isCurrentPlayer bool, currentPlayerName string) (string, string) {
	if isCurrentPlayer {
		return "Your turn", "Your prompt will appear here as soon as the server opens it."
	}
	if currentPlayerName == "" {
		return "Stand by", "Waiting for the current player to act."
	}
	return "Stand by", fmt.Sprintf("Waiting for %s to act.", currentPlayerName)
}

func gameOverWaitingCopy(isLeader bool) (string, string) {
	if isLeader {
		return "Game over", "You can restart this room or delete it when you are ready."
	}
	return "Game over", "Waiting for the lobby leader to restart or close the room."
}
