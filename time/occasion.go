package time

import (
	"time"

	"github.com/proidiot/gone/errors"
)

type PostponedOccasion interface {
	Cancel()
	Postpone(d time.Duration) error
	Wait() error
}

const ErrUnpostponedOccasion = errors.ErrorString(
	"Unable to wait on an occasion which has not been postponed",
)

const ErrCancelledOccasion = errors.ErrorString(
	"The occasion has already been cancelled",
)
