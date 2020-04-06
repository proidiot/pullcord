package trigger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/proidiot/gone/log"
	"github.com/stuphlabs/pullcord/config"
	pctime "github.com/stuphlabs/pullcord/time"
)

// DelayTrigger is a Triggerrer that delays the execution of another
// trigger for at least a minimum amount of time after the most recent request.
// The obvious analogy would be a screen saver, which will start after a
// certain period has elapsed, but the timer is reset quite often.
type DelayTrigger struct {
	DelayedTrigger               Triggerrer
	Delay                        time.Duration
	NewRepeatedOccurrenceDelayer func() pctime.RepeatedOccurrenceDelayer
	delayer                      pctime.RepeatedOccurrenceDelayer
}

func init() {
	config.MustRegisterResourceType(
		"delaytrigger",
		func() json.Unmarshaler {
			return new(DelayTrigger)
		},
	)
}

// UnmarshalJSON implements encoding/json.Unmarshaler.
func (d *DelayTrigger) UnmarshalJSON(input []byte) error {
	var t struct {
		DelayedTrigger config.Resource
		Delay          string
	}

	dec := json.NewDecoder(bytes.NewReader(input))
	if e := dec.Decode(&t); e != nil {
		return e
	}

	dt := t.DelayedTrigger.Unmarshalled
	switch dt := dt.(type) {
	case Triggerrer:
		d.DelayedTrigger = dt
	default:
		_ = log.Err(
			fmt.Sprintf(
				"Registry value is not a Trigger: %s",
				dt,
			),
		)
		return config.UnexpectedResourceType
	}

	dp, e := time.ParseDuration(t.Delay)
	if e != nil {
		return e
	}

	d.Delay = dp
	return nil
}

// Trigger sets or resets the delay after which it will execute the child
// trigger. The child trigger will be executed no sooner than the delay time
// after any particular call, but subsequent calls may extend that time out
// further (possibly indefinitely).
func (d *DelayTrigger) Trigger() error {
	_ = log.Debug("delaytrigger initiated")
	if d.delayer == nil {
		_ = log.Debug("creating delay timer")
		d.delayer = d.NewRepeatedOccurrenceDelayer()
		go func(t Triggerrer, r RepeatedOccurrence) {
			err := r.WaitForNext()
			for nil != err {
				_ = log.Debug("delaytrigger has expired")
				err = tr.Trigger()
				if nil != err {
					_ = log.Err(
						fmt.Sprintf(
							"delaytrigger received"+
								" an error:"+
								" %#v",
							err,
						),
					)
				}
				err = r.WaitForNext()
			}

			_ = log.Errr(
				fmt.Sprintf(
					"delaytrigger unable to wait for next"+
						" occurrence: %#v",
					err,
				),
			)
		}(d.DelayedTrigger, d.delayer)
	}

	_ = log.Debug("setting delay timer")
	d.delayer.DelayForNext(dla)

	_ = log.Debug("delaytrigger completed")
	return nil
}
