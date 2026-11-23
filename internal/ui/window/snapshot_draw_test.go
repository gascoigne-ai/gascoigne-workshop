package window

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// drawnElements draws the frame as the app does and says how many elements
// the draw walked.
func drawnElements(app *Component, screen *ebiten.Image) int {
	app.ctx.Stats = &core.DrawStats{}
	app.dirty = true
	app.Draw(screen)
	return app.ctx.Stats.Elements
}

// wholeElements is how many elements a full draw of the frame's tree walks.
func wholeElements(app *Component) int {
	app.ctx.Stats = &core.DrawStats{}
	view.Draw(app.ctx, app.frameTree(), ebiten.NewImage(app.w, app.h))
	return app.ctx.Stats.Elements
}

// topElements is how many elements the part above the base walks.
func topElements(app *Component) int {
	tree := app.frameTree()
	app.ctx.Stats = &core.DrawStats{}
	view.DrawFrom(app.ctx, tree, view.ModalLayer(app.ctx, tree), ebiten.NewImage(app.w, app.h))
	return app.ctx.Stats.Elements
}

// A fixed clock the caret blinks by, with the tree filter focused (a base
// field that keeps the focus under the dialog) and the settings dialog open
// and drawn once.
func snapshotApp(t *testing.T) (*Component, *time.Time, *ebiten.Image) {
	t.Helper()
	a := testApp(t)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return clock }
	scripted := scriptedFrames{t, a}
	scripted.idle("start", 3)
	scripted.click("focus the tree filter", 180, 22)
	a.over.Settings.Show()
	scripted.idle("dialog opens", 2)
	screen := ebiten.NewImage(a.w, a.h)
	drawnElements(a, screen)
	return a, &clock, screen
}

// The window's root keeps the base column, the scrim and then the dialog
// panel as direct children, so the split lands on the panel (pins how the
// markup slots flatten).
func TestTheWindowTreeSplitsAtTheSettingsPanel(t *testing.T) {
	a := testApp(t)
	a.over.Settings.Show()
	a.Step(core.Input{X: 5000, Y: 5000})
	tree := a.frameTree()
	layer := view.ModalLayer(a.ctx, tree)
	if layer != 2 {
		t.Fatalf("ModalLayer = %d, want 2 (base column, scrim, panel)", layer)
	}
	if got, want := tree.Children()[layer].Rect(), settingsRect(t, a); got != want {
		t.Fatalf("the layer is not the settings panel: %v, want %v", got, want)
	}
	if scrim := tree.Children()[layer-1].Rect(); scrim.W != a.w || scrim.H != a.h {
		t.Fatalf("the child before the panel is not the scrim: %v", scrim)
	}
}

func TestADragUnderTheSettingsDialogDrawsOnlyTheDialog(t *testing.T) {
	a, _, screen := snapshotApp(t)
	paints := a.snap.paints
	dialog := settingsRect(t, a)
	grabX, grabY := dialog.X+dialog.W/2, dialog.Y+20
	a.Step(core.Input{X: grabX, Y: grabY, Click: true, Held: true})
	drawnElements(a, screen) // pressing is an event: the base may be drawn again
	paints = a.snap.paints
	for i := 1; i <= 5; i++ {
		a.Step(core.Input{X: grabX + 7*i, Y: grabY + 3*i, Held: true})
		got := drawnElements(a, screen)
		if want := topElements(a); got != want {
			t.Fatalf("drag frame %d walked %d elements, want only the %d above the base", i, got, want)
		}
		if whole := wholeElements(a); got >= whole {
			t.Fatalf("drag frame %d walked %d of %d elements: the base was drawn", i, got, whole)
		}
	}
	if a.snap.paints != paints {
		t.Fatalf("the base was drawn %d more times during the drag", a.snap.paints-paints)
	}
}

func TestABaseChangeUnderTheDialogRedrawsTheBase(t *testing.T) {
	tests := []struct {
		name  string
		event func(t *testing.T, a *Component, clock *time.Time)
	}{
		{"a drained message", func(t *testing.T, a *Component, _ *time.Time) {
			a.bus.Dispatch(func() {})
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		}},
		{"a websocket message", func(t *testing.T, a *Component, _ *time.Time) {
			a.bus.Dispatch(wsReceived{wsTarget{tab: a.cur()}, request.Event{Dir: request.Info, Kind: request.KindInfo, Text: "hi"}})
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		}},
		{"a send result", func(t *testing.T, a *Component, _ *time.Time) {
			tab := a.cur()
			url := jsonServer(t).URL
			a.bus.Dispatch(func() { a.pointTabAt(tab, url); tab.Resp = nil; a.send(tab) })
			for i := 0; tab.Resp == nil && i < 400; i++ {
				a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
				time.Sleep(2 * time.Millisecond)
			}
			if tab.Resp == nil {
				t.Fatal("no response")
			}
		}},
		{"a key frame", func(t *testing.T, a *Component, _ *time.Time) {
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y, Enter: true})
		}},
		{"a live theme edit", func(t *testing.T, a *Component, _ *time.Time) {
			scripted := scriptedFrames{t, a}
			scripted.clickText("Light", 0)
		}},
		{"a resize", func(t *testing.T, a *Component, _ *time.Time) {
			a.setSize(a.w-40, a.h-30, 1)
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		}},
		{"a scale change", func(t *testing.T, a *Component, _ *time.Time) {
			a.setSize(a.w, a.h, 1.25)
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		}},
		{"a toast shown by a message", func(t *testing.T, a *Component, _ *time.Time) {
			a.bus.Dispatch(func() { a.toast("Saved", severity.Info) })
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		}},
		{"a toast that expired", func(t *testing.T, a *Component, clock *time.Time) {
			a.bus.Dispatch(func() { a.toast("Saved", severity.Info) })
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
			*clock = clock.Add(time.Minute)
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		}},
		{"the caret phase of a focused base field", func(t *testing.T, a *Component, clock *time.Time) {
			*clock = clock.Add(core.CaretHalfPeriod + time.Millisecond)
			a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		}},
	}
	for _, tc := range tests {
		for _, quietSteps := range []int{0, 3} { // Ebitengine may run several Updates per Draw
			t.Run(fmt.Sprintf("%s, %d steps after", tc.name, quietSteps), func(t *testing.T) {
				a, clock, screen := snapshotApp(t)
				if a.snap.geometry.caret == caretUnused {
					t.Fatal("the focused filter did not read the caret phase")
				}
				paints := a.snap.paints
				tc.event(t, a, clock)
				for range quietSteps {
					a.Step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
				}
				screen = ebiten.NewImage(a.w, a.h)
				got := drawnElements(a, screen)
				if whole := wholeElements(a); got != whole {
					t.Errorf("the draw walked %d elements, want the whole %d", got, whole)
				}
				if a.snap.paints != paints+1 {
					t.Errorf("the base was drawn %d times, want once", a.snap.paints-paints)
				}
			})
		}
	}
}

