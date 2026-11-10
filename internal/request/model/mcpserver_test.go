package model

import (
	"reflect"
	"testing"
)

func TestParseServerInfoReadsAnInitializeResult(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   MCPServer
		ok     bool
	}{
		{"full", `{"protocolVersion":"2025-06-18","capabilities":{"tools":{"listChanged":true},"resources":{}},"serverInfo":{"name":"example","version":"1.2.0"},"instructions":"Use echo."}`,
			MCPServer{Name: "example", Version: "1.2.0", Protocol: "2025-06-18", Capabilities: []string{"resources", "tools"}, Instructions: "Use echo."}, true},
		{"no capabilities", `{"protocolVersion":"2025-06-18","serverInfo":{"name":"bare"}}`,
			MCPServer{Name: "bare", Protocol: "2025-06-18"}, true},
		{"not json", `nope`, MCPServer{}, false},
		{"not an initialize result", `{"tools":[]}`, MCPServer{}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseServerInfo(test.result)
			if ok != test.ok || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %+v %v, want %+v %v", got, ok, test.want, test.ok)
			}
		})
	}
}
