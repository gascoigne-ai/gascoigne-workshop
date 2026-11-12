// Package markup builds views from HTML-style templates, so a layout
// can change without recompiling Go code, or be written inline in Go as
// a string. Runnable examples: package uitoolkit/examples.
//
//	<row h="28" pad="12 0" gap="12" bg="frame" control="statusbar" border-t="border">
//	  <text small muted ellipsis grow="1">{{status}}</text>
//	  <text small muted if="showVersion">{{version}}</text>
//	  <button for="b in buttons" on-click="b.click">{{b.label}}</button>
//	</row>
//
// A template is XML with HTML's leniency (bare flag attributes such as
// `small`, entities such as &nbsp;). Elements and attributes are tables
// (elements, attributes below), so a new one is a new row; anything not
// in them is an error that names the file and line. Text and attribute
// values may use {{path}} to read the Scope; `if` and `for="x in xs"` are
// the two directives, and an `on-click` or other `on-*` event attribute
// names a function of the scope with the type of the Go option's argument
// (handlers). A container takes its Go constructor's arguments as
// attributes: state and specs name a scope entry of their Go type
// (<scroll state="list"> wants a *widget.ScrollState), and a wrong type
// is an error that names it.
//
// # Finding your way
//
//	Layout      row col stack spacer hrule vrule
//	Text        text caption icon
//	Buttons     button primary-button link-button
//	Containers  scroll hscroll split (vertical as-ratio min-first min-second)
//	Widgets     tabs scroll-tabs table   separator active-tab
//	Dialogs     dialog dialog-title message-dialog   focus-scope
//	Overlays    menu toasts
//	Templates   include slot
//
// The usual form is an inline template in the method that feeds it, as
// with a Vue component's `template` option:
//
//	func (a *App) statusBar() Node {
//		return markup.View(`
//		  <row h="28" bg="frame"><text small muted>{{status}}</text></row>`,
//			markup.Scope{"status": a.status})
//	}
//
// A larger view lives in a .vml file next to its Go file, embedded with
// go:embed and parsed once with Load or MustLoad:
//
//	//go:embed row.vml
//	var files embed.FS
//	var rowTemplate = markup.MustLoad(files, "row.vml")
//
// The result is the same *view.El tree the Go DSL makes, so both styles
// share one runtime and can be mixed: a template can be a part of a
// hand-built view and the other way round.
package markup

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Scope holds what a template can read: strings, numbers, bools, lists
// ([]any, []string or []map[string]any), nested maps, func() handlers,
// and the typed state and specs that containers take (*widget.ScrollState).
type Scope map[string]any

// Error is a problem in a template, with where it is.
type Error struct {
	Name string
	Line int
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("%s:%d: %s", e.Name, e.Line, e.Msg) }

type attr struct {
	name, value string
	line        int
}

type node struct {
	tag      string
	attrs    []attr
	children []*node
	text     string // the text between the tags, for elements that hold text
	line     int
}

// Template is a parsed template file.
type Template struct {
	name string
	root *node
}

// maxViewCache bounds the parsed inline templates kept.
const maxViewCache = 256

var (
	viewMu    sync.Mutex
	viewCache = map[string]*Template{}
)

// View builds an inline template for scope. Each distinct source string is
// parsed once and kept, so a view written inside a method costs only the
// build on every frame. A syntax error or a bad scope panics: both are
// mistakes in the program, not in its input.
//
//	func (a *App) statusBar() Node {
//		return markup.View(`<row h="28"><text small>{{status}}</text></row>`,
//			markup.Scope{"status": a.status})
//	}
func View(src string, sc Scope) view.Node {
	viewMu.Lock()
	t, ok := viewCache[src]
	if !ok {
		t = Must("inline", src)
		if len(viewCache) >= maxViewCache { // sources built from data would grow it for ever
			clear(viewCache)
		}
		viewCache[src] = t
	}
	viewMu.Unlock()
	return t.MustBuild(sc)
}

