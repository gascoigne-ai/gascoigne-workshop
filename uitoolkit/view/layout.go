package view

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// plain is a behavior with no content of its own.
type plain struct{}

func (plain) measure(*El, *core.Ctx, int) (int, int) { return 0, 0 }
func (plain) layout(*El, *core.Ctx)                  {}
func (plain) draw(*El, *core.Ctx, *ebiten.Image)     {}
func (plain) update(*El, *core.Ctx)                  {}

// inner is the rect inside the padding.
func (e *El) inner() core.Rect {
	r := e.rect
	return core.Rect{
		X: r.X + e.padL, Y: r.Y + e.padT,
		W: max(r.W-e.padL-e.padR, 0), H: max(r.H-e.padT-e.padB, 0),
	}
}

// size is the element's wanted size: its fixed size where set, else
// what its behavior measures.
func (e *El) size(ctx *core.Ctx, maxW int) (w, h int) {
	w, h = e.b.measure(e, ctx, maxW)
	if e.w > 0 {
		w = e.w
	}
	if e.h > 0 {
		h = e.h
	}
	return e.clampSize(w, h)
}

func (e *El) place(ctx *core.Ctx, bounds core.Rect) {
	e.rect = bounds
	e.b.layout(e, ctx)
}

// flow lays children out along one axis: a Row (horizontal) or a Col.
type flow struct {
	plain
	horiz bool
}

func (f flow) main(width, height int) int {
	if f.horiz {
		return width
	}
	return height
}

func (f flow) cross(width, height int) int {
	if f.horiz {
		return height
	}
	return width
}

// Margins along the main axis (total and leading) and across it.
func (f flow) marginMain(element *El) int {
	if f.horiz {
		return element.mL + element.mR
	}
	return element.mT + element.mB
}

func (f flow) marginLead(element *El) int {
	if f.horiz {
		return element.mL
	}
	return element.mT
}

func (f flow) marginCross(element *El) int {
	if f.horiz {
		return element.mT + element.mB
	}
	return element.mL + element.mR
}

func (f flow) marginCrossLead(element *El) int {
	if f.horiz {
		return element.mT
	}
	return element.mL
}

// baseline is the distance from a child's top to its text baseline; a
// child that is not text has its bottom edge as baseline.
func (f flow) baseline(element *El, ctx *core.Ctx, hugCross int) int {
	if _, isText := element.b.(textB); isText {
		return ctx.Ascent(element.face(ctx))
	}
	return hugCross
}

func (f flow) measure(element *El, ctx *core.Ctx, maxW int) (int, int) {
	inW := max(maxW-element.padL-element.padR, 0)
	var main, cross, n, above, below int
	for _, k := range element.kids {
		if k.hidden || k.fl.on {
			continue
		}
		w, h := k.size(ctx, inW)
		main += f.main(w, h) + f.marginMain(k)
		cross = max(cross, f.cross(w, h)+f.marginCross(k))
		if element.alignItems == AlignBaseline && f.horiz {
			base := f.baseline(k, ctx, h)
			above, below = max(above, base), max(below, h-base)
		}
		n++
	}
	if above+below > cross {
		cross = above + below // a baseline-aligned row is as tall as its lines stacked on one baseline
	}
	if n > 1 {
		main += (n - 1) * element.gap
	}
	if f.horiz {
		return main + element.padL + element.padR, cross + element.padT + element.padB
	}
	return cross + element.padL + element.padR, main + element.padT + element.padB
}

