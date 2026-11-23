package connection_test

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-formal/formal"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// lastEvent is the latest event tried and whether the connection took it. Ghost: properties read
// it, the code does not hold it.
type lastEvent struct {
	From  tabs.ConnState
	Event string
	Taken bool
}

// eventsState is a real Connection, the one before the last event, what the user wants (a ghost)
// and the last event. The checker copies the Connection by value, so every action calls the real
// method on a copy.
type eventsState struct {
	Conn   tabs.Connection
	Before tabs.Connection
	Last   lastEvent
	Wanted bool // a connect was asked for and the user has not disconnected since
}

// stateNames names every connection state for a trace line; the order is the table's.
var stateNames = map[tabs.ConnState]string{
	tabs.ConnIdle: "Idle", tabs.ConnConnecting: "Connecting", tabs.ConnOpen: "Open",
	tabs.ConnClosing: "Closing", tabs.ConnRetrying: "Retrying",
}

var allStates = []tabs.ConnState{tabs.ConnIdle, tabs.ConnConnecting, tabs.ConnOpen, tabs.ConnClosing, tabs.ConnRetrying}

var allEvents = []string{"Begin", "Opened", "Failed", "Closed", "Disconnect"}

// eventAction applies one real call. The call reports whether the event was taken; Disconnect is
// never refused by the table, so it reports true.
func eventAction(name string, call func(*tabs.Connection) bool, wanted func(state eventsState, taken bool) bool) formal.Action[eventsState] {
	return formal.Action[eventsState]{Name: name, Next: func(state eventsState) []eventsState {
		state.Before = state.Conn
		state.Last = lastEvent{From: state.Conn.State(), Event: name}
		state.Last.Taken = call(&state.Conn)
		state.Wanted = wanted(state, state.Last.Taken)
		if overGen(state.Conn) {
			return nil
		}
		return []eventsState{state}
	}}
}

func eventsSpec(name string, impl ops) formal.Spec[eventsState] {
	keep := func(state eventsState, _ bool) bool { return state.Wanted }
	actions := []formal.Action[eventsState]{
		eventAction("Begin", func(conn *tabs.Connection) bool { _, ok := impl.begin(conn); return ok },
			func(state eventsState, taken bool) bool { return state.Wanted || taken }),
		eventAction("Opened", func(conn *tabs.Connection) bool { return impl.opened(conn, stubSession{Gen: conn.Gen}) }, keep),
		eventAction("Failed", impl.failed, keep),
		eventAction("Closed", impl.closed, keep),
		eventAction("Disconnect", func(conn *tabs.Connection) bool { impl.disconnect(conn); return true },
			func(eventsState, bool) bool { return false }),
		{Name: "ToggleReconnect", Next: func(state eventsState) []eventsState {
			state.Conn.Reconnect = !state.Conn.Reconnect
			state.Before, state.Last = state.Conn, lastEvent{From: state.Conn.State(), Event: "ToggleReconnect", Taken: true}
			return []eventsState{state}
		}},
	}
	var sometimes []formal.Property[eventsState]
	for _, from := range allStates {
		for _, event := range allEvents {
			sometimes = append(sometimes, formal.Property[eventsState]{
				Name:  "tried " + stateNames[from] + "/" + event,
				Holds: func(state eventsState) bool { return state.Last.From == from && state.Last.Event == event },
			})
		}
	}
	var init []eventsState
	for _, conn := range both() {
		init = append(init, eventsState{Conn: conn, Before: conn})
	}
	return formal.Spec[eventsState]{
		Name:    name,
		Init:    init,
		Actions: actions,
		Always: []formal.Property[eventsState]{
			{Name: "RefusedEventChangesNothing", Claim: "refused-event-changes-nothing", Holds: func(state eventsState) bool {
				return state.Last.Taken || reflect.DeepEqual(state.Conn, state.Before)
			}},
			{Name: "SessionOnlyWhileUp", Claim: "session-only-while-up", Holds: func(state eventsState) bool {
				up := state.Conn.State() == tabs.ConnOpen || state.Conn.State() == tabs.ConnClosing
				return (state.Conn.Session() != nil) == up
			}},
			{Name: "UserCloseNeverRetries", Claim: "user-close-never-retries", Holds: func(state eventsState) bool {
				now := state.Conn.State()
				return state.Wanted || (now != tabs.ConnConnecting && now != tabs.ConnRetrying)
			}},
			{Name: "RetryOnlyWhenAsked", Claim: "retry-only-when-asked", Holds: func(state eventsState) bool {
				dropped := state.Last.Event == "Failed" || state.Last.Event == "Closed"
				fromLive := state.Last.From == tabs.ConnConnecting || state.Last.From == tabs.ConnOpen
				if !state.Last.Taken || !dropped || !fromLive || state.Last.From == tabs.ConnClosing {
					return true
				}
				if state.Before.Reconnect {
					return state.Conn.State() == tabs.ConnRetrying
				}
				return state.Conn.State() == tabs.ConnIdle
			}},
		},
		Sometimes: sometimes,
		Done:      func(eventsState) bool { return true },
		Space:     formal.Space{Params: []formal.Param{{Name: "connects", Value: connects}}},
	}
}

// Every order of the five events, from both start states, with both reconnect settings flipped at
// any time, is explored on the real Connection: the state is the real value and each action calls
// the real method, so a pass is a proof about those methods within the printed space. The 25
// Sometimes properties prove each state-and-event cell was tried.
func TestConnectionEventsAreVerified(t *testing.T) {
	formal.Check(t, eventsSpec("connection-events", realOps()))
}

// Each broken variant is a bug the real methods could grow; the check must name the property.
func TestBrokenConnectionsAreFound(t *testing.T) {
	refusedBumps := realOps()
	refusedBumps.begin = func(conn *tabs.Connection) (int, bool) {
		gen, ok := conn.Begin()
		if !ok {
			conn.Gen++
		}
		return gen, ok
	}
	forgetsSession := realOps()
	forgetsSession.opened = func(conn *tabs.Connection, _ request.Session) bool { return conn.Opened(nil) }
	closeIgnored := realOps()
	closeIgnored.disconnect = func(conn *tabs.Connection) (request.Session, bool) {
		if conn.State() == tabs.ConnOpen {
			return nil, false
		}
		return conn.Disconnect()
	}
	alwaysRetries := realOps()
	alwaysRetries.failed = func(conn *tabs.Connection) bool {
		if conn.State() == tabs.ConnConnecting {
			conn.Reconnect = true
		}
		return conn.Failed()
	}
	cases := []struct {
		name         string
		impl         ops
		wantProperty string
	}{
		{"a refused Begin counts a connect", refusedBumps, "RefusedEventChangesNothing"},
		{"Opened keeps no session", forgetsSession, "SessionOnlyWhileUp"},
		{"Disconnect of an open connection is ignored", closeIgnored, "UserCloseNeverRetries"},
		{"Failed retries without being asked", alwaysRetries, "RetryOnlyWhenAsked"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome, err := formal.Run(eventsSpec("connection-broken", tc.impl))
			if err != nil {
				t.Fatal(err)
			}
			violated, ok := outcome.(formal.Violated)
			if !ok {
				t.Fatalf("outcome = %T, want a violation", outcome)
			}
			broken, ok := violated.Violation.(formal.InvariantBroken)
			if !ok || broken.Property != tc.wantProperty {
				t.Fatalf("violation = %#v, want property %q broken", violated.Violation, tc.wantProperty)
			}
		})
	}
}
