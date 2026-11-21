package core

import (
	"image/color"
	"maps"
)

// Token names a color role. A Theme maps tokens to colors, so switching
// theme recolors everything. Applications add their own tokens (for
// example "method.GET") with Theme.With.
type Token string

// Core tokens, from back to front: Frame is the window chrome, Bg the
// raised panes, Sunken lists and code, Alt zebra rows, Selected the
// current row.
const (
	Bg           Token = "bg"
	Frame        Token = "frame"
	Panel        Token = "panel"
	Sunken       Token = "sunken"
	Alt          Token = "alt"
	Border       Token = "border"
	Hover        Token = "hover"
	Selected     Token = "selected"
	Chip         Token = "chip"
	ChipText     Token = "chip-text"
	Rule         Token = "rule"
	Text         Token = "text"
	Muted        Token = "muted"
	Accent       Token = "accent"
	AccentLine   Token = "accent-line" // the accent as a line: outlines, underlines, marks
	AccentText   Token = "accent-text"
	OK           Token = "ok"
	Error        Token = "error"
	Shadow       Token = "shadow"
	Selection    Token = "selection"
	FindMatch    Token = "find-match"
	FindActive   Token = "find-active"
	BorderSoft   Token = "border-soft"   // hairlines between table rows, as the dashboard draws them
	BorderStrong Token = "border-strong" // control outlines (buttons, fields, menus)
	Info         Token = "info"
	InfoSoft     Token = "info-soft"
	Warn         Token = "warn"
	OKSoft       Token = "ok-soft" // pale backgrounds for status pills
	WarnSoft     Token = "warn-soft"
	ErrorSoft    Token = "error-soft"
	AccentSoft   Token = "accent-soft"
	Track        Token = "track"      // the empty part of a bar
	Faint        Token = "faint"      // the quietest text
	Hero         Token = "hero"       // the banner behind the window title, as a flat fallback
	HeroText     Token = "hero-text"  // text on the banner
	HeroMuted    Token = "hero-muted" // icons on the banner
)

// Size rules shared by every component, in logical pixels. Every size is a
// rule over three bases, so a component picks a class and a step, never a
// number (style.md, Sizes). A height is a line of text plus padding on
// both sides, line + 2*pad, so text centers on whole pixels. Spacing comes
// in named steps, whole multiples of Unit.
//
// The constants below are the sizes of the default faces. The sizes that
// follow the faces really in use are Ctx.Sizes (SizesFor); the constants
// stay as the documented default until every consumer reads Ctx.Sizes,
// and TestDefaultSizesEqualConstants keeps the two equal.
const (
	BaseLine      = 16 // line height of the default UI face (Go regular at 14 px)
	BaseSmallLine = 14 // line height of the default small face (Go regular at 12 px)
	Unit          = 2  // the grid every padding and gap is a multiple of
)

// Spacing steps, from tight to wide. Each role constant below is one of
// them, so two roles on the same step stay equal when the step moves.
const (
	StepHair   = 1 * Unit
	StepNear   = 2 * Unit
	StepTight  = 3 * Unit
	StepLoose  = 4 * Unit
	StepLabel  = 5 * Unit
	StepGutter = 6 * Unit
	StepDialog = 8 * Unit
)

// Height classes, so neighbours line up: RowH for list and table rows and
// standard controls, CompactH for chips, icon buttons and inline fields
// that sit in a row of that height, LabelH and NoteH for section captions
// and one-line notes.
const (
	RowH     = BaseLine + 6*Unit
	CompactH = BaseLine + 4*Unit
	SlimH    = BaseLine + 2*Unit      // a pill, and a button in the status bar: the step below CompactH
	LabelH   = BaseLine + 3*Unit      // a section caption
	NoteH    = BaseSmallLine + 2*Unit // a one-line note in the small face
)

// Spacing roles. Gutter is the inset between a container and its content,
// and the gap between its sections. Gaps between neighbours come in three
// steps: NearGap for items that read as one, TightGap for controls side by
// side (a field and its button, a row of chips), LooseGap for the blocks
// of a pane stacked in a column and for rows whose items need more air.
const (
	TabLead      = StepHair   // room before the first tab of a strip
	EdgeW        = StepHair   // the width of an open tab's outline
	NearGap      = StepNear   // between items that read as one: a label and its count, a heading and its lines
	ShadowOffset = StepNear   // how far a shadow is shifted down and right
	TightGap     = StepTight  // controls side by side
	TabGap       = StepTight  // space above tabs, so their outline never touches the banner or bar above
	RuleInset    = StepTight  // how far a menu's separator line stays off the menu's sides
	EdgeGrab     = StepTight  // how far inside its border a press grabs an edge (resize, or no drag)
	LooseGap     = StepLoose  // blocks of a pane stacked in a column
	CellPad      = StepLoose  // the inset of a table cell and of a header cell
	LabelPad     = StepLabel  // the side inset of the label of a pill or a tab, as of a button
	TabSepInset  = StepLabel  // how far the line between two tabs stays off the strip's top and bottom
	Gutter       = StepGutter // the inset between a container and its content; the gap between its sections
	DialogPad    = StepDialog // the inset of a dialog's title bar, toolbar, footer and message body
)

