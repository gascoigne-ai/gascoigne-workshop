package app

import (
	"fmt"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The frame builds the view once: Draw paints the tree endFrame built and
// laid out, unless a handler fired or a post-view step changed state. These
// tests pin that Draw never shows less than a fresh build would.

// firstDifference names the first line two dumps disagree on.
func firstDifference(got, want string) string {
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < max(len(gotLines), len(wantLines)); i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			return fmt.Sprintf("line %d\n  drawn: %s\n  fresh: %s", i+1, g, w)
		}
	}
	return "no difference"
}

// assertDrawnTreeIsFresh compares the tree Draw would paint with a fresh
// build laid out the same way.
func assertDrawnTreeIsFresh(t *testing.T, app *App, label string) {
	t.Helper()
	drawn := view.Dump(app.frameTree())
	if fresh := layoutDump(app); drawn != fresh {
		t.Fatalf("%s: Draw would paint a stale tree, %s", label, firstDifference(drawn, fresh))
	}
}

// A click that changes a setting shows in the tree Draw paints.
func TestDrawShowsTheSettingAClickChanged(t *testing.T) {
	a := testApp(t)
	a.menu = true
	a.Step(core.Input{X: 5000, Y: 5000})
	before := view.Dump(a.frameTree())
	x, y := textCenter(t, a, "Light")
	click(a, x, y)
	after := view.Dump(a.frameTree())
	if after == before {
		t.Fatal("the theme chips look the same after clicking Light")
	}
	assertDrawnTreeIsFresh(t, a, "after the click")
}

// scriptedFrames runs the frames of a session and checks the drawn tree
// after each one. If a step fails here, the signal that marks the tree
// stale is wrong: fix the signal, never this test.
type scriptedFrames struct {
	t *testing.T
	a *App
}

func (s scriptedFrames) step(label string, in core.Input) {
	s.t.Helper()
	s.a.Step(in)
	assertDrawnTreeIsFresh(s.t, s.a, label)
}

func (s scriptedFrames) idle(label string, frames int) {
	s.t.Helper()
	for i := 0; i < frames; i++ {
		s.step(fmt.Sprintf("%s (idle %d)", label, i), core.Input{X: s.a.lastIn.X, Y: s.a.lastIn.Y})
	}
}

func (s scriptedFrames) click(label string, x, y int) {
	s.t.Helper()
	s.step(label+" (press)", core.Input{X: x, Y: y, Click: true, Held: true})
	s.step(label+" (release)", core.Input{X: x, Y: y})
}

func (s scriptedFrames) clickText(text string, nth int) {
	s.t.Helper()
	lines := dumpedLines(s.a, fmt.Sprintf("text=%q", text))
	if nth >= len(lines) {
		s.t.Fatalf("no element with text %q (#%d)", text, nth)
	}
	x, y := lineCenter(s.t, lines[nth])
	s.click("click "+text, x, y)
}

func (s scriptedFrames) typeText(label, text string) {
	s.t.Helper()
	for _, char := range text {
		s.step(label, core.Input{X: s.a.lastIn.X, Y: s.a.lastIn.Y, Typed: []rune{char}})
	}
}

