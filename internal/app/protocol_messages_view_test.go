package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Sizes the Messages pane is pinned at: the full main area of the test
// window and the narrowest one.
const (
	messagesWideW   = winW - sidebarW
	messagesNarrowW = minMainW
	messagesPaneH   = 300
)

// messagesPaneStates are the looks of the Messages pane: a WebSocket before
// and after connecting (with a message in the log), an event stream, which
// has no composer, an MCP server with its pickers, and a gRPC stream, whose
// composer has no format button.
var messagesPaneStates = []struct {
	name        string
	placeholder string // the composer's hint; "" when there is no composer
	setUp       func(app *App, tab *reqTab)
}{
	{"websocket-idle", "Connect to send messages", func(app *App, tab *reqTab) {
		tab.Req.Protocol = "websocket"
	}},
	{"websocket-connected", "Message", func(app *App, tab *reqTab) {
		tab.Req.Protocol = "websocket"
		openSession(tab, &fakeSession{})
		tab.Reconnect = true
		sentAt := time.Date(2026, 11, 14, 10, 30, 0, 0, time.UTC)
		app.editor.WS.Log(tab.Tab, request.Event{Time: sentAt, Dir: request.Sent, Kind: request.KindText, Text: `{"op":"ping"}`})
		app.editor.WS.Log(tab.Tab, request.Event{Time: sentAt, Dir: request.Received, Kind: request.KindText, Text: `{"op":"ping"}`})
	}},
	{"sse", "", func(app *App, tab *reqTab) {
		tab.Req.Protocol = "sse"
	}},
	{"mcp-connected", "call TOOL {json}  ·  read URI  ·  prompt NAME {json}  ·  method {json}  ·  or raw JSON-RPC", func(app *App, tab *reqTab) {
		tab.Req.Protocol = "mcp"
		openSession(tab, &fakeSession{})
		tab.Tools = []request.MCPTool{{Name: "search"}}
		tab.Resources = []request.MCPResource{{URI: "file:///a.txt"}}
		tab.Prompts = []request.MCPPrompt{{Name: "review"}}
	}},
	{"grpc-stream", "Message", func(app *App, tab *reqTab) {
		tab.Req.Protocol = "grpc"
		app.editor.GRPC.ToggleStream(tab.Tab)
		openSession(tab, &fakeSession{})
	}},
}

// The Messages pane is pinned as text (element kinds, options, handlers,
// rects) in each of its looks at two widths, so moving it to markup cannot
// change it unseen.
func TestTheMessagesPaneLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", messagesWideW},
		{"narrow", messagesNarrowW},
	}
	for _, state := range messagesPaneStates {
		for _, size := range sizes {
			name := "messagespane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a := testApp(t)
				t.Cleanup(a.WaitWrites)
				tab := a.cur()
				state.setUp(a, tab)
				root := a.messagesPane(wsPanes[0], tab, nil)
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
}

// What Dump does not show of the composer: it edits the tab's message,
// sends on Enter and hints at what can be sent in each state; an event
// stream has no composer.
func TestTheComposerEditsTheTabsMessage(t *testing.T) {
	for _, state := range messagesPaneStates {
		t.Run(state.name, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			tab := a.cur()
			state.setUp(a, tab)
			field := findWidget(a.messagesPane(wsPanes[0], tab, nil), "*widget.fieldB")
			if state.placeholder == "" {
				if field != nil {
					t.Fatalf("a composer where none belongs: %+v", field.Behavior())
				}
				return
			}
			if field == nil {
				t.Fatal("no composer")
			}
			settings := fmt.Sprintf("%+v", field.Behavior())
			for _, want := range []string{"field:" + fmt.Sprintf("%p", a.editor.WS.Composer(tab.Tab)), "placeholder:" + state.placeholder + " segments:"} {
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

// Where the composer and its buttons are in the side-by-side layout of the
// test window, on the WebSocket sample.
const (
	composerY      = 750
	composerFieldX = 964
	formatButtonX  = 1172
	sendButtonX    = 1233
)

// openConnectedEcho shows the WebSocket sample with an open session that
// records what is sent.
func openConnectedEcho(t *testing.T) (*App, *fakeSession) {
	t.Helper()
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	openNamed(t, a, "Echo")
	session := &fakeSession{}
	openSession(a.cur(), session)
	a.Step(core.Input{})
	return a, session
}

// Typed text goes out on Enter and on the Send button, and the composer is
// emptied for the next message.
func TestTheComposerSendsOnEnterAndOnSend(t *testing.T) {
	cases := []struct {
		name string
		send func(app *App)
	}{
		{"enter", func(app *App) { app.Step(core.Input{Enter: true}) }},
		{"send button", func(app *App) { click(app, sendButtonX, composerY) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, session := openConnectedEcho(t)
			click(a, composerFieldX, composerY)
			a.typeText("hello")
			tc.send(a)
			until(t, a, "the send", func() bool { return len(session.log()) == 1 })
			if got := session.log()[0]; got != "hello" {
				t.Fatalf("sent %q, want hello", got)
			}
			if text := a.editor.WS.Composer(a.cur().Tab).Text; text != "" {
				t.Fatalf("composer keeps %q after sending", text)
			}
		})
	}
}

// The { } button makes indented JSON in the composer one line.
func TestTheFormatButtonCompactsTheComposersJSON(t *testing.T) {
	a, _ := openConnectedEcho(t)
	*a.editor.WS.Composer(a.cur().Tab) = widget.NewTextField("{\n  \"op\": \"ping\"\n}")
	click(a, formatButtonX, composerY)
	if got := a.editor.WS.Composer(a.cur().Tab).Text; got != `{"op":"ping"}` {
		t.Fatalf("composer = %q", got)
	}
}

// Right-clicking the composer opens the text menu.
func TestTheComposerHasTheTextMenu(t *testing.T) {
	a, _ := openConnectedEcho(t)
	rightClick(a, composerFieldX, composerY)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}

// A focused composer is a text widget with the keyboard: shortcuts must
// not take its keys.
func TestComposerFocusCountsAsTyping(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	if a.textFocused() {
		t.Fatal("typing with nothing focused")
	}
	a.editor.WS.Composer(a.cur().Tab).Focused = true
	if !a.textFocused() {
		t.Fatal("a focused composer is not typing")
	}
}
