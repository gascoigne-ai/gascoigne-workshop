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
func AlignItems(align Align) Option {
	return func(element *El) { element.alignItems = align }
}

// AlignSelf overrides the parent's AlignItems for one child.
func AlignSelf(align Align) Option { return func(element *El) { element.alignSelf = align } }

// JustifyContent distributes free space along the main axis; it does
// nothing when a child grows, since that child takes the space.
func JustifyContent(justify Justify) Option {
	return func(element *El) { element.justify = justify }
}

// MinW, MaxW, MinH and MaxH bound an element's size, whatever its
// content or its share of the free space would give.
func MinW(pixels int) Option { return func(element *El) { element.minW = pixels } }

// MaxW is the widest the element may be.
func MaxW(pixels int) Option { return func(element *El) { element.maxW = pixels } }

// MinH is the least height the element may have.
func MinH(pixels int) Option { return func(element *El) { element.minH = pixels } }

// MaxH is the tallest the element may be.
func MaxH(pixels int) Option { return func(element *El) { element.maxH = pixels } }

// Margin is space around an element, inside its parent's layout.
func Margin(pixels int) Option {
	return func(element *El) { element.mL, element.mT, element.mR, element.mB = pixels, pixels, pixels, pixels }
}

// MarginL is margin on the left side.
func MarginL(pixels int) Option { return func(element *El) { element.mL = pixels } }

// MarginT is margin on the top side.
func MarginT(pixels int) Option { return func(element *El) { element.mT = pixels } }

// MarginR is margin on the right side.
func MarginR(pixels int) Option { return func(element *El) { element.mR = pixels } }

// MarginB is margin on the bottom side.
func MarginB(pixels int) Option { return func(element *El) { element.mB = pixels } }

// MarginX is left and right margin.
func MarginX(pixels int) Option { return func(element *El) { element.mL, element.mR = pixels, pixels } }

// MarginY is top and bottom margin.
func MarginY(pixels int) Option { return func(element *El) { element.mT, element.mB = pixels, pixels } }

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
