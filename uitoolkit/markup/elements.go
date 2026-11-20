package markup

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// elementParts are what an element is built from: its constructor
// arguments (the values of its params, in their order), options, children
// and text.
type elementParts struct {
	args []any
	opts []view.Option
	kids []view.Item
	text string
}

// items are the options followed by the children, as a Go constructor
// takes them.
func (parts elementParts) items() []view.Item {
	return append(view.With(make([]view.Item, 0, len(parts.opts)+len(parts.kids)), parts.opts), parts.kids...)
}

// element builds one kind of element from its parts.
type element func(parts elementParts) (view.Node, error)

// container builds an element that takes options and children.
func container(constructor func(items ...view.Item) *view.El) element {
	return func(parts elementParts) (view.Node, error) { return constructor(parts.items()...), nil }
}

// optioned builds an element that takes options only.
func optioned(constructor func(opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) { return constructor(parts.opts...), nil }
}

// labelled builds an element whose constructor takes its text first.
func labelled(constructor func(label string, opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) { return constructor(parts.text, parts.opts...), nil }
}

// The arity helpers build an element from a Go constructor by reading its
// params, in order, as the constructor's leading arguments. args<N> are the
// constructors that end in options, itemArgs<N> those that end in items
// (options and children), textArgs1 those that take the text first. The
// param types are checked when the param is read, so the assertion cannot
// fail.

// args1 builds an element from a constructor of one argument and options.
func args1[A any](constructor func(first A, opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(A), parts.opts...), nil
	}
}

// args2 is args1 for two arguments.
func args2[A, B any](constructor func(first A, second B, opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(A), parts.args[1].(B), parts.opts...), nil
	}
}

// args3 is args1 for three arguments.
func args3[A, B, C any](constructor func(first A, second B, third C, opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(A), parts.args[1].(B), parts.args[2].(C), parts.opts...), nil
	}
}

// args5 is args1 for five arguments.
func args5[A, B, C, D, E any](constructor func(first A, second B, third C, fourth D, fifth E, opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(A), parts.args[1].(B), parts.args[2].(C), parts.args[3].(D), parts.args[4].(E), parts.opts...), nil
	}
}

// itemArgs1 builds an element from a constructor of one argument and
// items: the options, then the children.
func itemArgs1[A any](constructor func(first A, items ...view.Item) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(A), parts.items()...), nil
	}
}

// itemArgs2 is itemArgs1 for two arguments.
func itemArgs2[A, B any](constructor func(first A, second B, items ...view.Item) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(A), parts.args[1].(B), parts.items()...), nil
	}
}

// itemArgs5 is itemArgs1 for five arguments.
func itemArgs5[A, B, C, D, E any](constructor func(first A, second B, third C, fourth D, fifth E, items ...view.Item) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(A), parts.args[1].(B), parts.args[2].(C), parts.args[3].(D), parts.args[4].(E), parts.items()...), nil
	}
}

// textArgs1 builds an element from a constructor that takes the text, one
// argument and options (SegmentChip, Note, Lines).
func textArgs1[A any](constructor func(text string, first A, opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.text, parts.args[0].(A), parts.opts...), nil
	}
}

// slotNode is the node the caller put in the scope, as a Vue slot. It is
// read by its name param, a path, so each item of a for can bring its own
// node.
func slotNode(parts elementParts) (view.Node, error) {
	return parts.args[0].(view.Node), nil
}

// knownIcon is the error for an icon name widget.Icon would panic on.
func knownIcon(name string) error {
	if widget.HasIcon(name) {
		return nil
	}
	return fmt.Errorf("no icon %q; icons: %s", name, strings.Join(widget.IconNames(), " "))
}

// glyph builds an Icon from its name. An unknown name is a build error
// naming the template line, where widget.Icon would panic.
func glyph(parts elementParts) (view.Node, error) {
	name := parts.args[0].(string)
	if err := knownIcon(name); err != nil {
		return nil, err
	}
	return widget.Icon(name, parts.opts...), nil
}

// iconButton builds an IconButton around the icon its icon param names.
// The glyph grows so it fills the chip and sits centered in it, as every
// hand-written copy of the tree had it.
func iconButton(parts elementParts) (view.Node, error) {
	name := parts.args[0].(string)
	if err := knownIcon(name); err != nil {
		return nil, err
	}
	return widget.IconButton(widget.Icon(name, view.Grow(1)), parts.opts...), nil
}

