package game

import "errors"

var (
	errDuplicateCardIndex = errors.New("duplicate card index") // a card list contains the same card ID twice
	errInvalidCardIndex   = errors.New("invalid card index")   // card index out of the hand's range
	errInvalidDrawCount   = errors.New("invalid draw count")   // draw count is negative or exceeds deck size
	errInvalidPlayerIndex = errors.New("invalid player index") // player index outside [0, PlayerCount)
	errStateNil           = errors.New("state is nil")         // method called on a nil *State
	errInsufficientCoins  = errors.New("insufficient coins")   // player does not have enough coins for a cost
)
