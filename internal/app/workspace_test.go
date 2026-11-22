package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/postman"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/filebrowser"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestStartsFromTheWorkspaceFolder(t *testing.T) {
	dir := t.TempDir()
	c := request.Collection{Folder: request.Folder{Name: "Mine", Requests: []*request.Request{{Name: "One", Protocol: "rest", Method: "GET", URL: "u"}}}}
	if err := storeDisk.Save(dir, c); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, untouchedSettings(), inWorkspace(dir), withOptions(Options{}))
	if a.col().Name != "Mine" || a.cur().Req.Name != "One" {
		t.Fatalf("collection %q, tab %q", a.col().Name, a.cur().Req.Name)
	}
}

func TestEmptyWorkspaceStartsWithOneBlankRequest(t *testing.T) {
	a := testApp(t, untouchedSettings(), inWorkspace(t.TempDir()), withOptions(Options{}))
	if len(a.list.Tabs) != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("tabs %d", len(a.list.Tabs))
	}
}

func TestBrokenWorkspaceFallsBackWithoutOverwriting(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "x.req.json")
	if err := writeFile(bad, "{nope"); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, untouchedSettings(), inWorkspace(dir), withOptions(Options{}))
	if a.workspace() != "" || a.status == "Ready" {
		t.Fatalf("workspace %q status %q", a.workspace(), a.status)
	}
	a.saveWorkspace() // with no workspace and no default this must not write into dir
	if b := fileText(t, bad); b != "{nope" {
		t.Fatalf("broken file changed: %q", b)
	}
}

func TestSaveWritesTheWorkspaceAndClearsTheDirtyMark(t *testing.T) {
	dir := t.TempDir()
	a := testApp(t, untouchedSettings(), inWorkspace(dir), withOptions(Options{}))
	tab := a.cur()
	tab.Req.URL = "https://changed.test"
	if !dirty(tab) {
		t.Fatal("edit not dirty")
	}
	a.saveWorkspace()
	a.waitWrites()
	if dirty(tab) {
		t.Fatal("still dirty after saving")
	}
	got, err := storeDisk.Load(dir)
	if err != nil || got.Requests[0].URL != "https://changed.test" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestEditsAreSavedAfterAPause(t *testing.T) {
	dir := t.TempDir()
	a := testApp(t, untouchedSettings(), inWorkspace(dir), withOptions(Options{}))
	now := time.Now()
	a.now = func() time.Time { return now }
	a.step(core.Input{})
	a.cur().Req.Name = "Renamed"
	a.markStale() // the test edits the model directly; in the app an edit comes with input, which marks the keys stale
	a.step(core.Input{})
	if got, _ := storeDisk.Load(dir); len(got.Requests) > 0 && got.Requests[0].Name == "Renamed" {
		t.Fatal("saved before the pause")
	}
	now = now.Add(2 * time.Second)
	a.step(core.Input{})
	a.waitWrites()
	got, err := storeDisk.Load(dir)
	if err != nil || got.Requests[0].Name != "Renamed" {
		t.Fatalf("%+v %v", got, err)
	}
}

// The first save of a collection without a folder makes one for it in the
// collections folder, named after it, as New collection does.
func TestTheFirstSaveMakesTheCollectionAFolderAndRemembersIt(t *testing.T) {
	base := t.TempDir()
	a := testApp(t, untouchedSettings(), inWorkspace(""), withOptions(Options{DefaultWorkspace: filepath.Join(base, "ws")}))
	if a.workspace() != "" {
		t.Fatal("workspace without being asked")
	}
	a.saveWorkspace()
	a.waitWrites()
	want := filepath.Join(base, "collections", "github-rest-api")
	if a.workspace() != want {
		t.Fatalf("workspace %q, want %q", a.workspace(), want)
	}
	got, err := storeDisk.Load(want)
	if err != nil || len(got.Folders) == 0 {
		t.Fatalf("%+v %v", got, err)
	}
	st, err := settingsDisk.Load(a.settingsPath)
	if remembered := st.RememberedCollections(); err != nil || len(remembered) != 1 || remembered[0].Folder != want {
		t.Fatalf("remembered %+v %v", remembered, err)
	}
}

func TestRemembersTheLastWorkspace(t *testing.T) {
	dir := t.TempDir()
	if err := storeDisk.Save(dir, request.Collection{Folder: request.Folder{Name: "Last", Requests: []*request.Request{{Name: "R", Protocol: "rest", Method: "GET"}}}}); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(t.TempDir(), "settings.json")
	s := settings.Default()
	s.Workspace = dir
	if err := settingsDisk.Save(sp, s); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: sp}))
	if err != nil || a.col().Name != "Last" {
		t.Fatalf("%v %q", err, a.col().Name)
	}
}

