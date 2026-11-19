package markup

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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

// buildOne makes the element of n once, honoring `if`.
func (t *Template) buildOne(nodeValue *node, sc Scope) (view.Node, error) {
	var opts []view.Option
	for _, a := range nodeValue.attrs {
		switch a.name {
		case "for":
		case "if":
			ok, err := truthy(a.value, sc)
			if err != nil {
				return nil, t.errf(nodeValue, "%v", err)
			}
			if !ok {
				return nil, nil
			}
		case "name": // the icon's name, which is its text
			continue
		default:
			if _, isParam := elements[nodeValue.tag].param(a.name); isParam {
				continue // read by args
			}
			if h, ok := elements[nodeValue.tag].handler(a.name); ok {
				opt, err := t.bindHandler(nodeValue, a, h, sc)
				if err != nil {
					return nil, err
				}
				opts = append(opts, opt)
				continue
			}
			val, err := interpolate(a.value, sc)
			if err != nil {
				return nil, t.errf(nodeValue, "%v", err)
			}
			read, _ := elements[nodeValue.tag].option(a.name) // Parse checked it exists
			opt, err := read(val)
			if err != nil {
				return nil, t.errf(nodeValue, "%s: %v", a.name, err)
			}
			if opt != nil {
				opts = append(opts, opt)
			}
		}
	}
	var kids []view.Item
	for _, c := range nodeValue.children {
		k, err := t.build(c, sc)
		if err != nil {
			return nil, err
		}
		kids = append(kids, k)
	}
	text, err := interpolate(nodeValue.text, sc)
	if err != nil {
		return nil, t.errf(nodeValue, "%v", err)
	}
	if nodeValue.tag == "slot" { // a node the caller made, as a Vue slot
		for _, a := range nodeValue.attrs {
			if a.name == "name" {
				// a path, so each item of a for can bring its own node
				v, err := lookup(strings.TrimSpace(a.value), sc)
				if err != nil {
					return nil, t.errf(nodeValue, "slot: %v", err)
				}
				node, ok := v.(view.Node)
				if !ok {
					return nil, t.errf(nodeValue, "slot %q is not a view.Node", a.value)
				}
				return node, nil
			}
		}
		return nil, t.errf(nodeValue, `<slot> needs a name`)
	}
	if nodeValue.tag == "icon" {
		for _, a := range nodeValue.attrs {
			if a.name == "name" {
				if text, err = interpolate(a.value, sc); err != nil {
					return nil, t.errf(nodeValue, "%v", err)
				}
			}
		}
		if !widget.HasIcon(text) {
			return nil, t.errf(nodeValue, "no icon %q; icons: %s", text, strings.Join(widget.IconNames(), " "))
		}
	}
	args, err := t.args(nodeValue, elements[nodeValue.tag], sc)
	if err != nil {
		return nil, err
	}
	built, err := elements[nodeValue.tag].build(elementParts{args: args, opts: opts, kids: kids, text: text})
	if err != nil {
		return nil, t.errf(nodeValue, "%v", err)
	}
	return built, nil
}
