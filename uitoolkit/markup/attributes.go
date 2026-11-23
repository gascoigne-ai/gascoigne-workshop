package markup

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// attributeReader turns an attribute value into an option.
type attributeReader func(value string) (view.Option, error)

// splitOptions are the attributes only a split takes: their options
// configure the split behavior and would panic on another element.
var splitOptions = map[string]attributeReader{
	"vertical":   flagAttr("vertical", widget.Vertical),
	"as-ratio":   flagAttr("as-ratio", widget.AsRatio),
	"min-first":  intAttr(widget.MinFirst),
	"min-second": intAttr(widget.MinSecond),
}

// fieldOptions are the attributes only a field takes: their options set
// the field behavior and would panic on another element.
var fieldOptions = map[string]attributeReader{
	"placeholder": func(value string) (view.Option, error) { return widget.Placeholder(value), nil },
	"field-pad":   intAttr(widget.FieldPad),
}

// fieldHandlers are the field's own attributes that name a scope entry:
// its key handlers, the function that colors its text (styled) and the TextField
// its right-click menu edits.
var fieldHandlers = map[string]handlerAttr{
	"on-enter":  handler(widget.OnEnter),
	"on-tab":    handler(widget.OnTab),
	"styled":    handler(styledOption),
	"text-menu": stateHandler(widget.TextMenu),
}

// areaHandlers are the area's own attributes that name a scope entry: the
// TextField its right-click menu edits (the area's own, &area.TextField).
var areaHandlers = map[string]handlerAttr{
	"text-menu": stateHandler(widget.TextMenu),
}

// wrapOptions are the attributes only a code view and an area take.
var wrapOptions = map[string]attributeReader{
	"wrap": flagAttr("wrap", widget.Wrap),
}

// sliderOptions are the attributes only a slider takes.
var sliderOptions = map[string]attributeReader{
	"slider-mid": intAttr(widget.SliderMid),
}

// sliderHandlers get the slider's value while the thumb moves and when it
// is released.
var sliderHandlers = map[string]handlerAttr{
	"on-slide":     handler(widget.OnSlide),
	"on-slide-end": handler(widget.OnSlideEnd),
}

// handlerAttr binds an attribute to a function or value of the scope: the
// attribute value names it, and it must have the type of the Go option's
// argument. bind's error says what is wrong with the value.
type handlerAttr struct {
	bind func(value any) (view.Option, error)
}

func handler[F any](option func(F) view.Option) handlerAttr {
	want := fmt.Sprintf("%T", *new(F))
	return handlerAttr{
		bind: func(value any) (view.Option, error) {
			f, ok := value.(F)
			if !ok {
				return nil, fmt.Errorf("is not a %s", want)
			}
			return option(f), nil
		},
	}
}

// stateHandler is handler for an option that takes state the widget
// writes to (text-menu's *widget.TextField): a nil *T has the right type
// but would panic on first use, so it is rejected, as scopedState does.
func stateHandler[T any](option func(*T) view.Option) handlerAttr {
	typed := handler(option)
	want := fmt.Sprintf("%T", (*T)(nil))
	return handlerAttr{
		bind: func(value any) (view.Option, error) {
			bound, err := typed.bind(value)
			if err != nil {
				return nil, err
			}
			if value.(*T) == nil {
				return nil, fmt.Errorf("is a nil %s", want)
			}
			return bound, nil
		},
	}
}

// handlers are the generic attributes whose value names a scope entry,
// each the Go option in kebab-case: the events, and font-source, a
// *text.GoTextFaceSource that may be nil (the interface font).
var handlers = map[string]handlerAttr{
	"font-source":       handler(view.FontSource),
	"on-click":          handler(view.OnClick),
	"on-click-at":       handler(view.OnClickAt),
	"on-right-click":    handler(view.OnRightClick),
	"on-right-click-in": handler(view.OnRightClickIn),
	"on-middle-click":   handler(view.OnMiddleClick),
	"on-double-click":   handler(view.OnDoubleClick),
	"on-drag-over":      handler(view.OnDragOver),
	"on-change":         handler(view.OnChange),
	"on-select":         handler(view.OnSelect),
	"on-dismiss":        handler(view.OnDismiss),
	"moved":             stateHandler(view.Moved),
}

