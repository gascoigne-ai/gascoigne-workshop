// Package proto sends a model.Request over one protocol and reports the
// outcome as a model.Response. Protocols are looked up by name in a
// Registry; nothing here knows about the UI.
//
// WebSocket, SSE and MCP are written by hand on net/http: the part the
// workshop needs is small, the rules allow no new dependency, and the raw
// message log the UI shows is what such a client already handles. gRPC
// uses google.golang.org/grpc and protobuf (the user's choice), since
// HTTP/2 and protobuf are not a small part; there is no code generation,
// messages are built at run time from descriptors. The descriptors come
// from the server's reflection or from .proto files compiled with
// protocompile (the user's choice), which gives the same descriptors as
// reflection, so both sources share one code path. Rejected for .proto
// files: a hand-written parser on protoreflect (a parser and linker to
// keep up with the language) and an external protoc (a program to install,
// output to read back).
package proto

import (
	"sort"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

// NewRegistry returns the built-in protocols, one per row of
// model.Protocols.
func NewRegistry() port.Registry {
	return port.Registry{
		model.ProtocolREST:      REST{},
		model.ProtocolWebSocket: WebSocket{},
		model.ProtocolMCP:       MCP{},
		model.ProtocolSSE:       SSE{},
		model.ProtocolGRPC:      GRPC{},
	}
}

// Configured is NewRegistry with the HTTP-based protocols made from config.
// WebSocket takes the TLS part only: its dialer has no proxy or cookie jar.
func Configured(config port.ProtocolConfig) port.Registry {
	registry := NewRegistry()
	network := Network{Jar: config.Jar, Proxy: config.Proxy, CAFile: config.CACert, CertFile: config.ClientCert, KeyFile: config.ClientKey}
	registry[model.ProtocolREST] = REST{Jar: network.Jar, Proxy: network.Proxy, CAFile: network.CAFile, CertFile: network.CertFile, KeyFile: network.KeyFile}
	registry[model.ProtocolSSE] = SSE{Network: network}
	registry[model.ProtocolMCP] = MCP{Sampler: config.Sampler, Network: network}
	registry[model.ProtocolWebSocket] = WebSocket{Network: network}
	return registry
}

func sortKV(rows []model.KV) {
	sort.Slice(rows, func(index, other int) bool {
		if rows[index].Name != rows[other].Name {
			return rows[index].Name < rows[other].Name
		}
		return rows[index].Value < rows[other].Value
	})
}
