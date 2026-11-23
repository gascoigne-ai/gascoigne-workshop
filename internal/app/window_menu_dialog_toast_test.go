package app

import (
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMenuClosesOnEscapeAndOnOutsideClick(t *testing.T) {
	a := testApp(t)
	click(a, menuBtnX, menuBtnY)
	a.Step(core.Input{Escape: true})
	if a.over.Settings.Open() {
		t.Fatal("Escape must close the menu")
	}

	click(a, menuBtnX, menuBtnY)
	click(a, 100, treeRowY(4)) // outside the menu, on a request row
	if a.over.Settings.Open() {
		t.Fatal("an outside click must close the menu")
	}
	if len(a.tabs.Tabs()) != 1 {
		t.Fatal("the click that closes the menu must not reach the tree below")
	}

	click(a, menuBtnX, menuBtnY)
	click(a, menuBtnX, menuBtnY) // the button again toggles it shut
	if a.over.Settings.Open() {
		t.Fatal("the button must toggle the menu")
	}
}

// A menu opened at (500, 300): rows are 24px under 4px of padding, a
// separator is 9px: Copy 304..328, separator, Paste 337..361, Delete 361..385.
func openTestMenu(t *testing.T) (*App, *ran) {
	t.Helper()
	a := testApp(t)
	r := &ran{}
	paste := r.item("Paste")
	paste.Disabled = true
	a.openMenu(500, 300, []widget.MenuItem{r.item("Copy"), {Separator: true}, paste, r.item("Delete")})
	a.Step(core.Input{})
	if widget.PopupMenu(a.ctx) == nil {
		t.Fatal("test setup: the menu must be open")
	}
	return a, r
}

const (
	menuItemX   = 600
	menuCopyY   = 316
	menuPasteY  = 349
	menuDeleteY = 373
)

func TestClickingAMenuItemRunsItAndClosesTheMenu(t *testing.T) {
	a, r := openTestMenu(t)
	click(a, menuItemX, menuCopyY)
	if len(r.log) != 1 || r.log[0] != "Copy" || widget.PopupMenu(a.ctx) != nil {
		t.Fatalf("ran %v, menu open %v", r.log, widget.PopupMenu(a.ctx) != nil)
	}
}

func TestDisabledItemsDoNothingAndKeepTheMenuOpen(t *testing.T) {
	a, r := openTestMenu(t)
	click(a, menuItemX, menuPasteY)
	if len(r.log) != 0 || widget.PopupMenu(a.ctx) == nil {
		t.Fatalf("ran %v, menu open %v", r.log, widget.PopupMenu(a.ctx) != nil)
	}
}

func TestClickingOutsideClosesTheMenuWithoutReachingWhatIsBelow(t *testing.T) {
	a, r := openTestMenu(t)
	click(a, 100, treeRowY(4)) // a request row in the sidebar
	if widget.PopupMenu(a.ctx) != nil || len(r.log) != 0 {
		t.Fatalf("menu open %v ran %v", widget.PopupMenu(a.ctx) != nil, r.log)
	}
	if len(a.tabs.Tabs()) != 1 {
		t.Fatal("the click that closes a menu must not also open the request under it")
	}
}

func TestMenuKeyboardNavigationSkipsSeparatorsAndDisabledRows(t *testing.T) {
	a, r := openTestMenu(t)
	down := func() { a.Step(combo(ebiten.KeyArrowDown, false, false, false)) }
	up := func() { a.Step(combo(ebiten.KeyArrowUp, false, false, false)) }
	if widget.PopupMenu(a.ctx).Hot != 0 {
		t.Fatalf("the first enabled row starts highlighted, hot = %d", widget.PopupMenu(a.ctx).Hot)
	}
	down()
	if widget.PopupMenu(a.ctx).Hot != 3 {
		t.Fatalf("Down skips the separator and the disabled row: hot = %d", widget.PopupMenu(a.ctx).Hot)
	}
	down()
	if widget.PopupMenu(a.ctx).Hot != 0 {
		t.Fatalf("Down wraps: hot = %d", widget.PopupMenu(a.ctx).Hot)
	}
	up()
	if widget.PopupMenu(a.ctx).Hot != 3 {
		t.Fatalf("Up wraps backwards: hot = %d", widget.PopupMenu(a.ctx).Hot)
	}
	a.Step(combo(ebiten.KeyEnter, false, false, false))
	if len(r.log) != 1 || r.log[0] != "Delete" || widget.PopupMenu(a.ctx) != nil {
		t.Fatalf("Enter runs the highlighted row: ran %v", r.log)
	}
}

func TestEscapeClosesTheMenuFirst(t *testing.T) {
	a, r := openTestMenu(t)
	a.over.Settings.Show() // the settings menu is open too
	a.Step(core.Input{Escape: true})
	if widget.PopupMenu(a.ctx) != nil || !a.over.Settings.Open() || len(r.log) != 0 {
		t.Fatalf("context menu open %v, settings menu open %v: Escape closes the top one only", widget.PopupMenu(a.ctx) != nil, a.over.Settings.Open())
	}
}

func TestMenuNearTheWindowEdgeStaysOnScreen(t *testing.T) {
	a := testApp(t)
	r := &ran{}
	a.openMenu(1270, 790, []widget.MenuItem{r.item("Copy"), {Separator: true}, r.item("Paste"), r.item("Delete")})
	a.Step(core.Input{})
	// the menu is as wide as its content now: moved left so its right edge is
	// at the window's, and up to y=701 (89 high); the Copy row is 705..729
	click(a, 1280-widget.MenuWidth(a.ctx, widget.PopupMenu(a.ctx).Items)/2-4, 717)
	if len(r.log) != 1 || r.log[0] != "Copy" {
		t.Fatalf("ran %v: the menu must be flipped inside the window", r.log)
	}
}

func TestShortcutsAreIgnoredWhileAMenuIsOpen(t *testing.T) {
	a, _ := openTestMenu(t)
	a.Step(combo(ebiten.KeyT, true, false, false))
	if len(a.tabs.Tabs()) != 1 {
		t.Fatal("Ctrl+T must not open a tab behind a menu")
	}
}

// A confirm dialog is 380px wide, centered; with one line of body it is
// 128px high, so its buttons sit at y=434: OK at x=774, Cancel at x=686.
const (
	dialogOKX     = 774
	dialogCancelX = 686
	dialogButtonY = 434
)

func TestConfirmRunsOnEnterAndOnTheButton(t *testing.T) {
	for name, act := range map[string]func(app *App){
		"Enter":  func(app *App) { app.Step(core.Input{Enter: true}) },
		"button": func(app *App) { click(app, dialogOKX, dialogButtonY) },
	} {
		a := testApp(t)
		var ok int
		a.confirm("Delete request?", "This cannot be undone.", "Delete", func() { ok++ })
		a.Step(core.Input{})
		if a.over.Dialog() == nil {
			t.Fatalf("%s: no dialog", name)
		}
		act(a)
		if ok != 1 || a.over.Dialog() != nil {
			t.Fatalf("%s: onOK ran %d times, dialog open %v", name, ok, a.over.Dialog() != nil)
		}
	}
}

func TestConfirmCancelsOnEscapeCancelButtonAndOutsideClick(t *testing.T) {
	for name, act := range map[string]func(app *App){
		"Escape":  func(app *App) { app.Step(core.Input{Escape: true}) },
		"Cancel":  func(app *App) { click(app, dialogCancelX, dialogButtonY) },
		"outside": func(app *App) { click(app, 100, 700) },
	} {
		a := testApp(t)
		var ok int
		a.confirm("Delete request?", "This cannot be undone.", "Delete", func() { ok++ })
		a.Step(core.Input{})
		act(a)
		if ok != 0 || a.over.Dialog() != nil {
			t.Fatalf("%s: onOK ran %d times, dialog open %v", name, ok, a.over.Dialog() != nil)
		}
	}
}

func TestPromptReturnsTheTypedText(t *testing.T) {
	a := testApp(t)
	var got string
	a.prompt("Rename", "Old name", "Rename", func(text string) { got = text })
	a.Step(core.Input{})
	if a.over.Dialog() == nil || !a.over.Dialog().Input.Focused || a.over.Dialog().Input.SelectedText() != "Old name" {
		t.Fatalf("the field starts focused with its text selected: %+v", a.over.Dialog())
	}
	a.typeText("New name")
	a.Step(core.Input{Enter: true})
	if got != "New name" || a.over.Dialog() != nil {
		t.Fatalf("got %q, dialog open %v", got, a.over.Dialog() != nil)
	}
}

func TestDialogTrapsFocusAndBlocksShortcuts(t *testing.T) {
	a := testApp(t)
	a.prompt("Rename", "x", "Rename", func(string) {})
	a.Step(core.Input{})
	a.Step(core.Input{Tab: true})
	if !a.over.Dialog().Input.Focused || a.urlbar.Field(a.cur().Tab).Focused {
		t.Fatal("Tab must stay inside the dialog")
	}
	a.Step(combo(ebiten.KeyT, true, false, false))
	if len(a.tabs.Tabs()) != 1 {
		t.Fatal("global shortcuts are off while a dialog is open")
	}
}

// Toasts stack up from above the status bar at the right: 320x36 each
// with 6px between them; the newest is lowest.
func toastRect(count, index int) (x, y, w, h int) {
	total := count*36 + (count-1)*6
	return 1280 - 12 - 320, 800 - sizes.StatusH - 8 - total + index*42, 320, 36
}

func TestInfoToastsExpireAndErrorsStayUntilClicked(t *testing.T) {
	a := testApp(t)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return clock }
	a.toast("Saved", severity.Info)
	a.toast("Disk full", severity.Error)
	a.Step(core.Input{})
	if len(a.over.Toasts.Stack()) != 2 {
		t.Fatalf("toasts = %d", len(a.over.Toasts.Stack()))
	}

	clock = clock.Add(2 * time.Second)
	a.Step(core.Input{})
	if len(a.over.Toasts.Stack()) != 2 {
		t.Fatal("an info toast stays for a few seconds")
	}
	clock = clock.Add(2 * time.Second)
	a.Step(core.Input{})
	if len(a.over.Toasts.Stack()) != 1 || a.over.Toasts.Stack()[0].Kind != widget.ToastError {
		t.Fatalf("after 4s only the error is left: %+v", a.over.Toasts.Stack())
	}
	clock = clock.Add(10 * time.Minute)
	a.Step(core.Input{})
	if len(a.over.Toasts.Stack()) != 1 {
		t.Fatal("errors persist")
	}
	x, y, w, h := toastRect(1, 0)
	click(a, x+w/2, y+h/2)
	if len(a.over.Toasts.Stack()) != 0 {
		t.Fatal("a click dismisses a toast")
	}
}

