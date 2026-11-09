package tabs

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

type fakeSession struct{ closed bool }

func (session *fakeSession) Events() <-chan port.Event { return nil }
func (session *fakeSession) Send(string) error         { return nil }
func (session *fakeSession) Ping() error               { return nil }
func (session *fakeSession) Close() error              { session.closed = true; return nil }

func TestBeginMakesTheConnectionWantedAndNumbersIt(t *testing.T) {
	var conn Connection
	if conn.Active() {
		t.Fatal("a new connection is not active")
	}
	first := conn.Begin()
	second := conn.Begin()
	if !conn.Active() || !conn.Connecting || !conn.WantOpen || second != first+1 {
		t.Fatalf("conn %+v first %d second %d", conn, first, second)
	}
	if !conn.Current(second) || conn.Current(first) {
		t.Fatal("only the latest connect is current")
	}
}

func TestDisconnectAbandonsAConnectInProgressAndDropsItsAnswer(t *testing.T) {
	var conn Connection
	gen := conn.Begin()
	conn.LastEventID = "7"
	session, abandoned := conn.Disconnect()
	if session != nil || !abandoned || conn.Connecting || conn.WantOpen || conn.LastEventID != "" {
		t.Fatalf("session %v abandoned %v conn %+v", session, abandoned, conn)
	}
	if conn.Current(gen) {
		t.Fatal("the answer of the abandoned connect must be dropped")
	}
}

func TestDisconnectHandsBackTheOpenSessionToClose(t *testing.T) {
	var conn Connection
	conn.Begin()
	open := &fakeSession{}
	conn.Opened(open)
	session, abandoned := conn.Disconnect()
	if session != port.Session(open) || abandoned {
		t.Fatalf("session %v abandoned %v", session, abandoned)
	}
}

func TestClosedClearsTheSessionAndWhatTheServerAnnounced(t *testing.T) {
	var conn Connection
	conn.Begin()
	conn.Opened(&fakeSession{})
	conn.Tools = []model.MCPTool{{Name: "t"}}
	conn.Resources = []model.MCPResource{{Name: "r"}}
	conn.Prompts = []model.MCPPrompt{{Name: "p"}}
	conn.Closed()
	if conn.Session != nil || conn.Connecting || conn.Tools != nil || conn.Resources != nil || conn.Prompts != nil {
		t.Fatalf("conn %+v", conn)
	}
}

func TestReceivedRemembersTheEventIDOfAReceivedEventOnly(t *testing.T) {
	var conn Connection
	conn.Received(port.Event{Dir: port.Sent, ID: "1"})
	if conn.LastEventID != "" {
		t.Fatal("a sent event has no id to resume from")
	}
	conn.Received(port.Event{Dir: port.Received, ID: "2"})
	if conn.LastEventID != "2" {
		t.Fatalf("LastEventID = %q", conn.LastEventID)
	}
}

func TestRetryNeedsAWantedConnectionThatIsDown(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Connection)
		retry bool
		due   bool
	}{
		{"closed by the server and wanted", func(conn *Connection) { conn.Reconnect = true; conn.WantOpen = true }, true, true},
		{"reconnect is off", func(conn *Connection) { conn.WantOpen = true }, true, false},
		{"the user disconnected", func(conn *Connection) { conn.Reconnect = true }, false, false},
		{"already connected again", func(conn *Connection) { conn.Reconnect, conn.WantOpen, conn.Session = true, true, &fakeSession{} }, false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var conn Connection
			test.setup(&conn)
			if got := conn.CanRetry(); got != test.retry {
				t.Errorf("CanRetry = %v, want %v", got, test.retry)
			}
			if got := conn.RetryWanted(); got != test.due {
				t.Errorf("RetryWanted = %v, want %v", got, test.due)
			}
		})
	}
}
