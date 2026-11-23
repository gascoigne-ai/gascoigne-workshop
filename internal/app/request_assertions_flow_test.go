package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// pinnedResponse is the response the pinned results are checked against.
var pinnedResponse = &request.Response{Status: 200, StatusText: "OK", Body: []byte(`{"items":[{"name":"a"}]}`)}

// What Dump does not show of the results list: it scrolls with the tab's
// response scroll, not the one it is handed.
func TestTheResultsListScrollsWithTheTabsResponseScroll(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	tab := a.cur()
	tab.Req.Tests, tab.Resp = "status == 200", pinnedResponse
	a.editor.Assertions.Run(tab.Tab)
	root := a.responsePaneView(responsePanes[3], tab, &widget.ScrollState{})
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
	if results := a.editor.Assertions.Results(tab.Tab); len(results) != 1 || !results[0].OK {
		t.Fatalf("results %+v, want one passed", results)
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

// The window holds back shortcut keys while the tests area or the timeout
// field has the keyboard (the component answers through Typing).
func TestFocusInTheTestsAreaOrTheTimeoutFieldCountsAsTyping(t *testing.T) {
	cases := []struct {
		name string
		open func(t *testing.T) *App
		x, y int
	}{
		{"tests area", openTestsPane, assertAreaX, assertAreaY},
		{"timeout field", openSettingsPane, timeoutFieldX, chipsY},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := tc.open(t)
			t.Cleanup(a.WaitWrites)
			if a.textFocused() {
				t.Fatal("nothing is focused at the start")
			}
			click(a, tc.x, tc.y)
			if !a.textFocused() {
				t.Fatalf("%s has the keyboard but textFocused is false", tc.name)
			}
		})
	}
}
