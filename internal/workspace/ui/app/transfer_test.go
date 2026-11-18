package app

import (
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/adapter/store"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// release is a drag of node that ends on target with the pointer released,
// holding Alt when alt is set.
func release(a *App, node, target model.Node, alt bool) {
	a.dragNode, a.dropTarget = node, target
	a.step(core.Input{Alt: alt})
}

func toastTexts(a *App) string {
	var texts []string
	for _, toast := range a.toasts {
		texts = append(texts, toast.Text)
	}
	return strings.Join(texts, "\n")
}

func tabOf(a *App, req *model.Request) bool {
	for _, tab := range a.list.Tabs {
		if tab.Req == req {
			return true
		}
	}
	return false
}

func TestDragOntoAnotherCollectionsFolderMovesTheRequest(t *testing.T) {
	a, _, betaDir := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Requests[0]
	a.openRequest(req)
	a.cur().Req.URL = "http://edited.test"
	a.markStale()

	release(a, model.Of(req), model.Of(beta.Folders[0]), false)

	if len(alpha.Requests) != 0 || !beta.Folders[0].Contains(model.Of(req)) {
		t.Fatalf("Alpha %d requests; moved into Beta's folder: %v", len(alpha.Requests), beta.Folders[0].Contains(model.Of(req)))
	}
	if !tabOf(a, req) {
		t.Fatal("the open tab was closed")
	}
	if got := a.collectionToken(model.Of(req)); got != appearance.CollectionToken(1) {
		t.Fatalf("owner stripe %q, want Beta's", got)
	}
	if a.selected != model.Of(req) || !a.expanded[beta.Folders[0]] {
		t.Fatalf("selected %v, folder expanded %v", a.selected, a.expanded[beta.Folders[0]])
	}
	if got, err := store.Load(betaDir); err != nil || got.Folders[0].Requests[len(got.Folders[0].Requests)-1].URL != "http://edited.test" {
		t.Fatalf("Beta on disk: %+v %v", got.Folders, err)
	}
}

func TestDragOntoAnotherCollectionsRootMovesToItsEnd(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Requests[0]
	release(a, model.Of(req), model.Of(&beta.Folder), false)
	if len(alpha.Requests) != 0 || beta.Requests[len(beta.Requests)-1] != req {
		t.Fatalf("Alpha %d requests, Beta's last %v", len(alpha.Requests), beta.Requests[len(beta.Requests)-1])
	}
}

func TestDragOntoARequestOfAnotherCollectionLandsBeforeIt(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Folders[0].Requests[0]
	target := beta.Requests[0]
	release(a, model.Of(req), model.Of(target), false)
	if len(beta.Requests) != 2 || beta.Requests[0] != req || beta.Requests[1] != target {
		t.Fatalf("Beta's requests %v", names(beta.Requests))
	}
}

func names(reqs []*model.Request) []string {
	var out []string
	for _, req := range reqs {
		out = append(out, req.Name)
	}
	return out
}

func TestAltAtReleaseCopies(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Requests[0]
	release(a, model.Of(req), model.Of(beta.Folders[0]), true)
	if len(alpha.Requests) != 1 || alpha.Requests[0] != req {
		t.Fatal("a copy must leave the source")
	}
	copied := beta.Folders[0].Requests[len(beta.Folders[0].Requests)-1]
	if copied == req || copied.Name != req.Name || a.selected != model.Of(copied) {
		t.Fatalf("copy %p of %p named %q, selected %v", copied, req, copied.Name, a.selected)
	}
}

func TestDropInTheSameCollectionRenamesOnAClashWithAnInfoToast(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha").Collection
	req := alpha.Requests[0]
	alpha.Folders[0].AddRequest(model.Request{Name: req.Name})
	release(a, model.Of(req), model.Of(alpha.Folders[0]), false)
	if req.Name != "Alpha call 2" || !alpha.Folders[0].Contains(model.Of(req)) {
		t.Fatalf("name %q", req.Name)
	}
	if text := toastTexts(a); !strings.Contains(text, "Alpha call 2") || a.toasts[len(a.toasts)-1].IsErr {
		t.Fatalf("toasts %q", text)
	}
}

func TestMissingVariablesToastNamesAtMostFive(t *testing.T) {
	tests := []struct {
		name    string
		missing []string
		want    string
	}{
		{"one", []string{"a"}, "a"},
		{"five", []string{"a", "b", "c", "d", "e"}, "a, b, c, d, e"},
		{"seven", []string{"a", "b", "c", "d", "e", "f", "g"}, "a, b, c, d, e and 2 more"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := missingVarsText(test.missing, "Beta")
			if !strings.Contains(got, test.want) || !strings.Contains(got, "Beta") || strings.Contains(got, "{{") {
				t.Fatalf("text %q, want it to hold %q", got, test.want)
			}
			if test.name != "seven" && strings.Contains(got, "more") {
				t.Fatalf("text %q must not be capped", got)
			}
		})
	}
}

