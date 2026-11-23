package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// A file a newer build wrote: its version is above what this build reads.
// Not a fixture, since no such build exists yet.
const newerSettings = `{"version": 4, "appearance": {"theme": "light"}}`

// unchangedFile fails when path no longer holds want, or a backup appeared.
func unchangedFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("%s = %q, %v; want it left as %q", filepath.Base(path), got, err, want)
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Fatalf("%s was backed up as if broken", filepath.Base(path))
	}
}

func TestNewerSettingsAreNotWrittenThisSession(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(newerSettings), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, settingsAt(path))
	if a.settings.Theme != settings.Default().Theme {
		t.Fatalf("theme %q, want the defaults for this session", a.settings.Theme)
	}
	if toasts := errorToasts(a, "newer version"); len(toasts) != 1 || !strings.Contains(toasts[0], "settings.json") {
		t.Fatalf("toasts %q, want one naming settings.json and the newer version", toasts)
	}

	a.saveSettings(func(prefs *settings.Settings) { prefs.Layout = "side" })      // the form, auto-save on
	a.over.Settings.Send(prefs.Persist{})                                         // the Save button
	_ = a.over.Settings.SettingsStore().WriteText(path, []byte(`{"version": 1}`)) // the raw editor
	other := t.TempDir()
	if err := storeDisk.Save(other, request.Collection{Folder: request.Folder{Name: "Other"}}); err != nil {
		t.Fatal(err)
	}
	a.openCollectionAt(other) // remembers the open collections in the settings
	a.WaitWrites()
	unchangedFile(t, path, newerSettings)
}

func TestNewerStateIsNotWrittenThisSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	newer := `{"version": 2, "window": {"w": 900, "h": 700}}`
	if err := os.WriteFile(path, []byte(newer), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, memorySettings(), withOptions(startOptions{StatePath: path}))
	if toasts := errorToasts(a, "newer version"); len(toasts) != 1 || !strings.Contains(toasts[0], "state.json") {
		t.Fatalf("toasts %q, want one naming state.json and the newer version", toasts)
	}
	if got := a.RestoredWindow(4000, 3000); got != (state.Window{}) {
		t.Fatalf("window %+v, want the empty state for this session", got)
	}

	clock := time.Unix(1000, 0)
	a.now = func() time.Time { return clock }
	a.sidebarSplit.Size = 280
	a.Step(core.Input{})
	clock = clock.Add(10 * time.Second) // past the quiet spell that writes the state
	a.Step(core.Input{})
	a.WaitWrites()
	if err := a.SaveState(); err != nil { // on exit
		t.Fatal(err)
	}
	unchangedFile(t, path, newer)
}

func TestNewerCollectionIsNotOpened(t *testing.T) {
	dir := t.TempDir()
	if err := storeDisk.Save(dir, request.Collection{Folder: request.Folder{Name: "Newer", Requests: []*request.Request{{Name: "A", Protocol: request.ProtocolREST, URL: "http://x.test"}}}}); err != nil {
		t.Fatal(err)
	}
	folderFile := filepath.Join(dir, "folder.json")
	newer := `{"version": 2, "name": "Newer"}`
	if err := os.WriteFile(folderFile, []byte(newer), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testApp(t)
	before := len(a.cols.Set().Entries)
	a.openCollectionAt(dir)
	a.WaitWrites()
	if len(a.cols.Set().Entries) != before {
		t.Fatalf("%d collections open, want %d: the newer one is not opened", len(a.cols.Set().Entries), before)
	}
	toasts := errorToasts(a, "folder.json")
	if len(toasts) != 1 || !strings.Contains(toasts[0], "format 2, this build reads up to 1") {
		t.Fatalf("toasts %q, want one naming folder.json and both versions", toasts)
	}
	unchangedFile(t, folderFile, newer)
}

// A remembered collection that a newer build saved, met at start: it is not
// opened, a toast says why, and it stays remembered.
func TestNewerRememberedCollectionIsToastedAtStart(t *testing.T) {
	dir := t.TempDir()
	if err := storeDisk.Save(dir, request.Collection{Folder: request.Folder{Name: "Newer"}}); err != nil {
		t.Fatal(err)
	}
	newer := `{"version": 2, "name": "Newer"}`
	if err := os.WriteFile(filepath.Join(dir, "folder.json"), []byte(newer), 0o644); err != nil {
		t.Fatal(err)
	}
	prefs := settings.Default()
	prefs.RememberCollections([]request.OpenCollection{{Folder: dir}})
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	if err := settingsDisk.Save(settingsPath, prefs); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, settingsAt(settingsPath), withOptions(startOptions{StatePath: ""}))
	toasts := errorToasts(a, "folder.json")
	if len(toasts) != 1 || !strings.Contains(toasts[0], "format 2, this build reads up to 1") {
		t.Fatalf("toasts %q, want one naming folder.json and both versions", toasts)
	}
	if _, open := a.cols.Set().At(dir); open || len(a.cols.Unreadable()) != 1 {
		t.Fatalf("open %v, unreadable %+v; want it closed and still remembered", open, a.cols.Unreadable())
	}
	unchangedFile(t, filepath.Join(dir, "folder.json"), newer)
}
