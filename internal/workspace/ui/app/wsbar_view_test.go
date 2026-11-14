package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The toolbar while a connection opens is the one look the Messages pane
// pins do not show; it is pinned at both widths.
func TestTheMessagesToolbarWhileConnectingLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", messagesWideW},
		{"narrow", messagesNarrowW},
	}
	for _, size := range sizes {
		name := "messagestoolbar-connecting-" + size.name
		t.Run(name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			tab := a.cur()
			tab.Req.Protocol = "websocket"
			tab.Connecting = true
			root := Col(a.messagesToolbar(tab))
			Layout(a.ctx, root, size.width, messagesPaneH)
			want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got := Dump(root); got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

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
	a.logEvent(tab, reqport.Event{Time: time.Date(2026, 11, 14, 10, 30, 0, 0, time.UTC), Dir: reqport.Received, Kind: "text", Text: "hi"})
	if len(tab.wsView.Lines) == 0 {
		t.Fatal("the log is empty before Clear")
	}
	clickText(t, a, "Clear", 0)
	if n := len(tab.wsView.Lines); n != 0 {
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
			tab.Tools = []model.MCPTool{{Name: "search"}, {Name: "fetch"}}
			tab.Resources = []model.MCPResource{{URI: "file:///a.txt"}}
			tab.Prompts = []model.MCPPrompt{{Name: "review"}}
			a.step(core.Input{})
			clickText(t, a, "JSON-RPC", 0)
			clickText(t, a, tc.label, 0)
			if got := menuLabels(a); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("menu = %v, want %v", got, tc.want)
			}
		})
	}
}