func TestMovingAFolderAwayFromItsVariablesToastsWhatIsMissing(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	release(a, model.Of(alpha.Requests[0]), model.Of(&beta.Folder), false) // uses {{host}}, which only Alpha's env defines
	if text := toastTexts(a); !strings.Contains(text, "host") {
		t.Fatalf("toasts %q", text)
	}
}

func TestMoveIntoAnUnsavedCollectionIsAllowed(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha").Collection
	fresh := &model.Collection{Folder: model.Folder{Name: "Fresh"}}
	a.open = collections.Update(a.open, collections.Open{Collection: fresh})
	req := alpha.Requests[0]
	a.openRequest(req)
	a.cur().Req.URL = "http://edited.test"
	release(a, model.Of(req), model.Of(&fresh.Folder), false)
	if len(fresh.Requests) != 1 || fresh.Requests[0] != req || len(alpha.Requests) != 0 {
		t.Fatalf("Fresh %v, Alpha %v", names(fresh.Requests), names(alpha.Requests))
	}
	if !a.cur().Dirty() {
		t.Fatal("the request is not on disk anywhere, so its tab stays unsaved")
	}
}

func TestACrossCollectionMoveWritesTheTargetBeforeTheSource(t *testing.T) {
	a, alphaDir, betaDir := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Requests[0]
	release(a, model.Of(req), model.Of(&beta.Folder), false)
	a.waitWrites()
	gotBeta, err := store.Load(betaDir)
	if err != nil || gotBeta.Requests[len(gotBeta.Requests)-1].Name != req.Name {
		t.Fatalf("Beta on disk: %+v %v", names(gotBeta.Requests), err)
	}
	if gotAlpha, err := store.Load(alphaDir); err != nil || len(gotAlpha.Requests) != 0 {
		t.Fatalf("Alpha on disk: %+v %v", names(gotAlpha.Requests), err)
	}
}

func TestAFailedTargetWriteKeepsTheSourceOnDiskAndSaysSo(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes to a read-only folder")
	}
	a, alphaDir, betaDir := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	if err := os.Chmod(betaDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(betaDir, 0o700) })
	req := alpha.Requests[0]
	release(a, model.Of(req), model.Of(&beta.Folder), false)
	gotAlpha, err := store.Load(alphaDir)
	if err != nil || len(gotAlpha.Requests) != 1 {
		t.Fatalf("Alpha on disk lost the request before the target had it: %v %v", names(gotAlpha.Requests), err)
	}
	if len(a.toasts) == 0 || !a.toasts[len(a.toasts)-1].IsErr {
		t.Fatalf("toasts %q, want an error", toastTexts(a))
	}
}

func TestMovingAFolderResetsAVarsPaneThatEditedIt(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	a.varsTarget = folderVars(alpha.Folders[0])
	release(a, model.Of(alpha.Folders[0]), model.Of(&beta.Folder), false)
	if a.varsTarget != (varsTarget{}) {
		t.Fatalf("varsTarget %+v", a.varsTarget)
	}
}

func TestTheDropRowShowsACopyCueWhileAltIsHeldDuringADrag(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	a.dragNode, a.dropTarget = model.Of(alpha.Requests[0]), model.Of(beta.Folders[0])
	row := collection.Row{Name: "Nested", Folder: true, Dir: beta.Folders[0]}
	for _, test := range []struct {
		alt  bool
		want core.Token
	}{{false, core.AccentLine}, {true, core.OK}} {
		a.ctx.In = core.Input{Held: true, Alt: test.alt}
		if got := a.treeItemOf(row).DropBorder; got != test.want {
			t.Fatalf("alt %v: drop border %q, want %q", test.alt, got, test.want)
		}
	}
}
