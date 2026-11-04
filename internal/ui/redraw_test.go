package ui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

// settle runs idle frames and clears the dirty flag, as drawing does.
func settle(a *App, frames int) {
	for range frames {
		a.step(widget.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		a.dirty = false
	}
}

func TestAFreshAppNeedsADrawAndAnIdleOneDoesNot(t *testing.T) {
	a, _ := newTestApp(t)
	if !a.dirty {
		t.Fatal("the first frame must be drawn")
	}
	settle(a, 3)
	for range 300 {
		a.step(widget.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		if a.dirty && a.ctx.Tick%engine.HeartbeatTicks != 0 {
			t.Fatalf("an idle frame at tick %d asked for a redraw", a.ctx.Tick)
		}
		a.dirty = false
	}
}

func TestIdleHeartbeatRedrawsOnceInAWhile(t *testing.T) {
	a, _ := newTestApp(t)
	settle(a, 3)
	beats := 0
	for range 3 * engine.HeartbeatTicks {
		a.step(widget.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		if a.dirty {
			beats++
		}
		a.dirty = false
	}
	if beats < 2 || beats > 4 {
		t.Fatalf("%d heartbeat draws in %d idle frames", beats, 3*engine.HeartbeatTicks)
	}
}

func TestInputMakesTheNextFrameDirtyOnlyWhenItChangesSomething(t *testing.T) {
	a, _ := newTestApp(t)
	settle(a, 3)
	still := widget.Input{X: 10, Y: 10}
	a.step(still)
	a.dirty = false
	a.step(still)
	if a.dirty && a.ctx.Tick%engine.HeartbeatTicks != 0 {
		t.Fatal("a pointer that did not move is not activity")
	}
	a.dirty = false

	for name, in := range map[string]widget.Input{
		"a moved pointer": {X: 70, Y: 80},
		"a click":         {X: 50, Y: 60, Click: true, Held: true},
		"a held button":   {X: 50, Y: 60, Held: true},
		"the wheel":       {X: 50, Y: 60, Wheel: -1},
		"a right click":   {X: 50, Y: 60, RightClick: true},
		"typed text":      {X: 50, Y: 60, Typed: []rune("a")},
		"a key":           {X: 50, Y: 60, Keys: []ebiten.Key{ebiten.KeyF3}},
		"an edit key":     {X: 50, Y: 60, Edits: []widget.Edit{widget.EditLeft}},
		"Escape":          {X: 50, Y: 60, Escape: true},
		"a clipboard key": {X: 50, Y: 60, Paste: true},
	} {
		// map order is random: put the pointer at a known place first, so only
		// the case under test can be what makes the frame dirty
		a.step(widget.Input{X: 50, Y: 60})
		settle(a, 2)
		a.dirty = false
		a.step(in)
		if !a.dirty {
			t.Errorf("%s must request a redraw", name)
		}
		a.dirty = false
	}
}

func TestReleasingTheButtonIsActivity(t *testing.T) {
	a, _ := newTestApp(t)
	a.step(widget.Input{X: 5, Y: 5, Held: true})
	a.dirty = false
	a.step(widget.Input{X: 5, Y: 5}) // released, nothing else changed
	if !a.dirty {
		t.Fatal("the release ends a drag or a press: redraw")
	}
}

func TestAnArrivingResponseRequestsARedraw(t *testing.T) {
	srv := jsonServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.send(tab)
	for range 400 {
		a.dirty = false
		a.step(widget.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		if tab.resp != nil {
			if !a.dirty {
				t.Fatal("the frame that applied the response must be drawn")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("no response")
}

func TestToastsRequestRedrawsWhenShownAndWhenTheyExpire(t *testing.T) {
	a, _ := newTestApp(t)
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return clock }
	settle(a, 2)
	a.toast("hello", false)
	if !a.dirty {
		t.Fatal("showing a toast needs a draw")
	}
	settle(a, 2)
	clock = clock.Add(10 * time.Second)
	a.step(widget.Input{X: a.lastIn.X, Y: a.lastIn.Y})
	if !a.dirty || len(a.toasts) != 0 {
		t.Fatalf("an expired toast must be removed and drawn away (dirty %v, toasts %d)", a.dirty, len(a.toasts))
	}
}

func TestTheCaretBlinkRedrawsOnlyWhileATextWidgetHasFocus(t *testing.T) {
	a, _ := newTestApp(t)
	settle(a, 3)
	blinks := func(frames int) (n int) {
		for range frames {
			a.step(widget.Input{X: a.lastIn.X, Y: a.lastIn.Y})
			if a.dirty && a.ctx.Tick%engine.HeartbeatTicks != 0 {
				n++
			}
			a.dirty = false
		}
		return
	}
	if got := blinks(100); got != 0 {
		t.Fatalf("no focused field: %d blink redraws", got)
	}
	a.cur().url.Focused = true
	if got := blinks(100); got < 2 {
		t.Fatalf("a focused field must redraw for its caret, got %d", got)
	}
	a.cur().url.Focused = false
	if got := blinks(100); got != 0 {
		t.Fatalf("after losing focus: %d", got)
	}
}

func TestResizingOrChangingScaleRequestsARedraw(t *testing.T) {
	a, _ := newTestApp(t)
	settle(a, 2)
	if a.setSize(winW, winH, a.ctx.Scale) || a.dirty {
		t.Fatal("the same size is not a change")
	}
	if !a.setSize(900, winH, a.ctx.Scale) || !a.dirty {
		t.Fatal("a new width must request a redraw")
	}
	a.dirty = false
	if !a.setSize(900, winH, 2) || !a.dirty {
		t.Fatal("a new scale must request a redraw")
	}
}

func TestScreenshotModeAlwaysDraws(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png")})
	settle(a, 2)
	a.step(widget.Input{})
	a.shotUpdate()
	if !a.dirty && !a.alwaysDraw() {
		t.Fatal("a screenshot needs every frame")
	}
}

func TestUpdateRateDropsAfterIdleAndReturnsWithActivity(t *testing.T) {
	a, _ := newTestApp(t)
	if a.pacer.TPS() != engine.ActiveTPS {
		t.Fatalf("start at %d", a.pacer.TPS())
	}
	for range engine.IdleAfterTicks + 5 {
		a.pacer.NoteIdle(false)
	}
	if a.pacer.TPS() != engine.IdleTPS {
		t.Fatalf("after a quiet spell: %d", a.pacer.TPS())
	}
	a.pacer.NoteIdle(true)
	if a.pacer.TPS() != engine.ActiveTPS {
		t.Fatalf("activity restores the rate: %d", a.pacer.TPS())
	}
}

func TestDrawingClearsTheDirtyFlagAndAnIdleDrawDoesNothing(t *testing.T) {
	a, _ := newTestApp(t)
	img := ebiten.NewImage(winW, winH)
	a.dirty = true
	a.Draw(img)
	if a.dirty {
		t.Fatal("a draw consumes the dirty flag")
	}
	a.Draw(img) // nothing changed: must return without drawing or panicking
	if a.dirty {
		t.Fatal("an idle Draw must not mark anything")
	}
}
