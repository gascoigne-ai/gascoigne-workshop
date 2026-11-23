package body

import (
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the pane is pinned at: the main column of a 1280px window with the
// default sidebar, and the narrowest the dividers allow; the height the
// request pane has in the test window.
const (
	wideW   = 1020
	narrowW = 420
	paneH   = 300
)

var paneSizes = []struct {
	name  string
	width int
}{
	{"wide", wideW},
	{"narrow", narrowW},
}

// pendingCheck is a question about a path the fake has not answered yet.
type pendingCheck struct {
	path   string
	answer func(found bool)
}

// fake is every dependency of the pane at once: a real tab list, and a
// record of what the pane asked of the host.
type fake struct {
	set       *tabs.Set
	vars      []request.KV
	asked     []string // what the pane asked for, in order
	status    string
	shown     []*tabs.Tab
	checks    []pendingCheck
	statFiles bool // answer a check at once from the disk
	picks     []func(path string)
	describes []func(schema request.MessageSchema, err error)
}

func (fake *fake) Current() *tabs.Tab { return fake.set.Current() }
func (fake *fake) IsOpen(tab *tabs.Tab) bool {
	return slices.Contains(fake.set.Tabs(), tab)
}

// FormTable is a plain node: the real table is another component's.
func (fake *fake) FormTable(*tabs.Tab, *widget.ScrollState) Node { return Text("form table") }
func (fake *fake) GraphQL(*tabs.Tab) Node                        { return Text("graphql body") }
func (fake *fake) Message(text string) Node                      { return Text(text) }

func (fake *fake) VarsFor(*request.Request) []request.KV { return fake.vars }

func (fake *fake) Check(path string, answer func(found bool)) {
	if fake.statFiles {
		info, err := os.Stat(path)
		answer(err == nil && !info.IsDir())
		return
	}
	fake.checks = append(fake.checks, pendingCheck{path, answer})
}

func (fake *fake) Pick(title string, answer func(path string)) {
	fake.asked = append(fake.asked, "pick "+title)
	fake.picks = append(fake.picks, answer)
}

func (fake *fake) Describe(tab *tabs.Tab, answer func(request.MessageSchema, error)) {
	fake.asked = append(fake.asked, "describe")
	fake.describes = append(fake.describes, answer)
}

func (fake *fake) CommitEdit()            { fake.asked = append(fake.asked, "commit-edit") }
func (fake *fake) ShowBody(tab *tabs.Tab) { fake.shown = append(fake.shown, tab) }
func (fake *fake) SetStatus(text string)  { fake.status = text }

// world is a pane over a fake, driven like the window does.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	pane *Component
	tab  *tabs.Tab
}

// newWorld is a pane over one open request.
func newWorld(t *testing.T, req *request.Request) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet()}
	fake.set.Open(req)
	pane := New(Deps{Tabs: fake, Slots: fake, Vars: fake, Files: fake, Schemas: fake, Window: fake})
	return &world{t: t, ctx: ctx, fake: fake, pane: pane, tab: fake.set.Current()}
}

// view builds the Body pane of the open tab.
func (world *world) view() Node {
	var scroll widget.ScrollState
	return world.pane.View(world.ctx, world.tab, &scroll)
}

// open is a world whose request has a body of bodyType holding text, chosen as
// its chip does.
func open(t *testing.T, bodyType request.BodyType, text string) *world {
	t.Helper()
	world := newWorld(t, &request.Request{Name: "R", Protocol: "rest", Method: "GET", URL: "http://x"})
	*world.pane.Area(world.tab) = widget.NewTextArea(text)
	world.pane.ChooseType(world.tab, bodyType)
	return world
}

func TestTheBodyStateIsMadeFromTheRequestBodyOnFirstUse(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Body: "hello"})
	area, file := world.pane.Area(world.tab), world.pane.File(world.tab)
	if area.Text != "hello" || file.Text != "hello" {
		t.Fatalf("area %q file %q, want both hello", area.Text, file.Text)
	}
	if world.pane.Area(world.tab) != area || world.pane.File(world.tab) != file {
		t.Fatal("the state must keep its place between calls: views hold pointers into it")
	}
}

func TestTickDropsTheStateOfAClosedTab(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "first", Body: "a"})
	first := world.tab
	world.fake.set.Open(&request.Request{Name: "second", Body: "b"})
	world.pane.Area(first).SelectAll()
	world.pane.Area(first).Insert("typed and not saved")
	world.fake.set.Apply(tabs.Close{Index: 0})
	if world.pane.Tick() {
		t.Fatal("dropping a state changes nothing on screen")
	}
	if reopened := world.pane.Area(first); reopened.Text != "a" {
		t.Fatalf("the state of a closed tab was kept: %q", reopened.Text)
	}
}

