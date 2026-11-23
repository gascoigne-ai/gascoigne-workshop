package response

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Widths the find bar is pinned at: the response pane in the test window,
// and one so narrow that the field has almost no room.
const (
	findWideW   = 640
	findNarrowW = 220
)

// findBody has two hits for "id" and none for "zzz".
const findBody = `{"id":1,"ids":[1]}`

// withResponse is a world whose open tab shows body.
func withResponse(t *testing.T, body string) *world {
	t.Helper()
	world := newWorld(t)
	world.respond(request.Response{Status: 200, StatusText: "OK", Body: []byte(body)})
	return world
}

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
				world := withResponse(t, findBody)
				world.pane.OpenFind()
				world.pane.Find(world.tab, tc.query)
				for range tc.steps {
					world.pane.findStep(world.tab, 1)
				}
				assertDumpPinned(t, world.ctx, world.pane.findBar(world.ctx, world.tab), name, size.width, core.RowH)
			})
		}
	}
}

func TestOpenFindShowsTheBarFocusesItAndShowsTheResponsePane(t *testing.T) {
	world := withResponse(t, `{"id":1,"name":"x","ids":[1,2]}`)
	world.pane.OpenFind()
	look := world.pane.Look(world.tab)
	if !look.FindOpen || !world.pane.Field(world.tab).Focused {
		t.Fatalf("findOpen %v focused %v", look.FindOpen, world.pane.Field(world.tab).Focused)
	}
	if world.fake.committed != 1 || len(world.fake.shown) != 1 || world.fake.shown[0] != world.tab {
		t.Fatalf("committed %d, shown %v: a cell edit is written and the Response pane is shown first", world.fake.committed, world.fake.shown)
	}
}

func TestFindingStepsThroughTheMatchesAndWraps(t *testing.T) {
	world := withResponse(t, `{"id":1,"name":"x","ids":[1,2]}`)
	world.pane.OpenFind()
	world.pane.Find(world.tab, "id")
	if look := world.pane.Look(world.tab); look.Match != 0 || len(world.pane.Code(world.tab).Find("id")) != 2 {
		t.Fatalf("matchIdx %d, matches %d", look.Match, len(world.pane.Code(world.tab).Find("id")))
	}
	steps := []struct {
		name  string
		shift bool
		want  int
	}{{"Enter goes to the next match", false, 1}, {"Enter wraps", false, 0}, {"Shift+Enter goes back and wraps", true, 1}}
	for _, step := range steps {
		world.ctx.In = core.Input{Enter: true, Shift: step.shift}
		world.pane.findEnter(world.ctx, world.tab)
		if got := world.pane.Look(world.tab).Match; got != step.want {
			t.Fatalf("%s: idx %d, want %d", step.name, got, step.want)
		}
	}
}

func TestCloseFindHidesTheBarAndTheHighlights(t *testing.T) {
	world := withResponse(t, `{"id":1}`)
	world.pane.OpenFind()
	world.pane.Find(world.tab, "id")
	world.pane.CloseFind(world.tab)
	if world.pane.Look(world.tab).FindOpen || len(world.pane.Code(world.tab).Matches()) != 0 || world.pane.Field(world.tab).Focused {
		t.Fatal("closing hides the bar, drops the matches and the focus")
	}
	if world.tab.Resp == nil {
		t.Fatal("and must not touch the response")
	}
}

func TestANewResponseRerunsAnOpenSearch(t *testing.T) {
	world := withResponse(t, `{"id":1}`)
	world.pane.OpenFind()
	world.pane.Find(world.tab, "id")
	world.respond(request.Response{Status: 200, Body: []byte(`{"id":1,"idx":2,"idy":3}`)})
	if got := len(world.pane.Code(world.tab).Find("id")); got != 3 {
		t.Fatalf("matches after the new response: %d", got)
	}
	if got := world.pane.Look(world.tab).Match; got != 0 {
		t.Fatalf("matchIdx = %d", got)
	}
}

func TestFindInCompactViewKeepsOnlyMatchingLeavesTheirChildrenAndTheirPath(t *testing.T) {
	world := withResponse(t, compactBody)
	world.pane.SetMode(world.tab, viewer.Compact)
	world.pane.OpenFind()
	world.pane.Find(world.tab, "cy")
	text := world.viewText()
	if !strings.Contains(text, `"login": "cy"`) {
		t.Fatalf("the match must be visible and unfolded:\n%s", text)
	}
	for _, gone := range []string{`"alpha"`, `"beta"`, `"ann"`, `"bob"`, `"total"`} {
		if strings.Contains(text, gone) {
			t.Fatalf("%s is unrelated to the match and must be filtered out:\n%s", gone, text)
		}
	}
	if !strings.Contains(text, `"items": [`) || !strings.HasPrefix(text, "{") {
		t.Fatalf("the path to the match stays:\n%s", text)
	}
	if len(world.pane.Code(world.tab).Matches()) == 0 || world.pane.Look(world.tab).Match != 0 {
		t.Fatalf("matches %d idx %d", len(world.pane.Code(world.tab).Matches()), world.pane.Look(world.tab).Match)
	}
	// A matching block keeps all of its children.
	world.pane.Find(world.tab, "owner")
	text = world.viewText()
	for _, want := range []string{`"login": "ann"`, `"login": "bob"`, `"login": "cy"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("children of a matching block must stay, missing %s:\n%s", want, text)
		}
	}
	// Clearing the search restores the full compact view with its folds.
	world.pane.Find(world.tab, "")
	if !strings.Contains(world.viewText(), `"total": 3`) {
		t.Fatal("an empty query shows everything again")
	}
	world.pane.CloseFind(world.tab)
	if !strings.Contains(world.viewText(), `"total": 3`) {
		t.Fatal("closing the find bar restores the view")
	}
}

func TestFindWorksInEveryResponseView(t *testing.T) {
	for _, mode := range []struct {
		name string
		mode viewer.Mode
	}{{"pretty", viewer.Pretty}, {"raw", viewer.Raw}, {"compact", viewer.Compact}} {
		world := withResponse(t, compactBody)
		world.pane.SetMode(world.tab, mode.mode)
		world.pane.OpenFind()
		world.pane.Find(world.tab, "login")
		if got, match := len(world.pane.Code(world.tab).Matches()), world.pane.Look(world.tab).Match; got != 3 || match != 0 {
			t.Fatalf("%s: %d matches, current %d", mode.name, got, match)
		}
		world.pane.findStep(world.tab, 1)
		if got := world.pane.Look(world.tab).Match; got != 1 {
			t.Fatalf("%s: stepping goes to the second match, got %d", mode.name, got)
		}
		world.pane.CloseFind(world.tab)
		if len(world.pane.Code(world.tab).Matches()) != 0 {
			t.Fatalf("%s: closing the bar clears the highlights", mode.name)
		}
	}
}

// Cancelling a send clears the lines and nothing else of the find state.
func TestClearViewEmptiesTheLinesOnly(t *testing.T) {
	world := withResponse(t, findBody)
	world.pane.OpenFind()
	world.pane.ClearView(world.tab)
	if len(world.pane.Code(world.tab).Lines) != 0 {
		t.Fatalf("%d lines left", len(world.pane.Code(world.tab).Lines))
	}
	if !world.pane.Look(world.tab).FindOpen {
		t.Fatal("the find bar stays as it was")
	}
}
