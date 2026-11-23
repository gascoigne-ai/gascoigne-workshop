package ws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// paneStates are the looks of the Messages pane: a WebSocket before
// connecting and connected with traffic in the log, an MCP server with its
// pickers and an event stream, which has no composer.
var paneStates = []struct {
	name  string
	setUp func(world *world)
}{
	{"websocket-idle", func(world *world) {}},
	{"websocket-connected", func(world *world) {
		world.open(&recorder{})
		world.tab.Reconnect = true
		world.pane.Log(world.tab, request.Event{Time: sentAt, Dir: request.Sent, Kind: request.KindText, Text: `{"op":"ping"}`})
		world.pane.Log(world.tab, request.Event{Time: sentAt, Dir: request.Received, Kind: request.KindText, Text: `{"op":"ping"}`})
	}},
	{"sse", func(world *world) {}},
	{"mcp-connected", func(world *world) {
		world.open(&recorder{})
		world.tab.Tools = []request.MCPTool{{Name: "search"}}
		world.tab.Resources = []request.MCPResource{{URI: "file:///a.txt"}}
		world.tab.Prompts = []request.MCPPrompt{{Name: "review"}}
	}},
	{"grpc-stream", func(world *world) { world.open(&recorder{}) }},
}

// protocolOf is the protocol each pinned state shows.
var protocolOf = map[string]request.Protocol{
	"websocket-idle":      request.ProtocolWebSocket,
	"websocket-connected": request.ProtocolWebSocket,
	"sse":                 request.ProtocolSSE,
	"mcp-connected":       request.ProtocolMCP,
	"grpc-stream":         request.ProtocolGRPC,
}

// The Messages pane is pinned as text (element kinds, options, rects) at
// two widths; the goldens are the ones the pane had in the app before it
// moved here.
func TestThePaneLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", wideW},
		{"narrow", narrowW},
	}
	for _, state := range paneStates {
		for _, size := range sizes {
			name := "messagespane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t, protocolOf[state.name])
				state.setUp(world)
				assertDumpPinned(t, world.ctx, world.pane.View(world.ctx, world.tab), name, size.width, paneH)
			})
		}
	}
}

// The toolbar while a connection opens is the one look the pane pins do
// not show.
func TestTheToolbarWhileConnectingLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", wideW},
		{"narrow", narrowW},
	}
	for _, size := range sizes {
		name := "messagestoolbar-connecting-" + size.name
		t.Run(name, func(t *testing.T) {
			world := newWorld(t, request.ProtocolWebSocket)
			world.tab.Begin()
			assertDumpPinned(t, world.ctx, Col(world.pane.toolbar(world.ctx, world.tab)), name, size.width, paneH)
		})
	}
}

// What Dump does not show of the composer: it edits the tab's own message,
// sends on Enter and hints at what can be sent in each state; an event
// stream has no composer.
func TestTheComposerEditsTheTabsMessage(t *testing.T) {
	hints := map[string]string{
		"websocket-idle":      "Connect to send messages",
		"websocket-connected": "Message",
		"sse":                 "",
		"mcp-connected":       composersOfTheApp[request.ProtocolMCP].Hint,
		"grpc-stream":         "Message",
	}
	for _, state := range paneStates {
		t.Run(state.name, func(t *testing.T) {
			world := newWorld(t, protocolOf[state.name])
			state.setUp(world)
			fields := findWidgets(world.pane.View(world.ctx, world.tab), "*widget.fieldB")
			if hints[state.name] == "" {
				if len(fields) != 0 {
					t.Fatalf("a composer where none belongs: %+v", fields[0].Behavior())
				}
				return
			}
			if len(fields) != 1 {
				t.Fatalf("%d fields, want the composer", len(fields))
			}
			settings := fmt.Sprintf("%+v", fields[0].Behavior())
			for _, want := range []string{"field:" + fmt.Sprintf("%p", world.pane.Composer(world.tab)), "placeholder:" + hints[state.name] + " segments:"} {
				if !strings.Contains(settings, want) {
					t.Errorf("the composer lacks %q: %s", want, settings)
				}
			}
			if strings.Contains(settings, "onEnter:<nil>") {
				t.Errorf("the composer does not send on Enter: %s", settings)
			}
		})
	}
}

