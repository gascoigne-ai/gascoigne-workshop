package ui

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestAMenuWithoutIconsReservesNoSpaceForThem(t *testing.T) {
	a, _ := newTestApp(t)
	a.openMenu(10, 10, []menuItem{{label: "GET /users"}, {label: "POST /orders"}})
	row := view.Dump(a.menuRow(a.ctxMenu, 0, a.ctxMenu.items[0]))
	if strings.Contains(row, "w="+itoa(menuIconW)) {
		t.Fatalf("a menu with no icons keeps an empty icon column:\n%s", row)
	}
}

func TestAMenuWidthFitsItsContentWithinLimits(t *testing.T) {
	a, _ := newTestApp(t)
	a.openMenu(10, 10, []menuItem{{label: "ab"}})
	narrow := a.menuWidth(a.ctxMenu)
	a.openMenu(10, 10, []menuItem{{label: "ab"}, {label: strings.Repeat("a long label ", 20)}})
	wide := a.menuWidth(a.ctxMenu)
	if narrow >= wide || narrow < menuMinW || wide > menuMaxW {
		t.Fatalf("widths %d and %d, want a short menu narrower than a long one within %d..%d", narrow, wide, menuMinW, menuMaxW)
	}
	a.openMenu(10, 10, []menuItem{{label: "Copy"}, {label: "Paste", shortcut: "Ctrl+V"}})
	if a.menuWidth(a.ctxMenu) < a.menuWidth(&contextMenu{items: []menuItem{{label: "Paste"}}}) {
		t.Fatal("a shortcut text takes room")
	}
}

func TestAMenuItemWithAHintShowsItUnderItsLabel(t *testing.T) {
	a, _ := newTestApp(t)
	a.openMenu(10, 10, []menuItem{{label: "layout", hint: "Place the panes.\nExpects: one of side, stacked"}, {label: "theme"}})
	with := view.Dump(a.menuRow(a.ctxMenu, 0, a.ctxMenu.items[0]))
	without := view.Dump(a.menuRow(a.ctxMenu, 1, a.ctxMenu.items[1]))
	if !strings.Contains(with, "Expects: one of side, stacked") {
		t.Fatalf("the hint is not shown:\n%s", with)
	}
	if strings.Contains(without, "Expects") || strings.Count(without, "textB") != 1 {
		t.Fatalf("a row without a hint should be a single line:\n%s", without)
	}
}
