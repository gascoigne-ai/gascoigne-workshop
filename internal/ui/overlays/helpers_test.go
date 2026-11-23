package overlays

import (
	"context"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

const (
	winW = 1280
	winH = 800
)

// fakeWindow is a window of the test size that counts what was asked of it.
type fakeWindow struct{ stale, redraws int }

func (*fakeWindow) Size() (int, int) { return winW, winH }
func (fake *fakeWindow) Stale()      { fake.stale++ }
func (fake *fakeWindow) Redraw()     { fake.redraws++ }

// settingsWindow is the window as the settings dialog asks for it.
type settingsWindow struct{ *fakeWindow }

func (settingsWindow) Pointer() (int, int)                  { return 0, 0 }
func (settingsWindow) Toast(string, severity.Level)         {}
func (settingsWindow) SetStatus(string)                     {}
func (settingsWindow) OpenMenu(int, int, []widget.MenuItem) {}

// fakeSettings keeps the settings the dialog edits.
type fakeSettings struct{ current settings.Settings }

func (fake *fakeSettings) Settings() settings.Settings { return fake.current }
func (fake *fakeSettings) Edit(edit func(*settings.Settings)) error {
	edit(&fake.current)
	return nil
}
func (fake *fakeSettings) Apply(current settings.Settings) { fake.current = current }
func (fake *fakeSettings) SetScale(int)                    {}

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

// noDesktop has no file dialog, so the in-app browser is the way to ask.
type noDesktop struct{}

func (noDesktop) Available() bool   { return false }
func (noDesktop) Suggested() string { return "" }
func (noDesktop) Choose(context.Context, string, filedialog.Mode, string) (string, error) {
	return "", nil
}

type homeFolders struct{}

func (homeFolders) Home() (string, error) { return "/home/me", nil }
func (homeFolders) List(string) ([]workspace.FolderEntry, error) {
	return []workspace.FolderEntry{{Name: "a.json"}}, nil
}

type fakeBus struct{}

func (fakeBus) Run(context.Context, bus.Cmd) {}

// world is the overlays with fakes for everything they ask for.
type world struct {
	t      *testing.T
	ctx    *core.Ctx
	over   *Component
	window *fakeWindow
}

func newWorld(t *testing.T) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	window := &fakeWindow{}
	over := New(Deps{Ctx: ctx, Window: window, Clock: fakeClock{}, Desktop: noDesktop{}, Folders: homeFolders{}, Bus: fakeBus{}, Settings: settingsdialog.Deps{Window: settingsWindow{window}, Settings: &fakeSettings{current: settings.Default()}}})
	return &world{t: t, ctx: ctx, over: over, window: window}
}

// frame is one frame as the window runs it: the input is taken in and the
// overlays get their keys.
func (world *world) frame(in core.Input) {
	world.ctx.In = in
	world.over.Keys(world.ctx, in)
}

// dump is the open overlays laid out in a window, as text.
func (world *world) dump(slot func(Views) Node) string {
	root := Stack(slot(world.over.Views(world.ctx)))
	Layout(world.ctx, root, winW, winH)
	return Dump(root)
}

// pickerHost records what the picker asked.
type pickerHost struct{ picked, dismissed int }

func (*pickerHost) Size() (int, int)               { return winW, winH }
func (host *pickerHost) Pick(transfer.Destination) { host.picked++ }
func (host *pickerHost) Dismiss()                  { host.dismissed++ }

// openPicker opens a picker over two collections with a pickable row.
func (world *world) openPicker() *pickerHost {
	src := &request.Collection{Folder: request.Folder{Name: "Source"}}
	node := src.AddRequest(request.Request{Name: "x"})
	dst := &request.Collection{Folder: request.Folder{Name: "Target"}}
	set := collections.Update(collections.Update(collections.Set{}, collections.Open{Collection: src}), collections.Open{Collection: dst})
	chosen := request.Of(node)
	picker := destpicker.New(chosen, transfer.Move, transfer.Destinations(set, chosen, transfer.Move), func(*request.Collection) core.Token { return "collection.1" })
	host := &pickerHost{}
	world.over.OpenPicker(picker, host)
	return host
}

// confirmed counts the runs of a confirm's primary button.
type confirmed struct{ ran int }

func (c *confirmed) run() { c.ran++ }

func noop() {}