func writeFile(name, text string) error { return os.WriteFile(name, []byte(text), 0o644) }

func TestImportCurlAddsARequestFromTheClipboard(t *testing.T) {
	a := testApp(t)
	clip := &fakeClip{text: `curl -X POST 'https://api.test/v1/users?x=1' -H 'Content-Type: application/json' -d '{"a":1}'`}
	a.ctx.Clipboard = clip
	a.importCurl(&a.col().Folder)
	r := a.cur().Req
	if r.Name != "POST /v1/users" || r.Method != "POST" || r.Body != `{"a":1}` {
		t.Fatalf("%+v", r)
	}
	if _, ok := a.col().Folder.Parent(request.Of(r)); !ok {
		t.Fatal("request not in the collection")
	}
	n := len(a.list.Tabs)
	clip.text = "not curl"
	a.importCurl(&a.col().Folder)
	if len(a.list.Tabs) != n {
		t.Fatal("a bad command opened a tab")
	}
}

func TestFlushWorkspaceSavesPendingEditsAtExit(t *testing.T) {
	dir := t.TempDir()
	a := testApp(t, untouchedSettings(), inWorkspace(dir), withOptions(Options{}))
	a.cur().Req.URL = "https://edited-just-now.test" // the autosave pause has not passed
	if err := a.FlushWorkspace(); err != nil {
		t.Fatal(err)
	}
	got, err := storeDisk.Load(dir)
	if err != nil || got.Requests[0].URL != "https://edited-just-now.test" {
		t.Fatalf("%+v %v", got, err)
	}
	if err := testApp(t, untouchedSettings(), inWorkspace(""), withOptions(Options{})).FlushWorkspace(); err != nil {
		t.Fatalf("no workspace is no error: %v", err)
	}
}

func TestOpenWorkspaceAtSwitchesCollectionsAndSavesTheOldOne(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	if err := storeDisk.Save(second, request.Collection{Folder: request.Folder{Name: "Second", Requests: []*request.Request{{Name: "Two", Protocol: "rest", Method: "GET", URL: "u2"}}}}); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, untouchedSettings(), inWorkspace(first), withOptions(Options{}))
	a.cur().Req.URL = "https://unsaved.test"
	a.openWorkspaceAt(second)
	if a.col().Name != "Second" || len(a.list.Tabs) != 1 || a.cur().Req.Name != "Two" || a.workspace() != second {
		t.Fatalf("col %q tabs %d workspace %q", a.col().Name, len(a.list.Tabs), a.workspace())
	}
	if old, err := storeDisk.Load(first); err != nil || old.Requests[0].URL != "https://unsaved.test" {
		t.Fatalf("the old collection lost its edit: %+v %v", old, err)
	}
	st, err := settingsDisk.Load(a.settingsPath)
	if remembered := st.RememberedCollections(); err != nil || len(remembered) != 1 || remembered[0].Folder != second {
		t.Fatalf("remembered %+v %v", remembered, err)
	}
	broken := t.TempDir()
	os.WriteFile(filepath.Join(broken, "bad.req.json"), []byte("{nope"), 0o644)
	a.openWorkspaceAt(broken)
	if a.workspace() != second || a.col().Name != "Second" {
		t.Fatal("a folder that cannot be read must change nothing")
	}
	a.openWorkspaceAt("   ")
	if a.workspace() != second {
		t.Fatal("an empty path changes nothing")
	}
}

func TestImportOpenAPIAddsAFolderOfRequestsAndTheBaseURL(t *testing.T) {
	data, err := os.ReadFile("../collection/openapi/testdata/petstore.json")
	if err != nil {
		t.Fatal(err)
	}
	a := testApp(t)
	a.col().Vars = nil
	a.ctx.Clipboard = &fakeClip{text: string(data)}
	before := len(a.col().Folders)
	a.importOpenAPI(&a.col().Folder)
	if len(a.col().Folders) != before+1 {
		t.Fatalf("folders %d, want %d", len(a.col().Folders), before+1)
	}
	if !request.Defined("baseUrl", a.col().Vars) {
		t.Fatal("baseUrl not defined")
	}
	if a.cur().Req.URL == "" || !strings.HasPrefix(a.cur().Req.URL, "{{baseUrl}}") {
		t.Fatalf("the first imported request is not open: %q", a.cur().Req.URL)
	}
	n := len(a.list.Tabs)
	a.ctx.Clipboard = &fakeClip{text: "{}"}
	a.importOpenAPI(&a.col().Folder)
	if len(a.list.Tabs) != n || len(a.col().Folders) != before+1 {
		t.Fatal("a bad document changed the collection")
	}
}

