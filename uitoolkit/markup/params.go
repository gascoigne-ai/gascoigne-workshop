package markup

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// param is a constructor argument an element reads from one of its own
// attributes. Every param is required, as the Go argument is.
type param struct {
	name string
	read func(value string, scope Scope) (any, error)
}

// scoped is a param whose value names a scope entry of the Go type T, the
// way handlers bind functions: state and specs that a widget writes back
// to, which no string can stand for.
func scoped[T any](name string) param {
	var zero T
	want := fmt.Sprintf("%T", zero)
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		found, err := lookup(strings.TrimSpace(value), scope)
		if err != nil {
			return nil, err
		}
		typed, ok := found.(T)
		if !ok {
			return nil, fmt.Errorf("%s=%q is not a %s", name, value, want)
		}
		return typed, nil
	}}
}

// scopedState is scoped for a *T the widget reads and writes. A nil *T has
// the right type but would panic at layout, far from the template, so it
// is an error here that names the entry.
func scopedState[T any](name string) param {
	pointer := scoped[*T](name)
	want := fmt.Sprintf("%T", (*T)(nil))
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		found, err := pointer.read(value, scope)
		if err != nil {
			return nil, err
		}
		if found.(*T) == nil {
			return nil, fmt.Errorf("%s=%q is a nil %s", name, strings.TrimSpace(value), want)
		}
		return found, nil
	}}
}

// finiteFloat reads a number for a size or weight. ParseFloat also takes
// "NaN" and "Inf", which layout cannot turn into pixels.
func finiteFloat(text string) (float64, error) {
	number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("%q is not a number", text)
	}
	return number, nil
}

// realParam is a param that is a number, written in the template or read
// with {{path}}.
func realParam(name string) param {
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		text, err := interpolate(value, scope)
		if err != nil {
			return nil, err
		}
		number, err := finiteFloat(text)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return number, nil
	}}
}

// wholeParam is a param that is a whole number, written in the template or
// read with {{path}}.
func wholeParam(name string) param {
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		text, err := interpolate(value, scope)
		if err != nil {
			return nil, err
		}
		number, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			return nil, fmt.Errorf("%s: %q is not a whole number", name, text)
		}
		return number, nil
	}}
}

// boolParam is a param that is true or false, written in the template or
// read with {{path}}.
func boolParam(name string) param {
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		text, err := interpolate(value, scope)
		if err != nil {
			return nil, err
		}
		switch strings.TrimSpace(text) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
		return nil, fmt.Errorf("%s: %q is not true or false", name, text)
	}}
}

// tokenParam is a param that is a theme color, written in the template or
// read with {{path}}. Unlike a color attribute it is required, as the Go
// argument is, so an empty one is an error.
func tokenParam(name string) param {
	return param{name: name, read: func(value string, scope Scope) (any, error) {
		text, err := interpolate(value, scope)
		if err != nil {
			return nil, err
		}
		token := strings.TrimSpace(text)
		if token == "" {
			return nil, fmt.Errorf("%s is empty, not a color", name)
		}
		return core.Token(token), nil
	}}
}

// textParam is a param that is text, open to {{path}}.
func textParam(name string) param {
	return param{name: name, read: func(value string, scope Scope) (any, error) { return interpolate(value, scope) }}
}
