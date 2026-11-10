package model

import (
	"encoding/json"
	"sort"
)

// MCPServer is what an MCP server said about itself when the connection
// was initialized.
type MCPServer struct {
	Name         string
	Version      string
	Protocol     string   // the protocol version the server chose
	Capabilities []string // the names of the features it offers, sorted
	Instructions string
}

// ParseServerInfo reads the result of an initialize request. It returns
// false for anything else.
func ParseServerInfo(result string) (MCPServer, bool) {
	var init struct {
		ProtocolVersion string                     `json:"protocolVersion"`
		Capabilities    map[string]json.RawMessage `json:"capabilities"`
		ServerInfo      struct{ Name, Version string }
		Instructions    string
	}
	if json.Unmarshal([]byte(result), &init) != nil || init.ProtocolVersion == "" {
		return MCPServer{}, false
	}
	server := MCPServer{Name: init.ServerInfo.Name, Version: init.ServerInfo.Version, Protocol: init.ProtocolVersion, Instructions: init.Instructions}
	for name := range init.Capabilities {
		server.Capabilities = append(server.Capabilities, name)
	}
	sort.Strings(server.Capabilities)
	return server, true
}
