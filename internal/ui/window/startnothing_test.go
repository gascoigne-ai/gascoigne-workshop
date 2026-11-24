package window

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/clientconfig"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// startingWithNothing is the release build's start: no sample, no client
// config, no collection unless one is remembered or a folder is given.
func startingWithNothing(opts startOptions) appOption {
	opts.StartWith = collection.StartWithNothing
	return withOptions(opts)
}

// The release build used to start with an unsaved collection called
// "Workshop" (or the client config's tree). It starts with no collection and
// the one blank tab that closing the last collection leaves.
func TestStartingWithNothingOpensNoCollectionAndOneBlankTab(t *testing.T) {
	a := testApp(t, untouchedSettings(), startingWithNothing(startOptions{}))
	if got := len(a.cols.Set().Entries); got != 0 {
		t.Fatalf("%d collections open, want none", got)
	}
	if len(a.tabs.Tabs()) != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("%d tabs, current %q; want one blank tab", len(a.tabs.Tabs()), a.cur().Req.Name)
	}
	if _, owned := a.cols.Set().Owner(request.Node{Req: a.cur().Req}); owned {
		t.Fatal("the blank tab belongs to a collection")
	}
	if a.headerTitle() != "" {
		t.Fatalf("header title %q, want none", a.headerTitle())
	}
	a.Step(core.Input{})
	root := a.side.View(a.ctx, a.h)
	view.Layout(a.ctx, root, 260, 600)
	if !strings.Contains(view.Dump(root), "No collection is open") {
		t.Fatalf("the sidebar does not say that no collection is open:\n%s", view.Dump(root))
	}
}

type countingClients struct{ loads *int }

func (clients countingClients) Load() (clientconfig.Config, error) {
	*clients.loads++
	return clientconfig.Config{}, nil
}

func TestStartingWithNothingReadsNoClientConfig(t *testing.T) {
	loads := 0
	testApp(t, untouchedSettings(), startingWithNothing(startOptions{Clients: countingClients{&loads}}))
	if loads != 0 {
		t.Fatalf("the client config was read %d times", loads)
	}
}

func TestStartingWithNothingStillOpensTheWorkspaceFolder(t *testing.T) {
	dir := t.TempDir()
	mine := request.Collection{Folder: request.Folder{Name: "Mine", Requests: []*request.Request{{Name: "One", Protocol: "rest", Method: "GET", URL: "u"}}}}
	if err := storeDisk.Save(dir, mine); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, untouchedSettings(), inWorkspace(dir), startingWithNothing(startOptions{}))
	if len(a.cols.Set().Entries) != 1 || a.col().Name != "Mine" {
		t.Fatalf("%d collections, first %q; want only Mine", len(a.cols.Set().Entries), a.col().Name)
	}
}

func TestStartingWithNothingStillOpensTheRememberedCollections(t *testing.T) {
	dir := t.TempDir()
	if err := storeDisk.Save(dir, request.Collection{Folder: request.Folder{Name: "Kept"}}); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, withSettings(func(prefs *settings.Settings) {
		prefs.RememberCollections([]request.OpenCollection{{Folder: dir}})
	}), startingWithNothing(startOptions{}))
	if len(a.cols.Set().Entries) != 1 || a.col().Name != "Kept" {
		t.Fatalf("%d collections; want only Kept", len(a.cols.Set().Entries))
	}
}

// A release build used to remember its unsaved "Workshop" as an entry with
// no folder. That entry stands for nothing on disk, so it is skipped.
func TestStartingWithNothingSkipsARememberedUnsavedCollection(t *testing.T) {
	a := testApp(t, withSettings(func(prefs *settings.Settings) {
		prefs.RememberCollections([]request.OpenCollection{{Environment: "dev"}})
	}), startingWithNothing(startOptions{}))
	if got := len(a.cols.Set().Entries); got != 0 {
		t.Fatalf("%d collections open, want none", got)
	}
}
