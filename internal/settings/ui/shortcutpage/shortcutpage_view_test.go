package shortcutpage

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Window sizes the page is pinned at: wider than the settings dialog
// gives it, and so narrow that the action names are cut.
const (
	wideW, wideH     = 600, 500
	narrowW, narrowH = 220, 400
)

func newCtx(t *testing.T) *w.Ctx {
	t.Helper()
	ctx, err := w.NewCtx(w.Theme{Colors: map[w.Token]color.RGBA{w.Text: w.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func newHost() *fakeHost { return &fakeHost{current: settings.Default()} }

// press is the frame input of a key combination with Ctrl or without.
func press(key ebiten.Key, ctrl bool) w.Input {
	return w.Input{Keys: []ebiten.Key{key}, Ctrl: ctrl}
}

// recordOn records the key combination input for action id, as a click on
// the row's key and a key press would.
func recordOn(id string, input w.Input) func(page *Page, host *fakeHost) {
	return func(page *Page, host *fakeHost) {
		page.Start(id)
		page.RecordKey(host, input)
	}
}

// The laid-out page is pinned as text (testdata/*.dump, captured from the
// page built in Go), one case per state a row can show.
func TestTheShortcutPageLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		prepare       func(page *Page, host *fakeHost)
	}{
		{"default-wide", wideW, wideH, nil},
		{"default-narrow", narrowW, narrowH, nil},
		{"recording-wide", wideW, wideH, func(page *Page, _ *fakeHost) { page.Start("send") }},
		{"conflict-wide", wideW, wideH, recordOn("close-tab", press(ebiten.KeyEnter, true))},
		{"conflict-narrow", narrowW, narrowH, recordOn("close-tab", press(ebiten.KeyEnter, true))},
		{"refused-wide", wideW, wideH, recordOn("send", press(ebiten.KeyC, true))},
		{"plain-key-wide", wideW, wideH, recordOn("close-tab", press(ebiten.KeyJ, false))},
		{"unbound-narrow", narrowW, narrowH, func(page *Page, host *fakeHost) { page.Clear(host, "send") }},
		{"no-match-wide", wideW, wideH, func(page *Page, _ *fakeHost) { page.search.Text = "zzzz" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var page Page
			host := newHost()
			if tc.prepare != nil {
				tc.prepare(&page, host)
			}
			root := page.View(host)
			Layout(newCtx(t), root, tc.width, tc.height)
			want, err := os.ReadFile(filepath.Join("testdata", tc.name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got := Dump(root); got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// findAll is every element under root, in tree order, whose own line of
// Dump has all the parts.
func findAll(root *El, parts ...string) []*El {
	var found []*El
	line, _, _ := strings.Cut(Dump(root), "\n")
	matches := true
	for _, part := range parts {
		matches = matches && strings.Contains(line, part)
	}
	if matches {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findAll(child, parts...)...)
	}
	return found
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

// pageDriver runs the page frame by frame as the application does:
// rebuild the view, lay it out, deliver the input.
type pageDriver struct {
	t    *testing.T
	ctx  *w.Ctx
	page Page
	host *fakeHost
}

func newDriver(t *testing.T) *pageDriver {
	return &pageDriver{t: t, ctx: newCtx(t), host: newHost()}
}

// view is the page as the next frame lays it out.
func (driver *pageDriver) view() *El {
	root := driver.page.View(driver.host)
	Layout(driver.ctx, root, wideW, wideH)
	return root
}

// step runs one frame; input is built from the frame's laid-out view.
func (driver *pageDriver) step(input func(root *El) w.Input) {
	root := driver.view()
	driver.ctx.In = input(root)
	Update(driver.ctx, root)
}

// clickOn clicks the middle of the index-th element matching parts.
func (driver *pageDriver) clickOn(index int, parts ...string) {
	driver.t.Helper()
	driver.step(func(root *El) w.Input {
		found := findAll(root, parts...)
		if len(found) <= index {
			driver.t.Fatalf("%d elements match %q, want index %d:\n%s", len(found), parts, index, Dump(root))
		}
		rect := found[index].Rect()
		return w.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	})
}

// typeText types into the focused field, away from any button.
func (driver *pageDriver) typeText(text string) {
	driver.step(func(*El) w.Input { return w.Input{X: 5000, Y: 5000, Typed: []rune(text)} })
}

// Clicking a row's key starts recording for that row's action, and the
// next combination becomes its key.
func TestClickingARowsKeyRecordsThatRowsAction(t *testing.T) {
	driver := newDriver(t)
	driver.clickOn(0, `text="Ctrl+Enter"`)
	if got := driver.page.Recording(); got != "send" {
		t.Fatalf("recording %q, want send", got)
	}
	if found := findAll(driver.view(), `text="Press keys..."`); len(found) != 1 {
		t.Fatalf("the recording row does not ask for keys:\n%s", Dump(driver.view()))
	}
	driver.page.RecordKey(driver.host, press(ebiten.KeyK, true))
	if got := KeyText(driver.host, "send"); got != "Ctrl+K" {
		t.Fatalf("send is %q, want Ctrl+K", got)
	}
}

// A row's Clear unbinds only that row's action.
func TestARowsClearUnbindsOnlyItsAction(t *testing.T) {
	driver := newDriver(t)
	driver.clickOn(1, `text="Clear"`)
	if KeyText(driver.host, "send") != "Unbound" || KeyText(driver.host, "close-tab") != "Ctrl+W" {
		t.Fatalf("close-tab %q, send %q", KeyText(driver.host, "close-tab"), KeyText(driver.host, "send"))
	}
}

// Reset shows only on a changed row and gives it its default back.
func TestAChangedRowsResetBringsItsDefaultBack(t *testing.T) {
	driver := newDriver(t)
	driver.host.current.Shortcuts = map[string]string{"send": "Ctrl+K"}
	if found := findAll(driver.view(), `text="Reset"`); len(found) != 1 {
		t.Fatalf("%d Reset buttons, want one on the changed row", len(found))
	}
	driver.clickOn(0, `text="Reset"`)
	if got := KeyText(driver.host, "send"); got != "Ctrl+Enter" {
		t.Fatalf("send is %q, want its default back", got)
	}
}

// Replace on a conflict row gives it the key and unbinds the other action.
func TestReplaceOnAConflictRowTakesTheKey(t *testing.T) {
	driver := newDriver(t)
	recordOn("close-tab", press(ebiten.KeyEnter, true))(&driver.page, driver.host)
	driver.clickOn(0, `text="Replace"`)
	if KeyText(driver.host, "close-tab") != "Ctrl+Enter" || KeyText(driver.host, "send") != "Unbound" {
		t.Fatalf("close-tab %q, send %q", KeyText(driver.host, "close-tab"), KeyText(driver.host, "send"))
	}
}

// Typing into the search field narrows the table to the matching rows.
func TestTypingIntoTheSearchNarrowsTheRows(t *testing.T) {
	driver := newDriver(t)
	driver.clickOn(0, "fieldB")
	driver.typeText("enter")
	if driver.page.search.Text != "enter" {
		t.Fatalf("search %q, want what was typed", driver.page.search.Text)
	}
	dump := Dump(driver.view())
	if !strings.Contains(dump, "Send request") || strings.Contains(dump, "Close tab") {
		t.Fatalf("rows after typing:\n%s", dump)
	}
}

// What Dump does not print of the search field: its placeholder, its text
// menu and the state it edits.
func TestTheSearchFieldHasItsPlaceholderAndTextMenu(t *testing.T) {
	var page Page
	root := page.View(newHost())
	field := findWidget(root, "*widget.fieldB")
	if field == nil {
		t.Fatalf("no field:\n%s", Dump(root))
	}
	got := fmt.Sprintf("%+v", field.Behavior())
	for _, want := range []string{"placeholder:Search by action or key", fmt.Sprintf("%p", &page.search)} {
		if !strings.Contains(got, want) {
			t.Errorf("field %s, want %q in it", got, want)
		}
	}
}
