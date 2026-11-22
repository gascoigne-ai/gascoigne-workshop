package tabs

import (
	"context"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

type fakeSession struct{ closed bool }

func (session *fakeSession) Events() <-chan request.Event       { return nil }
func (session *fakeSession) Send(context.Context, string) error { return nil }
func (session *fakeSession) Ping(context.Context) error         { return nil }
func (session *fakeSession) Close(context.Context) error        { session.closed = true; return nil }

func TestBeginMakesTheConnectionWantedAndNumbersIt(t *testing.T) {
	var conn Connection
	if conn.Active() {
		t.Fatal("a new connection is not active")
	}
	first, _ := conn.Begin()
	second, ok := conn.Begin()
	if !conn.Active() || conn.State() != ConnConnecting || !ok || second != first+1 {
		t.Fatalf("conn %+v first %d second %d", conn, first, second)
	}
	if !conn.Current(second) || conn.Current(first) {
		t.Fatal("only the latest connect is current")
	}
}

func TestDisconnectAbandonsAConnectInProgressAndDropsItsAnswer(t *testing.T) {
	var conn Connection
	gen, _ := conn.Begin()
	conn.LastEventID = "7"
	session, abandoned := conn.Disconnect()
	if session != nil || !abandoned || conn.State() != ConnIdle || conn.Active() || conn.LastEventID != "" {
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
	if session != request.Session(open) || abandoned {
		t.Fatalf("session %v abandoned %v", session, abandoned)
	}
}

// A failed connect used to leave WantOpen set, so the button kept reading
// "Disconnect" with nothing open.
func TestFailedConnectIsNotActiveUnlessAReconnectIsAsked(t *testing.T) {
	tests := []struct {
		name      string
		reconnect bool
		active    bool
	}{
		{"reconnect off", false, false},
		{"reconnect on keeps retrying", true, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn := Connection{Reconnect: test.reconnect}
			conn.Begin()
			conn.Failed()
			if conn.State() == ConnConnecting || conn.Active() != test.active {
				t.Fatalf("conn %+v Active = %v, want %v", conn, conn.Active(), test.active)
			}
		})
	}
}

func TestClosedClearsTheSessionAndWhatTheServerAnnounced(t *testing.T) {
	var conn Connection
	conn.Begin()
	conn.Opened(&fakeSession{})
	conn.Tools = []request.MCPTool{{Name: "t"}}
	conn.Resources = []request.MCPResource{{Name: "r"}}
	conn.Prompts = []request.MCPPrompt{{Name: "p"}}
	conn.Closed()
	if conn.Session() != nil || conn.State() == ConnConnecting || conn.Tools != nil || conn.Resources != nil || conn.Prompts != nil {
		t.Fatalf("conn %+v", conn)
	}
}

// Regression: after the server closed an open session with reconnect off,
// the connection stayed wanted, so the button read Disconnect and a retry
// was offered although nothing asked for one.
func TestClosedByTheServerWithoutReconnectEndsInactive(t *testing.T) {
	var conn Connection
	conn.Begin()
	conn.Opened(&fakeSession{})
	conn.Closed()
	if conn.Active() || conn.Retrying() {
		t.Fatalf("Active = %v, Retrying = %v, want both false: %+v", conn.Active(), conn.Retrying(), conn)
	}
}

func TestClosedWithReconnectKeepsTheRetryPending(t *testing.T) {
	conn := Connection{Reconnect: true}
	conn.Begin()
	conn.Opened(&fakeSession{})
	conn.Closed()
	if !conn.Active() || !conn.Retrying() {
		t.Fatalf("conn %+v", conn)
	}
}

func TestReceivedRemembersTheEventIDOfAReceivedEventOnly(t *testing.T) {
	var conn Connection
	conn.Received(request.Event{Dir: request.Sent, ID: "1"})
	if conn.LastEventID != "" {
		t.Fatal("a sent event has no id to resume from")
	}
	conn.Received(request.Event{Dir: request.Received, ID: "2"})
	if conn.LastEventID != "2" {
		t.Fatalf("LastEventID = %q", conn.LastEventID)
	}
}

func TestRetryingIsTheDroppedConnectionTheUserStillWants(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Connection)
		retry bool
	}{
		{"closed by the server and reconnect on", func(conn *Connection) {
			conn.Reconnect = true
			conn.Begin()
			conn.Opened(&fakeSession{})
			conn.Closed()
		}, true},
		{"reconnect is off", func(conn *Connection) { conn.Begin(); conn.Opened(&fakeSession{}); conn.Closed() }, false},
		{"the user disconnected while retrying", func(conn *Connection) {
			conn.Reconnect = true
			conn.Begin()
			conn.Failed()
			conn.Disconnect()
		}, false},
		{"already connecting again", func(conn *Connection) {
			conn.Reconnect = true
			conn.Begin()
			conn.Failed()
			conn.Begin()
		}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var conn Connection
			test.setup(&conn)
			if got := conn.Retrying(); got != test.retry {
				t.Errorf("Retrying = %v, want %v", got, test.retry)
			}
		})
	}
}

