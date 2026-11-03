package ui

import "github.com/gascoigne-ai/gascoigne-workshop/widget"

// The window is drawn only when something may have changed (Ebitengine
// skips the GPU work of a frame whose Draw draws nothing), and updates
// slow down when the app has been idle for a while.

// Ticks after which a frame is drawn even when nothing changed, in case
// the window system lost what is on screen.
const heartbeatTicks = 120

// blinkTicks is the half period of the text caret's blink.
const blinkTicks = 30

// Ticks of no activity before the update rate drops, and the two rates.
const (
	idleAfterTicks = 60
	activeTPS      = 60
	idleTPS        = 30
)

// inputActive reports whether this frame's input differs from the last
// in a way that can change what is on screen: a moved pointer, a button
// held or just released, the wheel, typing or any key.
func inputActive(prev, in widget.Input) bool {
	return in.X != prev.X || in.Y != prev.Y ||
		in.Click || in.Held || prev.Held || in.RightClick || in.MiddleClick || in.Wheel != 0 ||
		len(in.Typed) > 0 || len(in.Keys) > 0 || len(in.Edits) > 0 ||
		in.Enter || in.Escape || in.Tab ||
		in.Undo || in.Redo || in.Copy || in.Cut || in.Paste || in.SelectAll
}

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

// caretShown reports whether a text caret is on screen, which blinks.
func (a *App) caretShown() bool {
	if a.textFocused() || (a.dialog != nil && a.dialog.input != nil && a.dialog.input.Focused) {
		return true
	}
	for _, in := range a.colorInputs {
		if in.Focused {
			return true
		}
	}
	return false
}

// markFrame decides whether this frame must be drawn: something changed
// (input, a response, a toast), a caret blinks, the heartbeat is due,
// or a screenshot is being taken.
func (a *App) markFrame(active bool) {
	tick := a.ctx.Tick
	if active || (tick%blinkTicks == 0 && a.caretShown()) || tick%heartbeatTicks == 0 || a.alwaysDraw() {
		a.dirty = true
	}
}

// noteIdle counts a frame as active or idle, for the update rate.
func (a *App) noteIdle(active bool) {
	if active {
		a.idleTicks = 0
		return
	}
	a.idleTicks++
}

// tpsFor is the update rate for the current idle time.
func (a *App) tpsFor() int {
	if a.idleTicks > idleAfterTicks {
		return idleTPS
	}
	return activeTPS
}

// alwaysDraw reports whether every frame must be drawn (screenshots).
func (a *App) alwaysDraw() bool { return a.shot.path != "" }