// Must is Parse for templates written in Go source: it panics on a
// syntax error, which is a programming mistake found the first time the
// program starts, the way regexp.MustCompile is.
func Must(name, src string) *Template {
	t, err := Parse(name, []byte(src))
	if err != nil {
		panic(err)
	}
	return t
}

// MustBuild is Build for a scope the caller controls: an error (an unknown
// name, a malformed value) is a programming mistake and panics.
func (t *Template) MustBuild(sc Scope) view.Node {
	n, err := t.Build(sc)
	if err != nil {
		panic(err)
	}
	return n
}

// Name is the name the template was parsed with.
func (t *Template) Name() string { return t.name }

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
	out := make([]view.Item, 0, len(parts.opts)+len(parts.kids))
	for _, option := range parts.opts {
		out = append(out, option)
	}
	return append(out, parts.kids...)
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

// bare builds an element whose constructor takes nothing, then applies the
// options to it.
func bare(constructor func() *view.El) element {
	return func(parts elementParts) (view.Node, error) { return withOpts(constructor(), parts.opts), nil }
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

// param is a constructor argument an element reads from one of its own
// attributes. Every param is required, as the Go argument is.
type param struct {
	name string
	read func(value string, scope Scope) (any, error)
}

// scoped is a param whose value names a scope entry of the Go type T, the
// way handlers bind functions: state and specs that a widget writes back
// to, which no string can stand for.
func scoped[T any](name string) param {
	var zero T
	want := fmt.Sprintf("%T", zero)
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		found, err := lookup(strings.TrimSpace(value), scope)
		if err != nil {
			return nil, err
		}
		typed, ok := found.(T)
		if !ok {
			return nil, fmt.Errorf("%s=%q is not a %s", name, value, want)
		}
		return typed, nil
	}}
}

// realParam is a param that is a number, written in the template or read
// with {{path}}.
func realParam(name string) param {
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		text, err := interpolate(value, scope)
		if err != nil {
			return nil, err
		}
		number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			return nil, fmt.Errorf("%s: %q is not a number", name, text)
		}
		return number, nil
	}}
}

// wholeParam is a param that is a whole number, written in the template or
// read with {{path}}.
func wholeParam(name string) param {
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		text, err := interpolate(value, scope)
		if err != nil {
			return nil, err
		}
		number, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			return nil, fmt.Errorf("%s: %q is not a whole number", name, text)
		}
		return number, nil
	}}
}

// tabStrip builds Tabs from its items and the active index.
func tabStrip(parts elementParts) (view.Node, error) {
	return widget.Tabs(parts.args[0].([]widget.TabItem), parts.args[1].(int), parts.opts...), nil
}

// scrollingTabStrip builds ScrollTabs from its state, items and the active index.
func scrollingTabStrip(parts elementParts) (view.Node, error) {
	return widget.ScrollTabs(parts.args[0].(*widget.ScrollState), parts.args[1].([]widget.TabItem), parts.args[2].(int), parts.opts...), nil
}

// textParam is a param that is text, open to {{path}}.
func textParam(name string) param {
	return param{name: name, read: func(value string, scope Scope) (any, error) { return interpolate(value, scope) }}
}

// dialogTitle builds a DialogTitle from its title.
func dialogTitle(parts elementParts) (view.Node, error) {
	return widget.DialogTitle(parts.args[0].(string), parts.items()...), nil
}