// attributes turn an attribute value (already interpolated) into an
// option. A flag is written bare (`small`) or `small="true"`; "false" is
// off. An option may be nil, meaning nothing to apply.
var attributes = map[string]func(value string) (view.Option, error){
	"w":                     intAttr(view.W),
	"h":                     intAttr(view.H),
	"gap":                   intAttr(view.Gap),
	"grow":                  floatAttr(view.Grow),
	"align-items":           enumAttr("align-items", alignments, view.AlignItems),
	"align-self":            enumAttr("align-self", alignments, view.AlignSelf),
	"justify-content":       enumAttr("justify-content", justifications, view.JustifyContent),
	"pad":                   padAttr,
	"pad-x":                 intAttr(view.PadX),
	"pad-y":                 intAttr(view.PadY),
	"pad-l":                 intAttr(view.PadL),
	"pad-t":                 intAttr(view.PadT),
	"pad-r":                 intAttr(view.PadR),
	"pad-b":                 intAttr(view.PadB),
	"margin":                intAttr(view.Margin),
	"margin-x":              intAttr(view.MarginX),
	"margin-y":              intAttr(view.MarginY),
	"margin-l":              intAttr(view.MarginL),
	"margin-t":              intAttr(view.MarginT),
	"margin-r":              intAttr(view.MarginR),
	"margin-b":              intAttr(view.MarginB),
	"bg":                    tokenAttr(view.Bg),
	"border":                tokenAttr(view.Border),
	"border-t":              tokenAttr(view.BorderT),
	"border-b":              tokenAttr(view.BorderB),
	"border-l":              tokenAttr(view.BorderL),
	"border-inset":          intAttr(view.BorderInset),
	"hover-bg":              tokenAttr(view.HoverBg),
	"hover-color":           tokenAttr(view.HoverColor),
	"min-w":                 intAttr(view.MinW),
	"max-w":                 intAttr(view.MaxW),
	"min-h":                 intAttr(view.MinH),
	"max-h":                 intAttr(view.MaxH),
	"shadow":                flagAttr("shadow", view.Shadow),
	"clip":                  flagAttr("clip", view.Clip),
	"modal":                 flagAttr("modal", view.Modal),
	"keep-on-outside-click": flagAttr("keep-on-outside-click", view.KeepOnOutsideClick),
	"float-center":          flagAttr("float-center", view.FloatCenter),
	"drag-handle":           flagAttr("drag-handle", view.DragHandle),
	"float-at":              intsAttr("float-at", 2, func(numbers []int) view.Option { return view.FloatAt(numbers[0], numbers[1]) }),
	"float-tr":              intsAttr("float-tr", 2, func(numbers []int) view.Option { return view.FloatTR(numbers[0], numbers[1]) }),
	"edge-t":                tokenAttr(view.EdgeT),
	"edge-l":                tokenAttr(view.EdgeL),
	"edge-r":                tokenAttr(view.EdgeR),
	"edge-b":                tokenAttr(view.EdgeB),
	"border-r":              tokenAttr(view.BorderR),
	"border-x":              tokenAttr(view.BorderX),
	"border-y":              tokenAttr(view.BorderY),
	"color":                 tokenAttr(view.Color),
	"control":               func(value string) (view.Option, error) { return view.Control(value), nil },
	"round-t":               func(value string) (view.Option, error) { return view.RoundT(value), nil },
	"round-b":               func(value string) (view.Option, error) { return view.RoundB(value), nil },
	"round-bl":              func(value string) (view.Option, error) { return view.RoundBL(value), nil },
	"tooltip":               func(value string) (view.Option, error) { return view.Tooltip(value), nil },
	"small":                 flagAttr("small", view.Small),
	"muted":                 flagAttr("muted", view.Muted),
	"bold":                  flagAttr("bold", view.Bold),
	"mono":                  flagAttr("mono", view.Mono),
	"ellipsis":              flagAttr("ellipsis", view.Ellipsis),
	"text-center":           flagAttr("text-center", view.TextCenter),
	"optional":              flagAttr("optional", view.Optional),
	"shrink":                flagAttr("shrink", view.Shrink),
	"lighten":               flagAttr("lighten", view.Lighten),
	"keep-in-view":          flagAttr("keep-in-view", view.KeepInView),
	"separator":             separatorAttr,
	"active-tab":            flagAttr("active-tab", widget.ActiveTab),
	"focus-scope":           flagAttr("focus-scope", view.FocusScope),
	"dismiss-margin":        intAttr(view.DismissMargin),
}

