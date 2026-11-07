package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/postman"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/store"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func workspaceApp(t *testing.T, dir string, opts Options) *App {
	t.Helper()
	opts.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	opts.WorkspaceDir = dir
	a, err := NewApp(opts)
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.step(core.Input{})
	return a
}

func TestStartsFromTheWorkspaceFolder(t *testing.T) {
	dir := t.TempDir()
	c := model.Collection{Folder: model.Folder{Name: "Mine", Requests: []*model.Request{{Name: "One", Protocol: "rest", Method: "GET", URL: "u"}}}}
	if err := store.Save(dir, c); err != nil {
		t.Fatal(err)
	}
	a := workspaceApp(t, dir, Options{})
	if a.col.Name != "Mine" || a.cur().req.Name != "One" {
		t.Fatalf("collection %q, tab %q", a.col.Name, a.cur().req.Name)
	}
}

func TestEmptyWorkspaceStartsWithOneBlankRequest(t *testing.T) {
	a := workspaceApp(t, t.TempDir(), Options{})
	if len(a.tabs) != 1 || a.cur().req.Name != "New Request" {
		t.Fatalf("tabs %d", len(a.tabs))
	}
}

func TestBrokenWorkspaceFallsBackWithoutOverwriting(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "x.req.json")
	if err := writeFile(bad, "{nope"); err != nil {
		t.Fatal(err)
	}
	a := workspaceApp(t, dir, Options{})
	if a.workspace != "" || a.status == "Ready" {
		t.Fatalf("workspace %q status %q", a.workspace, a.status)
	}
	a.saveWorkspace() // with no workspace and no default this must not write into dir
	if b := readFile(t, bad); b != "{nope" {
		t.Fatalf("broken file changed: %q", b)
	}
}

