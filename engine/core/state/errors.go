package state

import "errors"

var (
	errDuplicateCardIndex = errors.New("duplicate card index")
	errInvalidCardIndex   = errors.New("invalid card index")
	errInvalidDrawCount   = errors.New("invalid draw count")
	errInvalidPlayerIndex = errors.New("invalid player index")
	errStateNil           = errors.New("state is nil")
	errInsufficientCoins  = errors.New("insufficient coins")
)
