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
	var sb strings.Builder
	for {
		i := strings.Index(text, "{{")
		if i < 0 {
			sb.WriteString(text)
			return sb.String(), nil
		}
		j := strings.Index(text[i:], "}}")
		if j < 0 {
			return "", errors.New("{{ without }}")
		}
		sb.WriteString(text[:i])
		v, err := lookup(strings.TrimSpace(text[i+2:i+j]), sc)
		if err != nil {
			return "", err
		}
		fmt.Fprint(&sb, v)
		text = text[i+j+2:]
	}
}

// lookup finds a dotted path in the scope. Each step reads a key of a map
// or an exported field of a struct (through pointers), so a view model
// reaches the template as its own Go type (decision 4).
func lookup(path string, sc Scope) (any, error) {
	if path == "" {
		return nil, errors.New("an empty name")
	}
	var cur any = map[string]any(sc)
	parts := strings.Split(path, ".")
	for i, part := range parts {
		next, err := member(cur, part)
		if err != nil {
			return nil, fmt.Errorf("%q: %s %v", path, strings.Join(parts[:i], "."), err)
		}
		if next == nil {
			return nil, fmt.Errorf("unknown name %q", path)
		}
		cur = *next
	}
	return cur, nil
}

// member is the entry called name of a map or the exported field called
// name of a struct, nil when it has none. Its error says why value cannot
// have members: it is a nil pointer or not an object.
func member(value any, name string) (*any, error) {
	switch typed := value.(type) {
	case map[string]any:
		return mapEntry(typed, name), nil
	case Scope:
		return mapEntry(typed, name), nil
	}
	reflected := reflect.ValueOf(value)
	for reflected.Kind() == reflect.Pointer {
		if reflected.IsNil() {
			return nil, errors.New("is nil")
		}
		reflected = reflected.Elem()
	}
	if reflected.Kind() != reflect.Struct {
		return nil, errors.New("is not an object")
	}
	field, found := reflected.Type().FieldByName(name)
	if !found || !field.IsExported() {
		return nil, nil
	}
	fieldValue, err := reflected.FieldByIndexErr(field.Index)
	if err != nil { // the field is promoted through a nil embedded pointer
		return nil, fmt.Errorf("has a nil embedded struct before %s", name)
	}
	read := fieldValue.Interface()
	return &read, nil
}

// mapEntry is the entry called name, nil when there is none.
func mapEntry(entries map[string]any, name string) *any {
	entry, found := entries[name]
	if !found {
		return nil
	}
	return &entry
}

// truthy evaluates `name` or `!name`.
func truthy(expr string, sc Scope) (bool, error) {
	expr = strings.TrimSpace(expr)
	neg := strings.HasPrefix(expr, "!")
	v, err := lookup(strings.TrimSpace(strings.TrimPrefix(expr, "!")), sc)
	if err != nil {
		return false, err
	}
	return isSet(v) != neg, nil
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
