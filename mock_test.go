package tock

import (
	"sync"
	"testing"
	"time"
)

func TestNewTimer(t *testing.T) {
	c := NewMock()

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
	c := NewMock()

	var firedMutex sync.Mutex
	fired := []int{}
	t1 := c.NewTimer(3 * time.Second)
	t2 := c.NewTicker(2 * time.Second)
	t3 := c.NewTimer(1 * time.Second)

	done := make(chan struct{})
	defer close(done)
	go func() {
		// We will replace this with a real channel once t4 is registered.  This
		// stub channel is just so that the select loop has something to wait on
		// before t4 is set up.
		t4Chan := make(<-chan time.Time)

		for {
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
			case <-done:
				return
			}
		}
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

	expectFired("start")
	c.Advance(500 * time.Millisecond)
	expectFired("before any")
	c.Advance(500 * time.Millisecond)
	expectFired("stage 1", 3)
	c.Advance(1 * time.Second)
	expectFired("stage 2", 3, 2, 4)
	c.Advance(1 * time.Second)
	expectFired("stage 3", 3, 2, 4, 1)
	c.Advance(1 * time.Second)
	expectFired("stage 4", 3, 2, 4, 1, 2)
}

func TestBlockUntil(t *testing.T) {
	c := NewMock()

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
