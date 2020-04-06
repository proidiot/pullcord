package time

import (
	"time"
)

type Stopwatch interface {
	Reset() error
	Elapsed() (time.Duration, error)
}
