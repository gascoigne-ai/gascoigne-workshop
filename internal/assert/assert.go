// Package assert checks a response against a few lines of text, one
// assertion per line:
//
//	status == 200
//	status < 300
//	time < 500ms
//	header Content-Type contains json
//	header X-Request-Id exists
//	body contains "ok"
//	json total_count > 0
//	json items[0].name == "alpha"
//	json items.length >= 1
//	set token = json access_token
//	set id = header X-Request-Id
//	set code = status
//
// A set line stores a value of the response in a variable, for the next
// requests to use as {{token}}: Run reports it like an assertion that
// passes when the value exists, and Captures returns what to store.
//
// Lines that are empty or start with # are skipped. An assertion that
// cannot be parsed fails with the reason, so a typo is never a pass.
package assert

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// Result is the outcome of one assertion.
type Result struct {
	Line   string // the assertion as written
	OK     bool
	Detail string // what was found when it failed, or why it could not be checked
}

// Run checks every assertion of text against resp.
func Run(text string, resp *model.Response) []Result {
	var out []Result
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ok, detail := check(line, resp)
		out = append(out, Result{Line: line, OK: ok, Detail: detail})
	}
	return out
}

// Summary is "2/3 passed", or "" with no assertions.
func Summary(results []Result) string {
	if len(results) == 0 {
		return ""
	}
	n := 0
	for _, r := range results {
		if r.OK {
			n++
		}
	}
	return fmt.Sprintf("%d/%d passed", n, len(results))
}

// Capture is a value taken from a response to keep in a variable.
type Capture struct{ Name, Value string }

// Captures returns the values the set lines of text take from resp; lines
// whose value is missing are left out.
func Captures(text string, resp *model.Response) []Capture {
	var out []Capture
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if c, ok, _ := capture(line, resp); ok {
			out = append(out, c)
		}
	}
	return out
}

// capture evaluates a "set name = source" line: isSet says it is one,
// ok that the value was found.
func capture(line string, resp *model.Response) (c Capture, ok bool, detail string) {
	rest, isSet := strings.CutPrefix(line, "set ")
	if !isSet {
		return c, false, ""
	}
	name, source, found := strings.Cut(rest, "=")
	name, source = strings.TrimSpace(name), strings.TrimSpace(source)
	if !found || name == "" || source == "" {
		return c, false, "write: set name = json path, header Name or status"
	}
	if resp == nil || resp.Err != "" {
		return c, false, "there is no response to take it from"
	}
	kind, arg, _ := strings.Cut(source, " ")
	arg = strings.TrimSpace(arg)
	var value string
	switch kind {
	case "status":
		value = strconv.Itoa(resp.Status)
	case "header":
		for _, h := range resp.Headers {
			if strings.EqualFold(h.Name, arg) {
				value = h.Value
				found = true
				break
			}
			found = false
		}
		if !found {
			return c, false, "the header is not there"
		}
	case "json":
		var doc any
		if err := json.Unmarshal(resp.Body, &doc); err != nil {
			return c, false, "the body is not JSON"
		}
		v, has := lookup(doc, arg)
		if !has {
			return c, false, "no such path"
		}
		switch x := v.(type) {
		case string:
			value = x
		case float64:
			value = strconv.FormatFloat(x, 'f', -1, 64)
		case nil:
			value = "null"
		case bool:
			value = strconv.FormatBool(x)
		default:
			b, _ := json.Marshal(x)
			value = string(b)
		}
	default:
		return c, false, "take a value from status, header or json"
	}
	return Capture{Name: name, Value: value}, true, ""
}

func check(line string, resp *model.Response) (bool, string) {
	if strings.HasPrefix(line, "set ") {
		c, ok, detail := capture(line, resp)
		if ok {
			return true, c.Name + " = " + abbreviateValue(c.Value)
		}
		return false, detail
	}
	if resp == nil || resp.Err != "" {
		return false, "there is no response to check"
	}
	subject, rest, _ := strings.Cut(line, " ")
	rest = strings.TrimSpace(rest)
	switch subject {
	case "status":
		return compareNumber(float64(resp.Status), rest)
	case "time":
		return compareDuration(resp.Duration, rest)
	case "header":
		return checkHeader(resp, rest)
	case "body":
		return compareText(string(resp.Body), rest)
	case "json":
		return checkJSON(resp, rest)
	}
	return false, fmt.Sprintf("unknown subject %q: use status, time, header, body or json", subject)
}

var numberOps = []string{"==", "!=", "<=", ">=", "<", ">"}

// splitOp cuts "<op> <value>" off rest.
func splitOp(rest string, ops []string) (op, value string, ok bool) {
	for _, o := range ops {
		if r, found := strings.CutPrefix(rest, o); found {
			return o, strings.TrimSpace(r), true
		}
		if r, found := strings.CutPrefix(rest, o+" "); found {
			return o, strings.TrimSpace(r), true
		}
	}
	return "", "", false
}