// Bars and fixed-shape controls, built from a height class and a step.
const (
	ToolbarH       = RowH + 2*NearGap      // a bar of controls over a pane
	DialogToolbarH = RowH + 2*TightGap     // the row of controls under a dialog's title
	TabBarH        = RowH + 2*TightGap     // a strip of tabs, with TabGap above them
	DialogTitleH   = RowH + 2*LooseGap     // the title bar of a dialog
	IconBtnW       = CompactH + 2*NearGap  // an icon button: square room around a compact control
	WordBtnW       = IconBtnW + StepGutter // a button that holds one short word (Log, Raw, Reset)
	CheckboxSize   = BaseSmallLine         // the side of a checkbox: one small line
)

// Fixed geometry: limits on how a dialog sits in the window, not sizes of
// content, so they follow no line or step.
const (
	DialogMargin = 40  // kept free between a dialog and the window's edge
	DialogMaxH   = 560 // the tallest a dialog grows by itself
)

// Theme holds the colors and font sizes the widgets draw with.
type Theme struct {
	Colors                        map[Token]color.RGBA
	FontSize, SmallSize, MonoSize float64

	// Name is the theme's name, from its file.
	Name string

	// Controls are the per-control styles (corner radius, gradient,
	// outline) by control name; see ControlStyle and Theme.Style.
	Controls map[string]ControlStyle

	// RadiusScale multiplies every control's corner radius: 0 gives square
	// corners, 1 the theme's own.
	RadiusScale float64

	// IconStroke multiplies the stroke width of icons; 0 means 1.
	IconStroke float64

	// NoGradients turns every gradient into a flat fill.
	NoGradients bool
}

// Color returns the color of tok, or the text color for unknown tokens
// so a typo stays visible but readable.
func (t Theme) Color(tok Token) color.RGBA {
	if c, ok := t.Colors[tok]; ok {
		return c
	}
	return t.Colors[Text]
}

// With returns a copy of t with extra tokens added or replaced.
func (t Theme) With(extra map[Token]color.RGBA) Theme {
	colors := make(map[Token]color.RGBA, len(t.Colors)+len(extra))
	maps.Copy(colors, t.Colors)
	maps.Copy(colors, extra)
	t.Colors = colors
	return t
}

// WithTextSize returns a copy of t with the given font sizes in pixels.
func (t Theme) WithTextSize(font, small, mono float64) Theme {
	t.FontSize, t.SmallSize, t.MonoSize = font, small, mono
	return t
}

// Dark and Light follow the gascoigne CLI dashboard's palette
// (cmd/gascoigne/pages.css in gascoigne-cli): burgundy accent, warm
// near-black or cream surfaces. Its surface tokens map to Frame (the
// window chrome), Bg (raised panes) and Sunken (lists and code). Border,
// Rule, Chip, Hover and Selected keep the CLI's hues but are stronger
// than its soft web borders, so a native window's edges can be seen
// (see TestNonTextElementsAreVisible).

// ControlStyle is how one kind of control is drawn. The zero value is a
// flat, square control with no outline of its own.
type ControlStyle struct {
	Radius float64 // corner radius in logical pixels; larger than half the height makes a pill
	Shade  float64 // gradient strength: the fill mixes toward white above and black below
	Top    *color.RGBA
	Bottom *color.RGBA // explicit gradient ends, replacing Shade
	Border Token       // outline color token, "" for none
}

// HasGradient reports whether the control's fill is a gradient.
func (s ControlStyle) HasGradient() bool { return s.Shade > 0 || s.Top != nil }

// Style is the style of the named control with the theme's global
// switches applied: corners scaled by RadiusScale, and gradients dropped
// when NoGradients is set (the control then shows its own Bg color). A control the theme does not list is plain.
func (t Theme) Style(name string) ControlStyle {
	st := t.Controls[name]
	st.Radius *= t.RadiusScale
	if t.NoGradients {
		st.Shade = 0
		st.Top, st.Bottom = nil, nil
	}
	return st
}
