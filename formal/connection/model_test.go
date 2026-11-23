package connection_test

import (
	"context"
	"os"
	"testing"

	"github.com/gascoigne-ai/gascoigne-formal/formal"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// TestMain draws nothing, so a failing run prints plain text on every terminal.
func TestMain(m *testing.M) {
	os.Setenv(formal.VisualEnv, "off")
	os.Exit(m.Run())
}

// connects is the most connects a run starts: the space of the search. A run that would start one
// more is cut, which is what the printed "within: connects=3" says.
const connects = 3

// stubSession is a session that is a plain value, so the checker can encode and compare a state
// that holds one. Gen is the connect that produced it.
type stubSession struct{ Gen int }

func (stubSession) Events() <-chan request.Event       { return nil }
func (stubSession) Send(context.Context, string) error { return nil }
func (stubSession) Ping(context.Context) error         { return nil }
func (stubSession) Close(context.Context) error        { return nil }

// ops is the part of tabs.Connection the model calls; real is the methods themselves, and a
// broken variant replaces one with a bug the real code could grow.
type ops struct {
	begin      func(*tabs.Connection) (int, bool)
	opened     func(*tabs.Connection, request.Session) bool
	failed     func(*tabs.Connection) bool
	closed     func(*tabs.Connection) bool
	disconnect func(*tabs.Connection) (request.Session, bool)
}

func realOps() ops {
	return ops{
		begin:      (*tabs.Connection).Begin,
		opened:     (*tabs.Connection).Opened,
		failed:     (*tabs.Connection).Failed,
		closed:     (*tabs.Connection).Closed,
		disconnect: (*tabs.Connection).Disconnect,
	}
}

// both is the start states: a connection nothing was asked of, with and without a reconnect.
func both() []tabs.Connection {
	return []tabs.Connection{{}, {Reconnect: true}}
}

// overGen drops a successor the space does not contain: one that started more connects than
// connects.
func overGen(conn tabs.Connection) bool { return conn.Gen > connects }
