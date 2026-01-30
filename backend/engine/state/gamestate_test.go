package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ElMakina/backend/models"
)

func TestGameStateAddCoinsClampAndErrors(t *testing.T) {
	gs := &GameState{Players: []Player{{ID: 0, Name: "A", Coins: 1}}}
	_, err := gs.AddCoins(0, -2)
	require.Error(t, err)
	_, err = gs.AddCoins(0, 20)
	require.NoError(t, err)
	require.Equal(t, MaxCoins, gs.Players[0].Coins)
}

func TestDiscardAndReturnCards(t *testing.T) {
	gs := &GameState{
		Players: []Player{{ID: 0, Name: "A", Hand: []Card{{ID: 1, Role: models.TaxCollector}, {ID: 2, Role: models.Thief}}}},
		Deck:    Deck{},
	}
	_, err := gs.DiscardFromHand(0, 0)
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 1)
	require.Len(t, gs.Deck, 1)

	err = gs.ReturnCardsToDeck(0, []int{0})
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 0)
	require.Len(t, gs.Deck, 2)

	err = gs.ReturnCardsToDeck(0, []int{1})
	require.Error(t, err)
}

func TestSwapCardAndDraw(t *testing.T) {
	gs := &GameState{
		Players: []Player{{ID: 0, Name: "A", Hand: []Card{{ID: 1, Role: models.TaxCollector}}}},
		Deck:    Deck{{ID: 2, Role: models.Thief}, {ID: 3, Role: models.Colonel}},
	}

	err := gs.DrawCards(0, 1)
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 2)

	err = gs.SwapCard(0, 0)
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 2)
}

func TestSnapshotDoesNotAlias(t *testing.T) {
	gs := &GameState{
		Players: []Player{{ID: 0, Name: "A", Hand: []Card{{ID: 1, Role: models.TaxCollector}}}},
		Deck:    Deck{{ID: 2, Role: models.Thief}},
	}
	snapshot := gs.Snapshot()
	snapshot.Players[0].Hand[0].Role = models.Thief
	require.NotEqual(t, models.Thief, gs.Players[0].Hand[0].Role)
	snapshot.Deck[0].Role = models.Colonel
	require.NotEqual(t, models.Colonel, gs.Deck[0].Role)
}
