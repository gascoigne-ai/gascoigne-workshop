package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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

// restartsCaret reports whether this input shows the caret again: typing,
// editing keys and clicks do; a pointer moving over the window does not.
func restartsCaret(in core.Input) bool {
	return len(in.Typed) > 0 || len(in.Keys) > 0 || len(in.Edits) > 0 || in.Click || in.Enter ||
		in.Paste || in.Cut || in.Undo || in.Redo || in.SelectAll
}

// alwaysDraw reports whether every frame must be drawn (screenshots).
func (a *App) alwaysDraw() bool { return a.shot.path != "" }

// frameSize turns the window size (device independent) into what the app
// needs: the logical size it lays out in, the total scale (the display's
// factor times the interface scale) and the screen size in device pixels.
func frameSize(w, h int, dsf, ui float64) (lw, lh int, scale float64, sw, sh int) {
	return int(float64(w)/ui + 0.5), int(float64(h)/ui + 0.5), dsf * ui,
		int(float64(w)*dsf + 0.5), int(float64(h)*dsf + 0.5)
}
