package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestStateSnapshotCapturesTabsFoldersAndSizes(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: ""}))
	a.openRequest(a.col().Folders[0].Requests[0]) // User/<first>
	a.cur().paneTab, a.cur().respTab = 2, 1
	a.sidebarSplit.Size = 310
	a.paneSplit.Size = 0.4
	a.widthsFor(pane{label: "Params", editable: true}).Weights = []float64{1, 2, 3}
	s := a.snapshot()
	if len(s.Tabs) != 2 || s.Active != 1 {
		t.Fatalf("tabs %+v active %d", s.Tabs, s.Active)
	}
	if got := s.Tabs[1]; got.Path != "User/"+a.col().Folders[0].Requests[0].Name || got.Pane != 2 || got.Resp != 1 {
		t.Fatalf("tab %+v", got)
	}
	if s.SidebarW != 310 || s.PaneSplit != 0.4 || len(s.Columns["Params"]) != 3 {
		t.Fatalf("sizes %+v", s)
	}
	want := map[string]bool{"Repository": true}
	for _, p := range s.Expanded {
		delete(want, p)
	}
	if len(want) != 0 {
		t.Fatalf("expanded folders %v", s.Expanded)
	}
	if s.Window.W != winW || s.Window.H != winH {
		t.Fatalf("window %+v", s.Window)
	}
}

func TestStateRestoreRoundTripsThroughTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	a.openRequest(a.col().Folders[0].Requests[0])
	a.cur().paneTab = 3
	a.sidebarSplit.Size = 300
	a.expanded[a.col().Folders[2]] = true
	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}
	b := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	if len(b.list.Tabs) != 2 || b.list.Active != 1 || b.cur().paneTab != 3 {
		t.Fatalf("tabs %d active %d pane %d", len(b.list.Tabs), b.list.Active, b.cur().paneTab)
	}
	if b.cur().Req != b.col().Folders[0].Requests[0] || b.selected != request.Of(b.cur().Req) {
		t.Fatal("the restored tab must be the request in the new tree")
	}
	if b.sidebarSplit.Size != 300 || !b.expanded[b.col().Folders[2]] {
		t.Fatal("split and expanded folders restored")
	}
}

func TestStateRestoreDropsMissingRequestsAndBadValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	err := stateDisk.Save(path, state.State{
		Tabs:     []state.Tab{{Path: "Gone/Nothing"}, {Path: "User/" + collection.GitHubSample().Folders[0].Requests[0].Name, Pane: 99, Resp: -4}},
		Active:   7,
		Expanded: []string{"Nowhere"},
	})
	if err != nil {
		t.Fatal(err)
	}
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	if len(a.list.Tabs) != 1 || a.list.Active != 0 {
		t.Fatalf("tabs %d active %d", len(a.list.Tabs), a.list.Active)
	}
	if p := a.cur().paneTab; p < 0 || p > 4 {
		t.Fatalf("pane tab %d out of range", p)
	}
	if a.cur().respTab < 0 {
		t.Fatal("negative response tab")
	}
}

func TestStateWithNoTabsOpensTheFirstRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	stateDisk.Save(path, state.State{Tabs: []state.Tab{{Path: "Gone"}}})
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	if len(a.list.Tabs) != 1 {
		t.Fatalf("%d tabs", len(a.list.Tabs))
	}
}

func TestStateCorruptFileStartsFreshWithAStatusNote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	if len(a.list.Tabs) != 1 || a.status == "Ready" {
		t.Fatalf("tabs %d status %q", len(a.list.Tabs), a.status)
	}
}

func TestStateIsSavedAfterAQuietSpell(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	clock := time.Unix(1000, 0)
	a.now = func() time.Time { return clock }
	a.step(core.Input{})
	a.sidebarSplit.Size = 280
	a.markStale()        // the test edits the model directly; in the app an edit comes with input, which marks the keys stale
	a.step(core.Input{}) // change noticed, not yet written
	if s, _ := stateDisk.Load(path); s.SidebarW != 0 {
		t.Fatal("written before the debounce")
	}
	clock = clock.Add(2 * time.Second)
	a.step(core.Input{})
	a.waitWrites()
	if s, _ := stateDisk.Load(path); s.SidebarW != 280 {
		t.Fatalf("not written after the debounce: %+v", s)
	}
}

func TestEmptyStatePathWritesNothing(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: ""}))
	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}
}

func TestResponsesOfARequestComeBackAfterARestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	first := testApp(t, memorySettings(), withOptions(Options{StatePath: statePath}))
	tab := first.cur()
	pointTabAt(tab, jsonServer(t).URL)
	first.send(tab)
	waitForResponse(t, first, tab)

	second := testApp(t, memorySettings(), withOptions(Options{StatePath: statePath}))
	got := second.cur()
	if got.Req.Name != tab.Req.Name {
		t.Fatalf("the restored tab is %q, expected %q", got.Req.Name, tab.Req.Name)
	}
	if len(got.Earlier) != 1 || got.Earlier[0].Resp.Status != 200 || string(got.Earlier[0].Resp.Body) != `{"ok":true}` {
		t.Fatalf("history after a restart: %+v", got.Earlier)
	}
}

func TestAnAppWithoutAStateFileKeepsNoHistoryOnDisk(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: ""}))
	tab := a.cur()
	pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	if a.historyDir() != "" {
		t.Fatalf("historyDir = %q, want none without a state path", a.historyDir())
	}
}

func TestTheSettingsDialogSizeIsRememberedAcrossRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	first := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	first.prefs.Size.W, first.prefs.Size.H = 900, 480
	if err := first.SaveState(); err != nil {
		t.Fatal(err)
	}
	second := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	if second.prefs.Size.W != 900 || second.prefs.Size.H != 480 {
		t.Fatalf("the dialog size after a restart: %dx%d", second.prefs.Size.W, second.prefs.Size.H)
	}
}

// Pins that a hand-edited or damaged state.json with absurd sizes, column
// weights that are zero, negative or enormous, and duplicate or
// out-of-range tabs still opens a window that lays out and draws.
func TestHostileStateValuesStillGiveAWorkingWindow(t *testing.T) {
	first := "User/" + collection.GitHubSample().Folders[0].Requests[0].Name
	path := filepath.Join(t.TempDir(), "state.json")
	err := stateDisk.Save(path, state.State{
		SidebarW: 1 << 40, SettingsW: -5, SettingsH: 1 << 40, PaneSplit: 1e300,
		Tabs:    []state.Tab{{Path: first}, {Path: first}, {Path: first, Pane: -1, Resp: 1 << 30}},
		Active:  -3,
		Columns: map[string][]float64{"params": {0, 0, 0}, "headers": {-1, 1e300, -1e300}, "form": {1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: path}))
	for range 5 {
		a.step(core.Input{})
	}
	if len(a.list.Tabs) == 0 || a.list.Active < 0 || a.list.Active >= len(a.list.Tabs) {
		t.Fatalf("tabs %d active %d", len(a.list.Tabs), a.list.Active)
	}
}

func TestStateSnapshotNamesTheRequestPaneOfEachTab(t *testing.T) {
	a := testApp(t, memorySettings(), withOptions(Options{StatePath: ""}))
	a.openRequest(a.col().Folders[0].Requests[0])
	a.cur().paneTab = 1
	s := a.snapshot()
	if got, want := s.Tabs[1].PaneLabel, requestPanesFor(a.cur())[1].label; got != want {
		t.Errorf("PaneLabel = %q, want %q", got, want)
	}
}

// The index of a pane moves when a pane is added before it: Proto and then
// Schema were put after Body, so a gRPC tab saved on Headers came back on
// the pane that now has its number. The label says which one it was.
func TestStateRestoreFindsTheRequestPaneByItsLabel(t *testing.T) {
	tests := []struct {
		name string
		tab  state.Tab
		want string
	}{
		{"label beats a stale index", state.Tab{Pane: 2, PaneLabel: "Headers"}, "Headers"},
		{"an old file has the index only", state.Tab{Pane: 1}, "Body"},
		{"a label the tab no longer has opens the first pane", state.Tab{Pane: 1, PaneLabel: "Gone"}, "Params"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := testApp(t, memorySettings(), withOptions(Options{StatePath: ""}))
			call := a.col().Folders[0].Requests[0]
			call.Protocol = "grpc"
			test.tab.Path = "User/" + call.Name
			a.restore(state.State{Tabs: []state.Tab{test.tab}})
			if got := requestPanesFor(a.cur())[a.cur().paneTab].label; got != test.want {
				t.Errorf("restored pane = %q, want %q", got, test.want)
			}
		})
	}
}