func TestTheDrawnTreeMatchesAFreshBuildAcrossASession(t *testing.T) {
	a := testApp(t)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return clock }
	s := scriptedFrames{t, a}

	s.idle("start", 3)
	s.clickText("Repository Info", 0) // a tab switch
	s.clickText("Search Repos", 0)

	s.step("pick up a tree row", core.Input{X: 130, Y: 142, Click: true, Held: true})
	s.step("move the tree row", core.Input{X: 130, Y: 190, Held: true})
	for i := 0; i < 35; i++ { // rests over a folder long enough to open it (dragHover)
		s.step("drag over a folder", core.Input{X: 130, Y: 226, Held: true})
	}
	s.step("drop on the folder", core.Input{X: 130, Y: 226})

	s.clickText("Bulk edit", 0)
	s.typeText("type in the bulk text", "x")
	s.clickText("Headers", 0) // another table shows: the bulk text becomes rows (settleBulk)
	s.clickText("Params", 0)

	nameCell := dumpedLines(a, `text="NAME"`)
	if len(nameCell) == 0 {
		t.Fatal("no params table header")
	}
	dividerX, dividerY := lineCenter(t, nameCell[0])
	dividerX += 94/2 + 8 // the right edge of the NAME cell, where its column handle is
	s.step("press a column divider", core.Input{X: dividerX, Y: dividerY, Click: true, Held: true})
	for i := 1; i <= 4; i++ {
		s.step("drag a column divider", core.Input{X: dividerX + 15*i, Y: dividerY, Held: true})
	}
	s.step("drop the column divider", core.Input{X: dividerX + 60, Y: dividerY})

	drag := func(name string, x, y int) {
		s.step("press "+name, core.Input{X: x, Y: y, Click: true, Held: true})
		for i := 1; i <= 3; i++ {
			s.step("drag "+name, core.Input{X: x + 20*i, Y: y, Held: true})
		}
		s.step("drop "+name, core.Input{X: x + 60, Y: y})
	}
	drag("the sidebar divider", 261, 400)
	responseTab, _ := lineCenter(t, dumpedLines(a, `text="Response"`)[0])
	drag("the pane divider", responseTab-32-12, 400) // the label is 63 wide; its pane starts 12 before it
	if a.sidebarSplit.Size == 0 || a.paneSplit.Size == 0 {
		t.Fatalf("the dividers did not move: sidebar %v, panes %v", a.sidebarSplit.Size, a.paneSplit.Size)
	}
	s.step("wheel over the tree", core.Input{X: 130, Y: 150, Wheel: -3})
	s.step("wheel over the request", core.Input{X: 500, Y: 300, Wheel: -3})

	s.click("focus the tree filter", 180, 22)
	s.typeText("type in the tree filter", "rep")
	s.step("right click the filter", core.Input{X: 180, Y: 22, RightClick: true}) // a menu opens
	s.idle("menu open", 2)
	s.step("escape closes the menu", core.Input{X: 180, Y: 22, Escape: true})
	s.click("clear the focus", 900, 600)

	s.click("start a cell edit", 500, 270)
	s.typeText("type in the cell", "ab")
	s.click("click elsewhere ends the edit", 900, 600) // commitEdit runs after the view

	a.toast("Saved", severity.Info)
	s.idle("toast shown", 2)
	clock = clock.Add(10 * time.Second)
	s.idle("toast expired", 2)

	tab := a.cur()
	pointTabAt(tab, jsonServer(t).URL)
	tab.Resp = nil
	a.send(tab)
	for i := 0; tab.Resp == nil && i < 400; i++ { // the response arrives through the bus
		s.step("waiting for the response", core.Input{})
		time.Sleep(2 * time.Millisecond)
	}
	if tab.Resp == nil {
		t.Fatal("no response")
	}
	s.idle("response shown", 2)

	a.menu = true
	s.idle("settings dialog opens", 2)
	dialog := settingsRect(t, a)
	grabX, grabY := dialog.X+dialog.W/2, dialog.Y+20
	s.step("press the dialog title", core.Input{X: grabX, Y: grabY, Click: true, Held: true})
	for i := 1; i <= 8; i++ {
		s.step("drag the dialog", core.Input{X: grabX + 10*i, Y: grabY + 6*i, Held: true})
	}
	s.step("drop the dialog", core.Input{X: grabX + 80, Y: grabY + 48})
	edge := settingsRect(t, a)
	s.step("press the dialog edge", core.Input{X: edge.X + edge.W - 3, Y: edge.Y + edge.H/2, Click: true, Held: true})
	for i := 1; i <= 4; i++ {
		s.step("resize the dialog", core.Input{X: edge.X + edge.W - 3 + 15*i, Y: edge.Y + edge.H/2, Held: true})
	}
	s.step("release the dialog edge", core.Input{X: edge.X + edge.W + 60, Y: edge.Y + edge.H/2})
	s.step("scroll the settings to the top", core.Input{X: edge.X + 200, Y: edge.Y + 200, Wheel: 50}) // pressing the edge may have landed on the scrollbar
	s.clickText("Light", 0)
	s.clickText("+", 0)
	s.clickText("Dark", 0)
	s.step("wheel over the settings", core.Input{X: edge.X + 200, Y: edge.Y + 200, Wheel: -5})
	s.click("focus the settings search", 400, 184)
	s.typeText("type in the settings search", "font")
	s.step("escape", core.Input{Escape: true})

	a.setSize(1000, 700, 1) // the window is resized between the step and the draw
	assertDrawnTreeIsFresh(t, a, "right after the resize")
	s.idle("after the resize", 2)
	a.setSize(1000, 700, 1.25)
	assertDrawnTreeIsFresh(t, a, "right after the scale change")
	s.idle("after the scale change", 2)
}

