package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// The window is drawn when something may have changed and, whatever
// happens, at least engine.MinFPS times a second (Ebitengine skips the GPU
// work of a frame whose Draw draws nothing), and updates slow down when the
// app has been idle for a while.

// setSize records the window size and device scale and reports whether
// either changed; a change needs a redraw.
func (a *App) setSize(w, h int, scale float64) bool {
	changed := w != a.w || h != a.h || (scale > 0 && scale != a.ctx.Scale)
	a.w, a.h = w, h
	if err := a.ctx.SetScale(scale); err != nil {
		a.shot.err = err
	}
	if changed {
		a.dirty = true
	}
	return changed
}

// markFrame marks the frame dirty when engine.MustDraw says so.
func (a *App) markFrame(active bool) {
	if engine.MustDraw(a.ctx.Tick, a.pacer.TPS(), active, a.alwaysDraw()) {
		a.dirty = true
	}
}

// alwaysDraw reports whether every frame must be drawn (screenshots).
func (a *App) alwaysDraw() bool { return a.shot.path != "" }

// frame is what Layout works out from the window size.
type frame struct {
	LW, LH      int     // logical size the UI lays out in
	SW, SH      int     // screen size in device pixels
	Scale       float64 // device pixels per logical pixel
	WindowScale float64 // window units per logical pixel, for remembered sizes
}

// frameSize turns the window size (in the units the window system uses)
// into a frame. reported is the scale Ebitengine knows and actual the
// display's real one: they differ when the desktop scales the display
// without telling the window (Xwayland), whose window is then in real
// pixels. ui is the interface scale on top.
func frameSize(w, h int, reported, actual, ui float64) frame {
	scale := actual * ui
	return frame{
		LW: int(float64(w)*reported/scale + 0.5), LH: int(float64(h)*reported/scale + 0.5),
		SW: int(float64(w)*reported + 0.5), SH: int(float64(h)*reported + 0.5),
		Scale: scale, WindowScale: scale / reported,
	}
}

// displayScale is the display's real scale: what the window system
// reports, unless that is 1 and the desktop says otherwise (a compositor
// that scales Xwayland windows itself).
func (a *App) displayScale(reported float64) float64 {
	if reported > 1 || a.display == nil {
		return reported
	}
	if factor := a.display.Factor(); factor > 0 {
		return factor
	}
	return reported
}
