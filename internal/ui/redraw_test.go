package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// settle runs idle frames and clears the dirty flag, as drawing does.
func settle(a *App, frames int) {
	for range frames {
		a.step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		a.dirty = false
	}
}

func TestAFreshAppNeedsADrawAndAnIdleOneIsDrawnOnlyAtTheFloorRate(t *testing.T) {
	a, _ := newTestApp(t)
	if !a.dirty {
		t.Fatal("the first frame must be drawn")
	}
	settle(a, 3)
	const frames = 600
	drawn, gap, longest := 0, 0, 0
	for range frames {
		a.step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
		gap++
		if a.dirty {
			drawn++
			longest = max(longest, gap)
			gap = 0
		}
		a.dirty = false
	}
	if drawn == frames {
		t.Fatal("an idle app drew every frame")
	}
	// at 60 updates a second the floor is every 6th frame; the rate drops to
	// 30 when idle, where it is every 3rd, so no gap is longer than 6 frames
	if longest > engine.ActiveTPS/engine.MinFPS {
		t.Fatalf("the longest wait between draws was %d frames, more than the %d FPS floor allows", longest, engine.MinFPS)
	}
}

func TestInputMakesTheNextFrameDirtyOnlyWhenItChangesSomething(t *testing.T) {
	a, _ := newTestApp(t)
	settle(a, 3)
	still := core.Input{X: 10, Y: 10}
	a.step(still)
	a.dirty = false
	a.step(still)
	if a.dirty && a.ctx.Tick%(a.pacer.TPS()/engine.MinFPS) != 0 { // only the floor may draw it
		t.Fatal("a pointer that did not move is not activity")
	}
	a.dirty = false

	for name, in := range map[string]core.Input{
		"a moved pointer": {X: 70, Y: 80},
		"a click":         {X: 50, Y: 60, Click: true, Held: true},
		"a held button":   {X: 50, Y: 60, Held: true},
		"the wheel":       {X: 50, Y: 60, Wheel: -1},
		"a right click":   {X: 50, Y: 60, RightClick: true},
		"typed text":      {X: 50, Y: 60, Typed: []rune("a")},
		"a key":           {X: 50, Y: 60, Keys: []ebiten.Key{ebiten.KeyF3}},
		"an edit key":     {X: 50, Y: 60, Edits: []core.Edit{core.EditLeft}},
		"Escape":          {X: 50, Y: 60, Escape: true},
		"a clipboard key": {X: 50, Y: 60, Paste: true},
	} {
		// map order is random: put the pointer at a known place first, so only
		// the case under test can be what makes the frame dirty
		a.step(core.Input{X: 50, Y: 60})
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
	a.step(core.Input{X: 5, Y: 5, Held: true})
	a.dirty = false
	a.step(core.Input{X: 5, Y: 5}) // released, nothing else changed
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
		a.step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
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
	a.step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
	if !a.dirty || len(a.toasts) != 0 {
		t.Fatalf("an expired toast must be removed and drawn away (dirty %v, toasts %d)", a.dirty, len(a.toasts))
	}
}

// The caret blinks on a clock of ticks and every frame at the floor is
// drawn, so it keeps blinking with or without a focused field and needs no
// special redraw of its own.
func TestTheFloorRateKeepsRedrawingWithAndWithoutAFocusedField(t *testing.T) {
	a, _ := newTestApp(t)
	settle(a, 3)
	draws := func(frames int) (n int) {
		for range frames {
			a.step(core.Input{X: a.lastIn.X, Y: a.lastIn.Y})
			if a.dirty {
				n++
			}
			a.dirty = false
		}
		return
	}
	unfocused := draws(120)
	a.cur().url.Focused = true
	focused := draws(120)
	if unfocused < 120*engine.MinFPS/engine.ActiveTPS || focused < 120*engine.MinFPS/engine.ActiveTPS {
		t.Fatalf("draws in 120 frames: unfocused %d, focused %d, want at least %d each", unfocused, focused, 120*engine.MinFPS/engine.ActiveTPS)
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
	a.step(core.Input{})
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

func TestFrameSizeAppliesTheInterfaceScaleOnTopOfTheDisplayScale(t *testing.T) {
	cases := []struct {
		name         string
		w, h         int
		dsf, ui      float64
		lw, lh       int
		scale        float64
		screenW, scH int
	}{
		{"no scaling", 1000, 800, 1, 1, 1000, 800, 1, 1000, 800},
		{"ten percent bigger", 1100, 880, 1, 1.1, 1000, 800, 1.1, 1100, 880},
		{"hidpi and bigger", 1000, 600, 2, 1.25, 800, 480, 2.5, 2000, 1200},
	}
	for _, c := range cases {
		lw, lh, scale, sw, sh := frameSize(c.w, c.h, c.dsf, c.ui)
		if lw != c.lw || lh != c.lh || scale != c.scale || sw != c.screenW || sh != c.scH {
			t.Errorf("%s: got %d %d %v %d %d", c.name, lw, lh, scale, sw, sh)
		}
	}
}

func TestInterfaceScaleSliderResizesLiveAndSavesOnRelease(t *testing.T) {
	a, path := newTestApp(t)
	click(a, menuBtnX, menuBtnY)
	a.prefs.Search = view.NewTextField("interface scale")
	a.step(core.Input{})
	a.step(core.Input{X: scaleTrackX0 + (scaleTrackX1-scaleTrackX0)/3, Y: scaleRowY, Click: true, Held: true})
	a.step(core.Input{X: scaleTrackX1 + 400, Y: scaleRowY, Held: true}) // far right: the maximum
	if a.settings.UIScalePct != 200 {
		t.Fatalf("not applied while dragging: %d", a.settings.UIScalePct)
	}
	if got, _ := settings.Load(path); got.UIScalePct == 200 {
		t.Fatal("saved before the release")
	}
	a.step(core.Input{X: scaleTrackX1 + 400, Y: scaleRowY})
	if got, _ := settings.Load(path); got.UIScalePct != 200 {
		t.Fatalf("not saved after the release: %+v", got)
	}
}

func TestNewAppsStartAtTheDisplayScale(t *testing.T) {
	a, _ := newTestApp(t)
	if a.settings.UIScale() != 1 {
		t.Fatalf("default scale %v", a.settings.UIScale())
	}
}

// The interface scale slider sits where the scale chooser was: its row at
// y=250, and its track within x=1030..1190 of the settings menu.
const (
	scaleRowY    = 274 // with "interface scale" typed in the search field
	scaleTrackX0 = 453
	scaleTrackX1 = 949
)
