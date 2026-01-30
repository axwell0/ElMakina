package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
)

func TestSwapCard(t *testing.T) {
	originalCard := state.Card{ID: 100, Role: models.Thief}
	player := state.Player{
		ID:    0,
		Name:  "A",
		Hand:  []state.Card{originalCard},
		Coins: 2,
	}
	deck := state.Deck{
		{ID: 1, Role: models.Businesswoman},
		{ID: 2, Role: models.Colonel},
		{ID: 3, Role: models.Politician},
	}
	gs := &state.GameState{
		Players: []state.Player{player},
		Deck:    deck,
	}

	err := gs.SwapCard(0, 0)
	require.NoError(t, err)

	require.Len(t, gs.Players[0].Hand, 1)
	require.Equal(t, deck[0], gs.Players[0].Hand[0])

	expectedDeck := state.Deck{
		deck[1],
		deck[2],
		originalCard,
	}
	require.Equal(t, expectedDeck, gs.Deck)
}
