package proto

import (
	"context"
	"fmt"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// Direction says where an Event came from.
type Direction int

// Event directions.
const (
	Info     Direction = iota // the connection itself: opened, closed
	Sent                      // a message this side sent
	Received                  // a message from the peer
)

// Event is one line of a streaming protocol's log.
type Event struct {
	Time time.Time
	Dir  Direction
	Kind string // "text", "binary", "ping", "pong", "close", "open"
	Text string // the message, or the description for Info events
	Size int    // bytes of the message
	Code int    // close status code, for "close"
}

// Session is a live, two-way connection. Events carries everything that
// happens, including this side's own sends, in order, and is closed when
// the session ends. A consumer that stops reading slows the connection
// down; no event is dropped.
type Session interface {
	Events() <-chan Event
	Send(text string) error
	Ping() error
	Close() error
}

// Streamer is implemented by protocols that keep a connection open.
// Protocol.Send of such a protocol is the one-shot form: connect, send
// the body, wait for the first reply, disconnect.
type Streamer interface {
	Connect(ctx context.Context, req model.Request) (Session, error)
}

// Connect opens a session with the streaming protocol req names.
func (r Registry) Connect(ctx context.Context, req model.Request) (Session, error) {
	p, ok := r[req.Protocol]
	if !ok {
		return nil, fmt.Errorf("unknown protocol %q", req.Protocol)
	}
	s, ok := p.(Streamer)
	if !ok {
		return nil, fmt.Errorf("protocol %q does not stream", req.Protocol)
	}
	return s.Connect(ctx, req)
}
