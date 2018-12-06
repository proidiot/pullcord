package trigger

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/proidiot/gone/log"
	"github.com/stuphlabs/pullcord/config"
)

// CompoundTrigger is a TriggerHandler that allows more than one trigger to be
// fired off at a time. It any trigger returns an error, it isn't guaranteed
// that all triggers will fire.
type CompoundTrigger struct {
	Triggers []TriggerHandler
}

func init() {
	config.RegisterResourceType(
		"compoundtrigger",
		func() json.Unmarshaler {
			return new(CompoundTrigger)
		},
	)
}

// UnmarshalJSON implements encoding/json.Unmarshaler.
func (c *CompoundTrigger) UnmarshalJSON(input []byte) error {
	var t struct {
		Triggers []config.Resource
	}

	dec := json.NewDecoder(bytes.NewReader(input))
	if e := dec.Decode(&t); e != nil {
		return e
	}

	c.Triggers = make([]TriggerHandler, len(t.Triggers))

	for _, i := range t.Triggers {
		th := i.Unmarshaled
		switch th := th.(type) {
		case TriggerHandler:
			c.Triggers = append(c.Triggers, th)
		default:
			log.Err(
				fmt.Sprintf(
					"Registry value is not a"+
						" RequestFilter: %s",
					th,
				),
			)
			return config.UnexpectedResourceType
		}
	}

	return nil
}

// Trigger executes all the child triggers, exiting immediately after a single
// failure.
func (c *CompoundTrigger) Trigger() error {
	for _, t := range c.Triggers {
		if err := t.Trigger(); err != nil {
			return err
		}
	}
	return nil
}
