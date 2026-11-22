// Package outbox is the order in which a connection's sends and pings go
// out: one at a time, in the order they were asked for.
package outbox

import "context"

// Send is a message or ping waiting in an Outbox; it writes with the
// context it is run with.
type Send func(ctx context.Context) error

// Outbox is the queue of sends of one connection and whether one of them
// is in flight. Both are facts about the same queue, so one value holds
// them and only Enqueue and Done change them. The zero value is an idle,
// empty outbox.
type Outbox struct {
	queue []Send // sends waiting for the one in flight, in order
	busy  bool   // a send is in flight
}

// Enqueue adds send to the queue. It returns the send to start now and
// true when nothing was in flight; otherwise send waits for Done.
func (outbox *Outbox) Enqueue(send Send) (Send, bool) {
	outbox.queue = append(outbox.queue, send)
	if outbox.busy {
		return nil, false
	}
	outbox.busy = true
	return outbox.pop(), true
}

// Done records that the send in flight finished. It returns the next send
// to start and true, or false when the queue is empty and the outbox is
// idle again.
func (outbox *Outbox) Done() (Send, bool) {
	if len(outbox.queue) == 0 {
		outbox.busy = false
		return nil, false
	}
	return outbox.pop(), true
}

func (outbox *Outbox) pop() Send {
	next := outbox.queue[0]
	outbox.queue = outbox.queue[1:]
	return next
}

// Busy reports whether a send is in flight.
func (outbox *Outbox) Busy() bool { return outbox.busy }
