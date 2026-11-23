package tabstrip

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Widths the strip is pinned at: the main column of a 1280px window with
// the 260px sidebar, and the narrowest main column the dividers allow.
const (
	wideW   = 1020
	narrowW = 420
)

// stripY is the middle of the strip, from its top.
const stripY = sizes.TabsH / 2

// fake is every dependency of the strip at once: a real tab list, and a
// record of what the strip asked of the host.
type fake struct {
	set      *tabs.Set
	dirty    map[*tabs.Tab]bool
	menus    [][3]int // posX, posY, index of each tab menu asked for
	newCount int      // requests the "+" made
}

func (fake *fake) Tabs() []*tabs.Tab        { return fake.set.Tabs() }
func (fake *fake) Active() int              { return fake.set.Active() }
func (fake *fake) Update(msg tabs.Msg)      { fake.set.Apply(msg) }
func (fake *fake) Dirty(tab *tabs.Tab) bool { return fake.dirty[tab] }
func (fake *fake) Stripe(*request.Request) core.Token {
	return theme.CollectionToken(0)
}
func (fake *fake) OpenTabMenu(posX, posY, index int) {
	fake.menus = append(fake.menus, [3]int{posX, posY, index})
}

// NewRequest opens "New Request", "New Request 2", ... as the application does.
func (fake *fake) NewRequest() {
	fake.newCount++
	name := "New Request"
	if fake.newCount > 1 {
		name = fmt.Sprintf("New Request %d", fake.newCount)
	}
	fake.set.Open(&request.Request{Name: name, Protocol: request.ProtocolREST, Method: "GET"})
}

func (fake *fake) Hint(label, action string) string {
	keys := map[string]string{"close-tab": "Ctrl+W", "new-request": "Ctrl+T"}
	return label + " (" + keys[action] + ")"
}

// world is a strip over a fake, driven frame by frame like the window does.
type world struct {
	t     *testing.T
	ctx   *core.Ctx
	fake  *fake
	strip *Component
	width int
}

func rest(name, method string) *request.Request {
	return &request.Request{Name: name, Protocol: request.ProtocolREST, Method: method}
}

// newWorld is a strip of width over the requests, the one at active shown.
func newWorld(t *testing.T, width int, active int, reqs ...*request.Request) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet(), dirty: map[*tabs.Tab]bool{}}
	for _, req := range reqs {
		fake.set.Open(req)
	}
	fake.set.Apply(tabs.Activate{Index: active})
	strip := New(Deps{Tabs: fake, Marks: fake, Menus: fake, Requests: fake, Hints: fake})
	return &world{t: t, ctx: ctx, fake: fake, strip: strip, width: width}
}

// twoTabs is "Search Repos" and "Repository Info", the second one active.
func twoTabs(t *testing.T) *world {
	return newWorld(t, wideW, 1, rest("Search Repos", "GET"), rest("Repository Info", "GET"))
}

// frame runs one frame: the strip is built, laid out and given in.
func (world *world) frame(in core.Input) {
	world.ctx.Tick++
	world.ctx.In = in
	world.strip.BeginFrame(in)
	root := world.strip.View(world.ctx)
	Layout(world.ctx, root, world.width, sizes.TabsH)
	Update(world.ctx, root)
}

func (world *world) click(posX, posY int) {
	world.frame(core.Input{X: posX, Y: posY, Click: true, Held: true})
	world.frame(core.Input{X: posX, Y: posY})
}

// tabX is the left edge and width of tab index, from the layout of the
// view: 14px before the badge slot, 8px gaps, the name, a fixed slot for
// the unsaved dot and a 24px close button. Nothing but the name changes a
// tab's width.
func (world *world) tabX(index int) (left, width int) {
	x := 0
	for i, tab := range world.fake.Tabs() {
		wd := 14 + sizes.TabBadgeW + 8 + world.ctx.Width(world.ctx.F.UI, tab.Req.Name) + 8 + sizes.TabDirtyW + 24
		if i == index {
			return x, wd
		}
		x += wd
	}
	return x, 0
}

// names are the names of the open tabs, in order.
func (world *world) names() []string {
	var names []string
	for _, tab := range world.fake.Tabs() {
		names = append(names, tab.Req.Name)
	}
	return names
}

