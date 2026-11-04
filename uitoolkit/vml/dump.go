package vml

import (
	"fmt"
	"strings"
)

// Dump describes a view tree as indented text: each element's kind,
// content, attributes that differ from the defaults and, once the tree
// has been laid out, its rectangle. Two trees that dump the same look the
// same, which is how tests compare a hand-built view with one loaded from
// a template.
func Dump(n Node) string {
	var sb strings.Builder
	dump(&sb, n, 0)
	return sb.String()
}

func dump(sb *strings.Builder, e *El, depth int) {
	if e == nil {
		return
	}
	fmt.Fprintf(sb, "%s%s", strings.Repeat("  ", depth), strings.TrimPrefix(fmt.Sprintf("%T", e.b), "vml."))
	add := func(name string, v any) { fmt.Fprintf(sb, " %s=%v", name, v) }
	if e.text != "" {
		add("text", fmt.Sprintf("%q", e.text))
	}
	for _, f := range []struct {
		name string
		v    int
	}{
		{"w", e.w}, {"h", e.h}, {"padL", e.padL}, {"padT", e.padT}, {"padR", e.padR}, {"padB", e.padB}, {"borderInset", e.borderInset}, {"floatTop", e.fl.top}, {"floatRight", e.fl.right}, {"floatMode", e.fl.mode}, {"mL", e.mL}, {"mT", e.mT}, {"mR", e.mR}, {"mB", e.mB}, {"gap", e.gap},
	} {
		if f.v != 0 {
			add(f.name, f.v)
		}
	}
	if e.grow != 0 {
		add("grow", e.grow)
	}
	for _, f := range []struct {
		name string
		v    string
	}{
		{"bg", string(e.bg)}, {"border", string(e.border)}, {"borderT", string(e.borderT)},
		{"borderB", string(e.borderB)}, {"borderL", string(e.borderL)}, {"borderR", string(e.borderR)}, {"fg", string(e.fg)},
		{"control", e.control}, {"tip", e.tip}, {"hoverBg", string(e.hoverBg)}, {"edge", string(e.edgeTok)},
	} {
		if f.v != "" {
			add(f.name, f.v)
		}
	}
	for _, f := range []struct {
		name string
		v    bool
	}{
		{"small", e.small}, {"bold", e.bold}, {"mono", e.mono}, {"ellipsis", e.ellipsis}, {"center", e.ctr},
		{"optional", e.optional}, {"float", e.fl.on}, {"edgeTop", e.edgeTop}, {"edgeLeft", e.edgeLeft}, {"edgeRight", e.edgeRight}, {"lighten", e.lighten}, {"onClick", e.onClick != nil},
	} {
		if f.v {
			add(f.name, true)
		}
	}
	if e.rect.W != 0 || e.rect.H != 0 {
		fmt.Fprintf(sb, " rect=%d,%d,%d,%d", e.rect.X, e.rect.Y, e.rect.W, e.rect.H)
	}
	sb.WriteByte('\n')
	for _, k := range e.kids {
		dump(sb, k, depth+1)
	}
}
