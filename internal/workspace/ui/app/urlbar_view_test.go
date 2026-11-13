package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the URL bar is pinned at: the main column of the test window,
// and the narrowest one the sidebar divider allows.
const (
	urlWideW   = winW - sidebarW
	urlNarrowW = minMainW
)

// grpcMethod is the method the gRPC states call.
const grpcMethod = "grpc.health.v1.Health/Check"

// urlBarState is a named look of the URL bar and how to give the open
// tab that look.
type urlBarState struct {
	name  string
	setUp func(t *testing.T, a *App, tab *reqTab)
}

// urlBarStates put the open tab in each look the URL bar has: one per
// protocol, the gRPC chip empty and streaming, an open connection, a
// request in flight and a URL with variables, which has a tooltip.
var urlBarStates = []urlBarState{
	{"rest-get", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol, tab.Req.Method = "rest", "GET"
		pointTabAt(tab, "https://api.example.test/users")
	}},
	{"rest-post-vars", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol, tab.Req.Method = "rest", "POST"
		pointTabAt(tab, "{{base}}/users/{{id}}")
	}},
	{"websocket", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol = "websocket"
		pointTabAt(tab, "ws://127.0.0.1:1/")
	}},
	{"websocket-connected", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol = "websocket"
		pointTabAt(tab, echoServer(t))
		a.toggleConnection(tab)
		t.Cleanup(func() { a.disconnect(tab) })
	}},
	{"sse", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol = "sse"
		pointTabAt(tab, "http://127.0.0.1:1/events")
	}},
	{"mcp", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol = "mcp"
		pointTabAt(tab, "http://127.0.0.1:1/mcp")
	}},
	{"grpc-no-method", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol, tab.Req.Method = "grpc", ""
		pointTabAt(tab, "grpc://127.0.0.1:1")
	}},
	{"grpc-method", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol, tab.Req.Method = "grpc", grpcMethod
		pointTabAt(tab, "grpc://127.0.0.1:1")
	}},
	{"grpc-stream", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol, tab.Req.Method = "grpc", grpcMethod
		pointTabAt(tab, "grpc://127.0.0.1:1")
		a.toggleStream(tab)
	}},
	{"sending", func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Protocol, tab.Req.Method = "rest", "GET"
		pointTabAt(tab, silentServer(t))
		a.send(tab)
	}},
}

// The URL bar is pinned as text (element kinds, options, handlers,
// rectangles) at two widths in every look, so moving it to markup cannot
// change a pixel unnoticed.
func TestTheURLBarLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", urlWideW},
		{"narrow", urlNarrowW},
	}
	for _, state := range urlBarStates {
		for _, size := range sizes {
			name := "urlbar-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				tab := a.cur()
				state.setUp(t, a, tab)
				root := a.urlBar(tab)
				Layout(a.ctx, root, size.width, urlH)
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

// Points of the URL bar in the test window: the protocol chip, the method
// chip and the Stream chip right of it (gRPC), and the URL field.
const (
	protocolChipX = sidebarW + core.Gutter + 32
	methodChipX   = protocolChipX + 32 + 1 + 42
	streamChipX   = protocolChipX + 32 + 1 + grpcMethodW + 28
	urlFieldX     = 1100
)

// Every part of the URL bar says what it does on hover; the field tells
// what its variables resolve to.
func TestTheURLBarsPartsHaveTooltips(t *testing.T) {
	cases := []struct {
		name  string
		state string
		x     int
		tip   string
	}{
		{"protocol", "rest-get", protocolChipX, "Protocol: click to change"},
		{"method", "rest-get", methodChipX, "Method: click to change"},
		{"grpc method", "grpc-method", methodChipX, "Method: click to list the server's methods"},
		{"stream", "grpc-method", streamChipX, "Stream: keep the call open and show every message in the Messages pane"},
		{"send", "rest-get", sendX, "Send the request (Ctrl+Enter)"},
		{"cancel", "sending", sendX, "Cancel the running request"},
		{"connect", "websocket", sendX, "Open the connection (Ctrl+Enter)"},
		{"disconnect", "websocket-connected", sendX, "Close the connection (Ctrl+Enter)"},
		{"field", "rest-post-vars", urlFieldX, "Resolves to {{base}}/users/{{id}}\nNot defined: base, id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			stateNamed(t, tc.state).setUp(t, a, a.cur())
			if got := tipAt(a, tc.x, urlBarY); got != tc.tip {
				t.Fatalf("tooltip %q, want %q", got, tc.tip)
			}
		})
	}
}

// silentServer takes requests and never answers them, so a request sent
// to it stays in flight until it is cancelled.
func silentServer(t *testing.T) string {
	t.Helper()
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })
	return srv.URL
}

