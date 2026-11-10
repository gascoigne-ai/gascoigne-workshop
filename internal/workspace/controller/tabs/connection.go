package tabs

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

// Connection is the state of a tab that keeps a connection open (WebSocket,
// SSE, MCP): whether it is wanted, being made or up, and what the server
// announced over it.
type Connection struct {
	Session     port.Session // the open connection, nil when there is none
	Connecting  bool         // a connect is in progress
	Gen         int          // numbers the connects, so a late answer of an old one is ignored
	WantOpen    bool         // the user wants the connection open, so a drop is retried
	Reconnect   bool         // reopen a dropped connection
	LastEventID string       // the last id an SSE stream sent, offered again when it reconnects

	Resources []model.MCPResource // the resources an MCP server announced
	Prompts   []model.MCPPrompt   // the prompts an MCP server announced
	Tools     []model.MCPTool     // the tools an MCP server announced
	Server    *model.MCPServer    // what an MCP server said about itself, nil before it did
}

// Active reports whether the connection is up, being made or wanted, so the
// next click on its button disconnects instead of connecting.
func (conn *Connection) Active() bool { return conn.Session != nil || conn.Connecting || conn.WantOpen }

// Begin starts a connect and returns its number.
func (conn *Connection) Begin() int {
	conn.WantOpen = true
	conn.Gen++
	conn.Connecting = true
	return conn.Gen
}

// Current reports whether gen is the latest connect; an answer of an older
// one is dropped.
func (conn *Connection) Current(gen int) bool { return gen == conn.Gen }

// Opened records the session a connect produced.
func (conn *Connection) Opened(session port.Session) {
	conn.Session, conn.Connecting = session, false
}

// Failed records that a connect did not succeed. Without a reconnect asked
// for the connection is no longer wanted, so the tab offers Connect again
// instead of a Disconnect that closes nothing.
func (conn *Connection) Failed() {
	conn.Connecting = false
	if !conn.Reconnect {
		conn.WantOpen = false
	}
}

// Closed records that the session ended: nothing is open and what the
// server announced no longer holds.
func (conn *Connection) Closed() {
	conn.Session, conn.Connecting = nil, false
	conn.Tools, conn.Resources, conn.Prompts, conn.Server = nil, nil, nil, nil
}

// Disconnect records that the user no longer wants the connection. It
// returns the open session, which the caller closes, or reports that a
// connect in progress was abandoned, whose answer is dropped from now on.
func (conn *Connection) Disconnect() (session port.Session, abandoned bool) {
	conn.WantOpen = false
	conn.LastEventID = "" // a deliberate disconnect starts the stream over next time
	if conn.Session != nil {
		return conn.Session, false
	}
	if conn.Connecting {
		conn.Gen++
		conn.Connecting = false
		return nil, true
	}
	return nil, false
}

// Received takes what a received event announces: the server's own
// description, its tools, resources and prompts, and the id to resume an event stream from.
func (conn *Connection) Received(event port.Event) {
	if event.Dir == port.Info && event.Kind == "initialize" {
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

// CanRetry reports whether a retry should open the connection now: the user
// still wants it and nothing is open or being made.
func (conn *Connection) CanRetry() bool {
	return conn.WantOpen && conn.Session == nil && !conn.Connecting
}

// RetryWanted reports whether a dropped connection is to be reopened.
func (conn *Connection) RetryWanted() bool { return conn.Reconnect && conn.WantOpen }