// An idle or a dragging frame builds the tree once; a frame in which a
// handler fired builds a second one for Draw.
func TestDrawReusesTheTreeWhenNothingFired(t *testing.T) {
	a := testApp(t)
	a.menu = true
	a.Step(core.Input{X: 5000, Y: 5000})

	builds := func(in core.Input) int {
		before := a.builds
		a.Step(in)
		a.frameTree()
		return a.builds - before
	}
	if got := builds(core.Input{X: 5000, Y: 5000}); got != 1 {
		t.Fatalf("an idle frame builds %d trees, want 1", got)
	}
	if got := builds(core.Input{X: 700, Y: 300, Held: true}); got != 1 {
		t.Fatalf("a drag frame builds %d trees, want 1", got)
	}
	x, y := textCenter(t, a, "Light")
	if got := builds(core.Input{X: x, Y: y, Click: true, Held: true}); got != 2 {
		t.Fatalf("a click frame builds %d trees, want 2 (the handler changed state)", got)
	}
}

// A window resize between the step and the draw drops the stored tree: it
// was built for the old size.
func TestResizeInvalidatesTheTree(t *testing.T) {
	a := testApp(t)
	a.Step(core.Input{X: 5000, Y: 5000})
	before := a.builds
	a.setSize(900, 600, 1)
	a.frameTree()
	if got := a.builds - before; got != 1 {
		t.Fatalf("Draw after a resize built %d trees, want 1 fresh one", got)
	}
	assertDrawnTreeIsFresh(t, a, "after the resize")
}

// The autosave runs after the view was built and marks the tabs of the
// collection saved; the tree Draw paints must show that.
func TestTheDrawnTreeShowsATabMarkedSavedByTheAutosave(t *testing.T) {
	a, _ := collectionsApp(t)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return clock }
	s := scriptedFrames{t, a}
	s.idle("start", 2)
	tab := a.cur()
	tab.Req.URL = "https://example.test/changed"
	s.step("edited", core.Input{X: 600, Y: 400}) // input makes the app look at the collection again
	if !tab.Dirty() {
		t.Fatal("the edit did not mark the tab unsaved")
	}
	for i := 0; i < 20 && tab.Dirty(); i++ {
		clock = clock.Add(time.Second)
		s.idle("waiting for the autosave", 1)
	}
	if tab.Dirty() {
		t.Fatal("the autosave never ran")
	}
}

// Each change a step makes after it built the view, without a handler, marks
// the tree stale. In a real frame a handler usually fires with them (a drag
// over a row runs OnDragOver), so the session above cannot tell these apart.
func TestAStepThatChangesWhatTheViewShowsMarksTheTreeStale(t *testing.T) {
	folder := &request.Folder{Name: "closed"}
	cases := map[string]func(a *App){
		"a toast":            func(a *App) { a.toast("Saved", severity.Info) },
		"a settings message": func(a *App) { a.Send(prefs.Toast{Text: "hi", Level: severity.Info}) },
		"bulk text turned to rows": func(a *App) {
			a.kv.ToggleBulk(kvlogic.Table{Tab: a.cur().Tab, Kind: kvlogic.Params})
			*a.kv.BulkArea() = widget.NewTextArea("a: b")
			a.cur().paneTab = headersTabIdx // another table shows: the next step applies the text
			a.Step(core.Input{X: 5000, Y: 5000})
		},
		"a drag resting over a folder": func(a *App) {
			a.side.Tree().Reset()
			holdDrag(a, request.Node{Req: &request.Request{}}, request.Node{Dir: folder})
			for range 29 { // one tick short of opening the folder
				a.side.Tick()
			}
			a.dragHover()
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			a := testApp(t)
			a.Step(core.Input{X: 5000, Y: 5000})
			a.treeStale = false
			change(a)
			if !a.treeStale {
				t.Fatal("the tree stays unmarked")
			}
		})
	}
}
