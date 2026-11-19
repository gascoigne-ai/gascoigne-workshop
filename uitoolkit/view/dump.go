package view

import (
	"fmt"
	"strconv"
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
	for _, attribute := range dumpedAttributes {
		if value := attribute.get(element); value != "" {
			fmt.Fprintf(sb, " %s=%s", attribute.name, value)
		}
	}
	if describer, ok := element.b.(Describer); ok {
		sb.WriteString(" " + describer.Describe())
	}
	if element.rect.W != 0 || element.rect.H != 0 {
		fmt.Fprintf(sb, " rect=%d,%d,%d,%d", element.rect.X, element.rect.Y, element.rect.W, element.rect.H)
	}
	sb.WriteByte('\n')
	for _, k := range element.kids {
		dump(sb, k, depth+1)
	}
}

// dumpAttribute is one attribute of an element that Dump prints: get
// returns its value as text, "" when the element has the zero value, so
// an element prints only what it uses.
type dumpAttribute struct {
	name string
	get  func(element *El) string
}

// dumpedAttributes holds every attribute of El that shapes the look or
// the behavior, in the order Dump prints them; a new field of El gets a
// row here. Only the layout's own state (hidden, rect) is left out: rect
// is printed separately, hidden follows from Optional and the width.
var dumpedAttributes = []dumpAttribute{
	{"text", func(element *El) string {
		if element.text == "" {
			return ""
		}
		return fmt.Sprintf("%q", element.text)
	}},
	{"font", func(element *El) string {
		if element.fontSource == nil {
			return ""
		}
		return "custom"
	}},
	numberAttribute("w", func(element *El) int { return element.w }),
	numberAttribute("h", func(element *El) int { return element.h }),
	numberAttribute("padL", func(element *El) int { return element.padL }),
	numberAttribute("padT", func(element *El) int { return element.padT }),
	numberAttribute("padR", func(element *El) int { return element.padR }),
	numberAttribute("padB", func(element *El) int { return element.padB }),
	numberAttribute("borderInset", func(element *El) int { return element.borderInset }),
	numberAttribute("floatTop", func(element *El) int { return element.fl.dumpTop() }),
	numberAttribute("floatRight", func(element *El) int { return element.fl.dumpRight() }),
	numberAttribute("floatMode", func(element *El) int { return int(element.fl.mode) }),
	numberAttribute("mL", func(element *El) int { return element.mL }),
	numberAttribute("mT", func(element *El) int { return element.mT }),
	numberAttribute("mR", func(element *El) int { return element.mR }),
	numberAttribute("mB", func(element *El) int { return element.mB }),
	numberAttribute("gap", func(element *El) int { return element.gap }),
	numberAttribute("minW", func(element *El) int { return element.minW }),
	numberAttribute("maxW", func(element *El) int { return element.maxW }),
	numberAttribute("minH", func(element *El) int { return element.minH }),
	numberAttribute("maxH", func(element *El) int { return element.maxH }),
	numberAttribute("dismissMargin", func(element *El) int { return element.dismissMargin }),
	{"grow", func(element *El) string {
		if element.grow == 0 {
			return ""
		}
		return fmt.Sprint(element.grow)
	}},
	textAttribute("bg", func(element *El) string { return string(element.bg) }),
	textAttribute("border", func(element *El) string { return string(element.border) }),
	textAttribute("borderT", func(element *El) string { return string(element.borderT) }),
	textAttribute("borderB", func(element *El) string { return string(element.borderB) }),
	textAttribute("borderL", func(element *El) string { return string(element.borderL) }),
	textAttribute("borderR", func(element *El) string { return string(element.borderR) }),
	textAttribute("fg", func(element *El) string { return string(element.fg) }),
	textAttribute("control", func(element *El) string { return element.control }),
	textAttribute("tip", func(element *El) string { return element.tip }),
	textAttribute("hoverBg", func(element *El) string { return string(element.hoverBg) }),
	textAttribute("hoverColor", func(element *El) string { return string(element.hoverColor) }),
	textAttribute("edge", func(element *El) string { return string(element.edgeTok) }),
	textAttribute("roundCtl", func(element *El) string { return element.roundCtl }),
	numberAttribute("roundCorners", func(element *El) int { return int(element.roundCorners) }),
	numberAttribute("alignItems", func(element *El) int { return int(element.alignItems) }),
	numberAttribute("alignSelf", func(element *El) int { return int(element.alignSelf) }),
	numberAttribute("justify", func(element *El) int { return int(element.justify) }),
	flagAttribute("small", func(element *El) bool { return element.small }),
	flagAttribute("bold", func(element *El) bool { return element.bold }),
	flagAttribute("mono", func(element *El) bool { return element.mono }),
	flagAttribute("ellipsis", func(element *El) bool { return element.ellipsis }),
	flagAttribute("center", func(element *El) bool { return element.ctr }),
	flagAttribute("optional", func(element *El) bool { return element.optional }),
	flagAttribute("shrink", func(element *El) bool { return element.shrink }),
	flagAttribute("float", func(element *El) bool { return element.fl.on }),
	flagAttribute("edgeTop", func(element *El) bool { return element.edgeTop }),
	flagAttribute("edgeLeft", func(element *El) bool { return element.edgeLeft }),
	flagAttribute("edgeRight", func(element *El) bool { return element.edgeRight }),
	flagAttribute("lighten", func(element *El) bool { return element.lighten }),
	flagAttribute("onClick", func(element *El) bool { return element.onClick != nil }),
	flagAttribute("onRightClick", func(element *El) bool { return element.onRight != nil }),
	flagAttribute("onMiddleClick", func(element *El) bool { return element.onMiddle != nil }),
	flagAttribute("onDoubleClick", func(element *El) bool { return element.onDouble != nil }),
	flagAttribute("onDragOver", func(element *El) bool { return element.onDragOver != nil }),
	flagAttribute("onChange", func(element *El) bool { return element.onChange != nil }),
	flagAttribute("modal", func(element *El) bool { return element.modal }),
	flagAttribute("focusScope", func(element *El) bool { return element.focusScope }),
	flagAttribute("shadow", func(element *El) bool { return element.shadow }),
	flagAttribute("onSelect", func(element *El) bool { return element.onSelect != nil }),
	flagAttribute("onDismiss", func(element *El) bool { return element.onDismiss != nil }),
	flagAttribute("clip", func(element *El) bool { return element.clip }),
	flagAttribute("dragHandle", func(element *El) bool { return element.dragHandle }),
	flagAttribute("reveal", func(element *El) bool { return element.reveal }),
	flagAttribute("moved", func(element *El) bool { return element.fl.moved != nil }),
}

func numberAttribute(name string, get func(element *El) int) dumpAttribute {
	return dumpAttribute{name, func(element *El) string {
		if value := get(element); value != 0 {
			return strconv.Itoa(value)
		}
		return ""
	}}
}

func textAttribute(name string, get func(element *El) string) dumpAttribute {
	return dumpAttribute{name, get}
}

func flagAttribute(name string, get func(element *El) bool) dumpAttribute {
	return dumpAttribute{name, func(element *El) string {
		if get(element) {
			return "true"
		}
		return ""
	}}
}

// kindName is the type of the element's behavior without its package,
// so a component prints the same wherever it lives.
func kindName(element *El) string {
	name := strings.TrimPrefix(fmt.Sprintf("%T", element.b), "*")
	return name[strings.LastIndex(name, ".")+1:]
}
