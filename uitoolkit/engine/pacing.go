// Package engine holds the app-independent parts of the frame loop:
// deciding when a frame must be drawn and how fast to update, and the
// key-binding model. Nothing here knows about the workshop's screens.
package engine

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// HeartbeatTicks is the number of ticks after which a frame is drawn
// even when nothing changed, in case the window system lost the content.
const HeartbeatTicks = 120

// blinkTicks is the half period of the text caret's blink.
const blinkTicks = 30

// Ticks of no activity before the update rate drops, and the two rates.
const (
	IdleAfterTicks = 60
	ActiveTPS      = 60
	IdleTPS        = 30
)

// InputActive reports whether this frame's input differs from the last
// one in a way that can change what is on screen. A resting pointer and
// held-still keys are not activity.
func InputActive(prev, in core.Input) bool {
	return in.X != prev.X || in.Y != prev.Y ||
		in.Click || in.Held || prev.Held || in.RightClick || in.MiddleClick || in.Wheel != 0 ||
		len(in.Typed) > 0 || len(in.Keys) > 0 || len(in.Edits) > 0 ||
		in.Enter || in.Escape || in.Tab ||
		in.Undo || in.Redo || in.Copy || in.Cut || in.Paste || in.SelectAll
}

// Pacer tracks idle time to pick the update rate and decides, per
// frame, whether it must be drawn.
type Pacer struct {
	idle int // frames in a row with nothing to redraw
}

// NoteIdle counts a frame as active or idle, for the update rate.
func (p *Pacer) NoteIdle(active bool) {
	if active {
		p.idle = 0
		return
	}
	p.idle++
}

// TPS is the update rate for the current idle time.
func (p *Pacer) TPS() int {
	if p.idle > IdleAfterTicks {
		return IdleTPS
	}
	return ActiveTPS
}

// MustDraw reports whether the frame at tick must be drawn: something
// changed, the caret blinks, the heartbeat is due, or always is set.
func MustDraw(tick int, active, caret, always bool) bool {
	return active || (tick%blinkTicks == 0 && caret) || tick%HeartbeatTicks == 0 || always
}
