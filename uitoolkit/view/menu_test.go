package view

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestNextSelectableSkipsSeparatorsAndDisabledRowsAndWraps(t *testing.T) {
	items := []MenuItem{{Label: "a"}, {Separator: true}, {Label: "b", Disabled: true}, {Label: "c"}}
	tests := []struct {
		name      string
		from, dir int
		want      int
	}{
		{"down from the first skips to c", 0, 1, 3},
		{"down from the last wraps to a", 3, 1, 0},
		{"up from the first wraps to c", 0, -1, 3},
		{"up from c lands on a", 3, -1, 0},
		{"from nothing down finds the first", -1, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NextSelectable(items, tt.from, tt.dir); got != tt.want {
				t.Fatalf("NextSelectable(%d, %d) = %d, want %d", tt.from, tt.dir, got, tt.want)
			}
		})
	}
	if got := NextSelectable([]MenuItem{{Separator: true}, {Label: "x", Disabled: true}}, -1, 1); got != -1 {
		t.Fatalf("a menu with nothing to pick returned %d, want -1", got)
	}
}

func TestMenuKeysMoveTheHighlightAndEnterPicks(t *testing.T) {
	menu := NewMenu(10, 10, []MenuItem{{Label: "a"}, {Label: "b"}, {Label: "c"}})
	if menu.Hot != 0 {
		t.Fatalf("a new menu highlights %d, want the first row", menu.Hot)
	}
	tests := []struct {
		name    string
		input   core.Input
		wantHot int
		want    int // the picked row, -1 for none
	}{
		{"down", core.Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}}, 1, -1},
		{"up goes back", core.Input{Keys: []ebiten.Key{ebiten.KeyArrowUp}}, 0, -1},
		{"up from the first wraps to the last", core.Input{Keys: []ebiten.Key{ebiten.KeyArrowUp}}, 2, -1},
		{"enter picks the highlighted row", core.Input{Enter: true}, 2, 2},
		{"enter as a key", core.Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, 2, 2},
		{"nothing pressed", core.Input{}, 2, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := menu.Keys(tt.input); got != tt.want || menu.Hot != tt.wantHot {
				t.Fatalf("Keys picked %d with the highlight on %d, want %d and %d", got, menu.Hot, tt.want, tt.wantHot)
			}
		})
	}
}

func TestMenuWidthFitsItsContentWithinLimits(t *testing.T) {
	c := testCtx(t)
	width := func(items ...MenuItem) int { return Menu(c, NewMenu(0, 0, items)).w }
	narrow := width(MenuItem{Label: "ab"})
	wide := width(MenuItem{Label: "ab"}, MenuItem{Label: strings.Repeat("a long label ", 20)})
	if narrow >= wide || narrow < MenuMinW || wide > MenuMaxW {
		t.Fatalf("widths %d and %d, want a short menu narrower than a long one within %d..%d", narrow, wide, MenuMinW, MenuMaxW)
	}
	if width(MenuItem{Label: "Copy"}, MenuItem{Label: "Paste", Shortcut: "Ctrl+V"}) < width(MenuItem{Label: "Paste"}) {
		t.Fatal("a shortcut text takes room")
	}
}

func TestMenuRowsReserveAnIconColumnOnlyWhenSomeRowHasAnIcon(t *testing.T) {
	c := testCtx(t)
	plain := Dump(Menu(c, NewMenu(0, 0, []MenuItem{{Label: "GET /users"}, {Label: "POST /orders"}})))
	if strings.Contains(plain, "w=14") {
		t.Fatalf("a menu with no icons keeps an empty icon column:\n%s", plain)
	}
	iconed := Dump(Menu(c, NewMenu(0, 0, []MenuItem{{Label: "Copy", Icon: "copy"}, {Label: "Other"}})))
	if strings.Count(iconed, "w=14") != 2 {
		t.Fatalf("both rows keep the icon column so the labels line up:\n%s", iconed)
	}
}

func TestMenuItemWithAHintShowsItUnderItsLabel(t *testing.T) {
	c := testCtx(t)
	dump := Dump(Menu(c, NewMenu(0, 0, []MenuItem{{Label: "layout", Hint: "Place the panes.\nExpects: one of side, stacked"}, {Label: "theme"}})))
	if !strings.Contains(dump, "Expects: one of side, stacked") || !strings.Contains(dump, "Place the panes.") {
		t.Fatalf("the hint lines are not shown:\n%s", dump)
	}
}

func TestMenuRowClickPicksOnlyEnabledRows(t *testing.T) {
	c := testCtx(t)
	var picked []int
	menu := Menu(c, NewMenu(5, 5, []MenuItem{{Label: "a"}, {Separator: true}, {Label: "b", Disabled: true}, {Label: "c"}}),
		OnSelect(func(index int) { picked = append(picked, index) }))
	root := Stack(menu)
	Layout(c, root, 400, 300)
	for index, row := range menu.kids {
		click(c, row.rect.X+row.rect.W/2, row.rect.Y+row.rect.H/2)
		Update(c, root)
		_ = index
	}
	if len(picked) != 2 || picked[0] != 0 || picked[1] != 3 {
		t.Fatalf("picked %v, want only the enabled rows 0 and 3", picked)
	}
}