// toastCorner builds Toasts from its stack, the window size and the inset
// at the bottom. Toasts takes no options: it places itself in the window.
func toastCorner(parts elementParts) (view.Node, error) {
	if len(parts.opts) > 0 {
		return nil, errors.New("<toasts> takes no options, only stack, width, height and bottom-inset")
	}
	return widget.Toasts(parts.args[0].(*widget.ToastStack), parts.args[1].(int), parts.args[2].(int), parts.args[3].(int)), nil
}

// holding is what an element may contain.
type holding int

const (
	holdsChildren holding = iota
	holdsText             // text and no children: text, buttons
	holdsNothing          // a widget made from its arguments alone: tabs, table
)

// elementSpec is one row of the elements table. options and handlers
// are attributes only this element takes; a handler's value names a
// scope entry. A plain element stands in for a node built elsewhere (a
// slot's), so it takes only its params and the directives: an option on
// it would have nothing to apply to.
type elementSpec struct {
	build    element
	holds    holding
	plain    bool
	params   []param
	options  map[string]attributeReader
	handlers map[string]handlerAttr
}

// takes is whether the element accepts an attribute called name: a param, a
// directive, or an option or handler, own or generic.
func (spec elementSpec) takes(name string) bool {
	if _, isParam := spec.param(name); isParam || directives[name] {
		return true
	}
	if spec.plain {
		return false
	}
	_, isOption := spec.option(name)
	_, isHandler := spec.handler(name)
	return isOption || isHandler
}

// handler is the binding of the attribute called name on this element:
// one of its own handlers or a generic one.
func (spec elementSpec) handler(name string) (handlerAttr, bool) {
	if bind, ok := spec.handlers[name]; ok {
		return bind, true
	}
	bind, ok := handlers[name]
	return bind, ok
}

// option is the reader of the attribute called name on this element: one
// of its own options or a generic attribute.
func (spec elementSpec) option(name string) (attributeReader, bool) {
	if read, ok := spec.options[name]; ok {
		return read, true
	}
	read, ok := attributes[name]
	return read, ok
}

