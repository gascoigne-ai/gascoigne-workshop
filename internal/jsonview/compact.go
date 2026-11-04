package jsonview

import (
	"bytes"
	"encoding/json"
	"strings"
)

// node is a parsed JSON value that remembers key order and the exact
// text of its scalars.
type node struct {
	scalar   string // the value as written, for strings, numbers, true, false, null
	open     byte   // '{' or '[' for a container, 0 for a scalar
	keys     []string
	children []*node
}

func parse(dec *json.Decoder) (*node, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Delim:
		n := &node{open: byte(v)}
		for dec.More() {
			if v == '{' {
				k, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := k.(string)
				n.keys = append(n.keys, key)
			}
			child, err := parse(dec)
			if err != nil {
				return nil, err
			}
			n.children = append(n.children, child)
		}
		if _, err := dec.Token(); err != nil { // the closing delimiter
			return nil, err
		}
		return n, nil
	case string:
		b, _ := json.Marshal(v)
		return &node{scalar: unescapeHTML(string(b))}, nil
	case json.Number:
		return &node{scalar: v.String()}, nil
	case bool:
		if v {
			return &node{scalar: "true"}, nil
		}
		return &node{scalar: "false"}, nil
	}
	return &node{scalar: "null"}, nil
}

// unescapeHTML undoes json.Marshal's < style escapes for < > &.
func unescapeHTML(s string) string {
	return strings.NewReplacer(`<`, "<", `>`, ">", `&`, "&").Replace(s)
}

func quote(key string) string {
	b, _ := json.Marshal(key)
	return unescapeHTML(string(b))
}

// inline is n on one line: {"a": 1, "b": [1, 2]}.
func (n *node) inline() string {
	if n.open == 0 {
		return n.scalar
	}
	if len(n.children) == 0 {
		return string(n.open) + string(closer(n.open))
	}
	parts := make([]string, len(n.children))
	for i, c := range n.children {
		parts[i] = c.inline()
		if n.open == '{' {
			parts[i] = quote(n.keys[i]) + ": " + parts[i]
		}
	}
	return string(n.open) + strings.Join(parts, ", ") + string(closer(n.open))
}

func closer(open byte) byte {
	if open == '{' {
		return '}'
	}
	return ']'
}

// previewMax is how long the one-line preview of a folded block may be,
// not counting its indentation.
const previewMax = 80

// preview is the inline form cut at previewMax, ending in "…}" or "…]".
func (n *node) preview() string {
	s := n.inline()
	if len(s) <= previewMax {
		return s
	}
	cut := []rune(s)[:previewMax-2]
	// Do not cut inside a key/value pair's string quote more than needed.
	return strings.TrimRight(string(cut), " ,") + ", …" + string(closer(n.open))
}

// Compact formats body as compactly as it can while staying readable,
// the way a browser's JSON viewer does: every object or array whose
// one-line form fits in width columns (with its indent and key) stays on
// one line, and only larger ones are broken up. Block opening lines are
// foldable, carry their one-line preview in Folded for when they are
// folded, and every line knows its nesting Depth. A body that is not JSON
// gives its plain lines.
func Compact(body []byte, width int) []Line {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	root, err := parse(dec)
	if err != nil || !json.Valid(body) {
		return plain(string(body))
	}
	var lines []Line
	var emit func(n *node, depth int, prefix, suffix string)
	emit = func(n *node, depth int, prefix, suffix string) {
		indent := strings.Repeat("  ", depth)
		one := indent + prefix + n.inline() + suffix
		if n.open == 0 || len(n.children) == 0 || len(one) <= width {
			lines = append(lines, Line{Spans: tokenizeLine(one), Depth: depth})
			return
		}
		open := indent + prefix + string(n.open)
		lines = append(lines, Line{
			Spans:    tokenizeLine(open),
			Foldable: true,
			Depth:    depth,
			Folded:   tokenizeLine(indent + prefix + n.preview() + suffix),
		})
		for i, c := range n.children {
			cp, cs := "", ","
			if i == len(n.children)-1 {
				cs = ""
			}
			if n.open == '{' {
				cp = quote(n.keys[i]) + ": "
			}
			emit(c, depth+1, cp, cs)
		}
		lines = append(lines, Line{Spans: tokenizeLine(indent + string(closer(n.open)) + suffix), Depth: depth})
	}
	emit(root, 0, "", "")
	return lines
}