// messageDialog builds a MessageDialog from its state.
func messageDialog(parts elementParts) (view.Node, error) {
	return widget.MessageDialog(parts.args[0].(*widget.DialogState), parts.items()...), nil
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

// holding is what an element may contain.
type holding int

const (
	holdsChildren holding = iota
	holdsText             // text and no children: text, buttons
	holdsNothing          // a widget made from its arguments alone: tabs, table
)

// kinds describe each holding in the Reference.
var kinds = map[holding]string{
	holdsChildren: "container",
	holdsText:     "text and no children",
	holdsNothing:  "no children and no text",
}

// elementSpec is one row of the elements table. options are attributes
// only this element takes.
type elementSpec struct {
	build   element
	holds   holding
	params  []param
	options map[string]attributeReader
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

func (spec elementSpec) optionNames() []string {
	names := make([]string, 0, len(spec.options))
	for name := range spec.options {
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
	"hrule":          {build: bare(view.HRule), holds: holdsText},
	"vrule":          {build: bare(view.VRule), holds: holdsText},
	"text":           {build: labelled(view.Text), holds: holdsText},
	"primary-button": {build: labelled(widget.PrimaryButton), holds: holdsText},
	"link-button":    {build: labelled(widget.LinkButton), holds: holdsText},
	"include":        {build: placeholder, holds: holdsText},
	"icon":           {build: labelled(widget.Icon), holds: holdsText},
	"slot":           {build: placeholder, holds: holdsText},
	"caption":        {build: labelled(view.Caption), holds: holdsText},
	"button":         {build: labelled(widget.Button), holds: holdsText},
	"scroll":         {build: scroller(widget.Scroll), params: []param{scoped[*widget.ScrollState]("state")}},
	"hscroll":        {build: scroller(widget.HScroll), params: []param{scoped[*widget.ScrollState]("state")}},
	"split": {build: splitter, options: splitOptions,
		params: []param{scoped[*widget.SplitState]("state"), realParam("default")}},
	"tabs": {build: tabStrip, holds: holdsNothing,
		params: []param{scoped[[]widget.TabItem]("items"), wholeParam("active")}},
	"scroll-tabs": {build: scrollingTabStrip, holds: holdsNothing,
		params: []param{scoped[*widget.ScrollState]("state"), scoped[[]widget.TabItem]("items"), wholeParam("active")}},
	"table":          {build: grid, holds: holdsNothing, params: []param{scoped[widget.TableSpec]("spec")}},
	"dialog":         {build: container(widget.Dialog)},
	"dialog-title":   {build: dialogTitle, params: []param{textParam("title")}},
	"message-dialog": {build: messageDialog, params: []param{scoped[*widget.DialogState]("state")}},
	"menu": {build: popupMenu, holds: holdsNothing,
		params: []param{scoped[*core.Ctx]("ctx"), scoped[*widget.MenuState]("state")}},
	"toasts": {build: toastCorner, holds: holdsNothing,
		params: []param{scoped[*widget.ToastStack]("stack"), wholeParam("width"), wholeParam("height"), wholeParam("bottom-inset")}},
}

// withOpts applies options to an element that was built without them.
func withOpts(node view.Node, opts []view.Option) view.Node {
	for _, option := range opts {
		option(node)
	}
	return node
}

// directives are attributes that control building, not looks.
var directives = map[string]bool{"if": true, "for": true}

// handlerAttr binds an attribute to a function of the scope: the attribute
// value names it, and it must have the type of the Go option's argument.
type handlerAttr struct {
	want string // the function type, for the error
	bind func(value any) (view.Option, bool)
}

func handler[F any](option func(F) view.Option) handlerAttr {
	var zero F
	return handlerAttr{
		want: fmt.Sprintf("%T", zero),
		bind: func(value any) (view.Option, bool) {
			f, ok := value.(F)
			if !ok {
				return nil, false
			}
			return option(f), true
		},
	}
}

// handlers are the event attributes, each the Go option in kebab-case.
var handlers = map[string]handlerAttr{
	"on-click":          handler(view.OnClick),
	"on-click-at":       handler(view.OnClickAt),
	"on-right-click":    handler(view.OnRightClick),
	"on-right-click-in": handler(view.OnRightClickIn),
	"on-middle-click":   handler(view.OnMiddleClick),
	"on-drag-over":      handler(view.OnDragOver),
	"on-change":         handler(view.OnChange),
	"on-select":         handler(view.OnSelect),
	"on-dismiss":        handler(view.OnDismiss),
}

// attributes turn an attribute value (already interpolated) into an
// option. A flag is written bare (`small`) or `small="true"`; "false" is
// off. An option may be nil, meaning nothing to apply.
var attributes = map[string]func(v string) (view.Option, error){
	"w":               intAttr(view.W),
	"h":               intAttr(view.H),
	"gap":             intAttr(view.Gap),
	"grow":            floatAttr(view.Grow),
	"align-items":     enumAttr("align-items", alignments, view.AlignItems),
	"align-self":      enumAttr("align-self", alignments, view.AlignSelf),
	"justify-content": enumAttr("justify-content", justifications, view.JustifyContent),
	"pad":             padAttr,
	"pad-x":           intAttr(view.PadX),
	"pad-y":           intAttr(view.PadY),
	"pad-l":           intAttr(view.PadL),
	"pad-t":           intAttr(view.PadT),
	"pad-r":           intAttr(view.PadR),
	"pad-b":           intAttr(view.PadB),
	"margin":          intAttr(view.Margin),
	"margin-x":        intAttr(view.MarginX),
	"margin-y":        intAttr(view.MarginY),
	"margin-l":        intAttr(view.MarginL),
	"margin-t":        intAttr(view.MarginT),
	"margin-r":        intAttr(view.MarginR),
	"margin-b":        intAttr(view.MarginB),
	"bg":              tokenAttr(view.Bg),
	"border":          tokenAttr(view.Border),
	"border-t":        tokenAttr(view.BorderT),
	"border-b":        tokenAttr(view.BorderB),
	"border-l":        tokenAttr(view.BorderL),
	"border-inset":    intAttr(view.BorderInset),
	"hover-bg":        tokenAttr(view.HoverBg),
	"hover-color":     tokenAttr(view.HoverColor),
	"min-w":           intAttr(view.MinW),
	"max-w":           intAttr(view.MaxW),
	"min-h":           intAttr(view.MinH),
	"max-h":           intAttr(view.MaxH),
	"field-pad":       intAttr(widget.FieldPad),
	"placeholder":     func(v string) (view.Option, error) { return widget.Placeholder(v), nil },
	"shadow":          flagAttr("shadow", view.Shadow),
	"clip":            flagAttr("clip", view.Clip),
	"modal":           flagAttr("modal", view.Modal),
	"float-center":    flagAttr("float-center", view.FloatCenter),
	"float-at":        intsAttr("float-at", 2, func(n []int) view.Option { return view.FloatAt(n[0], n[1]) }),
	"float-tr":        intsAttr("float-tr", 2, func(n []int) view.Option { return view.FloatTR(n[0], n[1]) }),
	"edge-t":          tokenAttr(view.EdgeT),
	"edge-l":          tokenAttr(view.EdgeL),
	"edge-r":          tokenAttr(view.EdgeR),
	"border-r":        tokenAttr(view.BorderR),
	"border-x":        tokenAttr(view.BorderX),
	"border-y":        tokenAttr(view.BorderY),
	"color":           tokenAttr(view.Color),
	"control":         func(v string) (view.Option, error) { return view.Control(v), nil },
	"tooltip":         func(v string) (view.Option, error) { return view.Tooltip(v), nil },
	"small":           flagAttr("small", view.Small),
	"muted":           flagAttr("muted", view.Muted),
	"bold":            flagAttr("bold", view.Bold),
	"mono":            flagAttr("mono", view.Mono),
	"ellipsis":        flagAttr("ellipsis", view.Ellipsis),
	"text-center":     flagAttr("text-center", view.TextCenter),
	"optional":        flagAttr("optional", view.Optional),
	"lighten":         flagAttr("lighten", view.Lighten),
	"keep-in-view":    flagAttr("keep-in-view", view.KeepInView),
	"separator":       intAttr(view.Separator),
	"active-tab":      flagAttr("active-tab", widget.ActiveTab),
	"focus-scope":     flagAttr("focus-scope", view.FocusScope),
}

func intAttr(f func(int) view.Option) func(string) (view.Option, error) {
	return func(v string) (view.Option, error) {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return nil, fmt.Errorf("%q is not a whole number", v)
		}
		return f(n), nil
	}
}

func floatAttr(f func(float64) view.Option) func(string) (view.Option, error) {
	return func(v string) (view.Option, error) {
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", v)
		}
		return f(n), nil
	}
}

func tokenAttr(f func(core.Token) view.Option) func(string) (view.Option, error) {
	return func(v string) (view.Option, error) {
		if strings.TrimSpace(v) == "" {
			return nil, errors.New("a color token is needed")
		}
		return f(core.Token(strings.TrimSpace(v))), nil
	}
}

func flagAttr(name string, f func() view.Option) func(string) (view.Option, error) {
	return func(v string) (view.Option, error) {
		switch strings.TrimSpace(v) {
		case "", "true", name: // a bare attribute reads as its own name
			return f(), nil
		case "false":
			return nil, nil
		}
		return nil, fmt.Errorf("%q is not true or false", v)
	}
}

// padAttr is `pad="12"` for all sides or `pad="12 4"` for horizontal and vertical.
func padAttr(v string) (view.Option, error) {
	f := strings.Fields(v)
	nums := make([]int, len(f))
	for i, s := range f {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("%q is not a whole number", s)
		}
		nums[i] = n
	}
	switch len(nums) {
	case 1:
		return view.Pad(nums[0]), nil
	case 2:
		x, y := nums[0], nums[1]
		return func(e *view.El) { view.PadX(x)(e); view.PadY(y)(e) }, nil
	}
	return nil, fmt.Errorf("pad takes one or two numbers, not %q", v)
}

