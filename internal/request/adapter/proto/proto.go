// Package proto sends a model.Request over one protocol and reports the
// outcome as a model.Response. Protocols are looked up by name in a
// Registry; nothing here knows about the UI.
package proto

import (
	"sort"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

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

// Configured is NewRegistry with REST and MCP made from config.
func Configured(config port.ProtocolConfig) Registry {
	registry := NewRegistry()
	registry["rest"] = REST{Jar: config.Jar, Proxy: config.Proxy, CAFile: config.CACert, CertFile: config.ClientCert, KeyFile: config.ClientKey}
	registry["mcp"] = MCP{Sampler: config.Sampler}
	return registry
}

func sortKV(rows []model.KV) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Name != rows[j].Name {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Value < rows[j].Value
	})
}
