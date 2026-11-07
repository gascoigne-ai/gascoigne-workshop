// Package proto sends a model.Request over one protocol and reports the
// outcome as a model.Response. Protocols are looked up by name in a
// Registry; nothing here knows about the UI.
package proto

import (
	"context"
	"fmt"
	"sort"

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

// NewRegistry returns the built-in protocols.
func NewRegistry() Registry {
	return Registry{
		"rest":      REST{},
		"websocket": WebSocket{},
		"mcp":       MCP{},
		"sse":       SSE{},
		"grpc":      GRPC{},
	}
}

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

func sortKV(rows []model.KV) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Name != rows[j].Name {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Value < rows[j].Value
	})
}

// Messages in Response.Err for the two ways a request can be stopped.
const (
	ErrCancelled = "Cancelled"
	ErrTimedOut  = "Timed out"
)