// Keys claims Undo and typing while the open tab's area or path field has the
// keyboard, so a table's Undo and a shortcut with no Ctrl or Alt wait; Escape and
// Ctrl shortcuts are never claimed.
func TestTheBodyClaimsUndoAndTypedKeysWhileFocused(t *testing.T) {
	letter := core.Input{Keys: []ebiten.Key{ebiten.KeyA}}
	tests := []struct {
		name   string
		focus  string // "area", "file" or "" for none
		hidden bool   // the focused widget belongs to a tab that is not shown
		input  core.Input
		want   bool
	}{
		{"area and Undo", "area", false, core.Input{Undo: true, Ctrl: true, Keys: []ebiten.Key{ebiten.KeyZ}}, true},
		{"area and Redo", "area", false, core.Input{Redo: true, Ctrl: true, Keys: []ebiten.Key{ebiten.KeyY}}, true},
		{"area and a letter", "area", false, letter, true},
		{"area and an arrow", "area", false, core.Input{Keys: []ebiten.Key{ebiten.KeyArrowLeft}}, true},
		{"area and Escape", "area", false, core.Input{Escape: true, Keys: []ebiten.Key{ebiten.KeyEscape}}, false},
		{"area and Ctrl+Enter", "area", false, core.Input{Ctrl: true, Keys: []ebiten.Key{ebiten.KeyEnter}}, false},
		{"area and a function key", "area", false, core.Input{Keys: []ebiten.Key{ebiten.KeyF5}}, false},
		{"area and no key", "area", false, core.Input{}, false},
		{"file and a letter", "file", false, letter, true},
		{"file and Undo", "file", false, core.Input{Undo: true, Ctrl: true, Keys: []ebiten.Key{ebiten.KeyZ}}, true},
		{"nothing focused and a letter", "", false, letter, false},
		{"nothing focused and Undo", "", false, core.Input{Undo: true, Ctrl: true, Keys: []ebiten.Key{ebiten.KeyZ}}, false},
		{"area of a hidden tab", "area", true, letter, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			world := newWorld(t, &request.Request{Name: "first"})
			first := world.tab
			world.fake.set.Open(&request.Request{Name: "second"})
			if !test.hidden {
				world.fake.set.Apply(tabs.Activate{Index: 0})
			}
			switch test.focus {
			case "area":
				world.pane.Area(first).Focused = true
			case "file":
				world.pane.File(first).Focused = true
			}
			if got := world.pane.Keys(test.input); got != test.want {
				t.Fatalf("Keys = %v, want %v", got, test.want)
			}
			if typing := world.pane.Typing(); typing != (test.focus != "" && !test.hidden) {
				t.Fatalf("Typing = %v", typing)
			}
		})
	}
}

// A table cell edit in progress is written first, so the type change sees
// the rows it leaves.
func TestChoosingABodyTypeCommitsTheCellEditFirst(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", BodyType: request.BodyJSON, Body: "path.bin"})
	world.pane.Area(world.tab).SelectAll()
	world.pane.Area(world.tab).Insert("typed")
	world.pane.File(world.tab).SelectAll()
	world.pane.File(world.tab).Insert("/tmp/file.bin")

	world.pane.ChooseType(world.tab, request.BodyFile)
	if !reflect.DeepEqual(world.fake.asked, []string{"commit-edit"}) {
		t.Fatalf("asked %v, want the commit", world.fake.asked)
	}
	if world.tab.Req.BodyType != request.BodyFile || world.tab.Req.Body != "/tmp/file.bin" {
		t.Fatalf("type %q body %q, want File with the path field's text", world.tab.Req.BodyType, world.tab.Req.Body)
	}
	world.pane.ChooseType(world.tab, request.BodyText)
	if world.tab.Req.BodyType != request.BodyText || world.tab.Req.Body != "typed" {
		t.Fatalf("type %q body %q, want Text with the area's text", world.tab.Req.BodyType, world.tab.Req.Body)
	}
}

func TestReplaceTypesTextOverTheAreaAsOneUndoableEdit(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", BodyType: request.BodyJSON, Body: "old"})
	world.pane.Replace(world.tab, "new")
	if area := world.pane.Area(world.tab); area.Text != "new" || world.tab.Req.Body != "new" {
		t.Fatalf("area %q request %q, want new in both", area.Text, world.tab.Req.Body)
	}
	world.pane.Area(world.tab).Undo()
	if got := world.pane.Area(world.tab).Text; got != "old" {
		t.Fatalf("after Undo the area holds %q, want old", got)
	}
}

func TestFormatReindentsJSONAndReportsInvalidJSON(t *testing.T) {
	world := open(t, request.BodyJSON, `{"a":1}`)
	world.pane.format(world.tab)
	if want := "{\n  \"a\": 1\n}"; world.tab.Req.Body != want {
		t.Fatalf("body %q, want %q", world.tab.Req.Body, want)
	}
	world.pane.Replace(world.tab, `{"a":`)
	world.pane.format(world.tab)
	if world.tab.Req.Body != `{"a":` || world.fake.status == "" {
		t.Fatalf("body %q status %q: invalid JSON is left alone and reported", world.tab.Req.Body, world.fake.status)
	}
}

