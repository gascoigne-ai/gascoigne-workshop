package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// plain is a behavior with no content of its own.
type plain struct{}

func (plain) measure(*El, *widget.Ctx, int) (int, int) { return 0, 0 }
func (plain) layout(*El, *widget.Ctx)                  {}
func (plain) draw(*El, *widget.Ctx, *ebiten.Image)     {}
func (plain) update(*El, *widget.Ctx)                  {}

// inner is the rect inside the padding.
func (e *El) inner() widget.Rect {
	r := e.rect
	return widget.Rect{
		X: r.X + e.padL, Y: r.Y + e.padT,
		W: max(r.W-e.padL-e.padR, 0), H: max(r.H-e.padT-e.padB, 0),
	}
}

// size is the element's wanted size: its fixed size where set, else
// what its behavior measures.
func (e *El) size(c *widget.Ctx, maxW int) (w, h int) {
	w, h = e.b.measure(e, c, maxW)
	if e.w > 0 {
		w = e.w
	}
	if e.h > 0 {
		h = e.h
	}
	return w, h
}

func (e *El) place(c *widget.Ctx, r widget.Rect) {
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

func (f flow) measure(e *El, c *widget.Ctx, maxW int) (int, int) {
	inW := max(maxW-e.padL-e.padR, 0)
	var main, cross, n int
	for _, k := range e.kids {
		if k.hidden || k.float {
			continue
		}
		w, h := k.size(c, inW)
		main += f.main(w, h)
		cross = max(cross, f.cross(w, h))
		n++
	}
	if n > 1 {
		main += (n - 1) * e.gap
	}
	if f.horiz {
		return main + e.padL + e.padR, cross + e.padT + e.padB
	}
	return cross + e.padL + e.padR, main + e.padT + e.padB
}

func (f flow) layout(e *El, c *widget.Ctx) {
	in := e.inner()
	avail := f.main(in.W, in.H)

	// Fixed and content-sized children take their size first; the rest
	// is shared by the growing ones.
	sizes := make([]int, len(e.kids))
	for i, k := range e.kids {
		w, h := k.size(c, in.W)
		sizes[i] = f.main(w, h)
	}
	e.dropOptional(f, sizes, avail)

	used, growSum, n := 0, 0.0, 0
	for i, k := range e.kids {
		if k.hidden || k.float {
			continue
		}
		n++
		if f.fixedMain(k) == 0 && k.grow > 0 {
			growSum += k.grow
			continue
		}
		used += sizes[i]
	}
	free := max(avail-used-max(n-1, 0)*e.gap, 0)

	pos := f.main(in.X, in.Y)
	lastGrow := -1
	for i, k := range e.kids {
		if !k.hidden && !k.float && f.fixedMain(k) == 0 && k.grow > 0 {
			lastGrow = i
		}
	}
	given := 0
	for i, k := range e.kids {
		if k.hidden || k.float {
			k.rect = widget.Rect{}
			continue
		}
		size := sizes[i]
		if f.fixedMain(k) == 0 && k.grow > 0 {
			size = int(float64(free) * k.grow / growSum)
			given += size
			if i == lastGrow {
				size += free - given // rounding remainder
			}
		}
		k.place(c, f.rectFor(in, k, pos, size, c))
		pos += size + e.gap
	}
}

func (f flow) fixedMain(k *El) int {
	if f.horiz {
		return k.w
	}
	return k.h
}

// rectFor builds a child's rect: size along the main axis, and across
// it either its fixed size or the full inner extent.
func (f flow) rectFor(in widget.Rect, k *El, pos, size int, c *widget.Ctx) widget.Rect {
	if f.horiz {
		h := in.H
		if k.h > 0 {
			h = k.h
		}
		return widget.Rect{X: pos, Y: in.Y + (in.H-h)/2, W: size, H: h}
	}
	w := in.W
	if k.w > 0 {
		w = k.w
	}
	return widget.Rect{X: in.X, Y: pos, W: w, H: size}
}

// dropOptional hides optional children when everything does not fit.
func (e *El) dropOptional(f flow, sizes []int, avail int) {
	total, n := 0, 0
	for i, k := range e.kids {
		k.hidden = false
		if k.float {
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

func (stack) measure(e *El, c *widget.Ctx, maxW int) (int, int) {
	var w, h int
	for _, k := range e.kids {
		if k.float {
			continue
		}
		kw, kh := k.size(c, maxW)
		w, h = max(w, kw), max(h, kh)
	}
	return w + e.padL + e.padR, h + e.padT + e.padB
}

func (stack) layout(e *El, c *widget.Ctx) {
	in := e.inner()
	for _, k := range e.kids {
		if !k.float {
			k.place(c, in)
			continue
		}
		w, h := k.size(c, in.W)
		w, h = min(w, in.W), min(h, in.H)
		k.place(c, widget.Rect{X: in.X + in.W - k.floatRight - w, Y: in.Y + k.floatTop, W: w, H: h})
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
func HRule() *El { return newEl(plain{}, []Item{H(1), Bg(widget.Border)}) }

// VRule is a one-pixel vertical line in the border color.
func VRule() *El { return newEl(plain{}, []Item{W(1), Bg(widget.Border)}) }

// VRuleInset is a vertical line shortened by inset pixels at both ends,
// for dividing the chips of a bar.
func VRuleInset(inset int) *El { return Row(W(1), PadXY(0, inset), VRule()) }

// with appends opts to base, so a component's defaults come first and
// the caller's options override them.
func with(base []Item, opts []Option) []Item {
	for _, o := range opts {
		base = append(base, o)
	}
	return base
}
