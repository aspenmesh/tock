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

	Timer struct {
		C <-chan time.Time

		// one of mock/real is set
		mock *mockClock
		real *time.Timer

		outC chan<- time.Time
		when time.Time
	}

	Ticker struct {
		C <-chan time.Time

		// one of mock/real is set
		mock *mockClock
		real *time.Ticker

		outC   chan<- time.Time
		when   time.Time
		period time.Duration
	}
)
