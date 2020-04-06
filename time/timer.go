package time

import (
	"time"
)

type Timer interface {
	Reset(d time.Duration) error
	Cancel() error
	Wait() error
}
