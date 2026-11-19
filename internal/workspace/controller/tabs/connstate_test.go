package tabs

import "testing"

// legalConn is every transition the connection allows; a cell that is not
// listed is refused.
var legalConn = map[[2]int]ConnState{
	{int(ConnIdle), int(connBegin)}:            ConnConnecting,
	{int(ConnIdle), int(connDisconnect)}:       ConnIdle,
	{int(ConnConnecting), int(connBegin)}:      ConnConnecting,
	{int(ConnConnecting), int(connOpened)}:     ConnOpen,
	{int(ConnConnecting), int(connFailed)}:     ConnRetrying,
	{int(ConnConnecting), int(connDisconnect)}: ConnIdle,
	{int(ConnOpen), int(connClosed)}:           ConnRetrying,
	{int(ConnOpen), int(connDisconnect)}:       ConnClosing,
	{int(ConnClosing), int(connClosed)}:        ConnIdle,
	{int(ConnClosing), int(connDisconnect)}:    ConnClosing,
	{int(ConnRetrying), int(connBegin)}:        ConnConnecting,
	{int(ConnRetrying), int(connDisconnect)}:   ConnIdle,
}

func TestEveryConnectionTransitionIsLegalOrRefused(t *testing.T) {
	for state := ConnState(0); state < connStateCount; state++ {
		for event := connEvent(0); event < connEventCount; event++ {
			want, legal := legalConn[[2]int{int(state), int(event)}]
			got, ok := transition(state, event)
			if ok != legal || (legal && got != want) {
				t.Errorf("state %d event %d: got (%d, %v), want (%d, %v)", state, event, got, ok, want, legal)
			}
		}
	}
	if len(legalConn) != 12 {
		t.Fatalf("%d legal cells listed, want 12", len(legalConn))
	}
}
