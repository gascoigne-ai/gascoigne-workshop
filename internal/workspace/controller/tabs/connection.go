package tabs

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

// Connection is the state of a tab that keeps a connection open (WebSocket,
// SSE, MCP): where it is in its life (ConnState), and what the server
// announced over it. Its state and session are private: they move only
// through the events below, by the table in connstate.go. The session is
// there exactly while the connection is open or closing.
type Connection struct {
	state       ConnState
	session     port.Session // the open connection, nil unless open or closing
	Gen         int          // numbers the connects, so a late answer of an old one is ignored
	Reconnect   bool         // reopen a dropped connection; the user's policy, read when one drops
	LastEventID string       // the last id an SSE stream sent, offered again when it reconnects

	Resources []model.MCPResource // the resources an MCP server announced
	Prompts   []model.MCPPrompt   // the prompts an MCP server announced
	Tools     []model.MCPTool     // the tools an MCP server announced
	Server    *model.MCPServer    // what an MCP server said about itself, nil before it did
}

// State is where the connection is in its life.
func (conn *Connection) State() ConnState { return conn.state }

// Session is the open connection, nil when there is none.
func (conn *Connection) Session() port.Session { return conn.session }

// Active reports whether the connection is up, being made or due to be
// retried, so the next click on its button disconnects instead of connecting.
func (conn *Connection) Active() bool { return conn.state != ConnIdle }

// Retrying reports whether a dropped connection waits to be reopened.
func (conn *Connection) Retrying() bool { return conn.state == ConnRetrying }

// step moves the connection by event and reports whether the table allows
// it; a refused event changes nothing. A drop that the tab does not want
// retried ends idle.
func (conn *Connection) step(event connEvent) bool {
	to, ok := transition(conn.state, event)
	if !ok {
		return false
	}
	if to == ConnRetrying && !conn.Reconnect {
		to = ConnIdle
	}
	conn.state = to
	if to != ConnOpen && to != ConnClosing {
		conn.session = nil
	}
	return true
}

// Begin starts a connect and returns its number. It is refused while the
// connection is open or closing.
func (conn *Connection) Begin() (gen int, ok bool) {
	if !conn.step(connBegin) {
		return 0, false
	}
	conn.Gen++
	return conn.Gen, true
}

// Current reports whether gen is the latest connect; an answer of an older
// one is dropped.
func (conn *Connection) Current(gen int) bool { return gen == conn.Gen }

// Opened records the session a connect produced. It reports false, and the
// caller closes the session, when no connect was in progress.
func (conn *Connection) Opened(session port.Session) bool {
	if !conn.step(connOpened) {
		return false
	}
	conn.session = session
	return true
}

// Failed records that a connect did not succeed. Without a reconnect asked
// for the connection is no longer wanted, so the tab offers Connect again
// instead of a Disconnect that closes nothing.
func (conn *Connection) Failed() bool { return conn.step(connFailed) }

// Closed records that the session ended: nothing is open and what the
// server announced no longer holds. Without a reconnect asked for, or after
// a Disconnect, the connection is no longer wanted, as in Failed.
func (conn *Connection) Closed() bool {
	if !conn.step(connClosed) {
		return false
	}
	conn.Tools, conn.Resources, conn.Prompts, conn.Server = nil, nil, nil, nil
	return true
}

// Disconnect records that the user no longer wants the connection. It
// returns the open session, which the caller closes, or reports that a
// connect in progress was abandoned, whose answer is dropped from now on.
func (conn *Connection) Disconnect() (session port.Session, abandoned bool) {
	conn.LastEventID = "" // a deliberate disconnect starts the stream over next time
	wasConnecting := conn.state == ConnConnecting
	session = conn.session
	if !conn.step(connDisconnect) {
		return nil, false
	}
	if wasConnecting {
		conn.Gen++
	}
	return session, wasConnecting
}

// Received takes what a received event announces: the server's own
// description, its tools, resources and prompts, and the id to resume an event stream from.
func (conn *Connection) Received(event port.Event) {
	if event.Dir == port.Info && event.Kind == port.KindInitialize {
		if server, ok := model.ParseServerInfo(event.Text); ok {
			conn.Server = &server
		}
		return
	}
	if event.Dir != port.Received {
		return
	}
	if tools := model.ParseTools(event.Text); tools != nil {
		conn.Tools = tools
	}
	if event.ID != "" {
		conn.LastEventID = event.ID
	}
	if resources := model.ParseResources(event.Text); resources != nil {
		conn.Resources = resources
	}
	if prompts := model.ParsePrompts(event.Text); prompts != nil {
		conn.Prompts = prompts
	}
}
