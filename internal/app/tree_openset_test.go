package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/openset"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// savedCollection writes a collection called name to a new folder: one
// request that sends {{host}}, a folder Nested, a variable only it has
// and an environment Dev that sets host.
func savedCollection(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	col := request.Collection{
		Folder: request.Folder{
			Name:     name,
			Requests: []*request.Request{{Name: name + " call", Protocol: "rest", Method: "GET", URL: "http://{{host}}/" + name}},
			Folders:  []*request.Folder{{Name: "Nested", Requests: []*request.Request{{Name: "Inner", Protocol: "rest", Method: "GET"}}}},
		},
		Vars: []request.KV{{Name: "only" + name, Value: name, Enabled: true}},
		Envs: []request.Environment{{Name: "Dev", Vars: []request.KV{{Name: "host", Value: name + ".dev", Enabled: true}}}},
	}
	if err := storeDisk.Save(dir, col); err != nil {
		t.Fatal(err)
	}
	return dir
}

// twoCollectionsApp starts on the collection Alpha and opens Beta beside
// it, as the tree's Open command will.
func twoCollectionsApp(t *testing.T) (a *App, alphaDir, betaDir string) {
	t.Helper()
	return twoCollectionsAppWith(t, Options{})
}

// twoCollectionsAppWith is twoCollectionsApp started with opts.
func twoCollectionsAppWith(t *testing.T, opts Options) (a *App, alphaDir, betaDir string) {
	t.Helper()
	alphaDir, betaDir = savedCollection(t, "Alpha"), savedCollection(t, "Beta")
	a = testApp(t, untouchedSettings(), inWorkspace(alphaDir), withOptions(opts))
	if err := a.addCollection(betaDir); err != nil {
		t.Fatal(err)
	}
	return a, alphaDir, betaDir
}

// entryNamed is the open collection called name.
func entryNamed(t *testing.T, app *App, name string) collections.Entry {
	t.Helper()
	for _, entry := range app.open.Entries {
		if entry.Collection.Name == name {
			return entry
		}
	}
	t.Fatalf("%s is not open", name)
	return collections.Entry{}
}

func TestASecondCollectionOpensBesideTheFirst(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	if len(a.open.Entries) != 2 || a.open.Entries[0].Dir != alphaDir || a.open.Entries[1].Dir != betaDir {
		t.Fatalf("open %+v", a.open.Entries)
	}
	if a.cur().Req.Name != "Beta call" {
		t.Fatalf("the new collection's first request is shown, got %q", a.cur().Req.Name)
	}
	roots := 0
	for _, row := range a.visibleRows() {
		if a.isRoot(row.Dir) {
			roots++
		}
	}
	if roots != 2 {
		t.Fatalf("%d collection roots in the tree, want 2", roots)
	}
	if err := a.addCollection(betaDir); err != nil || len(a.open.Entries) != 2 {
		t.Fatalf("opening an open folder again: %v, %d open", err, len(a.open.Entries))
	}
}

func TestSavingWritesTheCollectionOfTheActiveTabToItsFolder(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	a.openRequest(entryNamed(t, a, "Beta").Collection.Requests[0])
	a.cur().Req.URL = "http://beta.edited"
	a.saveWorkspace()
	a.waitWrites()
	if got, err := storeDisk.Load(betaDir); err != nil || got.Requests[0].URL != "http://beta.edited" {
		t.Fatalf("Beta on disk: %+v %v", got.Requests, err)
	}
	if got, err := storeDisk.Load(alphaDir); err != nil || got.Requests[0].URL != "http://{{host}}/Alpha" {
		t.Fatalf("Alpha on disk: %+v %v", got.Requests, err)
	}
}

func TestEachCollectionIsSavedToItsOwnFolderAfterAPause(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	now := time.Now()
	a.now = func() time.Time { return now }
	a.step(core.Input{})
	entryNamed(t, a, "Alpha").Collection.Requests[0].Name = "Alpha renamed"
	entryNamed(t, a, "Beta").Collection.Folders[0].Name = "Beta folder"
	a.markStale() // the test edits the model directly
	a.step(core.Input{})
	now = now.Add(2 * time.Second)
	a.step(core.Input{})
	a.waitWrites()
	alpha, err := storeDisk.Load(alphaDir)
	if err != nil || alpha.Requests[0].Name != "Alpha renamed" || alpha.Folders[0].Name != "Nested" {
		t.Fatalf("Alpha on disk: %+v %v", alpha, err)
	}
	beta, err := storeDisk.Load(betaDir)
	if err != nil || beta.Folders[0].Name != "Beta folder" || beta.Requests[0].Name != "Beta call" {
		t.Fatalf("Beta on disk: %+v %v", beta, err)
	}
}

