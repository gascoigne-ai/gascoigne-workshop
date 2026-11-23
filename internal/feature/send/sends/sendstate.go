package sends

// SendState is whether a request of an owner is in flight.
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
