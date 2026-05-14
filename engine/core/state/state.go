package state

import (
	"fmt"
	"slices"
)

const MaxCoins = 12

type GameState struct {
	TurnNumber         int // Current turn number.
	players            []Player
	Deck               Deck // Draw pile.
	CurrentPlayerIndex int  // Whose turn it is.
	MaxCoins           int  // Maximum coins a player can hold in this game.
}

// DiscardDetail captures the exact card removed from a player's hand.
type DiscardDetail struct {
	PlayerIndex int
	Card        Card
}

// DrawDetail captures the exact cards drawn into a player's hand.
type DrawDetail struct {
	PlayerIndex int
	Cards       []Card
}

// ReturnToDeckDetail captures which cards were returned versus kept after exchange.
type ReturnToDeckDetail struct {
	PlayerIndex int
	Returned    []Card
	Kept        []Card
}

// SwapAndRevealDetail captures the revealed card returned to the deck and the card drawn back.
type SwapAndRevealDetail struct {
	PlayerIndex int
	Revealed    Card
	Drawn       Card
}
// TODO: use this everywhere
func (g *GameState) PlayerAt(playerIndex int) (*Player, error) {
	if g == nil {
		return nil, errStateNil
	}
	if playerIndex >= len(g.players) || playerIndex < 0 {
		return nil, errInvalidPlayerIndex
	}

	return &g.players[playerIndex], nil
}

// Snapshot creates a deep copy of GameState for persistence.
func (g *GameState) Snapshot() *GameState {
	if g == nil {
		return nil
	}
	newState := *g
	// Deep-copy slices so callers can mutate the snapshot safely.
	newState.players = slices.Clone(g.players)
	for i := range newState.players {
		newState.players[i].Hand = slices.Clone(g.players[i].Hand)
	}

	// Deck copy preserves draw order at snapshot time.
	newState.Deck = slices.Clone(g.Deck)
	return &newState
}

// AddCoins adds or removes coins from a player. Enforces non-negative balance
// and MaxCoins cap. Returns new coin count or error.
//
// This is a pointer receiver method (func (g *GameState) ...) because it mutates g.
// In Go, pointer receivers modify the original; value receivers work on a copy.
// All GameState mutation methods use pointer receivers for this reason.
func (g *GameState) AddCoins(playerIndex, coinDelta int) (int, error) {
	if g == nil {
		return 0, errStateNil
	}
	if playerIndex < 0 || playerIndex >= len(g.players) {
		return 0, fmt.Errorf("%w: %d", errInvalidPlayerIndex, playerIndex)
	}
	player := &g.players[playerIndex]
	newCoins := player.Coins + coinDelta
	if newCoins < 0 {
		return 0, fmt.Errorf("%w: have %d, need %d", errInsufficientCoins, player.Coins, -coinDelta)
	}

	maxCoins := g.MaxCoins
	if maxCoins == 0 {
		maxCoins = MaxCoins
	}
	if newCoins > maxCoins {
		newCoins = maxCoins
	}

	player.Coins = newCoins
	return newCoins, nil
}

// DrawCards deals cards from the deck and returns the exact drawn cards.
func (g *GameState) DrawCards(playerIndex, count int) (DrawDetail, error) {
	if g == nil {
		return DrawDetail{}, errStateNil
	}
	if playerIndex < 0 || playerIndex >= len(g.players) {
		return DrawDetail{}, fmt.Errorf("%w: %d", errInvalidPlayerIndex, playerIndex)
	}
	player := &g.players[playerIndex]
	drawn, err := g.Deck.Draw(count)
	if err != nil {
		return DrawDetail{}, fmt.Errorf("draw from deck: %w", err)
	}
	// Drawn cards are appended in draw order.
	player.Hand = append(player.Hand, drawn...)
	return DrawDetail{
		PlayerIndex: playerIndex,
		Cards:       slices.Clone(drawn),
	}, nil
}