// Parse reads a template. name is used in error messages.
func Parse(name string, src []byte) (*Template, error) {
	d := xml.NewDecoder(bytes.NewReader(src))
	d.Strict = false
	d.Entity = xml.HTMLEntity
	line := func() int { return 1 + bytes.Count(src[:min(int(d.InputOffset()), len(src))], []byte("\n")) }
	fail := func(l int, format string, args ...any) (*Template, error) {
		return nil, &Error{Name: name, Line: l, Msg: fmt.Sprintf(format, args...)}
	}
	var stack []*node
	var root *node
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fail(line(), "%v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			l := line()
			if _, ok := elements[t.Name.Local]; !ok {
				return fail(l, "unknown element <%s>%s", t.Name.Local, hint(t.Name.Local, elementNames(), "elements"))
			}
			n := &node{tag: t.Name.Local, line: l}
			spec := elements[n.tag]
			for _, a := range t.Attr {
				_, isParam := spec.param(a.Name.Local)
				if _, ok := spec.option(a.Name.Local); !ok && !isParam && !directives[a.Name.Local] && !isHandler(a.Name.Local) && !(a.Name.Local == "name" && (n.tag == "icon" || n.tag == "slot" || n.tag == "include")) {
					return fail(l, "unknown attribute %q on <%s>%s", a.Name.Local, n.tag, hint(a.Name.Local, append(attributeNames(), spec.ownNames()...), "attributes"))
				}
				n.attrs = append(n.attrs, attr{a.Name.Local, a.Value, l})
			}
			if missing := n.missingParam(spec); missing != "" {
				return fail(l, "<%s> needs a %s attribute", n.tag, missing)
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				if elements[parent.tag].holds != holdsChildren {
					return fail(l, "<%s> cannot contain <%s>", parent.tag, n.tag)
				}
				parent.children = append(parent.children, n)
			} else {
				if root != nil {
					return fail(l, "a template has one root element")
				}
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) == 0 {
				return fail(line(), "unexpected </%s>", t.Name.Local)
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			s := strings.TrimSpace(string(t))
			if s == "" {
				continue
			}
			if len(stack) == 0 {
				return fail(line(), "text outside the root element")
			}
			parent := stack[len(stack)-1]
			if elements[parent.tag].holds != holdsText {
				return fail(line(), "text is only allowed inside <text>, <caption> and <button>, not <%s>", parent.tag)
			}
			if parent.text != "" {
				parent.text += " "
			}
			parent.text += strings.Join(strings.Fields(s), " ")
		}
	}
	if root == nil {
		return fail(1, "the template is empty")
	}
	if len(stack) > 0 {
		return fail(line(), "<%s> is not closed", stack[len(stack)-1].tag)
	}
	return &Template{name: name, root: root}, nil
}

