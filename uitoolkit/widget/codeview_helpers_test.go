package widget

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// contentWidth is contentWidthOf with a geometry built on the spot; the
// code builds the geometry once per Update and Draw, the tests do not care.
func (v *CodeView) contentWidth(ctx *core.Ctx, rect core.Rect) int {
	return v.contentWidthOf(v.geometry(ctx, rect), rect)
}
