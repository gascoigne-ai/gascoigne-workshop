package tabs

// ConnState is where a tab's connection is in its life. The zero value is
// ConnIdle, so a new tab is not connected.
type ConnState uint8

const (
	ConnIdle       ConnState = iota // nothing open, nothing wanted
	ConnConnecting                  // a connect is in progress
	ConnOpen                        // a session is up
	ConnClosing                     // the user asked to close; the session has not ended yet
	ConnRetrying                    // the connection dropped and a reconnect is due
	connStateCount
)

// connEvent is something that moves a connection from one state to another.
type connEvent uint8

const (
	connBegin connEvent = iota
	connOpened
	connFailed
	connClosed
	connDisconnect
	connEventCount
)

// connCell is one cell of the transition table; the zero cell refuses the
// event.
type connCell struct {
	to ConnState
	ok bool
}

// connTransitions is the whole machine, a row per state and a column per
// event; a cell left out is refused. A move into ConnRetrying becomes a move
// into ConnIdle when the tab does not ask for a reconnect (see Connection.step).
var connTransitions = [connStateCount][connEventCount]connCell{
	ConnIdle: {
		connBegin:      {ConnConnecting, true},
		connDisconnect: {ConnIdle, true}, // nothing to close; still forgets the stream position
	},
	ConnConnecting: {
		connBegin:      {ConnConnecting, true}, // a newer connect supersedes this one
		connOpened:     {ConnOpen, true},
		connFailed:     {ConnRetrying, true},
		connDisconnect: {ConnIdle, true},
	},
	ConnOpen: {
		connClosed:     {ConnRetrying, true},
		connDisconnect: {ConnClosing, true},
	},
	ConnClosing: {
		connClosed:     {ConnIdle, true}, // the user wanted it closed: never a retry
		connDisconnect: {ConnClosing, true},
	},
	ConnRetrying: {
		connBegin:      {ConnConnecting, true},
		connDisconnect: {ConnIdle, true},
	},
}

// transition is the state event leads to from state, and false when the
// event is refused there.
func transition(state ConnState, event connEvent) (ConnState, bool) {
	cell := connTransitions[state][event]
	return cell.to, cell.ok
}