// stripStates are the looks of the strip: one tab, several with the
// badges of each protocol, a long dirty name, and more tabs than fit.
var stripStates = []struct {
	name  string
	setUp func(t *testing.T) *world
}{
	{"one", func(t *testing.T) *world { return newWorld(t, wideW, 0, rest("Search Repos", "GET")) }},
	{"several", func(t *testing.T) *world {
		return newWorld(t, wideW, 1, rest("Search Repos", "GET"), rest("Repository Info", "GET"),
			&request.Request{Name: "Echo", Protocol: request.ProtocolWebSocket},
			&request.Request{Name: "List Tools", Protocol: request.ProtocolMCP})
	}},
	{"dirty-long", func(t *testing.T) *world {
		world := newWorld(t, wideW, 0, rest("Search repositories by stars, newest first, then by name", "DELETE"),
			&request.Request{Name: "Echo", Protocol: request.ProtocolWebSocket})
		world.fake.dirty[world.fake.Tabs()[0]] = true
		return world
	}},
	{"overflow", func(t *testing.T) *world {
		world := newWorld(t, wideW, 0, rest("Search Repos", "GET"))
		for range 14 {
			world.fake.NewRequest()
		}
		return world
	}},
}

// The strip is pinned as text (element kinds, options, sizes, colors) at a
// wide and a narrow column, so a change of the template shows as a diff.
func TestTheStripLaysOutAsPinned(t *testing.T) {
	sizesWide := []struct {
		name  string
		width int
	}{
		{"wide", wideW},
		{"narrow", narrowW},
	}
	for _, state := range stripStates {
		for _, size := range sizesWide {
			name := "requesttabs-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := state.setUp(t)
				root := world.strip.View(world.ctx)
				Layout(world.ctx, root, size.width, sizes.TabsH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// The "+" is at the right end of the strip, its middle 20px from the edge.
const plusX = wideW - 20

// Clicking the "+" opens a new request in a new, active tab.
func TestClickingThePlusOpensANewRequestTab(t *testing.T) {
	world := newWorld(t, wideW, 0, rest("Search Repos", "GET"))
	world.click(plusX, stripY)
	if len(world.fake.Tabs()) != 2 || world.fake.Active() != 1 {
		t.Fatalf("%d tabs, active %d; want 2 tabs with the new one active", len(world.fake.Tabs()), world.fake.Active())
	}
}

// Right-clicking a tab asks the host for its menu where the pointer is.
func TestRightClickingATabAsksForItsMenuAtThePointer(t *testing.T) {
	world := twoTabs(t)
	left, _ := world.tabX(1)
	world.frame(core.Input{X: left + 20, Y: stripY, RightClick: true})
	if want := [][3]int{{left + 20, stripY, 1}}; len(world.fake.menus) != 1 || world.fake.menus[0] != want[0] {
		t.Fatalf("menus asked for %v, want %v", world.fake.menus, want)
	}
}

func TestCloseButtonClosesOnlyThatTab(t *testing.T) {
	world := twoTabs(t) // active: the second
	left, width := world.tabX(0)
	world.click(left+width-12, stripY) // the cross of the first tab
	if got := world.names(); len(got) != 1 || got[0] != "Repository Info" {
		t.Fatalf("tabs %q", got)
	}
}

func TestClickingTheLabelSelectsAndDoesNotClose(t *testing.T) {
	world := twoTabs(t)
	left, _ := world.tabX(0)
	world.click(left+30, stripY)
	if len(world.fake.Tabs()) != 2 || world.fake.Active() != 0 {
		t.Fatalf("%d tabs, active %d", len(world.fake.Tabs()), world.fake.Active())
	}
}

func TestMiddleButtonDragReordersTabsAndDoesNotCloseThem(t *testing.T) {
	world := twoTabs(t)
	first := world.fake.Tabs()[0]
	l0, _ := world.tabX(0)
	l1, w1 := world.tabX(1)
	world.frame(core.Input{X: l0 + 20, Y: stripY, MiddleClick: true, MiddleHeld: true}) // press the first tab
	if len(world.fake.Tabs()) != 2 {
		t.Fatal("a middle press must not close the tab any more")
	}
	world.frame(core.Input{X: l1 + w1/2, Y: stripY, MiddleHeld: true}) // drag over the second
	if world.fake.Tabs()[1] != first {
		t.Fatal("the dragged tab must move to the second place")
	}
	world.frame(core.Input{X: l1 + w1/2, Y: stripY}) // release
	if world.fake.set.Drag() != -1 {
		t.Fatalf("drag = %d after the release", world.fake.set.Drag())
	}
	world.frame(core.Input{X: l0 + 20, Y: stripY, MiddleHeld: true}) // held again, but not pressed on a tab
	if world.fake.Tabs()[1] != first {
		t.Fatal("after the release, moving must not drag")
	}
}

func TestMiddleClickWithoutMovingLeavesTheTabsAlone(t *testing.T) {
	world := twoTabs(t)
	left, width := world.tabX(1)
	world.frame(core.Input{X: left + width/2, Y: stripY, MiddleClick: true, MiddleHeld: true})
	world.frame(core.Input{X: left + width/2, Y: stripY})
	if got := world.names(); len(got) != 2 || world.fake.set.Current().Req.Name != "Repository Info" {
		t.Fatalf("tabs %q, current %q", got, world.fake.set.Current().Req.Name)
	}
}

func TestTabWidthDoesNotChangeWhenTheRequestChanges(t *testing.T) {
	world := twoTabs(t)
	left, width := world.tabX(0)
	tab := world.fake.Tabs()[0]
	tab.Req.Method = "DELETE"    // a longer badge
	world.fake.dirty[tab] = true // and an unsaved dot
	world.frame(core.Input{})

	// The cross sits where it did before the edits; if the tab had grown it
	// would be somewhere else and the click would only select the tab.
	world.click(left+width-12, stripY)
	if got := world.names(); len(got) != 1 || got[0] != "Repository Info" {
		t.Fatalf("tabs %q: the tab changed size when its request changed", got)
	}
}

func TestTheTabSlotsFitTheirContentAtEveryTextSize(t *testing.T) {
	for size := settings.MinFontSize; size <= settings.MaxFontSize; size++ {
		ctx, err := core.NewCtx(theme.ThemeFor("dark", theme.FontSizes{UI: size, Small: max(size-2, 9), Code: size - 1}))
		if err != nil {
			t.Fatal(err)
		}
		for _, badge := range sizes.BadgeLabels {
			if got := ctx.Width(ctx.F.Small, badge); got-sizes.BadgeOverhang > sizes.BadgeSlot(ctx, sizes.TabBadgeW)+sizes.BadgeOverhang {
				t.Errorf("%d: %q is %dpx wide, the tab badge slot is %dpx", size, badge, got, sizes.BadgeSlot(ctx, sizes.TabBadgeW))
			}
			if got := ctx.Width(ctx.F.Small, badge); got-sizes.BadgeOverhang > sizes.BadgeSlot(ctx, sizes.BadgeW)+sizes.BadgeOverhang {
				t.Errorf("%d: %q is %dpx wide, the tree badge slot is %dpx", size, badge, got, sizes.BadgeSlot(ctx, sizes.BadgeW))
			}
		}
		if size == settings.DefaultFontSize && (sizes.BadgeSlot(ctx, sizes.TabBadgeW) != sizes.TabBadgeW || sizes.BadgeSlot(ctx, sizes.BadgeW) != sizes.BadgeW) {
			t.Errorf("at the default size the slots must stay %d and %d", sizes.TabBadgeW, sizes.BadgeW)
		}
		if got := ctx.Width(ctx.F.UI, DirtyMark); got > sizes.TabDirtyW {
			t.Errorf("%d: the dirty mark is %dpx wide, the slot is %dpx", size, got, sizes.TabDirtyW)
		}
	}
}

func TestDraggingATabReordersIt(t *testing.T) {
	world := twoTabs(t)
	first := world.fake.Tabs()[0]
	l0, w0 := world.tabX(0)
	l1, w1 := world.tabX(1)

	world.frame(core.Input{X: l0 + 20, Y: stripY, Click: true, Held: true}) // press the first tab
	world.frame(core.Input{X: l1 + w1/2, Y: stripY, Held: true})            // drag over the second
	if world.fake.Tabs()[1] != first {
		t.Fatalf("the dragged tab must move to the second place")
	}
	world.frame(core.Input{X: l1 + w1/2, Y: stripY}) // release
	world.frame(core.Input{X: l0 + w0/2, Y: stripY, Held: true})
	if world.fake.Tabs()[1] != first {
		t.Fatal("after release, moving over another tab must not drag")
	}
	if world.fake.set.Drag() != -1 {
		t.Fatalf("drag = %d after release", world.fake.set.Drag())
	}
}

func TestDraggingKeepsTheActiveTabActive(t *testing.T) {
	world := twoTabs(t) // second tab active
	active := world.fake.set.Current()
	l0, _ := world.tabX(0)
	l1, w1 := world.tabX(1)
	world.frame(core.Input{X: l0 + 20, Y: stripY, Click: true, Held: true})
	world.frame(core.Input{X: l1 + w1/2, Y: stripY, Held: true})
	if world.fake.set.Current() == active {
		t.Fatal("pressing the first tab activates it, so it is the active one")
	}
	if world.fake.set.Current() != world.fake.Tabs()[1] {
		t.Fatal("the active tab must travel with the drag")
	}
}

// manyTabs is a strip with more tabs than fit, the last one active and
// scrolled into view.
func manyTabs(t *testing.T) *world {
	world := newWorld(t, wideW, 0, rest("Search Repos", "GET"))
	for range 14 {
		world.fake.NewRequest()
	}
	world.frame(core.Input{})
	return world
}

func TestManyTabsScrollTheStripAndKeepTheActiveTabInView(t *testing.T) {
	world := manyTabs(t)
	if world.strip.ScrollX() <= 0 {
		t.Fatalf("the new, active tab is past the right edge: X = %d", world.strip.ScrollX())
	}
	x := world.strip.ScrollX()
	world.frame(core.Input{X: 340, Y: stripY, WheelX: 1}) // two-finger swipe right: back toward the first tabs
	if world.strip.ScrollX() >= x {
		t.Fatalf("a sideways swipe must scroll the strip back: %d -> %d", x, world.strip.ScrollX())
	}
	x = world.strip.ScrollX()
	world.frame(core.Input{X: 340, Y: stripY, Wheel: -1}) // the mouse wheel scrolls it too
	if world.strip.ScrollX() <= x {
		t.Fatalf("the wheel must scroll the strip forward: %d -> %d", x, world.strip.ScrollX())
	}
	x = world.strip.ScrollX()
	world.click(4, stripY) // the left arrow, at the strip's left edge
	if world.strip.ScrollX() >= x {
		t.Fatalf("clicking the left arrow must scroll back: %d -> %d", x, world.strip.ScrollX())
	}
	if len(world.fake.Tabs()) != 15 || world.fake.Active() != 14 {
		t.Fatalf("an arrow click must not select or close a tab: %d tabs, active %d", len(world.fake.Tabs()), world.fake.Active())
	}
}

func TestLeftDragOnAnOverflowingTabStripPansItInsteadOfReorderingTabs(t *testing.T) {
	world := manyTabs(t)
	order := append([]*tabs.Tab(nil), world.fake.Tabs()...)
	x0 := world.strip.ScrollX() // the active (last) tab was revealed: the strip is at its right end
	world.frame(core.Input{X: 340, Y: stripY, Click: true, Held: true})
	world.frame(core.Input{X: 440, Y: stripY, Held: true}) // drag right: earlier tabs come into view
	world.frame(core.Input{X: 540, Y: stripY, Held: true})
	if want := x0 - 200; world.strip.ScrollX() != want {
		t.Fatalf("X = %d, want %d: the strip follows the pointer", world.strip.ScrollX(), want)
	}
	for i := range order {
		if world.fake.Tabs()[i] != order[i] {
			t.Fatalf("tab %d moved: a left drag on an overflowing strip must not reorder", i)
		}
	}
	world.frame(core.Input{X: 540, Y: stripY}) // release
	if world.fake.set.Drag() != -1 {
		t.Fatalf("drag = %d", world.fake.set.Drag())
	}
}
