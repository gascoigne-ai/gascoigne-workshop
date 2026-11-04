package vml

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
func (e *El) size(c *core.Ctx, maxW int) (w, h int) {
	w, h = e.b.measure(e, c, maxW)
	if e.w > 0 {
		w = e.w
	}
	if e.h > 0 {
		h = e.h
	}
	return e.clampSize(w, h)
}

func (e *El) place(c *core.Ctx, r core.Rect) {
	e.rect = r
	e.b.layout(e, c)
}

// flow lays children out along one axis: a Row (horizontal) or a Col.
type flow struct {
	plain
	horiz bool
}

func (f flow) main(w, h int) int {
	if f.horiz {
		return w
	}
	return h
}

func (f flow) cross(w, h int) int {
	if f.horiz {
		return h
	}
	return w
}

// Margins along the main axis (total and leading) and across it.
func (f flow) marginMain(k *El) int {
	if f.horiz {
		return k.mL + k.mR
	}
	return k.mT + k.mB
}

func (f flow) marginLead(k *El) int {
	if f.horiz {
		return k.mL
	}
	return k.mT
}

func (f flow) marginCross(k *El) int {
	if f.horiz {
		return k.mT + k.mB
	}
	return k.mL + k.mR
}

func (f flow) marginCrossLead(k *El) int {
	if f.horiz {
		return k.mT
	}
	return k.mL
}

// baseline is the distance from a child's top to its text baseline; a
// child that is not text has its bottom edge as baseline.
func (f flow) baseline(k *El, c *core.Ctx, hugCross int) int {
	if _, isText := k.b.(textB); isText {
		return c.Ascent(k.face(c))
	}
	return hugCross
}

func (f flow) measure(e *El, c *core.Ctx, maxW int) (int, int) {
	inW := max(maxW-e.padL-e.padR, 0)
	var main, cross, n, above, below int
	for _, k := range e.kids {
		if k.hidden || k.fl.on {
			continue
		}
		w, h := k.size(c, inW)
		main += f.main(w, h) + f.marginMain(k)
		cross = max(cross, f.cross(w, h)+f.marginCross(k))
		if e.alignItems == AlignBaseline && f.horiz {
			base := f.baseline(k, c, h)
			above, below = max(above, base), max(below, h-base)
		}
		n++
	}
	if above+below > cross {
		cross = above + below // a baseline-aligned row is as tall as its lines stacked on one baseline
	}
	if n > 1 {
		main += (n - 1) * e.gap
	}
	if f.horiz {
		return main + e.padL + e.padR, cross + e.padT + e.padB
	}
	return cross + e.padL + e.padR, main + e.padT + e.padB
}

