package markup

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Scope holds what a template can read: strings, numbers, bools, slices
// of any element type, nested maps, structs (or pointers to them) whose
// exported fields a path reads, func handlers, and the typed state and
// specs that containers take (*widget.ScrollState).
type Scope map[string]any

// interpolate replaces each {{path}} in s with the value it names.
func interpolate(text string, sc Scope) (string, error) {
	return parseInterp(text).eval(sc)
}

// interpPart is a literal of a text or one {{path}} of it, already split
// at the dots; a literal has a nil path.
type interpPart struct {
	literal string
	path    []string
}

// interp is a text read once, at Parse, so a build does not search for
// braces or split paths. err is a {{ without }}, reported when the text is
// evaluated, as before.
type interp struct {
	parts []interpPart
	err   error
}

// isLiteral is whether the text has no {{path}}.
func (in interp) isLiteral() bool {
	return in.err == nil && (len(in.parts) == 0 || (len(in.parts) == 1 && in.parts[0].path == nil))
}

// parseInterp splits text into literals and paths.
func parseInterp(text string) interp {
	var in interp
	for {
		i := strings.Index(text, "{{")
		if i < 0 {
			in.parts = append(in.parts, interpPart{literal: text})
			return in
		}
		j := strings.Index(text[i:], "}}")
		if j < 0 {
			return interp{err: errors.New("{{ without }}")}
		}
		in.parts = append(in.parts, interpPart{literal: text[:i]}, interpPart{path: splitPath(text[i+2 : i+j])})
		text = text[i+j+2:]
	}
}

// splitPath is the steps of the dotted path in text; an empty text is the
// single empty step, which lookupPath reports.
func splitPath(text string) []string { return strings.Split(strings.TrimSpace(text), ".") }

// eval replaces each path with the value it names in sc.
func (in interp) eval(sc Scope) (string, error) {
	if in.err != nil {
		return "", in.err
	}
	if len(in.parts) == 1 && in.parts[0].path == nil {
		return in.parts[0].literal, nil
	}
	var sb strings.Builder
	for _, part := range in.parts {
		if part.path == nil {
			sb.WriteString(part.literal)
			continue
		}
		v, err := lookupPath(part.path, sc)
		if err != nil {
			return "", err
		}
		if text, isText := v.(string); isText {
			sb.WriteString(text)
			continue
		}
		fmt.Fprint(&sb, v)
	}
	return sb.String(), nil
}

// lookup finds a dotted path in the scope.
func lookup(path string, sc Scope) (any, error) {
	return lookupPath(splitPath(path), sc)
}

// lookupPath finds a path, split at its dots, in the scope. Each step reads
// a key of a map or an exported field of a struct (through pointers), so a
// view model reaches the template as its own Go type (no methods: see the
// package doc).
func lookupPath(parts []string, sc Scope) (any, error) {
	if len(parts) == 1 && parts[0] == "" {
		return nil, errors.New("an empty name")
	}
	var cur any = map[string]any(sc)
	for i, part := range parts {
		next, found, err := member(cur, part)
		if err != nil {
			return nil, fmt.Errorf("%q: %s %v", strings.Join(parts, "."), strings.Join(parts[:i], "."), err)
		}
		if !found {
			return nil, fmt.Errorf("unknown name %q", strings.Join(parts, "."))
		}
		cur = next
	}
	return cur, nil
}

// member is the entry called name of a map or the exported field called
// name of a struct; found is false when it has none. Its error says why
// value cannot have members: it is a nil pointer or not an object.
func member(value any, name string) (member any, found bool, err error) {
	switch typed := value.(type) {
	case map[string]any:
		member, found = typed[name]
		return member, found, nil
	case Scope:
		member, found = typed[name]
		return member, found, nil
	}
	reflected := reflect.ValueOf(value)
	for reflected.Kind() == reflect.Pointer {
		if reflected.IsNil() {
			return nil, false, errors.New("is nil")
		}
		reflected = reflected.Elem()
	}
	if reflected.Kind() != reflect.Struct {
		return nil, false, errors.New("is not an object")
	}
	field, isField := reflected.Type().FieldByName(name)
	if !isField || !field.IsExported() {
		return nil, false, nil
	}
	fieldValue, err := reflected.FieldByIndexErr(field.Index)
	if err != nil { // the field is promoted through a nil embedded pointer
		return nil, false, fmt.Errorf("has a nil embedded struct before %s", name)
	}
	return fieldValue.Interface(), true, nil
}

// truthy evaluates `name` or `!name`.
func truthy(expr string, sc Scope) (bool, error) {
	negated, path := splitCondition(expr)
	return truthyPath(negated, path, sc)
}

// splitCondition reads `name` or `!name` of an `if`.
func splitCondition(expr string) (negated bool, path []string) {
	expr = strings.TrimSpace(expr)
	return strings.HasPrefix(expr, "!"), splitPath(strings.TrimPrefix(expr, "!"))
}

// truthyPath is truthy for a condition read by splitCondition.
func truthyPath(negated bool, path []string, sc Scope) (bool, error) {
	v, err := lookupPath(path, sc)
	if err != nil {
		return false, err
	}
	return isSet(v) != negated, nil
}

// isSet is whether `if` shows its element for value: by its kind, so a
// struct field of a named type (a nil *Request, an empty Mode string)
// reads as its underlying kind does.
func isSet(value any) bool {
	if value == nil {
		return false
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Bool:
		return reflected.Bool()
	case reflect.String, reflect.Slice, reflect.Array:
		return reflected.Len() > 0
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Chan, reflect.Map:
		return !reflected.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return !reflected.IsZero()
	}
	return true
}

// asList is the items of a slice or array of any element type, so a
// []Row from the app is a list as a []any is.
func asList(value any) ([]any, error) {
	reflected := reflect.ValueOf(value)
	if kind := reflected.Kind(); kind != reflect.Slice && kind != reflect.Array {
		return nil, errors.New("is not a list")
	}
	items := make([]any, reflected.Len())
	for i := range items {
		items[i] = reflected.Index(i).Interface()
	}
	return items, nil
}
