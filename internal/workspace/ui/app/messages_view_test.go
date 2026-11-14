package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
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

// recordingSession is an open session that records what it is asked to
// send, so a test can see what the composer sent.
type recordingSession struct {
	mu   sync.Mutex
	sent []string
}

func (s *recordingSession) Events() <-chan reqport.Event { return nil }
func (s *recordingSession) Close() error                 { return nil }
func (s *recordingSession) Ping() error                  { return s.Send("ping") }
func (s *recordingSession) Send(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, text)
	return nil
}

func (s *recordingSession) log() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.sent...)
}

// messagesPaneStates are the looks of the Messages pane: a WebSocket before
// and after connecting (with a message in the log), an event stream, which
// has no composer, an MCP server with its pickers, and a gRPC stream, whose
// composer has no format button.
var messagesPaneStates = []struct {
	name        string
	placeholder string // the composer's hint; "" when there is no composer
	setUp       func(a *App, tab *reqTab)
}{
	{"websocket-idle", "Connect to send messages", func(a *App, tab *reqTab) {
		tab.Req.Protocol = "websocket"
	}},
	{"websocket-connected", "Message", func(a *App, tab *reqTab) {
		tab.Req.Protocol = "websocket"
		tab.Session = &recordingSession{}
		tab.Reconnect = true
		sentAt := time.Date(2026, 11, 14, 10, 30, 0, 0, time.UTC)
		a.logEvent(tab, reqport.Event{Time: sentAt, Dir: reqport.Sent, Kind: "text", Text: `{"op":"ping"}`})
		a.logEvent(tab, reqport.Event{Time: sentAt, Dir: reqport.Received, Kind: "text", Text: `{"op":"ping"}`})
	}},
	{"sse", "", func(a *App, tab *reqTab) {
		tab.Req.Protocol = "sse"
	}},
	{"mcp-connected", "call TOOL {json}  ·  read URI  ·  prompt NAME {json}  ·  method {json}  ·  or raw JSON-RPC", func(a *App, tab *reqTab) {
		tab.Req.Protocol = "mcp"
		tab.Session = &recordingSession{}
		tab.Tools = []model.MCPTool{{Name: "search"}}
		tab.Resources = []model.MCPResource{{URI: "file:///a.txt"}}
		tab.Prompts = []model.MCPPrompt{{Name: "review"}}
	}},
	{"grpc-stream", "Message", func(a *App, tab *reqTab) {
		tab.Req.Protocol = "grpc"
		tab.grpcStream = true
		tab.Session = &recordingSession{}
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
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
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
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
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
			for _, want := range []string{"field:" + fmt.Sprintf("%p", &tab.msg), "placeholder:" + state.placeholder + " segments:"} {
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
func TestTheMessageLogShowsTheTabsView(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	code := findWidget(a.messagesPane(wsPanes[0], tab, nil), "widget.codeB")
	if code == nil {
		t.Fatal("no message log")
	}
	if got, want := fmt.Sprintf("%+v", code.Behavior()), fmt.Sprintf("{Base:{} view:%p}", &tab.wsView); got != want {
		t.Fatalf("log %s, want %s", got, want)
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
func openConnectedEcho(t *testing.T) (*App, *recordingSession) {
	t.Helper()
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	openNamed(t, a, "Echo")
	session := &recordingSession{}
	a.cur().Session = session
	a.step(core.Input{})
	return a, session
}

// Typed text goes out on Enter and on the Send button, and the composer is
// emptied for the next message.
func TestTheComposerSendsOnEnterAndOnSend(t *testing.T) {
	cases := []struct {
		name string
		send func(a *App)
	}{
		{"enter", func(a *App) { a.step(core.Input{Enter: true}) }},
		{"send button", func(a *App) { click(a, sendButtonX, composerY) }},
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
			if text := a.cur().msg.Text; text != "" {
				t.Fatalf("composer keeps %q after sending", text)
			}
		})
	}
}

// The { } button makes indented JSON in the composer one line.
func TestTheFormatButtonCompactsTheComposersJSON(t *testing.T) {
	a, _ := openConnectedEcho(t)
	a.cur().msg = widget.NewTextField("{\n  \"op\": \"ping\"\n}")
	click(a, formatButtonX, composerY)
	if got := a.cur().msg.Text; got != `{"op":"ping"}` {
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
