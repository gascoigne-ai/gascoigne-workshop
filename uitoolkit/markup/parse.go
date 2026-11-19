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
// values may use {{path}} to read the Scope, where each step of the path
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
//	Buttons     button primary-button link-button icon-button choice
//	Rows        list-row form-row check-row
//	Inputs      field (placeholder field-pad segments text-menu on-enter on-tab)
//	            area (text-menu wrap) checkbox
//	            slider (slider-mid on-slide on-slide-end)
//	Viewers     code (wrap) image
//	Containers  scroll hscroll split (vertical as-ratio min-first min-second)
//	Widgets     tab-bar tabs scroll-tabs table   separator active-tab
//	Dialogs     dialog dialog-title dialog-toolbar message-dialog resizable-dialog   focus-scope dismiss-margin
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

// Parse reads a template. name is used in error messages.
func Parse(name string, src []byte) (*Template, error) {
	d := xml.NewDecoder(bytes.NewReader(src))
	d.Strict = false
	d.Entity = xml.HTMLEntity
	line := func() int { return 1 + bytes.Count(src[:min(int(d.InputOffset()), len(src))], []byte("\n")) }
	fail := func(value int, format string, args ...any) (*Template, error) {
		return nil, &Error{Name: name, Line: value, Msg: fmt.Sprintf(format, args...)}
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
				if standsIn[n.tag] && a.Name.Local != "name" && !directives[a.Name.Local] {
					return fail(l, "unknown attribute %q on <%s>: it takes only name, if and for", a.Name.Local, n.tag)
				}
				_, isParam := spec.param(a.Name.Local)
				_, isHandler := spec.handler(a.Name.Local)
				if _, ok := spec.option(a.Name.Local); !ok && !isParam && !directives[a.Name.Local] && !isHandler && !(a.Name.Local == "name" && (n.tag == "icon" || n.tag == "slot")) {
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