func intAttr(callback func(int) view.Option) func(string) (view.Option, error) {
	return func(value string) (view.Option, error) {
		number, err := wholeNumber(value)
		if err != nil {
			return nil, err
		}
		return callback(number), nil
	}
}

func floatAttr(callback func(float64) view.Option) func(string) (view.Option, error) {
	return func(value string) (view.Option, error) {
		n, err := finiteFloat(value)
		if err != nil {
			return nil, err
		}
		return callback(n), nil
	}
}

// tokenAttr applies no option for an empty token, so a template switches
// a color off per item (border="{{dropBorder}}") instead of Go patching
// the built node or a second template.
func tokenAttr(callback func(core.Token) view.Option) func(string) (view.Option, error) {
	return func(value string) (view.Option, error) {
		token := strings.TrimSpace(value)
		if token == "" {
			return nil, nil
		}
		return callback(core.Token(token)), nil
	}
}

func flagAttr(name string, callback func() view.Option) func(string) (view.Option, error) {
	return func(value string) (view.Option, error) {
		if bare := strings.TrimSpace(value); bare == "" || bare == name { // a bare attribute reads as its own name
			return callback(), nil
		}
		on, err := truth(value)
		if err != nil || !on {
			return nil, err
		}
		return callback(), nil
	}
}

// separatorAttr applies no option for 0, so a tab strip puts
// separator="{{inset}}" on every tab and gives the first 0. Go's
// Separator(0) still draws a full-height line.
func separatorAttr(value string) (view.Option, error) {
	inset, err := wholeNumber(value)
	if err != nil {
		return nil, err
	}
	if inset == 0 {
		return nil, nil
	}
	return view.Separator(inset), nil
}

// padAttr is `pad="12"` for all sides or `pad="12 4"` for horizontal and vertical.
func padAttr(value string) (view.Option, error) {
	f := strings.Fields(value)
	nums := make([]int, len(f))
	for i, s := range f {
		number, err := wholeNumber(s)
		if err != nil {
			return nil, err
		}
		nums[i] = number
	}
	switch len(nums) {
	case 1:
		return view.Pad(nums[0]), nil
	case 2:
		x, y := nums[0], nums[1]
		return func(element *view.El) { view.PadX(x)(element); view.PadY(y)(element) }, nil
	}
	return nil, fmt.Errorf("pad takes one or two numbers, not %q", value)
}

// intsAttr reads exactly n whole numbers separated by spaces.
func intsAttr(name string, count int, callback func([]int) view.Option) func(string) (view.Option, error) {
	return func(value string) (view.Option, error) {
		fields := strings.Fields(value)
		if len(fields) != count {
			return nil, fmt.Errorf("%s takes %d numbers, not %q", name, count, value)
		}
		nums := make([]int, count)
		for i, s := range fields {
			number, err := wholeNumber(s)
			if err != nil {
				return nil, err
			}
			nums[i] = number
		}
		return callback(nums), nil
	}
}

var alignments = map[string]view.Align{
	"default": view.AlignDefault, "start": view.AlignStart, "center": view.AlignCenter,
	"end": view.AlignEnd, "stretch": view.AlignStretch, "baseline": view.AlignBaseline,
}

var justifications = map[string]view.Justify{
	"start": view.JustifyStart, "center": view.JustifyCenter, "end": view.JustifyEnd,
	"between": view.JustifyBetween, "around": view.JustifyAround,
}

// enumAttr reads one word out of a table of names.
func enumAttr[T any](name string, table map[string]T, callback func(T) view.Option) func(string) (view.Option, error) {
	return func(value string) (view.Option, error) {
		val, ok := table[strings.TrimSpace(value)]
		if !ok {
			names := slices.Sorted(maps.Keys(table))
			return nil, fmt.Errorf("%s is one of %s, not %q", name, strings.Join(names, ", "), value)
		}
		return callback(val), nil
	}
}

// styledOption reads the styled attribute's scope function, a plain
// func(string) []widget.Span.
func styledOption(styler func(text string) []widget.Span) view.Option {
	return widget.Styled(styler)
}