// Build makes the view tree for scope. A root whose `if` is false gives a
// nil node, which view ignores.
func (t *Template) Build(sc Scope) (view.Node, error) {
	n, err := t.build(t.root, sc)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (t *Template) errf(n *node, format string, args ...any) error {
	return &Error{Name: t.name, Line: n.line, Msg: fmt.Sprintf(format, args...)}
}

// build makes the element of n, or a fragment for a `for`.
func (t *Template) build(n *node, sc Scope) (view.Node, error) {
	var forAttr *attr
	for i := range n.attrs {
		if n.attrs[i].name == "for" {
			forAttr = &n.attrs[i]
		}
	}
	if forAttr == nil {
		return t.buildOne(n, sc)
	}
	name, listExpr, ok := strings.Cut(forAttr.value, " in ")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return nil, t.errf(n, `for must read "item in list", not %q`, forAttr.value)
	}
	v, err := lookup(strings.TrimSpace(listExpr), sc)
	if err != nil {
		return nil, t.errf(n, "%v", err)
	}
	list, err := asList(v)
	if err != nil {
		return nil, t.errf(n, "%s: %v", strings.TrimSpace(listExpr), err)
	}
	var nodes []view.Item
	for i, it := range list {
		child := Scope{}
		for k, val := range sc {
			child[k] = val
		}
		child[name], child["$index"] = it, i
		el, err := t.buildOne(n, child)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, el)
	}
	return view.Frag(nodes...), nil
}

