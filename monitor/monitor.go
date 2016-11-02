package monitor

import (
	"github.com/fitstar/falcore"
	"github.com/stuphlabs/pullcord/trigger"
)

// Monitor is an interface that describes a monitorring system which can check
// if a named service is up or not. It is possible that an implementation could
// be made that has no service status caching (in which case the expected
// behavior of Reprobe from the perspective of this interface would be the same
// as Status, and the expected behavior of SetStatusUp from the perspective of
// this interface would be to do nothing), but such an implementation would not
// be generally useful (though perhaps it could be useful in a specific case).
//
// Status is a function that, given a service's name, indicates whether the
// monitor believes the service is up. Depending on the Monitor implementation,
// the status reported by this function could be the result of some level of
// caching, and so this is the preferred way to determine the status of the
// service.
//
// Reprobe is a function that, given a service's name, causes the monitor to
// explicitly check whether the service is up. In a non-caching Monitor
// implementation, this would be identical to Status, but in a caching Monitor
// implementation, this would bypass (and possibly purge) the cache. This
// function should be called if and only if cache bypassing behavior is desired
// when performing a status check on a service.
//
// SetStatusUp is a function that, given a service's name, indicates to a
// caching Monitor implementation that an action has occurred that has the same
// effect as a positive probe (for example, if the underlying service has
// directly responded to a part of the code other than the Monitor system).
// This function would have no effect on a non-caching Monitor implementation.
//
// MonitorFilter is a function that, given a service's name, generates a
// Falcore RequestFilter that forwards the request to the service if the
// service is up. This filter will run one of two specified Triggers depending
// on whether the service is up (presumably in the interest of either bringing
// the service up or keeping the service up). If the service is not up, a page
// will be displayed indicating that fact after the Trigger has been started.
type Monitor interface {
	Status(serviceName string) (up bool, err error)
	Reprobe(serviceName string) (up bool, err error)
	SetStatusUp(serviceName string) error
	MonitorFilter(
		serviceName string,
		onUp *trigger.TriggerHandler,
		onDown *trigger.TriggerHandler,
	) (*falcore.RequestFilter, error)
}

