package time

import (
	"time"
)

type RealStopwatch struct {
	LastReset time.Time
}

var _ Stopwatch = new(RealStopwatch)

func (r *RealStopwatch) Reset() error {
	r.LastReset = time.Now()
	return nil
}

func (r *RealStopwatch) Elapsed() (time.Duration, error) {
	return time.Since(r.LastReset), nil
}
