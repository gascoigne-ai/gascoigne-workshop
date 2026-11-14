package settingsui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	core "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Windows the dialog is pinned in: the default one, and one small enough
// that the dialog shrinks to fit inside its margins.
const (
	frameWideW, frameWideH   = 1280, 800
	frameSmallW, frameSmallH = 600, 420
)

// windowHost is the fake host in a window of its own size; it counts the
// requests to close the dialog, which the fake ignores.
type windowHost struct {
	*fakeHost
	width, height int
	closed        int
}

func (h *windowHost) Size() (int, int) { return h.width, h.height }

func (h *windowHost) Send(msg prefs.Msg) {
	if _, ok := msg.(prefs.Close); ok {
		h.closed++
	}
	h.fakeHost.Send(msg)
}

// frameCase is one state of the dialog: the window, the page and what is
// done before it is drawn.
type frameCase struct {
	name          string
	width, height int
	page          string
	prepare       func(t *testing.T, dialog *SettingsDialog, host *windowHost)
}

// openRawWithProblems writes settings.json with an unknown key and a
// wrong type, then opens the raw view on it.
func openRawWithProblems(t *testing.T, dialog *SettingsDialog, host *windowHost) {
	t.Helper()
	text := "{\n  \"theme\": 3,\n  \"nonsense\": true\n}\n"
	if err := os.WriteFile(host.path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	dialog.OpenRaw(host)
}

func openRaw(_ *testing.T, dialog *SettingsDialog, host *windowHost) { dialog.OpenRaw(host) }

func turnAutoSaveOff(_ *testing.T, _ *SettingsDialog, host *windowHost) {
	host.current.AutoSave = false
}

func leaveUnsaved(_ *testing.T, _ *SettingsDialog, host *windowHost) { host.unsaved = true }

// The whole dialog, laid out in a window, is pinned as text
// (testdata/frame-*.dump): its size, the title bar in each Save state, the
// list of pages and the page, the form or the raw text.
func TestTheSettingsDialogLaysOutAsPinned(t *testing.T) {
	cases := []frameCase{
		{"colors-wide", frameWideW, frameWideH, "colors", nil},
		{"colors-small", frameSmallW, frameSmallH, "colors", nil},
		{"appearance-wide", frameWideW, frameWideH, "appearance", nil},
		{"unknown-page-wide", frameWideW, frameWideH, "nowhere", nil},
		{"autosave-off-wide", frameWideW, frameWideH, "colors", turnAutoSaveOff},
		{"unsaved-small", frameSmallW, frameSmallH, "colors", leaveUnsaved},
		{"raw-wide", frameWideW, frameWideH, "colors", openRaw},
		{"raw-small", frameSmallW, frameSmallH, "colors", openRaw},
		{"raw-problems-wide", frameWideW, frameWideH, "colors", openRawWithProblems},
		{"raw-problems-small", frameSmallW, frameSmallH, "colors", openRawWithProblems},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var dialog SettingsDialog
			host := &windowHost{fakeHost: newHost(t), width: tc.width, height: tc.height}
			dialog.Families = testFamilies(t)
			dialog.Cat = tc.page
			if tc.prepare != nil {
				tc.prepare(t, &dialog, host)
			}
			ctx, err := core.NewCtx(testTheme())
			if err != nil {
				t.Fatal(err)
			}
			root := Stack(dialog.View(host)) // the dialog floats: a plain root would be the window
			Layout(ctx, root, tc.width, tc.height)
			want, err := os.ReadFile(filepath.Join("testdata", "frame-"+tc.name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got := Dump(root); got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// frameDriver runs the whole dialog frame by frame in the default window,
// as the application does.
type frameDriver struct {
	t      *testing.T
	ctx    *core.Ctx
	dialog SettingsDialog
	host   *windowHost
}

func newFrameDriver(t *testing.T) *frameDriver {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	driver := &frameDriver{t: t, ctx: ctx, host: &windowHost{fakeHost: newHost(t), width: frameWideW, height: frameWideH}}
	driver.dialog.Families = testFamilies(t)
	return driver
}

// root is the dialog as the next frame draws it, laid out.
func (driver *frameDriver) root() *El {
	root := Stack(driver.dialog.View(driver.host))
	Layout(driver.ctx, root, frameWideW, frameWideH)
	return root
}

// step runs one frame; input is built from the frame's laid-out view.
func (driver *frameDriver) step(input func(root *El) core.Input) {
	root := driver.root()
	driver.ctx.In = input(root)
	Update(driver.ctx, root)
}

// clickOn clicks the middle of the first element matching parts.
func (driver *frameDriver) clickOn(parts ...string) {
	driver.t.Helper()
	driver.step(func(root *El) core.Input {
		found := findAll(root, parts...)
		if len(found) == 0 {
			driver.t.Fatalf("nothing matches %q:\n%s", parts, Dump(root))
		}
		rect := found[0].Rect()
		return core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	})
}

// key sends input away from every element, to whatever has the focus.
func (driver *frameDriver) key(input core.Input) {
	input.X, input.Y = 5000, 5000
	driver.step(func(*El) core.Input { return input })
}

func TestTheJSONButtonShowsTheFileAndFormGoesBack(t *testing.T) {
	driver := newFrameDriver(t)
	driver.clickOn(`text="JSON"`)
	if !driver.dialog.Raw || driver.dialog.RawText.Text != "{}\n" {
		t.Fatalf("raw %v with %q, want the raw view of the minimal settings", driver.dialog.Raw, driver.dialog.RawText.Text)
	}
	driver.clickOn(`text="Form"`)
	if driver.dialog.Raw {
		t.Fatal("Form left the raw view open")
	}
}

func TestTheSaveButtonWritesTheSettings(t *testing.T) {
	driver := newFrameDriver(t)
	driver.host.current.AutoSave = false
	driver.clickOn(`text="Save"`)
	if driver.host.saved != 1 {
		t.Fatalf("saved %d times, want once", driver.host.saved)
	}
}

func TestTheLitSaveButtonWritesThePendingChanges(t *testing.T) {
	driver := newFrameDriver(t)
	driver.host.unsaved = true
	driver.clickOn(`text="Save"`)
	if driver.host.saved != 1 || driver.host.unsaved {
		t.Fatalf("saved %d times, unsaved %v; want one write and nothing pending", driver.host.saved, driver.host.unsaved)
	}
}

func TestTheCloseButtonAndAClickOutsideCloseTheDialog(t *testing.T) {
	driver := newFrameDriver(t)
	driver.clickOn("icon=cross")
	if driver.host.closed != 1 {
		t.Fatalf("the close button sent %d closes, want 1", driver.host.closed)
	}
	driver.step(func(*El) core.Input { return core.Input{X: 5, Y: 5, Click: true} })
	if driver.host.closed != 2 {
		t.Fatalf("a click outside sent %d closes in all, want 2", driver.host.closed)
	}
}

func TestTypingASearchFiltersAndScrollsThePageToTheTop(t *testing.T) {
	driver := newFrameDriver(t)
	driver.dialog.Cat = "appearance"
	driver.dialog.scroll.Y = 80
	driver.clickOn("fieldB")
	driver.key(core.Input{Typed: []rune("theme")})
	if driver.dialog.Search.Text != "theme" || driver.dialog.scroll.Y != 0 {
		t.Fatalf("search %q at scroll %d, want theme at 0", driver.dialog.Search.Text, driver.dialog.scroll.Y)
	}
}

func TestTypingGoodJSONInTheRawViewAppliesIt(t *testing.T) {
	driver := newFrameDriver(t)
	driver.clickOn(`text="JSON"`)
	driver.clickOn("areaB")
	edits := []core.Edit{core.EditEnd}
	for range 8 {
		edits = append(edits, core.EditBackspace)
	}
	driver.key(core.Input{Edits: edits})
	driver.key(core.Input{Typed: []rune(`{"theme": "light"}`)})
	if driver.host.current.Theme != "light" || len(driver.dialog.Problems) != 0 {
		t.Fatalf("theme %q with problems %v, want light applied", driver.host.current.Theme, driver.dialog.Problems)
	}
}

// What Dump does not show of the title bar's search field: its state and
// its placeholder.
func TestTheSearchFieldIsBoundToTheSearchText(t *testing.T) {
	driver := newFrameDriver(t)
	field := findWidget(driver.root(), "*widget.fieldB")
	if field == nil {
		t.Fatal("the dialog has no search field")
	}
	settings := fmt.Sprintf("%+v", field.Behavior())
	for _, want := range []string{
		fmt.Sprintf("field:%p", &driver.dialog.Search),
		"placeholder:Search settings",
	} {
		if !strings.Contains(settings, want) {
			t.Errorf("search field %s, want %s", settings, want)
		}
	}
}

// What Dump does not show of the raw view's editor: its state.
func TestTheRawEditorIsBoundToTheRawText(t *testing.T) {
	driver := newFrameDriver(t)
	driver.dialog.OpenRaw(driver.host)
	area := findWidget(driver.root(), "widget.areaB")
	if area == nil {
		t.Fatal("the raw view has no editor")
	}
	if settings, want := fmt.Sprintf("%+v", area.Behavior()), fmt.Sprintf("area:%p", &driver.dialog.RawText); !strings.Contains(settings, want) {
		t.Errorf("raw editor %s, want %s", settings, want)
	}
}

// pickFromTextMenu right-clicks the first element matching parts and picks
// label from the text menu that opens.
func (driver *frameDriver) pickFromTextMenu(label string, parts ...string) {
	driver.t.Helper()
	driver.step(func(root *El) core.Input {
		found := findAll(root, parts...)
		if len(found) == 0 {
			driver.t.Fatalf("nothing matches %q:\n%s", parts, Dump(root))
		}
		rect := found[0].Rect()
		return core.Input{X: rect.X + 4, Y: rect.Y + 4, RightClick: true}
	})
	menu := PopupMenu(driver.ctx)
	if menu == nil {
		driver.t.Fatalf("a right click on %q opened no menu", parts)
	}
	for _, item := range menu.Items {
		if item.Label == label {
			item.Run()
			return
		}
	}
	driver.t.Fatalf("the text menu has no %s", label)
}

// The right-click menus of the search field and of the raw editor edit
// their own text.
func TestTheTextMenusEditTheirOwnText(t *testing.T) {
	driver := newFrameDriver(t)
	driver.dialog.Search = NewTextField("theme")
	driver.pickFromTextMenu("Select all", "fieldB")
	if got := driver.dialog.Search.SelectedText(); got != "theme" {
		t.Errorf("the search menu selected %q, want the search text", got)
	}
	driver.dialog.OpenRaw(driver.host)
	driver.pickFromTextMenu("Select all", "areaB")
	if got := driver.dialog.RawText.SelectedText(); got != "{}\n" {
		t.Errorf("the raw menu selected %q, want the raw text", got)
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
