package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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
	if err := storeDisk.Save(home, c); err != nil {
		t.Fatal(err)
	}
	return testApp(t, untouchedSettings(), inWorkspace(home), withOptions(Options{DefaultWorkspace: home})), base
}

func TestTheCollectionMenuListsTheCollectionActions(t *testing.T) {
	a, _ := collectionsApp(t)
	var labels []string
	for _, item := range a.collectionMenu() {
		labels = append(labels, item.Label)
	}
	want := []string{"New collection...", "Open collection folder...", "", "Rename collection...", "Save collection", "", "Close collection", "Delete collection..."}
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

func TestNewCollectionMakesAFolderBesideTheWorkspaceAndOpensItBesideTheOpenOne(t *testing.T) {
	a, base := collectionsApp(t)
	home := a.workspace()
	a.createCollection("  Café API ")
	want := filepath.Join(base, "collections", "café-api")
	if len(a.open.Entries) != 2 || a.workspace() != home || a.open.Entries[1].Dir != want || a.open.Entries[1].Collection.Name != "Café API" {
		t.Fatalf("open %+v; want Home then %q", a.open.Entries, want)
	}
	if owner, _ := a.open.Owner(model.Of(a.cur().Req)); owner.Dir != want || a.cur().Req.Name != "New Request" {
		t.Fatalf("an empty collection opens one blank request, shown %q of %q", a.cur().Req.Name, owner.Dir)
	}
	if _, err := storeDisk.Load(want); err != nil {
		t.Fatal(err)
	}
	if got := a.settings.RememberedCollections(); len(got) != 2 || got[0].Folder != home || got[1].Folder != want {
		t.Fatalf("remembered collections %+v", got)
	}
}

func TestNewCollectionRefusesAnEmptyNameAndKeepsTheOpenOne(t *testing.T) {
	a, _ := collectionsApp(t)
	before := a.workspace()
	a.createCollection("   ")
	if a.workspace() != before || a.col().Name != "Home" {
		t.Fatalf("the open collection changed: %q %q", a.workspace(), a.col().Name)
	}
	if !toastSays(a, "empty") {
		t.Fatal("no toast says why")
	}
}

// Renaming moved the folder but left a gRPC request's proto paths in the
// old one; the next save then wrote them as absolute paths that no longer
// exist.
func TestRenameCollectionMovesTheProtoPathsWithTheFolder(t *testing.T) {
	a, base := collectionsApp(t)
	old := a.workspace()
	request := a.col().Requests[0]
	request.Proto = model.ProtoSource{Files: []string{filepath.Join(old, "protos", "health.proto")}}
	a.renameCollection(a.col(), "Office")
	want := filepath.Join(base, "office", "protos", "health.proto")
	if got := request.Proto.Files; len(got) != 1 || got[0] != want {
		t.Fatalf("proto files %v, want [%s]", got, want)
	}
}

func TestRenameCollectionRenamesTheFolderAndKeepsTheContent(t *testing.T) {
	a, base := collectionsApp(t)
	a.renameCollection(a.col(), "Office")
	want := filepath.Join(base, "office")
	if a.workspace() != want || a.col().Name != "Office" {
		t.Fatalf("workspace %q, name %q", a.workspace(), a.col().Name)
	}
	got, err := storeDisk.Load(want)
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
	got, err := storeDisk.Load(a.workspace())
	if err != nil || got.Requests[0].URL != "https://changed.test" {
		t.Fatalf("loaded %+v, %v", got.Requests, err)
	}
}

func TestDeleteCollectionOpensTheNextOne(t *testing.T) {
	a, base := collectionsApp(t)
	a.createCollection("Second")
	second := a.open.Entries[1].Dir // made beside Home
	a.openWorkspaceAt(filepath.Join(base, "workspace"))
	if a.workspace() != filepath.Join(base, "workspace") {
		t.Fatalf("workspace %q", a.workspace())
	}
	a.deleteCollection(a.col())
	a.waitWrites()
	if _, err := os.Stat(filepath.Join(base, "workspace")); !os.IsNotExist(err) {
		t.Fatalf("the deleted folder is still there: %v", err)
	}
	if a.workspace() != second || a.col().Name != "Second" {
		t.Fatalf("opened %q (%q), want %q", a.workspace(), a.col().Name, second)
	}
}

func TestDeletingTheLastCollectionOpensANewEmptyOne(t *testing.T) {
	a, base := collectionsApp(t)
	a.deleteCollection(a.col())
	a.waitWrites()
	if a.workspace() == "" || !strings.HasPrefix(a.workspace(), filepath.Join(base, "collections")) || a.col().Name != "My collection" {
		t.Fatalf("opened %q (%q)", a.workspace(), a.col().Name)
	}
	if _, err := os.Stat(filepath.Join(base, "workspace")); !os.IsNotExist(err) {
		t.Fatalf("the deleted folder is still there: %v", err)
	}
}

func TestDeleteAsksFirst(t *testing.T) {
	a, base := collectionsApp(t)
	a.confirmDeleteCollection(a.col())
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
	a.setWorkspace("")
	a.promptRenameCollection(a.col())
	a.confirmDeleteCollection(a.col())
	if a.dialog != nil {
		t.Fatal("a dialog opened for a collection that is not on disk")
	}
}
