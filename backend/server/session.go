package server

import (
	"ElMakina/backend/engine"
	"ElMakina/backend/engine/state"
)

// GameSession is the "Translation Layer".
//
// In the lobby, we know users by their unique IDs (like "alice-123").
// In the Engine, they are just indices (0, 1, 2).
// GameSession holds the dictionary that says "alice-123 is actually Player 0".
//
// Without this, the Engine would be shouting into the void, and the
// transport layer wouldn't know which WebSocket connection to send the
// "Your Turn" message to.
type GameSession struct {
	MatchID       string
	LobbyID       string
	Game          *engine.Game
	PlayerIDs     []string
	IndexByPlayer map[string]int
	PlayerAvatars []string
	RNGSeed       string
}

// Forfeit marks a player as out of the game by emptying their hand.
// This is used when a player disconnects mid-game.
func (s *GameSession) Forfeit(playerID string) bool {
	if s == nil || s.Game == nil {
		return false
	}
	index, ok := s.IndexByPlayer[playerID]
	if !ok {
		return false
	}
	if index < 0 || index >= len(s.Game.CurrentState.Players) {
		return false
	}
	s.Game.CurrentState.Players[index].Hand = []state.Card{}
	return true
}
