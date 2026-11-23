// Package sends is the store of requests in flight: which request is being
// sent, under which number, and how to stop it. The window owns one store
// and only its UI goroutine touches it; a send goroutine runs the protocol
// and returns a Result on the bus, which the window offers to Accept.
//
// Why sending is a machine apart from the connection of a tab
// (workspace/tabs.Connection): a gRPC tab can have a unary call in flight
// and then switch Stream on, which one merged machine would forbid
// (TestASendInFlightDoesNotForbidAConnect, in the window's tests, since it
// needs both packages).
//
// Why the key is the request and not the tab: this package is a controller
// of feature send and may not import workspace/tabs. One tab shows one
// request (tabs.Set focuses the tab that already shows it), so the request
// names the tab. Why one counter for the store: a number is never reused, so
// the late result of a cancelled send, or of a send of a closed tab, cannot
// match a later send even when the same request opens in a new tab. A
// counter per owner, as a tab had, restarts with the tab.
//
// Rejected: keeping the state in tabs.Tab (it is the domain of sending, not
// of a tab); a generic store keyed by *tabs.Tab (one instantiation that only
// dodges the layer rule); a slice scanned by owner (a map makes "one flight
// per owner" true by construction). Revisit when a request can be open in two
// tabs, when a tab may run two sends at once (a GraphQL schema load beside a
// send), or when a tab's Req pointer is replaced under an open tab.
package sends

import (
	"context"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Number names one send. One counter numbers every send of the store and a
// number is never reused.
type Number uint64

// Result is what a send goroutine returns on the bus. Owner is the request
// of the tab that sent, never the helper request of a GraphQL schema load.
type Result struct {
	Owner    *request.Request
	Number   Number
	Response request.Response
}

// flight is one send in progress.
type flight struct {
	state  SendState
	number Number
	cancel context.CancelFunc
}

// step moves the flight along the transition table, and reports whether
// the event was legal in its state.
func (f *flight) step(event sendEvent) bool {
	to, ok := sendTransition(f.state, event)
	if ok {
		f.state = to
	}
	return ok
}

// Sends holds the sends in flight, at most one per owner. An entry exists
// only while its owner is sending. It is not persisted.
type Sends struct {
	last    Number
	flights map[*request.Request]*flight
}

// New returns a store with no send in flight.
func New() *Sends { return &Sends{flights: map[*request.Request]*flight{}} }

// Begin starts a send of owner and returns its number. It starts none while
// owner is sending. cancel stops the send; Cancel calls it.
func (s *Sends) Begin(owner *request.Request, cancel context.CancelFunc) (Number, bool) {
	if _, sending := s.flights[owner]; sending {
		return 0, false
	}
	started := &flight{}
	if !started.step(sendBegin) {
		return 0, false
	}
	s.last++
	started.number, started.cancel = s.last, cancel
	s.flights[owner] = started
	return started.number, true
}

// Accept reports whether result is still wanted, and if so ends the send of
// its owner. The result of a send that was cancelled, or of an owner that
// started again, is not wanted.
func (s *Sends) Accept(result Result) bool {
	current, sending := s.flights[result.Owner]
	if !sending || current.number != result.Number || !current.step(sendAccept) {
		return false
	}
	delete(s.flights, result.Owner)
	return true
}

// Cancel stops the send of owner by calling its cancel func and ends it, so
// its late result is dropped. It reports whether owner was sending.
func (s *Sends) Cancel(owner *request.Request) bool {
	current, sending := s.flights[owner]
	if !sending || !current.step(sendCancel) {
		return false
	}
	delete(s.flights, owner)
	current.cancel()
	return true
}

// Sending reports whether owner has a send in flight.
func (s *Sends) Sending(owner *request.Request) bool {
	_, sending := s.flights[owner]
	return sending
}
