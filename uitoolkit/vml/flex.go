package vml

// Align places a child across its parent's axis (vertically in a Row,
// horizontally in a Col).
type Align int

const (
	AlignDefault  Align = iota // stretch; a child with a fixed cross size is centered in a Row and left-aligned in a Col
	AlignStart                 // top / left, at the child's own size
	AlignCenter                // centered, at the child's own size
	AlignEnd                   // bottom / right, at the child's own size
	AlignStretch               // fills the cross axis unless it has a fixed cross size, then it is at the start
	AlignBaseline              // in a Row: text baselines line up
)

// Justify spreads free main-axis space among children that do not grow.
type Justify int

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifyBetween // free space between the children
	JustifyAround  // free space around each child
)

// AlignItems sets how a Row or Col places its children across it.
func AlignItems(a Align) Option { return func(e *El) { e.alignItems = a } }

// AlignSelf overrides the parent's AlignItems for one child.
func AlignSelf(a Align) Option { return func(e *El) { e.alignSelf = a } }

// JustifyContent distributes free space along the main axis; it does
// nothing when a child grows, since that child takes the space.
func JustifyContent(j Justify) Option { return func(e *El) { e.justify = j } }

// MinW, MaxW, MinH and MaxH bound an element's size, whatever its
// content or its share of the free space would give.
func MinW(n int) Option { return func(e *El) { e.minW = n } }
func MaxW(n int) Option { return func(e *El) { e.maxW = n } }
func MinH(n int) Option { return func(e *El) { e.minH = n } }
func MaxH(n int) Option { return func(e *El) { e.maxH = n } }

// Margin is space around an element, inside its parent's layout.
func Margin(n int) Option { return func(e *El) { e.mL, e.mT, e.mR, e.mB = n, n, n, n } }

// MarginL, MarginT, MarginR and MarginB are margin on one side.
func MarginL(n int) Option { return func(e *El) { e.mL = n } }
func MarginT(n int) Option { return func(e *El) { e.mT = n } }
func MarginR(n int) Option { return func(e *El) { e.mR = n } }
func MarginB(n int) Option { return func(e *El) { e.mB = n } }

// MarginX is left and right margin.
func MarginX(n int) Option { return func(e *El) { e.mL, e.mR = n, n } }

// MarginY is top and bottom margin.
func MarginY(n int) Option { return func(e *El) { e.mT, e.mB = n, n } }

// clampSize applies MinW/MaxW/MinH/MaxH.
func (e *El) clampSize(w, h int) (int, int) {
	if e.minW > 0 {
		w = max(w, e.minW)
	}
	if e.maxW > 0 {
		w = min(w, e.maxW)
	}
	if e.minH > 0 {
		h = max(h, e.minH)
	}
	if e.maxH > 0 {
		h = min(h, e.maxH)
	}
	return w, h
}
