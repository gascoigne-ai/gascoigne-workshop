package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"reflect"
	"testing"
)

func TestClickingARequestInTheTreeOpensItsTab(t *testing.T) {
	a := testApp(t)
	if len(a.tabs.Tabs()) != 1 || a.cur().Req.Name != "Search Repos" {
		t.Fatalf("start: %d tabs, current %q", len(a.tabs.Tabs()), a.cur().Req.Name)
	}

	click(a, 100, treeRowY(4)) // Repository Info
	if len(a.tabs.Tabs()) != 2 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("after click: %d tabs, current %q", len(a.tabs.Tabs()), a.cur().Req.Name)
	}

	click(a, 100, treeRowY(4)) // the same request again focuses its tab
	if len(a.tabs.Tabs()) != 2 {
		t.Fatalf("reopening must not add a tab, have %d", len(a.tabs.Tabs()))
	}
}

func TestClickingAFolderTogglesIt(t *testing.T) {
	a := testApp(t)
	before := len(collection.Rows(&a.col().Folder, a.side.Tree().Expanded()))
	click(a, 100, treeRowY(1)) // User folder, collapsed at start
	if got := len(collection.Rows(&a.col().Folder, a.side.Tree().Expanded())); got != before+1 {
		t.Fatalf("rows %d -> %d, want one more (Get User)", before, got)
	}
	click(a, 100, treeRowY(1))
	if got := len(collection.Rows(&a.col().Folder, a.side.Tree().Expanded())); got != before {
		t.Fatalf("second click must collapse again, rows = %d", got)
	}
}

func TestTreeScrollsWhenItDoesNotFit(t *testing.T) {
	a := testApp(t)
	a.h = 200                       // the tree area is about 128px: five rows
	for _, i := range []int{0, 2} { // User and Realtime
		a.side.Tree().Expanded()[a.col().Folders[i]] = true
	}
	a.Step(core.Input{})
	rowsN := len(collection.Rows(&a.col().Folder, a.side.Tree().Expanded()))
	if rowsN*core.RowH <= a.h-headerH-sizes.StatusH {
		t.Fatalf("test setup: %d rows fit, nothing to scroll", rowsN)
	}

	a.Step(core.Input{X: 100, Y: 100, Wheel: -5})
	if a.side.ScrollY() <= 0 {
		t.Fatalf("tree scroll = %d after wheeling over it", a.side.ScrollY())
	}

	// scroll to the end, then click the last row: it must open that request
	a.Step(core.Input{X: 100, Y: 100, Wheel: -50})
	last := collection.Rows(&a.col().Folder, a.side.Tree().Expanded())[rowsN-1]
	click(a, 100, a.h-sizes.StatusH-core.RowH/2)
	if a.cur().Req != last.Req {
		t.Fatalf("clicking the bottom row opened %q, want %q", a.cur().Req.Name, last.Name)
	}
}

func TestRightClickOnATreeRowOpensItsMenu(t *testing.T) {
	a := testApp(t)
	rightClick(a, 100, treeRowY(rowInfo))
	if widget.PopupMenu(a.ctx) == nil {
		t.Fatal("no menu")
	}
	if got, want := menuLabels(a), []string{"Rename", "Duplicate", "Move to...", "Copy to...", "Copy URL", "", "Delete"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("request menu = %q, want %q", got, want)
	}
	a.Step(core.Input{Escape: true})
	rightClick(a, 100, treeRowY(rowRepo))
	if got, want := menuLabels(a), []string{"New request", "New folder", "Import cURL from clipboard", "Import OpenAPI from clipboard", "Import Postman from clipboard", "Export as Postman...", "Variables", "", "Rename", "Duplicate", "Move to...", "Copy to...", "Delete"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("folder menu = %q, want %q", got, want)
	}
	a.Step(core.Input{Escape: true})
	rightClick(a, 100, treeRowY(rowRoot))
	if got, want := menuLabels(a), []string{"New request", "New folder", "Import cURL from clipboard", "Import OpenAPI from clipboard", "Import Postman from clipboard", "Export as Postman...", "Variables", "", "Rename collection...", "Move up", "Move down", "", "New collection...", "Open collection folder...", "", "Close collection", "Delete collection..."}; !reflect.DeepEqual(got, want) {
		t.Fatalf("root menu = %q, want %q (the collection itself is renamed and deleted as a collection, not duplicated)", got, want)
	}
}

