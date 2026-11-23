package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// rootRow is the index of the tree row of the collection called name.
func rootRow(t *testing.T, app *App, name string) int {
	t.Helper()
	for i, row := range app.side.VisibleRows() {
		if app.cols.Set().IsRoot(row.Dir) && row.Name == name {
			return i
		}
	}
	t.Fatalf("no root row %q", name)
	return -1
}

// openNames are the names of the open collections in tree order.
func openNames(app *App) []string {
	var names []string
	for _, entry := range app.cols.Set().Entries {
		names = append(names, entry.Collection.Name)
	}
	return names
}

// rememberedFolders are the folders of the remembered collections, in order.
func rememberedFolders(app *App) []string {
	var folders []string
	for _, item := range app.settings.RememberedCollections() {
		folders = append(folders, item.Folder)
	}
	return folders
}

func TestCtrlOOpensAnotherCollectionBesideTheOpenOne(t *testing.T) {
	a, alphaDir, _ := twoCollectionsApp(t)
	gammaDir := savedCollection(t, "Gamma")
	a.Step(combo(ebiten.KeyO, true, false, false))
	if a.dialog == nil {
		t.Fatal("no prompt for the folder")
	}
	a.typeText(gammaDir)
	a.Step(core.Input{Enter: true})
	if got := openNames(a); !reflect.DeepEqual(got, []string{"Alpha", "Beta", "Gamma"}) {
		t.Fatalf("open %v", got)
	}
	if a.cur().Req.Name != "Gamma call" {
		t.Fatalf("shown %q, want Gamma's first request", a.cur().Req.Name)
	}
	if got := rememberedFolders(a); len(got) != 3 || got[2] != gammaDir {
		t.Fatalf("remembered %v", got)
	}

	// an open folder is focused, not opened twice
	a.Step(combo(ebiten.KeyO, true, false, false))
	a.typeText(alphaDir)
	a.Step(core.Input{Enter: true})
	if got := openNames(a); len(got) != 3 {
		t.Fatalf("open %v", got)
	}
	if owner, _ := a.cols.Set().Owner(request.Of(a.cur().Req)); owner.Dir != alphaDir {
		t.Fatalf("shown %q of %q, want Alpha's", a.cur().Req.Name, owner.Dir)
	}
}

func TestOpeningAFolderThatCannotBeReadKeepsTheOpenOnes(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	broken := t.TempDir()
	if err := writeFile(filepath.Join(broken, "x.req.json"), "{nope"); err != nil {
		t.Fatal(err)
	}
	a.openCollectionAt(broken)
	if got := openNames(a); !reflect.DeepEqual(got, []string{"Alpha", "Beta"}) {
		t.Fatalf("open %v", got)
	}
	if !toastSays(a, "Cannot open") {
		t.Fatal("no toast says why")
	}
}

func TestTheRootRowMenuMovesAndClosesItsCollection(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	rightClick(a, 100, treeRowY(rootRow(t, a, "Beta")))
	if !itemByLabel(t, a, "Move down").Disabled || itemByLabel(t, a, "Move up").Disabled {
		t.Fatal("the last collection moves up only")
	}
	pickMenu(t, a, "Move up")
	if got := openNames(a); !reflect.DeepEqual(got, []string{"Beta", "Alpha"}) {
		t.Fatalf("open %v after Move up", got)
	}
	if got := rememberedFolders(a); !reflect.DeepEqual(got, []string{betaDir, alphaDir}) {
		t.Fatalf("remembered %v", got)
	}
	if rootRow(t, a, "Beta") != 0 {
		t.Fatal("Beta's root is not the first row")
	}

	rightClick(a, 100, treeRowY(rootRow(t, a, "Beta")))
	if itemByLabel(t, a, "Move down").Disabled || !itemByLabel(t, a, "Move up").Disabled {
		t.Fatal("the first collection moves down only")
	}
	pickMenu(t, a, "Move down")
	if got := openNames(a); !reflect.DeepEqual(got, []string{"Alpha", "Beta"}) {
		t.Fatalf("open %v after Move down", got)
	}

	rightClick(a, 100, treeRowY(rootRow(t, a, "Beta")))
	pickMenu(t, a, "Close collection")
	if got := openNames(a); !reflect.DeepEqual(got, []string{"Alpha"}) {
		t.Fatalf("open %v after Close", got)
	}
	if _, err := os.Stat(filepath.Join(betaDir, "folder.json")); err != nil {
		t.Fatalf("closing removed files: %v", err)
	}
}

func TestTheRootRowMenuListsTheCollectionActions(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	rightClick(a, 100, treeRowY(rootRow(t, a, "Alpha")))
	want := []string{
		"New request", "New folder", "Import cURL from clipboard", "Import OpenAPI from clipboard", "Import Postman from clipboard", "Export as Postman...", "Variables",
		"", "Rename collection...", "Move up", "Move down",
		"", "New collection...", "Open collection folder...",
		"", "Close collection", "Delete collection...",
	}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("root menu %q, want %q", got, want)
	}
}

