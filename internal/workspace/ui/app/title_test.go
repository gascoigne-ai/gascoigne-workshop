package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestTitleShowsTheCollectionAndAnUnsavedMark(t *testing.T) {
	a, _ := newTestApp(t)
	if got := a.Title(); got != a.col.Name+" - "+appName {
		t.Fatalf("title %q", got)
	}
	a.cur().Req.URL = "https://changed.test"
	if got := a.Title(); !strings.HasPrefix(got, dirtyMark+" ") {
		t.Fatalf("an edited request in the in-memory sample is unsaved: %q", got)
	}

	dir := t.TempDir()
	w := workspaceApp(t, dir, Options{})
	now := time.Now()
	w.now = func() time.Time { return now }
	if strings.HasPrefix(w.Title(), dirtyMark) {
		t.Fatalf("a fresh workspace is saved: %q", w.Title())
	}
	w.cur().Req.URL = "https://typed.test"
	if !strings.HasPrefix(w.Title(), dirtyMark) {
		t.Fatalf("a pending edit shows: %q", w.Title())
	}
	w.step(core.Input{})           // the autosave notices the change
	now = now.Add(2 * time.Second) // and the pause passes
	w.step(core.Input{})
	if strings.HasPrefix(w.Title(), dirtyMark) {
		t.Fatalf("after the autosave: %q", w.Title())
	}
}