func (f flow) layout(element *El, ctx *core.Ctx) {
	in := element.inner()
	avail, crossAvail := f.main(in.W, in.H), f.cross(in.W, in.H)

	// Fixed and content-sized children take their size first (margins
	// included); the rest is shared by the growing ones.
	sizes, hug := make([]int, len(element.kids)), make([]int, len(element.kids))
	for i, k := range element.kids {
		w, h := k.size(ctx, in.W)
		sizes[i], hug[i] = f.main(w, h)+f.marginMain(k), f.cross(w, h)
	}
	element.dropOptional(sizes, avail)
	element.shrinkToFit(f, sizes, avail)

	used, growSum, n := 0, 0.0, 0
	for i, k := range element.kids {
		if k.hidden || k.fl.on {
			continue
		}
		n++
		if f.growing(k) {
			growSum += k.grow
			used += f.marginMain(k)
			continue
		}
		used += sizes[i]
	}
	free := max(avail-used-max(n-1, 0)*element.gap, 0)

	pos, extra := f.main(in.X, in.Y), 0
	if growSum == 0 {
		pos, extra = f.justified(element, pos, free, n)
	}
	baseMax, baseBelow := 0, 0 // the line of baseline-aligned children: above and below the baseline
	if element.alignItems == AlignBaseline && f.horiz {
		for i, k := range element.kids {
			if !k.hidden && !k.fl.on {
				base := f.baseline(k, ctx, hug[i])
				baseMax, baseBelow = max(baseMax, base), max(baseBelow, hug[i]-base)
			}
		}
	}
	lineTop := max((crossAvail-baseMax-baseBelow)/2, 0) // the line is centered in the row

	lastGrow, given := -1, 0
	for i, k := range element.kids {
		if !k.hidden && !k.fl.on && f.growing(k) {
			lastGrow = i
		}
	}
	for i, k := range element.kids {
		if k.hidden || k.fl.on {
			k.rect = core.Rect{}
			continue
		}
		outer := sizes[i]
		size := outer - f.marginMain(k)
		if f.growing(k) {
			size = int(float64(free) * k.grow / growSum)
			given += size
			if i == lastGrow {
				size += free - given // rounding remainder
			}
			size = f.clampMain(k, size)
			outer = size + f.marginMain(k)
		}
		crossSize, crossOff := f.crossPlacement(element, k, ctx, crossAvail, hug[i], baseMax, lineTop)
		k.place(ctx, f.rect(in, pos+f.marginLead(k), crossOff, size, crossSize))
		pos += outer + element.gap + extra
	}
}

// growing reports whether a child takes a share of the free space.
func (f flow) growing(element *El) bool {
	fixed := element.h
	if f.horiz {
		fixed = element.w
	}
	return fixed == 0 && element.grow > 0
}

func (f flow) clampMain(element *El, size int) int {
	if f.horiz {
		w, _ := element.clampSize(size, 0)
		return w
	}
	_, h := element.clampSize(0, size)
	return h
}

// justified returns the start position and the extra space after each
// child for the container's JustifyContent.
func (f flow) justified(element *El, pos, free, count int) (start, extra int) {
	switch element.justify {
	case JustifyCenter:
		return pos + free/2, 0
	case JustifyEnd:
		return pos + free, 0
	case JustifyBetween:
		if count > 1 {
			return pos, free / (count - 1)
		}
	case JustifyAround:
		if count > 0 {
			return pos + free/count/2, free / count
		}
	}
	return pos, 0
}

// crossPlacement is a child's size across the axis and its offset from
// the start of the container's inner area, margins excluded from the
// size and included in the offset by the caller.
func (f flow) crossPlacement(element *El, child *El, ctx *core.Ctx, crossAvail, hug, baseMax, lineTop int) (size, off int) {
	avail := max(crossAvail-f.marginCross(child), 0)
	fixed := child.w
	if f.horiz {
		fixed = child.h
	}
	align := child.alignSelf
	if align == AlignDefault {
		align = element.alignItems
	}
	clamp := func(count int) int {
		if f.horiz {
			_, h := child.clampSize(0, count)
			return h
		}
		w, _ := child.clampSize(count, 0)
		return w
	}
	size = clamp(hug)
	switch align {
	case AlignStart:
	case AlignCenter:
		off = (avail - size) / 2
	case AlignEnd:
		off = avail - size
	case AlignBaseline:
		off = lineTop + baseMax - f.baseline(child, ctx, hug)
	case AlignStretch:
		if fixed == 0 {
			size = clamp(avail)
		}
	default: // AlignDefault: stretch; a fixed size is centered in a Row, left in a Col
		if fixed == 0 {
			size = clamp(avail)
		} else if f.horiz {
			off = (avail - size) / 2
		}
	}
	return size, f.marginCrossLead(child) + off
}

// rect builds a child's rect from its main-axis start and size and its
// cross-axis offset and size.
func (f flow) rect(in core.Rect, mainPos, crossOff, mainSize, crossSize int) core.Rect {
	if f.horiz {
		return core.Rect{X: mainPos, Y: in.Y + crossOff, W: mainSize, H: crossSize}
	}
	return core.Rect{X: in.X + crossOff, Y: mainPos, W: crossSize, H: mainSize}
}

// dropOptional hides optional children when everything does not fit.
func (e *El) dropOptional(sizes []int, avail int) {
	total, n := 0, 0
	for i, k := range e.kids {
		k.hidden = false
		if k.fl.on {
			continue
		}
		total += sizes[i]
		n++
	}
	total += max(n-1, 0) * e.gap
	if total <= avail {
		return
	}
	for _, k := range e.kids {
		if k.optional {
			k.hidden = true
		}
	}
}

