package state

import (
	"fmt"
	"math/rand/v2"

	models "github.com/your-org/elmakina/engine/core/types"
	"github.com/your-org/elmakina/internal/apperrors"
)

type Deck []Card

// Draw removes and returns n cards from the front of the deck.
func (d *Deck) Draw(n int) ([]Card, error) {
	if n < 0 {
		return nil, fmt.Errorf("draw count %d is invalid", n)
	}
	if n > len(*d) {
		return nil, fmt.Errorf("draw %d cards from deck with %d cards: %w", n, len(*d), apperrors.ErrDeckExhausted)
	}

	drawn := (*d)[:n]
	*d = (*d)[n:]
	return drawn, nil
}

func (d *Deck) Shuffle() {
	rand.Shuffle(len(*d), func(i, j int) {
		(*d)[i], (*d)[j] = (*d)[j], (*d)[i]
	})
}

func NewDeck(cardsPerRole int) Deck {
	roles := models.Roles
	totalCards := len(roles) * cardsPerRole
	cards := make(Deck, 0, totalCards)
	cardID := 0

	for _, role := range roles {
		for i := 0; i < cardsPerRole; i++ {
			cards = append(cards, Card{ID: cardID, Role: role})
			cardID++
		}
	}
	cards.Shuffle()
	return cards
}