func TestRenameThroughTheMenuUpdatesTheTreeAndTheTab(t *testing.T) {
	a := testApp(t)
	info := findReq(a, "Repository Info")
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Rename")
	if a.over.Dialog() == nil || a.over.Dialog().Input.Text != "Repository Info" {
		t.Fatalf("dialog = %+v", a.over.Dialog())
	}
	a.typeText("Repo details")
	a.Step(core.Input{Enter: true})
	if info.Name != "Repo details" {
		t.Fatalf("name = %q", info.Name)
	}
	click(a, 100, treeRowY(rowInfo))
	if a.cur().Req != info || a.cur().Req.Name != "Repo details" {
		t.Fatal("the open tab must show the new name")
	}
}

func TestRenameToATakenNameIsRefusedWithAToast(t *testing.T) {
	a := testApp(t)
	info := findReq(a, "Repository Info")
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Rename")
	a.typeText("Repository Tags")
	a.Step(core.Input{Enter: true})
	if info.Name != "Repository Info" || len(a.over.Toasts.Stack()) != 1 || a.over.Toasts.Stack()[0].Kind != widget.ToastError {
		t.Fatalf("name %q toasts %+v", info.Name, a.over.Toasts.Stack())
	}
}

func TestDuplicateAddsACopyRightAfter(t *testing.T) {
	a := testApp(t)
	before := len(collection.Rows(&a.col().Folder, a.side.Tree().Expanded()))
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Duplicate")
	rows := collection.Rows(&a.col().Folder, a.side.Tree().Expanded())
	if len(rows) != before+1 || rows[rowInfo+1].Name != "Repository Info copy" {
		t.Fatalf("rows = %d, next row %q", len(rows), rows[rowInfo+1].Name)
	}
}

func TestDeleteAsksFirstAndClosesTheRequestsTab(t *testing.T) {
	a := testApp(t)
	click(a, 100, treeRowY(rowInfo)) // opens it in a tab
	info := a.cur().Req
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Delete")
	if a.over.Dialog() == nil || findReq(a, "Repository Info") == nil {
		t.Fatal("deleting must ask first and change nothing yet")
	}
	a.Step(core.Input{Escape: true})
	if findReq(a, "Repository Info") == nil {
		t.Fatal("cancelling keeps the request")
	}

	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Delete")
	a.Step(core.Input{Enter: true})
	if findReq(a, "Repository Info") != nil {
		t.Fatal("the request must be gone")
	}
	for _, tab := range a.openTabs() {
		if tab.Req == info {
			t.Fatal("its tab must be closed")
		}
	}
}

func TestDeletingAFolderClosesTheTabsOfEverythingInIt(t *testing.T) {
	a := testApp(t) // Search Repos (inside Repository) is open
	rightClick(a, 100, treeRowY(rowRepo))
	pickMenu(t, a, "Delete")
	a.Step(core.Input{Enter: true})
	if len(a.col().Folders) != 2 {
		t.Fatalf("folders = %d", len(a.col().Folders))
	}
	if len(a.tabs.Tabs()) != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("the open request was inside: %d tabs, current %q", len(a.tabs.Tabs()), a.cur().Req.Name)
	}
}

func TestCopyURLPutsTheExpandedURLOnTheClipboard(t *testing.T) {
	a := testApp(t)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Copy URL")
	if clip.text != "https://api.github.com/repos/golang/go" {
		t.Fatalf("clipboard = %q", clip.text)
	}
}

func TestNewRequestAndNewFolderFromTheFolderMenu(t *testing.T) {
	a := testApp(t)
	repo := a.col().Folders[1]
	rightClick(a, 100, treeRowY(rowRepo))
	pickMenu(t, a, "New request")
	if len(repo.Requests) != 4 || repo.Requests[3].Name != "New Request" || a.cur().Req != repo.Requests[3] {
		t.Fatalf("requests %v, current %q", requestNames(repo), a.cur().Req.Name)
	}
	rightClick(a, 100, treeRowY(rowRepo))
	pickMenu(t, a, "New folder")
	if len(repo.Folders) != 1 || repo.Folders[0].Name != "New Folder" || !a.side.Tree().Expanded()[repo] {
		t.Fatalf("folders %v expanded %v", folderNames(repo), a.side.Tree().Expanded()[repo])
	}
}

