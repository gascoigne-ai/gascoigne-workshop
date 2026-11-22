package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
				a.step(combo(ebiten.KeyF, true, false, false))
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
	a.step(combo(ebiten.KeyF, true, false, false))
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

func TestCtrlFOpensFindAndTypingSearches(t *testing.T) {
	a := jsonResponse(t, `{"id":1,"name":"x","ids":[1,2]}`)
	tab := a.cur()
	a.step(combo(ebiten.KeyF, true, false, false))
	if !tab.look.FindOpen || !tab.find.Focused {
		t.Fatalf("findOpen %v focused %v", tab.look.FindOpen, tab.find.Focused)
	}
	a.typeText("id")
	if tab.look.Match != 0 || len(tab.view.Find("id")) != 2 {
		t.Fatalf("matchIdx %d, matches %d", tab.look.Match, len(tab.view.Find("id")))
	}
	a.step(core.Input{Enter: true})
	if tab.look.Match != 1 {
		t.Fatalf("Enter goes to the next match, idx %d", tab.look.Match)
	}
	a.step(core.Input{Enter: true}) // wraps
	if tab.look.Match != 0 {
		t.Fatalf("wrap: idx %d", tab.look.Match)
	}
	a.step(core.Input{Enter: true, Shift: true})
	if tab.look.Match != 1 {
		t.Fatalf("Shift+Enter goes back and wraps, idx %d", tab.look.Match)
	}
}

func TestFindButtonsAndClose(t *testing.T) {
	a := jsonResponse(t, `{"id":1,"ids":[1]}`)
	tab := a.cur()
	click(a, findBtnX, toolbarY)
	if !tab.look.FindOpen {
		t.Fatal("the Find button opens the bar")
	}
	click(a, findFieldX, findBarY)
	a.typeText("id")
	click(a, findNextX, findBarY)
	if tab.look.Match != 1 {
		t.Fatalf("next: %d", tab.look.Match)
	}
	click(a, findPrevX, findBarY)
	if tab.look.Match != 0 {
		t.Fatalf("previous: %d", tab.look.Match)
	}
	click(a, findCloseX, findBarY)
	if tab.look.FindOpen {
		t.Fatal("× closes the bar")
	}
}

func TestEscapeClosesFindBeforeAnythingElse(t *testing.T) {
	a := jsonResponse(t, `{"id":1}`)
	tab := a.cur()
	a.step(combo(ebiten.KeyF, true, false, false))
	a.typeText("id")
	a.step(core.Input{Escape: true})
	if tab.look.FindOpen || len(tab.view.Find("")) != 0 {
		t.Fatal("Escape closes the find bar")
	}
	if tab.Resp == nil {
		t.Fatal("and must not touch the response")
	}
}

func TestANewResponseRerunsAnOpenSearch(t *testing.T) {
	a := jsonResponse(t, `{"id":1}`)
	tab := a.cur()
	a.step(combo(ebiten.KeyF, true, false, false))
	a.typeText("id")

	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		io.WriteString(writer, `{"id":1,"idx":2,"idy":3}`)
	}))
	defer srv.Close()
	sendAndWait(t, a, srv.URL)
	if got := len(tab.view.Find("id")); got != 3 {
		t.Fatalf("matches after the new response: %d", got)
	}
	if tab.look.Match != 0 {
		t.Fatalf("matchIdx = %d", tab.look.Match)
	}
}

func TestFindInCompactViewKeepsOnlyMatchingLeavesTheirChildrenAndTheirPath(t *testing.T) {
	a := jsonResponse(t, compactBody)
	tab := a.cur()
	click(a, compactX, toolbarY)
	a.openFind()
	a.typeText("cy")
	text := viewText(tab)
	if !strings.Contains(text, `"login": "cy"`) {
		t.Fatalf("the match must be visible and unfolded:\n%s", text)
	}
	for _, gone := range []string{`"alpha"`, `"beta"`, `"ann"`, `"bob"`, `"total"`, `cyan`} {
		if gone == "cyan" {
			continue // "cyan" also contains the query: it is a match, so it stays
		}
		if strings.Contains(text, gone) {
			t.Fatalf("%s is unrelated to the match and must be filtered out:\n%s", gone, text)
		}
	}
	if !strings.Contains(text, `"items": [`) || !strings.HasPrefix(text, "{") {
		t.Fatalf("the path to the match stays:\n%s", text)
	}
	if len(tab.view.Matches()) == 0 || tab.look.Match != 0 {
		t.Fatalf("matches %d idx %d", len(tab.view.Matches()), tab.look.Match)
	}
	// A matching block keeps all of its children.
	a.find(tab, "owner")
	text = viewText(tab)
	for _, want := range []string{`"login": "ann"`, `"login": "bob"`, `"login": "cy"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("children of a matching block must stay, missing %s:\n%s", want, text)
		}
	}
	// Clearing the search restores the full compact view with its folds.
	a.find(tab, "")
	if !strings.Contains(viewText(tab), `"total": 3`) {
		t.Fatal("an empty query shows everything again")
	}
	a.closeFind(tab)
	if !strings.Contains(viewText(tab), `"total": 3`) {
		t.Fatal("closing the find bar restores the view")
	}
}

func TestFindWorksInEveryResponseView(t *testing.T) {
	for _, mode := range []struct {
		name string
		mode viewer.Mode
	}{{"pretty", viewer.Pretty}, {"raw", viewer.Raw}, {"compact", viewer.Compact}} {
		a := jsonResponse(t, compactBody)
		tab := a.cur()
		a.setView(tab, mode.mode)
		a.openFind()
		a.typeText("login")
		if len(tab.view.Matches()) != 3 || tab.look.Match != 0 {
			t.Fatalf("%s: %d matches, current %d", mode.name, len(tab.view.Matches()), tab.look.Match)
		}
		a.findStep(tab, 1)
		if tab.look.Match != 1 {
			t.Fatalf("%s: stepping goes to the second match, got %d", mode.name, tab.look.Match)
		}
		a.closeFind(tab)
		if len(tab.view.Matches()) != 0 {
			t.Fatalf("%s: closing the bar clears the highlights", mode.name)
		}
	}
}