func TestTreeEditsChangeTheCollectionThatHoldsTheNode(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	a.newRequestIn(beta.Folders[0])
	if len(beta.Folders[0].Requests) != 2 || len(alpha.Folders[0].Requests) != 1 {
		t.Fatalf("Beta's folder has %d requests, Alpha's %d", len(beta.Folders[0].Requests), len(alpha.Folders[0].Requests))
	}
	a.duplicateNode(request.Of(alpha.Requests[0]))
	if len(alpha.Requests) != 2 || len(beta.Requests) != 1 {
		t.Fatalf("Alpha has %d requests, Beta %d", len(alpha.Requests), len(beta.Requests))
	}
}

// Variables never leak across collections: a request is sent with its own
// collection's variables and environment only.
func TestVariablesAndEnvironmentsStayInTheirCollection(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	a.openRequest(beta.Requests[0])
	a.selectEnv(0) // the chip acts on the collection of the active tab
	if env := entryNamed(t, a, "Beta").Environment(); env == nil || env.Name != "Dev" {
		t.Fatalf("Beta's environment %+v", env)
	}
	if env := entryNamed(t, a, "Alpha").Environment(); env != nil {
		t.Fatalf("Alpha's environment changed to %+v", env)
	}
	if got := request.Expand(beta.Requests[0].URL+" {{onlyBeta}} {{onlyAlpha}}", a.varsFor(beta.Requests[0])); got != "http://Beta.dev/Beta Beta {{onlyAlpha}}" {
		t.Fatalf("Beta resolves to %q", got)
	}
	if got := request.Expand(alpha.Requests[0].URL+" {{onlyBeta}}", a.varsFor(alpha.Requests[0])); got != "http://{{host}}/Alpha {{onlyBeta}}" {
		t.Fatalf("Alpha resolves to %q", got)
	}
}

func TestACapturedValueGoesToTheCollectionOfItsRequest(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	a.setVariable(alpha.Requests[0], "token", "abc") // Beta's tab is the active one
	if !request.Defined("token", alpha.Vars) || request.Defined("token", beta.Vars) {
		t.Fatalf("Alpha %+v, Beta %+v", alpha.Vars, beta.Vars)
	}
}

func TestTheHeaderShowsTheCollectionOfTheActiveTab(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	if title := a.Title(); title != "Beta - "+appName {
		t.Fatalf("title %q", title)
	}
	a.selectEnv(0)
	if a.envName() != "Dev" {
		t.Fatalf("chip %q", a.envName())
	}
	a.updateTabs(tabs.Activate{Index: 0}) // Alpha's tab
	if title, env := a.Title(), a.envName(); title != "Alpha - "+appName || env != "" {
		t.Fatalf("title %q, chip %q", title, env)
	}
}

func TestClosingACollectionClosesItsTabsAndTheLastLeavesAnEmptyTree(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	beta := entryNamed(t, a, "Beta").Collection
	a.closeCollection(beta)
	if len(a.open.Entries) != 1 || a.open.Entries[0].Dir != alphaDir {
		t.Fatalf("open %+v", a.open.Entries)
	}
	for _, tab := range a.list.Tabs {
		if beta.Folder.Contains(request.Of(tab.Req)) {
			t.Fatalf("Beta's tab %q is still open", tab.Req.Name)
		}
	}
	if _, err := os.Stat(betaDir); err != nil {
		t.Fatalf("closing must leave the folder on disk: %v", err)
	}
	a.closeCollection(a.col())
	if len(a.open.Entries) != 0 || len(a.visibleRows()) != 0 || a.Title() != appName {
		t.Fatalf("open %d, rows %d, title %q", len(a.open.Entries), len(a.visibleRows()), a.Title())
	}
	a.step(core.Input{}) // the window still draws with nothing open
	st, err := settingsDisk.Load(a.settingsPath)
	if err != nil || len(st.Collections) != 0 || st.Workspace != "" {
		t.Fatalf("remembered %+v %q %v", st.Collections, st.Workspace, err)
	}
}

func TestDeletingOneOfTwoCollectionsKeepsTheOther(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	a.deleteCollection(entryNamed(t, a, "Beta").Collection)
	if len(a.open.Entries) != 1 || a.open.Entries[0].Dir != alphaDir {
		t.Fatalf("open %+v", a.open.Entries)
	}
	if _, err := os.Stat(filepath.Join(betaDir, "folder.json")); err == nil {
		t.Fatal("Beta's files are still there")
	}
	if a.col().Name != "Alpha" || a.cur().Req.Name == "Beta call" {
		t.Fatalf("shown %q, tab %q", a.col().Name, a.cur().Req.Name)
	}
}

