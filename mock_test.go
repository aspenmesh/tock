package tock

import (
	"sync"
	"testing"
	"time"
)

var defaultOpts MockOptions = MockOptions{Yield: true}

func TestNewTimer(t *testing.T) {
	c := NewMock(defaultOpts)

	for _, d := range []time.Duration{
		5 * time.Second,
		10 * time.Second,
		7 * time.Second,
		3 * time.Second,
		20 * time.Second,
		10 * time.Second,
	} {
		c.NewTimer(d)
	}

	expected := []time.Duration{
		3 * time.Second,
		5 * time.Second,
		7 * time.Second,
		10 * time.Second,
		10 * time.Second,
		20 * time.Second,
	}
	if len(c.sleepers) != len(expected) {
		t.Errorf("Unexpected sleepers: %d", len(c.sleepers))
	}
	for i, d := range expected {
		if sleeperWhen(c.sleepers[i]) != c.Now().Add(d) {
			t.Errorf("Unexpected sleepers[%d]: %v", i, sleeperWhen(c.sleepers[i]))
		}
	}
}

func TestAdvance(t *testing.T) {
	c := NewMock(defaultOpts)

	var firedMutex sync.Mutex
	fired := []int{}
	t1 := c.NewTimer(3 * time.Second)
	t2 := c.NewTicker(2 * time.Second)
	t3 := c.NewTimer(1 * time.Second)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// We will replace this with a real channel once t4 is registered.  This
		// stub channel is just so that the select loop has something to wait on
		// before t4 is set up.
		t4Chan := make(<-chan time.Time)

		for firingsLeft := 5; firingsLeft > 0; firingsLeft-- {
			select {
			case <-t1.C:
				firedMutex.Lock()
				fired = append(fired, 1)
				firedMutex.Unlock()
			case <-t2.C:
				firedMutex.Lock()
				fired = append(fired, 2)
				firedMutex.Unlock()
			case <-t3.C:
				firedMutex.Lock()
				fired = append(fired, 3)
				firedMutex.Unlock()
				t4 := c.NewTimer(1 * time.Second)
				t4Chan = t4.C
			case <-t4Chan:
				firedMutex.Lock()
				fired = append(fired, 4)
				firedMutex.Unlock()
			}
		}
		wg.Done()
	}()

	expectFired := func(stage string, nums ...int) {
		firedMutex.Lock()
		defer firedMutex.Unlock()
		if len(nums) != len(fired) {
			t.Errorf("In %s, expected %d fired found %d (%v)", stage, len(nums), len(fired), fired)
			return
		}
		for i, n := range nums {
			if fired[i] != n {
				t.Errorf("In %s, expected fired[%d] to be %d and it was %d", stage, i, n, fired[i])
			}
		}
	}

	c.Advance(500 * time.Millisecond)
	c.Advance(500 * time.Millisecond)
	c.Advance(3 * time.Second)
	wg.Wait()
	expectFired("done", 3, 2, 4, 1, 2)
}

func TestBlockUntil(t *testing.T) {
	c := NewMock(defaultOpts)

	c.BlockUntil(0)
	t1 := c.NewTimer(3 * time.Second)
	c.BlockUntil(1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.BlockUntil(2)
		c.Advance(4 * time.Second)
		c.BlockUntil(0)
		wg.Done()
	}()
	t2 := c.NewTimer(5 * time.Second)
	<-t1.C
	t2.Stop()
	wg.Wait()
}

func TestAfter(t *testing.T) {
	c := NewMock(defaultOpts)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-c.After(1 * time.Second)
		c.BlockUntil(0)
		wg.Done()
	}()
	c.BlockUntil(1)
	c.Advance(1 * time.Second)
	wg.Wait()
	c.BlockUntil(0)
}

func TestSleep(t *testing.T) {
	c := NewMock(defaultOpts)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.Sleep(1 * time.Second)
		c.BlockUntil(0)
		wg.Done()
	}()
	c.BlockUntil(1)
	c.Advance(500 * time.Millisecond)
	c.Advance(500 * time.Millisecond)
	wg.Wait()
	c.BlockUntil(0)
}

func TestSince(t *testing.T) {
	c := NewMock(defaultOpts)
	d := time.Date(1, time.January, 1, 0, 0, 5, 0, time.UTC)
	s := c.Since(d)
	if s != -5*time.Second {
		t.Errorf("Expected -5 seconds since zero time, got %v", s)
	}
	c.Advance(5 * time.Second)
	s = c.Since(d)
	if s != 0 {
		t.Errorf("Expected 0 seconds since 5-second time, got %v", s)
	}
	c.Advance(24 * time.Hour)
	s = c.Since(d)
	if s != 24*time.Hour {
		t.Errorf("Expected 24 hours since 24 hour time, got %v", s)
	}
}

func TestUntil(t *testing.T) {
	c := NewMock(defaultOpts)
	d := time.Date(1, time.January, 1, 0, 0, 5, 0, time.UTC)
	u := c.Until(d)
	if u != 5*time.Second {
		t.Errorf("Expected 5 seconds until from zero time, got %v", u)
	}
	c.Advance(5 * time.Second)
	u = c.Until(d)
	if u != 0 {
		t.Errorf("Expected 0 seconds until from 5-second time, got %v", u)
	}
	c.Advance(24 * time.Hour)
	u = c.Until(d)
	if u != -24*time.Hour {
		t.Errorf("Expected -24 hours until from 24 hour time, got %v", u)
	}
}
