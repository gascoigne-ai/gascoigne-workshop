package tabs

// SendState is whether a request of the tab is in flight.
type SendState uint8

const (
	SendIdle SendState = iota
	SendSending
	sendStateCount
)

// sendEvent is something that moves a send from one state to another.
type sendEvent uint8

const (
	sendBegin sendEvent = iota
	sendAccept
	sendCancel
	sendEventCount
)

type sendCell struct {
	to SendState
	ok bool
}

var sendTransitions = [sendStateCount][sendEventCount]sendCell{
	SendIdle:    {sendBegin: {SendSending, true}},
	SendSending: {sendAccept: {SendIdle, true}, sendCancel: {SendIdle, true}},
}

// sendTransition is the state event leads to from state, and false when the
// event is refused there.
func sendTransition(state SendState, event sendEvent) (SendState, bool) {
	cell := sendTransitions[state][event]
	return cell.to, cell.ok
}

// Send is the sending of a tab's request: whether one is in flight and
// which one is current. It is a machine of its own, apart from Connection:
// a gRPC tab can have a unary call in flight and then switch Stream on, which
// one merged machine would forbid (TestASendInFlightDoesNotForbidAConnect).
type Send struct {
	state SendState
	Seq   int // numbers the sends, so a late result of an old one is ignored
}

// Sending reports whether a send is in flight.
func (send *Send) Sending() bool { return send.state == SendSending }

// BeginSend starts a send and returns its number; it does not start one
// while another is running.
func (send *Send) BeginSend() (seq int, started bool) {
	if !send.step(sendBegin) {
		return 0, false
	}
	send.Seq++
	return send.Seq, true
}

// Accept reports whether the answer of send seq is still wanted, and if so
// ends sending. The answer of a send that was cancelled or replaced is not.
func (send *Send) Accept(seq int) bool {
	if seq != send.Seq {
		return false
	}
	return send.step(sendAccept)
}

// cancelSend ends the running send and makes its late answer an old one. It
// reports whether a send was running.
func (send *Send) cancelSend() bool {
	if !send.step(sendCancel) {
		return false
	}
	send.Seq++
	return true
}

func (send *Send) step(event sendEvent) bool {
	to, ok := sendTransition(send.state, event)
	if ok {
		send.state = to
	}
	return ok
}
