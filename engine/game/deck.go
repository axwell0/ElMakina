package game

import (
	"fmt"
	"math/rand/v2"
)

type Deck []Card

func (d *Deck) Draw(n int) ([]Card, error) {
	if n < 0 {
		return nil, fmt.Errorf("%w: %d", errInvalidDrawCount, n)
	}
	if n > len(*d) {
		return nil, fmt.Errorf("draw %d cards from deck with %d cards", n, len(*d))
	}
	drawn := (*d)[:n:n]
	*d = (*d)[n:]
	return drawn, nil
}

func (d *Deck) Shuffle() {
	rand.Shuffle(len(*d), func(i, j int) {
		(*d)[i], (*d)[j] = (*d)[j], (*d)[i]
	})
}

func NewDeck(cardsPerRole int) Deck {
	totalCards := len(Roles) * cardsPerRole
	cards := make(Deck, 0, totalCards)
	cardID := 0
	for _, role := range Roles {
		for i := 0; i < cardsPerRole; i++ {
			cards = append(cards, Card{ID: cardID, Role: role})
			cardID++
		}
	}
	cards.Shuffle()
	return cards
}
