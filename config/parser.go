package config

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/proidiot/gone/log"

	"github.com/stuphlabs/pullcord"
)

// Parser extracts configuration info from an io.Reader.
type Parser struct {
	Reader io.Reader
}

// Server extracts the .../pullcord.Server component from the config io.Reader.
func (p Parser) Server() (pullcord.Server, error) {
	registrationMutex.Lock()
	defer registrationMutex.Unlock()

	var config struct {
		Resources map[string]json.RawMessage
		Server    json.RawMessage
	}

	dec := json.NewDecoder(p.Reader)
	registry = make(map[string]*Resource)

	if e := dec.Decode(&config); e != nil {
		log.Crit(
			fmt.Sprintf(
				"Unable to decode resource: %#v",
				e,
			),
		)
		return nil, e
	}

	unregisterredResources = config.Resources
	for name := range config.Resources {
		log.Debug(fmt.Sprintf("Assessing resource: %s", name))
		if _, present := registry[name]; !present {
			log.Debug(
				fmt.Sprintf(
					"Resource does not already exist in"+
						" the registry: %s",
					name,
				),
			)
			r := new(Resource)
			registry[name] = r
			if e := r.unmarshalByName(name); e != nil {
				return nil, e
			}

			r.complete = true
			log.Debug(
				fmt.Sprintf(
					"Saved resource to registry: %s: %#v",
					name,
					r.Unmarshaled,
				),
			)
		} else {
			log.Debug(
				fmt.Sprintf(
					"Resource already exists in the"+
						" registry: %s",
					name,
				),
			)
		}
	}

	rserver := new(Resource)
	if e := json.Unmarshal(config.Server, rserver); e != nil {
		return nil, e
	}

	if server, ok := rserver.Unmarshaled.(pullcord.Server); ok {
		return server, nil
	}

	err := fmt.Errorf(
		"not a server: %s - %#v",
		config.Server,
		rserver.Unmarshaled,
	)
	log.Crit(err.Error())

	return nil, err
}