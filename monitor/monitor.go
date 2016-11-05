package monitor

import (
	"github.com/fitstar/falcore"
	"github.com/stuphlabs/pullcord/trigger"
)

// Monitor is an interface that describes a system which can check if a service
// is up or not. It is possible that an implementation could be made that has
// no service status caching, but such an implementation might not be generally
// useful.
//
// Status is a function that indicates whether the monitor believes the service
// is up. Depending on the Monitor implementation, the status reported by this
// function could be the result of some level of caching.
//
// MonitorFilter is a function that generates a Falcore RequestFilter that
// forwards the request to the service if the service is up. This filter will
// run a specified Trigger on depending on whether the service is up (presumably in
// the interest of either bringing the service up).If the service is not up, a page
// will be displayed indicating that fact after the Trigger has been started.
type Monitor interface {
	Status() (up bool, err error)
	MonitorFilter(
		ifUp *trigger.TriggerHandler,
		ifDown *trigger.TriggerHandler,
		always *trigger.TriggerHandler,
	) (*falcore.RequestFilter, error)
}

