package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history/adapter/historyfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

// Alpha and Beta both hold Nested/Inner; the history of one must not show
// in the other, and the history written before several collections could
// be open (keyed by the bare path) still belongs to the first.
func TestHistoryIsKeptPerCollectionAndOldHistoryStaysWithTheFirst(t *testing.T) {
	a, _, _ := twoCollectionsAppWith(t, Options{StatePath: filepath.Join(t.TempDir(), "state.json")})
	alphaInner := entryNamed(t, a, "Alpha").Collection.Folders[0].Requests[0]
	betaInner := entryNamed(t, a, "Beta").Collection.Folders[0].Requests[0]
	old := []history.Entry{{At: time.Now(), Status: 201, Body: []byte("old")}}
	if err := historyfile.Dir(a.historyDir()).Save("Nested/Inner", old); err != nil {
		t.Fatal(err)
	}
	if got := a.loadHistory(alphaInner); len(got) != 1 || got[0].Resp.Status != 201 {
		t.Fatalf("Alpha's old history %+v", got)
	}
	if got := a.loadHistory(betaInner); len(got) != 0 {
		t.Fatalf("Beta reads Alpha's old history: %+v", got)
	}

	a.openRequest(betaInner)
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200, Body: []byte("beta")}
	a.saveHistory(tab)
	a.WaitWrites()
	if got := a.loadHistory(betaInner); len(got) != 1 || got[0].Resp.Status != 200 {
		t.Fatalf("Beta's history %+v", got)
	}
	if got := a.loadHistory(alphaInner); len(got) != 1 || got[0].Resp.Status != 201 {
		t.Fatalf("Alpha's history after Beta's send %+v", got)
	}
}

func TestASentRowOpensTheRequestOfItsOwnCollection(t *testing.T) {
	a, _, betaDir := twoCollectionsAppWith(t, Options{StatePath: filepath.Join(t.TempDir(), "state.json")})
	alphaInner := entryNamed(t, a, "Alpha").Collection.Folders[0].Requests[0]
	betaInner := entryNamed(t, a, "Beta").Collection.Folders[0].Requests[0]
	a.openRequest(betaInner)
	tab := a.cur()
	pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	row := a.side.Sent().Rows("")[0]
	if row.Collection != betaDir || row.Path != "Nested/Inner" {
		t.Fatalf("row %+v", row)
	}

	a.openRequest(alphaInner)
	a.side.Sent().Open(a, row)
	if a.cur().Req != betaInner {
		t.Fatalf("opened %q of the wrong collection", a.cur().Req.Name)
	}
	// a row written before rows named their collection is the first one's
	a.side.Sent().Open(a, history.Sent{Path: "Nested/Inner"})
	if a.cur().Req != alphaInner {
		t.Fatal("an old row did not open the first collection's request")
	}
}

// Started on a folder given by -workspace, the open set is not the
// remembered list; picking an environment updates the folder's entry in
// the list and keeps the rest instead of replacing it.
func TestAnEnvironmentPickKeepsTheRememberedListItDidNotComeFrom(t *testing.T) {
	alphaDir, otherDir := savedCollection(t, "Alpha"), savedCollection(t, "Other")
	cases := []struct {
		name       string
		remembered []request.OpenCollection
		want       []request.OpenCollection
	}{
		{
			name:       "listed",
			remembered: []request.OpenCollection{{Folder: otherDir, Environment: "Dev"}, {Folder: alphaDir}},
			want:       []request.OpenCollection{{Folder: otherDir, Environment: "Dev"}, {Folder: alphaDir, Environment: "Dev"}},
		},
		{
			name:       "not listed",
			remembered: []request.OpenCollection{{Folder: otherDir}},
			want:       []request.OpenCollection{{Folder: otherDir}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			settingsPath := filepath.Join(t.TempDir(), "settings.json")
			start := settings.Default()
			start.Collections = tc.remembered
			if err := settingsDisk.Save(settingsPath, start); err != nil {
				t.Fatal(err)
			}
			a, err := NewApp(withAdapters(Options{SettingsPath: settingsPath, WorkspaceDir: alphaDir}))
			if err != nil {
				t.Fatal(err)
			}
			a.selectEnv(0)
			written, err := settingsDisk.Load(settingsPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(written.Collections) != len(tc.want) {
				t.Fatalf("remembered %+v, want %+v", written.Collections, tc.want)
			}
			for i := range tc.want {
				if written.Collections[i] != tc.want[i] {
					t.Fatalf("remembered %+v, want %+v", written.Collections, tc.want)
				}
			}
		})
	}
}

// A remembered folder that fails to load (a drive not mounted yet, a
// broken file) was left out of the list the next change of the open set
// wrote, so it was forgotten for good.
func TestACollectionThatFailedToLoadStaysInTheRememberedList(t *testing.T) {
	alphaDir, brokenDir := savedCollection(t, "Alpha"), t.TempDir()
	if err := os.WriteFile(filepath.Join(brokenDir, "folder.json"), []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	start := settings.Default()
	start.Collections = []request.OpenCollection{{Folder: alphaDir}, {Folder: brokenDir, Environment: "Dev"}}
	if err := settingsDisk.Save(settingsPath, start); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: settingsPath}))
	if err != nil {
		t.Fatal(err)
	}
	a.rememberCollections()
	written, err := settingsDisk.Load(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(written.Collections) != 2 || written.Collections[1] != (request.OpenCollection{Folder: brokenDir, Environment: "Dev"}) {
		t.Fatalf("remembered %+v, want the broken folder kept after %q", written.Collections, alphaDir)
	}
}

func TestAnOutOfRangeSettingIsClampedAndNamedInTheStatusLine(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"timeoutSec": 99999999999}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: settingsPath}))
	if err != nil {
		t.Fatal(err)
	}
	if a.settings.TimeoutSec != 3600 || !strings.Contains(a.notices.Status(), "timeoutSec") {
		t.Fatalf("timeout %d, status %q", a.settings.TimeoutSec, a.notices.Status())
	}
	a.settings.TimeoutSec = 10
	a.revertSettings() // reading the file again applies the clamped value too
	if a.settings.TimeoutSec != 3600 {
		t.Fatalf("after revert: timeout %d, want the clamped 3600", a.settings.TimeoutSec)
	}
}
