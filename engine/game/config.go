package game

import (
	"time"
)

const (
	DefaultMaxCoins      = 12
	DefaultStartingCoins = 2
)

// Config holds all tunable parameters for a game of El Makina.
// This is passed into game.NewState to control deck size, coin economy, and turn timers.
// The server stores one Config per room and persists it across restarts.
type Config struct {
	Turn    TurnConfig    `json:"turn"`    // time limits for each prompt phase in milliseconds
	Setup   CardConfig    `json:"setup"`   // deck construction parameters
	Economy EconomyConfig `json:"economy"` // coin limits and starting wealth
}

// TurnConfig sets per-prompt timeout durations.
// In production these give players a limited window to act.
type TurnConfig struct {
	ChallengeTimeout int `json:"challengeTimeout"`
	CounterTimeout   int `json:"counterTimeout"`
	TurnTimeout      int `json:"turnTimeout"`
}

// CardConfig controls how the deck is built.
// CardCopies determines how many copies of each role card go into the deck (default 3).
// CardsPerPlayer can override how many cards each player draws (defaults to 2 or 3 based on player count).
type CardConfig struct {
	CardsPerPlayer int `json:"cardsPerPlayer,omitempty"`
	CardCopies     int `json:"cardCopies,omitempty"`
}

// EconomyConfig controls starting wealth and the soft coin cap.
// MaxCoins and StartingCoins are both validated in newState: starting > max is rejected.
type EconomyConfig struct {
	MaxCoins      int `json:"maxCoins"`
	StartingCoins int `json:"startingCoins"`
}

func DefaultConfig() Config {
	return Config{
		Turn: TurnConfig{
			TurnTimeout:      int((60 * time.Second) / time.Millisecond),
			CounterTimeout:   int((30 * time.Second) / time.Millisecond),
			ChallengeTimeout: int((30 * time.Second) / time.Millisecond),
		},
		Setup: CardConfig{CardCopies: 3},
		Economy: EconomyConfig{
			MaxCoins:      DefaultMaxCoins,
			StartingCoins: DefaultStartingCoins,
		},
	}
}
