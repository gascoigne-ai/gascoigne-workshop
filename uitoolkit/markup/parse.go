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
// values may use {{path}} to read the Scope (and the fallback scopes
// Build takes, read after it), where each step of the path
// is a map key or an exported struct field, so `for="row in rows"` walks
// a []Row and {{row.Name}} reads its field; `if` and `for="x in xs"` are
// the two directives (`$index` is the item's index, `if` hides on false,
// nil, empty and zero), and an `on-click` or other `on-*` event attribute
// names a function of the scope with the type of the Go option's argument
// (handlers). A container or widget takes its Go constructor's arguments
// as attributes: state and specs name a scope entry of their Go type
// (<scroll state="list"> wants a *widget.ScrollState), and a wrong type
// is an error that names it, as is a nil state pointer. A <slot name="path"/>
// puts in a view.Node of the scope, such as a view.Custom component, which
// keeps its own layout and input. Options and
// handlers that only work on one widget (a field's on-enter) are valid on
// that element only.
//
// An option is switched off per item by its value: a color attribute
// (bg, border, edge-t, hover-bg, color...) given an empty token applies
// nothing, so border="{{drop}}" outlines only the item whose drop is a
// token; separator="0" applies no separator, unlike Go's Separator(0).
//
// # Finding your way
//
//	Layout      row col stack spacer hrule vrule vrule-inset
//	Text        text caption icon lines note   font-source
//	Buttons     button primary-button link-button icon-button segment-chip segment-icon-chip
//	Rows        list-row form-row check-row
//	Inputs      field (placeholder field-pad styled text-menu on-enter on-tab)
//	            area (text-menu wrap) checkbox
//	            slider (slider-mid on-slide on-slide-end)
//	Viewers     code (wrap) image
//	Containers  scroll hscroll split (vertical as-ratio min-first min-second)
//	Widgets     toolbar tab-bar tabs scroll-tabs table   separator active-tab
//	Dialogs     dialog dialog-title dialog-subtitle dialog-toolbar message-dialog resizable-dialog   focus-scope dismiss-margin keep-on-outside-click
//	Overlays    menu toasts
//	Templates   slot
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
// A larger view is a template string in a const at the end of its Go file,
// parsed once at package level with Must; its name ("row.vml") is only
// what an error message calls it:
//
//	var rowTemplate = markup.Must("row.vml", rowSource)
//
// Load and MustLoad read a template from an fs.FS (an embed.FS or a folder)
// for a program that keeps its templates in files.
//
// The result is the same *view.El tree the Go DSL makes, so both styles
// share one runtime and can be mixed: a template can be a part of a
// hand-built view and the other way round.
//
// A template holds no expressions: bindings are lookups and the only
// directives are if and for, so logic stays in Go and a template stays
// checkable. Methods are not reachable from a scope, since a name would then
// mean two things. Moving a view from Go to a template must not change a
// pixel; golden scenes and pinned view.Dump output are the check.
package markup

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Error is a problem in a template, with where it is.
type Error struct {
	Name string
	Line int
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("%s:%d: %s", e.Name, e.Line, e.Msg) }

// attrKind says what the build does with an attribute, decided at Parse.
type attrKind int

const (
	attrInert   attrKind = iota // a param (read by args) or `for`: no option
	attrIf                      // the `if` directive
	attrHandler                 // names a scope entry that becomes an option
	attrOption                  // a value that becomes an option
)

type attr struct {
	name, value string
	line        int

	kind   attrKind
	parts  interp          // attrOption: the value, read at Parse
	read   attributeReader // attrOption: its option reader
	static bool            // attrOption: the value has no {{path}} and opt is its option, made at Parse
	opt    view.Option     // the static option; nil when the attribute switches itself off
	bind   handlerAttr     // attrHandler
	path   []string        // attrHandler and attrIf: the scope entry, split
	negate bool            // attrIf: `!name`
}

// forLoop is the `for` of an element, read at Parse: "name in list".
type forLoop struct {
	value    string // the attribute, for the error
	name     string
	listExpr string
	path     []string
	valid    bool
}

// paramBinding is a param the element sets: the place of its value in the
// constructor's arguments and the attribute that gives it.
type paramBinding struct {
	index int
	param param
	set   *attr
}

type node struct {
	tag       string
	spec      *elementSpec
	attrs     []attr
	forLoop   *forLoop
	params    []paramBinding
	children  []*node
	text      string // the text between the tags, for elements that hold text
	textParts interp
	line      int
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

// MustBuild is Build for scopes the caller controls: an error (an unknown
// name, a malformed value) is a programming mistake and panics.
func (t *Template) MustBuild(sc Scope, fallbacks ...Scope) view.Node {
	n, err := t.Build(sc, fallbacks...)
	if err != nil {
		panic(err)
	}
	return n
}

// Name is the name the template was parsed with.
func (t *Template) Name() string { return t.name }

// parser is the state of one Parse: the open elements and the root.
type parser struct {
	name    string
	src     []byte
	decoder *xml.Decoder
	stack   []*node
	root    *node
}

// line is the line the decoder has read up to.
func (p *parser) line() int {
	return 1 + bytes.Count(p.src[:min(int(p.decoder.InputOffset()), len(p.src))], []byte("\n"))
}

func (p *parser) fail(line int, format string, args ...any) error {
	return &Error{Name: p.name, Line: line, Msg: fmt.Sprintf(format, args...)}
}

// Parse reads a template. name is used in error messages.
func Parse(name string, src []byte) (*Template, error) {
	decoder := xml.NewDecoder(bytes.NewReader(src))
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity
	p := &parser{name: name, src: src, decoder: decoder}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, p.fail(p.line(), "%v", err)
		}
		if err := p.token(token); err != nil {
			return nil, err
		}
	}
	if p.root == nil {
		return nil, p.fail(1, "the template is empty")
	}
	if len(p.stack) > 0 {
		return nil, p.fail(p.line(), "<%s> is not closed", p.stack[len(p.stack)-1].tag)
	}
	p.root.readTexts()
	return &Template{name: name, root: p.root}, nil
}

