package window

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// A tab can point past the panes of its protocol (a socket has fewer
// response panes than HTTP): building the view shows the last pane instead.
func TestThePaneClampsItsActiveTabToItsPanes(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	tab := a.cur()
	a.editor.ShowPane(tab.Tab, 99)
	tab.respTab = 99
	a.view()
	if want := len(a.editor.PaneTabs(tab.Tab)) - 1; a.editor.PaneOf(tab.Tab) != want {
		t.Errorf("paneTab %d, want %d", a.editor.PaneOf(tab.Tab), want)
	}
	if want := len(a.responsePanesFor(tab)) - 1; tab.respTab != want {
		t.Errorf("respTab %d, want %d", tab.respTab, want)
	}
}

// paneIndex is where the pane labeled label stands among labels.
func paneIndex(t *testing.T, labels []string, label string) int {
	t.Helper()
	for i, candidate := range labels {
		if candidate == label {
			return i
		}
	}
	t.Fatalf("no %q pane", label)
	return -1
}

// Clicking a pane's tab shows that pane, on either side.
func TestClickingAPaneTabShowsThatPane(t *testing.T) {
	cases := []struct {
		name   string
		nth    int // "Headers" is a request pane first, then a response pane
		panes  func(a *Component, tab *reqTab) []string
		active func(a *Component, tab *reqTab) int
	}{
		{"request", 0, func(a *Component, tab *reqTab) []string { return tabLabels(a.editor.PaneTabs(tab.Tab)) },
			func(a *Component, tab *reqTab) int { return a.editor.PaneOf(tab.Tab) }},
		{"response", 1, func(a *Component, tab *reqTab) []string {
			var labels []string
			for _, panel := range a.responsePanesFor(tab) {
				labels = append(labels, panel.label)
			}
			return labels
		}, func(_ *Component, tab *reqTab) int { return tab.respTab }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			tab := a.cur()
			respondWith(200, "OK")(a)
			clickText(t, a, "Headers", test.nth)
			if want := paneIndex(t, test.panes(a, tab), "Headers"); test.active(a, tab) != want {
				t.Errorf("active pane %d, want %d", test.active(a, tab), want)
			}
		})
	}
}

// Each pane's tab strip scrolls with its own state on the tab, which the
// dump does not show, and selecting a tab there shows the pane from its
// top. The handler is called straight: after a click, layout clamps a
// short pane's scroll to the top whatever the handler did.
func TestThePaneTabStripsSelectFromTheTopWithTheirOwnState(t *testing.T) {
	cases := []struct {
		name   string
		strip  func(tab *reqTab) *widget.ScrollState
		active func(tab *reqTab) int
		scroll func(tab *reqTab) *widget.ScrollState
	}{
		{"response",
			func(tab *reqTab) *widget.ScrollState { return &tab.respTabsX },
			func(tab *reqTab) int { return tab.respTab },
			func(tab *reqTab) *widget.ScrollState { return &tab.respScroll }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			tab := a.cur()
			want := fmt.Sprintf("state:%p", test.strip(tab))
			var strip Node
			for _, candidate := range findWidgets(a.view(), "*widget.hscrollB") {
				if strings.Contains(fmt.Sprintf("%+v", candidate.Behavior()), want) {
					strip = candidate
				}
			}
			if strip == nil {
				t.Fatalf("no tab strip with %s", want)
			}
			test.scroll(tab).Y = 40
			strip.SelectHandler()(1)
			if active := test.active(tab); active != 1 {
				t.Errorf("active pane %d, want 1", active)
			}
			if y := test.scroll(tab).Y; y != 0 {
				t.Errorf("pane scrolled to %d, want the top", y)
			}
		})
	}
}
