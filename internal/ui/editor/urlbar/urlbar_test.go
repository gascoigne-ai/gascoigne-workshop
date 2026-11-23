package urlbar

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the bar is pinned at: the main column of a 1280px window with the
// default sidebar, and the narrowest the dividers allow.
const (
	wideW   = 1020
	narrowW = 420
)

// grpcMethod is the method the gRPC states call.
const grpcMethod = "grpc.health.v1.Health/Check"

// fakeSession is an open connection that does nothing.
type fakeSession struct{}

func (fakeSession) Events() <-chan request.Event                { return nil }
func (fakeSession) Send(ctx context.Context, text string) error { return nil }
func (fakeSession) Ping(ctx context.Context) error              { return nil }
func (fakeSession) Close(ctx context.Context) error             { return nil }

// fake is every dependency of the bar at once: a real tab list, and a
// record of what the bar asked of the host.
type fake struct {
	set       *tabs.Set
	sending   map[*tabs.Tab]bool
	streaming map[*tabs.Tab]bool
	asked     []string // what the bar asked for, in order
}

func (fake *fake) Current() *tabs.Tab { return fake.set.Current() }
func (fake *fake) IsOpen(tab *tabs.Tab) bool {
	for _, open := range fake.set.Tabs() {
		if open == tab {
			return true
		}
	}
	return false
}
func (fake *fake) Send(*tabs.Tab)                   { fake.asked = append(fake.asked, "send") }
func (fake *fake) Cancel(*tabs.Tab)                 { fake.asked = append(fake.asked, "cancel") }
func (fake *fake) Sending(tab *tabs.Tab) bool       { return fake.sending[tab] }
func (fake *fake) ToggleConnection(*tabs.Tab)       { fake.asked = append(fake.asked, "toggle-connection") }
func (fake *fake) PickMethod(*tabs.Tab)             { fake.asked = append(fake.asked, "pick-method") }
func (fake *fake) Streaming(tab *tabs.Tab) bool     { return fake.streaming[tab] }
func (fake *fake) ToggleStream(tab *tabs.Tab)       { fake.streaming[tab] = !fake.streaming[tab] }
func (fake *fake) Hint(label, action string) string { return label + " (Ctrl+Enter)" }

func (fake *fake) Socket(tab *tabs.Tab) bool {
	connection := tab.Req.Protocol.Info().Connection
	return connection == request.Always || (connection == request.OnStream && fake.streaming[tab])
}

func (fake *fake) Offered(current request.Protocol) []request.Protocol {
	return []request.Protocol{request.ProtocolREST, request.ProtocolWebSocket, request.ProtocolMCP}
}

func (fake *fake) Choose(tab *tabs.Tab, protocol request.Protocol) { tab.Req.Protocol = protocol }

// Segments colors nothing: the dump does not show colors of a field's text.
func (fake *fake) Segments(req *request.Request, raw string) []widget.Span { return nil }

// Tip is what the URL resolves to when no variable is defined.
func (fake *fake) Tip(tab *tabs.Tab) string {
	names := request.VarNames(tab.Req.URL)
	if len(names) == 0 {
		return ""
	}
	return "Resolves to " + request.Expand(tab.Req.URL, nil) + "\nNot defined: " + strings.Join(names, ", ")
}

// world is a bar over a fake, driven frame by frame like the window does.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	bar  *Component
	tab  *tabs.Tab
}

// newWorld is a bar over one open request.
func newWorld(t *testing.T, req *request.Request) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet(), sending: map[*tabs.Tab]bool{}, streaming: map[*tabs.Tab]bool{}}
	fake.set.Open(req)
	bar := New(Deps{Tabs: fake, Sends: fake, Connections: fake, GRPC: fake, Protocols: fake, Styles: fake, Hints: fake})
	return &world{t: t, ctx: ctx, fake: fake, bar: bar, tab: fake.set.Current()}
}

// look is a named look of the bar and how to give the open tab that look.
type look struct {
	name  string
	setUp func(world *world)
}

