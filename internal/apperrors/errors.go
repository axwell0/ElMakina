package apperrors

import "errors"

var (
	ErrRoomNotFound        = errors.New("room not found")
	ErrSessionNotFound     = errors.New("session not found")
	ErrInvalidPhase        = errors.New("invalid lobby phase")
	ErrUnauthorizedCommand = errors.New("unauthorized command")
	ErrDeckExhausted       = errors.New("deck exhausted")
	ErrNoPrompt            = errors.New("no prompt")
	ErrStalePrompt         = errors.New("stale prompt")
)
