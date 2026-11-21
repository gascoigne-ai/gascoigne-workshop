package core

// Sizes are the control heights and widths for one pair of UI line heights:
// the rules of style.md (Sizes) applied to the faces the Ctx really draws
// with, so a text size the user picks never clips a row. Ctx.Sizes holds
// the value; widgets read it every frame and nothing writes it but
// NewCtx, SetTheme and SetFonts.
//
// Why a struct of fields: layout reads it every frame and it changes only
// with the theme or fonts. Rejected: a map keyed by scope name (untyped,
// and a typo compiles). Revisit when a pane needs sizes per face (rows in
// the mono face): add fields.
type Sizes struct {
	Line, SmallLine int // line heights of the UI and the small face

	Row, Compact, Slim, Label, Note int // height classes, see theme.go

	Toolbar, TabBar, DialogTitle, DialogToolbar int // bars, built from Row and a step

	IconBtnW, WordBtnW int // widths of fixed-shape buttons
	Checkbox           int // the side of a checkbox
}

// The UI line height is limited so a huge font cannot blow up every row.
const (
	MinLine = 13
	MaxLine = 22
)

// SizesFor applies the size rules to a UI line height, clamped to
// MinLine..MaxLine, and a small line height, clamped to 1..line.
func SizesFor(line, small int) Sizes {
	line = min(max(line, MinLine), MaxLine)
	small = min(max(small, 1), line)
	row := line + 6*Unit
	compact := line + 4*Unit
	iconBtnW := compact + 2*NearGap
	return Sizes{
		Line:          line,
		SmallLine:     small,
		Row:           row,
		Compact:       compact,
		Slim:          line + 2*Unit,
		Label:         line + 3*Unit,
		Note:          small + 2*Unit,
		Toolbar:       row + 2*NearGap,
		TabBar:        row + 2*TightGap,
		DialogTitle:   row + 2*LooseGap,
		DialogToolbar: row + 2*TightGap,
		IconBtnW:      iconBtnW,
		WordBtnW:      iconBtnW + StepGutter,
		Checkbox:      small,
	}
}

// DefaultSizes are the sizes of the default faces; they equal the
// constants RowH, CompactH and the rest, which stay as documented defaults
// until every consumer reads Ctx.Sizes.
func DefaultSizes() Sizes { return SizesFor(BaseLine, BaseSmallLine) }

// Scope is the sizes and spacings under the names templates use, in one
// place. Only the heights and widths follow the fonts; the spacings are
// steps of Unit and never move.
func (s Sizes) Scope() map[string]any {
	return map[string]any{
		"rowH": s.Row, "compactH": s.Compact, "slimH": s.Slim, "labelH": s.Label, "noteH": s.Note,
		"toolbarH": s.Toolbar, "tabBarH": s.TabBar, "dialogToolbarH": s.DialogToolbar, "dialogTitleH": s.DialogTitle,
		"iconBtnW": s.IconBtnW, "wordBtnW": s.WordBtnW, "checkboxSize": s.Checkbox,

		"tabLead": TabLead, "edgeW": EdgeW, "nearGap": NearGap, "shadowOffset": ShadowOffset,
		"tightGap": TightGap, "tabGap": TabGap, "ruleInset": RuleInset, "edgeGrab": EdgeGrab,
		"looseGap": LooseGap, "cellPad": CellPad, "labelPad": LabelPad, "tabSepInset": TabSepInset,
		"gutter": Gutter, "dialogPad": DialogPad,
	}
}