// missingParam is the first param of spec that n does not set, "" when it
// sets them all.
func (n *node) missingParam(spec elementSpec) string {
	for _, wanted := range spec.params {
		if !slices.ContainsFunc(n.attrs, func(set attr) bool { return set.name == wanted.name }) {
			return wanted.name
		}
	}
	return ""
}

// args reads the values of the element's params from n's attributes, in
// the order of the params.
func (t *Template) args(n *node, spec elementSpec, scope Scope) ([]any, error) {
	values := make([]any, len(spec.params))
	for _, set := range n.attrs {
		for i, wanted := range spec.params {
			if set.name != wanted.name {
				continue
			}
			value, err := wanted.read(set.value, scope)
			if err != nil {
				return nil, t.errf(n, "%v", err)
			}
			values[i] = value
		}
	}
	return values, nil
}

func isHandler(name string) bool {
	_, ok := handlers[name]
	return ok
}

// bindHandler reads the scope function an event attribute names.
func (t *Template) bindHandler(n *node, a attr, h handlerAttr, sc Scope) (view.Option, error) {
	value, err := lookup(strings.TrimSpace(a.value), sc)
	if err != nil {
		return nil, t.errf(n, "%v", err)
	}
	opt, ok := h.bind(value)
	if !ok {
		return nil, t.errf(n, "%s=%q is not a %s", a.name, a.value, h.want)
	}
	return opt, nil
}

