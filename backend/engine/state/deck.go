package state

import (
	"math/rand/v2"

	"ElMakina/backend/models"
)

type Deck []Card

var DeckRNG *rand.Rand

func (d *Deck) Draw(n int) []Card {
	if n >= len(*d) {
		drawn := *d
		*d = (*d)[:0]
		return drawn
	}

	drawn := (*d)[:n]
	*d = (*d)[n:]
	return drawn
}

// Shuffle shuffles the deck using a provided RNG when available.
// Falls back to DeckRNG if rng is nil, then to global rand.Shuffle.
func (d *Deck) Shuffle(rng *rand.Rand) {
	if rng == nil {
		if DeckRNG != nil {
			DeckRNG.Shuffle(len(*d), func(i, j int) {
				(*d)[i], (*d)[j] = (*d)[j], (*d)[i]
			})
		} else {
			rand.Shuffle(len(*d), func(i, j int) {
				(*d)[i], (*d)[j] = (*d)[j], (*d)[i]
			})
		}
		return
	}
	rng.Shuffle(len(*d), func(i, j int) {
		(*d)[i], (*d)[j] = (*d)[j], (*d)[i]
	})
}

// NewDeck builds a deck and shuffles it using the provided RNG.
// TODO: a good idea here is to pass in an array of roles (and possible their counts) instead of relying on models.Roles.
// This would allow players to customize the deck composition. (e.g., 2 Assassin cards, 3 Duke cards)
func NewDeck(cardsPerRole int, rng *rand.Rand) Deck {
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
	cards.Shuffle(rng)
	return cards
}