func (world *world) point(protocol request.Protocol, method, url string) {
	world.tab.Req.Protocol, world.tab.Req.Method, world.tab.Req.URL = protocol, method, url
}

// looks put the open tab in each look the bar has.
var looks = []look{
	{"rest-get", func(world *world) { world.point("rest", "GET", "https://api.example.test/users") }},
	{"rest-post-vars", func(world *world) { world.point("rest", "POST", "{{base}}/users/{{id}}") }},
	{"websocket", func(world *world) { world.point("websocket", "", "ws://127.0.0.1:1/") }},
	{"websocket-connected", func(world *world) {
		world.point("websocket", "", "ws://127.0.0.1:1/")
		world.tab.Begin()
		world.tab.Opened(fakeSession{})
	}},
	{"sse", func(world *world) { world.point("sse", "", "http://127.0.0.1:1/events") }},
	{"mcp", func(world *world) { world.point("mcp", "", "http://127.0.0.1:1/mcp") }},
	{"grpc-no-method", func(world *world) { world.point("grpc", "", "grpc://127.0.0.1:1") }},
	{"grpc-method", func(world *world) { world.point("grpc", grpcMethod, "grpc://127.0.0.1:1") }},
	{"grpc-stream", func(world *world) {
		world.point("grpc", grpcMethod, "grpc://127.0.0.1:1")
		world.fake.streaming[world.tab] = true
	}},
	{"sending", func(world *world) {
		world.point("rest", "GET", "http://127.0.0.1:1/")
		world.fake.sending[world.tab] = true
	}},
}

// The bar is pinned as text (element kinds, options, handlers, rects) at two
// widths for each look, so a change of the template cannot change it unseen.
func TestTheURLBarLaysOutAsPinned(t *testing.T) {
	sizesToPin := []struct {
		name  string
		width int
	}{
		{"wide", wideW},
		{"narrow", narrowW},
	}
	for _, state := range looks {
		for _, size := range sizesToPin {
			name := "urlbar-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t, &request.Request{Name: "R"})
				state.setUp(world)
				root := world.bar.View(world.ctx)
				Layout(world.ctx, root, size.width, sizes.URLBarH)
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

func TestNext(t *testing.T) {
	chipOrder := []request.Protocol{"rest", "websocket", "mcp", "sse", "grpc"}
	tests := []struct {
		name      string
		cur, want request.Protocol
	}{
		{"advances", "rest", "websocket"},
		{"wraps", "grpc", "rest"},
		{"unknown starts over", "carrier-pigeon", "rest"},
	}
	for _, tt := range tests {
		if got := next(chipOrder, tt.cur); got != tt.want {
			t.Errorf("%s: next(%q) = %q, want %q", tt.name, tt.cur, got, tt.want)
		}
	}
}

func TestTheMethodChipCyclesThroughMethodOrderAndWraps(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Protocol: "rest", Method: "GET", URL: "http://x"})
	for _, want := range []string{"POST", "PUT", "PATCH", "DELETE", "GET"} {
		world.bar.CycleMethod(world.tab)
		if world.tab.Req.Method != want {
			t.Fatalf("method %q, want %q", world.tab.Req.Method, want)
		}
	}
}

// What the bar claims of a frame while its field has the keyboard: Undo
// and Redo are the field's, and so is a frame of typing keys, which no
// shortcut may take (a binding with no Ctrl or Alt waits for the keyboard).
// Escape is never claimed: it still cancels a running request. A shortcut
// such as Ctrl+Enter or Ctrl+Space is the application's.
func TestURLFieldClaimsUndoAndTypedKeysWhileFocused(t *testing.T) {
	tests := []struct {
		name    string
		focused bool
		hidden  bool // the focused field belongs to a tab that is not shown
		input   core.Input
		want    bool
	}{
		{"focused and Undo", true, false, core.Input{Undo: true, Ctrl: true, Keys: []ebiten.Key{ebiten.KeyZ}}, true},
		{"focused and Redo", true, false, core.Input{Redo: true, Ctrl: true, Keys: []ebiten.Key{ebiten.KeyY}}, true},
		{"focused and a letter", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyA}}, true},
		{"focused and an arrow", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyArrowLeft}}, true},
		{"focused and Escape", true, false, core.Input{Escape: true, Keys: []ebiten.Key{ebiten.KeyEscape}}, false},
		{"focused and Ctrl+Enter", true, false, core.Input{Ctrl: true, Keys: []ebiten.Key{ebiten.KeyEnter}}, false},
		{"focused and Ctrl+Space", true, false, core.Input{Ctrl: true, Keys: []ebiten.Key{ebiten.KeySpace}}, false},
		{"focused and a function key", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyF5}}, false},
		{"focused and no key", true, false, core.Input{}, false},
		{"not focused and Undo", false, false, core.Input{Undo: true, Ctrl: true, Keys: []ebiten.Key{ebiten.KeyZ}}, false},
		{"not focused and a letter", false, false, core.Input{Keys: []ebiten.Key{ebiten.KeyA}}, false},
		{"focused field of a hidden tab", true, true, core.Input{Keys: []ebiten.Key{ebiten.KeyA}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			world := newWorld(t, &request.Request{Name: "first", URL: "http://a"})
			first := world.tab
			world.fake.set.Open(&request.Request{Name: "second", URL: "http://b"})
			focus := first
			if !test.hidden {
				world.fake.set.Apply(tabs.Activate{Index: 0})
			}
			world.bar.Field(focus).Focused = test.focused
			if got := world.bar.Keys(test.input); got != test.want {
				t.Fatalf("Keys = %v, want %v", got, test.want)
			}
			if typing := world.bar.Typing(); typing != (test.focused && !test.hidden) {
				t.Fatalf("Typing = %v", typing)
			}
		})
	}
}