// token adds one token of the template to the tree.
func (p *parser) token(token xml.Token) error {
	switch typed := token.(type) {
	case xml.StartElement:
		return p.startElement(typed)
	case xml.EndElement:
		if len(p.stack) == 0 {
			return p.fail(p.line(), "unexpected </%s>", typed.Name.Local)
		}
		p.stack = p.stack[:len(p.stack)-1]
	case xml.CharData:
		return p.charData(string(typed))
	}
	return nil
}

// startElement opens an element: its tag and attributes are checked
// against the table and it is put into its parent, or made the root.
func (p *parser) startElement(start xml.StartElement) error {
	line := p.line()
	tag := start.Name.Local
	spec, known := elements[tag]
	if !known {
		return p.fail(line, "unknown element <%s>%s", tag, hint(tag, elementNames(), "elements"))
	}
	item := &node{tag: tag, line: line}
	for _, set := range start.Attr {
		if err := p.checkAttr(item, spec, set.Name.Local, line); err != nil {
			return err
		}
		item.attrs = append(item.attrs, attr{name: set.Name.Local, value: set.Value, line: line})
	}
	item.prepare(&spec)
	if missing := item.missingParam(spec); missing != "" {
		return p.fail(line, "<%s> needs a %s attribute", tag, missing)
	}
	if err := p.attach(item, line); err != nil {
		return err
	}
	p.stack = append(p.stack, item)
	return nil
}

// checkAttr is an error when the element does not take the attribute.
func (p *parser) checkAttr(item *node, spec elementSpec, name string, line int) error {
	if spec.takes(name) {
		return nil
	}
	if spec.plain {
		return p.fail(line, "unknown attribute %q on <%s>: it takes only %s, if and for", name, item.tag, strings.Join(spec.paramNames(), ", "))
	}
	return p.fail(line, "unknown attribute %q on <%s>%s", name, item.tag, hint(name, append(attributeNames(), spec.ownNames()...), "attributes"))
}

// attach puts item into the open element, or makes it the root.
func (p *parser) attach(item *node, line int) error {
	if len(p.stack) == 0 {
		if p.root != nil {
			return p.fail(line, "a template has one root element")
		}
		p.root = item
		return nil
	}
	parent := p.stack[len(p.stack)-1]
	if elements[parent.tag].holds != holdsChildren {
		return p.fail(line, "<%s> cannot contain <%s>", parent.tag, item.tag)
	}
	parent.children = append(parent.children, item)
	return nil
}

// charData adds the text between tags to the open element, which must hold
// text. Whitespace between elements is dropped.
func (p *parser) charData(data string) error {
	text := strings.TrimSpace(data)
	if text == "" {
		return nil
	}
	if len(p.stack) == 0 {
		return p.fail(p.line(), "text outside the root element")
	}
	parent := p.stack[len(p.stack)-1]
	if elements[parent.tag].holds != holdsText {
		return p.fail(p.line(), "text is only allowed inside <text>, <caption> and <button>, not <%s>", parent.tag)
	}
	if parent.text != "" {
		parent.text += " "
	}
	parent.text += strings.Join(strings.Fields(text), " ")
	return nil
}

// readTexts reads the text of the element and of its children.
func (item *node) readTexts() {
	item.textParts = parseInterp(item.text)
	for _, child := range item.children {
		child.readTexts()
	}
}

// prepare decides, once, what a build does with each attribute of item:
// everything that does not depend on the scope is read here. A static
// attribute whose value is bad is left to the build, which reports it with
// its line, and only when the element is shown. The options an attribute
// reader makes hold only the values they were made from and are applied to
// an element without changing themselves, so one option serves every build.
func (item *node) prepare(spec *elementSpec) {
	item.spec = spec
	for i := range item.attrs {
		set := &item.attrs[i]
		switch {
		case set.name == "for":
			item.forLoop = readFor(set.value)
		case set.name == "if":
			set.kind = attrIf
			set.negate, set.path = splitCondition(set.value)
		case isParam(spec, set.name) || directives[set.name]:
			set.kind = attrInert
		default:
			set.kind = attrOption
			set.prepareOption(spec)
		}
		item.bindParam(set)
	}
}

// bindParam notes that set gives the value of a param, if it is one.
func (item *node) bindParam(set *attr) {
	for index, wanted := range item.spec.params {
		if wanted.name == set.name {
			item.params = append(item.params, paramBinding{index: index, param: wanted, set: set})
		}
	}
}

func isParam(spec *elementSpec, name string) bool {
	_, found := spec.param(name)
	return found
}

// prepareOption reads a handler or an option attribute.
func (set *attr) prepareOption(spec *elementSpec) {
	if bind, ok := spec.handler(set.name); ok {
		set.kind, set.bind, set.path = attrHandler, bind, splitPath(set.value)
		return
	}
	set.read, _ = spec.option(set.name) // Parse checked it exists
	set.parts = parseInterp(set.value)
	if !set.parts.isLiteral() {
		return
	}
	opt, err := set.read(set.value)
	if err != nil {
		return
	}
	set.static, set.opt = true, opt
}

// readFor reads `item in list`; an invalid one is reported by the build.
func readFor(value string) *forLoop {
	name, listExpr, ok := strings.Cut(value, " in ")
	name = strings.TrimSpace(name)
	return &forLoop{value: value, name: name, listExpr: strings.TrimSpace(listExpr), path: splitPath(listExpr), valid: ok && name != ""}
}