func TestAToastMarksTheTreeStaleAndTheFrameDirty(t *testing.T) {
	a := testApp(t)
	a.treeStale, a.dirty = false, false
	a.toast("hello", severity.Info)
	if !a.treeStale || !a.dirty {
		t.Fatalf("stale %v dirty %v", a.treeStale, a.dirty)
	}
}

func TestClickingOneToastDismissesOnlyThatOne(t *testing.T) {
	a := testApp(t)
	a.toast("first", severity.Error)
	a.toast("second", severity.Error)
	a.Step(core.Input{})
	x, y, w, h := toastRect(2, 0)
	click(a, x+w/2, y+h/2)
	if len(a.over.Toasts.Stack()) != 1 || a.over.Toasts.Stack()[0].Text != "second" {
		t.Fatalf("toasts = %+v", a.over.Toasts.Stack())
	}
}

func TestOnlyTheNewestToastsAreKept(t *testing.T) {
	a := testApp(t)
	for i := 0; i < 8; i++ {
		a.toast(strings.Repeat("x", i+1), severity.Error)
	}
	if len(a.over.Toasts.Stack()) != widget.ToastMax || len(a.over.Toasts.Stack()[len(a.over.Toasts.Stack())-1].Text) != 8 {
		t.Fatalf("%d toasts, newest %q", len(a.over.Toasts.Stack()), a.over.Toasts.Stack()[len(a.over.Toasts.Stack())-1].Text)
	}
}