// ReturnCardsToDeck moves selected cards back to the deck and reports both sides.
func (g *GameState) ReturnCardsToDeck(playerIndex int, cardIndices []int) (ReturnToDeckDetail, error) {
	if g == nil {
		return ReturnToDeckDetail{}, errStateNil
	}
	if playerIndex < 0 || playerIndex >= len(g.players) {
		return ReturnToDeckDetail{}, fmt.Errorf("%w: %d", errInvalidPlayerIndex, playerIndex)
	}
	player := &g.players[playerIndex]

	keep := make([]Card, 0, len(player.Hand))
	returnCards := make([]Card, 0, len(cardIndices))
	selected := make(map[int]struct{}, len(cardIndices))

	// Validate all indices before mutating any hand/deck state.
	for _, idx := range cardIndices {
		if idx < 0 || idx >= len(player.Hand) {
			return ReturnToDeckDetail{}, fmt.Errorf("%w: %d", errInvalidCardIndex, idx)
		}
		if _, exists := selected[idx]; exists {
			return ReturnToDeckDetail{}, fmt.Errorf("%w: %d", errDuplicateCardIndex, idx)
		}
		selected[idx] = struct{}{}
	}

	// Partition hand into returned and kept slices.
	for i, card := range player.Hand {
		if _, ok := selected[i]; ok {
			returnCards = append(returnCards, card)
			continue
		}
		keep = append(keep, card)
	}

	// Commit partition results and push returned cards to deck tail.
	player.Hand = keep
	g.Deck = append(g.Deck, returnCards...)

	return ReturnToDeckDetail{
		PlayerIndex: playerIndex,
		Returned:    slices.Clone(returnCards),
		Kept:        slices.Clone(keep),
	}, nil
}

// SwapAndRevealCard exchanges a chosen card with the top of the deck and returns both cards.
func (g *GameState) SwapAndRevealCard(playerIndex, cardIndex int) (SwapAndRevealDetail, error) {
	if g == nil {
		return SwapAndRevealDetail{}, errStateNil
	}
	if playerIndex < 0 || playerIndex >= len(g.players) {
		return SwapAndRevealDetail{}, fmt.Errorf("%w: %d", errInvalidPlayerIndex, playerIndex)
	}
	player := &g.players[playerIndex]
	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return SwapAndRevealDetail{}, fmt.Errorf("%w: %d", errInvalidCardIndex, cardIndex)
	}

	// Remove revealed card, return it to deck, then draw replacement.
	revealed := player.Hand[cardIndex]
	player.Hand = append(player.Hand[:cardIndex], player.Hand[cardIndex+1:]...)
	g.Deck = append(g.Deck, revealed)
	drawn, err := g.Deck.Draw(1)
	if err != nil {
		return SwapAndRevealDetail{}, fmt.Errorf("draw replacement card: %w", err)
	}
	player.Hand = append(player.Hand, drawn...)

	return SwapAndRevealDetail{
		PlayerIndex: playerIndex,
		Revealed:    revealed,
		Drawn:       drawn[0],
	}, nil
}

// DiscardFromHand removes a card and returns the exact mutation details.
func (g *GameState) DiscardFromHand(playerIndex, cardIndex int) (DiscardDetail, error) {
	if g == nil {
		return DiscardDetail{}, errStateNil
	}
	if playerIndex < 0 || playerIndex >= len(g.players) {
		return DiscardDetail{}, fmt.Errorf("%w: %d", errInvalidPlayerIndex, playerIndex)
	}
	player := &g.players[playerIndex]
	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return DiscardDetail{}, fmt.Errorf("%w: %d", errInvalidCardIndex, cardIndex)
	}

	// Remove chosen card from hand, then append it to deck/discard tracking.
	card := player.Hand[cardIndex]
	player.Hand = append(player.Hand[:cardIndex], player.Hand[cardIndex+1:]...)
	g.Deck = append(g.Deck, card)

	return DiscardDetail{PlayerIndex: playerIndex, Card: card}, nil
}

func (g *GameState) CurrentPlayer() Player {
	return g.players[g.CurrentPlayerIndex]
}
