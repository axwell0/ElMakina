package config

import (
	"time"
)

const (
	DefaultMaxCoins      = 12
	DefaultStartingCoins = 2
)

type GameConfig struct {
	Turn    TurnConfig    `json:"turn"`
	Setup   CardConfig    `json:"setup"`
	Economy EconomyConfig `json:"economy"`
}

type TurnConfig struct {
	ChallengeTimeoutMillis int `json:"challengeTimeoutMillis"`
	CounterTimeoutMillis   int `json:"counterTimeoutMillis"`
	TurnTimeoutMillis      int `json:"turnTimeoutMillis"`
}

type CardConfig struct {
	CardsPerPlayer int `json:"cardsPerPlayer,omitempty"`
	DeckCopies     int `json:"deckCopies,omitempty"`
}

type EconomyConfig struct {
	MaxCoins      int `json:"maxCoins"`
	StartingCoins int `json:"startingCoins"`
}

func DefaultGameConfig() GameConfig {
	return GameConfig{
		Turn: TurnConfig{
			TurnTimeoutMillis:      int((60 * time.Second) / time.Millisecond),
			CounterTimeoutMillis:   int((30 * time.Second) / time.Millisecond),
			ChallengeTimeoutMillis: int((30 * time.Second) / time.Millisecond),
		},
		Setup: CardConfig{DeckCopies: 3},
		Economy: EconomyConfig{
			MaxCoins:      DefaultMaxCoins,
			StartingCoins: DefaultStartingCoins,
		},
	}
}
