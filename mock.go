package tock

import "time"

type mockClock struct {
	now time.Time
}

//var _ Clock = &MockClock{}

func NewMock() *mockClock {
	return &mockClock{
		now: time.Now(),
	}
}

func (c *mockClock) Now() time.Time {
	return c.now
}