// What a snapshot may skip is drawing, never a change: while the image is
// fresh the base children must dump as they did when it was drawn.
func TestTheSnapshotMatchesTheBaseAcrossASession(t *testing.T) {
	a := testApp(t)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return clock }
	screen := ebiten.NewImage(a.w, a.h)
	var drawnDump string
	lastPaints := 0
	reuses := 0
	baseDump := func() (string, bool) {
		tree := a.frameTree()
		layer := view.ModalLayer(a.ctx, tree)
		if layer < 0 {
			return "", false
		}
		var sb strings.Builder
		for _, kid := range tree.Children()[:layer] {
			sb.WriteString(view.Dump(kid))
		}
		return sb.String(), true
	}
	frame := func(label string, in core.Input, steps int) {
		t.Helper()
		a.Step(in)
		for range steps - 1 {
			a.Step(core.Input{X: in.X, Y: in.Y})
		}
		now, open := baseDump()
		a.dirty = true
		a.Draw(screen)
		if !open {
			lastPaints = a.snap.paints
			return
		}
		if a.snap.paints != lastPaints {
			lastPaints, drawnDump = a.snap.paints, now
			return
		}
		reuses++
		if now != drawnDump {
			t.Fatalf("%s: the base was reused but changed, %s", label, firstDifference(drawnDump, now))
		}
	}
	idle := func(label string, frames int) {
		for range frames {
			frame(label, core.Input{X: a.lastIn.X, Y: a.lastIn.Y}, 1)
		}
	}
	clickAt := func(label string, x, y int) {
		frame(label+" (press)", core.Input{X: x, Y: y, Click: true, Held: true}, 1)
		frame(label+" (release)", core.Input{X: x, Y: y}, 2)
	}
	clickText := func(text string) {
		x, y := lineCenter(t, dumpedLines(a, fmt.Sprintf("text=%q", text))[0])
		clickAt("click "+text, x, y)
	}

	idle("start", 3)
	clickAt("focus the tree filter", 180, 22)
	a.over.Settings.Show()
	idle("dialog opens", 3)
	dialog := settingsRect(t, a)
	grabX, grabY := dialog.X+dialog.W/2, dialog.Y+20
	frame("press the dialog title", core.Input{X: grabX, Y: grabY, Click: true, Held: true}, 1)
	for i := 1; i <= 8; i++ {
		frame("drag the dialog", core.Input{X: grabX + 10*i, Y: grabY + 6*i, Held: true}, 1+i%2)
	}
	frame("drop the dialog", core.Input{X: grabX + 80, Y: grabY + 48}, 1)
	clickText("Light")
	clickText("Dark")
	clickAt("focus the settings search", 400, 184)
	for _, char := range "font" {
		frame("type in the settings search", core.Input{X: 400, Y: 184, Typed: []rune{char}}, 1)
	}
	idle("idle", 4)

	a.bus.Dispatch(wsReceived{wsTarget{tab: a.cur()}, request.Event{Dir: request.Info, Kind: request.KindInfo, Text: "hi"}})
	idle("a websocket message", 2)
	a.bus.Dispatch(func() { a.toast("Saved", severity.Info) })
	idle("toast shown", 2)
	clock = clock.Add(10 * time.Second)
	idle("toast expired", 2)

	tab := a.cur()
	url := jsonServer(t).URL
	a.bus.Dispatch(func() { a.pointTabAt(tab, url); tab.Resp = nil; a.send(tab) })
	for i := 0; tab.Resp == nil && i < 400; i++ { // a send in flight while the clock advances
		clock = clock.Add(100 * time.Millisecond)
		idle("waiting for the response", 1)
		time.Sleep(2 * time.Millisecond)
	}
	idle("response shown", 2)

	a.setSize(1000, 700, 1)
	screen = ebiten.NewImage(a.w, a.h)
	idle("after the resize", 2)
	a.setSize(1000, 700, 1.25)
	screen = ebiten.NewImage(int(float64(a.w)*1.25), int(float64(a.h)*1.25))
	idle("after the scale change", 2)

	a.confirm("Sure?", "Really?", "OK", func() {})
	idle("a confirm over the settings", 3)
	frame("escape closes the confirm", core.Input{Escape: true}, 1)
	frame("escape closes the settings", core.Input{Escape: true}, 1)
	idle("closed", 2)
	if reuses == 0 {
		t.Fatal("the session never reused the snapshot: the test checks nothing")
	}
}
