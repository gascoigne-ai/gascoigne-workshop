package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// The tree has no unsaved marks of its own: what is not yet saved shows as
// the dot on a tab and in the window title, so these tests check those.

// tabIndex is the place in the strip of the tab showing req.
func tabIndex(t *testing.T, app *App, req *request.Request) int {
	t.Helper()
	for index, tab := range app.list.Tabs {
		if tab.Req == req {
			return index
		}
	}
	t.Fatalf("no tab shows %q", req.Name)
	return -1
}

// shownTab is what the strip shows of the tab of req: its name and mark.
func shownTab(t *testing.T, app *App, req *request.Request) requestTabItem {
	t.Helper()
	index := tabIndex(t, app, req)
	return app.requestTab(index, app.openTabs()[index])
}

// activate makes the tab of req the active one.
func activate(t *testing.T, app *App, req *request.Request) {
	t.Helper()
	app.updateTabs(tabs.Activate{Index: tabIndex(t, app, req)})
}

// titleMarked reports whether the window title carries the unsaved dot.
func titleMarked(app *App) bool { return strings.HasPrefix(app.Title(), dirtyMark+" ") }

// autosaveAfterPause lets the autosave notice what changed and then the
// pause pass, so every collection with edits is written, whichever is
// focused, and waits for the writes.
func autosaveAfterPause(app *App) {
	now := time.Now()
	app.now = func() time.Time { return now }
	app.markStale() // the tests edit the model directly, with no input to mark the keys stale
	app.Step(core.Input{})
	now = now.Add(2 * saveDebounce)
	app.Step(core.Input{})
	app.WaitWrites()
}

// exists reports whether the file or folder at path is there.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestTheUnsavedMarksFollowTheCollectionOfEachTab(t *testing.T) {
	a, alphaDir, _ := twoCollectionsApp(t)
	now := time.Now()
	a.now = func() time.Time { return now } // no pause passes, so nothing saves by itself
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	alphaCall, betaCall := alpha.Requests[0], beta.Requests[0]

	alphaCall.URL = "http://alpha.edited"
	a.markStale() // the test edits the model directly
	if !shownTab(t, a, alphaCall).Dirty || shownTab(t, a, betaCall).Dirty {
		t.Fatalf("after editing Alpha: Alpha's tab %v, Beta's %v", shownTab(t, a, alphaCall).Dirty, shownTab(t, a, betaCall).Dirty)
	}
	activate(t, a, betaCall)
	if titleMarked(a) {
		t.Fatalf("Beta has no edits, title %q", a.Title())
	}
	activate(t, a, alphaCall)
	if !titleMarked(a) {
		t.Fatalf("Alpha has an edit, title %q", a.Title())
	}

	betaCall.URL = "http://beta.edited"
	a.markStale()
	a.saveCollection(beta)
	a.WaitWrites()
	if !shownTab(t, a, alphaCall).Dirty || shownTab(t, a, betaCall).Dirty {
		t.Fatalf("after saving Beta: Alpha's tab %v, Beta's %v", shownTab(t, a, alphaCall).Dirty, shownTab(t, a, betaCall).Dirty)
	}
	if !titleMarked(a) {
		t.Fatalf("Alpha is still not saved, title %q", a.Title())
	}
	activate(t, a, betaCall)
	if titleMarked(a) {
		t.Fatalf("Beta is saved, title %q", a.Title())
	}
	if got, err := storeDisk.Load(alphaDir); err != nil || got.Requests[0].URL != "http://{{host}}/Alpha" {
		t.Fatalf("saving Beta wrote Alpha: %+v %v", got.Requests, err)
	}
}

// Both collections hold Nested/Inner: a rename in Beta must find Beta's by
// pointer, never Alpha's by its path.
func TestRenamingARequestInOneCollectionLeavesTheSamePathInTheOther(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	alphaInner, betaInner := alpha.Folders[0].Requests[0], beta.Folders[0].Requests[0]
	a.openRequest(betaInner)
	a.openRequest(alphaInner) // the other collection's tab is the active one
	if len(a.list.Tabs) != 4 {
		t.Fatalf("%d tabs, want both Inners beside the two calls", len(a.list.Tabs))
	}

	a.renameNode(request.Of(betaInner))
	a.typeText("Core")
	a.Step(core.Input{Enter: true})
	if betaInner.Name != "Core" || alphaInner.Name != "Inner" {
		t.Fatalf("Beta's %q, Alpha's %q", betaInner.Name, alphaInner.Name)
	}
	if got := shownTab(t, a, betaInner).Name; got != "Core" {
		t.Fatalf("Beta's tab %q", got)
	}
	if shown := shownTab(t, a, alphaInner); shown.Name != "Inner" || shown.Dirty {
		t.Fatalf("Alpha's tab %+v", shown)
	}
	if a.cur().Req != alphaInner {
		t.Fatalf("the active tab moved to %q", a.cur().Req.Name)
	}

	autosaveAfterPause(a)
	if !exists(filepath.Join(betaDir, "nested", "core.req.json")) || exists(filepath.Join(betaDir, "nested", "inner.req.json")) {
		t.Fatal("Beta's request file was not renamed")
	}
	if !exists(filepath.Join(alphaDir, "nested", "inner.req.json")) || exists(filepath.Join(alphaDir, "nested", "core.req.json")) {
		t.Fatal("Alpha's request file changed")
	}
	if shownTab(t, a, betaInner).Dirty {
		t.Fatal("Beta's tab is still marked after the autosave")
	}
}

