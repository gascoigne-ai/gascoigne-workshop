package markup

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Build makes the view tree for scope. A root whose `if` is false gives a
// nil node, which view ignores.
func (t *Template) Build(sc Scope) (view.Node, error) {
	n, err := t.build(t.root, sc)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (t *Template) errf(item *node, format string, args ...any) error {
	return &Error{Name: t.name, Line: item.line, Msg: fmt.Sprintf(format, args...)}
}

// build makes the element of n, or a fragment for a `for`.
func (t *Template) build(item *node, sc Scope) (view.Node, error) {
	var forAttr *attr
	for i := range item.attrs {
		if item.attrs[i].name == "for" {
			forAttr = &item.attrs[i]
		}
	}
	if forAttr == nil {
		return t.buildOne(item, sc)
	}
	name, listExpr, ok := strings.Cut(forAttr.value, " in ")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return nil, t.errf(item, `for must read "item in list", not %q`, forAttr.value)
	}
	v, err := lookup(strings.TrimSpace(listExpr), sc)
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	list, err := asList(v)
	if err != nil {
		return nil, t.errf(item, "%s: %v", strings.TrimSpace(listExpr), err)
	}
	var nodes []view.Item
	for i, it := range list {
		child := Scope{}
		maps.Copy(child, sc)
		child[name], child["$index"] = it, i
		el, err := t.buildOne(item, child)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, el)
	}
	return view.Frag(nodes...), nil
}

// missingParam is the first param of spec that element does not set, ""
// when it sets them all.
func (element *node) missingParam(spec elementSpec) string {
	for _, wanted := range spec.params {
		if !slices.ContainsFunc(element.attrs, func(set attr) bool { return set.name == wanted.name }) {
			return wanted.name
		}
	}
	return ""
}

// args reads the values of the element's params from its attributes, in
// the order of the params.
func (t *Template) args(element *node, spec elementSpec, scope Scope) ([]any, error) {
	values := make([]any, len(spec.params))
	for _, set := range element.attrs {
		for i, wanted := range spec.params {
			if set.name != wanted.name {
				continue
			}
			value, err := wanted.read(set.value, scope)
			if err != nil {
				return nil, t.errf(element, "%v", err)
			}
			values[i] = value
		}
	}
	return values, nil
}

// bindHandler reads the scope function an event attribute names.
func (t *Template) bindHandler(item *node, attribute attr, handler handlerAttr, sc Scope) (view.Option, error) {
	value, err := lookup(strings.TrimSpace(attribute.value), sc)
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	opt, err := handler.bind(value)
	if err != nil {
		return nil, t.errf(item, "%s=%q %v", attribute.name, attribute.value, err)
	}
	return opt, nil
}

// buildOne makes the element of item once, honoring `if`.
func (t *Template) buildOne(item *node, sc Scope) (view.Node, error) {
	spec := elements[item.tag]
	opts, shown, err := t.options(item, spec, sc)
	if err != nil || !shown {
		return nil, err
	}
	var kids []view.Item
	for _, child := range item.children {
		built, err := t.build(child, sc)
		if err != nil {
			return nil, err
		}
		kids = append(kids, built)
	}
	text, err := interpolate(item.text, sc)
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	args, err := t.args(item, spec, sc)
	if err != nil {
		return nil, err
	}
	built, err := spec.build(elementParts{args: args, opts: opts, kids: kids, text: text})
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	return built, nil
}

// options reads the attributes of item that are options or handlers. It
// reports false when `if` hides the element.
func (t *Template) options(item *node, spec elementSpec, sc Scope) (opts []view.Option, shown bool, err error) {
	for _, set := range item.attrs {
		if set.name == "if" {
			show, err := truthy(set.value, sc)
			if err != nil {
				return nil, false, t.errf(item, "%v", err)
			}
			if !show {
				return nil, false, nil
			}
			continue
		}
		opt, err := t.option(item, spec, set, sc)
		if err != nil {
			return nil, false, err
		}
		if opt != nil {
			opts = append(opts, opt)
		}
	}
	return opts, true, nil
}

// option is the option an attribute makes: nil for a directive, a param
// (read by args) or an option that is switched off.
func (t *Template) option(item *node, spec elementSpec, set attr, sc Scope) (view.Option, error) {
	if _, isParam := spec.param(set.name); isParam || directives[set.name] {
		return nil, nil
	}
	if bind, ok := spec.handler(set.name); ok {
		return t.bindHandler(item, set, bind, sc)
	}
	value, err := interpolate(set.value, sc)
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	read, _ := spec.option(set.name) // Parse checked it exists
	opt, err := read(value)
	if err != nil {
		return nil, t.errf(item, "%s: %v", set.name, err)
	}
	return opt, nil
}