func TestImportPostmanAddsAFolderAndItsVariables(t *testing.T) {
	data, err := os.ReadFile("../collection/postman/testdata/collection.json")
	if err != nil {
		t.Fatal(err)
	}
	a := testApp(t)
	a.col().Vars = []request.KV{{Name: "baseUrl", Value: "mine", Enabled: true}}
	a.ctx.Clipboard = &fakeClip{text: string(data)}
	before := len(a.col().Folders)
	a.importPostman(&a.col().Folder)
	if len(a.col().Folders) != before+1 || a.col().Folders[before].Name != "Echo API" {
		t.Fatalf("folders %d", len(a.col().Folders))
	}
	if len(a.col().Vars) != 3 || a.col().Vars[0].Value != "mine" || a.col().Vars[1].Name != "token" || a.col().Vars[2].Name != "apiKey" {
		t.Fatalf("an existing variable must not be overwritten: %+v", a.col().Vars)
	}
	if a.cur().Req.Name != "Form" { // requests of a folder come before its subfolders
		t.Fatalf("the first imported request is not open: %q", a.cur().Req.Name)
	}
	n := len(a.col().Folders)
	a.ctx.Clipboard = &fakeClip{text: `{"openapi":"3.0.0"}`}
	a.importPostman(&a.col().Folder)
	if len(a.col().Folders) != n {
		t.Fatal("a bad document changed the collection")
	}
}

func TestExportPostmanWritesACollectionFileThatImportsBack(t *testing.T) {
	a := testApp(t) // no dialog program: the in-app file browser asks
	a.folders = folders.Disk{}
	a.exportPostman(&a.col().Folder)
	if a.browser == nil || !strings.HasSuffix(a.browser.path.Text, ".postman_collection.json") {
		t.Fatalf("browser %+v", a.browser)
	}
	path := filepath.Join(t.TempDir(), "out.json")
	typePath(a, path)
	a.browse(filebrowser.Entered{})
	a.waitWrites() // written in the background
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := postman.Parse(data)
	if err != nil || res.Folder.Name != a.col().Name || len(res.Folder.Folders) == 0 {
		t.Fatalf("%v %+v", err, res.Folder)
	}
	found := false
	for _, ts := range a.toasts {
		found = found || strings.Contains(ts.Text, "left out") // the sample has WebSocket and MCP requests
	}
	if !found {
		t.Fatal("the message must say what was left out")
	}
}

// Every build reopens what it remembered (the dev build in its own -dev
// directory, so it never sees the release build's list); a folder given
// with -workspace still opens alone.
func TestStartOpensTheRememberedWorkspaceUnlessAFolderIsGiven(t *testing.T) {
	remembered, flagged := t.TempDir(), t.TempDir()
	for dir, name := range map[string]string{remembered: "Last", flagged: "Asked"} {
		c := request.Collection{Folder: request.Folder{Name: name, Requests: []*request.Request{{Name: "R", Protocol: "rest", Method: "GET"}}}}
		if err := storeDisk.Save(dir, c); err != nil {
			t.Fatal(err)
		}
	}
	sample := func() request.Collection {
		return request.Collection{Folder: request.Folder{Name: "Sample", Requests: []*request.Request{{Name: "S", Protocol: "rest", Method: "GET"}}}}
	}
	sp := filepath.Join(t.TempDir(), "settings.json")
	s := settings.Default()
	s.Workspace = remembered
	if err := settingsDisk.Save(sp, s); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{"the flag wins", Options{SettingsPath: sp, Sample: sample, WorkspaceDir: flagged}, "Asked"},
		{"remembered is opened", Options{SettingsPath: sp, Sample: sample}, "Last"},
		{"the sample on a first start", Options{Sample: sample}, "Sample"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a, err := NewApp(withAdapters(test.opts))
			if err != nil || a.col().Name != test.want {
				t.Fatalf("%v: collection %q, want %q", err, a.col().Name, test.want)
			}
		})
	}
}

// An OpenAPI path item that is not an object was dropped without a word;
// the import now says so in a toast.
func TestImportOpenAPIWarnsAboutDroppedPathItems(t *testing.T) {
	a := testApp(t)
	a.ctx.Clipboard = &fakeClip{text: `{"openapi":"3.0.0","info":{"title":"Odd"},"paths":{"/ok":{"get":{"summary":"List"}},"/bad":"nope"}}`}
	a.importOpenAPI(&a.col().Folder)
	for _, toast := range a.toasts {
		if strings.Contains(toast.Text, "/bad") {
			return
		}
	}
	t.Fatalf("no toast names the dropped path; toasts %+v", a.toasts)
}
