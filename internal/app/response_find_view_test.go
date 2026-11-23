package app

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	"io"
	"net/http"
	"net/http/httptest"
)

// findBody has two hits for "id" and none for "zzz".
const findBody = `{"id":1,"ids":[1]}`

// The counter beside the field follows the typed text: nothing for an
// empty query, the active hit of all hits, or that there is none.
func TestTheFindCounterFollowsTheQuery(t *testing.T) {
	a := jsonResponse(t, findBody)
	a.Step(combo(ebiten.KeyF, true, false, false))
	a.typeText("id")
	textCenter(t, a, "1/2")
	a.Step(core.Input{Enter: true})
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
	if a.resp.Field(tab.Tab).Text != "ids" || len(a.resp.Code(tab.Tab).Matches()) != 1 {
		t.Fatalf("query %q with %d matches, want %q with 1", a.resp.Field(tab.Tab).Text, len(a.resp.Code(tab.Tab).Matches()), "ids")
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
	a.Step(combo(ebiten.KeyF, true, false, false))
	if !a.resp.Look(tab.Tab).FindOpen || !a.resp.Field(tab.Tab).Focused {
		t.Fatalf("findOpen %v focused %v", a.resp.Look(tab.Tab).FindOpen, a.resp.Field(tab.Tab).Focused)
	}
	a.typeText("id")
	if a.resp.Look(tab.Tab).Match != 0 || len(a.resp.Code(tab.Tab).Find("id")) != 2 {
		t.Fatalf("matchIdx %d, matches %d", a.resp.Look(tab.Tab).Match, len(a.resp.Code(tab.Tab).Find("id")))
	}
	a.Step(core.Input{Enter: true})
	if a.resp.Look(tab.Tab).Match != 1 {
		t.Fatalf("Enter goes to the next match, idx %d", a.resp.Look(tab.Tab).Match)
	}
	a.Step(core.Input{Enter: true}) // wraps
	if a.resp.Look(tab.Tab).Match != 0 {
		t.Fatalf("wrap: idx %d", a.resp.Look(tab.Tab).Match)
	}
	a.Step(core.Input{Enter: true, Shift: true})
	if a.resp.Look(tab.Tab).Match != 1 {
		t.Fatalf("Shift+Enter goes back and wraps, idx %d", a.resp.Look(tab.Tab).Match)
	}
}

func TestFindButtonsAndClose(t *testing.T) {
	a := jsonResponse(t, `{"id":1,"ids":[1]}`)
	tab := a.cur()
	click(a, findBtnX, toolbarY)
	if !a.resp.Look(tab.Tab).FindOpen {
		t.Fatal("the Find button opens the bar")
	}
	click(a, findFieldX, findBarY)
	a.typeText("id")
	click(a, findNextX, findBarY)
	if a.resp.Look(tab.Tab).Match != 1 {
		t.Fatalf("next: %d", a.resp.Look(tab.Tab).Match)
	}
	click(a, findPrevX, findBarY)
	if a.resp.Look(tab.Tab).Match != 0 {
		t.Fatalf("previous: %d", a.resp.Look(tab.Tab).Match)
	}
	click(a, findCloseX, findBarY)
	if a.resp.Look(tab.Tab).FindOpen {
		t.Fatal("× closes the bar")
	}
}

func TestEscapeClosesFindBeforeAnythingElse(t *testing.T) {
	a := jsonResponse(t, `{"id":1}`)
	tab := a.cur()
	a.Step(combo(ebiten.KeyF, true, false, false))
	a.typeText("id")
	a.Step(core.Input{Escape: true})
	if a.resp.Look(tab.Tab).FindOpen || len(a.resp.Code(tab.Tab).Find("")) != 0 {
		t.Fatal("Escape closes the find bar")
	}
	if tab.Resp == nil {
		t.Fatal("and must not touch the response")
	}
}

func TestANewResponseRerunsAnOpenSearch(t *testing.T) {
	a := jsonResponse(t, `{"id":1}`)
	tab := a.cur()
	a.Step(combo(ebiten.KeyF, true, false, false))
	a.typeText("id")

	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		io.WriteString(writer, `{"id":1,"idx":2,"idy":3}`)
	}))
	defer srv.Close()
	sendAndWait(t, a, srv.URL)
	if got := len(a.resp.Code(tab.Tab).Find("id")); got != 3 {
		t.Fatalf("matches after the new response: %d", got)
	}
	if a.resp.Look(tab.Tab).Match != 0 {
		t.Fatalf("matchIdx = %d", a.resp.Look(tab.Tab).Match)
	}
}
