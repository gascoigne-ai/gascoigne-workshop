// Package miniyaml reads the part of YAML that API descriptions use:
// block mappings and sequences by indentation, plain, single- and
// double-quoted scalars, flow sequences and mappings on one line, literal
// (|) and folded (>) block scalars, comments and document markers. It does
// not read anchors, aliases, tags, merge keys, multi-line flow collections
// or multiple documents, and says so with a line number instead of guessing.
//
// The result is what encoding/json would give for the same data
// (map[string]any, []any, string, float64, bool, nil).
package miniyaml

import (
	"fmt"
	"strconv"
	"strings"
)

type line struct {
	no     int // 1-based line number in the input
	indent int
	text   string // without indent and comment
}

// Error is a problem in the input, with where it is.
type Error struct {
	Line int
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

type parser struct {
	lines []line
	pos   int
	raw   []string // the input lines, for block scalars
}

// Parse reads one YAML document.
func Parse(data []byte) (any, error) {
	p := &parser{raw: strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")}
	for i, l := range p.raw {
		if strings.Contains(l, "\t") && strings.HasPrefix(strings.TrimLeft(l, " "), "\t") {
			return nil, &Error{i + 1, "tabs cannot indent YAML"}
		}
		text := stripComment(l)
		trimmed := strings.TrimSpace(text)
		if trimmed == "" || trimmed == "---" || trimmed == "..." {
			continue
		}
		p.lines = append(p.lines, line{no: i + 1, indent: len(text) - len(strings.TrimLeft(text, " ")), text: strings.TrimRight(strings.TrimLeft(text, " "), " ")})
	}
	if len(p.lines) == 0 {
		return nil, &Error{1, "the document is empty"}
	}
	v, err := p.block(p.lines[0].indent)
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.lines) {
		return nil, &Error{p.lines[p.pos].no, "unexpected content (is the indentation consistent?)"}
	}
	return v, nil
}

// stripComment removes a # comment that is not inside quotes and starts
// at the beginning of the line or after a space.
func stripComment(s string) string {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			if i == 0 || strings.ContainsRune(" [{,:-", rune(s[i-1])) {
				quote = c
			}
		case c == '#' && (i == 0 || s[i-1] == ' '):
			return s[:i]
		}
	}
	return s
}

// block parses the collection that starts at the current line, whose
// lines are indented exactly indent.
func (p *parser) block(indent int) (any, error) {
	l := p.lines[p.pos]
	if l.text == "-" || strings.HasPrefix(l.text, "- ") {
		return p.sequence(indent)
	}
	if _, _, ok := splitKey(l.text); ok {
		return p.mapping(indent)
	}
	p.pos++
	return scalar(l.text, l.no)
}

// splitKey cuts "key: value" at the first colon that ends a key.
func splitKey(s string) (key, rest string, ok bool) {
	if s == "" || s[0] == '[' || s[0] == '{' {
		return "", "", false
	}
	if s[0] == '"' || s[0] == '\'' {
		end := closingQuote(s)
		if end < 0 || end+1 >= len(s) || s[end+1] != ':' || (end+2 < len(s) && s[end+2] != ' ') {
			return "", "", false
		}
		k, err := scalar(s[:end+1], 0)
		ks, _ := k.(string)
		if err != nil {
			return "", "", false
		}
		return ks, strings.TrimSpace(s[end+2:]), true
	}
	for i := 0; i < len(s); i++ {
		if s[i] == ':' && (i+1 == len(s) || s[i+1] == ' ') {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), true
		}
	}
	return "", "", false
}

func closingQuote(s string) int {
	q := s[0]
	for i := 1; i < len(s); i++ {
		if q == '"' && s[i] == '\\' {
			i++
			continue
		}
		if s[i] == q {
			if q == '\'' && i+1 < len(s) && s[i+1] == '\'' {
				i++
				continue
			}
			return i
		}
	}
	return -1
}

