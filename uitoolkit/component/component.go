// Package component builds vml views from HTML-style component
// templates, so a layout can change without recompiling Go code, or be
// written inline in Go as a string.
//
//	<row h="28" pad="12 0" gap="12" bg="frame" control="statusbar" border-t="border">
//	  <text small muted ellipsis grow="1">{{status}}</text>
//	  <text small muted if="showVersion">{{version}}</text>
//	  <button for="b in buttons" onclick="b.click">{{b.label}}</button>
//	</row>
//
// A template is XML with HTML's leniency (bare flag attributes such as
// `small`, entities such as &nbsp;). Elements and attributes are tables
// (elements, attributes below), so a new one is a new row; anything not
// in them is an error that names the file and line. Text and attribute
// values may use {{path}} to read the Scope; `if` and `for="x in xs"` are
// the two directives, and `onclick` names a func() of the scope.
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
// The result is the same *vml.El tree the Go DSL makes, so both styles
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
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
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
func View(src string, sc Scope) vml.Node {
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
func (t *Template) MustBuild(sc Scope) vml.Node {
	n, err := t.Build(sc)
	if err != nil {
		panic(err)
	}
	return n
}

// Name is the name the template was parsed with.
func (t *Template) Name() string { return t.name }

// element builds one kind of element from its options, children and text.
type element func(opts []vml.Option, kids []vml.Item, text string) vml.Node

func items(opts []vml.Option, kids []vml.Item) []vml.Item {
	out := make([]vml.Item, 0, len(opts)+len(kids))
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
	"row":     {func(o []vml.Option, k []vml.Item, _ string) vml.Node { return vml.Row(items(o, k)...) }, false},
	"col":     {func(o []vml.Option, k []vml.Item, _ string) vml.Node { return vml.Col(items(o, k)...) }, false},
	"stack":   {func(o []vml.Option, k []vml.Item, _ string) vml.Node { return vml.Stack(items(o, k)...) }, false},
	"spacer":  {func(o []vml.Option, _ []vml.Item, _ string) vml.Node { return vml.Spacer(o...) }, true},
	"hr":      {func(o []vml.Option, _ []vml.Item, _ string) vml.Node { return withOpts(vml.HRule(), o) }, true},
	"vr":      {func(o []vml.Option, _ []vml.Item, _ string) vml.Node { return withOpts(vml.VRule(), o) }, true},
	"text":    {func(o []vml.Option, _ []vml.Item, t string) vml.Node { return vml.Text(t, o...) }, true},
	"caption": {func(o []vml.Option, _ []vml.Item, t string) vml.Node { return vml.Caption(t, o...) }, true},
	"button":  {func(o []vml.Option, _ []vml.Item, t string) vml.Node { return vml.Button(t, o...) }, true},
}

// withOpts applies options to an element that was built without them.
func withOpts(n vml.Node, opts []vml.Option) vml.Node {
	for _, o := range opts {
		o(n)
	}
	return n
}

// directives are attributes that control building, not looks.
var directives = map[string]bool{"if": true, "for": true, "onclick": true}

// attributes turn an attribute value (already interpolated) into an
// option. A flag is written bare (`small`) or `small="true"`; "false" is
// off. An option may be nil, meaning nothing to apply.
var attributes = map[string]func(v string) (vml.Option, error){
	"w":        intAttr(vml.W),
	"h":        intAttr(vml.H),
	"gap":      intAttr(vml.Gap),
	"grow":     floatAttr(vml.Grow),
	"pad":      padAttr,
	"pad-x":    intAttr(vml.PadX),
	"pad-y":    intAttr(vml.PadY),
	"pad-l":    intAttr(vml.PadL),
	"pad-t":    intAttr(vml.PadT),
	"pad-r":    intAttr(vml.PadR),
	"pad-b":    intAttr(vml.PadB),
	"margin":   intAttr(vml.Margin),
	"margin-x": intAttr(vml.MarginX),
	"margin-y": intAttr(vml.MarginY),
	"margin-l": intAttr(vml.MarginL),
	"margin-t": intAttr(vml.MarginT),
	"margin-r": intAttr(vml.MarginR),
	"margin-b": intAttr(vml.MarginB),
	"margin":   intAttr(vml.Margin),
	"margin-x": intAttr(vml.MarginX),
	"margin-y": intAttr(vml.MarginY),
	"margin-l": intAttr(vml.MarginL),
	"margin-t": intAttr(vml.MarginT),
	"margin-r": intAttr(vml.MarginR),
	"margin-b": intAttr(vml.MarginB),
	"bg":       tokenAttr(vml.Bg),
	"border":   tokenAttr(vml.Border),
	"border-t": tokenAttr(vml.BorderT),
	"border-b": tokenAttr(vml.BorderB),
	"border-l": tokenAttr(vml.BorderL),
	"border-r": tokenAttr(vml.BorderR),
	"border-x": tokenAttr(vml.BorderX),
	"border-y": tokenAttr(vml.BorderY),
	"color":    tokenAttr(vml.Color),
	"control":  func(v string) (vml.Option, error) { return vml.Control(v), nil },
	"tip":      func(v string) (vml.Option, error) { return vml.Tooltip(v), nil },
	"small":    flagAttr("small", vml.Small),
	"muted":    flagAttr("muted", vml.Muted),
	"bold":     flagAttr("bold", vml.Bold),
	"mono":     flagAttr("mono", vml.Mono),
	"ellipsis": flagAttr("ellipsis", vml.Ellipsis),
	"center":   flagAttr("center", vml.Center),
	"optional": flagAttr("optional", vml.Optional),
}

func intAttr(f func(int) vml.Option) func(string) (vml.Option, error) {
	return func(v string) (vml.Option, error) {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return nil, fmt.Errorf("%q is not a whole number", v)
		}
		return f(n), nil
	}
}

