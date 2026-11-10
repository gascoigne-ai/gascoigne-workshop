// Package port is what the request domain needs from the outside: the
// protocols that send a request, and the types that cross that border.
package port

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Protocol performs a request. Transport failures are reported in
// Response.Err rather than as an error, so the UI always has something
// to show; the error return is for misuse (bad method, bad URL).
type Protocol interface {
	Send(ctx context.Context, req model.Request) (model.Response, error)
}

// Registry maps protocol names ("rest", "websocket", "mcp") to
// implementations.
type Registry map[string]Protocol

// Names lists the registered protocol names in sorted order.
func (r Registry) Names() []string {
	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Send runs req with the protocol it names.
func (r Registry) Send(ctx context.Context, req model.Request) (model.Response, error) {
	p, ok := r[req.Protocol]
	if !ok {
		return model.Response{}, fmt.Errorf("unknown protocol %q", req.Protocol)
	}
	return p.Send(ctx, req)
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
	ID   string // the id of a server-sent event, for resuming after a reconnect
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

// BinarySender is implemented by sessions that can send binary messages
// (WebSocket). Others take text only.
type BinarySender interface {
	SendBinary(data []byte) error
}

// Requester is implemented by sessions that can send one message and hand
// back its answer (MCP). The text is what Send takes, except a raw
// message; the result is the JSON of the answer, and a server's error
// comes back as the error.
type Requester interface {
	Request(ctx context.Context, text string) (result string, err error)
}

// Streamer is implemented by protocols that keep a connection open.
// Protocol.Send of such a protocol is the one-shot form: connect, send
// the body, wait for the first reply, disconnect.
type Streamer interface {
	Connect(ctx context.Context, req model.Request) (Session, error)
}

// Sampler produces the reply to a sampling request, or reports false when
// the request is declined. It may block, for example on a person.
type Sampler func(ctx context.Context, request SamplingRequest) (SamplingResult, bool)

// SamplingRequest is what a server asked a model to continue.
type SamplingRequest struct {
	Messages     []SamplingMessage
	SystemPrompt string
	MaxTokens    int
}

// SamplingMessage is one turn of a sampling request, as text.
type SamplingMessage struct{ Role, Text string }

// SamplingResult is the reply given to a sampling request.
type SamplingResult struct{ Text, Model string }

// ProtocolConfig is what the built-in protocols are made from: the cookie
// jar and network settings of REST, and the sampler of MCP.
type ProtocolConfig struct {
	Jar                                  http.CookieJar
	Proxy, CACert, ClientCert, ClientKey string
	Sampler                              Sampler
}

// GRPCMethods lists the methods of the gRPC server a request points at as
// "package.Service/Method".
type GRPCMethods func(ctx context.Context, req model.Request) ([]string, error)
