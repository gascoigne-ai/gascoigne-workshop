package sends

import "testing"

var legalSend = map[[2]int]SendState{
	{int(SendIdle), int(sendBegin)}:     SendSending,
	{int(SendSending), int(sendAccept)}: SendIdle,
	{int(SendSending), int(sendCancel)}: SendIdle,
}

func TestEverySendTransitionIsLegalOrRefused(t *testing.T) {
	for state := SendState(0); state < sendStateCount; state++ {
		for event := sendEvent(0); event < sendEventCount; event++ {
			want, legal := legalSend[[2]int{int(state), int(event)}]
			got, ok := sendTransition(state, event)
			if ok != legal || (legal && got != want) {
				t.Errorf("state %d event %d: got (%d, %v), want (%d, %v)", state, event, got, ok, want, legal)
			}
		}
	}
}
