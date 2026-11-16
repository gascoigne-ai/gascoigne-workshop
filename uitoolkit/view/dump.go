package view

import (
	"fmt"
	"strings"
)

// Dump describes a view tree as indented text: each element's kind,
// content, attributes that differ from the defaults and, once the tree
// has been laid out, its rectangle. Two trees that dump the same look the
// same, which is how tests compare a hand-built view with one loaded from
// a template.
func Dump(node Node) string {
	var sb strings.Builder
	dump(&sb, node, 0)
	return sb.String()
}

func dump(sb *strings.Builder, element *El, depth int) {
	if element == nil {
		return
	}
	fmt.Fprintf(sb, "%s%s", strings.Repeat("  ", depth), kindName(element))
	add := func(name string, value any) { fmt.Fprintf(sb, " %s=%v", name, value) }
	if element.text != "" {
		add("text", fmt.Sprintf("%q", element.text))
	}
	if element.fontSource != nil {
		add("font", "custom")
	}
	for _, f := range []struct {
		name string
		v    int
	}{
		{"w", element.w}, {"h", element.h}, {"padL", element.padL}, {"padT", element.padT}, {"padR", element.padR}, {"padB", element.padB}, {"borderInset", element.borderInset}, {"floatTop", element.fl.top}, {"floatRight", element.fl.right}, {"floatMode", element.fl.mode}, {"mL", element.mL}, {"mT", element.mT}, {"mR", element.mR}, {"mB", element.mB}, {"gap", element.gap},
	} {
		if f.v != 0 {
			add(f.name, f.v)
		}
	}
	if element.grow != 0 {
		add("grow", element.grow)
	}
	for _, f := range []struct {
		name string
		v    string
	}{
		{"bg", string(element.bg)}, {"border", string(element.border)}, {"borderT", string(element.borderT)},
		{"borderB", string(element.borderB)}, {"borderL", string(element.borderL)}, {"borderR", string(element.borderR)}, {"fg", string(element.fg)},
		{"control", element.control}, {"tip", element.tip}, {"hoverBg", string(element.hoverBg)}, {"hoverColor", string(element.hoverColor)}, {"edge", string(element.edgeTok)},
	} {
		if f.v != "" {
			add(f.name, f.v)
		}
	}
	for _, f := range []struct {
		name string
		v    bool
	}{
		{"small", element.small}, {"bold", element.bold}, {"mono", element.mono}, {"ellipsis", element.ellipsis}, {"center", element.ctr},
		{"optional", element.optional}, {"shrink", element.shrink}, {"float", element.fl.on}, {"edgeTop", element.edgeTop}, {"edgeLeft", element.edgeLeft}, {"edgeRight", element.edgeRight}, {"lighten", element.lighten}, {"onClick", element.onClick != nil}, {"onClickAt", element.onClickAt != nil}, {"onRightClick", element.onRight != nil}, {"onRightClickIn", element.onRightIn != nil},
		{"onMiddleClick", element.onMiddle != nil}, {"onDoubleClick", element.onDouble != nil}, {"onDragOver", element.onDragOver != nil}, {"onChange", element.onChange != nil},
		{"modal", element.modal}, {"focusScope", element.focusScope}, {"shadow", element.shadow}, {"onSelect", element.onSelect != nil}, {"onDismiss", element.onDismiss != nil},
	} {
		if f.v {
			add(f.name, true)
		}
	}
	if custom, ok := element.b.(customB); ok {
		if describer, ok := custom.Behavior.(Describer); ok {
			sb.WriteString(" " + describer.Describe())
		}
	}
	if element.rect.W != 0 || element.rect.H != 0 {
		fmt.Fprintf(sb, " rect=%d,%d,%d,%d", element.rect.X, element.rect.Y, element.rect.W, element.rect.H)
	}
	sb.WriteByte('\n')
	for _, k := range element.kids {
		dump(sb, k, depth+1)
	}
}

// kindName is the type of the element's behavior without its package,
// so a component prints the same wherever it lives.
func kindName(element *El) string {
	var behavior any = element.b
	if custom, ok := element.b.(customB); ok {
		behavior = custom.Behavior
	}
	name := strings.TrimPrefix(fmt.Sprintf("%T", behavior), "*")
	return name[strings.LastIndex(name, ".")+1:]
}