func TestPlusCreatesTheRequestInTheSelectedFolder(t *testing.T) {
	a := testApp(t)
	user := a.col().Folders[0]
	click(a, 100, treeRowY(rowUser)) // selects (and expands) the User folder
	a.Step(combo(ebiten.KeyT, true, false, false))
	if len(user.Requests) != 2 || a.cur().Req != user.Requests[1] {
		t.Fatalf("User now has %v", requestNames(user))
	}
}

func TestDraggingARequestOntoAFolderMovesIt(t *testing.T) {
	a := testApp(t)
	tags := findReq(a, "Repository Tags")
	user := a.col().Folders[0]
	a.Step(core.Input{X: 100, Y: treeRowY(rowTags), Click: true, Held: true})
	a.Step(core.Input{X: 100, Y: treeRowY(rowUser), Held: true})
	a.Step(core.Input{X: 100, Y: treeRowY(rowUser)}) // release over the User folder
	if got, ok := a.col().Folder.Parent(request.Of(tags)); !ok || got != user {
		t.Fatalf("Tags is in %v", got)
	}
	if a.cur().Req != tags {
		t.Fatal("pressing the row opened it, and it stays the open request")
	}
}

func TestDraggingARequestOntoARequestInsertsBeforeIt(t *testing.T) {
	a := testApp(t)
	repo := a.col().Folders[1]
	a.Step(core.Input{X: 100, Y: treeRowY(rowTags), Click: true, Held: true})
	a.Step(core.Input{X: 100, Y: treeRowY(rowSearch), Held: true})
	a.Step(core.Input{X: 100, Y: treeRowY(rowSearch)})
	if got := requestNames(repo); !reflect.DeepEqual(got, []string{"Repository Tags", "Search Repos", "Repository Info"}) {
		t.Fatalf("Repository = %v", got)
	}
}

func TestADropThatIsNotAllowedChangesNothingAndSaysWhy(t *testing.T) {
	a := testApp(t)
	a.side.Tree().Expanded()[a.col().Folders[1]] = true
	// drag the Repository folder onto its own request: that is moving it into itself's parent, fine;
	// drag it onto a request inside itself is the refused case
	a.Step(core.Input{X: 100, Y: treeRowY(rowRepo), Click: true, Held: true})
	a.Step(core.Input{X: 100, Y: treeRowY(rowInfo), Held: true})
	a.Step(core.Input{X: 100, Y: treeRowY(rowInfo)})
	if len(a.col().Folders) != 3 || len(a.col().Folders[1].Requests) != 3 {
		t.Fatal("a refused move changes nothing")
	}
	if len(a.over.Toasts.Stack()) != 1 || a.over.Toasts.Stack()[0].Kind != widget.ToastError {
		t.Fatalf("toasts = %+v", a.over.Toasts.Stack())
	}
}

func TestReleasingOnTheSameRowIsJustAClick(t *testing.T) {
	a := testApp(t)
	a.Step(core.Input{X: 100, Y: treeRowY(rowInfo), Click: true, Held: true})
	a.Step(core.Input{X: 100, Y: treeRowY(rowInfo)})
	if got := requestNames(a.col().Folders[1]); !reflect.DeepEqual(got, []string{"Search Repos", "Repository Info", "Repository Tags"}) {
		t.Fatalf("Repository = %v", got)
	}
}

