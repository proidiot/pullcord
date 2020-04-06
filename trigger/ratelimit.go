package trigger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/proidiot/gone/log"
	"github.com/stuphlabs/pullcord/config"
	pctime "github.com/stuphlabs/pullcord/time"
)

// ErrRateLimitExceeded indicates that the trigger has been called more than
// the allowed number of times within the specified duration, and so the
// guarded trigger will not be called.
var ErrRateLimitExceeded = errors.New("Rate limit exceeded for trigger")

// RateLimitTrigger is a Triggerrer that will prevent a guarded trigger
// from being called more than a specified number of times over a specified
// duration.
type RateLimitTrigger struct {
	GuardedTrigger   Triggerrer
	MaxAllowed       uint
	Period           time.Duration
	NewStopwatch     func() pctime.Stopwatch
	previousTriggers []pctime.Stopwatch
}

func init() {
	config.MustRegisterResourceType(
		"ratelimittrigger",
		func() json.Unmarshaler {
			return new(RateLimitTrigger)
		},
	)
}

// UnmarshalJSON implements encoding/json.Unmarshaler.
func (r *RateLimitTrigger) UnmarshalJSON(input []byte) error {
	var t struct {
		GuardedTrigger config.Resource
		MaxAllowed     uint
		Period         string
	}

	dec := json.NewDecoder(bytes.NewReader(input))
	if e := dec.Decode(&t); e != nil {
		return e
	}

	gt := t.GuardedTrigger.Unmarshalled
	switch gt := gt.(type) {
	case Triggerrer:
		r.GuardedTrigger = gt
	default:
		_ = log.Err(
			fmt.Sprintf(
				"Registry value is not a Trigger: %s",
				gt,
			),
		)
		return config.UnexpectedResourceType
	}

	p, e := time.ParseDuration(t.Period)
	if e != nil {
		return e
	}

	r.Period = p

	r.MaxAllowed = t.MaxAllowed

	return nil
}

/*
// NewRateLimitTrigger initializes a RateLimitTrigger. It may no longer be
// strictly necessary.
func NewRateLimitTrigger(
	guardedTrigger Triggerrer,
	maxAllowed uint,
	period time.Duration,
) *RateLimitTrigger {
	return &RateLimitTrigger{
		guardedTrigger,
		maxAllowed,
		period,
		nil,
	}
}
*/

// Trigger executes its guarded trigger if and only if it has not be called more
// than the allowed number of times within the specified rolling window of time.
// If the rate limit is exceeded, ErrRateLimitExceeded will be returned, and
// the guarded trigger will not be called.
func (r *RateLimitTrigger) Trigger() error {
	_ = log.Debug("rate limit trigger initiated")

	if r.previousTriggers != nil {
		_ = log.Debug("determine if rate limit has been exceeded")
		for len(r.previousTriggers) > 0 {
			pt := r.previousTriggers[0]
			elapsed, err := pt.Elapsed()
			if nil != err {
				panic(err)
			}
			if elapsed > r.Period {
				r.previousTriggers = r.previousTriggers[1:]
			} else {
				break
			}
		}

		if uint(len(r.previousTriggers)) >= r.MaxAllowed {
			_ = log.Debug("rate limit has been exceeded")
			return ErrRateLimitExceeded
		}
	} else {
		_ = log.Debug("first rate limited trigger")
		r.previousTriggers = make([]pctime.Stopwatch, 0)
	}

	if nil == r.NewStopwatch {
		r.NewStopwatch = func() pctime.Stopwatch {
			s := new(pctime.RealStopwatch)
			err := s.Reset()
			if nil != err {
				panic(err)
			}
			return s
		}
	}
	r.previousTriggers = append(r.previousTriggers, r.NewStopwatch())

	_ = log.Debug("rate limit not exceeded, cascading the trigger")
	return r.GuardedTrigger.Trigger()
}
