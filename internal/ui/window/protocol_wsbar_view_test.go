package window

import (
	"reflect"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Ping sends a ping frame on the open session.
func TestTheToolbarPingSendsAPing(t *testing.T) {
	a, session := openConnectedEcho(t)
	clickText(t, a, "Ping", 0)
	until(t, a, "the ping", func() bool { return len(session.log()) == 1 })
	if got := session.log()[0]; got != "ping" {
		t.Fatalf("sent %q, want ping", got)
	}
}

// Clear empties the message log.
func TestTheToolbarClearEmptiesTheLog(t *testing.T) {
	a, _ := openConnectedEcho(t)
	tab := a.cur()
	a.editor.WS.Log(tab.Tab, request.Event{Time: time.Date(2026, 11, 14, 10, 30, 0, 0, time.UTC), Dir: request.Received, Kind: request.KindText, Text: "hi"})
	if len(a.editor.WS.Lines(tab.Tab)) == 0 {
		t.Fatal("the log is empty before Clear")
	}
	clickText(t, a, "Clear", 0)
	if n := len(a.editor.WS.Lines(tab.Tab)); n != 0 {
		t.Fatalf("the log keeps %d lines", n)
	}
}

// The Auto-reconnect chip switches reconnecting on and off.
func TestTheToolbarChipTogglesAutoReconnect(t *testing.T) {
	a, _ := openConnectedEcho(t)
	tab := a.cur()
	start := tab.Reconnect
	clickText(t, a, "Auto-reconnect", 0)
	if tab.Reconnect == start {
		t.Fatalf("Reconnect stays %v after one click", start)
	}
	clickText(t, a, "Auto-reconnect", 0)
	if tab.Reconnect != start {
		t.Fatalf("Reconnect = %v after two clicks, want %v", tab.Reconnect, start)
	}
}

// On an MCP server each picker opens a menu of what the server offers.
func TestTheToolbarPickersOpenTheServersMenus(t *testing.T) {
	cases := []struct {
		label string
		want  []string
	}{
		{"Tools (2)", []string{"search", "fetch"}},
		{"Resources (1)", []string{"file:///a.txt"}},
		{"Prompts (1)", []string{"review"}},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			a, _ := openConnectedEcho(t)
			tab := a.cur()
			tab.Req.Protocol = "mcp"
			tab.Tools = []request.MCPTool{{Name: "search"}, {Name: "fetch"}}
			tab.Resources = []request.MCPResource{{URI: "file:///a.txt"}}
			tab.Prompts = []request.MCPPrompt{{Name: "review"}}
			a.Step(core.Input{})
			clickText(t, a, "JSON-RPC", 0)
			clickText(t, a, tc.label, 0)
			if got := menuLabels(a); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("menu = %v, want %v", got, tc.want)
			}
		})
	}
}
