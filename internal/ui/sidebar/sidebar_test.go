package sidebar

import (
	"image/color"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// fake is every dependency of the sidebar at once, recording what it was asked.
// Its Collections, Nodes and the other sets of methods are separate types
// below, since several interfaces share method names with different arguments.
type fake struct {
	set     collections.Set
	shown   []*request.Request
	nodes   []string // "rename name", "delete name", ...
	opened  [][]widget.MenuItem
	created int
}

type fakeCollections struct{ *fake }
type fakeNodes struct{ *fake }
type fakeRequests struct{ *fake }
type fakeMenus struct{ *fake }
type fakeWorkspace struct{ *fake }
type fakeSent struct{}

func (fake fakeCollections) Set() collections.Set { return fake.set }
func (fake fakeCollections) Focused() *request.Collection {
	if len(fake.set.Entries) == 0 {
		return nil
	}
	return fake.set.Entries[0].Collection
}
func (fake fakeRequests) Show(req *request.Request) { fake.shown = append(fake.shown, req) }
func (fake fakeNodes) Rename(node request.Node) {
	fake.nodes = append(fake.nodes, "rename "+node.Name())
}
func (fake fakeNodes) Delete(node request.Node) {
	fake.nodes = append(fake.nodes, "delete "+node.Name())
}
func (fake fakeNodes) Duplicate(node request.Node) {
	fake.nodes = append(fake.nodes, "duplicate "+node.Name())
}
func (fake fakeNodes) Pick(node request.Node, mode transfer.Mode) {
	fake.nodes = append(fake.nodes, "pick "+node.Name())
}
func (fake fakeMenus) Open(posX, posY int, items []widget.MenuItem) {
	fake.opened = append(fake.opened, items)
}
func (fake fakeWorkspace) NewCollection()               {}
func (fake fakeWorkspace) OpenFolder()                  { fake.created++ }
func (fake fakeWorkspace) Save(col *request.Collection) {}

func (fakeSent) SentStore() history.SentStore         { return nil }
func (fakeSent) Toast(string, severity.Level)         {}
func (fakeSent) SetStatus(string)                     {}
func (fakeSent) OpenRequest(folder, path string) bool { return false }
func (fakeSent) Badge(request.Protocol, string) Node  { return Row() }

// noop covers the interfaces no test here looks into.
type noop struct{}

func (noop) Drop(transfer.Transfer)                   {}
func (noop) NewRequest(*request.Folder)               {}
func (noop) NewFolder(*request.Folder)                {}
func (noop) Import(*request.Folder, logic.ImportKind) {}
func (noop) Export(*request.Folder)                   {}
func (noop) CopyURL(*request.Request)                 {}
func (noop) EditVars(*request.Folder)                 {}
func (noop) Rename(*request.Collection)               {}
func (noop) Move(*request.Collection, int)            {}
func (noop) Close(*request.Collection)                {}
func (noop) Delete(*request.Collection)               {}

// sampleCollection is "Shop": a folder "orders" holding "list", and a request "home".
func sampleCollection() *request.Collection {
	list := &request.Request{Name: "list", Protocol: "rest", Method: "GET"}
	home := &request.Request{Name: "home", Protocol: "rest", Method: "POST"}
	return &request.Collection{Folder: request.Folder{
		Name:     "Shop",
		Folders:  []*request.Folder{{Name: "orders", Requests: []*request.Request{list}}},
		Requests: []*request.Request{home},
	}}
}

func newSidebar(t *testing.T, fake *fake) (*core.Ctx, *Component) {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	var none noop
	side := New(Deps{
		Collections: fakeCollections{fake}, Requests: fakeRequests{fake}, Nodes: fakeNodes{fake},
		Drops: none, Folders: none, Extras: none, Collection: none,
		Workspace: fakeWorkspace{fake}, Menus: fakeMenus{fake}, Sent: fakeSent{},
	})
	return ctx, side
}

// withShop is a sidebar over one open collection, its root and folder open.
func withShop(t *testing.T) (*core.Ctx, *Component, *fake, *request.Collection) {
	t.Helper()
	col := sampleCollection()
	fake := &fake{set: collections.Update(collections.Set{}, collections.Open{Collection: col})}
	ctx, side := newSidebar(t, fake)
	side.Tree().ExpandAll(&col.Folder)
	return ctx, side, fake, col
}

const sidebarW, sidebarH = 260, 400

func laidOut(ctx *core.Ctx, side *Component) Node {
	root := side.View(ctx)
	Layout(ctx, root, sidebarW, sidebarH)
	return root
}

// find is the first element whose own Dump line has part.
func find(node Node, part string) Node {
	line, _, _ := strings.Cut(Dump(node), "\n")
	if strings.Contains(line, part) {
		return node
	}
	for _, child := range node.Children() {
		if found := find(child, part); found != nil {
			return found
		}
	}
	return nil
}

func pointerOn(t *testing.T, ctx *core.Ctx, root Node, part string, input core.Input) {
	t.Helper()
	node := find(root, part)
	if node == nil {
		t.Fatalf("no element with %s in:\n%s", part, Dump(root))
	}
	rect := node.Rect()
	input.X, input.Y = rect.X+rect.W/2, rect.Y+rect.H/2
	ctx.In = input
	Update(ctx, root)
}

func TestTheSidebarListsTheOpenCollectionsRows(t *testing.T) {
	ctx, side, _, _ := withShop(t)
	dump := Dump(laidOut(ctx, side))
	for _, want := range []string{"Shop", "orders", "list", "home", "GET", "POST", "Collections"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %q:\n%s", want, dump)
		}
	}
}