func TestClipboardFailureBecomesAnErrorToast(t *testing.T) {
	a := testApp(t)
	a.ctx.Clipboard = core.ReportClipboard(&fakeClip{err: errors.New("no clipboard tool found")}, a.clipboardNote)
	a.ctx.Clipboard.Write("x")
	if len(a.over.Toasts.Stack()) != 1 || a.over.Toasts.Stack()[0].Kind != widget.ToastError || !strings.Contains(a.over.Toasts.Stack()[0].Text, "no clipboard tool") {
		t.Fatalf("toasts = %+v", a.over.Toasts.Stack())
	}
	a.ctx.Clipboard = core.ReportClipboard(&fakeClip{}, a.clipboardNote)
	a.ctx.Clipboard.Write("hello")
	if len(a.over.Toasts.Stack()) != 1 || a.notices.Status() != "Copied 5 characters" {
		t.Fatalf("a successful copy is a status line, not a toast: toasts %d status %q", len(a.over.Toasts.Stack()), a.notices.Status())
	}
}

func TestTextFieldMenuStatesAndActions(t *testing.T) {
	a := testApp(t)
	clip := &fakeClip{text: "PASTED"}
	a.ctx.Clipboard = clip
	rightClick(a, 1100, urlBarY)
	want := []string{"Undo", "Redo", "", "Cut", "Copy", "Paste", "", "Select all"}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("text menu = %q", got)
	}
	for _, l := range []string{"Undo", "Redo", "Cut", "Copy"} {
		if !itemByLabel(t, a, l).Disabled {
			t.Errorf("%s must be disabled with nothing to undo or selected", l)
		}
	}
	if itemByLabel(t, a, "Paste").Disabled || itemByLabel(t, a, "Select all").Disabled {
		t.Fatal("Paste and Select all are available")
	}
	if itemByLabel(t, a, "Copy").Shortcut != "Ctrl+C" {
		t.Fatalf("shortcut text = %q", itemByLabel(t, a, "Copy").Shortcut)
	}

	pickMenu(t, a, "Select all")
	url := a.urlbar.Field(a.cur().Tab)
	if url.SelectedText() != url.Text {
		t.Fatal("Select all")
	}
	rightClick(a, 1100, urlBarY)
	if url.SelectedText() != url.Text {
		t.Fatal("a right click must not clear the selection")
	}
	if itemByLabel(t, a, "Copy").Disabled || itemByLabel(t, a, "Cut").Disabled {
		t.Fatal("Cut and Copy are available with a selection")
	}
	original := url.Text
	pickMenu(t, a, "Copy")
	if clip.text != original {
		t.Fatalf("clipboard = %q", clip.text)
	}
	rightClick(a, 1100, urlBarY)
	pickMenu(t, a, "Cut")
	if url.Text != "" || a.cur().Req.URL != "" {
		t.Fatalf("cut: field %q request %q (the model must follow)", url.Text, a.cur().Req.URL)
	}
	clip.text = "https://x.test"
	rightClick(a, 1100, urlBarY)
	pickMenu(t, a, "Paste")
	if url.Text != "https://x.test" || a.cur().Req.URL != "https://x.test" {
		t.Fatalf("paste: field %q request %q", url.Text, a.cur().Req.URL)
	}
	rightClick(a, 1100, urlBarY)
	pickMenu(t, a, "Undo")
	if url.Text != "" {
		t.Fatalf("undo: %q", url.Text)
	}
}

