package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/statefile"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func stateApp(t *testing.T, path string) *App {
	t.Helper()
	a, err := NewApp(withAdapters(Options{StatePath: path}))
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.step(core.Input{})
	return a
}

func TestStateSnapshotCapturesTabsFoldersAndSizes(t *testing.T) {
	a := stateApp(t, "")
	a.openRequest(a.col.Folders[0].Requests[0]) // User/<first>
	a.cur().paneTab, a.cur().respTab = 2, 1
	a.sidebarSplit.Size = 310
	a.paneSplit.Size = 0.4
	a.widthsFor(pane{label: "Params", editable: true}).Weights = []float64{1, 2, 3}
	s := a.snapshot()
	if len(s.Tabs) != 2 || s.Active != 1 {
		t.Fatalf("tabs %+v active %d", s.Tabs, s.Active)
	}
	if got := s.Tabs[1]; got.Path != "User/"+a.col.Folders[0].Requests[0].Name || got.Pane != 2 || got.Resp != 1 {
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
	a := stateApp(t, path)
	a.openRequest(a.col.Folders[0].Requests[0])
	a.cur().paneTab = 3
	a.sidebarSplit.Size = 300
	a.expanded[a.col.Folders[2]] = true
	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}
	b := stateApp(t, path)
	if len(b.tabs) != 2 || b.active != 1 || b.cur().paneTab != 3 {
		t.Fatalf("tabs %d active %d pane %d", len(b.tabs), b.active, b.cur().paneTab)
	}
	if b.cur().Req != b.col.Folders[0].Requests[0] || b.selected != b.cur().Req {
		t.Fatal("the restored tab must be the request in the new tree")
	}
	if b.sidebarSplit.Size != 300 || !b.expanded[b.col.Folders[2]] {
		t.Fatal("split and expanded folders restored")
	}
}

func TestStateRestoreDropsMissingRequestsAndBadValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	err := statefile.Save(path, state.State{
		Tabs:     []state.Tab{{Path: "Gone/Nothing"}, {Path: "User/" + collection.Sample().Folders[0].Requests[0].Name, Pane: 99, Resp: -4}},
		Active:   7,
		Expanded: []string{"Nowhere"},
	})
	if err != nil {
		t.Fatal(err)
	}
	a := stateApp(t, path)
	if len(a.tabs) != 1 || a.active != 0 {
		t.Fatalf("tabs %d active %d", len(a.tabs), a.active)
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
	statefile.Save(path, state.State{Tabs: []state.Tab{{Path: "Gone"}}})
	a := stateApp(t, path)
	if len(a.tabs) != 1 {
		t.Fatalf("%d tabs", len(a.tabs))
	}
}

func TestStateCorruptFileStartsFreshWithAStatusNote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := stateApp(t, path)
	if len(a.tabs) != 1 || a.status == "Ready" {
		t.Fatalf("tabs %d status %q", len(a.tabs), a.status)
	}
}

func TestStateIsSavedAfterAQuietSpell(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	a := stateApp(t, path)
	clock := time.Unix(1000, 0)
	a.now = func() time.Time { return clock }
	a.step(core.Input{})
	a.sidebarSplit.Size = 280
	a.step(core.Input{}) // change noticed, not yet written
	if s, _ := statefile.Load(path); s.SidebarW != 0 {
		t.Fatal("written before the debounce")
	}
	clock = clock.Add(2 * time.Second)
	a.step(core.Input{})
	if s, _ := statefile.Load(path); s.SidebarW != 280 {
		t.Fatalf("not written after the debounce: %+v", s)
	}
}

func TestEmptyStatePathWritesNothing(t *testing.T) {
	a := stateApp(t, "")
	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}
}

func TestResponsesOfARequestComeBackAfterARestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	first := stateApp(t, statePath)
	tab := first.cur()
	pointTabAt(tab, jsonServer(t).URL)
	first.send(tab)
	waitForResponse(t, first, tab)

	second := stateApp(t, statePath)
	got := second.cur()
	if got.Req.Name != tab.Req.Name {
		t.Fatalf("the restored tab is %q, expected %q", got.Req.Name, tab.Req.Name)
	}
	if len(got.Earlier) != 1 || got.Earlier[0].Resp.Status != 200 || string(got.Earlier[0].Resp.Body) != `{"ok":true}` {
		t.Fatalf("history after a restart: %+v", got.Earlier)
	}
}

func TestAnAppWithoutAStateFileKeepsNoHistoryOnDisk(t *testing.T) {
	a := stateApp(t, "")
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
	first := stateApp(t, path)
	first.prefs.Size.W, first.prefs.Size.H = 900, 480
	if err := first.SaveState(); err != nil {
		t.Fatal(err)
	}
	second := stateApp(t, path)
	if second.prefs.Size.W != 900 || second.prefs.Size.H != 480 {
		t.Fatalf("the dialog size after a restart: %dx%d", second.prefs.Size.W, second.prefs.Size.H)
	}
}