// The open menu lives in the context, so any element can open one and the
// app only mounts PopupMenuLayer once.
func TestPopupMenuOpensPicksAndClosesBeforeRunning(t *testing.T) {
	c := testCtx(t)
	if MenuOpen(c) || PopupMenuLayer(c) != nil {
		t.Fatal("a new context has no open menu")
	}
	var stateWhenRun bool
	OpenMenu(c, 5, 5, []MenuItem{
		{Label: "a", Run: func() { stateWhenRun = MenuOpen(c) }},
		{Label: "b"},
	})
	if !MenuOpen(c) || PopupMenu(c).Hot != 0 {
		t.Fatal("OpenMenu opens it with the first row highlighted")
	}
	layer := PopupMenuLayer(c)
	root := Stack(layer)
	Layout(c, root, 400, 300)
	row := layer.kids[0]
	click(c, row.rect.X+row.rect.W/2, row.rect.Y+row.rect.H/2)
	Update(c, root)
	if MenuOpen(c) || stateWhenRun {
		t.Fatalf("a pick closes the menu first (open after: %v, open while running: %v)", MenuOpen(c), stateWhenRun)
	}
}

func TestPopupMenuKeysMoveAndEnterRuns(t *testing.T) {
	c := testCtx(t)
	ran := ""
	OpenMenu(c, 0, 0, []MenuItem{{Label: "a", Run: func() { ran = "a" }}, {Label: "b", Run: func() { ran = "b" }}})
	PopupMenuKeys(c, core.Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}})
	if PopupMenu(c).Hot != 1 || ran != "" {
		t.Fatalf("Down moves only the highlight: hot %d, ran %q", PopupMenu(c).Hot, ran)
	}
	PopupMenuKeys(c, core.Input{Enter: true})
	if ran != "b" || MenuOpen(c) {
		t.Fatalf("Enter runs the highlighted row and closes: ran %q, open %v", ran, MenuOpen(c))
	}
}

func TestPopupMenuClickOutsideDismisses(t *testing.T) {
	c := testCtx(t)
	OpenMenu(c, 5, 5, []MenuItem{{Label: "a"}})
	root := Stack(PopupMenuLayer(c))
	Layout(c, root, 400, 300)
	click(c, 390, 290)
	Update(c, root)
	if MenuOpen(c) {
		t.Fatal("a click outside closes the menu")
	}
}

// ---- TextMenu -------------------------------------------------------------

func rightClick(c *core.Ctx, el *El) {
	c.In = core.Input{X: el.rect.X + 2, Y: el.rect.Y + 2, RightClick: true}
}

func openTextMenu(t *testing.T, field *TextField, changes *int) (*core.Ctx, *fakeClipboard) {
	t.Helper()
	c := testCtx(t)
	clipboard := &fakeClipboard{}
	c.Clipboard = clipboard
	el := Field(field, W(200), H(28), OnChange(func() { *changes++ }), TextMenu(field))
	root := Row(el)
	Layout(c, root, 400, 100)
	rightClick(c, el)
	Update(c, root)
	if !MenuOpen(c) {
		t.Fatal("a right click on the field opens its text menu")
	}
	return c, clipboard
}

func menuItemNamed(t *testing.T, c *core.Ctx, label string) MenuItem {
	t.Helper()
	for _, item := range PopupMenu(c).Items {
		if item.Label == label {
			return item
		}
	}
	t.Fatalf("no %q row in the text menu", label)
	return MenuItem{}
}

func TestTextMenuDisablesWhatTheFieldCannotDo(t *testing.T) {
	field := NewTextField("")
	changes := 0
	c, _ := openTextMenu(t, &field, &changes)
	for _, label := range []string{"Undo", "Redo", "Cut", "Copy", "Select all"} {
		if !menuItemNamed(t, c, label).Disabled {
			t.Errorf("%s should be disabled for an empty, untouched field", label)
		}
	}
	if menuItemNamed(t, c, "Paste").Disabled {
		t.Error("Paste is always available")
	}
}

func TestTextMenuCutAndPasteEditTheFieldAndReportTheChange(t *testing.T) {
	field := NewTextField("hello world")
	field.SelectAll()
	changes := 0
	c, clipboard := openTextMenu(t, &field, &changes)

	menuItemNamed(t, c, "Cut").Run()
	if clipboard.text != "hello world" || field.Text != "" || changes != 1 {
		t.Fatalf("cut: clipboard %q, field %q, changes %d", clipboard.text, field.Text, changes)
	}
	menuItemNamed(t, c, "Paste").Run()
	if field.Text != "hello world" || changes != 2 {
		t.Fatalf("paste: field %q, changes %d", field.Text, changes)
	}
	menuItemNamed(t, c, "Undo").Run()
	if field.Text != "" || changes != 3 {
		t.Fatalf("undo: field %q, changes %d", field.Text, changes)
	}
}

// Secrets never reach the clipboard: a masked field cannot cut or copy.
func TestTextMenuNeverCopiesAMaskedField(t *testing.T) {
	field := NewTextField("hunter2")
	field.Masked = true
	field.SelectAll()
	changes := 0
	c, clipboard := openTextMenu(t, &field, &changes)
	for _, label := range []string{"Cut", "Copy"} {
		if !menuItemNamed(t, c, label).Disabled {
			t.Errorf("%s of a masked field must be disabled", label)
		}
	}
	if clipboard.text != "" {
		t.Fatalf("clipboard = %q", clipboard.text)
	}
}
