package tock

import "time"

type (
	// Clock is a wrapper around the time package.
	Clock interface {
		Now() time.Time
		After(duration time.Duration) <-chan time.Time
		Sleep(duration time.Duration)
		Since(t time.Time) time.Duration
		Until(t time.Time) time.Duration
		NewTimer(duration time.Duration) *Timer
		NewTicker(duration time.Duration) *Ticker
	}

	// Timer is created from NewTimer() and sends to C when elapsed
	Timer struct {
		C <-chan time.Time

		// one of mock/real is set
		mock *mockClock
		real *time.Timer

		outC  chan<- time.Time
		when  time.Time
		stopC chan struct{}
	}

	// Ticker is created from NewTicker() and sends to C every duration interval
	Ticker struct {
		C <-chan time.Time

		// one of mock/real is set
		mock *mockClock
		real *time.Ticker

		outC   chan<- time.Time
		when   time.Time
		period time.Duration
		stopC  chan struct{}
	}
)
