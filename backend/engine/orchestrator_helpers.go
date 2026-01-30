package engine

import "time"

func timerChan(clock Clock, d time.Duration) <-chan time.Time {
	if d <= 0 {
		return nil
	}
	return clock.After(d)
}

func inputErrors(input InputProvider) <-chan error {
	type errorReporter interface {
		Errors() <-chan error
	}
	if r, ok := input.(errorReporter); ok {
		return r.Errors()
	}
	return nil
}