func TestRenamingAFolderInOneCollectionLeavesTheSamePathInTheOther(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	alphaInner, betaInner := alpha.Folders[0].Requests[0], beta.Folders[0].Requests[0]
	a.openRequest(alphaInner)
	a.openRequest(betaInner)

	if err := a.applyTree(tree.Rename{Node: request.Of(beta.Folders[0]), Name: "Shared"}).Err; err != nil {
		t.Fatal(err)
	}
	if beta.Folders[0].Name != "Shared" || alpha.Folders[0].Name != "Nested" {
		t.Fatalf("Beta's folder %q, Alpha's %q", beta.Folders[0].Name, alpha.Folders[0].Name)
	}
	if shownTab(t, a, alphaInner).Dirty || shownTab(t, a, betaInner).Dirty {
		t.Fatal("renaming a folder marks the requests in it")
	}

	autosaveAfterPause(a)
	if !exists(filepath.Join(betaDir, "shared", "inner.req.json")) || exists(filepath.Join(betaDir, "nested")) {
		t.Fatal("Beta's folder was not renamed on disk")
	}
	if !exists(filepath.Join(alphaDir, "nested", "inner.req.json")) || exists(filepath.Join(alphaDir, "shared")) {
		t.Fatal("Alpha's folder changed on disk")
	}
	if got, err := storeDisk.Load(betaDir); err != nil || got.Folders[0].Name != "Shared" || got.Folders[0].Requests[0].Name != "Inner" {
		t.Fatalf("Beta on disk: %+v %v", got.Folders, err)
	}
}

// Renaming Beta saves every collection first, so its writes in flight go to
// the old folder. Alpha's pending edit was written then, but its tab kept
// the dot: only Ctrl+S and the autosave cleared the marks of the tabs, and
// with nothing left to save neither ran again, so the dot stayed for good.
func TestRenamingACollectionMovesOnlyItsFolderAndClearsWhatItSaved(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	now := time.Now()
	a.now = func() time.Time { return now }
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	alphaCall, betaCall := alpha.Requests[0], beta.Requests[0]
	alphaCall.URL = "http://alpha.pending"
	a.markStale()

	a.renameCollection(beta, "Gamma")
	gammaDir := filepath.Join(filepath.Dir(betaDir), "gamma")
	if got := entryNamed(t, a, "Gamma").Dir; got != gammaDir {
		t.Fatalf("Gamma is in %q, want %q", got, gammaDir)
	}
	if got := entryNamed(t, a, "Alpha").Dir; got != alphaDir {
		t.Fatalf("Alpha moved to %q", got)
	}
	if got := rememberedFolders(a); !slices.Equal(got, []string{alphaDir, gammaDir}) {
		t.Fatalf("remembered %v", got)
	}
	if exists(betaDir) {
		t.Fatal("Beta's old folder is still there")
	}
	if got, err := storeDisk.Load(alphaDir); err != nil || got.Name != "Alpha" || got.Requests[0].URL != "http://alpha.pending" {
		t.Fatalf("Alpha on disk: %+v %v", got, err)
	}
	if shownTab(t, a, alphaCall).Dirty {
		t.Fatal("Alpha was saved with the rename, but its tab is still marked")
	}
	activate(t, a, alphaCall)
	if title := a.Title(); title != "Alpha - "+appName {
		t.Fatalf("title %q", title)
	}
	activate(t, a, betaCall)
	if title := a.Title(); title != "Gamma - "+appName {
		t.Fatalf("title %q", title)
	}

	betaCall.URL = "http://gamma.edited"
	autosaveAfterPause(a)
	if got, err := storeDisk.Load(gammaDir); err != nil || got.Requests[0].URL != "http://gamma.edited" {
		t.Fatalf("Gamma on disk: %+v %v", got.Requests, err)
	}
	if exists(betaDir) {
		t.Fatal("a later save made Beta's old folder again")
	}
}

// The tab of Alpha's request is open but Beta's is active and Beta's
// request is selected: the rename still acts on Alpha, and a name only
// Beta has is not taken in Alpha.
func TestRenamingARequestWhileAnotherCollectionsTabIsActive(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	alphaCall, betaCall := alpha.Requests[0], beta.Requests[0]
	if a.cur().Req != betaCall || a.side.Tree().Selected() != request.Of(betaCall) {
		t.Fatalf("active %q, selected %v", a.cur().Req.Name, a.side.Tree().Selected())
	}

	if err := a.applyTree(tree.Rename{Node: request.Of(alphaCall), Name: "Beta call"}).Err; err != nil {
		t.Fatalf("a name used only in Beta is refused in Alpha: %v", err)
	}
	if alphaCall.Name != "Beta call" || betaCall.Name != "Beta call" {
		t.Fatalf("Alpha's %q, Beta's %q", alphaCall.Name, betaCall.Name)
	}
	if a.cur().Req != betaCall {
		t.Fatalf("the active tab moved to %q", a.cur().Req.Name)
	}
	if shownTab(t, a, betaCall).Dirty || !shownTab(t, a, alphaCall).Dirty {
		t.Fatal("only Alpha's renamed request is unsaved")
	}
	if titleMarked(a) {
		t.Fatalf("Beta has no edits, title %q", a.Title())
	}

	autosaveAfterPause(a)
	if !exists(filepath.Join(alphaDir, "beta-call.req.json")) || exists(filepath.Join(alphaDir, "alpha-call.req.json")) {
		t.Fatal("Alpha's request file was not renamed")
	}
	if !exists(filepath.Join(betaDir, "beta-call.req.json")) {
		t.Fatal("Beta's request file is gone")
	}
	if got, err := storeDisk.Load(betaDir); err != nil || got.Requests[0].URL != "http://{{host}}/Beta" {
		t.Fatalf("Beta on disk: %+v %v", got.Requests, err)
	}
	if shownTab(t, a, alphaCall).Dirty {
		t.Fatal("Alpha's tab is still marked after the autosave")
	}
}
