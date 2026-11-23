package state

import (
	"slices"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// A state file written when one collection was open at a time names no
// collection: its tabs and folders belong to the one open now.
func TestCollectionAt(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		dirs []string
		want int
	}{
		{"a saved collection by its folder", "/b", []string{"/a", "/b"}, 1},
		{"a folder no longer open", "/c", []string{"/a", "/b"}, -1},
		{"none open", "", nil, -1},
		{"the one not saved", "", []string{"/a", ""}, 1},
		{"an older file: the first", "", []string{"/a", "/b"}, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CollectionAt(test.ref, test.dirs); got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestClampKeepsTheWindowOnTheMonitor(t *testing.T) {
	cases := []struct {
		name string
		in   Window
		want Window
	}{
		{"inside", Window{W: 800, H: 600, X: 100, Y: 100}, Window{W: 800, H: 600, X: 100, Y: 100}},
		{"off to the right", Window{W: 800, H: 600, X: 5000, Y: 100}, Window{W: 800, H: 600, X: 1920 - 800, Y: 100}},
		{"off to the left", Window{W: 800, H: 600, X: -900, Y: -50}, Window{W: 800, H: 600, X: 0, Y: 0}},
		{"bigger than the monitor", Window{W: 4000, H: 3000, X: 0, Y: 0}, Window{W: 1920, H: 1080, X: 0, Y: 0}},
		{"tiny", Window{W: 10, H: 10, X: 0, Y: 0}, Window{W: MinWindowW, H: MinWindowH, X: 0, Y: 0}},
		{"unset stays unset", Window{}, Window{}},
	}
	for _, c := range cases {
		if got := c.in.Clamp(1920, 1080); got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
}

func TestTabPaneIn(t *testing.T) {
	labels := []string{"Params", "Headers", "Body"}
	tests := []struct {
		name string
		tab  Tab
		want int
	}{
		{"found by its label", Tab{PaneLabel: "Body", Pane: 0}, 2},
		{"the label wins over the index", Tab{PaneLabel: "Headers", Pane: 2}, 1},
		{"a label no longer there keeps the fallback", Tab{PaneLabel: "Gone", Pane: 1}, 9},
		{"an older file: the index", Tab{Pane: 1}, 1},
		{"an index past the end keeps the fallback", Tab{Pane: 3}, 9},
		{"a negative index keeps the fallback", Tab{Pane: -1}, 9},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.tab.PaneIn(labels, 9); got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestKeyIsEqualForEqualStates(t *testing.T) {
	first := State{Window: Window{W: 800, H: 600}, Tabs: []Tab{{Path: "A"}}, Welcomed: "v1"}
	same := State{Window: Window{W: 800, H: 600}, Tabs: []Tab{{Path: "A"}}, Welcomed: "v1"}
	other := State{Window: Window{W: 800, H: 601}, Tabs: []Tab{{Path: "A"}}, Welcomed: "v1"}
	if Key(first) == "" || Key(first) != Key(same) {
		t.Errorf("equal states have keys %q and %q", Key(first), Key(same))
	}
	if Key(first) == Key(other) {
		t.Errorf("different states share the key %q", Key(first))
	}
}

func TestRememberDropsTabsOutsideTheTreeAndRemapsActive(t *testing.T) {
	tests := []struct {
		name       string
		open       []Tab
		active     int
		wantPaths  []string
		wantActive int
	}{
		{"all in the tree", []Tab{{Path: "A"}, {Path: "B"}}, 1, []string{"A", "B"}, 1},
		{"a tab before the active one is dropped", []Tab{{Path: ""}, {Path: "B"}, {Path: "C"}}, 2, []string{"B", "C"}, 1},
		{"the active tab is outside the tree", []Tab{{Path: "A"}, {Path: ""}}, 1, []string{"A"}, 0},
		{"none in the tree", []Tab{{Path: ""}}, 0, nil, 0},
		{"an active index past the tabs stays", []Tab{{Path: "A"}}, 5, []string{"A"}, 5},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, active := Remember(test.open, test.active)
			var paths []string
			for _, tab := range got {
				paths = append(paths, tab.Path)
			}
			if !slices.Equal(paths, test.wantPaths) || active != test.wantActive {
				t.Errorf("Remember = %v active %d, want %v active %d", paths, active, test.wantPaths, test.wantActive)
			}
		})
	}
}

func TestReopenSkipsMissingRequestsAndKeepsTheActiveOne(t *testing.T) {
	first, third := &request.Request{Name: "first"}, &request.Request{Name: "third"}
	found := map[string]*request.Request{"A": first, "C": third}
	saved := State{Tabs: []Tab{{Path: "A"}, {Path: "B"}, {Path: "C"}}, Active: 2}
	reopened, active := saved.Reopen(func(tab Tab) *request.Request { return found[tab.Path] })
	if len(reopened) != 2 || reopened[0].Req != first || reopened[1].Req != third || reopened[1].Tab.Path != "C" {
		t.Fatalf("reopened %+v", reopened)
	}
	if active != 1 {
		t.Errorf("active = %d, want 1: the third tab is the second one found", active)
	}
	_, active = State{Tabs: saved.Tabs, Active: 1}.Reopen(func(tab Tab) *request.Request { return found[tab.Path] })
	if active != 0 {
		t.Errorf("active = %d, want 0 when the active tab is gone", active)
	}
}

func TestRememberExpandedSplitsSavedFromUnsaved(t *testing.T) {
	var saved State
	saved.RememberExpanded("", []string{"Open"})
	saved.RememberExpanded("/work/api", []string{"A", "A/B"})
	saved.RememberExpanded("/work/none", nil)
	if !slices.Equal(saved.Expanded, []string{"Open"}) {
		t.Errorf("Expanded = %v", saved.Expanded)
	}
	if got := saved.ExpandedIn["/work/api"]; !slices.Equal(got, []string{"A", "A/B"}) || len(saved.ExpandedIn) != 1 {
		t.Errorf("ExpandedIn = %v", saved.ExpandedIn)
	}
}

func TestExpandedByCollection(t *testing.T) {
	saved := State{Expanded: []string{"Open"}, ExpandedIn: map[string][]string{"/work/api": {"A"}, "": {"ignored"}}}
	got := saved.ExpandedByCollection()
	if len(got) != 2 || !slices.Equal(got[""], []string{"Open"}) || !slices.Equal(got["/work/api"], []string{"A"}) {
		t.Errorf("ExpandedByCollection = %v", got)
	}
}

// A hand-edited file may hold any number: sizes are never negative and the
// pane split stays strictly inside the window.
func TestRestoredSizesNeverGoNegativeAndPaneSplitStaysInside(t *testing.T) {
	tests := []struct {
		name string
		in   State
		want Sizes
	}{
		{"in range", State{SidebarW: 300, SettingsW: 900, SettingsH: 480, PaneSplit: 0.4}, Sizes{300, 900, 480, 0.4, true}},
		{"negative", State{SidebarW: -1, SettingsW: -5, SettingsH: -9}, Sizes{}},
		{"split at the edges", State{PaneSplit: 1}, Sizes{}},
		{"split above one", State{PaneSplit: 1e300}, Sizes{}},
		{"split below zero", State{PaneSplit: -0.5}, Sizes{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.in.Sizes(); got != test.want {
				t.Errorf("Sizes = %+v, want %+v", got, test.want)
			}
		})
	}
}
