package time

import (
	"sync"
	"time"
)

type RealRepeatedOccurrenceDelayer struct {
	_    [0]func()
	tmr  *time.Timer
	next chan struct{}
	mtx  sync.Mutex
}

var _ RepeatedOccurrenceDelayer = new(RealRepeatedOccurrenceDelayer)

func (r *RealRepeatedOccurrenceDelayer) WaitForNext() error {
	var waiter <-chan struct{}
	r.mtx.Lock()
	if nil == r.next {
		r.next = make(chan struct{})
	}
	waiter = r.next
	r.mtx.Unlock()
	<-waiter
	return nil
}

func (r *RealRepeatedOccurrenceDelayer) channelReset() {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if nil != r.next {
		close(r.next)
	}
	r.next = make(chan struct{})
}

func (r *RealRepeatedOccurrenceDelayer) DelayForNext(d time.Duration) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if nil != r.tmr {
		if !r.tmr.Stop() {
			<-r.tmr.C
		}
	}
	if nil == r.next {
		r.next = make(chan struct{})
	}
	r.tmr = time.AfterFunc(d, func() {
		r.channelReset()
	})
	return nil
}