// shrinkToFit takes what the children overflow the row by from its Shrink
// children, first to last, each down to its minimum size along the axis.
// A row that fits is left as it is.
func (e *El) shrinkToFit(axis flow, sizes []int, avail int) {
	used, n := 0, 0
	for i, k := range e.kids {
		if k.hidden || k.fl.on {
			continue
		}
		n++
		if axis.growing(k) {
			used += axis.marginMain(k)
			continue
		}
		used += sizes[i]
	}
	over := used + max(n-1, 0)*e.gap - avail
	for i, k := range e.kids {
		if over <= 0 {
			return
		}
		if !k.shrink || k.hidden || k.fl.on || axis.growing(k) {
			continue
		}
		give := min(sizes[i]-axis.marginMain(k)-axis.minMain(k), over)
		if give > 0 {
			sizes[i] -= give
			over -= give
		}
	}
}

// minMain is a child's MinW in a Row, its MinH in a Col.
func (f flow) minMain(element *El) int {
	if f.horiz {
		return element.minW
	}
	return element.minH
}

// stack layers its children over each other, each filling the element
// unless it floats.
type stack struct{ plain }

func (stack) measure(element *El, ctx *core.Ctx, maxW int) (int, int) {
	var w, h int
	for _, k := range element.kids {
		if k.fl.on {
			continue
		}
		kw, kh := k.size(ctx, maxW)
		w, h = max(w, kw), max(h, kh)
	}
	return w + element.padL + element.padR, h + element.padT + element.padB
}

func (stack) layout(element *El, ctx *core.Ctx) {
	in := element.inner()
	for _, k := range element.kids {
		if !k.fl.on {
			k.place(ctx, in)
			continue
		}
		w, h := k.size(ctx, in.W)
		w, h = min(w, in.W), min(h, in.H)
		k.place(ctx, floatRect(k, in, w, h))
	}
}

// Row lays its children out left to right.
func Row(items ...Item) *El { return newEl(flow{horiz: true}, items) }

// Col lays its children out top to bottom.
func Col(items ...Item) *El { return newEl(flow{}, items) }

// Stack layers its children; use Float on the ones that should hover
// over the rest, such as a menu.
func Stack(items ...Item) *El { return newEl(stack{}, items) }

// Spacer is an empty element that takes all the free space, pushing its
// neighbours apart.
func Spacer(opts ...Option) *El { return newEl(plain{}, with([]Item{Grow(1)}, opts)) }

// HRule is a one-pixel horizontal line in the border color; options
// override that look (a margin, another color).
func HRule(opts ...Option) *El { return newEl(plain{}, with([]Item{H(1), Bg(core.Border)}, opts)) }

// VRule is a one-pixel vertical line in the border color; options
// override that look.
func VRule(opts ...Option) *El { return newEl(plain{}, with([]Item{W(1), Bg(core.Border)}, opts)) }

// VRuleInset is a vertical line shortened by inset pixels at both ends,
// for dividing the chips of a bar. The options apply to the box that
// holds the line.
func VRuleInset(inset int, opts ...Option) *El {
	return Row(with([]Item{W(1), PadY(inset), VRule()}, opts)...)
}

// with appends opts to base, so a component's defaults come first and
// the caller's options override them.
func with(base []Item, opts []Option) []Item {
	for _, o := range opts {
		base = append(base, o)
	}
	return base
}

// floatRect places a floating child in the stack's inner rect.
func floatRect(element *El, in core.Rect, width, height int) core.Rect {
	switch element.fl.mode {
	case floatPoint:
		x, y := in.X+element.fl.right, in.Y+element.fl.top // floatRight holds x, floatTop holds y
		if x+width > in.X+in.W {
			x = in.X + in.W - width
		}
		if y+height > in.Y+in.H {
			y -= height // no room below: open above
		}
		return core.Rect{X: max(x, in.X), Y: max(y, in.Y), W: width, H: height}
	case floatMiddle:
		r := core.Rect{X: in.X + (in.W-width)/2, Y: in.Y + (in.H-height)/2, W: width, H: height}
		if p := element.fl.moved; p != nil {
			x, y := min(max(r.X+p.X, in.X), in.X+in.W-width), min(max(r.Y+p.Y, in.Y), in.Y+in.H-height)
			p.X, p.Y = x-r.X, y-r.Y
			r.X, r.Y = x, y
		}
		return r
	}
	return core.Rect{X: in.X + in.W - element.fl.right - width, Y: in.Y + element.fl.top, W: width, H: height}
}
