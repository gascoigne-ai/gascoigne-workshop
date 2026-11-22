package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// switchB is the toggle switch: a track with a round knob that sits at the
// right when on.
type switchB struct {
	Base
	on       bool
	disabled bool
}

// Sizes come from the size rules: the track is as wide as an icon button and
// a hairline taller than a checkbox each side, so the knob is a checkbox.
func (switchB) Measure(_ *El, ctx *core.Ctx, _ int) (int, int) {
	return ctx.Sizes.IconBtnW, ctx.Sizes.Checkbox + 2*core.StepHair
}

// Update runs OnChange when the track is clicked; the caller owns the value.
// A disabled switch ignores the click.
func (b switchB) Update(element *El, ctx *core.Ctx) {
	if b.disabled || !ctx.In.Hit(element.Rect()) {
		return
	}
	ctx.Fired++
	element.Changed()
}

func (b switchB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	track := element.Rect()
	knob := ctx.Sizes.Checkbox
	pill := func(side int) float64 { return float64(side) / 2 * min(ctx.Th.RadiusScale, 1) }
	trackColor, knobColor, edge := core.Track, core.Muted, core.BorderStrong
	if b.on && !b.disabled {
		trackColor, knobColor, edge = core.Accent, core.AccentText, core.Accent
	}
	if !b.disabled && ctx.Hovered(track) && !b.on {
		edge = core.Muted
	}
	fill := ctx.Color(trackColor)
	ctx.FillStyled(dst, track, fill, fill, pill(track.H))
	ctx.OutlineStyled(dst, track, ctx.Color(edge), pill(track.H))
	x := track.X + core.StepHair
	if b.on {
		x = track.X + track.W - core.StepHair - knob
	}
	knobRect := core.Rect{X: x, Y: track.Y + (track.H-knob)/2, W: knob, H: knob}
	paint := ctx.Color(knobColor)
	ctx.FillStyled(dst, knobRect, paint, paint, pill(knob))
}

// Switch is a toggle: an accent track with the knob at the right when on, a
// muted track with the knob at the left when off. OnChange runs when it is
// clicked; the caller flips the value it passes in, as for Checkbox. It has
// no keyboard focus, like Checkbox: a setting reached from the keyboard is
// reached through the raw JSON view.
func Switch(on bool, opts ...Option) *El {
	return Custom(switchB{on: on}, With(nil, opts)...)
}

// DisabledSwitch is a Switch in the muted look that ignores input, for a
// setting that cannot be changed now.
func DisabledSwitch(on bool, opts ...Option) *El {
	return Custom(switchB{on: on, disabled: true}, With(nil, opts)...)
}