// stateNamed is the URL bar state called name.
func stateNamed(t *testing.T, name string) urlBarState {
	t.Helper()
	for _, state := range urlBarStates {
		if state.name == name {
			return state
		}
	}
	t.Fatalf("no URL bar state %q", name)
	return urlBarStates[0]
}

// Clicking a chip changes what it shows: the protocol and the method
// cycle, Stream turns streaming on.
func TestClickingTheURLBarsChipsChangesTheRequest(t *testing.T) {
	cases := []struct {
		name  string
		state string
		x     int
		check func(tab *reqTab) bool
	}{
		{"protocol", "rest-get", protocolChipX, func(tab *reqTab) bool { return tab.Req.Protocol != "rest" }},
		{"method", "rest-get", methodChipX, func(tab *reqTab) bool { return tab.Req.Method != "GET" }},
		{"stream", "grpc-method", streamChipX, func(tab *reqTab) bool { return tab.grpcStream }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			tab := a.cur()
			stateNamed(t, tc.state).setUp(t, a, tab)
			click(a, tc.x, urlBarY)
			if !tc.check(tab) {
				t.Fatalf("after the click: protocol %q, method %q, stream %v", tab.Req.Protocol, tab.Req.Method, tab.grpcStream)
			}
		})
	}
}

// The button of the bar does what its label says: Send starts the
// request, Cancel stops it, Connect opens the connection.
func TestTheURLBarsButtonFollowsItsLabel(t *testing.T) {
	cases := []struct {
		name  string
		state string
		check func(tab *reqTab) bool
	}{
		{"send", "rest-get", func(tab *reqTab) bool { return tab.Sending }},
		{"cancel", "sending", func(tab *reqTab) bool { return !tab.Sending }},
		{"connect", "websocket", func(tab *reqTab) bool { return tab.WantOpen }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			tab := a.cur()
			stateNamed(t, tc.state).setUp(t, a, tab)
			click(a, sendX, urlBarY)
			if !tc.check(tab) {
				t.Fatalf("after the click: sending %v, connection wanted %v", tab.Sending, tab.WantOpen)
			}
			a.cancelSend(tab)
			a.disconnect(tab)
		})
	}
}

// Typing into the URL field edits the request, which Dump does not show,
// and Enter sends it.
func TestTypingAURLEditsTheRequestAndEnterSendsIt(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	stateNamed(t, "rest-get").setUp(t, a, tab)
	click(a, urlFieldX, urlBarY)
	a.typeText("/1")
	if tab.Req.URL != "https://api.example.test/users/1" {
		t.Fatalf("URL %q", tab.Req.URL)
	}
	a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyEnter}, Enter: true})
	if !tab.Sending {
		t.Fatal("Enter in the URL field should send the request")
	}
	a.cancelSend(tab)
}

// The URL field edits the tab's URL, shows a placeholder while empty,
// colors its variables and sends on Enter: settings Dump does not print,
// read from the field's own widget.
func TestTheURLFieldIsWiredToTheTab(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	field := findWidget(a.urlBar(tab), "*widget.fieldB")
	if field == nil {
		t.Fatal("the URL bar has no field")
	}
	settings := fmt.Sprintf("%+v", field.Behavior())
	for _, want := range []string{
		fmt.Sprintf("field:%p", &tab.url),
		"placeholder:Enter URL ",
		"segments:0x",
		"onEnter:0x",
	} {
		if !strings.Contains(settings, want) {
			t.Errorf("the URL field lacks %q: %s", want, settings)
		}
	}
}

// findWidget is the first element under root whose behavior has the Go
// type kind.
func findWidget(root Node, kind string) Node {
	if fmt.Sprintf("%T", root.Behavior()) == kind {
		return root
	}
	for _, child := range root.Children() {
		if found := findWidget(child, kind); found != nil {
			return found
		}
	}
	return nil
}

// Right-clicking the URL field opens the text menu.
func TestTheURLFieldHasTheTextMenu(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	rightClick(a, urlFieldX, urlBarY)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}
