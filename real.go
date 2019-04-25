package tock

import (
	"time"
)

type realClock struct {
}

var _ Clock = &realClock{}

func NewReal() *realClock {
	return &realClock{}
}

func (c *realClock) Now() time.Time {
	return time.Now()
}

func (c *realClock) After(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

func (c *realClock) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (c *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (c *realClock) Until(t time.Time) time.Duration {
	return time.Until(t)
}

func (c *realClock) NewTimer(duration time.Duration) *Timer {
	return &Timer{real: time.NewTimer(duration)}
}

func (c *realClock) NewTicker(duration time.Duration) *Ticker {
	return &Ticker{real: time.NewTicker(duration)}
}

func (t *Timer) Reset(d time.Duration) bool {
	if t.real != nil {
		return t.real.Reset(d)
	}

	panic("Not implemented")
}

func (t *Timer) Stop() bool {
	if t.real != nil {
		return t.real.Stop()
	}
	return stopMockTimer(t)
}

func (t *Ticker) Stop() {
	if t.real != nil {
		t.real.Stop()
		return
	}
	stopMockTicker(t)
}
