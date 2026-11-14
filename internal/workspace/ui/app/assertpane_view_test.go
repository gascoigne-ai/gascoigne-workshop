package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// assertSizes are the widths the two Tests panes are pinned at: the
// request pane of the test window and the narrowest main area.
var assertSizes = []struct {
	name  string
	width int
}{
	{"wide", settingsWideW},
	{"narrow", settingsNarrowW},
}

// pinAssertDump lays root out at width and compares its Dump with the
// file testdata/<name>.dump.
func pinAssertDump(t *testing.T, a *App, root Node, width int, name string) {
	t.Helper()
	Layout(a.ctx, root, width, settingsPaneH)
	want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Dump(root); got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// The request's Tests pane is pinned as text (element kinds, options,
// rects) at two widths, so moving it to a template cannot change it unseen.
func TestTheAssertionsPaneLaysOutAsPinned(t *testing.T) {
	for _, size := range assertSizes {
		name := "assertions-" + size.name
		t.Run(name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			tab := a.cur()
			pinAssertDump(t, a, a.assertionsPane(testsPane, tab, &widget.ScrollState{}), size.width, name)
		})
	}
}

// What Dump does not show of the assertions area: it edits the tab's own
// tests area.
func TestTheAssertionsAreaEditsTheTabsTests(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	areas := areaStates(a.assertionsPane(testsPane, tab, &widget.ScrollState{}))
	if len(areas) != 1 || !strings.Contains(areas[0], fmt.Sprintf("%p", &tab.tests)) {
		t.Fatalf("areas %q, want one editing %p", areas, &tab.tests)
	}
}

// pinnedResponse is the response the pinned results are checked against.
var pinnedResponse = &model.Response{Status: 200, StatusText: "OK", Body: []byte(`{"items":[{"name":"a"}]}`)}

// resultsPaneStates are the looks of the response's Tests pane: no
// assertions, all passed, and passes mixed with failures and a line that
// does not parse.
var resultsPaneStates = []struct {
	name  string
	tests string
}{
	{"none", ""},
	{"passed", "status == 200\njson items.length > 0"},
	{"mixed", "status == 200\nstatus == 404\njson items[0].name == \"b\"\nnot an assertion"},
}

// The response's Tests pane is pinned as text for each state at two
// widths, so moving it to a template cannot change it unseen.
func TestTheResultsPaneLaysOutAsPinned(t *testing.T) {
	for _, state := range resultsPaneStates {
		for _, size := range assertSizes {
			name := "results-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				tab := a.cur()
				tab.testResults = assert.Run(state.tests, pinnedResponse)
				pinAssertDump(t, a, a.resultsPane(responsePanes[3], tab, nil), size.width, name)
			})
		}
	}
}

// What Dump does not show of the results list: it scrolls with the tab's
// response scroll, not the one it is handed.
func TestTheResultsListScrollsWithTheTabsResponseScroll(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	tab.testResults = assert.Run("status == 200", pinnedResponse)
	root := a.resultsPane(responsePanes[3], tab, &widget.ScrollState{})
	if got := fmt.Sprintf("%+v", root.Behavior()); !strings.Contains(got, fmt.Sprintf("%p", &tab.respScroll)) {
		t.Fatalf("scroll %s, want the state %p", got, &tab.respScroll)
	}
}

// The assertions area in the test window's request Tests pane.
const (
	assertAreaX     = 500
	assertAreaY     = 400
	testsPaneTabIdx = 5
)

// openTestsPane shows the request's Tests pane of the open request, with
// a response to check the assertions against.
func openTestsPane(t *testing.T) *App {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	a.cur().paneTab = testsPaneTabIdx
	a.cur().Resp = pinnedResponse
	a.step(core.Input{})
	return a
}

// Typing an assertion makes it the request's tests and checks it at once
// against the response on screen.
func TestTypingAnAssertionChecksItLive(t *testing.T) {
	a := openTestsPane(t)
	click(a, assertAreaX, assertAreaY)
	a.typeText("status == 200")
	tab := a.cur()
	if tab.Req.Tests != "status == 200" {
		t.Fatalf("tests %q, want the typed line", tab.Req.Tests)
	}
	if len(tab.testResults) != 1 || !tab.testResults[0].OK {
		t.Fatalf("results %+v, want one passed", tab.testResults)
	}
}

// Right-clicking the assertions area opens the text menu.
func TestTheAssertionsAreaHasTheTextMenu(t *testing.T) {
	a := openTestsPane(t)
	rightClick(a, assertAreaX, assertAreaY)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}