func TestNoCollectionShowsTheOpenAndNewButtons(t *testing.T) {
	ctx, side := newSidebar(t, &fake{})
	dump := Dump(laidOut(ctx, side))
	for _, want := range []string{"No collection is open", "Open collection folder...", "New collection..."} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %q:\n%s", want, dump)
		}
	}
}

func TestClickingARequestRowShowsIt(t *testing.T) {
	ctx, side, fake, col := withShop(t)
	pointerOn(t, ctx, laidOut(ctx, side), `text="home"`, core.Input{Click: true, Held: true})
	if len(fake.shown) != 1 || fake.shown[0] != col.Folder.Requests[0] {
		t.Fatalf("shown %v, want the home request", fake.shown)
	}
	if side.Tree().Selected() != (request.Node{Req: col.Folder.Requests[0]}) {
		t.Errorf("selected %v", side.Tree().Selected())
	}
}

func TestRightClickOpensTheRowMenu(t *testing.T) {
	tests := []struct {
		name string
		row  string
		want []string
	}{
		{"request", `text="home"`, []string{"Rename", "Duplicate", "Move to...", "Copy to...", "Copy URL", "", "Delete"}},
		{"folder", `text="orders"`, []string{"New request", "New folder", "Import cURL from clipboard", "Import OpenAPI from clipboard",
			"Import Postman from clipboard", "Export as Postman...", "Variables", "", "Rename", "Duplicate", "Move to...", "Copy to...", "Delete"}},
		{"collection root", `text="Shop"`, []string{"New request", "New folder", "Import cURL from clipboard", "Import OpenAPI from clipboard",
			"Import Postman from clipboard", "Export as Postman...", "Variables", "", "Rename collection...", "Move up", "Move down", "",
			"New collection...", "Open collection folder...", "", "Close collection", "Delete collection..."}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, side, fake, _ := withShop(t)
			pointerOn(t, ctx, laidOut(ctx, side), test.row, core.Input{RightClick: true})
			if len(fake.opened) != 1 {
				t.Fatalf("%d menus opened, want 1", len(fake.opened))
			}
			var got []string
			for _, item := range fake.opened[0] {
				got = append(got, item.Label)
			}
			if strings.Join(got, "|") != strings.Join(test.want, "|") {
				t.Errorf("menu %q, want %q", got, test.want)
			}
		})
	}
}

func TestKeysDeleteAndF2CallNodes(t *testing.T) {
	_, side, fake, col := withShop(t)
	side.Tree().Select(request.Node{Req: col.Folder.Requests[0]})
	side.Tree().Focus()
	side.Keys(core.Input{Keys: []ebiten.Key{ebiten.KeyF2, ebiten.KeyDelete}}, false, sidebarW)
	if got := strings.Join(fake.nodes, ","); got != "rename home,delete home" {
		t.Errorf("node calls %q", got)
	}
}

func TestKeysDoNothingWhileTextFocused(t *testing.T) {
	_, side, fake, col := withShop(t)
	side.Tree().Select(request.Node{Req: col.Folder.Requests[0]})
	side.Tree().Focus()
	if side.Keys(core.Input{Keys: []ebiten.Key{ebiten.KeyDelete}}, true, sidebarW) {
		t.Error("Keys claimed a key while a field has the keyboard")
	}
	if len(fake.nodes) != 0 || side.Tree().Focused() {
		t.Errorf("node calls %q, focused %v: typing must take the keyboard from the tree", fake.nodes, side.Tree().Focused())
	}
}

func TestEnterTogglesAFolderAndShowsARequest(t *testing.T) {
	_, side, fake, col := withShop(t)
	orders := col.Folder.Folders[0]
	side.Tree().Select(request.Node{Dir: orders})
	side.Tree().Focus()
	side.Keys(core.Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, false, sidebarW)
	if side.Tree().Expanded()[orders] {
		t.Error("Enter on an open folder left it open")
	}
	side.Tree().Select(request.Node{Req: col.Folder.Requests[0]})
	side.Keys(core.Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, false, sidebarW)
	if len(fake.shown) != 1 {
		t.Errorf("shown %d requests, want 1", len(fake.shown))
	}
}

func TestAClickBesideTheSidebarTakesTheKeyboard(t *testing.T) {
	_, side, _, _ := withShop(t)
	side.Tree().Focus()
	side.Keys(core.Input{Click: true, X: sidebarW + 50}, false, sidebarW)
	if side.Tree().Focused() {
		t.Error("a click in the editor left the keyboard in the tree")
	}
}

// The filter is the one text field the sidebar owns; the window asks
// whether it has the keyboard before a shortcut may take a key.
func TestTypingIsTheFilter(t *testing.T) {
	_, side := newSidebar(t, &fake{})
	if side.Typing() {
		t.Fatal("typing with the filter unfocused")
	}
	side.Filter().Focused = true
	if !side.Typing() {
		t.Fatal("not typing with the filter focused")
	}
}
