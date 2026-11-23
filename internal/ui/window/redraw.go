package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// The window is drawn when something may have changed and, whatever
// happens, at least engine.MinFPS times a second (Ebitengine skips the GPU
// work of a frame whose Draw draws nothing), and updates slow down when the
// app has been idle for a while.

// setSize records the window size and device scale and reports whether
// either changed; a change needs a redraw.
func (a *Component) setSize(width, height int, scale float64) bool {
	changed := width != a.w || height != a.h || (scale > 0 && scale != a.ctx.Scale)
	a.w, a.h = width, height
	if err := a.ctx.SetScale(scale); err != nil {
		a.err = err
	}
	if changed {
		a.dirty = true
		a.drawTree = nil // built for the old size and scale
	}
	return changed
}

// markFrame marks the frame dirty when engine.MustDraw says so.
func (a *Component) markFrame(active bool) {
	if engine.MustDraw(a.ctx.Tick, a.pacer.TPS(), active, a.alwaysDraw()) {
		a.dirty = true
	}
}

// alwaysDraw reports whether every frame must be drawn (a capture, startStart.DrawEveryFrame).
func (a *Component) alwaysDraw() bool { return a.drawEveryFrame }
