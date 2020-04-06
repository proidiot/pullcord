package time

import (
	"time"
)

type SnoozeAlarm interface {
	Unplug() error
	Snooze(d time.Duration) error
	TakeNap() Nap
}
