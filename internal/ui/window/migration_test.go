package window

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

// testdata/one-collection holds what the version with one open collection
// wrote, byte for byte: it was captured by running that version's App
// (commit 94958d3, whose settings, state and store formats are those of
// 3505a8e, before task 071) with a test that saved the start collection
// to a folder, opened it, added the environments Dev and Prod, chose Dev,
// opened two more tabs, collapsed User and saved the state. The only edit
// is the collection's folder in settings.json, a temp path then, which is
// the placeholder "WORKSPACE" here.

// oldInstall copies the captured files to a temp dir, so the App never
// writes into testdata, and fills in the folder. It returns the paths of
// settings.json and state.json and the collection's folder.
func oldInstall(t *testing.T) (settingsPath, statePath, dir string) {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(filepath.Join("testdata", "one-collection"))); err != nil {
		t.Fatal(err)
	}
	dir = filepath.Join(root, "workspace")
	settingsPath, statePath = filepath.Join(root, "settings.json"), filepath.Join(root, "state.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	quoted, err := json.Marshal(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte(`"WORKSPACE"`), quoted, 1)
	if err := os.WriteFile(settingsPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return settingsPath, statePath, dir
}

// assertOldSessionRestored checks what the captured session had open: the
// one collection with Dev, its three tabs with Echo active, the root and
// the folders the file lists open. User, collapsed in that session, is not
// checked: a start opens every folder and the state only adds to that, in
// the version that wrote the file as in this one.
func assertOldSessionRestored(t *testing.T, app *Component, dir string) {
	t.Helper()
	if len(app.cols.Set().Entries) != 1 || app.workspace() != dir || app.envName() != "Dev" {
		t.Fatalf("open %+v, environment %q", app.cols.Set().Entries, app.envName())
	}
	var names []string
	for _, tab := range app.openTabs() {
		names = append(names, tab.Req.Name)
	}
	if want := []string{"Get User", "Repository Info", "Echo"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("tabs %v, want %v", names, want)
	}
	if app.cur().Req.Name != "Echo" {
		t.Fatalf("active %q, want Echo", app.cur().Req.Name)
	}
	col := app.col()
	open := map[string]bool{}
	for _, folder := range col.Folders {
		open[folder.Name] = app.side.Tree().Expanded()[folder]
	}
	if !app.side.Tree().Expanded()[&col.Folder] || !open["Repository"] || !open["Realtime"] {
		t.Fatalf("root open %v, folders %v", app.side.Tree().Expanded()[&col.Folder], open)
	}
}

// writtenState is the part of state.json that changed with several
// collections: tabs and open folders name their collection's folder.
type writtenState struct {
	Expanded   []string            `json:"expanded"`
	ExpandedIn map[string][]string `json:"expandedIn"`
	Tabs       []struct {
		Collection string `json:"collection"`
		Path       string `json:"path"`
	} `json:"tabs"`
}

// A user of the version with one collection keeps the session: the files
// that version wrote open as a set of one with its environment, tabs and
// folders; the next state save and the next change of the set or its
// environment write the new form, which opens the same again.
func TestFilesOfTheOneCollectionVersionOpenAsASetOfOneAndAreRewritten(t *testing.T) {
	settingsPath, statePath, dir := oldInstall(t)
	a := testApp(t, settingsAt(settingsPath), withOptions(startOptions{StatePath: statePath}))
	assertOldSessionRestored(t, a, dir)

	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}
	var stored writtenState
	readJSON(t, statePath, &stored)
	if len(stored.Expanded) != 0 || len(stored.ExpandedIn) != 1 || !containsAll(stored.ExpandedIn[dir], "Repository", "Realtime") {
		t.Fatalf("open folders written as %v and %v", stored.Expanded, stored.ExpandedIn)
	}
	for _, tab := range stored.Tabs {
		if tab.Collection != dir {
			t.Fatalf("tab %q written for collection %q, want %q", tab.Path, tab.Collection, dir)
		}
	}

	a.selectEnv(1) // Prod, then back, so the choice is written and stays Dev
	a.selectEnv(0)
	var written struct {
		General map[string]json.RawMessage `json:"general"`
	}
	readJSON(t, settingsPath, &written)
	if _, old := written.General["workspace"]; old {
		t.Fatalf("the old workspace key is still written: %s", written.General)
	}
	if _, old := written.General["environment"]; old {
		t.Fatalf("the old environment key is still written: %s", written.General)
	}
	var list []struct{ Folder, Environment string }
	if err := json.Unmarshal(written.General["collections"], &list); err != nil || len(list) != 1 || list[0].Folder != dir || list[0].Environment != "Dev" {
		t.Fatalf("collections written as %s (%v)", written.General["collections"], err)
	}

	assertOldSessionRestored(t, testApp(t, settingsAt(settingsPath), withOptions(startOptions{StatePath: statePath})), dir)
}

// containsAll says whether list holds every one of wanted.
func containsAll(list []string, wanted ...string) bool {
	for _, item := range wanted {
		if !slices.Contains(list, item) {
			return false
		}
	}
	return true
}

// readJSON decodes the file at path into value.
func readJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, value); err != nil {
		t.Fatalf("%s: %v\n%s", path, err, raw)
	}
}
