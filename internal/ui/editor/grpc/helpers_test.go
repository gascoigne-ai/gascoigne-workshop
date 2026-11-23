package grpc

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
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

// toast is one message the component showed over the window.
type toast struct {
	text  string
	level severity.Level
}

// menu is one menu the component opened.
type menu struct {
	posX, posY int
	items      []widget.MenuItem
}

// fake is the host of the component: real tabs, and a record of what the
// component asked of it. Listings and file choices are held back until the
// test answers them, as the background work would.
type fake struct {
	set    *tabs.Set
	folder string
	vars   []request.KV

	listed   []request.Request
	answers  []func(methods []string, err error)
	titles   []string
	chosen   []func(path string)
	menus    []menu
	toasts   []toast
	statuses []string

	disconnected          []*tabs.Tab
	streamingAtDisconnect []bool
	pane                  *Component
}

func (fake *fake) IsOpen(tab *tabs.Tab) bool { return slices.Contains(fake.set.Tabs(), tab) }
func (fake *fake) List(req request.Request, answer func(methods []string, err error)) {
	fake.listed = append(fake.listed, req)
	fake.answers = append(fake.answers, answer)
}
func (fake *fake) VarsFor(*request.Request) []request.KV { return fake.vars }
func (fake *fake) Folder(*request.Request) string        { return fake.folder }
func (fake *fake) Pick(title string, answer func(path string)) {
	fake.titles = append(fake.titles, title)
	fake.chosen = append(fake.chosen, answer)
}
func (fake *fake) Disconnect(tab *tabs.Tab) {
	fake.disconnected = append(fake.disconnected, tab)
	fake.streamingAtDisconnect = append(fake.streamingAtDisconnect, fake.pane.Streaming(tab))
}
func (fake *fake) Open(posX, posY int, items []widget.MenuItem) {
	fake.menus = append(fake.menus, menu{posX, posY, items})
}
func (fake *fake) Toast(text string, level severity.Level) {
	fake.toasts = append(fake.toasts, toast{text, level})
}
func (fake *fake) SetStatus(text string) { fake.statuses = append(fake.statuses, text) }

// world is the component over a fake, with one open gRPC request in a
// collection folder.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	pane *Component
	tab  *tabs.Tab
	dir  string
}

func newWorld(t *testing.T) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	fake := &fake{set: tabs.NewSet(), folder: dir}
	fake.set.Open(&request.Request{Name: "R", Protocol: request.ProtocolGRPC, URL: "grpc://localhost:50051"})
	pane := New(Deps{Tabs: fake, Methods: fake, Vars: fake, Files: fake, Connections: fake, Window: fake})
	fake.pane = pane
	tab := fake.set.Current()
	tab.MarkSaved()
	return &world{t: t, ctx: ctx, fake: fake, pane: pane, tab: tab, dir: dir}
}

// protoPath is where the collection keeps the .proto file name.
func (world *world) protoPath(name string) string { return filepath.Join(world.dir, "protos", name) }

// answer ends the oldest listing nobody answered yet.
func (world *world) answer(methods []string, err error) {
	world.t.Helper()
	if len(world.fake.answers) == 0 {
		world.t.Fatal("no listing waits for an answer")
	}
	answer := world.fake.answers[0]
	world.fake.answers = world.fake.answers[1:]
	answer(methods, err)
}

// view builds the Proto pane of the open tab.
func (world *world) view() Node {
	var scroll widget.ScrollState
	return world.pane.View(world.ctx, world.tab, &scroll)
}

// text is the pane as lines of text.
func (world *world) text() string {
	root := world.view()
	Layout(world.ctx, root, wideW, paneH)
	return Dump(root)
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

// chipLabelled is the element whose only child is the text label, nil
// when there is none.
func chipLabelled(root Node, label string) Node {
	if kids := root.Children(); len(kids) == 1 && strings.HasPrefix(Dump(kids[0]), `textB text="`+label+`" `) {
		return root
	}
	for _, child := range root.Children() {
		if found := chipLabelled(child, label); found != nil {
			return found
		}
	}
	return nil
}

// removeButton is the pick of the nth Remove button of the pane.
func removeButton(n int) func(Node) Node {
	return func(root Node) Node {
		var found []Node
		var walk func(Node)
		walk = func(node Node) {
			if strings.Contains(strings.SplitN(Dump(node), "\n", 2)[0], "tip=Remove") {
				found = append(found, node)
			}
			for _, child := range node.Children() {
				walk(child)
			}
		}
		walk(root)
		if n >= len(found) {
			return nil
		}
		return found[n]
	}
}

// fieldAt is the pick of the nth text field of the pane.
func fieldAt(n int) func(Node) Node {
	return func(root Node) Node {
		fields := findWidgets(root, "*widget.fieldB")
		if n >= len(fields) {
			return nil
		}
		return fields[n]
	}
}

// frame builds the pane, lays it out at the wide size and runs one frame of
// input on it, over the node pick finds when it is set.
func (world *world) frame(in core.Input, pick func(root Node) Node) {
	world.t.Helper()
	root := world.view()
	Layout(world.ctx, root, wideW, paneH)
	if pick != nil {
		target := pick(root)
		if target == nil {
			world.t.Fatalf("nothing to point at in:\n%s", Dump(root))
		}
		at := target.Rect()
		in.X, in.Y = at.X+at.W/2, at.Y+at.H/2
	}
	world.ctx.In = in
	Update(world.ctx, root)
}

// clickOn presses the node pick finds and lets go.
func (world *world) clickOn(pick func(Node) Node) {
	world.t.Helper()
	world.frame(core.Input{Click: true, Held: true}, pick)
	world.frame(core.Input{}, nil)
}

// click presses the chip or button labelled label and lets go.
func (world *world) click(label string) {
	world.t.Helper()
	world.clickOn(func(root Node) Node { return chipLabelled(root, label) })
}

// typeInto clicks the nth text field and types text in it.
func (world *world) typeInto(n int, text string) {
	world.t.Helper()
	world.frame(core.Input{Click: true, Held: true}, fieldAt(n))
	world.frame(core.Input{}, nil)
	world.frame(core.Input{Typed: []rune(text)}, nil)
}
