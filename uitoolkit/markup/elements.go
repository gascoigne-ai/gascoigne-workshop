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

// insetRule builds a VRuleInset from its inset.
func insetRule(parts elementParts) (view.Node, error) {
	return view.VRuleInset(parts.args[0].(int), parts.opts...), nil
}

// labelled builds an element whose constructor takes its text first.
func labelled(constructor func(label string, opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) { return constructor(parts.text, parts.opts...), nil }
}

// placeholder is the builder of an element that buildOne makes itself.
func placeholder(elementParts) (view.Node, error) { return nil, nil }

// scroller builds a container whose constructor takes its scroll state
// first (Scroll, HScroll).
func scroller(constructor func(state *widget.ScrollState, items ...view.Item) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(*widget.ScrollState), parts.items()...), nil
	}
}

// picture builds an Image from its picture, which may be nil: the widget
// draws nothing until the app has one.
func picture(parts elementParts) (view.Node, error) {
	return widget.Image(parts.args[0].(*ebiten.Image), parts.opts...), nil
}

// checkbox builds a Checkbox from whether it is on.
func checkbox(parts elementParts) (view.Node, error) {
	return widget.Checkbox(parts.args[0].(bool), parts.opts...), nil
}

// listEntry builds a ListRow from whether it is selected.
func listEntry(parts elementParts) (view.Node, error) {
	return widget.ListRow(parts.args[0].(bool), parts.items()...), nil
}

// segment builds a Choice from its label and whether it is the chosen one,
// so one element stands for the PrimaryButton and Button pair.
func segment(parts elementParts) (view.Node, error) {
	return widget.Choice(parts.text, parts.args[0].(bool), parts.opts...), nil
}

// formLine builds a FormRow from its label and the width of the label
// column; the children are its controls.
func formLine(parts elementParts) (view.Node, error) {
	return widget.FormRow(parts.args[0].(string), parts.args[1].(int), parts.items()...), nil
}

// tickLine builds a CheckRow from whether its box is ticked; the children
// are its label and hint, and on-click on it makes the whole row the
// click target.
func tickLine(parts elementParts) (view.Node, error) {
	return widget.CheckRow(parts.args[0].(bool), parts.items()...), nil
}

// tabStrip builds Tabs from its items and the active index.
func tabStrip(parts elementParts) (view.Node, error) {
	return widget.Tabs(parts.args[0].([]widget.TabItem), parts.args[1].(int), parts.opts...), nil
}

// scrollingTabStrip builds ScrollTabs from its state, items and the active index.
func scrollingTabStrip(parts elementParts) (view.Node, error) {
	return widget.ScrollTabs(parts.args[0].(*widget.ScrollState), parts.args[1].([]widget.TabItem), parts.args[2].(int), parts.opts...), nil
}

// iconButton builds an IconButton around the icon its icon param names.
// The glyph grows so it fills the chip and sits centered in it, as every
// hand-written copy of the tree had it. An unknown name is a build error
// naming the template line, where widget.Icon would panic.
func iconButton(parts elementParts) (view.Node, error) {
	name := parts.args[0].(string)
	if !widget.HasIcon(name) {
		return nil, fmt.Errorf("no icon %q; icons: %s", name, strings.Join(widget.IconNames(), " "))
	}
	return widget.IconButton(widget.Icon(name, view.Grow(1)), parts.opts...), nil
}

// noteLine builds a Note from its text and its tone.
func noteLine(parts elementParts) (view.Node, error) {
	return widget.Note(parts.text, parts.args[0].(core.Token), parts.opts...), nil
}

// stackedLines builds Lines from its text, one Text per line, in its
// color. The color is the Go argument, so it is a param here and not the
// generic color option, which would color the column and not its texts.
func stackedLines(parts elementParts) (view.Node, error) {
	return view.Lines(parts.text, parts.args[0].(core.Token), parts.opts...), nil
}

// dialogTitle builds a DialogTitle from its title.
func dialogTitle(parts elementParts) (view.Node, error) {
	return widget.DialogTitle(parts.args[0].(string), parts.items()...), nil
}

// messageDialog builds a MessageDialog from its state.
func messageDialog(parts elementParts) (view.Node, error) {
	return widget.MessageDialog(parts.args[0].(*widget.DialogState), parts.items()...), nil
}

// resizableDialog builds a ResizableDialog from its state and its size
// bounds, in the Go order: minimum width and height, maximum width and
// height.
func resizableDialog(parts elementParts) (view.Node, error) {
	return widget.ResizableDialog(parts.args[0].(*widget.ResizeState), parts.args[1].(int), parts.args[2].(int), parts.args[3].(int), parts.args[4].(int), parts.items()...), nil
}

// popupMenu builds a Menu from the context it measures its rows with and
// its state.
func popupMenu(parts elementParts) (view.Node, error) {
	return widget.Menu(parts.args[0].(*core.Ctx), parts.args[1].(*widget.MenuState), parts.opts...), nil
}

// toastCorner builds Toasts from its stack, the window size and the inset
// at the bottom. Toasts takes no options: it places itself in the window.
func toastCorner(parts elementParts) (view.Node, error) {
	if len(parts.opts) > 0 {
		return nil, errors.New("<toasts> takes no options, only stack, width, height and bottom-inset")
	}
	return widget.Toasts(parts.args[0].(*widget.ToastStack), parts.args[1].(int), parts.args[2].(int), parts.args[3].(int)), nil
}

// grid builds a Table from its spec.
func grid(parts elementParts) (view.Node, error) {
	return widget.Table(parts.args[0].(widget.TableSpec), parts.opts...), nil
}