func (p *parser) mapping(indent int) (any, error) {
	m := map[string]any{}
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if l.indent < indent {
			break
		}
		if l.indent > indent {
			return nil, &Error{l.no, "unexpected indentation"}
		}
		key, rest, ok := splitKey(l.text)
		if !ok {
			if l.text == "-" || strings.HasPrefix(l.text, "- ") {
				break
			}
			return nil, &Error{l.no, fmt.Sprintf("expected \"key: value\", got %q", l.text)}
		}
		if _, dup := m[key]; dup {
			return nil, &Error{l.no, fmt.Sprintf("the key %q appears twice", key)}
		}
		p.pos++
		v, err := p.value(rest, indent, l.no, true)
		if err != nil {
			return nil, err
		}
		m[key] = v
	}
	return m, nil
}

func (p *parser) sequence(indent int) (any, error) {
	list := []any{}
	for p.pos < len(p.lines) {
		l := p.lines[p.pos]
		if l.indent != indent || !(l.text == "-" || strings.HasPrefix(l.text, "- ")) {
			break
		}
		rest := strings.TrimSpace(strings.TrimPrefix(l.text, "-"))
		if rest == "" {
			p.pos++
			v, err := p.nested(indent, l.no)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
			continue
		}
		// "- - x" is a sequence inside the item
		if rest == "-" || strings.HasPrefix(rest, "- ") {
			offset := indent + (len(l.text) - len(rest))
			p.lines[p.pos] = line{no: l.no, indent: offset, text: rest}
			v, err := p.sequence(offset)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
			continue
		}
		// "- key: value" starts a mapping whose later keys sit under the key
		if _, _, ok := splitKey(rest); ok && !strings.HasPrefix(rest, "[") && !strings.HasPrefix(rest, "{") {
			offset := indent + (len(l.text) - len(rest))
			p.lines[p.pos] = line{no: l.no, indent: offset, text: rest}
			v, err := p.mapping(offset)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
			continue
		}
		p.pos++
		v, err := p.value(rest, indent, l.no, false)
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

// nested parses the block that follows a line with nothing after its
// indicator, or null when nothing is indented under it.
func (p *parser) nested(parentIndent, no int) (any, error) {
	if p.pos >= len(p.lines) || p.lines[p.pos].indent <= parentIndent {
		return nil, nil
	}
	return p.block(p.lines[p.pos].indent)
}

// value parses what follows "key:" or "- ": inline text, a block scalar
// or a nested block (a sequence may sit at the key's own indent).
func (p *parser) value(rest string, indent, no int, isKey bool) (any, error) {
	switch {
	case rest == "":
		if isKey && p.pos < len(p.lines) && p.lines[p.pos].indent == indent &&
			(p.lines[p.pos].text == "-" || strings.HasPrefix(p.lines[p.pos].text, "- ")) {
			return p.sequence(indent)
		}
		return p.nested(indent, no)
	case rest[0] == '|' || rest[0] == '>':
		return p.blockScalar(rest, indent, no)
	case strings.HasPrefix(rest, "&") || strings.HasPrefix(rest, "*") || strings.HasPrefix(rest, "!"):
		return nil, &Error{no, "anchors, aliases and tags are not supported"}
	}
	return scalar(rest, no)
}

// blockScalar reads a | or > scalar: the following lines indented deeper
// than the key, kept as lines (|) or folded into spaces (>).
func (p *parser) blockScalar(header string, indent, no int) (any, error) {
	strip := strings.HasSuffix(header, "-")
	var body []string
	blockIndent := -1
	last := no // the last input line the scalar uses
	for i := no; i < len(p.raw); i++ {
		raw := p.raw[i]
		if strings.TrimSpace(raw) == "" {
			body = append(body, "")
			continue
		}
		ind := len(raw) - len(strings.TrimLeft(raw, " "))
		if ind <= indent || (blockIndent >= 0 && ind < blockIndent) {
			break
		}
		if blockIndent < 0 {
			blockIndent = ind
		}
		body = append(body, raw[blockIndent:])
		last = i + 1
	}
	for len(body) > 0 && body[len(body)-1] == "" {
		body = body[:len(body)-1]
	}
	for p.pos < len(p.lines) && p.lines[p.pos].no <= last {
		p.pos++
	}
	text := strings.Join(body, "\n")
	if header[0] == '>' {
		text = foldLines(body)
	}
	if !strip && len(body) > 0 {
		text += "\n"
	}
	return text, nil
}

// foldLines joins the lines of a folded scalar: a blank line is a line
// break, other line breaks are spaces.
func foldLines(lines []string) string {
	var sb strings.Builder
	for i, l := range lines {
		switch {
		case l == "":
			sb.WriteByte('\n')
		case i > 0 && lines[i-1] != "":
			sb.WriteByte(' ')
			sb.WriteString(l)
		default:
			sb.WriteString(l)
		}
	}
	return sb.String()
}

// scalar reads a one-line value: quoted text, a flow collection, null,
// a boolean, a number or plain text.
func scalar(s string, no int) (any, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return nil, nil
	case s[0] == '"':
		end := closingQuote(s)
		if end != len(s)-1 {
			return nil, &Error{no, "text after a quoted string, or an unterminated quote"}
		}
		u, err := strconv.Unquote(s)
		if err != nil { // YAML escapes JSON does not have, such as \/ or \e: keep the text
			return s[1 : len(s)-1], nil
		}
		return u, nil
	case s[0] == '\'':
		end := closingQuote(s)
		if end != len(s)-1 {
			return nil, &Error{no, "text after a quoted string, or an unterminated quote"}
		}
		return strings.ReplaceAll(s[1:end], "''", "'"), nil
	case s[0] == '[' || s[0] == '{':
		fp := &flow{s: s, no: no}
		v, err := fp.value()
		if err != nil {
			return nil, err
		}
		fp.skip()
		if fp.i != len(fp.s) {
			return nil, &Error{no, "text after the flow collection"}
		}
		return v, nil
	}
	return plain(s), nil
}

// plain converts unquoted text by the YAML core schema.
func plain(s string) any {
	switch s {
	case "null", "Null", "NULL", "~":
		return nil
	case "true", "True", "TRUE":
		return true
	case "false", "False", "FALSE":
		return false
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && !(len(s) > 1 && s[0] == '0') {
		return float64(n)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !strings.ContainsAny(s, "_xXnN") && (s[0] != '0' || strings.HasPrefix(s, "0.")) {
		return f
	}
	return s
}

// flow reads [a, b] and {k: v} written on one line.
type flow struct {
	s  string
	i  int
	no int
}

func (f *flow) skip() {
	for f.i < len(f.s) && f.s[f.i] == ' ' {
		f.i++
	}
}

func (f *flow) value() (any, error) {
	f.skip()
	if f.i >= len(f.s) {
		return nil, &Error{f.no, "the flow collection is not closed (it must fit on one line)"}
	}
	switch f.s[f.i] {
	case '[':
		f.i++
		list := []any{}
		for {
			f.skip()
			if f.i < len(f.s) && f.s[f.i] == ']' {
				f.i++
				return list, nil
			}
			v, err := f.value()
			if err != nil {
				return nil, err
			}
			list = append(list, v)
			f.skip()
			if f.i < len(f.s) && f.s[f.i] == ',' {
				f.i++
			}
		}
	case '{':
		f.i++
		m := map[string]any{}
		for {
			f.skip()
			if f.i < len(f.s) && f.s[f.i] == '}' {
				f.i++
				return m, nil
			}
			k, err := f.item(":")
			if err != nil {
				return nil, err
			}
			f.skip()
			if f.i >= len(f.s) || f.s[f.i] != ':' {
				return nil, &Error{f.no, "expected : in the flow mapping"}
			}
			f.i++
			v, err := f.value()
			if err != nil {
				return nil, err
			}
			ks, _ := k.(string)
			m[ks] = v
			f.skip()
			if f.i < len(f.s) && f.s[f.i] == ',' {
				f.i++
			}
		}
	}
	return f.item(",]}")
}

// item reads one scalar up to any of stops.
func (f *flow) item(stops string) (any, error) {
	f.skip()
	if f.i < len(f.s) && (f.s[f.i] == '"' || f.s[f.i] == '\'') {
		end := closingQuote(f.s[f.i:])
		if end < 0 {
			return nil, &Error{f.no, "unterminated quote"}
		}
		v, err := scalar(f.s[f.i:f.i+end+1], f.no)
		f.i += end + 1
		return v, err
	}
	start := f.i
	for f.i < len(f.s) && !strings.ContainsRune(stops, rune(f.s[f.i])) {
		f.i++
	}
	return plain(strings.TrimSpace(f.s[start:f.i])), nil
}