// buildOne makes the element of n once, honoring `if`.
func (t *Template) buildOne(n *node, sc Scope) (view.Node, error) {
	var opts []view.Option
	for _, a := range n.attrs {
		switch a.name {
		case "for":
		case "if":
			ok, err := truthy(a.value, sc)
			if err != nil {
				return nil, t.errf(n, "%v", err)
			}
			if !ok {
				return nil, nil
			}
		case "name": // the icon's name, which is its text
			continue
		default:
			if _, isParam := elements[n.tag].param(a.name); isParam {
				continue // read by args
			}
			if h, ok := handlers[a.name]; ok {
				opt, err := t.bindHandler(n, a, h, sc)
				if err != nil {
					return nil, err
				}
				opts = append(opts, opt)
				continue
			}
			val, err := interpolate(a.value, sc)
			if err != nil {
				return nil, t.errf(n, "%v", err)
			}
			read, _ := elements[n.tag].option(a.name) // Parse checked it exists
			opt, err := read(val)
			if err != nil {
				return nil, t.errf(n, "%s: %v", a.name, err)
			}
			if opt != nil {
				opts = append(opts, opt)
			}
		}
	}
	var kids []view.Item
	for _, c := range n.children {
		k, err := t.build(c, sc)
		if err != nil {
			return nil, err
		}
		kids = append(kids, k)
	}
	text, err := interpolate(n.text, sc)
	if err != nil {
		return nil, t.errf(n, "%v", err)
	}
	if n.tag == "include" { // another defined template, built with this scope
		for _, a := range n.attrs {
			if a.name == "name" {
				defined.Lock()
				other, ok := defined.m[a.value]
				defined.Unlock()
				if !ok {
					return nil, t.errf(n, "no template defined as %q; defined: %s", a.value, strings.Join(definedNames(), " "))
				}
				return other.Build(sc)
			}
		}
		return nil, t.errf(n, `<include> needs a name`)
	}
	if n.tag == "slot" { // a node the caller made, as a Vue slot
		for _, a := range n.attrs {
			if a.name == "name" {
				v, ok := sc[a.value]
				if !ok {
					return nil, t.errf(n, "unknown slot %q", a.value)
				}
				node, ok := v.(view.Node)
				if !ok {
					return nil, t.errf(n, "slot %q is not a view.Node", a.value)
				}
				return node, nil
			}
		}
		return nil, t.errf(n, `<slot> needs a name`)
	}
	if n.tag == "icon" {
		for _, a := range n.attrs {
			if a.name == "name" {
				if text, err = interpolate(a.value, sc); err != nil {
					return nil, t.errf(n, "%v", err)
				}
			}
		}
		if !widget.HasIcon(text) {
			return nil, t.errf(n, "no icon %q; icons: %s", text, strings.Join(widget.IconNames(), " "))
		}
	}
	args, err := t.args(n, elements[n.tag], sc)
	if err != nil {
		return nil, err
	}
	built, err := elements[n.tag].build(elementParts{args: args, opts: opts, kids: kids, text: text})
	if err != nil {
		return nil, t.errf(n, "%v", err)
	}
	return built, nil
}

// interpolate replaces each {{path}} in s with the value it names.
func interpolate(s string, sc Scope) (string, error) {
	var sb strings.Builder
	for {
		i := strings.Index(s, "{{")
		if i < 0 {
			sb.WriteString(s)
			return sb.String(), nil
		}
		j := strings.Index(s[i:], "}}")
		if j < 0 {
			return "", errors.New("{{ without }}")
		}
		sb.WriteString(s[:i])
		v, err := lookup(strings.TrimSpace(s[i+2:i+j]), sc)
		if err != nil {
			return "", err
		}
		fmt.Fprint(&sb, v)
		s = s[i+j+2:]
	}
}

// lookup finds a dotted path in the scope.
func lookup(path string, sc Scope) (any, error) {
	if path == "" {
		return nil, errors.New("an empty name")
	}
	var cur any = map[string]any(sc)
	for i, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			if s, isScope := cur.(Scope); isScope {
				m, ok = s, true
			}
		}
		if !ok {
			return nil, fmt.Errorf("%q: %s is not an object", path, strings.Join(strings.Split(path, ".")[:i], "."))
		}
		v, found := m[part]
		if !found {
			return nil, fmt.Errorf("unknown name %q", path)
		}
		cur = v
	}
	return cur, nil
}

// truthy evaluates `name` or `!name`.
func truthy(expr string, sc Scope) (bool, error) {
	expr = strings.TrimSpace(expr)
	neg := strings.HasPrefix(expr, "!")
	v, err := lookup(strings.TrimSpace(strings.TrimPrefix(expr, "!")), sc)
	if err != nil {
		return false, err
	}
	var ok bool
	switch x := v.(type) {
	case nil:
	case bool:
		ok = x
	case string:
		ok = x != ""
	case int:
		ok = x != 0
	case float64:
		ok = x != 0
	case []any:
		ok = len(x) > 0
	case []string:
		ok = len(x) > 0
	case []map[string]any:
		ok = len(x) > 0
	default:
		ok = true
	}
	return ok != neg, nil
}

func asList(v any) ([]any, error) {
	switch x := v.(type) {
	case []any:
		return x, nil
	case []string:
		out := make([]any, len(x))
		for i, s := range x {
			out[i] = s
		}
		return out, nil
	case []map[string]any:
		out := make([]any, len(x))
		for i, m := range x {
			out[i] = m
		}
		return out, nil
	}
	return nil, errors.New("is not a list")
}