// What Dump does not show of the log: it draws the tab's own message view.
func TestTheLogShowsTheTabsView(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	codes := findWidgets(world.pane.View(world.ctx, world.tab), "*widget.codeB")
	if len(codes) != 1 {
		t.Fatalf("%d logs, want 1", len(codes))
	}
	if got, want := fmt.Sprintf("%+v", codes[0].Behavior()), fmt.Sprintf("&{Base:{} view:%p wrap:false}", &world.pane.state(world.tab).log); got != want {
		t.Fatalf("log %s, want %s", got, want)
	}
}

// Pins what differs between the composers of the protocols that connect:
// only WebSocket offers the { } button that makes the message one line of
// JSON.
func TestOnlyTheWebSocketComposerOffersFormat(t *testing.T) {
	for _, protocol := range []request.Protocol{request.ProtocolWebSocket, request.ProtocolMCP, request.ProtocolGRPC} {
		world := newWorld(t, protocol)
		dump := Dump(world.pane.View(world.ctx, world.tab))
		if formats := strings.Contains(dump, "{ }"); formats != (protocol == request.ProtocolWebSocket) {
			t.Errorf("%s offers { }: %v", protocol, formats)
		}
	}
}

func TestTheAutoReconnectChipTogglesTheTabsPolicy(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	start := world.tab.Reconnect
	world.click("Auto-reconnect")
	if world.tab.Reconnect == start {
		t.Fatalf("Reconnect stays %v after one click", start)
	}
	world.click("Auto-reconnect")
	if world.tab.Reconnect != start {
		t.Fatalf("Reconnect = %v after two clicks, want %v", world.tab.Reconnect, start)
	}
}

// On an MCP server each picker opens a menu of what the server offers.
func TestEachPickerOpensAMenuOfWhatTheServerOffers(t *testing.T) {
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
			world := newWorld(t, request.ProtocolMCP)
			world.tab.Tools = []request.MCPTool{{Name: "search"}, {Name: "fetch"}}
			world.tab.Resources = []request.MCPResource{{URI: "file:///a.txt"}}
			world.tab.Prompts = []request.MCPPrompt{{Name: "review"}}
			world.click(tc.label)
			if len(world.fake.menus) != 1 {
				t.Fatalf("%d menus opened, want 1", len(world.fake.menus))
			}
			var got []string
			for _, item := range world.fake.menus[0].items {
				got = append(got, item.Label)
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Fatalf("menu = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestToolPickerAsksTheHostToOpenTheForm(t *testing.T) {
	world := newWorld(t, request.ProtocolMCP)
	world.tab.Tools = []request.MCPTool{{Name: "search"}, {Name: "fetch"}}
	world.click("Tools (2)")
	world.fake.menus[0].items[1].Run()
	if len(world.fake.forms) != 1 || world.fake.forms[0].Name != "fetch" {
		t.Fatalf("forms %+v, want the fetch form", world.fake.forms)
	}
}

func TestTheToolbarPingAndClearActOnTheTab(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	session := &recorder{}
	world.open(session)
	world.pane.Log(world.tab, request.Event{Time: sentAt, Dir: request.Received, Kind: request.KindText, Text: "hi"})
	world.click("Ping")
	world.fake.runAll(t)
	if got := session.log(); len(got) != 1 || got[0] != "ping" {
		t.Fatalf("sent %q, want ping", got)
	}
	world.click("Clear")
	if got := len(world.pane.Lines(world.tab)); got != 0 {
		t.Fatalf("the log keeps %d lines", got)
	}
}