func TestARestartRestoresEveryOpenCollectionWithItsTabsAndEnvironment(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	statePath := filepath.Join(t.TempDir(), "state.json")
	alphaDir, betaDir := savedCollection(t, "Alpha"), savedCollection(t, "Beta")
	start := func() *App {
		t.Helper()
		a, err := NewApp(withAdapters(Options{SettingsPath: settingsPath, StatePath: statePath}))
		if err != nil {
			t.Fatal(err)
		}
		a.w, a.h = winW, winH
		a.step(core.Input{})
		return a
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: settingsPath, StatePath: statePath, WorkspaceDir: alphaDir}))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.addCollection(betaDir); err != nil {
		t.Fatal(err)
	}
	beta := entryNamed(t, a, "Beta").Collection
	a.openRequest(beta.Folders[0].Requests[0])
	a.selectEnv(0)
	a.expanded[entryNamed(t, a, "Alpha").Collection.Folders[0]] = false
	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}

	b := start()
	if len(b.open.Entries) != 2 || b.open.Entries[0].Dir != alphaDir || b.open.Entries[1].Dir != betaDir {
		t.Fatalf("restored %+v", b.open.Entries)
	}
	if env := entryNamed(t, b, "Beta").Environment(); env == nil || env.Name != "Dev" {
		t.Fatalf("Beta's environment %+v", env)
	}
	if env := entryNamed(t, b, "Alpha").Environment(); env != nil {
		t.Fatalf("Alpha's environment %+v", env)
	}
	restoredBeta := entryNamed(t, b, "Beta").Collection
	if b.cur().Req != restoredBeta.Folders[0].Requests[0] {
		t.Fatalf("active tab %q, want Beta's Inner", b.cur().Req.Name)
	}
	stored, err := (statefile.Files{}).Load(statePath)
	if err != nil || len(stored.ExpandedIn[betaDir]) != 1 || len(stored.ExpandedIn[alphaDir]) != 0 || stored.Tabs[len(stored.Tabs)-1].Collection != betaDir {
		t.Fatalf("remembered %+v %v", stored, err)
	}
}

// A user of a version with one collection keeps working: the settings'
// workspace and environment open as a set of one, an old state file's tabs
// and folders land in it, and the next change writes the new form only.
func TestTheSingleSavedCollectionOfAnOlderVersionStillOpens(t *testing.T) {
	dir := savedCollection(t, "Alpha")
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	old := settings.Default()
	old.Workspace, old.Environment = dir, "Dev"
	if err := settingsDisk.Save(settingsPath, old); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := (statefile.Files{}).Save(statePath, state.State{Expanded: []string{"Nested"}, Tabs: []state.Tab{{Path: "Alpha call"}, {Path: "Nested/Inner"}}, Active: 1}); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: settingsPath, StatePath: statePath}))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.open.Entries) != 1 || a.workspace() != dir || a.envName() != "Dev" {
		t.Fatalf("open %+v, environment %q", a.open.Entries, a.envName())
	}
	if len(a.list.Tabs) != 2 || a.cur().Req != a.col().Folders[0].Requests[0] || !a.expanded[a.col().Folders[0]] {
		t.Fatalf("%d tabs, active %q", len(a.list.Tabs), a.cur().Req.Name)
	}
	a.selectEnv(-1)
	a.selectEnv(0)
	st, err := settingsDisk.Load(settingsPath)
	if err != nil || st.Workspace != "" || st.Environment != "" || len(st.Collections) != 1 || st.Collections[0] != (openset.OpenCollection{Folder: dir, Environment: "Dev"}) {
		t.Fatalf("written %+v %v", st, err)
	}
}

func TestTabsAndRootRowsShowTheColorOfTheirCollection(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha"), entryNamed(t, a, "Beta")
	alphaReq, betaReq := alpha.Collection.Folder.Requests[0], beta.Collection.Folder.Requests[0]
	a.openRequest(alphaReq)
	if got, want := shownTab(t, a, alphaReq).Stripe, theme.CollectionToken(0); got != want {
		t.Fatalf("tab of Alpha stripe = %q, want %q", got, want)
	}
	if got, want := shownTab(t, a, betaReq).Stripe, theme.CollectionToken(1); got != want {
		t.Fatalf("tab of Beta stripe = %q, want %q", got, want)
	}
	for _, row := range a.visibleRows() {
		if !a.isRoot(row.Dir) {
			continue
		}
		owner, _ := a.open.Owner(request.Of(row.Dir))
		if got := a.treeItemOf(row).Stripe; got == "" || (owner.Collection == beta.Collection) == (got == theme.CollectionToken(0)) {
			t.Fatalf("root row of %s has stripe %q", owner.Collection.Name, got)
		}
	}
}

// Switching tabs used to leave the tree on the request last clicked, so a
// tab of another collection was not found in it.
func TestSwitchingTabsSelectsAndRevealsTheRequestInTheTree(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha")
	req := alpha.Collection.Folder.Requests[0]
	a.openRequest(req)
	a.selected = request.Node{}
	a.expanded[&alpha.Collection.Folder] = false
	a.updateTabs(tabs.Activate{Index: tabIndex(t, a, entryNamed(t, a, "Beta").Collection.Folder.Requests[0])})
	a.updateTabs(tabs.Activate{Index: tabIndex(t, a, req)})
	if a.selected != request.Of(req) {
		t.Fatalf("selected = %v, want the request of the active tab", a.selected)
	}
	if !a.expanded[&alpha.Collection.Folder] {
		t.Fatal("the collection holding the active request is not expanded")
	}
}
