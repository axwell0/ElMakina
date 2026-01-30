package engine

import (
	"ElMakina/backend/actions"
	"ElMakina/backend/engine/state"
	"fmt"
	"math/rand/v2"
)

// Game is the root object for a match instance. Think of it as the "Kernel" of the engine.
// It is strictly responsible for two things: holding the current state of truth (GameState)
// and providing the machinery to move that state forward (the Orchestrator).
//
// A key design choice here is that Game is entirely transport-agnostic. It has no idea
// if it's being talked to via WebSockets, a CLI, or a series of unit tests.
// It interacts with the outside world purely through the InputProvider interface (see orchestrator_turn.go).
// This decoupling is what allows us to "replay" games just by feeding the engine a list of
// past actions—since the engine doesn't care WHERE the actions come from, only that they are valid.
//
// See: orchestrator_turn.go:29 for the main game loop.
// See: state/gamestate.go:47 for the GameState definition.
// See: server/ws/session_runner.go:201 for the bridge to WebSockets.
type Game struct {
	CurrentState state.GameState     // The "One Source of Truth" for players, cards, and coins.
	TurnState    state.TurnState     // Temporary "scratchpad" for the current turn (e.g., how many times has Tax been called?)
	History      []state.GameState   // A literal paper trail of every state change, used for replays.
	ActiveFlow   *actions.FlowRunner // Manages actions that need multiple steps (like Exchange where you draw then discard).
	rng          *rand.Rand
}

// NewGame creates a new game with optional RNG for deterministic behavior.
// If rng is nil, uses default random number generator. Deals cards based on player count:
// 2-4 players get 3 cards each, 5+ players get 2 cards each.
//
// The cardsPerPlayer logic scales the starting hand based on the crowd size
// to keep the game's pace and balance tight. 2-4 players get more "lives" (cards)
// because the game is slower, while 5+ players get fewer to prevent it from dragging
// too long or running out of deck early.
func NewGame(playerNames []string, rng *rand.Rand) (*Game, error) {
	if len(playerNames) < 2 {
		return nil, fmt.Errorf("at least 2 players required")
	}
	if len(playerNames) > 9 { // Assuming max 6 for now?
		return nil, fmt.Errorf("max 9 players allowed")
	}

	cardsPerPlayer := 2
	if len(playerNames) <= 4 {
		cardsPerPlayer = 3
	}
	deck := state.NewDeck(3, rng)

	players := make([]state.Player, len(playerNames))
	for i, name := range playerNames {
		hand := deck.Draw(cardsPerPlayer)
		players[i] = state.Player{
			ID:    i,
			Name:  name,
			Hand:  hand,
			Coins: 2,
		}
	}

	gs := state.GameState{
		TurnNumber:         1,
		Players:            players,
		Deck:               deck,
		CurrentPlayerIndex: 0,
	}

	return &Game{
		CurrentState: gs,
		TurnState:    state.TurnState{},
		History:      make([]state.GameState, 0),
		rng:          rng,
	}, nil
}

// CheckGameOver returns the winner if only one player has cards remaining.
// Returns nil if game is still ongoing.
func (g *Game) CheckGameOver() *state.Player {
	if g == nil {
		return nil
	}
	gs := &g.CurrentState
	activePlayers := 0
	var lastPlayer *state.Player
	for i := range gs.Players {
		if len(gs.Players[i].Hand) > 0 {
			activePlayers++
			lastPlayer = &gs.Players[i]
		}
	}
	if activePlayers == 1 {
		return lastPlayer
	}
	return nil
}

// validatePlayerIndex checks that index is within bounds for the players slice.
func (g *Game) validatePlayerIndex(index int, label string) error {
	if index < 0 || index >= len(g.CurrentState.Players) {
		return fmt.Errorf("%s index %d out of range", label, index)
	}
	return nil
}

// IncrementTurn advances to the next active player (skips eliminated players).
// Also increments turn number and resets per-turn state.
func (g *Game) IncrementTurn() {
	gs := &g.CurrentState
	if gs == nil || len(gs.Players) == 0 {
		return
	}

	start := gs.CurrentPlayerIndex
	for {
		gs.CurrentPlayerIndex = (gs.CurrentPlayerIndex + 1) % len(gs.Players)
		// Loop until we find a player with cards
		if len(gs.Players[gs.CurrentPlayerIndex].Hand) > 0 {
			break
		}
		// If we looped all the way back, everyone is dead (should not happen if game over check works)
		if gs.CurrentPlayerIndex == start {
			break
		}
	}

	gs.TurnNumber++
	g.TurnState.Reset()
}
