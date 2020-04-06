package testutil

import (
	"errors"
	"sync"
	"time"

	pctime "github.com/stuphlabs/pullcord/time"
)

type FakeTimer struct {
	timer chan interface{}
	stopped bool
	mtx sync.Mutex
}

func (f *FakeTimer) Reset(d time.Duration) bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	return !f.stopped
}

func (f *FakeTimer) Stop() bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	wasStopped := f.stopped
	f.stopped = true
	return wasStopped
}

func (f *FakeTimer) Waiter() <-chan interface{} {
	// this mutex was meant for f.stopped, but yolo
	f.mtx.Lock()
	defer f.mtx.Unlock()

	if nil == f.timer {
		f.timer = make(chan interface{})
	}

	if !f.stopped {
		return f.timer
	}

	return nil
}

func (f *FakeTimer) Trip() error {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	if f.stopped {
		return errors.New("FakeTimer has already been stopped")
	}

	if nil != f.timer {
		close(f.timer)
	}

	f.stopped = true
	return nil
}

var _ pctime.Timer = &FakeTimer{}