func elementNames() []string {
	names := make([]string, 0, len(elements))
	for n := range elements {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func attributeNames() []string {
	names := make([]string, 0, len(attributes)+len(directives)+len(handlers))
	for n := range attributes {
		names = append(names, n)
	}
	for n := range directives {
		names = append(names, n)
	}
	for n := range handlers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// hint is the end of an error message about an unknown name: the
// closest known name, when one is near, and the list of all of them.
func hint(got string, known []string, what string) string {
	best, bestD := "", 3
	for _, k := range known {
		if d := distance(got, k); d < bestD {
			best, bestD = k, d
		}
	}
	out := ""
	if best != "" {
		out = fmt.Sprintf(" (did you mean %q?)", best)
	}
	return out + "; " + what + ": " + strings.Join(known, " ")
}

// distance is the edit distance between a and b.
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// Reference describes every element and attribute a template may use, as
// text. It is built from the same tables the parser checks against, so
// it cannot be out of date.
func Reference() string {
	var sb strings.Builder
	sb.WriteString("Elements:\n")
	for _, n := range elementNames() {
		fmt.Fprintf(&sb, "  <%s>  %s", n, kinds[elements[n].holds])
		if params := elements[n].paramNames(); len(params) > 0 {
			fmt.Fprintf(&sb, "; takes %s", strings.Join(params, " "))
		}
		if own := elements[n].optionNames(); len(own) > 0 {
			fmt.Fprintf(&sb, "; options %s", strings.Join(own, " "))
		}
		sb.WriteByte('\n')
	}
	sb.WriteString("Attributes (Go option in kebab-case):\n")
	for _, n := range attributeNames() {
		fmt.Fprintf(&sb, "  %s\n", n)
	}
	sb.WriteString("<include name=\"x\"/> builds the template registered with markup.Define(\"x\", ...) with this scope.\n")
	sb.WriteString("<slot name=\"x\"/> puts the view.Node the scope holds under x here.\n")
	sb.WriteString("<icon name=\"trash\"/> draws a built-in or registered icon (widget.IconNames).\n")
	sb.WriteString("Text and attribute values may use {{path}}; if=\"name\" or \"!name\"; for=\"item in list\"; on-click=\"handler\" (and the other on-* events, each a func of the scope).\n")
	sb.WriteString("What an element takes (state=\"x\", ...) are its Go constructor's arguments, all required; state and specs name a scope entry of the Go type.\n")
	return sb.String()
}

// intsAttr reads exactly n whole numbers separated by spaces.
func intsAttr(name string, n int, f func([]int) view.Option) func(string) (view.Option, error) {
	return func(v string) (view.Option, error) {
		fields := strings.Fields(v)
		if len(fields) != n {
			return nil, fmt.Errorf("%s takes %d numbers, not %q", name, n, v)
		}
		nums := make([]int, n)
		for i, s := range fields {
			x, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("%q is not a whole number", s)
			}
			nums[i] = x
		}
		return f(nums), nil
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
func enumAttr[T any](name string, table map[string]T, f func(T) view.Option) func(string) (view.Option, error) {
	return func(v string) (view.Option, error) {
		val, ok := table[strings.TrimSpace(v)]
		if !ok {
			names := make([]string, 0, len(table))
			for k := range table {
				names = append(names, k)
			}
			sort.Strings(names)
			return nil, fmt.Errorf("%s is one of %s, not %q", name, strings.Join(names, ", "), v)
		}
		return f(val), nil
	}
}

var defined = struct {
	sync.Mutex
	m map[string]*Template
}{m: map[string]*Template{}}

// Define registers a template under a name, for <include name="..."/> in
// other templates. A syntax error panics: it is a mistake in the program.
func Define(name, src string) {
	t := Must(name, src)
	defined.Lock()
	defined.m[name] = t
	defined.Unlock()
}

func definedNames() []string {
	defined.Lock()
	defer defined.Unlock()
	names := make([]string, 0, len(defined.m))
	for n := range defined.m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