// splitter builds a Split from its state and default size.
func splitter(parts elementParts) (view.Node, error) {
	return widget.Split(parts.args[0].(*widget.SplitState), parts.args[1].(float64), parts.items()...), nil
}

// slider builds a Slider from its state, range, step and value.
func slider(parts elementParts) (view.Node, error) {
	return widget.Slider(parts.args[0].(*widget.SliderState), parts.args[1].(int), parts.args[2].(int), parts.args[3].(int), parts.args[4].(int), parts.opts...), nil
}

// stateful builds a widget whose constructor takes the state it edits
// first, then options (Field, Area, Code).
func stateful[T any](constructor func(state *T, opts ...view.Option) *view.El) element {
	return func(parts elementParts) (view.Node, error) {
		return constructor(parts.args[0].(*T), parts.opts...), nil
	}
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
// scope entry.
type elementSpec struct {
	build    element
	holds    holding
	params   []param
	options  map[string]attributeReader
	handlers map[string]handlerAttr
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
var elements = map[string]elementSpec{
	"row":            {build: container(view.Row)},
	"col":            {build: container(view.Col)},
	"stack":          {build: container(view.Stack)},
	"spacer":         {build: optioned(view.Spacer), holds: holdsText},
	"hrule":          {build: optioned(view.HRule), holds: holdsText},
	"vrule":          {build: optioned(view.VRule), holds: holdsText},
	"vrule-inset":    {build: insetRule, holds: holdsText, params: []param{wholeParam("inset")}},
	"text":           {build: labelled(view.Text), holds: holdsText},
	"primary-button": {build: labelled(widget.PrimaryButton), holds: holdsText},
	"link-button":    {build: labelled(widget.LinkButton), holds: holdsText},
	"icon":           {build: labelled(widget.Icon), holds: holdsText},
	"slot":           {build: placeholder, holds: holdsText},
	"caption":        {build: labelled(view.Caption), holds: holdsText},
	"lines":          {build: stackedLines, holds: holdsText, params: []param{tokenParam("color")}},
	"note":           {build: noteLine, holds: holdsText, params: []param{tokenParam("tone")}},
	"button":         {build: labelled(widget.Button), holds: holdsText},
	"choice":         {build: segment, holds: holdsText, params: []param{boolParam("selected")}},
	"icon-button":    {build: iconButton, holds: holdsNothing, params: []param{textParam("icon")}},
	"scroll":         {build: scroller(widget.Scroll), params: []param{scopedState[widget.ScrollState]("state")}},
	"hscroll":        {build: scroller(widget.HScroll), params: []param{scopedState[widget.ScrollState]("state")}},
	"split": {build: splitter, options: splitOptions,
		params: []param{scopedState[widget.SplitState]("state"), realParam("default")}},
	"tab-bar": {build: container(widget.TabBar)},
	"tabs": {build: tabStrip, holds: holdsNothing,
		params: []param{scoped[[]widget.TabItem]("items"), wholeParam("active")}},
	"scroll-tabs": {build: scrollingTabStrip, holds: holdsNothing,
		params: []param{scopedState[widget.ScrollState]("state"), scoped[[]widget.TabItem]("items"), wholeParam("active")}},
	"table":          {build: grid, holds: holdsNothing, params: []param{scoped[widget.TableSpec]("spec")}},
	"dialog":         {build: container(widget.Dialog)},
	"dialog-title":   {build: dialogTitle, params: []param{textParam("title")}},
	"dialog-toolbar": {build: container(widget.DialogToolbar)},
	"message-dialog": {build: messageDialog, params: []param{scopedState[widget.DialogState]("state")}},
	"resizable-dialog": {build: resizableDialog, params: []param{scopedState[widget.ResizeState]("state"),
		wholeParam("min-width"), wholeParam("min-height"), wholeParam("max-width"), wholeParam("max-height")}},
	"menu": {build: popupMenu, holds: holdsNothing,
		params: []param{scopedState[core.Ctx]("ctx"), scopedState[widget.MenuState]("state")}},
	"toasts": {build: toastCorner, holds: holdsNothing,
		params: []param{scopedState[widget.ToastStack]("stack"), wholeParam("width"), wholeParam("height"), wholeParam("bottom-inset")}},
	"field": {build: stateful(widget.Field), holds: holdsNothing, options: fieldOptions, handlers: fieldHandlers,
		params: []param{scopedState[widget.TextField]("state")}},
	"area": {build: stateful(widget.Area), holds: holdsNothing, options: wrapOptions, handlers: areaHandlers,
		params: []param{scopedState[widget.TextArea]("state")}},
	"checkbox":  {build: checkbox, holds: holdsNothing, params: []param{boolParam("on")}},
	"list-row":  {build: listEntry, params: []param{boolParam("selected")}},
	"form-row":  {build: formLine, params: []param{textParam("label"), wholeParam("label-width")}},
	"check-row": {build: tickLine, params: []param{boolParam("on")}},
	"code": {build: stateful(widget.Code), holds: holdsNothing, options: wrapOptions,
		params: []param{scopedState[widget.CodeView]("state")}},
	"image": {build: picture, holds: holdsNothing, params: []param{scoped[*ebiten.Image]("img")}},
	"slider": {build: slider, holds: holdsNothing, options: sliderOptions, handlers: sliderHandlers,
		params: []param{scopedState[widget.SliderState]("state"), wholeParam("min"), wholeParam("max"), wholeParam("step"), wholeParam("value")}},
}

// standsIn are the elements that stand in for a node built elsewhere (a
// slot's node): they take only name and the directives, because an option
// on them would have nothing to apply to.
var standsIn = map[string]bool{"slot": true}

// directives are attributes that control building, not looks.
var directives = map[string]bool{"if": true, "for": true}