func TestSaveWritesTheWorkspaceAndClearsTheDirtyMark(t *testing.T) {
	dir := t.TempDir()
	a := workspaceApp(t, dir, Options{})
	tab := a.cur()
	tab.req.URL = "https://changed.test"
	if !dirty(tab) {
		t.Fatal("edit not dirty")
	}
	a.saveWorkspace()
	if dirty(tab) {
		t.Fatal("still dirty after saving")
	}
	got, err := store.Load(dir)
	if err != nil || got.Requests[0].URL != "https://changed.test" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestEditsAreSavedAfterAPause(t *testing.T) {
	dir := t.TempDir()
	a := workspaceApp(t, dir, Options{})
	now := time.Now()
	a.now = func() time.Time { return now }
	a.step(core.Input{})
	a.cur().req.Name = "Renamed"
	a.step(core.Input{})
	if got, _ := store.Load(dir); len(got.Requests) > 0 && got.Requests[0].Name == "Renamed" {
		t.Fatal("saved before the pause")
	}
	now = now.Add(2 * time.Second)
	a.step(core.Input{})
	got, err := store.Load(dir)
	if err != nil || got.Requests[0].Name != "Renamed" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestSavingWithoutAWorkspaceUsesTheDefaultOneAndRemembersIt(t *testing.T) {
	def := filepath.Join(t.TempDir(), "ws")
	a := workspaceApp(t, "", Options{DefaultWorkspace: def})
	if a.workspace != "" {
		t.Fatal("workspace without being asked")
	}
	a.saveWorkspace()
	if a.workspace != def {
		t.Fatalf("workspace %q", a.workspace)
	}
	got, err := store.Load(def)
	if err != nil || len(got.Folders) == 0 {
		t.Fatalf("%+v %v", got, err)
	}
	st, err := settings.Load(a.settingsPath)
	if err != nil || st.Workspace != def {
		t.Fatalf("remembered %q %v", st.Workspace, err)
	}
}

func TestRemembersTheLastWorkspace(t *testing.T) {
	dir := t.TempDir()
	if err := store.Save(dir, model.Collection{Folder: model.Folder{Name: "Last", Requests: []*model.Request{{Name: "R", Protocol: "rest", Method: "GET"}}}}); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(t.TempDir(), "settings.json")
	s := settings.Default()
	s.Workspace = dir
	if err := s.Save(sp); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(Options{SettingsPath: sp})
	if err != nil || a.col.Name != "Last" {
		t.Fatalf("%v %q", err, a.col.Name)
	}
}

func writeFile(p, s string) error { return os.WriteFile(p, []byte(s), 0o644) }
func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestImportCurlAddsARequestFromTheClipboard(t *testing.T) {
	a, _ := newTestApp(t)
	clip := &fakeClip{text: `curl -X POST 'https://api.test/v1/users?x=1' -H 'Content-Type: application/json' -d '{"a":1}'`}
	a.ctx.Clipboard = clip
	a.importCurl(&a.col.Folder)
	r := a.cur().req
	if r.Name != "POST /v1/users" || r.Method != "POST" || r.Body != `{"a":1}` {
		t.Fatalf("%+v", r)
	}
	if _, ok := a.col.Folder.Parent(r); !ok {
		t.Fatal("request not in the collection")
	}
	n := len(a.tabs)
	clip.text = "not curl"
	a.importCurl(&a.col.Folder)
	if len(a.tabs) != n {
		t.Fatal("a bad command opened a tab")
	}
}

func TestFlushWorkspaceSavesPendingEditsAtExit(t *testing.T) {
	dir := t.TempDir()
	a := workspaceApp(t, dir, Options{})
	a.cur().req.URL = "https://edited-just-now.test" // the autosave pause has not passed
	if err := a.FlushWorkspace(); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(dir)
	if err != nil || got.Requests[0].URL != "https://edited-just-now.test" {
		t.Fatalf("%+v %v", got, err)
	}
	if err := workspaceApp(t, "", Options{}).FlushWorkspace(); err != nil {
		t.Fatalf("no workspace is no error: %v", err)
	}
}

func TestOpenWorkspaceAtSwitchesCollectionsAndSavesTheOldOne(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	if err := store.Save(second, model.Collection{Folder: model.Folder{Name: "Second", Requests: []*model.Request{{Name: "Two", Protocol: "rest", Method: "GET", URL: "u2"}}}}); err != nil {
		t.Fatal(err)
	}
	a := workspaceApp(t, first, Options{})
	a.cur().req.URL = "https://unsaved.test"
	a.openWorkspaceAt(second)
	if a.col.Name != "Second" || len(a.tabs) != 1 || a.cur().req.Name != "Two" || a.workspace != second {
		t.Fatalf("col %q tabs %d workspace %q", a.col.Name, len(a.tabs), a.workspace)
	}
	if old, err := store.Load(first); err != nil || old.Requests[0].URL != "https://unsaved.test" {
		t.Fatalf("the old collection lost its edit: %+v %v", old, err)
	}
	st, err := settings.Load(a.settingsPath)
	if err != nil || st.Workspace != second {
		t.Fatalf("remembered %q %v", st.Workspace, err)
	}
	broken := t.TempDir()
	os.WriteFile(filepath.Join(broken, "bad.req.json"), []byte("{nope"), 0o644)
	a.openWorkspaceAt(broken)
	if a.workspace != second || a.col.Name != "Second" {
		t.Fatal("a folder that cannot be read must change nothing")
	}
	a.openWorkspaceAt("   ")
	if a.workspace != second {
		t.Fatal("an empty path changes nothing")
	}
}

func TestImportOpenAPIAddsAFolderOfRequestsAndTheBaseURL(t *testing.T) {
	data, err := os.ReadFile("../openapi/testdata/petstore.json")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := newTestApp(t)
	a.col.Vars = nil
	a.ctx.Clipboard = &fakeClip{text: string(data)}
	before := len(a.col.Folders)
	a.importOpenAPI(&a.col.Folder)
	if len(a.col.Folders) != before+1 {
		t.Fatalf("folders %d, want %d", len(a.col.Folders), before+1)
	}
	if !model.Defined("baseUrl", a.col.Vars) {
		t.Fatal("baseUrl not defined")
	}
	if a.cur().req.URL == "" || !strings.HasPrefix(a.cur().req.URL, "{{baseUrl}}") {
		t.Fatalf("the first imported request is not open: %q", a.cur().req.URL)
	}
	n := len(a.tabs)
	a.ctx.Clipboard = &fakeClip{text: "{}"}
	a.importOpenAPI(&a.col.Folder)
	if len(a.tabs) != n || len(a.col.Folders) != before+1 {
		t.Fatal("a bad document changed the collection")
	}
}

func TestImportPostmanAddsAFolderAndItsVariables(t *testing.T) {
	data, err := os.ReadFile("../postman/testdata/collection.json")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := newTestApp(t)
	a.col.Vars = []model.KV{{Name: "baseUrl", Value: "mine", Enabled: true}}
	a.ctx.Clipboard = &fakeClip{text: string(data)}
	before := len(a.col.Folders)
	a.importPostman(&a.col.Folder)
	if len(a.col.Folders) != before+1 || a.col.Folders[before].Name != "Echo API" {
		t.Fatalf("folders %d", len(a.col.Folders))
	}
	if len(a.col.Vars) != 1 || a.col.Vars[0].Value != "mine" {
		t.Fatalf("an existing variable must not be overwritten: %+v", a.col.Vars)
	}
	if a.cur().req.Name != "Form" { // requests of a folder come before its subfolders
		t.Fatalf("the first imported request is not open: %q", a.cur().req.Name)
	}
	n := len(a.col.Folders)
	a.ctx.Clipboard = &fakeClip{text: `{"openapi":"3.0.0"}`}
	a.importPostman(&a.col.Folder)
	if len(a.col.Folders) != n {
		t.Fatal("a bad document changed the collection")
	}
}

func TestExportPostmanWritesACollectionFileThatImportsBack(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no dialog program: the path is asked as text
	a, _ := newTestApp(t)
	a.exportPostman(&a.col.Folder)
	if a.dialog == nil || a.dialog.input == nil || !strings.HasSuffix(a.dialog.input.Text, ".postman_collection.json") {
		t.Fatalf("dialog %+v", a.dialog)
	}
	path := filepath.Join(t.TempDir(), "out.json")
	a.dialog.input.SelectAll()
	a.dialog.input.Insert(path)
	a.dialog.buttons[len(a.dialog.buttons)-1].run()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := postman.Parse(data)
	if err != nil || res.Folder.Name != a.col.Name || len(res.Folder.Folders) == 0 {
		t.Fatalf("%v %+v", err, res.Folder)
	}
	found := false
	for _, ts := range a.toasts {
		found = found || strings.Contains(ts.text, "left out") // the sample has WebSocket and MCP requests
	}
	if !found {
		t.Fatal("the message must say what was left out")
	}
}
