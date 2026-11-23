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

// env is what a build reads names from: the loop variables of the `for`
// elements around the element being built, innermost first, then the
// caller's own scope, then its fallback scopes in order. It is passed by
// value and never written to; a `for` makes a new env, it does not copy a
// scope.
type env struct {
	own       Scope
	fallbacks []Scope
	loop      *binding
}

// binding is the variable of one `for`: its item and index, written again
// for each item. Nothing keeps it past the build of that item (the built
// view holds values, not the env), so one binding serves the whole loop.
type binding struct {
	name  string
	item  any
	index int
	outer *binding
}

// indexName is the name a `for` binds its position to.
const indexName = "$index"

// first is the entry a path starts at: found is false when no scope has the
// name. An entry holding nil is found, so it shadows the scopes after it.
func (e env) first(name string) (value any, found bool) {
	for loop := e.loop; loop != nil; loop = loop.outer {
		if name == indexName {
			return loop.index, true
		}
		if name == loop.name {
			return loop.item, true
		}
	}
	if value, found = e.own[name]; found {
		return value, true
	}
	for _, fallback := range e.fallbacks {
		if value, found = fallback[name]; found {
			return value, true
		}
	}
	return nil, false
}

// interpolate replaces each {{path}} in s with the value it names.
func interpolate(text string, e env) (string, error) {
	return parseInterp(text).eval(e)
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

// eval replaces each path with the value it names in the scopes.
func (in interp) eval(e env) (string, error) {
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
		v, err := lookupPath(part.path, e)
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
func lookup(path string, e env) (any, error) {
	return lookupPath(splitPath(path), e)
}

// lookupPath finds a path, split at its dots, in the scope. Each step reads
// a key of a map or an exported field of a struct (through pointers), so a
// view model reaches the template as its own Go type (no methods: see the
// package doc).
func lookupPath(parts []string, e env) (any, error) {
	if len(parts) == 1 && parts[0] == "" {
		return nil, errors.New("an empty name")
	}
	cur, found := e.first(parts[0])
	if !found {
		return nil, fmt.Errorf("unknown name %q", strings.Join(parts, "."))
	}
	for i := 1; i < len(parts); i++ {
		next, found, err := member(cur, parts[i])
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
func truthy(expr string, e env) (bool, error) {
	negated, path := splitCondition(expr)
	return truthyPath(negated, path, e)
}

// splitCondition reads `name` or `!name` of an `if`.
func splitCondition(expr string) (negated bool, path []string) {
	expr = strings.TrimSpace(expr)
	return strings.HasPrefix(expr, "!"), splitPath(strings.TrimPrefix(expr, "!"))
}

// truthyPath is truthy for a condition read by splitCondition.
func truthyPath(negated bool, path []string, e env) (bool, error) {
	v, err := lookupPath(path, e)
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

// list is a slice or array of any element type, so a []Row from the app is a
// list as a []any is. A []any is read in place; any other kind is read
// through reflect, item by item, without a copy of the list.
type list struct {
	plain     []any
	reflected reflect.Value
	size      int
}

// asList is value as a list.
func asList(value any) (list, error) {
	if plain, isPlain := value.([]any); isPlain {
		return list{plain: plain, size: len(plain)}, nil
	}
	reflected := reflect.ValueOf(value)
	if kind := reflected.Kind(); kind != reflect.Slice && kind != reflect.Array {
		return list{}, errors.New("is not a list")
	}
	return list{reflected: reflected, size: reflected.Len()}, nil
}

// at is item i.
func (items list) at(i int) any {
	if items.plain != nil {
		return items.plain[i]
	}
	return items.reflected.Index(i).Interface()
}