func (f flow) layout(e *El, c *core.Ctx) {
	in := e.inner()
	avail, crossAvail := f.main(in.W, in.H), f.cross(in.W, in.H)

	// Fixed and content-sized children take their size first (margins
	// included); the rest is shared by the growing ones.
	sizes, hug := make([]int, len(e.kids)), make([]int, len(e.kids))
	for i, k := range e.kids {
		w, h := k.size(c, in.W)
		sizes[i], hug[i] = f.main(w, h)+f.marginMain(k), f.cross(w, h)
	}
	e.dropOptional(f, sizes, avail)

	used, growSum, n := 0, 0.0, 0
	for i, k := range e.kids {
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
	free := max(avail-used-max(n-1, 0)*e.gap, 0)

	pos, extra := f.main(in.X, in.Y), 0
	if growSum == 0 {
		pos, extra = f.justified(e, pos, free, n)
	}
	baseMax, baseBelow := 0, 0 // the line of baseline-aligned children: above and below the baseline
	if e.alignItems == AlignBaseline && f.horiz {
		for i, k := range e.kids {
			if !k.hidden && !k.fl.on {
				base := f.baseline(k, c, hug[i])
				baseMax, baseBelow = max(baseMax, base), max(baseBelow, hug[i]-base)
			}
		}
	}
	lineTop := max((crossAvail-baseMax-baseBelow)/2, 0) // the line is centered in the row

	lastGrow, given := -1, 0
	for i, k := range e.kids {
		if !k.hidden && !k.fl.on && f.growing(k) {
			lastGrow = i
		}
	}
	for i, k := range e.kids {
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
		crossSize, crossOff := f.crossPlacement(e, k, c, crossAvail, hug[i], baseMax, lineTop)
		k.place(c, f.rect(in, pos+f.marginLead(k), crossOff, size, crossSize))
		pos += outer + e.gap + extra
	}
}

// growing reports whether a child takes a share of the free space.
func (f flow) growing(k *El) bool {
	fixed := k.h
	if f.horiz {
		fixed = k.w
	}
	return fixed == 0 && k.grow > 0
}

func (f flow) clampMain(k *El, size int) int {
	if f.horiz {
		w, _ := k.clampSize(size, 0)
		return w
	}
	_, h := k.clampSize(0, size)
	return h
}

// justified returns the start position and the extra space after each
// child for the container's JustifyContent.
func (f flow) justified(e *El, pos, free, n int) (start, extra int) {
	switch e.justify {
	case JustifyCenter:
		return pos + free/2, 0
	case JustifyEnd:
		return pos + free, 0
	case JustifyBetween:
		if n > 1 {
			return pos, free / (n - 1)
		}
	case JustifyAround:
		if n > 0 {
			return pos + free/n/2, free / n
		}
	}
	return pos, 0
}

// crossPlacement is a child's size across the axis and its offset from
// the start of the container's inner area, margins excluded from the
// size and included in the offset by the caller.
func (f flow) crossPlacement(e *El, k *El, c *core.Ctx, crossAvail, hug, baseMax, lineTop int) (size, off int) {
	avail := max(crossAvail-f.marginCross(k), 0)
	fixed := k.w
	if f.horiz {
		fixed = k.h
	}
	align := k.alignSelf
	if align == AlignDefault {
		align = e.alignItems
	}
	clamp := func(n int) int {
		if f.horiz {
			_, h := k.clampSize(0, n)
			return h
		}
		w, _ := k.clampSize(n, 0)
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
		off = lineTop + baseMax - f.baseline(k, c, hug)
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
	return size, f.marginCrossLead(k) + off
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
func (e *El) dropOptional(f flow, sizes []int, avail int) {
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

// stack layers its children over each other, each filling the element
// unless it floats.
type stack struct{ plain }

func (stack) measure(e *El, c *core.Ctx, maxW int) (int, int) {
	var w, h int
	for _, k := range e.kids {
		if k.fl.on {
			continue
		}
		kw, kh := k.size(c, maxW)
		w, h = max(w, kw), max(h, kh)
	}
	return w + e.padL + e.padR, h + e.padT + e.padB
}

func (stack) layout(e *El, c *core.Ctx) {
	in := e.inner()
	for _, k := range e.kids {
		if !k.fl.on {
			k.place(c, in)
			continue
		}
		w, h := k.size(c, in.W)
		w, h = min(w, in.W), min(h, in.H)
		k.place(c, floatRect(k, in, w, h))
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

// HRule is a one-pixel horizontal line in the border color.
func HRule() *El { return newEl(plain{}, []Item{H(1), Bg(core.Border)}) }

// VRule is a one-pixel vertical line in the border color.
func VRule() *El { return newEl(plain{}, []Item{W(1), Bg(core.Border)}) }

// VRuleInset is a vertical line shortened by inset pixels at both ends,
// for dividing the chips of a bar.
func VRuleInset(inset int) *El { return Row(W(1), PadY(inset), VRule()) }

// with appends opts to base, so a component's defaults come first and
// the caller's options override them.
func with(base []Item, opts []Option) []Item {
	for _, o := range opts {
		base = append(base, o)
	}
	return base
}

// floatRect places a floating child in the stack's inner rect.
func floatRect(k *El, in core.Rect, w, h int) core.Rect {
	switch k.fl.mode {
	case floatPoint:
		x, y := in.X+k.fl.right, in.Y+k.fl.top // floatRight holds x, floatTop holds y
		if x+w > in.X+in.W {
			x = in.X + in.W - w
		}
		if y+h > in.Y+in.H {
			y -= h // no room below: open above
		}
		return core.Rect{X: max(x, in.X), Y: max(y, in.Y), W: w, H: h}
	case floatMiddle:
		return core.Rect{X: in.X + (in.W-w)/2, Y: in.Y + (in.H-h)/2, W: w, H: h}
	}
	return core.Rect{X: in.X + in.W - k.fl.right - w, Y: in.Y + k.fl.top, W: w, H: h}
}
