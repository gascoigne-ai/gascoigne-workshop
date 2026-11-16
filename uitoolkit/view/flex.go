package view

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
func AlignItems(alignValue Align) Option {
	return func(element *El) { element.alignItems = alignValue }
}

// AlignSelf overrides the parent's AlignItems for one child.
func AlignSelf(alignValue Align) Option { return func(element *El) { element.alignSelf = alignValue } }

// JustifyContent distributes free space along the main axis; it does
// nothing when a child grows, since that child takes the space.
func JustifyContent(justifyValue Justify) Option {
	return func(element *El) { element.justify = justifyValue }
}

// MinW, MaxW, MinH and MaxH bound an element's size, whatever its
// content or its share of the free space would give.
func MinW(count int) Option { return func(element *El) { element.minW = count } }
func MaxW(count int) Option { return func(element *El) { element.maxW = count } }
func MinH(count int) Option { return func(element *El) { element.minH = count } }
func MaxH(count int) Option { return func(element *El) { element.maxH = count } }

// Margin is space around an element, inside its parent's layout.
func Margin(count int) Option {
	return func(element *El) { element.mL, element.mT, element.mR, element.mB = count, count, count, count }
}

// MarginL, MarginT, MarginR and MarginB are margin on one side.
func MarginL(count int) Option { return func(element *El) { element.mL = count } }
func MarginT(count int) Option { return func(element *El) { element.mT = count } }
func MarginR(count int) Option { return func(element *El) { element.mR = count } }
func MarginB(count int) Option { return func(element *El) { element.mB = count } }

// MarginX is left and right margin.
func MarginX(count int) Option { return func(element *El) { element.mL, element.mR = count, count } }

// MarginY is top and bottom margin.
func MarginY(count int) Option { return func(element *El) { element.mT, element.mB = count, count } }

// clampSize applies MinW/MaxW/MinH/MaxH.
func (e *El) clampSize(width, height int) (int, int) {
	if e.minW > 0 {
		width = max(width, e.minW)
	}
	if e.maxW > 0 {
		width = min(width, e.maxW)
	}
	if e.minH > 0 {
		height = max(height, e.minH)
	}
	if e.maxH > 0 {
		height = min(height, e.maxH)
	}
	return width, height
}
