package time

import (
	"sync"
	"time"
)

type RealTimer struct {
	_ [0]func()
	timer *time.Timer
	expired chan bool
	mtx *sync.Mutex
}

func (r *RealTimer) Reset(d time.Duration) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if !r.cancelled {
		if nil != r.timer {

		}
	}
	return r.Timer.Reset(d)
}

func (r *RealTimer) Cancel() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if nil != r.timer {
		if !r.timer.Stop() {
			<-r.timer.C
		}
	}

	r.cancelled = true

	return nil
}

func (r *RealTimer) Stop() bool {
	return r.Timer.Stop()
}

func (r *RealTimer) Waiter() <-chan interface{} {
	out := make(chan interface{})

	go func(outchan chan<- interface{}, inchan <-chan time.Time) {
		outchan <-inchan
	}(out, r.Timer.C)

	return out
}

var _ Timer = &RealTimer{}

func NewTimer(d time.Duration) Timer {
	return &RealTimer{
		Timer: time.NewTimer(d),
	}
}
