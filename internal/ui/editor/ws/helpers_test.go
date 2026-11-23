package ws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/outbox"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the pane is pinned at: the main column of a 1280px window with the
// sidebar open and the narrowest main area, and the height of the pane.
const (
	wideW   = 1020
	narrowW = 420
	paneH   = 300
)

// toast is one message the pane showed over the window.
type toast struct {
	text  string
	level severity.Level
}

// menu is one menu the pane opened.
type menu struct {
	posX, posY int
	items      []widget.MenuItem
}

// fake is the host of the component: real tabs, the sends and forms it was
// asked for, and the protocol table of the host.
type fake struct {
	set     *tabs.Set
	queued  []outbox.Send
	forms   []request.MCPTool
	menus   []menu
	toasts  []toast
	pending map[request.Protocol]Composer
}

func (fake *fake) Current() *tabs.Tab { return fake.set.Current() }
func (fake *fake) IsOpen(tab *tabs.Tab) bool {
	return slices.Contains(fake.set.Tabs(), tab)
}
func (fake *fake) Queue(_ *tabs.Tab, send outbox.Send) { fake.queued = append(fake.queued, send) }
func (fake *fake) OpenToolForm(_ *tabs.Tab, tool request.MCPTool) {
	fake.forms = append(fake.forms, tool)
}
func (fake *fake) Open(posX, posY int, items []widget.MenuItem) {
	fake.menus = append(fake.menus, menu{posX, posY, items})
}
func (fake *fake) Toast(text string, level severity.Level) {
	fake.toasts = append(fake.toasts, toast{text, level})
}
func (fake *fake) Composer(tab *tabs.Tab) Composer { return fake.pending[tab.Req.Protocol] }

// composersOfTheApp is the protocol table the host keeps, as far as the
// pane reads it.
var composersOfTheApp = map[request.Protocol]Composer{
	request.ProtocolWebSocket: {Shown: true, FormatsJSON: true},
	request.ProtocolMCP:       {Shown: true, Hint: "call TOOL {json}  ·  read URI  ·  prompt NAME {json}  ·  method {json}  ·  or raw JSON-RPC"},
	request.ProtocolSSE:       {},
	request.ProtocolGRPC:      {Shown: true},
}

// world is the component over a fake, with one open request.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	pane *Component
	tab  *tabs.Tab
}

func newWorld(t *testing.T, protocol request.Protocol) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet(), pending: composersOfTheApp}
	fake.set.Open(&request.Request{Name: "R", Protocol: protocol})
	pane := New(Deps{Tabs: fake, Outbox: fake, Tools: fake, Window: fake, Composers: fake})
	return &world{t: t, ctx: ctx, fake: fake, pane: pane, tab: fake.set.Current()}
}

// open puts the tab in the open state with session, as a finished connect
// would.
func (world *world) open(session request.Session) {
	world.tab.Begin()
	world.tab.Opened(session)
}

// recorder is an open session that records what it is asked to send.
type recorder struct {
	mu   sync.Mutex
	sent []string
}

func (session *recorder) Events() <-chan request.Event { return nil }
func (session *recorder) Close(context.Context) error  { return nil }
func (session *recorder) Ping(ctx context.Context) error {
	return session.Send(ctx, "ping")
}
func (session *recorder) Send(_ context.Context, text string) error {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.sent = append(session.sent, text)
	return nil
}
func (session *recorder) log() []string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return slices.Clone(session.sent)
}

// binaryRecorder is a recorder that also takes binary messages.
type binaryRecorder struct {
	recorder
	binary [][]byte
}

func (session *binaryRecorder) SendBinary(_ context.Context, data []byte) error {
	session.binary = append(session.binary, data)
	return nil
}

// runAll runs the sends the pane queued, in order.
func (fake *fake) runAll(t *testing.T) {
	t.Helper()
	for _, send := range fake.queued {
		if err := send(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

// logText is the log of tab as plain text, a line each.
func (world *world) logText() string {
	var text strings.Builder
	for _, line := range world.pane.Lines(world.tab) {
		for _, span := range line.Spans {
			text.WriteString(span.Text)
		}
		text.WriteByte('\n')
	}
	return text.String()
}

// findWidgets is every node under root whose behavior has the Go type
// kind, in tree order.
func findWidgets(root Node, kind string) []Node {
	var found []Node
	if fmt.Sprintf("%T", root.Behavior()) == kind {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findWidgets(child, kind)...)
	}
	return found
}

// assertDumpPinned lays root out in a window of width by height and
// compares its dump with testdata/name.dump.
func assertDumpPinned(t *testing.T, ctx *core.Ctx, root Node, name string, width, height int) {
	t.Helper()
	Layout(ctx, root, width, height)
	want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Dump(root); got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// labelled is the element whose only child is the text label, nil when
// there is none.
func labelled(root Node, label string) Node {
	if kids := root.Children(); len(kids) == 1 && strings.HasPrefix(Dump(kids[0]), `textB text="`+label+`" `) {
		return root
	}
	for _, child := range root.Children() {
		if found := labelled(child, label); found != nil {
			return found
		}
	}
	return nil
}

// click builds the pane, lays it out and clicks the element labelled label.
func (world *world) click(label string) {
	world.t.Helper()
	root := world.pane.View(world.ctx, world.tab)
	Layout(world.ctx, root, wideW, paneH)
	target := labelled(root, label)
	if target == nil {
		world.t.Fatalf("nothing labelled %q in:\n%s", label, Dump(root))
	}
	at := target.Rect()
	world.ctx.In = core.Input{X: at.X + at.W/2, Y: at.Y + at.H/2, Click: true, Held: true}
	Update(world.ctx, root)
	world.ctx.In = core.Input{}
}
