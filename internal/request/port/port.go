// Package port is what the request domain needs from the outside: the
// protocols that send a request, and the types that cross that border.
package port

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Protocol performs a request. Transport failures are reported in
// Response.Err rather than as an error, so the UI always has something
// to show; the error return is for misuse (bad method, bad URL).
type Protocol interface {
	Send(ctx context.Context, req model.Request) (model.Response, error)
}

// Registry maps each protocol of model.Protocols to its implementation.
type Registry map[model.Protocol]Protocol

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

// EventKind says what a session reports an Event to be. It is what this
// side decided, never text a peer chose: the name a server gives an event
// is Event.Name.
type EventKind string

// Event kinds.
const (
	KindOpen         EventKind = "open"         // the connection is up
	KindClose        EventKind = "close"        // the connection ended
	KindEnd          EventKind = "end"          // this side of a stream is closed
	KindInitialize   EventKind = "initialize"   // an MCP server's description of itself (Info)
	KindInfo         EventKind = "info"         // a note of the UI itself
	KindError        EventKind = "error"        // a problem (Info) or a JSON-RPC error response (Received)
	KindText         EventKind = "text"         // a WebSocket text frame
	KindBinary       EventKind = "binary"       // a WebSocket binary frame
	KindPing         EventKind = "ping"         // a WebSocket ping
	KindPong         EventKind = "pong"         // a WebSocket pong
	KindMessage      EventKind = "message"      // a gRPC message or a server-sent event
	KindRequest      EventKind = "request"      // a JSON-RPC request
	KindResponse     EventKind = "response"     // a JSON-RPC result
	KindNotification EventKind = "notification" // a JSON-RPC notification
	KindInvalid      EventKind = "invalid"      // a message that is not JSON-RPC
)

// String is the kind as shown.
func (kind EventKind) String() string { return string(kind) }

// CarriesPayload reports whether an empty Text of this kind is an empty
// message and not a control event that has nothing to say.
func (kind EventKind) CarriesPayload() bool { return kind == KindText || kind == KindBinary }

// Event is one line of a streaming protocol's log. Kind is set by the
// session and Name is the peer's untrusted text, so a server cannot pose as
// an error or as the initialize note by naming its event. Events are not
// persisted.
type Event struct {
	Time time.Time
	Dir  Direction
	Kind EventKind // what the session says this is; code compares it
	Name string    // what the peer called it (a server-sent event's name); empty otherwise, never compared
	Text string    // the message, or the description for Info events
	Size int       // bytes of the message
	Code int       // close status code, for KindClose
	ID   string    // the id of a server-sent event, for resuming after a reconnect

	// Payload is what the session parsed out of Text, nil when it parsed
	// nothing. Only the MCP session sets it, and only on the answer to a
	// list request.
	Payload Payload
}

// Payload is a typed message a session announces next to the text of its
// event: ToolsList, ResourcesList or PromptsList. The set is closed.
type Payload interface{ payload() }

// ToolsList is the answer to tools/list. An empty list is an answer: the
// server has no tools now.
type ToolsList struct{ Tools []model.MCPTool }

// ResourcesList is the answer to resources/list.
type ResourcesList struct{ Resources []model.MCPResource }

// PromptsList is the answer to prompts/list.
type PromptsList struct{ Prompts []model.MCPPrompt }

func (ToolsList) payload()     {}
func (ResourcesList) payload() {}
func (PromptsList) payload()   {}

// Label names the event for a line that has no text: the peer's name for
// a message that has one, the kind otherwise.
func (event Event) Label() string {
	if event.Kind == KindMessage && event.Name != "" {
		return event.Name
	}
	return event.Kind.String()
}

// Session is a live, two-way connection. Events carries everything that
// happens, including this side's own sends, in order, and is closed when
// the session ends. A consumer that stops reading slows the connection
// down; no event is dropped. Each method that writes takes the context
// of that write; ending it stops the write, while the session lives on
// until Close. A context that has already ended sends and logs nothing, and
// Close with one skips the goodbye (the MCP DELETE).
type Session interface {
	Events() <-chan Event
	Send(ctx context.Context, text string) error
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}

// BinarySender is implemented by sessions that can send binary messages
// (WebSocket). Others take text only.
type BinarySender interface {
	SendBinary(ctx context.Context, data []byte) error
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

// GRPCSchema describes the request message of the method a gRPC request
// names.
type GRPCSchema func(ctx context.Context, req model.Request) (model.MessageSchema, error)