func TestOffersEveryBodyChipAndNothingElse(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R"})
	for _, id := range []request.BodyType{request.BodyNone, request.BodyJSON, request.BodyText, request.BodyForm,
		request.BodyMultipart, request.BodyGraphQL, request.BodyFile} {
		if !world.pane.Offers(id) {
			t.Errorf("%q is not offered", id)
		}
	}
	if world.pane.Offers("carrier-pigeon") {
		t.Error("an unknown type is offered")
	}
}

// grpcWorld is a pane whose request names a gRPC method.
func grpcWorld(t *testing.T) *world {
	return newWorld(t, &request.Request{Name: "R", Protocol: "grpc", URL: "grpc://127.0.0.1:1", Method: "grpc.health.v1.Health/Check"})
}

func (world *world) schemaView() Node {
	var scroll widget.ScrollState
	return world.pane.SchemaView(world.ctx, world.tab, &scroll)
}

func healthSchema() request.MessageSchema {
	return request.MessageSchema{Name: "HealthCheckRequest", Fields: []request.SchemaField{{Name: "service", Type: "string"}}}
}

func TestFillBodyReplacesTheTextAndShowsTheBodyPane(t *testing.T) {
	world := grpcWorld(t)
	schema := healthSchema()
	world.pane.fill(world.tab, &schema)
	if want := `{"service":""}`; world.tab.Req.Body != want || world.pane.Area(world.tab).Text != want {
		t.Fatalf("body %q area %q, want %q", world.tab.Req.Body, world.pane.Area(world.tab).Text, want)
	}
	if !slices.Equal(world.fake.shown, []*tabs.Tab{world.tab}) {
		t.Fatalf("shown %v, want the Body pane of the tab", world.fake.shown)
	}
	world.pane.fill(world.tab, nil)
	if len(world.fake.shown) != 1 {
		t.Fatal("no schema, nothing to fill")
	}
}

func TestTheSchemaIsAskedOncePerRequestAndReloadAsksAgain(t *testing.T) {
	world := grpcWorld(t)
	world.schemaView()
	world.schemaView()
	if len(world.fake.describes) != 1 {
		t.Fatalf("asked %d times while the answer is on its way, want 1", len(world.fake.describes))
	}
	world.fake.describes[0](healthSchema(), nil)
	world.schemaView()
	if len(world.fake.describes) != 1 {
		t.Fatalf("asked %d times after the answer, want 1", len(world.fake.describes))
	}
	world.pane.reload(world.tab)
	world.schemaView()
	if len(world.fake.describes) != 2 {
		t.Fatalf("asked %d times after Reload, want 2", len(world.fake.describes))
	}
	world.fake.describes[1](request.MessageSchema{}, os.ErrNotExist)
	world.tab.Req.Method = "grpc.health.v1.Health/Watch"
	world.schemaView()
	if len(world.fake.describes) != 3 {
		t.Fatalf("asked %d times after another method, want 3", len(world.fake.describes))
	}
}

func TestSchemaPendingWhileAskedOrRunning(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Protocol: "grpc", Method: ""})
	if world.pane.SchemaPending(world.tab) {
		t.Fatal("with no method there is nothing to wait for")
	}
	world.tab.Req.Method = "grpc.health.v1.Health/Check"
	if !world.pane.SchemaPending(world.tab) {
		t.Fatal("a method not asked about yet is pending")
	}
	world.schemaView()
	if !world.pane.SchemaPending(world.tab) {
		t.Fatal("a question on its way is pending")
	}
	world.fake.describes[0](healthSchema(), nil)
	if world.pane.SchemaPending(world.tab) {
		t.Fatal("an answered question is not pending")
	}
	world.tab.Req.Method = "grpc.health.v1.Health/Watch"
	if !world.pane.SchemaPending(world.tab) {
		t.Fatal("another method is pending again")
	}
}

func TestAFileCheckAnswerForAnOlderPathIsDropped(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", BodyType: request.BodyFile, Body: "/old.bin"})
	world.view()
	world.view()
	if len(world.fake.checks) != 1 {
		t.Fatalf("%d checks for one path, want 1", len(world.fake.checks))
	}
	*world.pane.File(world.tab) = widget.NewTextField("/new.bin")
	world.view()
	if len(world.fake.checks) != 2 {
		t.Fatalf("%d checks after a new path, want 2", len(world.fake.checks))
	}
	world.fake.checks[0].answer(true) // the old path answers late
	if got := world.pane.fileState(world.tab); got != fileUnknown {
		t.Fatalf("state %v after an answer for an older path, want unknown", got)
	}
	world.fake.checks[1].answer(false)
	if got := world.pane.fileState(world.tab); got != fileMissing {
		t.Fatalf("state %v after the answer for the path, want missing", got)
	}
}
