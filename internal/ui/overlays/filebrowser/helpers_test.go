package filebrowser

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

const (
	winW = 1280
	winH = 800
)

// fakeDesktop is a file dialog that exists when choose is set; suggest is
// the program it names for native dialogs.
type fakeDesktop struct {
	choose  func(ctx context.Context, title string, mode filedialog.Mode, name string) (string, error)
	suggest string
}

func (fake *fakeDesktop) Available() bool   { return fake.choose != nil }
func (fake *fakeDesktop) Suggested() string { return fake.suggest }
func (fake *fakeDesktop) Choose(ctx context.Context, title string, mode filedialog.Mode, name string) (string, error) {
	return fake.choose(ctx, title, mode, name)
}

// fakeFolders lists the folders it is given and has a home.
type fakeFolders struct {
	home    string
	entries map[string][]workspace.FolderEntry
}

func (fake fakeFolders) Home() (string, error) { return fake.home, nil }
func (fake fakeFolders) List(dir string) ([]workspace.FolderEntry, error) {
	return fake.entries[dir], nil
}

// fakeBus runs a command at once and keeps the message it returns, as a
// frame would take it: drain runs the ones that are funcs.
type fakeBus struct{ waiting []bus.Msg }

func (fake *fakeBus) Run(ctx context.Context, cmd bus.Cmd) {
	if msg := cmd(ctx); msg != nil {
		fake.waiting = append(fake.waiting, msg)
	}
}

func (fake *fakeBus) drain() {
	waiting := fake.waiting
	fake.waiting = nil
	for _, msg := range waiting {
		msg.(func())()
	}
}

type toast struct {
	text  string
	level severity.Level
}

// fakeWindow is a window of the test size that keeps its toasts.
type fakeWindow struct{ toasts []toast }

func (*fakeWindow) Size() (int, int) { return winW, winH }
func (fake *fakeWindow) Toast(text string, level severity.Level) {
	fake.toasts = append(fake.toasts, toast{text, level})
}

type question struct {
	title, body, okLabel string
	run                  func()
}

// fakeDialogs keeps the questions asked; one that was asked and not answered
// is a dialog open over the browser.
type fakeDialogs struct{ asked []question }

func (fake *fakeDialogs) Confirm(title, body, okLabel string, run func()) {
	fake.asked = append(fake.asked, question{title, body, okLabel, run})
}
func (fake *fakeDialogs) Asking() bool { return len(fake.asked) > 0 }

// world is a browser component with fakes for everything it asks for.
type world struct {
	t       *testing.T
	ctx     *core.Ctx
	c       *Component
	desktop *fakeDesktop
	bus     *fakeBus
	window  *fakeWindow
	dialogs *fakeDialogs
}

func newWorld(t *testing.T, folders Folders) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	w := &world{t: t, ctx: ctx, desktop: &fakeDesktop{}, bus: &fakeBus{}, window: &fakeWindow{}, dialogs: &fakeDialogs{}}
	w.c = New(Deps{Desktop: w.desktop, Folders: folders, Bus: w.bus, Window: w.window, Dialogs: w.dialogs})
	return w
}

// step is one frame as the window runs it: the input is taken in, the
// browser gets its keys, and the view is built, laid out and updated.
func (w *world) step(in core.Input) {
	w.ctx.In = in
	w.c.Keys(w.ctx, in)
	root := Stack(w.c.View(w.ctx))
	Layout(w.ctx, root, winW, winH)
	Update(w.ctx, root)
}

// click is a press and its release at a point.
func (w *world) click(x, y int) {
	w.step(core.Input{X: x, Y: y, Click: true, Held: true})
	w.step(core.Input{X: x, Y: y})
}

// layoutDump is the browser as text with every element's rectangle.
func (w *world) layoutDump() string {
	root := Stack(w.c.View(w.ctx))
	Layout(w.ctx, root, winW, winH)
	return Dump(root)
}

var rectPattern = regexp.MustCompile(`rect=(-?\d+),(-?\d+),(\d+),(\d+)`)

// textCenter is the middle of the first element that shows text exactly,
// once a frame has laid the browser out.
func (w *world) textCenter(text string) (x, y int) {
	w.t.Helper()
	w.step(core.Input{X: 5000, Y: 5000})
	needle := fmt.Sprintf("text=%q", text)
	for _, line := range strings.Split(w.layoutDump(), "\n") {
		if !strings.Contains(line, needle) {
			continue
		}
		match := rectPattern.FindStringSubmatch(line)
		if match == nil {
			w.t.Fatalf("no rectangle on %q", line)
		}
		var left, top, width, height int
		fmt.Sscan(match[1], &left)
		fmt.Sscan(match[2], &top)
		fmt.Sscan(match[3], &width)
		fmt.Sscan(match[4], &height)
		return left + width/2, top + height/2
	}
	w.t.Fatalf("no element with text %q", text)
	return 0, 0
}
