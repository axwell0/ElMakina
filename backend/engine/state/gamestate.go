package state

import (
	"ElMakina/backend/models"
	"fmt"
	"math/rand/v2"
)

const MaxCoins = 12 // Maximum coins a player can hold

type StepKind string

const (
	StepCardSelect StepKind = "card_select" // Player must select cards from hand
)

type StepContext string

const (
	ContextDiscard  StepContext = "discard"  // Discarding cards (Coup, Assassinate)
	ContextExchange StepContext = "exchange" // Exchanging cards with deck
)

// StepRequest is sent to InputProvider when an action needs additional input.
// Used for multi-step actions like Exchange (card selection) or Investigate.
type StepRequest struct {
	ActionID models.ActionID `json:"action_id"`         // Action requiring input
	Kind     StepKind        `json:"kind"`              // Type of input needed
	Actor    int             `json:"actor_index"`       // Player who must respond
	Count    int             `json:"count"`             // Number of items to select
	Context  StepContext     `json:"context"`           // Context for the input
	Options  []string        `json:"options,omitempty"` // Valid options (if applicable)
}

// GameState is the "One Source of Truth" for a specific moment in time.
//
// Think of this as a Polaroid snapshot. It contains the exact position of every piece
// on the board: who has which cards, how many coins are in their pockets, and whose
// turn it is.
//
// CRITICAL RULE: GameState should never be mutated "loosely". We use methods like
// `AddCoins` or `DiscardFromHand` because they enforce the laws of the game
// (e.g., you can't have negative coins).
//
// We use `Snapshot()` to create deep copies when persisting state (e.g., replays).
// Outside of persistence, the live GameState is treated as read-only by callers.
//
// See: Snapshot() at line 66 for deep copy.
// See: AddCoins() at line 118 for coin management.
// See: DiscardFromHand() at line 87 for card management.
// See: engine/game.go:19 for how Game holds GameState.
type GameState struct {
	TurnNumber         int      // Current turn number
	Players            []Player // All players in the game
	Deck               Deck     // Draw pile
	CurrentPlayerIndex int      // Whose turn it is

	// Transient fields (not serialized to JSON)
	DiscardEvents  []DiscardEvent // Cards discarded this turn
	PendingRequest *StepRequest   // Active multi-step request
	seed           *rand.Rand
}

// DiscardEvent records a card being discarded from a player's hand.
type DiscardEvent struct {
	PlayerIndex int  // Player who discarded
	Card        Card // Card that was discarded
}

// Snapshot creates a deep copy of GameState for persistence/replay.
func (g *GameState) Snapshot() *GameState {
	if g == nil {
		return nil
	}
	newState := *g
	newState.Players = make([]Player, len(g.Players))
	for i := range g.Players {
		newState.Players[i] = g.Players[i]
		newState.Players[i].Hand = append([]Card{}, g.Players[i].Hand...)
	}

	newState.Deck = make(Deck, len(g.Deck))
	copy(newState.Deck, g.Deck)
	newState.DiscardEvents = nil
	newState.PendingRequest = nil

	return &newState
}

// DiscardFromHand removes a card from a player's hand and returns it to the deck.
// Records a DiscardEvent for logging. Returns error if indices are invalid.
func (g *GameState) DiscardFromHand(playerIndex int, cardIndex int) (Card, error) {
	if playerIndex < 0 || playerIndex >= len(g.Players) {
		return Card{}, fmt.Errorf("invalid player index")
	}
	player := &g.Players[playerIndex]
	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return Card{}, fmt.Errorf("invalid card index")
	}

	card := player.Hand[cardIndex]
	player.Hand = append(player.Hand[:cardIndex], player.Hand[cardIndex+1:]...)
	g.Deck = append(g.Deck, card)
	g.DiscardEvents = append(g.DiscardEvents, DiscardEvent{PlayerIndex: playerIndex, Card: card})

	return card, nil
}

// DrainDiscardEvents returns and clears all pending discard events.
// Called after each action to emit logs.
func (g *GameState) DrainDiscardEvents() []DiscardEvent {
	if len(g.DiscardEvents) == 0 {
		return nil
	}
	events := make([]DiscardEvent, len(g.DiscardEvents))
	copy(events, g.DiscardEvents)
	g.DiscardEvents = nil
	return events
}

// AddCoins adds or removes coins from a player. Enforces non-negative balance
// and MaxCoins cap. Returns new coin count or error.
func (g *GameState) AddCoins(playerIndex int, delta int) (int, error) {
	if playerIndex < 0 || playerIndex >= len(g.Players) {
		return 0, fmt.Errorf("invalid player index %d", playerIndex)
	}
	player := &g.Players[playerIndex]
	newCoins := player.Coins + delta
	if newCoins < 0 {
		return 0, fmt.Errorf("insufficient coins: have %d, need %d", player.Coins, -delta)
	}
	if newCoins > MaxCoins {
		newCoins = MaxCoins
	}
	player.Coins = newCoins
	return newCoins, nil
}

// DrawCards deals cards from the deck to a player's hand.
func (g *GameState) DrawCards(playerIndex int, count int) error {
	if playerIndex < 0 || playerIndex >= len(g.Players) {
		return fmt.Errorf("invalid player index %d", playerIndex)
	}
	player := &g.Players[playerIndex]
	drawn := g.Deck.Draw(count)
	player.Hand = append(player.Hand, drawn...)
	return nil
}

// ReturnCardsToDeck moves specified cards from a player's hand back to the deck.
// Used by Exchange action. Validates all indices before modifying state.
func (g *GameState) ReturnCardsToDeck(playerIndex int, cardIndices []int) error {
	if playerIndex < 0 || playerIndex >= len(g.Players) {
		return fmt.Errorf("invalid player index %d", playerIndex)
	}
	player := &g.Players[playerIndex]

	keep := make([]Card, 0)
	returnCards := make([]Card, 0)

	for _, idx := range cardIndices {
		if idx < 0 || idx >= len(player.Hand) {
			return fmt.Errorf("invalid card index %d", idx)
		}
	}

	for i, card := range player.Hand {
		shouldReturn := false
		for _, idx := range cardIndices {
			if i == idx {
				shouldReturn = true
				break
			}
		}
		if shouldReturn {
			returnCards = append(returnCards, card)
		} else {
			keep = append(keep, card)
		}
	}

	player.Hand = keep
	g.Deck = append(g.Deck, returnCards...)

	return nil
}

// SwapCard exchanges a card from hand with the top card of the deck,
// placing the revealed card on the bottom.
func (g *GameState) SwapCard(playerIndex int, cardIndex int) error {
	if playerIndex < 0 || playerIndex >= len(g.Players) {
		return fmt.Errorf("invalid player index")
	}
	player := &g.Players[playerIndex]
	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return fmt.Errorf("invalid card index")
	}

	card := player.Hand[cardIndex]
	player.Hand = append(player.Hand[:cardIndex], player.Hand[cardIndex+1:]...)
	g.Deck = append(g.Deck, card)
	drawn := g.Deck.Draw(1)
	player.Hand = append(player.Hand, drawn...)

	return nil
}
