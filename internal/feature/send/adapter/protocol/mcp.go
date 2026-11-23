package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// MCP speaks the Model Context Protocol, JSON-RPC 2.0, to a server over
// streamable HTTP (the URL is http or https) or stdio (the URL is
// "stdio:" followed by the command line). It is a small hand-written
// client, as the WebSocket one, so the project keeps no SDK dependency.
type MCP struct {
	// Sampler, when set, answers the sampling requests a server sends
	// (it asks the client to produce a model reply); without one the
	// client does not offer sampling and refuses such requests.
	Sampler request.Sampler

	// Network is the cookie jar, proxy and TLS setting of an HTTP server.
	Network Network
}

const (
	mcpVersion        = "2025-06-18"
	mcpInitTimeout    = 30 * time.Second
	mcpGoodbyeTimeout = 2 * time.Second // the DELETE that ends an HTTP session
	mcpMaxMessage     = 16 << 20
)

// mcpTransport moves JSON-RPC messages to and from a server. Messages
// from the server go to the deliver function the transport was built with.
type mcpTransport interface {
	send(ctx context.Context, msg []byte) error
	setVersion(value string)
	close(ctx context.Context) error
}

// Connect implements Streamer: it opens the transport, runs the
// initialize handshake and asks for the server's tools, resources and
// prompts. Everything is logged as events.
func (m MCP) Connect(ctx context.Context, req request.Request) (request.Session, error) {
	s, err := m.connect(ctx, req)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// connect is Connect with the concrete session, which Send needs.
func (m MCP) connect(ctx context.Context, req request.Request) (*mcpSession, error) {
	s := &mcpSession{eventLog: newEventLog(), sampler: m.Sampler, waiters: map[int64]chan rpcAnswer{}}
	tr, err := openTransport(req, m.Network, s)
	if err != nil {
		return nil, err
	}
	s.tr = tr
	s.emit(request.Event{Dir: request.Info, Kind: request.KindOpen, Text: "Connecting to " + req.URL})
	init, err := m.initialize(ctx, s)
	if err != nil {
		s.Close(context.WithoutCancel(ctx)) // the goodbye must not die with a timed-out connect
		return nil, err
	}
	tr.setVersion(init.ProtocolVersion)
	if listener, ok := tr.(interface{ listen() }); ok {
		listener.listen() // servers may push notifications on a stream of their own
	}
	if err := s.notify("notifications/initialized"); err != nil {
		s.Close(context.WithoutCancel(ctx))
		return nil, err
	}
	name := init.ServerInfo.Name
	if name == "" {
		name = req.URL
	}
	s.emit(request.Event{Dir: request.Info, Kind: request.KindOpen, Text: "Connected to " + name})
	s.requestLists(init.Capabilities)
	return s, nil
}

// openTransport builds the stdio transport for a "stdio:" URL and the
// streamable HTTP one for any other.
func openTransport(req request.Request, network Network, s *mcpSession) (mcpTransport, error) {
	if strings.HasPrefix(req.URL, request.StdioPrefix) {
		return newMCPStdio(strings.TrimSpace(strings.TrimPrefix(req.URL, request.StdioPrefix)), s.deliver, s.finish)
	}
	return newMCPHTTP(req, network, s.deliver, s.finish, s.note)
}

// mcpInitResult is what the server answers to initialize.
type mcpInitResult struct {
	ProtocolVersion string                     `json:"protocolVersion"`
	Capabilities    map[string]json.RawMessage `json:"capabilities"`
	ServerInfo      struct{ Name, Version string }
}

// initialize sends the initialize request, offering sampling when the
// client has a Sampler, and reads the server's answer.
func (m MCP) initialize(ctx context.Context, s *mcpSession) (mcpInitResult, error) {
	ictx, cancel := context.WithTimeout(ctx, mcpInitTimeout)
	defer cancel()
	params := initializeParams{ProtocolVersion: mcpVersion, ClientInfo: clientInfo{Name: "gascoigne-workshop", Version: "0"}}
	if m.Sampler != nil {
		params.Capabilities.Sampling = &struct{}{}
	}
	result, err := s.call(ictx, "initialize", params)
	if err != nil {
		return mcpInitResult{}, err
	}
	var init mcpInitResult
	if err := json.Unmarshal(result, &init); err != nil { // the session works without the server's description
		s.note(fmt.Errorf("read the initialize result: %w", err))
	}
	s.emit(request.Event{Dir: request.Info, Kind: request.KindInitialize, Text: string(result)}) // the UI shows what the server said about itself
	return init, nil
}

// requestLists asks for the lists the server's capabilities announce.
func (s *mcpSession) requestLists(capabilities map[string]json.RawMessage) {
	for _, list := range []struct{ capability, method string }{
		{"tools", "tools/list"}, {"resources", "resources/list"}, {"prompts", "prompts/list"},
	} {
		if _, ok := capabilities[list.capability]; ok {
			if _, err := s.request(context.Background(), list.method, nil); err != nil {
				s.emit(request.Event{Dir: request.Info, Kind: request.KindError, Text: err.Error()})
			}
		}
	}
}

// Send implements Protocol as the one-shot form: connect, list the
// tools and return that answer as the body.
func (m MCP) Send(ctx context.Context, req request.Request) (request.Response, error) {
	start := time.Now()
	s, err := m.connect(ctx, req)
	if err != nil {
		return request.Response{Err: stopReason(ctx, err), Duration: time.Since(start)}, nil
	}
	defer s.Close(context.WithoutCancel(ctx))
	result, err := s.call(ctx, "tools/list", nil)
	if err != nil {
		return request.Response{Err: stopReason(ctx, err), Duration: time.Since(start)}, nil
	}
	return request.Response{Status: http.StatusOK, StatusText: "OK", Duration: time.Since(start), Size: len(result), Body: result}, nil
}
