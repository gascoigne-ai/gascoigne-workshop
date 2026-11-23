package markup

import (
	"fmt"
	"slices"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Build makes the view tree for scope. A name is read from the variables of
// the `for` elements around it (innermost first, `$index` before the name of
// the loop), then from scope, then from each fallback in order; an entry that
// holds nil still shadows the ones after it. A build reads the scopes and
// never writes into one, so the caller may share a fallback between builds
// (the fixed sizes of the app, see ui/sizes.Build) instead of merging it into
// a new map for each. A root whose `if` is false gives a nil node, which view
// ignores.
//
// A `for` does not copy the scope: it binds its variable in an env that
// lives for the build, one binding per loop, written again for each item.
// That is safe because nothing a build makes keeps the env (the handlers and
// values it hands to view are read out of the scope when they are built).
// Rejected: a scope copy per item, which made a loop cost the size of the
// scope for every item. Revisit when a build output keeps a reference to the
// scope; then one binding per item.
func (t *Template) Build(sc Scope, fallbacks ...Scope) (view.Node, error) {
	n, err := t.build(t.root, env{own: sc, fallbacks: fallbacks})
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (t *Template) errf(item *node, format string, args ...any) error {
	return &Error{Name: t.name, Line: item.line, Msg: fmt.Sprintf(format, args...)}
}

// build makes the element of n, or a fragment for a `for`.
func (t *Template) build(item *node, sc env) (view.Node, error) {
	loop := item.forLoop
	if loop == nil {
		return t.buildOne(item, sc)
	}
	if !loop.valid {
		return nil, t.errf(item, `for must read "item in list", not %q`, loop.value)
	}
	v, err := lookupPath(loop.path, sc)
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	items, err := asList(v)
	if err != nil {
		return nil, t.errf(item, "%s: %v", loop.listExpr, err)
	}
	nodes := make([]view.Item, 0, items.size)
	bound := &binding{name: loop.name, outer: sc.loop}
	inner := sc
	inner.loop = bound
	for i := 0; i < items.size; i++ {
		bound.item, bound.index = items.at(i), i
		el, err := t.buildOne(item, inner)
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
// the order of the attributes.
func (t *Template) args(element *node, scope env) ([]any, error) {
	values := make([]any, len(element.spec.params))
	for _, bound := range element.params {
		value, err := bound.param.read(bound.set.value, scope)
		if err != nil {
			return nil, t.errf(element, "%v", err)
		}
		values[bound.index] = value
	}
	return values, nil
}

// bindHandler reads the scope function an event attribute names.
func (t *Template) bindHandler(item *node, set *attr, sc env) (view.Option, error) {
	value, err := lookupPath(set.path, sc)
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	opt, err := set.bind.bind(value)
	if err != nil {
		return nil, t.errf(item, "%s=%q %v", set.name, set.value, err)
	}
	return opt, nil
}

// buildOne makes the element of item once, honoring `if`.
func (t *Template) buildOne(item *node, sc env) (view.Node, error) {
	opts, shown, err := t.options(item, sc)
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
	text, err := item.textParts.eval(sc)
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	args, err := t.args(item, sc)
	if err != nil {
		return nil, err
	}
	built, err := item.spec.build(elementParts{args: args, opts: opts, kids: kids, text: text})
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	return built, nil
}

// options reads the attributes of item that are options or handlers. It
// reports false when `if` hides the element.
func (t *Template) options(item *node, sc env) (opts []view.Option, shown bool, err error) {
	for i := range item.attrs {
		set := &item.attrs[i]
		switch set.kind {
		case attrIf:
			show, err := truthyPath(set.negate, set.path, sc)
			if err != nil {
				return nil, false, t.errf(item, "%v", err)
			}
			if !show {
				return nil, false, nil
			}
		case attrHandler:
			opt, err := t.bindHandler(item, set, sc)
			if err != nil {
				return nil, false, err
			}
			opts = append(opts, opt)
		case attrOption:
			opt, err := t.option(item, set, sc)
			if err != nil {
				return nil, false, err
			}
			if opt != nil {
				opts = append(opts, opt)
			}
		}
	}
	return opts, true, nil
}

// option is the option an attribute makes, nil when it is switched off.
func (t *Template) option(item *node, set *attr, sc env) (view.Option, error) {
	if set.static {
		return set.opt, nil
	}
	value, err := set.parts.eval(sc)
	if err != nil {
		return nil, t.errf(item, "%v", err)
	}
	opt, err := set.read(value)
	if err != nil {
		return nil, t.errf(item, "%s: %v", set.name, err)
	}
	return opt, nil
}
