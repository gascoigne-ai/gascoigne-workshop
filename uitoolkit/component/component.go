// Package component builds view views from HTML-style component
// templates, so a layout can change without recompiling Go code, or be
// written inline in Go as a string.
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
// the two directives, and `on-click` names a func() of the scope.
//
// The usual form is an inline template in the method that feeds it, as
// with a Vue component's `template` option:
//
//	func (a *App) statusBar() Node {
//		return component.View(`
//		  <row h="28" bg="frame"><text small muted>{{status}}</text></row>`,
//			component.Scope{"status": a.status})
//	}
//
// The result is the same *view.El tree the Go DSL makes, so both styles
// share one runtime and can be mixed: a template can be a part of a
// hand-built view and the other way round.
package component

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Scope holds what a template can read: strings, numbers, bools, lists
// ([]any, []string or []map[string]any), nested maps, and func() handlers.
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
	text     string // the text between the tags, for leaf elements
	line     int
}

// Template is a parsed template file.
type Template struct {
	name string
	root *node
}

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
//		return component.View(`<row h="28"><text small>{{status}}</text></row>`,
//			component.Scope{"status": a.status})
//	}
func View(src string, sc Scope) view.Node {
	viewMu.Lock()
	t, ok := viewCache[src]
	if !ok {
		t = Must("inline", src)
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

// element builds one kind of element from its options, children and text.
type element func(opts []view.Option, kids []view.Item, text string) view.Node

func items(opts []view.Option, kids []view.Item) []view.Item {
	out := make([]view.Item, 0, len(opts)+len(kids))
	for _, o := range opts {
		out = append(out, o)
	}
	return append(out, kids...)
}

// elements are the tags a template may use. leaf tags take text and no
// children.
var elements = map[string]struct {
	build element
	leaf  bool
}{
	"row":            {func(o []view.Option, k []view.Item, _ string) view.Node { return view.Row(items(o, k)...) }, false},
	"col":            {func(o []view.Option, k []view.Item, _ string) view.Node { return view.Col(items(o, k)...) }, false},
	"stack":          {func(o []view.Option, k []view.Item, _ string) view.Node { return view.Stack(items(o, k)...) }, false},
	"spacer":         {func(o []view.Option, _ []view.Item, _ string) view.Node { return view.Spacer(o...) }, true},
	"hrule":          {func(o []view.Option, _ []view.Item, _ string) view.Node { return withOpts(view.HRule(), o) }, true},
	"vrule":          {func(o []view.Option, _ []view.Item, _ string) view.Node { return withOpts(view.VRule(), o) }, true},
	"text":           {func(o []view.Option, _ []view.Item, t string) view.Node { return view.Text(t, o...) }, true},
	"primary-button": {func(o []view.Option, _ []view.Item, t string) view.Node { return view.PrimaryButton(t, o...) }, true},
	"link-button":    {func(o []view.Option, _ []view.Item, t string) view.Node { return view.LinkButton(t, o...) }, true},
	"include":        {func([]view.Option, []view.Item, string) view.Node { return nil }, true},
	"icon":           {func(o []view.Option, _ []view.Item, name string) view.Node { return view.Icon(name, o...) }, true},
	"slot":           {func(o []view.Option, _ []view.Item, _ string) view.Node { return nil }, true},
	"caption":        {func(o []view.Option, _ []view.Item, t string) view.Node { return view.Caption(t, o...) }, true},
	"button":         {func(o []view.Option, _ []view.Item, t string) view.Node { return view.Button(t, o...) }, true},
}

// withOpts applies options to an element that was built without them.
func withOpts(n view.Node, opts []view.Option) view.Node {
	for _, o := range opts {
		o(n)
	}
	return n
}

// directives are attributes that control building, not looks.
var directives = map[string]bool{"if": true, "for": true, "on-click": true}

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
	"field-pad":       intAttr(view.FieldPad),
	"placeholder":     func(v string) (view.Option, error) { return view.Placeholder(v), nil },
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
			for _, a := range t.Attr {
				if _, ok := attributes[a.Name.Local]; !ok && !directives[a.Name.Local] && !(a.Name.Local == "name" && (n.tag == "icon" || n.tag == "slot" || n.tag == "include")) {
					return fail(l, "unknown attribute %q on <%s>%s", a.Name.Local, n.tag, hint(a.Name.Local, attributeNames(), "attributes"))
				}
				n.attrs = append(n.attrs, attr{a.Name.Local, a.Value, l})
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				if elements[parent.tag].leaf {
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
			if !elements[parent.tag].leaf {
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
		case "on-click":
			v, err := lookup(strings.TrimSpace(a.value), sc)
			if err != nil {
				return nil, t.errf(n, "%v", err)
			}
			f, ok := v.(func())
			if !ok {
				return nil, t.errf(n, "%s is not a func()", a.value)
			}
			opts = append(opts, view.OnClick(f))
		default:
			val, err := interpolate(a.value, sc)
			if err != nil {
				return nil, t.errf(n, "%v", err)
			}
			opt, err := attributes[a.name](val)
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
		if !view.HasIcon(text) {
			return nil, t.errf(n, "no icon %q; icons: %s", text, strings.Join(view.IconNames(), " "))
		}
	}
	return elements[n.tag].build(opts, kids, text), nil
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
	names := make([]string, 0, len(attributes)+len(directives))
	for n := range attributes {
		names = append(names, n)
	}
	for n := range directives {
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
		kind := "container"
		if elements[n].leaf {
			kind = "text and no children"
		}
		fmt.Fprintf(&sb, "  <%s>  %s\n", n, kind)
	}
	sb.WriteString("Attributes (Go option in kebab-case):\n")
	for _, n := range attributeNames() {
		fmt.Fprintf(&sb, "  %s\n", n)
	}
	sb.WriteString("<include name=\"x\"/> builds the template registered with component.Define(\"x\", ...) with this scope.\n")
	sb.WriteString("<slot name=\"x\"/> puts the view.Node the scope holds under x here.\n")
	sb.WriteString("<icon name=\"trash\"/> draws a built-in or registered icon (view.IconNames).\n")
	sb.WriteString("Text and attribute values may use {{path}}; if=\"name\" or \"!name\"; for=\"item in list\"; on-click=\"handler\".\n")
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
