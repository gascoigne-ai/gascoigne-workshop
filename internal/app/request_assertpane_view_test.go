package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
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

// The request's Tests pane is pinned as text (element kinds, options,
// rects) at two widths, so moving it to a template cannot change it unseen.
func TestTheAssertionsPaneLaysOutAsPinned(t *testing.T) {
	for _, size := range assertSizes {
		name := "assertions-" + size.name
		t.Run(name, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			tab := a.cur()
			assertDumpPinned(t, a, a.assertionsPane(testsPane, tab, &widget.ScrollState{}), name, size.width, settingsPaneH)
		})
	}
}

// What Dump does not show of the assertions area: it edits the tab's own
// tests area.
func TestTheAssertionsAreaEditsTheTabsTests(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	tab := a.cur()
	areas := behaviorStates(a.assertionsPane(testsPane, tab, &widget.ScrollState{}), "*widget.areaB")
	if len(areas) != 1 || !strings.Contains(areas[0], fmt.Sprintf("%p", &tab.tests)) {
		t.Fatalf("areas %q, want one editing %p", areas, &tab.tests)
	}
}

// dumpedText is the quoted text on an element's own Dump line.
var dumpedText = regexp.MustCompile(`text=("(?:[^"\\]|\\.)*")`)

// The help above the assertions used to be one line cut off at the pane's
// edge; in the narrowest pane it now takes several lines that all fit.
func TestTheAssertionsHelpWrapsInsideANarrowPane(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	root := a.assertionsPane(testsPane, a.cur(), &widget.ScrollState{})
	Layout(a.ctx, root, settingsNarrowW, settingsPaneH)
	pane := root.Rect()
	lines := 0
	var walk func(element *El)
	walk = func(element *El) {
		for _, child := range element.Children() {
			walk(child)
		}
		line, _, _ := strings.Cut(Dump(element), "\n")
		match := dumpedText.FindStringSubmatch(line)
		if !strings.HasPrefix(strings.TrimSpace(line), "textB ") || match == nil {
			return
		}
		text, err := strconv.Unquote(match[1])
		if err != nil {
			t.Fatal(err)
		}
		face := a.ctx.F.UI
		if strings.Contains(line, "small=true") {
			face = a.ctx.F.Small
		}
		lines++
		rect := element.Rect()
		if width := a.ctx.Width(face, text); width > rect.W || rect.X < pane.X || rect.X+rect.W > pane.X+pane.W {
			t.Errorf("help line %q is %d px wide in a %d px rect at x=%d, pane %d px", text, width, rect.W, rect.X, pane.W)
		}
	}
	walk(root)
	if lines < 2 {
		t.Fatalf("the help is %d line(s), want several:\n%s", lines, Dump(root))
	}
}

// The wrapped help took five full-size lines and left the editor 158 of
// its 254 px in the narrowest pane; small lines give most of it back.
func TestTheAssertionsHelpLeavesTheEditorMostOfTheNarrowPane(t *testing.T) {
	const minEditorH = 190
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	root := a.assertionsPane(testsPane, a.cur(), &widget.ScrollState{})
	Layout(a.ctx, root, settingsNarrowW, settingsPaneH)
	children := root.Children()
	editor := children[len(children)-1]
	if editor.Rect().H < minEditorH {
		t.Fatalf("the editor is %d px high, want at least %d:\n%s", editor.Rect().H, minEditorH, Dump(root))
	}
}

// pinnedResponse is the response the pinned results are checked against.
var pinnedResponse = &request.Response{Status: 200, StatusText: "OK", Body: []byte(`{"items":[{"name":"a"}]}`)}

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
				a := testApp(t)
				t.Cleanup(a.WaitWrites)
				tab := a.cur()
				tab.testResults = assert.Run(state.tests, pinnedResponse)
				assertDumpPinned(t, a, a.resultsPane(responsePanes[3], tab, nil), name, size.width, settingsPaneH)
			})
		}
	}
}

// What Dump does not show of the results list: it scrolls with the tab's
// response scroll, not the one it is handed.
func TestTheResultsListScrollsWithTheTabsResponseScroll(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
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
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	a.cur().paneTab = testsPaneTabIdx
	a.cur().Resp = pinnedResponse
	a.Step(core.Input{})
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