func compareNumber(got float64, rest string) (bool, string) {
	op, val, ok := splitOp(rest, numberOps)
	if !ok {
		return false, "expected an operator: == != < <= > >="
	}
	want, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return false, fmt.Sprintf("%q is not a number", val)
	}
	if applyNumber(op, got, want) {
		return true, ""
	}
	return false, "got " + strconv.FormatFloat(got, 'f', -1, 64)
}

func applyNumber(op string, a, b float64) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	}
	return a >= b
}

func compareDuration(got time.Duration, rest string) (bool, string) {
	op, val, ok := splitOp(rest, numberOps[2:]) // no == on a time
	if !ok {
		return false, "expected an operator: < <= > >="
	}
	want, err := time.ParseDuration(val)
	if err != nil {
		return false, fmt.Sprintf("%q is not a duration like 500ms or 2s", val)
	}
	if applyNumber(op, float64(got), float64(want)) {
		return true, ""
	}
	return false, "got " + got.Round(time.Millisecond).String()
}

func compareText(got, rest string) (bool, string) {
	op, val, ok := splitOp(rest, []string{"contains", "==", "!="})
	if !ok {
		return false, "expected contains, == or !="
	}
	want := unquote(val)
	var pass bool
	switch op {
	case "contains":
		pass = strings.Contains(got, want)
	case "==":
		pass = got == want
	default:
		pass = got != want
	}
	if pass {
		return true, ""
	}
	return false, "got " + abbreviate(got)
}

func checkHeader(resp *model.Response, rest string) (bool, string) {
	name, rest, _ := strings.Cut(rest, " ")
	rest = strings.TrimSpace(rest)
	var values []string
	for _, h := range resp.Headers {
		if strings.EqualFold(h.Name, name) {
			values = append(values, h.Value)
		}
	}
	if rest == "exists" {
		if len(values) > 0 {
			return true, ""
		}
		return false, "the header is not there"
	}
	if len(values) == 0 {
		return false, "the header is not there"
	}
	ok, detail := compareText(strings.Join(values, ", "), rest)
	return ok, detail
}

func checkJSON(resp *model.Response, rest string) (bool, string) {
	path, rest, _ := strings.Cut(rest, " ")
	rest = strings.TrimSpace(rest)
	var doc any
	if err := json.Unmarshal(resp.Body, &doc); err != nil {
		return false, "the body is not JSON"
	}
	v, found := lookup(doc, path)
	if rest == "exists" {
		if found {
			return true, ""
		}
		return false, "no such path"
	}
	if !found {
		return false, "no such path"
	}
	op, val, ok := splitOp(rest, append([]string{"contains"}, numberOps...))
	if !ok {
		return false, "expected exists, contains or an operator: == != < <= > >="
	}
	switch got := v.(type) {
	case float64:
		if op == "contains" {
			return false, "contains needs a string or a list"
		}
		want, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return false, fmt.Sprintf("%q is not a number", val)
		}
		if applyNumber(op, got, want) {
			return true, ""
		}
		return false, "got " + strconv.FormatFloat(got, 'f', -1, 64)
	case string:
		return compareText(got, textOp(op, val))
	case bool, nil:
		want := unquote(val)
		have := fmt.Sprint(got)
		if got == nil {
			have = "null"
		}
		switch op {
		case "==":
			return have == want, "got " + have
		case "!=":
			return have != want, "got " + have
		}
		return false, "only == and != apply to " + have
	case []any:
		if op == "contains" {
			want := unquote(val)
			for _, item := range got {
				if fmt.Sprint(item) == want {
					return true, ""
				}
			}
			return false, "the list does not contain it"
		}
		return false, "compare a list through .length"
	}
	return false, "compare a value, not an object"
}

func textOp(op, val string) string { return op + " " + val }

// lookup walks a path like items[0].name or total_count; ".length" gives the
// size of a list or a string when the object has no such key.
func lookup(doc any, path string) (any, bool) {
	cur := doc
	for _, part := range strings.Split(path, ".") {
		name, idx := part, ""
		if i := strings.IndexByte(part, '['); i >= 0 && strings.HasSuffix(part, "]") {
			name, idx = part[:i], part[i+1:len(part)-1]
		}
		if name != "" {
			m, ok := cur.(map[string]any)
			if ok {
				next, has := m[name]
				if !has {
					return nil, false
				}
				cur = next
			} else if name == "length" {
				switch x := cur.(type) {
				case []any:
					cur = float64(len(x))
				case string:
					cur = float64(len([]rune(x)))
				default:
					return nil, false
				}
			} else {
				return nil, false
			}
		}
		if idx != "" {
			n, err := strconv.Atoi(idx)
			list, ok := cur.([]any)
			if err != nil || !ok || n < 0 || n >= len(list) {
				return nil, false
			}
			cur = list[n]
		}
	}
	return cur, true
}

// unquote removes one pair of double quotes around s.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
		return s[1 : len(s)-1]
	}
	return s
}

func abbreviate(s string) string {
	r := []rune(s)
	if len(r) > 40 {
		return strconv.Quote(string(r[:40])) + "…"
	}
	return strconv.Quote(s)
}

func abbreviateValue(s string) string {
	r := []rune(s)
	if len(r) > 30 {
		return string(r[:30]) + "…"
	}
	return s
}
