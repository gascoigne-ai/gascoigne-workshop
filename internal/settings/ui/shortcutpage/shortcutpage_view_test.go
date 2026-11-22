package shortcutpage

import (
	"flag"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Window sizes the page is pinned at: wider than the settings dialog gives
// it, and the least it gets, at the smallest dialog (520 wide, less its
// page list of 170 and a pad of 16 each side).
const (
	wideW, wideH     = 600, 500
	narrowW, narrowH = 318, 400
)

var update = flag.Bool("update", false, "rewrite testdata/*.dump")

func newCtx(t *testing.T) *w.Ctx {
	t.Helper()
	ctx, err := w.NewCtx(w.Theme{Colors: map[w.Token]color.RGBA{w.Text: w.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	ctx.HostOS = "linux" // the same shortcut text on every machine
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
		{"unbound-narrow", narrowW, narrowH, func(page *Page, host *fakeHost) { page.Start("send"); page.Clear(host, "send") }},
		{"selected-narrow", narrowW, narrowH, func(page *Page, host *fakeHost) {
			host.current.Shortcuts = map[string]string{"send": "Ctrl+Shift+Tab"}
			page.Start("send")
			page.RecordKey(host, w.Input{Keys: []ebiten.Key{ebiten.KeyTab}, Ctrl: true, Shift: true})
		}},
		{"no-match-wide", wideW, wideH, func(page *Page, _ *fakeHost) { page.search.Text = "zzzz" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var page Page
			host := newHost()
			if tc.prepare != nil {
				tc.prepare(&page, host)
			}
			root := page.View(newCtx(t), host)
			Layout(newCtx(t), root, tc.width, tc.height)
			path := filepath.Join("testdata", tc.name+".dump")
			if *update {
				if err := os.WriteFile(path, []byte(Dump(root)), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
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
	root := driver.page.View(driver.ctx, driver.host)
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

// Clicking a row selects it and starts recording its key; the next
// combination becomes its key.
func TestClickingARowSelectsItAndRecordsItsKey(t *testing.T) {
	driver := newDriver(t)
	driver.clickOn(0, `text="Send request"`)
	if got := driver.page.Recording(); got != "send" {
		t.Fatalf("recording %q, want send", got)
	}
	if got := driver.page.Selected(); got != "send" {
		t.Fatalf("selected %q, want send", got)
	}
	if found := findAll(driver.view(), `text="Press keys..."`); len(found) != 1 {
		t.Fatalf("the recording row does not ask for keys:\n%s", Dump(driver.view()))
	}
	driver.page.RecordKey(driver.host, press(ebiten.KeyK, true))
	if got := KeyText(driver.host, "send"); got != "Ctrl+K" {
		t.Fatalf("send is %q, want Ctrl+K", got)
	}
}

// The recording prompt is an accent chip, the row's key chips are plain.
func TestTheRecordingRowShowsAnAccentChip(t *testing.T) {
	driver := newDriver(t)
	driver.page.Start("send")
	if found := findAll(driver.view(), "control=primary", "bg=accent"); len(found) != 1 {
		t.Fatalf("%d accent chips, want one:\n%s", len(found), Dump(driver.view()))
	}
}

// Source is User only for an override that parses, the same test as
// Effective: a stored text that does not parse is ignored, so the row shows
// the default and says so. A cleared binding is stored as "" and parses.
func TestSourceIsUserOnlyForAParsableOverride(t *testing.T) {
	action := Action{ID: "send", Name: "Send request", Default: engine.Binding{Key: ebiten.KeyEnter, Ctrl: true, Bound: true}}
	cases := []struct {
		name      string
		shortcuts map[string]string
		want      string
	}{
		{"none", nil, "Default"},
		{"valid", map[string]string{"send": "Ctrl+K"}, "User"},
		{"unparseable", map[string]string{"send": "Ctrl+Nonsense"}, "Default"},
		{"cleared", map[string]string{"send": ""}, "User"},
		{"other action", map[string]string{"close-tab": "Ctrl+K"}, "Default"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Source(settings.Settings{Shortcuts: tc.shortcuts}, action); got != tc.want {
				t.Fatalf("Source = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTheTableShowsItsColumnsAndEachRowsSource(t *testing.T) {
	driver := newDriver(t)
	driver.host.current.Shortcuts = map[string]string{"send": "Ctrl+K"}
	dump := Dump(driver.view())
	for _, want := range []string{`text="ACTION"`, `text="KEYSTROKES"`, `text="SOURCE"`, `text="Default"`, `text="User"`, `text="Close tab"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("no %s in\n%s", want, dump)
		}
	}
	if strings.Count(dump, `text="User"`) != 1 || strings.Count(dump, `text="Default"`) != 1 {
		t.Errorf("one row is User and the other Default:\n%s", dump)
	}
}

// The selection shows only while its action passes the filter.
func TestTheSelectionIsDroppedWhenFilteredOut(t *testing.T) {
	driver := newDriver(t)
	driver.page.Start("send")
	driver.page.search = NewTextField("close")
	dump := Dump(driver.view())
	if strings.Contains(dump, "icon=pencil") || strings.Contains(dump, "tip=Leave the action without a key") {
		t.Fatalf("a filtered-out row still shows the selection:\n%s", dump)
	}
}

func TestOnlyTheSelectedRowShowsAPencilAndItsDetail(t *testing.T) {
	driver := newDriver(t)
	dump := Dump(driver.view())
	if strings.Contains(dump, "icon=pencil") || strings.Contains(dump, "tip=Leave the action without a key") {
		t.Fatalf("nothing is selected, yet:\n%s", dump)
	}
	driver.page.Start("close-tab")
	root := driver.view()
	if got := len(findAll(root, "icon=pencil")); got != 1 {
		t.Fatalf("%d pencils, want one:\n%s", got, Dump(root))
	}
	if got := len(findAll(root, "tip=Leave the action without a key")); got != 1 {
		t.Fatalf("%d Unbind buttons, want one:\n%s", got, Dump(root))
	}
}

// Typing into the filter narrows the table to the matching rows.
func TestTypingIntoTheFilterNarrowsTheRows(t *testing.T) {
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

// What Dump does not print of the filter field: its placeholder, its text
// menu and the state it edits.
func TestTheFilterFieldHasItsPlaceholderAndTextMenu(t *testing.T) {
	var page Page
	root := page.View(newCtx(t), newHost())
	field := findWidget(root, "*widget.fieldB")
	if field == nil {
		t.Fatalf("no field:\n%s", Dump(root))
	}
	got := fmt.Sprintf("%+v", field.Behavior())
	for _, want := range []string{"placeholder:Filter by action or key", fmt.Sprintf("%p", &page.search)} {
		if !strings.Contains(got, want) {
			t.Errorf("field %s, want %q in it", got, want)
		}
	}
}

// An unbound action used to show the "Plain key" warning: its empty
// binding has no Ctrl or Alt, so it counted as a key that types text.
func TestAnUnboundActionRowHasNoWarningNote(t *testing.T) {
	driver := newDriver(t)
	driver.page.Start("send")
	driver.page.recording = ""
	driver.page.Clear(driver.host, "send")
	if found := findAll(driver.view(), `text="Plain key`); len(found) != 0 {
		t.Fatalf("an unbound row shows the plain key warning:\n%s", Dump(driver.view()))
	}
	if !strings.Contains(Dump(driver.view()), `text="Unbound"`) {
		t.Fatalf("an unbound row does not say so:\n%s", Dump(driver.view()))
	}
}

// Unbind, in the selected row's detail, unbinds only that row's action.
func TestUnbindInTheDetailUnbindsOnlyItsAction(t *testing.T) {
	driver := newDriver(t)
	driver.page.Start("send")
	driver.page.recording = ""
	driver.clickOn(0, "tip=Leave the action without a key")
	if KeyText(driver.host, "send") != "Unbound" || KeyText(driver.host, "close-tab") != "Ctrl+W" {
		t.Fatalf("close-tab %q, send %q", KeyText(driver.host, "close-tab"), KeyText(driver.host, "send"))
	}
}

// Backspace while recording unbinds the action, as it did before.
func TestBackspaceWhileRecordingUnbindsTheAction(t *testing.T) {
	driver := newDriver(t)
	driver.page.Start("send")
	driver.page.RecordKey(driver.host, press(ebiten.KeyBackspace, false))
	if got := KeyText(driver.host, "send"); got != "Unbound" {
		t.Fatalf("send is %q, want Unbound", got)
	}
}

// Reset shows only in the detail of a changed selected row and gives it its
// default back.
func TestAChangedRowsResetBringsItsDefaultBack(t *testing.T) {
	driver := newDriver(t)
	driver.host.current.Shortcuts = map[string]string{"send": "Ctrl+K"}
	driver.page.Start("close-tab")
	if found := findAll(driver.view(), "tip=Back to the default key"); len(found) != 0 {
		t.Fatalf("%d Reset buttons on an unchanged selected row", len(found))
	}
	driver.page.Start("send")
	driver.page.recording = ""
	if found := findAll(driver.view(), "tip=Back to the default key"); len(found) != 1 {
		t.Fatalf("%d Reset buttons, want one on the changed row", len(found))
	}
	driver.clickOn(0, "tip=Back to the default key")
	if got := KeyText(driver.host, "send"); got != "Ctrl+Enter" {
		t.Fatalf("send is %q, want its default back", got)
	}
}

// An override that does not parse is ignored, so its row has nothing to
// reset: Reset and the Source column follow one test.
func TestAnUnparsableOverrideShowsNoResetAndSourceDefault(t *testing.T) {
	driver := newDriver(t)
	driver.host.current.Shortcuts = map[string]string{"send": "Ctrl+Nonsense"}
	driver.page.Start("send")
	driver.page.recording = ""
	dump := Dump(driver.view())
	if strings.Contains(dump, "tip=Back to the default key") || strings.Contains(dump, `text="User"`) {
		t.Fatalf("an ignored override is shown as a change:\n%s", dump)
	}
}

// Replace on a conflict gives the row the key and unbinds the other action.
func TestReplaceOnAConflictRowTakesTheKey(t *testing.T) {
	driver := newDriver(t)
	recordOn("close-tab", press(ebiten.KeyEnter, true))(&driver.page, driver.host)
	driver.clickOn(0, `text="Replace"`)
	if KeyText(driver.host, "close-tab") != "Ctrl+Enter" || KeyText(driver.host, "send") != "Unbound" {
		t.Fatalf("close-tab %q, send %q", KeyText(driver.host, "close-tab"), KeyText(driver.host, "send"))
	}
}

// At most one of the conflict, the refusal and the warning shows, in the
// detail of the selected row.
func TestTheDetailShowsOneNoteAtATime(t *testing.T) {
	cases := []struct {
		name  string
		input w.Input
		want  string
	}{
		{"conflict", press(ebiten.KeyEnter, true), "Used by Send request"},
		{"refusal", press(ebiten.KeyC, true), "belongs to text editing"},
		{"warning", press(ebiten.KeyJ, false), "Plain key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newDriver(t)
			recordOn("close-tab", tc.input)(&driver.page, driver.host)
			dump := Dump(driver.view())
			if got := strings.Count(dump, tc.want); got != 1 {
				t.Fatalf("%d of %q, want 1:\n%s", got, tc.want, dump)
			}
			notes := 0
			for _, marker := range []string{"Used by", "belongs to text editing", "Plain key"} {
				if strings.Contains(dump, marker) {
					notes++
				}
			}
			if notes != 1 {
				t.Fatalf("%d notes shown, want one:\n%s", notes, dump)
			}
		})
	}
}

// Arrow keys move the selection through the visible rows and stop at the
// ends; Enter records the selected row; while recording, keys belong to
// the recording.
func TestNavigateMovesTheSelectionAndEnterStartsRecording(t *testing.T) {
	var page Page
	host := newHost()
	down, up, enter := press(ebiten.KeyArrowDown, false), press(ebiten.KeyArrowUp, false), w.Input{Enter: true}
	steps := []struct {
		input    w.Input
		selected string
	}{
		{down, "close-tab"}, // none selected: Down picks the first
		{down, "send"},
		{down, "send"}, // stops at the end
		{up, "close-tab"},
		{up, "close-tab"}, // stops at the start
	}
	for index, step := range steps {
		page.Navigate(step.input, host)
		if got := page.Selected(); got != step.selected {
			t.Fatalf("step %d: selected %q, want %q", index, got, step.selected)
		}
	}
	page.Navigate(enter, host)
	if page.Recording() != "close-tab" {
		t.Fatalf("Enter: recording %q, want close-tab", page.Recording())
	}
	page.Navigate(down, host)
	if page.Selected() != "close-tab" {
		t.Fatalf("Down while recording moved the selection to %q", page.Selected())
	}
}

func TestNavigateWithNothingSelectedDoesNotRecordOnEnter(t *testing.T) {
	var page Page
	page.Navigate(w.Input{Enter: true}, newHost())
	if page.Recording() != "" {
		t.Fatalf("recording %q with no selection", page.Recording())
	}
}

// Navigation goes through the rows the filter leaves.
func TestNavigateSkipsFilteredOutRows(t *testing.T) {
	var page Page
	page.search = NewTextField("send")
	page.Start("close-tab") // selected, but filtered out
	page.recording = ""
	page.Navigate(press(ebiten.KeyArrowDown, false), newHost())
	if page.Selected() != "send" {
		t.Fatalf("selected %q, want the first visible row", page.Selected())
	}
}

// longHost has an action with the longest key text the page shows.
func longHost() *fakeHost {
	host := newHost()
	host.current.Shortcuts = map[string]string{"send": "Ctrl+Shift+Tab"}
	return host
}

// At the narrowest the page gets, the key chips of the longest text and the
// Source text are not cut by their cells.
func TestKeyChipsAndSourceAreNotCutAtTheNarrowestWidth(t *testing.T) {
	for _, width := range []int{narrowW, wideW} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			var page Page
			ctx, host := newCtx(t), longHost()
			root := page.View(ctx, host)
			Layout(ctx, root, width, narrowH)
			for _, key := range []string{"Ctrl", "Shift", "Tab"} {
				chips := findAll(root, "textB", fmt.Sprintf("text=%q", key), "fg=chip-text")
				if len(chips) == 0 {
					t.Fatalf("no chip %q:\n%s", key, Dump(root))
				}
				if got, want := chips[0].Rect().W, ctx.Width(ctx.F.Small, key); got < want {
					t.Errorf("chip %q is %dpx wide, its text needs %d", key, got, want)
				}
			}
			for _, source := range []string{"Default", "User"} {
				for _, found := range findAll(root, "textB", fmt.Sprintf("text=%q", source)) {
					if got, want := found.Rect().W, ctx.Width(ctx.F.Small, source); got < want {
						t.Errorf("%s is %dpx wide, its text needs %d", source, got, want)
					}
				}
			}
			for _, cell := range findAll(root, "clip=true") {
				for _, chip := range findAll(cell, "control=button") {
					if chip.Rect().X+chip.Rect().W > cell.Rect().X+cell.Rect().W {
						t.Errorf("chip at x=%d..%d is cut by its cell ending at %d", chip.Rect().X, chip.Rect().X+chip.Rect().W, cell.Rect().X+cell.Rect().W)
					}
				}
			}
		})
	}
}

// Dragging a divider keeps every column at least MinColumnPx wide, so the
// header captions and the table stay readable.
func TestDraggingADividerKeepsTheMinimumColumnWidth(t *testing.T) {
	driver := newDriver(t)
	driver.ctx.In = w.Input{}
	root := driver.view()
	header := findAll(root, "control=table")[0].Children()[0]
	cells := header.Children()[0].Children()
	boundary := cells[2].Rect().X + cells[2].Rect().W // between Keystrokes and Source
	driver.step(func(*El) w.Input { return w.Input{X: boundary, Y: header.Rect().Y + 5, Click: true, Held: true} })
	driver.step(func(*El) w.Input { return w.Input{X: boundary + 4000, Y: header.Rect().Y + 5, Held: true} })
	source := driver.view()
	last := findAll(source, "control=table")[0].Children()[0].Children()[0].Children()
	if got := last[len(last)-1].Rect().W; got < MinColumnPx {
		t.Fatalf("the Source column is %dpx wide, want at least %d", got, MinColumnPx)
	}
}
