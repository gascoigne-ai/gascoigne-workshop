package theme

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestRequestBadge(t *testing.T) {
	tests := []struct {
		protocol     request.Protocol
		method, want string
	}{
		{"rest", "GET", "GET"}, {"", "POST", "POST"}, {"rest", "DELETE", "DEL"},
		{"websocket", "", "WS"}, {"mcp", "GET", "MCP"},
	}
	for _, tt := range tests {
		if got := RequestBadge(tt.protocol, tt.method); got != tt.want {
			t.Errorf("RequestBadge(%q,%q) = %q, want %q", tt.protocol, tt.method, got, tt.want)
		}
	}
}
