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