// connIn is a connection brought to state by the events that lead there.
func connIn(t *testing.T, state ConnState) Connection {
	t.Helper()
	conn := Connection{Reconnect: true}
	steps := map[ConnState]func(){
		ConnIdle:       func() {},
		ConnConnecting: func() { conn.Begin() },
		ConnOpen:       func() { conn.Begin(); conn.Opened(&fakeSession{}) },
		ConnClosing:    func() { conn.Begin(); conn.Opened(&fakeSession{}); conn.Disconnect() },
		ConnRetrying:   func() { conn.Begin(); conn.Failed() },
	}
	steps[state]()
	if conn.State() != state {
		t.Fatalf("setup reached %d, want %d", conn.State(), state)
	}
	return conn
}

func TestARefusedEventChangesNothing(t *testing.T) {
	tests := []struct {
		name  string
		state ConnState
		do    func(*Connection) bool
	}{
		{"Opened on idle", ConnIdle, func(conn *Connection) bool { return conn.Opened(&fakeSession{}) }},
		{"Failed on idle", ConnIdle, func(conn *Connection) bool { return conn.Failed() }},
		{"Closed on idle", ConnIdle, func(conn *Connection) bool { return conn.Closed() }},
		{"Closed on connecting", ConnConnecting, func(conn *Connection) bool { return conn.Closed() }},
		{"Begin on open", ConnOpen, func(conn *Connection) bool { _, ok := conn.Begin(); return ok }},
		{"Opened on open", ConnOpen, func(conn *Connection) bool { return conn.Opened(&fakeSession{}) }},
		{"Failed on open", ConnOpen, func(conn *Connection) bool { return conn.Failed() }},
		{"Begin on closing", ConnClosing, func(conn *Connection) bool { _, ok := conn.Begin(); return ok }},
		{"Failed on closing", ConnClosing, func(conn *Connection) bool { return conn.Failed() }},
		{"Closed on retrying", ConnRetrying, func(conn *Connection) bool { return conn.Closed() }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn := connIn(t, test.state)
			gen, session := conn.Gen, conn.Session()
			conn.Tools = []request.MCPTool{{Name: "t"}}
			if test.do(&conn) {
				t.Fatal("the event was accepted")
			}
			if conn.State() != test.state || conn.Gen != gen || conn.Session() != session || conn.Tools == nil {
				t.Fatalf("state %d gen %d: conn %+v", test.state, gen, conn)
			}
		})
	}
}

// The session is there exactly while the connection is open or closing.
func TestTheSessionExistsExactlyInOpenAndClosing(t *testing.T) {
	events := map[string]func(*Connection){
		"begin":      func(conn *Connection) { conn.Begin() },
		"opened":     func(conn *Connection) { conn.Opened(&fakeSession{}) },
		"failed":     func(conn *Connection) { conn.Failed() },
		"closed":     func(conn *Connection) { conn.Closed() },
		"disconnect": func(conn *Connection) { conn.Disconnect() },
	}
	for state := ConnIdle; state < connStateCount; state++ {
		for name, event := range events {
			for _, reconnect := range []bool{false, true} {
				conn := connIn(t, state)
				conn.Reconnect = reconnect
				event(&conn)
				hasSession := conn.Session() != nil
				wantSession := conn.State() == ConnOpen || conn.State() == ConnClosing
				if hasSession != wantSession {
					t.Errorf("%s from state %d (reconnect %v): state %d, session %v", name, state, reconnect, conn.State(), hasSession)
				}
			}
		}
	}
}

