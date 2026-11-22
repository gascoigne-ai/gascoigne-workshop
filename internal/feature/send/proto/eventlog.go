package proto

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"sync"
	"sync/atomic"
	"time"
)

// eventLog is the queue of events a streaming session shows: one writer
// side that waits for room, and a close that never waits.
type eventLog struct {
	events chan request.Event
	done   chan struct{} // closed by close: frees an emit that waits for room
	now    func() time.Time

	stop   sync.Once  // closes done; close must not need gate, which emit holds while it waits
	gate   sync.Mutex // guards ended against a send on the closed events channel
	ended  atomic.Bool
	closed atomic.Bool // the user closed the session, so its end is no loss
}

func newEventLog() *eventLog {
	return &eventLog{events: make(chan request.Event, eventBuffer), done: make(chan struct{}), now: time.Now}
}

// Events implements Session.
func (log *eventLog) Events() <-chan request.Event { return log.events }

// emit queues event, waiting for a slow consumer, unless the log was
// closed or has ended.
func (log *eventLog) emit(event request.Event) {
	event.Time = log.now()
	log.gate.Lock()
	defer log.gate.Unlock()
	if log.ended.Load() {
		return
	}
	select {
	case log.events <- event:
	case <-log.done:
	}
}

// finish ends the event stream; more than one call is harmless.
func (log *eventLog) finish() {
	log.gate.Lock()
	defer log.gate.Unlock()
	if log.ended.CompareAndSwap(false, true) {
		close(log.events)
	}
}

// close marks the user's close and frees an emit that waits for room.
func (log *eventLog) close() {
	log.closed.Store(true)
	log.stopWaiting()
}

// stopWaiting frees an emit that waits for room, without calling it a
// user close.
func (log *eventLog) stopWaiting() { log.stop.Do(func() { close(log.done) }) }

// closing reports that the user closed the session.
func (log *eventLog) closing() bool { return log.closed.Load() }
