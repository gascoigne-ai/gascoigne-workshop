package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/adapter/store"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// collectionsApp opens the collection "Home" in <base>/workspace, the
// default workspace, so new collections go to <base>/collections.
func collectionsApp(t *testing.T) (*App, string) {
	t.Helper()
	base := t.TempDir()
	home := filepath.Join(base, "workspace")
	c := model.Collection{Folder: model.Folder{Name: "Home", Requests: []*model.Request{{Name: "One", Protocol: "rest", Method: "GET", URL: "u"}}}}
	if err := store.Save(home, c); err != nil {
		t.Fatal(err)
	}
	return workspaceApp(t, home, Options{DefaultWorkspace: home}), base
}

func TestTheCollectionMenuListsTheCollectionActions(t *testing.T) {
	a, _ := collectionsApp(t)
	var labels []string
	for _, item := range a.collectionMenu() {
		labels = append(labels, item.Label)
	}
	want := []string{"New collection...", "Open collection folder...", "", "Rename collection...", "Save collection", "", "Delete collection..."}
	if !reflect.DeepEqual(labels, want) {
		t.Fatalf("menu %q, want %q", labels, want)
	}
}

func TestRightClickOnTheCollectionsChipOpensTheCollectionMenu(t *testing.T) {
	a, _ := collectionsApp(t)
	rightClick(a, 40, headerH/2)
	if got := menuLabels(a); len(got) == 0 || got[0] != "New collection..." {
		t.Fatalf("menu %q", got)
	}
}

func TestNewCollectionMakesAFolderBesideTheWorkspaceAndOpensIt(t *testing.T) {
	a, base := collectionsApp(t)
	a.createCollection("  Café API ")
	want := filepath.Join(base, "collections", "café-api")
	if a.workspace != want || a.col.Name != "Café API" {
		t.Fatalf("workspace %q, name %q; want %q", a.workspace, a.col.Name, want)
	}
	if len(a.list.Tabs) != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("an empty collection opens one blank request, got %d tabs", len(a.list.Tabs))
	}
	if _, err := store.Load(want); err != nil {
		t.Fatal(err)
	}
	if a.settings.Workspace != want {
		t.Fatalf("remembered workspace %q", a.settings.Workspace)
	}
}

func TestNewCollectionRefusesAnEmptyNameAndKeepsTheOpenOne(t *testing.T) {
	a, _ := collectionsApp(t)
	before := a.workspace
	a.createCollection("   ")
	if a.workspace != before || a.col.Name != "Home" {
		t.Fatalf("the open collection changed: %q %q", a.workspace, a.col.Name)
	}
	if !toastSays(a, "empty") {
		t.Fatal("no toast says why")
	}
}

func TestRenameCollectionRenamesTheFolderAndKeepsTheContent(t *testing.T) {
	a, base := collectionsApp(t)
	a.renameCollection("Office")
	want := filepath.Join(base, "office")
	if a.workspace != want || a.col.Name != "Office" {
		t.Fatalf("workspace %q, name %q", a.workspace, a.col.Name)
	}
	got, err := store.Load(want)
	if err != nil || got.Name != "Office" || len(got.Requests) != 1 {
		t.Fatalf("loaded %+v, %v", got.Name, err)
	}
	if _, err := os.Stat(filepath.Join(base, "workspace")); !os.IsNotExist(err) {
		t.Fatalf("the old folder is still there: %v", err)
	}
	a.step(core.Input{})
	a.waitWrites()
	if _, err := os.Stat(filepath.Join(base, "workspace")); !os.IsNotExist(err) {
		t.Fatalf("a later write made the old folder again: %v", err)
	}
}

func TestSaveCollectionWritesTheEditsNow(t *testing.T) {
	a, _ := collectionsApp(t)
	a.cur().Req.URL = "https://changed.test"
	a.saveWorkspace()
	a.waitWrites()
	got, err := store.Load(a.workspace)
	if err != nil || got.Requests[0].URL != "https://changed.test" {
		t.Fatalf("loaded %+v, %v", got.Requests, err)
	}
}

func TestDeleteCollectionOpensTheNextOne(t *testing.T) {
	a, base := collectionsApp(t)
	a.createCollection("Second")
	second := a.workspace
	a.openWorkspaceAt(filepath.Join(base, "workspace"))
	if a.workspace != filepath.Join(base, "workspace") {
		t.Fatalf("workspace %q", a.workspace)
	}
	a.deleteCollection()
	a.waitWrites()
	if _, err := os.Stat(filepath.Join(base, "workspace")); !os.IsNotExist(err) {
		t.Fatalf("the deleted folder is still there: %v", err)
	}
	if a.workspace != second || a.col.Name != "Second" {
		t.Fatalf("opened %q (%q), want %q", a.workspace, a.col.Name, second)
	}
}

func TestDeletingTheLastCollectionOpensANewEmptyOne(t *testing.T) {
	a, base := collectionsApp(t)
	a.deleteCollection()
	a.waitWrites()
	if a.workspace == "" || !strings.HasPrefix(a.workspace, filepath.Join(base, "collections")) || a.col.Name != "My collection" {
		t.Fatalf("opened %q (%q)", a.workspace, a.col.Name)
	}
	if _, err := os.Stat(filepath.Join(base, "workspace")); !os.IsNotExist(err) {
		t.Fatalf("the deleted folder is still there: %v", err)
	}
}

func TestDeleteAsksFirst(t *testing.T) {
	a, base := collectionsApp(t)
	a.confirmDeleteCollection()
	if a.dialog == nil || !strings.Contains(a.dialog.Title, "Home") {
		t.Fatalf("dialog %+v", a.dialog)
	}
	a.closeDialog()
	if _, err := os.Stat(filepath.Join(base, "workspace")); err != nil {
		t.Fatalf("cancelling deleted it: %v", err)
	}
}

func TestRenameAndDeleteNeedACollectionOnDisk(t *testing.T) {
	a, _ := collectionsApp(t)
	a.workspace = ""
	a.promptRenameCollection()
	a.confirmDeleteCollection()
	if a.dialog != nil {
		t.Fatal("a dialog opened for a collection that is not on disk")
	}
}

// toastSays is true when a toast on the screen contains text.
func toastSays(a *App, text string) bool {
	for _, toast := range a.toasts {
		if strings.Contains(toast.Text, text) {
			return true
		}
	}
	return false
}