// The Params table rewrites the URL; the field then starts over with the
// request's URL, caret and undo history included.
func TestReloadTakesTheRequestURL(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", URL: "http://x/"})
	field := world.bar.Field(world.tab)
	if field.Text != "http://x/" {
		t.Fatalf("a new field holds %q", field.Text)
	}
	world.tab.Req.URL = "http://x/?a=1"
	world.bar.Reload(world.tab)
	if world.bar.Field(world.tab) != field || field.Text != "http://x/?a=1" {
		t.Fatalf("the field keeps its place and takes the new URL: %q", field.Text)
	}
}

func TestTickDropsTheFieldOfAClosedTab(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "first", URL: "http://a"})
	first := world.tab
	world.fake.set.Open(&request.Request{Name: "second", URL: "http://b"})
	firstField := world.bar.Field(first)
	firstField.Text = "typed and not saved"
	world.fake.set.Apply(tabs.Close{Index: 0})
	world.bar.Tick()
	if reopened := world.bar.Field(first); reopened.Text != "http://a" {
		t.Fatalf("the field of a closed tab was kept: %q", reopened.Text)
	}
}

func TestTypingIntoTheFieldEditsTheRequest(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Protocol: "rest", Method: "GET", URL: "http://x/"})
	world.bar.Type(world.tab, "y")
	if world.tab.Req.URL != "http://x/y" || world.bar.Field(world.tab).Text != "http://x/y" {
		t.Fatalf("request %q, field %q", world.tab.Req.URL, world.bar.Field(world.tab).Text)
	}
}

func TestSendAndFocusURLActOnTheCurrentTab(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", URL: "http://x/"})
	world.bar.Send()
	if len(world.fake.asked) != 1 || world.fake.asked[0] != "send" {
		t.Fatalf("asked %v", world.fake.asked)
	}
	world.bar.FocusURL()
	if !world.bar.Typing() || !world.bar.Field(world.tab).HasSel {
		t.Fatal("FocusURL puts the keyboard in the field with its text selected")
	}
}
