# Tock

[![godoc](https://godoc.org/github.com/aspenmesh/tock?status.svg)](http://godoc.org/github.com/aspenmesh/tock)

a mock clock library for golang

# Usage

First, wherever you use timer functions, use them through the tock interface instead:

```go
import (
  "time"
)

func mySleeper() bool {
  time.Sleep(1 * time.Second)
  return true
}
```

replace with:

```go
import (
  "github.com/aspenmesh/tock"
)

var clock := tock.NewReal()

func mySleeper() bool {
  clock.Sleep(1 * time.Second)
  return true
}
```

This causes no behavior change.  `tock.NewReal()` is a real clock - it passes
through `clock.Sleep` to `time.Sleep` and so on.

Next, in your unit tests, don't use `tock.NewReal()`, instead use `tock.NewMock()`.

For example, in mySleepers.go:
```go
import (
  "github.com/aspenmesh/tock"
)

func mySleeper(clock tock.Clock) {
  clock.Sleep(1 * time.Second)
  return true
}
```

In the real implementation, call like:
```go
  complete := mySleeper(tock.NewReal())
```

In the unit test, you can do:
```go
func TestMySleeper(t *testing.T) {
  c := tock.NewMock()
  go func() {
    c.BlockUntil(1)
    c.Advance(1 * time.Second)
  }()
  complete := mySleeper(c)
  if ! complete {
    t.Errorf("Complete wasn't true")
  }
}
```

# Why?

Using real wall time in unit tests is risky.  Fundamentally, a timer is
guaranteed to fire no-earlier-than the time you ask for, but there is no
guarantee that it won't fire later.  Potentially quite a bit later, especially
when you are running unit tests on a busy CI system.  This problem seems to
exaggerate right before a release for some reason :-) .

Also, you want to keep timers short when you are unit testing so that the unit
tests run fast, but this increases the risk that timers don't fire exactly when
you are expecting them to.  It's also sub-ideal to use different timer values
in unit tests than in the real implementation because you have to remember to
update the "mock" values and convert back and forth between the mock timebase
and the real timebase when debugging.

Instead you can use a mock clock library which lets you implement your own fake
wall clock that you can advance under test conditions.  You get fast unit
tests, and guarantees that timers fire exactly when you expect.

# Why this library?

There were a few existing mock clock libraries for golang.  There were a couple
of things that were important to us:

1. The channel that timers and tickers wait on is named `C`, directly in the
struct, not accessible via interface (like `C()` or `Chan()`).  This matches
the real `time` library.  This reduces the amount of code you have to change to
use tock.

2. Tickers are re-queued directly in the Advance() thread, so if you register a
ticker for every 500 Milliseconds, it is guaranteed to fire at 0.5, 1.0, 1.5,
2.0 seconds of fake wall time.  An alternative implementation that didn't work
for me uses a gofunc to re-register the ticker so it races to re-register and
the firing may not be regular.  (In reality, you shouldn't count on Tickers
firing at exactly 0.5, 1.0, etc but some unit tests do)

3. It is safe for a timer callback to re-register another timer.

4. Timer notification yields using `runtime.Gosched()` and a short
`time.Sleep()` to maximize the likelihood that downstream gofuncs,
notifications, etc happen before moving on.

# What's better?

You may be better off minimizing bare usage of timers at all, and instead try
to use channels whenever possible.  You can use timers to publish to those
channels or cancel contexts but try to constrain the timers to a small area.
This can make it much easier to write robust and fast unit tests.

# Inspired by

This was inspired by jonboulle's great
[clockwork](https://github.com/jonboulle/clockwork) library.