func TestADropWithoutReconnectGoesIdleNotRetrying(t *testing.T) {
	for _, state := range []ConnState{ConnConnecting, ConnOpen} {
		conn := connIn(t, state)
		conn.Reconnect = false
		if state == ConnConnecting {
			conn.Failed()
		} else {
			conn.Closed()
		}
		if conn.State() != ConnIdle {
			t.Errorf("from state %d: state %d, want idle", state, conn.State())
		}
	}
}

// A closing connection ends idle even with reconnect on: the user closed it.
func TestAClosedConnectionTheUserEndedDoesNotRetry(t *testing.T) {
	conn := connIn(t, ConnClosing)
	conn.Closed()
	if conn.State() != ConnIdle {
		t.Fatalf("state %d", conn.State())
	}
}

func TestReceivedKeepsTheServerInfoUntilTheSessionEnds(t *testing.T) {
	var conn Connection
	conn.Received(request.Event{Dir: request.Info, Kind: request.KindInitialize, Text: `{"protocolVersion":"2025-06-18","serverInfo":{"name":"mock","version":"1"}}`})
	if conn.Server == nil || conn.Server.Name != "mock" {
		t.Fatalf("Server = %+v", conn.Server)
	}
	conn.Begin()
	conn.Opened(&fakeSession{})
	conn.Closed()
	if conn.Server != nil {
		t.Fatalf("Server after close = %+v", conn.Server)
	}
}

// Pins that only the session's own initialize note describes the server:
// a server-sent event that carries that name and JSON does not.
func TestReceivedTakesServerInfoOnlyFromTheInitializeNote(t *testing.T) {
	text := `{"protocolVersion":"2025-06-18","serverInfo":{"name":"mock","version":"1"}}`
	var conn Connection
	conn.Received(request.Event{Dir: request.Received, Kind: request.KindMessage, Name: "initialize", Text: text})
	if conn.Server != nil {
		t.Fatalf("an event named initialize set Server = %+v", conn.Server)
	}
	conn.Received(request.Event{Dir: request.Info, Kind: request.KindInitialize, Text: text})
	if conn.Server == nil {
		t.Fatal("the initialize note did not set Server")
	}
}

// A WebSocket echo shaped like a tools list used to fill Tools, because
// Received parsed the text of every event of every protocol.
func TestAWebSocketEchoShapedLikeAToolsListLeavesToolsAlone(t *testing.T) {
	var conn Connection
	conn.Received(request.Event{Dir: request.Received, Kind: request.KindText, Text: `{"result":{"tools":[{"name":"echo"}]}}`})
	if conn.Tools != nil {
		t.Fatalf("Tools = %+v", conn.Tools)
	}
}

func TestReceivedTakesTheListsOfThePayload(t *testing.T) {
	var conn Connection
	conn.Received(request.Event{Dir: request.Received, Payload: request.ToolsList{Tools: []request.MCPTool{{Name: "a"}}}})
	conn.Received(request.Event{Dir: request.Received, Payload: request.ResourcesList{Resources: []request.MCPResource{{URI: "file:///a"}}}})
	conn.Received(request.Event{Dir: request.Received, Payload: request.PromptsList{Prompts: []request.MCPPrompt{{Name: "p"}}}})
	if len(conn.Tools) != 1 || len(conn.Resources) != 1 || len(conn.Prompts) != 1 {
		t.Fatalf("conn %+v", conn)
	}
}

func TestAnEmptyToolsListClearsTheTools(t *testing.T) {
	conn := Connection{Tools: []request.MCPTool{{Name: "a"}}}
	conn.Received(request.Event{Dir: request.Received, Payload: request.ToolsList{}})
	if len(conn.Tools) != 0 {
		t.Fatalf("Tools = %+v", conn.Tools)
	}
}
