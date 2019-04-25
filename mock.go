package tock

import (
	"runtime"
	"sort"
	"sync"
	"time"
)

// MockClock is a Clock where time only advances under your control
type MockClock interface {
	Clock
	Advance(duration time.Duration)
	BlockUntil(n int)
}

type mockClock struct {
	advanceMutex sync.Mutex // Mutexes Advance()
	mutex        sync.Mutex // Protects all internal data

	now             time.Time
	sleepersChanged []chan int

	// Sleepers are an empty interface because we want to preserve the struct
	// with a channel C for the Ticker or Timer we return to users.
	// These are sorted by time.
	sleepers []interface{}
}

var _ MockClock = &mockClock{}

// NewMock creates a new mock Clock, starting at zero time
func NewMock() *mockClock {
	return &mockClock{
		now: time.Time{},
	}
}

// Now is the current time, starting at zero time and controlled by Advance
func (c *mockClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

func sleeperWhen(sleeper interface{}) time.Time {
	switch s := sleeper.(type) {
	case *Timer:
		return s.when
	case *Ticker:
		return s.when
	default:
		return time.Time{}
	}
}

func notifySleeper(sleeper interface{}, now time.Time) {
	switch s := sleeper.(type) {
	case *Timer:
		s.outC <- now
	case *Ticker:
		s.outC <- now
	}

	// Give timers an opportunity to run - helpful if we're in the middle of a
	// long Advance.  Very well-written tests shouldn't rely on timers running
	// immediately at the time they are scheduled and don't need this.
	runtime.Gosched()
	time.Sleep(100 * time.Microsecond)
}

// must be holding mutex when calling
func (c *mockClock) insertSleeper(s interface{}) {
	t := sleeperWhen(s)

	i := sort.Search(len(c.sleepers), func(i int) bool {
		return t.Before(sleeperWhen(c.sleepers[i]))
	})
	c.sleepers = append(c.sleepers, &Timer{})
	copy(c.sleepers[i+1:], c.sleepers[i:])
	c.sleepers[i] = s

	for _, sc := range c.sleepersChanged {
		sc <- len(c.sleepers)
	}
}

// must be holding mutex when calling
func (c *mockClock) removeSleeper(s interface{}) bool {
	t := sleeperWhen(s)

	i := sort.Search(len(c.sleepers), func(i int) bool {
		return !t.Before(sleeperWhen(c.sleepers[i]))
	})
	// i is the first index where t >= c.sleepers[i], but may not be our
	// sleeper (e.g. many sleepers are scheduled at the same instant).
	for ; i < len(c.sleepers); i++ {
		if s == c.sleepers[i] {
			break
		}
	}
	if i == len(c.sleepers) {
		// couldn't find the timer in this list
		return false
	}

	c.sleepers = append(c.sleepers[:i], c.sleepers[i+1:]...)
	for _, sc := range c.sleepersChanged {
		sc <- len(c.sleepers)
	}
	return true
}

// NewTimer creates a new Timer that will send to C after duration
func (c *mockClock) NewTimer(duration time.Duration) *Timer {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	outC := make(chan time.Time)
	t := &Timer{
		C:    outC, // user sees receive-channel
		outC: outC, // we use it as a send-channel
		mock: c,
		when: c.now.Add(duration),
	}
	c.insertSleeper(t)

	return t
}

// NewTicker creates a new Ticker that will send to C every duration interval
func (c *mockClock) NewTicker(duration time.Duration) *Ticker {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	outC := make(chan time.Time)
	t := &Ticker{
		C:      outC, // user sees receive-channel
		outC:   outC, // we use it as a send-channel
		mock:   c,
		when:   c.now.Add(duration),
		period: duration,
	}
	c.insertSleeper(t)

	return t
}

// Advance simulated time by duration, firing any registerd Timers, etc.
func (c *mockClock) Advance(duration time.Duration) {
	// Take advanceMutex and then mutex.  We will drop/retake mutex
	// as we fire each timer, since some timers may register new timers
	c.advanceMutex.Lock()
	defer c.advanceMutex.Unlock()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	until := c.now.Add(duration)

	for {
		if len(c.sleepers) == 0 {
			c.now = until
			return
		}

		headWhen := sleeperWhen(c.sleepers[0])
		if headWhen.After(until) {
			c.now = until
			return
		}

		// Arrange for all our internal data to be correct before dropping the
		// lock to send - the timer may want to register new timers.
		head, remaining := c.sleepers[0], c.sleepers[1:]
		c.sleepers = remaining
		c.now = headWhen

		// Drop the mutex temporarily and notify
		c.mutex.Unlock()
		notifySleeper(head, c.now)
		c.mutex.Lock()

		switch s := head.(type) {
		case *Ticker:
			// Requeue ticker
			if s.mock == c {
				s.when = c.now.Add(s.period)
				c.insertSleeper(s)
			}
		case *Timer:
			// Discard timer, and notify that sleepers changed
			s.mock = nil
			for _, sc := range c.sleepersChanged {
				sc <- len(c.sleepers)
			}
		}
	}
}

func stopMockTimer(t *Timer) bool {
	c := t.mock

	if c == nil {
		// Timer already stopped
		return false
	}
	t.mock = nil

	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.removeSleeper(t)
}

func stopMockTicker(t *Ticker) {
	c := t.mock

	if c == nil {
		// Ticker already stopped
		return
	}
	t.mock = nil

	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.removeSleeper(t)
}

// BlockUntil will not return until the number of pending Timers etc. is exactly n
func (c *mockClock) BlockUntil(n int) {
	changed := make(chan int)

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(c.sleepers) == n {
		return
	}
	c.sleepersChanged = append(c.sleepersChanged, changed)

	c.mutex.Unlock()
	for {
		nowSleepers := <-changed
		if nowSleepers == n {
			c.mutex.Lock()
			for i, x := range c.sleepersChanged {
				if x == changed {
					c.sleepersChanged = append(c.sleepersChanged[:i], c.sleepersChanged[i+1:]...)
					break
				}
			}
			return
		}
	}
}

// A channel that receives after duration
// Equivalent to `c.NewTimer(duration).C`
func (c *mockClock) After(duration time.Duration) <-chan time.Time {
	return c.NewTimer(duration).C
}

// Blocks until duration has elapsed, returns immediately if duration <= 0
// Equivalent to `<-c.NewTimer(duration).C`
func (c *mockClock) Sleep(duration time.Duration) {
	if duration <= 0 {
		return
	}
	<-c.After(duration)
}

// Elasped since t relative to mock time, equivalent to `c.Now().Sub(t)`
func (c *mockClock) Since(t time.Time) time.Duration {
	return c.now.Sub(t)
}

// Time until t relative to mock time, equivalent to `t.Sub(c.Now())`
func (c *mockClock) Until(t time.Time) time.Duration {
	return t.Sub(c.now)
}
