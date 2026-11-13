package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the find bar is pinned at: the response pane in the test window,
// and one so narrow that the field has almost no room.
const (
	findWideW   = 640
	findNarrowW = 220
)

// findBody has two hits for "id" and none for "zzz".
const findBody = `{"id":1,"ids":[1]}`

// The find bar is pinned as text (element kinds, options, rectangles) at
// two widths in its key states: just opened, with hits and an active one,
// and with no hit.
func TestTheFindBarLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name  string
		query string
		steps int
	}{
		{"empty", "", 0},
		{"matches", "id", 1},
		{"no-match", "zzz", 0},
	}
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", findWideW},
		{"narrow", findNarrowW},
	}
	for _, tc := range cases {
		for _, size := range sizes {
			name := "find-" + tc.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a := jsonResponse(t, findBody)
				tab := a.cur()
				a.step(key(ebiten.KeyF, true, false))
				a.typeText(tc.query)
				for range tc.steps {
					a.step(core.Input{Enter: true})
				}
				root := a.findBar(tab)
				Layout(a.ctx, root, size.width, core.RowH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// The counter beside the field follows the typed text: nothing for an
// empty query, the active hit of all hits, or that there is none.
func TestTheFindCounterFollowsTheQuery(t *testing.T) {
	a := jsonResponse(t, findBody)
	a.step(key(ebiten.KeyF, true, false))
	a.typeText("id")
	textCenter(t, a, "1/2")
	a.step(core.Input{Enter: true})
	textCenter(t, a, "2/2")
	a.typeText("zzz")
	textCenter(t, a, "No matches")
}

// The field shows what was typed, which Dump does not show: pinned by
// reading the field the bar edits after typing into it.
func TestTypingInTheFindBarEditsTheTabsQuery(t *testing.T) {
	a := jsonResponse(t, findBody)
	tab := a.cur()
	click(a, findBtnX, toolbarY)
	click(a, findFieldX, findBarY)
	a.typeText("ids")
	if tab.find.Text != "ids" || len(tab.view.Matches()) != 1 {
		t.Fatalf("query %q with %d matches, want %q with 1", tab.find.Text, len(tab.view.Matches()), "ids")
	}
}

// Every button of the bar says what it does on hover.
func TestTheFindBarsButtonsHaveTooltips(t *testing.T) {
	cases := []struct {
		name string
		x    int
		tip  string
	}{
		{"previous", findPrevX, "Previous match"},
		{"next", findNextX, "Next match"},
		{"close", findCloseX, "Close search (Esc)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := jsonResponse(t, findBody)
			click(a, findBtnX, toolbarY)
			if got := tipAt(a, tc.x, findBarY); got != tc.tip {
				t.Fatalf("tooltip %q, want %q", got, tc.tip)
			}
		})
	}
}

// Right-clicking the find field opens the text menu, not the response's.
func TestTheFindFieldHasTheTextMenu(t *testing.T) {
	a := jsonResponse(t, findBody)
	click(a, findBtnX, toolbarY)
	rightClick(a, findFieldX, findBarY)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}
