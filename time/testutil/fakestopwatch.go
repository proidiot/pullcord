package testutil

import (
	"time"

	pctime "github.com/stuphlabs/pullcord/time"
)

type FakeStopwatch struct {
	elapsed time.Duration
}

func (f *FakeStopwatch) Reset() error {
	return nil
}

func (f *FakeStopwatch) Elapsed() (time.Duration, error) {
	return f.elapsed, nil
}

func (f *FakeStopwatch) SetElapsed(d time.Duration) {
	f.elapsed = d
}

var _ pctime.Stopwatch = new(FakeStopwatch)