func TestSecretFieldsCannotBeCutOrCopiedFromTheMenu(t *testing.T) {
	a := openAuthPane(t)
	click(a, authBearerX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("s3cret")
	a.auth.Field(a.cur().Tab, "token").SelectAll()
	rightClick(a, authFieldX, authRow1Y)
	if !itemByLabel(t, a, "Cut").Disabled || !itemByLabel(t, a, "Copy").Disabled {
		t.Fatal("a masked field must not offer Cut or Copy")
	}
	if itemByLabel(t, a, "Paste").Disabled {
		t.Fatal("pasting a secret in is fine")
	}
}

func TestCellEditFieldGetsTheTextMenuNotTheRowMenu(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y) // editing
	rightClick(a, valueCellX, paramsRow0Y)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}

func TestBodyEditorMenuPastesLinesAndKeepsTheModelInSync(t *testing.T) {
	a := openBodyPane(t)
	clip := &fakeClip{text: "{\n  \"a\": 1\n}"}
	a.ctx.Clipboard = clip
	click(a, jsonChipX, chipsY)
	rightClick(a, editorX, editorY)
	pickMenu(t, a, "Paste")
	if a.cur().Req.Body != "{\n  \"a\": 1\n}" || a.body.Area(a.cur().Tab).LineCount() != 3 {
		t.Fatalf("Body = %q", a.cur().Req.Body)
	}
}

func TestResponseMenu(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	rightClick(a, 1000, 400)
	want := []string{"Copy", "Copy body", "Save body...", "Open in browser", "Copy as cURL", "", "Select all", "Find"}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("response menu = %q", got)
	}
	if !itemByLabel(t, a, "Copy").Disabled {
		t.Fatal("Copy needs a selection")
	}
	pickMenu(t, a, "Copy body")
	if clip.text != `{"ok":true}` {
		t.Fatalf("clipboard = %q", clip.text)
	}
	rightClick(a, 1000, 400)
	pickMenu(t, a, "Select all")
	if a.resp.Code(a.cur().Tab).SelectedText() != a.resp.Code(a.cur().Tab).AllText() {
		t.Fatal("Select all")
	}
	rightClick(a, 1000, 400)
	if itemByLabel(t, a, "Copy").Disabled {
		t.Fatal("Copy works with a selection")
	}
	pickMenu(t, a, "Copy as cURL")
	if !strings.HasPrefix(clip.text, "curl -L ") {
		t.Fatalf("clipboard = %q", clip.text)
	}
	rightClick(a, 1000, 400)
	pickMenu(t, a, "Find")
	if !a.resp.Look(a.cur().Tab).FindOpen {
		t.Fatal("Find opens the find bar")
	}
}

