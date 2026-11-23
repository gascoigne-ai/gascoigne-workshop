package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// release is a drag of node that ends on target with the pointer released,
// holding Alt when alt is set.
func release(app *Component, node, target request.Node, alt bool) {
	holdDrag(app, node, target)
	app.Step(core.Input{Alt: alt})
}

func tabOf(app *Component, req *request.Request) bool {
	for _, tab := range app.tabs.Tabs() {
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
	a.keys.MarkStale()

	release(a, request.Of(req), request.Of(beta.Folders[0]), false)

	if len(alpha.Requests) != 0 || !beta.Folders[0].Contains(request.Of(req)) {
		t.Fatalf("Alpha %d requests; moved into Beta's folder: %v", len(alpha.Requests), beta.Folders[0].Contains(request.Of(req)))
	}
	if !tabOf(a, req) {
		t.Fatal("the open tab was closed")
	}
	if got := a.collectionToken(request.Of(req)); got != theme.CollectionToken(1) {
		t.Fatalf("owner stripe %q, want Beta's", got)
	}
	if a.side.Tree().Selected() != request.Of(req) || !a.side.Tree().Expanded()[beta.Folders[0]] {
		t.Fatalf("selected %v, folder expanded %v", a.side.Tree().Selected(), a.side.Tree().Expanded()[beta.Folders[0]])
	}
	if got, err := storeDisk.Load(betaDir); err != nil || got.Folders[0].Requests[len(got.Folders[0].Requests)-1].URL != "http://edited.test" {
		t.Fatalf("Beta on disk: %+v %v", got.Folders, err)
	}
}

func TestDragOntoAnotherCollectionsRootMovesToItsEnd(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Requests[0]
	release(a, request.Of(req), request.Of(&beta.Folder), false)
	if len(alpha.Requests) != 0 || beta.Requests[len(beta.Requests)-1] != req {
		t.Fatalf("Alpha %d requests, Beta's last %v", len(alpha.Requests), beta.Requests[len(beta.Requests)-1])
	}
}

func TestDragOntoARequestOfAnotherCollectionLandsBeforeIt(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Folders[0].Requests[0]
	target := beta.Requests[0]
	release(a, request.Of(req), request.Of(target), false)
	if len(beta.Requests) != 2 || beta.Requests[0] != req || beta.Requests[1] != target {
		t.Fatalf("Beta's requests %v", names(beta.Requests))
	}
}

func names(reqs []*request.Request) []string {
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
	release(a, request.Of(req), request.Of(beta.Folders[0]), true)
	if len(alpha.Requests) != 1 || alpha.Requests[0] != req {
		t.Fatal("a copy must leave the source")
	}
	copied := beta.Folders[0].Requests[len(beta.Folders[0].Requests)-1]
	if copied == req || copied.Name != req.Name || a.side.Tree().Selected() != request.Of(copied) {
		t.Fatalf("copy %p of %p named %q, selected %v", copied, req, copied.Name, a.side.Tree().Selected())
	}
}

func TestDropInTheSameCollectionRenamesOnAClashWithAnInfoToast(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha").Collection
	req := alpha.Requests[0]
	alpha.Folders[0].AddRequest(request.Request{Name: req.Name})
	release(a, request.Of(req), request.Of(alpha.Folders[0]), false)
	if req.Name != "Alpha call 2" || !alpha.Folders[0].Contains(request.Of(req)) {
		t.Fatalf("name %q", req.Name)
	}
	if text := toastTexts(a); !strings.Contains(text, "Alpha call 2") || a.over.Toasts.Stack()[len(a.over.Toasts.Stack())-1].Kind == widget.ToastError {
		t.Fatalf("toasts %q", text)
	}
}

func TestMovingAFolderAwayFromItsVariablesToastsWhatIsMissing(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	release(a, request.Of(alpha.Requests[0]), request.Of(&beta.Folder), false) // uses {{host}}, which only Alpha's env defines
	if text := toastTexts(a); !strings.Contains(text, "host") {
		t.Fatalf("toasts %q", text)
	}
}

func TestMoveIntoAnUnsavedCollectionIsAllowed(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha").Collection
	fresh := &request.Collection{Folder: request.Folder{Name: "Fresh"}}
	a.cols.Apply(collections.Open{Collection: fresh})
	req := alpha.Requests[0]
	a.openRequest(req)
	a.cur().Req.URL = "http://edited.test"
	release(a, request.Of(req), request.Of(&fresh.Folder), false)
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
	release(a, request.Of(req), request.Of(&beta.Folder), false)
	a.WaitWrites()
	gotBeta, err := storeDisk.Load(betaDir)
	if err != nil || gotBeta.Requests[len(gotBeta.Requests)-1].Name != req.Name {
		t.Fatalf("Beta on disk: %+v %v", names(gotBeta.Requests), err)
	}
	if gotAlpha, err := storeDisk.Load(alphaDir); err != nil || len(gotAlpha.Requests) != 0 {
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
	release(a, request.Of(req), request.Of(&beta.Folder), false)
	gotAlpha, err := storeDisk.Load(alphaDir)
	if err != nil || len(gotAlpha.Requests) != 1 {
		t.Fatalf("Alpha on disk lost the request before the target had it: %v %v", names(gotAlpha.Requests), err)
	}
	if len(a.over.Toasts.Stack()) == 0 || a.over.Toasts.Stack()[len(a.over.Toasts.Stack())-1].Kind != widget.ToastError {
		t.Fatalf("toasts %q, want an error", toastTexts(a))
	}
}

func TestMovingAFolderResetsAVarsPaneThatEditedIt(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	a.varsTarget = folderVars(alpha.Folders[0])
	release(a, request.Of(alpha.Folders[0]), request.Of(&beta.Folder), false)
	if a.varsTarget != (varsTarget{}) {
		t.Fatalf("varsTarget %+v", a.varsTarget)
	}
}

func TestTheDropRowShowsACopyCueWhileAltIsHeldDuringADrag(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	holdDrag(a, request.Of(alpha.Requests[0]), request.Of(beta.Folders[0]))
	for _, test := range []struct {
		alt  bool
		want core.Token
	}{{false, core.AccentLine}, {true, core.OK}} {
		a.ctx.In = core.Input{Held: true, Alt: test.alt}
		root := a.side.View(a.ctx)
		Layout(a.ctx, root, sidebarW-1, sidebarPinH)
		if got := Dump(root); !strings.Contains(got, "border="+string(test.want)) {
			t.Fatalf("alt %v: no row outlined %q in:\n%s", test.alt, test.want, got)
		}
	}
}

func TestThePickerMovesTheNodeToTheRowPickedWithEnter(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Requests[0]
	a.openTransferPicker(request.Of(req), transfer.Move)
	if a.over.Picker() == nil || !a.over.HoldsKeyboard() {
		t.Fatal("the picker did not open")
	}
	for index, row := range a.over.Picker().Rows() { // Beta's folder "Nested"
		if row.Folder == beta.Folders[0] {
			a.over.Picker().Select(index)
		}
	}
	a.Step(core.Input{Enter: true})
	if a.over.Picker() != nil || len(alpha.Requests) != 0 || !beta.Folders[0].Contains(request.Of(req)) {
		t.Fatalf("picker %v, Alpha %d requests, moved %v", a.over.Picker() != nil, len(alpha.Requests), beta.Folders[0].Contains(request.Of(req)))
	}
}

func TestThePickerCopiesWhenOpenedToCopy(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	a.openTransferPicker(request.Of(alpha.Requests[0]), transfer.Copy)
	a.over.Picker().Step(1) // Alpha's own root: a copy may go to the current folder
	for index, row := range a.over.Picker().Rows() {
		if row.Folder == &beta.Folder {
			a.over.Picker().Select(index)
		}
	}
	a.Step(core.Input{Enter: true})
	if len(alpha.Requests) != 1 || len(beta.Requests) != 2 {
		t.Fatalf("Alpha %d requests, Beta %d", len(alpha.Requests), len(beta.Requests))
	}
}

func TestEscapeClosesAQuestionOverThePickerBeforeThePicker(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha").Collection
	a.openTransferPicker(request.Of(alpha.Requests[0]), transfer.Move)
	a.confirm("Sure?", "", "Yes", func() {})
	a.Step(core.Input{Escape: true})
	if a.over.Dialog() != nil || a.over.Picker() == nil {
		t.Fatalf("dialog %v picker %v after the first Escape", a.over.Dialog() != nil, a.over.Picker() != nil)
	}
	a.Step(core.Input{Escape: true})
	if a.over.Picker() != nil {
		t.Fatal("the second Escape left the picker open")
	}
}

func TestThePickerDoesNotOpenForACollectionRoot(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha").Collection
	a.openTransferPicker(request.Of(&alpha.Folder), transfer.Move)
	a.openTransferPicker(request.Node{}, transfer.Move)
	if a.over.Picker() != nil {
		t.Fatal("a root or nothing cannot be moved")
	}
}

func TestTheRowMenuHasMoveToAndCopyToForRequestsAndFoldersButNotRoots(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha").Collection
	tests := []struct {
		name string
		node request.Node
		want bool
	}{
		{"request", request.Of(alpha.Requests[0]), true},
		{"folder", request.Of(alpha.Folders[0]), true},
		{"root", request.Of(&alpha.Folder), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := collection.Row{Name: test.node.Name(), Req: test.node.Req, Dir: test.node.Dir, Folder: test.node.Dir != nil}
			var labels []string
			for _, item := range a.side.RowMenu(row) {
				labels = append(labels, item.Label)
			}
			joined := strings.Join(labels, "|")
			if got := strings.Contains(joined, "Move to...") && strings.Contains(joined, "Copy to..."); got != test.want {
				t.Fatalf("labels %q", joined)
			}
		})
	}
}

func TestPickingMoveToFromTheMenuOpensTheMovePicker(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Requests[0]
	row := collection.Row{Name: req.Name, Req: req}
	for _, test := range []struct {
		label string
		move  bool
	}{{"Move to...", true}, {"Copy to...", false}} {
		for _, item := range a.side.RowMenu(row) {
			if item.Label == test.label {
				item.Run()
			}
		}
		if a.over.Picker() == nil {
			t.Fatalf("%s opened no picker", test.label)
		}
		for index, each := range a.over.Picker().Rows() {
			if each.Folder == &beta.Folder {
				a.over.Picker().Select(index)
			}
		}
		a.Step(core.Input{Enter: true})
		if moved := len(alpha.Requests) == 0; moved != test.move {
			t.Fatalf("%s: moved %v", test.label, moved)
		}
		if test.move { // put it back for the next round
			beta.Delete(request.Of(req))
			alpha.Attach(request.Of(req), request.Node{})
		}
	}
}

func TestAPickForANodeDeletedWhileThePickerWasOpenSaysSo(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha, beta := entryNamed(t, a, "Alpha").Collection, entryNamed(t, a, "Beta").Collection
	req := alpha.Requests[0]
	a.openTransferPicker(request.Of(req), transfer.Move)
	alpha.Delete(request.Of(req))
	a.over.Picker().Select(len(a.over.Picker().Rows()) - 1)
	a.Step(core.Input{Enter: true})
	if len(a.over.Toasts.Stack()) == 0 || a.over.Toasts.Stack()[len(a.over.Toasts.Stack())-1].Kind != widget.ToastError || beta.Contains(request.Of(req)) {
		t.Fatalf("toasts %q, in Beta %v", toastTexts(a), beta.Contains(request.Of(req)))
	}
}

// A prompt over the destination picker has the keys: Enter reaches neither
// the picker nor, through it, a move.
func TestPickerKeysAreIgnoredWhileAPromptIsOpen(t *testing.T) {
	a, _, _ := twoCollectionsApp(t)
	alpha := entryNamed(t, a, "Alpha").Collection
	a.openTransferPicker(request.Of(alpha.Requests[0]), transfer.Move)
	a.prompt("Name", "x", "OK", func(string) {})
	a.Step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if a.over.Picker() == nil {
		t.Fatal("Enter under a prompt reached the picker")
	}
}