// Close collection is Ctrl+Shift+W by default, next to Ctrl+W for the tab,
// and acts on the focused collection; Ctrl+W alone still closes only the tab.
func TestCtrlShiftWClosesTheFocusedCollectionAndCtrlWOnlyTheTab(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	closeCollection, _ := a.keymap().Chord("close-collection")
	if got := shortcutpage.BindingOf(closeCollection).String(); got != "Ctrl+Shift+W" {
		t.Fatalf("default %q, want Ctrl+Shift+W", got)
	}
	a.openRequest(entryNamed(t, a, "Beta").Collection.Folders[0].Requests[0])
	tabCount := len(a.tabs.Tabs())
	a.Step(combo(ebiten.KeyW, true, false, false))
	if got := openNames(a); len(got) != 2 || len(a.tabs.Tabs()) != tabCount-1 {
		t.Fatalf("Ctrl+W: open %v, %d tabs, want both open and %d tabs", got, len(a.tabs.Tabs()), tabCount-1)
	}
	a.Step(core.Input{Keys: []ebiten.Key{ebiten.KeyW}, Ctrl: true, Shift: true}) // Beta's tab is the active one
	if got := openNames(a); !reflect.DeepEqual(got, []string{"Alpha"}) {
		t.Fatalf("Ctrl+Shift+W: open %v", got)
	}
}

// A collection without a folder has nowhere to keep its edits, so closing
// it asks first, as deleting does; one with a folder is saved and closed.
func TestClosingAnEditedUnsavedCollectionAsksFirst(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	a.cur().Req.URL = "https://edited.test"
	a.requestCloseCollection(a.col())
	if a.dialog == nil || len(a.cols.Set().Entries) != 1 {
		t.Fatalf("closed without asking: dialog %v, open %d", a.dialog != nil, len(a.cols.Set().Entries))
	}
	a.Step(core.Input{Enter: true})
	if len(a.cols.Set().Entries) != 0 {
		t.Fatalf("still open after confirming: %d", len(a.cols.Set().Entries))
	}

	b, _, betaDir := twoCollectionsApp(t)
	beta := entryNamed(t, b, "Beta").Collection
	b.cur().Req.URL = "https://beta.edited"
	b.requestCloseCollection(beta)
	if b.dialog != nil || len(b.cols.Set().Entries) != 1 {
		t.Fatalf("a saved collection asked or stayed: dialog %v, open %d", b.dialog != nil, len(b.cols.Set().Entries))
	}
	b.WaitWrites()
	if data, err := os.ReadFile(filepath.Join(betaDir, "beta-call.req.json")); err != nil || !strings.Contains(string(data), "beta.edited") {
		t.Fatalf("Beta's edit was not saved: %s %v", data, err)
	}
}

// With nothing open the tree offers to open or make a collection, and the
// shortcuts that act on a collection do nothing and write nothing.
func TestTheEmptyTreeOffersOpenAndNew(t *testing.T) {
	a, base := collectionsApp(t)
	a.closeCollection(a.col())
	a.Step(core.Input{})
	textCenter(t, a, "No collection is open")
	a.Step(combo(ebiten.KeyS, true, false, false))
	a.Step(combo(ebiten.KeyT, true, false, false))
	if _, err := os.Stat(filepath.Join(base, "collections")); !os.IsNotExist(err) {
		t.Fatalf("something was written: %v", err)
	}

	clickText(t, a, "Open collection folder...", 0)
	if a.dialog == nil || a.dialog.Title != "Open collection folder" {
		t.Fatalf("dialog %+v", a.dialog)
	}
	a.Step(core.Input{Escape: true})
	clickText(t, a, "New collection...", 0)
	if a.dialog == nil || a.dialog.Title != "New collection" {
		t.Fatalf("dialog %+v", a.dialog)
	}
	a.typeText("Fresh")
	a.Step(core.Input{Enter: true})
	if got := openNames(a); !reflect.DeepEqual(got, []string{"Fresh"}) {
		t.Fatalf("open %v", got)
	}
}

func TestTheCollectionMenuIsDisabledForACollectionWithNothingOpen(t *testing.T) {
	a, _ := collectionsApp(t)
	a.closeCollection(a.col())
	for _, item := range a.side.CollectionMenu() {
		acts := item.Label == "New collection..." || item.Label == "Open collection folder..."
		if !item.Separator && item.Disabled == acts {
			t.Fatalf("%q disabled %v with nothing open", item.Label, item.Disabled)
		}
	}
}

// A socket of a closed collection's tab must not stay connected with no
// tab left to close it.
func TestClosingACollectionDisconnectsItsSockets(t *testing.T) {
	closed := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		conn, err := ws.Upgrade(writer, call)
		if err != nil {
			return
		}
		defer close(closed)
		for {
			if message, err := conn.Read(); err != nil || message.Type == ws.Close {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	a, _, _ := twoCollectionsApp(t)
	beta := entryNamed(t, a, "Beta").Collection
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	*a.urlbar.Field(tab.Tab) = widget.NewTextField(tab.Req.URL)
	a.send(tab)
	until(t, a, "connection", func() bool { return tab.Session() != nil })
	a.closeCollection(beta)
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("the socket of the closed collection stayed open")
	}
}