// openResponseRowMenu right-clicks down the response Headers table until
// a row menu opens (the exact row height is the table's business).
func openResponseRowMenu(t *testing.T, app *App) {
	t.Helper()
	app.cur().respTab = 1 // Headers
	app.Step(core.Input{})
	for y := 150; y < 500; y += 4 {
		rightClick(app, 900, y)
		if widget.PopupMenu(app.ctx) != nil {
			return
		}
	}
	t.Fatal("right-clicking the response headers opened no menu")
}

func TestResponseHeaderRowsHaveACopyMenu(t *testing.T) {
	a := jsonResponse(t, `{}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	openResponseRowMenu(t, a)
	labels := menuLabels(a)
	for _, want := range []string{"Copy value", "Copy name", "Copy as Name: Value", "Copy all headers"} {
		found := false
		for _, l := range labels {
			found = found || l == want
		}
		if !found {
			t.Fatalf("menu %v lacks %q", labels, want)
		}
	}
	for _, l := range labels {
		if strings.HasPrefix(l, "Insert") || l == "Delete" {
			t.Fatalf("a read-only table must not offer %q", l)
		}
	}
	h := a.cur().Resp.Headers[0]
	pickMenu(t, a, "Copy as Name: Value")
	if clip.text != h.Name+": "+h.Value {
		t.Fatalf("clipboard %q", clip.text)
	}
}

func TestResponseHeaderMenuCopiesNameValueAndAll(t *testing.T) {
	a := jsonResponse(t, `{}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	rows := a.cur().Resp.Headers
	openResponseRowMenu(t, a)
	pickMenu(t, a, "Copy value")
	if clip.text != rows[0].Value {
		t.Fatalf("value: %q", clip.text)
	}
	openResponseRowMenu(t, a)
	pickMenu(t, a, "Copy name")
	if clip.text != rows[0].Name {
		t.Fatalf("name: %q", clip.text)
	}
	openResponseRowMenu(t, a)
	pickMenu(t, a, "Copy all headers")
	if !strings.Contains(clip.text, rows[0].Name+": "+rows[0].Value+"\n") || strings.Count(clip.text, "\n") < len(rows)-1 {
		t.Fatalf("all: %q", clip.text)
	}
}

func TestEditableTableMenuAlsoCopiesCells(t *testing.T) {
	a := testApp(t)
	a.cur().Req.Params = []request.KV{{Name: "q", Value: "giou", Enabled: true}}
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	a.Step(core.Input{})
	rightClick(a, 400, paramsRow0Y)
	pickMenu(t, a, "Copy value")
	if clip.text != "giou" {
		t.Fatalf("clipboard %q", clip.text)
	}
}

func TestToolbarAndTabButtonsHaveTooltipsWithTheirShortcuts(t *testing.T) {
	a := jsonResponse(t, `{}`)
	for _, tt := range []struct {
		name string
		x, y int
		want string
	}{
		{"send", sendX, urlBarY, "Send the request (Ctrl+Enter)"},
		{"copy", copyBtnX, toolbarY, "Copy the response body"},
		{"curl", curlBtnX, toolbarY, "Copy the request as a cURL command"},
		{"menu", menuBtnX, menuBtnY, "Settings (Ctrl+,)"},
	} {
		if got := tipAt(a, tt.x, tt.y); got != tt.want {
			t.Errorf("%s: tooltip %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDeleteButtonIsATrashCanWithATooltip(t *testing.T) {
	a := testApp(t)
	found := false
	for x := 300; x < 900 && !found; x += 2 {
		found = tipAt(a, x, paramsRow0Y) == "Delete row"
	}
	if !found {
		t.Fatal("no Delete row tooltip along the first params row")
	}
}

func TestTooltipKeepsFramesComingWhileItWaits(t *testing.T) {
	a := testApp(t)
	a.Step(core.Input{X: sendX, Y: urlBarY})
	a.Draw(ebiten.NewImage(winW, winH))
	if !a.tips.Waiting() {
		t.Fatal("hovering a button with a tooltip must be waiting")
	}
	a.dirty = false
	a.Step(core.Input{X: sendX, Y: urlBarY}) // nothing else changed
	if !a.dirty {
		t.Fatal("frames must keep being drawn until the tooltip shows")
	}
}
