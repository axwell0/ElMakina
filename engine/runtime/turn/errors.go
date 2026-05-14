package turn

import "errors"

var (
	errActionDidNotComplete   = errors.New("action did not complete")
	errActorMismatch          = errors.New("main action actor mismatch")
	errChallengeSelection     = errors.New("invalid challenge selection context")
	errChallengerNotAllowed   = errors.New("challenger not allowed")
	errClockNil               = errors.New("clock is nil")
	errCounterActionNil       = errors.New("counter action is nil")
	errCounterActionNotFound  = errors.New("counter action not allowed")
	errCounterPlayerDuplicate = errors.New("counter player already responded")
	errCounterPlayerNotFound  = errors.New("counter player not allowed")
	errInputProviderNil       = errors.New("input provider is nil")
	errInvalidCardIndex       = errors.New("invalid card index")
	errInvalidChallengeRole   = errors.New("actor does not have claimed role")
	errInvalidChallengeTarget = errors.New("challenger cannot be the actor")
	errInvalidResponse        = errors.New("invalid response")
	errMainActionNil          = errors.New("main action is nil")
	errServiceStateNil        = errors.New("turn service state is nil")
	errUnsupportedActionStep  = errors.New("unsupported action step")
)
