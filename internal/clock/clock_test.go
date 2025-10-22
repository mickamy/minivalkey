package clock_test

import (
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
)

var (
	base = time.Date(2025, 10, 1, 12, 0, 0, 0, time.UTC)
)

func TestClock_Now(t *testing.T) {
	t.Parallel()

	c := clock.New(base)
	if got := c.Now(); !got.Equal(base) {
		t.Fatalf("Now() = %v, want %v", got, base)
	}
}

func TestClock_Advance(t *testing.T) {
	t.Parallel()

	c := clock.New(base)
	if got := c.Now(); !got.Equal(base) {
		t.Fatalf("Now() = %v, want %v", got, base)
	}
	delta := 5 * time.Second
	want := base.Add(delta)

	if got := c.Advance(delta); !got.Equal(want) {
		t.Fatalf("Advance(%v) = %v, want %v", delta, got, want)
	}
	if got := c.Now(); !got.Equal(want) {
		t.Fatalf("Now() after advance = %v, want %v", got, want)
	}
}

func TestClock_Base(t *testing.T) {
	t.Parallel()

	c := clock.New(base)

	if got := c.Base(); !got.Equal(base) {
		t.Fatalf("Base() = %v, want %v", got, base)
	}
}

func TestClock_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	c := clock.New(base)
	delta := 10 * time.Millisecond
	iterations := 1000
	done := make(chan struct{})

	// start goroutine to advance the clock
	go func() {
		for i := 0; i < iterations; i++ {
			c.Advance(delta)
		}
		done <- struct{}{}
	}()

	// concurrently read the time
	for i := 0; i < iterations; i++ {
		_ = c.Now()
	}

	<-done

	expectedTime := base.Add(time.Duration(iterations) * delta)
	if got := c.Now(); !got.Equal(expectedTime) {
		t.Fatalf("Final Now() = %v, want %v", got, expectedTime)
	}
}