// optionNames are the names of the element's own options and handlers.
func (spec elementSpec) optionNames() []string {
	names := make([]string, 0, len(spec.options)+len(spec.handlers))
	for name := range spec.options {
		names = append(names, name)
	}
	for name := range spec.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ownNames are the names of the attributes only this element takes: its
// params, then its options.
func (spec elementSpec) ownNames() []string {
	return append(spec.paramNames(), spec.optionNames()...)
}

// param is the element's param called name, if it has one.
func (spec elementSpec) param(name string) (param, bool) {
	for _, candidate := range spec.params {
		if candidate.name == name {
			return candidate, true
		}
	}
	return param{}, false
}

func (spec elementSpec) paramNames() []string {
	names := make([]string, len(spec.params))
	for i, candidate := range spec.params {
		names[i] = candidate.name
	}
	return names
}

// elements are the tags a template may use.
//
// A param is the Go constructor's argument of that name, in the order the
// constructor takes them (see the arity helpers). The image param may be a
// nil picture: the widget draws nothing until the app has one. A lines
// element takes its color as a param, not the generic color option, which
// would color the column and not its texts. on-click on a check-row makes
// the whole row the click target.
var elements = map[string]elementSpec{
	"row":            {build: container(view.Row)},
	"col":            {build: container(view.Col)},
	"stack":          {build: container(view.Stack)},
	"spacer":         {build: optioned(view.Spacer), holds: holdsNothing},
	"hrule":          {build: optioned(view.HRule), holds: holdsNothing},
	"vrule":          {build: optioned(view.VRule), holds: holdsNothing},
	"vrule-inset":    {build: args1(view.VRuleInset), holds: holdsNothing, params: []param{wholeParam("inset")}},
	"text":           {build: labelled(view.Text), holds: holdsText},
	"primary-button": {build: labelled(widget.PrimaryButton), holds: holdsText},
	"link-button":    {build: labelled(widget.LinkButton), holds: holdsText},
	"icon":           {build: glyph, holds: holdsNothing, params: []param{textParam("name")}},
	"slot":           {build: slotNode, holds: holdsNothing, plain: true, params: []param{scoped[view.Node]("name")}},
	"caption":        {build: labelled(view.Caption), holds: holdsText},
	"lines":          {build: textArgs1(view.Lines), holds: holdsText, params: []param{tokenParam("color")}},
	"note":           {build: textArgs1(widget.Note), holds: holdsText, params: []param{tokenParam("tone")}},
	"button":         {build: labelled(widget.Button), holds: holdsText},
	"segment-chip":   {build: textArgs1(widget.SegmentChip), holds: holdsText, params: []param{boolParam("selected")}},
	"icon-button":    {build: iconButton, holds: holdsNothing, params: []param{textParam("icon")}},
	"scroll":         {build: itemArgs1(widget.Scroll), params: []param{scopedState[widget.ScrollState]("state")}},
	"hscroll":        {build: itemArgs1(widget.HScroll), params: []param{scopedState[widget.ScrollState]("state")}},
	"split": {build: itemArgs2(widget.Split), options: splitOptions,
		params: []param{scopedState[widget.SplitState]("state"), realParam("default")}},
	"tab-bar": {build: container(widget.TabBar)},
	"tabs": {build: args2(widget.Tabs), holds: holdsNothing,
		params: []param{scoped[[]widget.TabItem]("items"), wholeParam("active")}},
	"scroll-tabs": {build: args3(widget.ScrollTabs), holds: holdsNothing,
		params: []param{scopedState[widget.ScrollState]("state"), scoped[[]widget.TabItem]("items"), wholeParam("active")}},
	"table":          {build: args1(widget.Table), holds: holdsNothing, params: []param{scoped[widget.TableSpec]("spec")}},
	"dialog":         {build: container(widget.Dialog)},
	"dialog-title":   {build: itemArgs1(widget.DialogTitle), params: []param{textParam("title")}},
	"dialog-toolbar": {build: container(widget.DialogToolbar)},
	"message-dialog": {build: itemArgs1(widget.MessageDialog), params: []param{scopedState[widget.DialogState]("state")}},
	"resizable-dialog": {build: itemArgs5(widget.ResizableDialog), params: []param{scopedState[widget.ResizeState]("state"),
		wholeParam("min-width"), wholeParam("min-height"), wholeParam("max-width"), wholeParam("max-height")}},
	"menu": {build: args2(widget.Menu), holds: holdsNothing,
		params: []param{scopedState[core.Ctx]("ctx"), scopedState[widget.MenuState]("state")}},
	"toasts": {build: toastCorner, holds: holdsNothing,
		params: []param{scopedState[widget.ToastStack]("stack"), wholeParam("width"), wholeParam("height"), wholeParam("bottom-inset")}},
	"field": {build: args1(widget.Field), holds: holdsNothing, options: fieldOptions, handlers: fieldHandlers,
		params: []param{scopedState[widget.TextField]("state")}},
	"area": {build: args1(widget.Area), holds: holdsNothing, options: wrapOptions, handlers: areaHandlers,
		params: []param{scopedState[widget.TextArea]("state")}},
	"checkbox":  {build: args1(widget.Checkbox), holds: holdsNothing, params: []param{boolParam("on")}},
	"list-row":  {build: itemArgs1(widget.ListRow), params: []param{boolParam("selected")}},
	"form-row":  {build: itemArgs2(widget.FormRow), params: []param{textParam("label"), wholeParam("label-width")}},
	"check-row": {build: itemArgs1(widget.CheckRow), params: []param{boolParam("on")}},
	"code": {build: args1(widget.Code), holds: holdsNothing, options: wrapOptions,
		params: []param{scopedState[widget.CodeView]("state")}},
	"image": {build: args1(widget.Image), holds: holdsNothing, params: []param{scoped[*ebiten.Image]("img")}},
	"slider": {build: args5(widget.Slider), holds: holdsNothing, options: sliderOptions, handlers: sliderHandlers,
		params: []param{scopedState[widget.SliderState]("state"), wholeParam("min"), wholeParam("max"), wholeParam("step"), wholeParam("value")}},
}

// directives are attributes that control building, not looks.
var directives = map[string]bool{"if": true, "for": true}