func TestFilterShowsMatchesByNameAndURL(t *testing.T) {
	a := testApp(t)
	clickDumped(t, a, "control=field", 0) // the filter field in the sidebar header, the first field of the window
	a.typeText("tags")
	rows := collection.Rows(&a.col().Folder, a.side.Tree().Expanded())
	_ = rows
	got := a.side.VisibleRows()
	var names []string
	for _, r := range got {
		names = append(names, r.Name)
	}
	if !reflect.DeepEqual(names, []string{"GitHub REST API", "Repository", "Repository Tags"}) {
		t.Fatalf("filter %q shows %q", "tags", names)
	}

	a.side.Filter().SelectAll()
	a.typeText("search/repos") // part of a URL, not of any name
	names = nil
	for _, r := range a.side.VisibleRows() {
		names = append(names, r.Name)
	}
	if !reflect.DeepEqual(names, []string{"GitHub REST API", "Repository", "Search Repos"}) {
		t.Fatalf("by URL: %q", names)
	}

	a.side.Filter().SelectAll()
	a.typeText("zzz")
	if got := a.side.VisibleRows(); len(got) != 1 {
		t.Fatalf("no match leaves just the root, got %d rows", len(got))
	}
	a.side.Filter().SelectAll()
	a.Step(core.Input{Edits: []core.Edit{core.EditBackspace}})
	if got := a.side.VisibleRows(); len(got) != len(collection.Rows(&a.col().Folder, a.side.Tree().Expanded())) {
		t.Fatal("an empty filter shows the tree as usual")
	}
}

func TestFilteredRowsStillOpenTheRightRequest(t *testing.T) {
	a := testApp(t)
	clickDumped(t, a, "control=field", 0) // the filter field, the first field of the window
	a.typeText("tags")
	click(a, 100, treeRowY(2)) // third visible row: Repository Tags
	if a.cur().Req.Name != "Repository Tags" {
		t.Fatalf("opened %q", a.cur().Req.Name)
	}
}

func TestTreeKeyboardNavigation(t *testing.T) {
	a := testApp(t)
	click(a, 100, treeRowY(rowInfo)) // focuses the tree and selects Repository Info
	a.Step(combo(ebiten.KeyArrowDown, false, false, false))
	if a.side.Tree().Selected() != request.Of(findReq(a, "Repository Tags")) {
		t.Fatalf("Down selects the next row, selected %v", a.side.Tree().Selected())
	}
	a.Step(combo(ebiten.KeyArrowUp, false, false, false))
	a.Step(combo(ebiten.KeyArrowUp, false, false, false))
	if a.side.Tree().Selected() != request.Of(findReq(a, "Search Repos")) {
		t.Fatal("Up moves back")
	}
	a.Step(combo(ebiten.KeyArrowDown, false, false, false))
	a.Step(combo(ebiten.KeyEnter, false, false, false))
	if a.cur().Req.Name != "Repository Info" {
		t.Fatalf("Enter opens the selected request, got %q", a.cur().Req.Name)
	}

	a.Step(combo(ebiten.KeyF2, false, false, false))
	if a.over.Dialog() == nil || a.over.Dialog().Input.Text != "Repository Info" {
		t.Fatal("F2 renames the selection")
	}
	a.Step(core.Input{Escape: true})
	a.Step(combo(ebiten.KeyDelete, false, false, false))
	if a.over.Dialog() == nil {
		t.Fatal("Delete asks before deleting the selection")
	}
}

func TestTreeKeysDoNothingWhileTyping(t *testing.T) {
	a := testApp(t)
	click(a, 100, treeRowY(rowInfo))
	click(a, 1100, urlBarY) // now the URL field has the keyboard
	a.Step(combo(ebiten.KeyDelete, false, false, false))
	a.Step(combo(ebiten.KeyF2, false, false, false))
	if a.over.Dialog() != nil {
		t.Fatal("Delete and F2 belong to the text field, not the tree")
	}
}

// Arrow keys on a tree with no rows (the last collection was closed while
// the tree had the keyboard) read rows[-1] and panicked.
func TestTreeArrowKeysDoNothingWithoutRows(t *testing.T) {
	a := driverApp(t, startOptions{})
	a.Driver().CloseAllCollections()
	for _, k := range []ebiten.Key{ebiten.KeyArrowDown, ebiten.KeyArrowUp, ebiten.KeyEnter, ebiten.KeyF2, ebiten.KeyDelete} {
		a.side.Tree().Focus()
		a.side.Keys(core.Input{Keys: []ebiten.Key{k}}, false, a.sidebarWidth())
	}
}