func floatAttr(f func(float64) vml.Option) func(string) (vml.Option, error) {
	return func(v string) (vml.Option, error) {
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", v)
		}
		return f(n), nil
	}
}

func tokenAttr(f func(core.Token) vml.Option) func(string) (vml.Option, error) {
	return func(v string) (vml.Option, error) {
		if strings.TrimSpace(v) == "" {
			return nil, errors.New("a color token is needed")
		}
		return f(core.Token(strings.TrimSpace(v))), nil
	}
}

func flagAttr(name string, f func() vml.Option) func(string) (vml.Option, error) {
	return func(v string) (vml.Option, error) {
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
func padAttr(v string) (vml.Option, error) {
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
		return vml.Pad(nums[0]), nil
	case 2:
		x, y := nums[0], nums[1]
		return func(e *vml.El) { vml.PadX(x)(e); vml.PadY(y)(e) }, nil
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
				if _, ok := attributes[a.Name.Local]; !ok && !directives[a.Name.Local] {
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
// nil node, which vml ignores.
func (t *Template) Build(sc Scope) (vml.Node, error) {
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
func (t *Template) build(n *node, sc Scope) (vml.Node, error) {
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
	var nodes []vml.Item
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
	return vml.Frag(nodes...), nil
}

// buildOne makes the element of n once, honoring `if`.
func (t *Template) buildOne(n *node, sc Scope) (vml.Node, error) {
	var opts []vml.Option
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
		case "onclick":
			v, err := lookup(strings.TrimSpace(a.value), sc)
			if err != nil {
				return nil, t.errf(n, "%v", err)
			}
			f, ok := v.(func())
			if !ok {
				return nil, t.errf(n, "%s is not a func()", a.value)
			}
			opts = append(opts, vml.OnClick(f))
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
	var kids []vml.Item
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
	sb.WriteString("Text and attribute values may use {{path}}; if=\"name\" or \"!name\"; for=\"item in list\"; onclick=\"handler\".\n")
	return sb.String()
}
