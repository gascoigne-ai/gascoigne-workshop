package markup

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// A param that cannot be read reports the attribute as it is written, with
// the path in full.
func TestParamErrorTexts(t *testing.T) {
	var nilState *widget.ScrollState
	tests := []struct {
		name string
		src  string
		own  Scope
		want string
	}{
		{"scoped, unknown", `<scroll state=" s.x "><text>a</text></scroll>`, Scope{}, `t.vml:1: unknown name "s.x"`},
		{"scoped, wrong type", `<scroll state=" s "><text>a</text></scroll>`, Scope{"s": 3}, `t.vml:1: state=" s " is not a *widget.ScrollState`},
		{"scoped state, nil", `<scroll state=" s "><text>a</text></scroll>`, Scope{"s": nilState}, `t.vml:1: state="s" is a nil *widget.ScrollState`},
		{"token, empty", `<note tone=" ">a</note>`, Scope{}, `t.vml:1: tone is empty, not a color`},
		{"token, unknown path", `<note tone="{{c}}">a</note>`, Scope{}, `t.vml:1: unknown name "c"`},
		{"whole number, bad", `<vrule-inset inset="x"/>`, Scope{}, `t.vml:1: inset: "x" is not a whole number`},
		{"whole number, bad from a path", `<vrule-inset inset="{{n}}"/>`, Scope{"n": "x"}, `t.vml:1: inset: "x" is not a whole number`},
		{"interpolated, unclosed", `<vrule-inset inset="{{n"/>`, Scope{}, `t.vml:1: {{ without }}`},
		{"interpolated, unknown", `<icon name="a{{n.m}}b"/>`, Scope{"n": map[string]any{}}, `t.vml:1: unknown name "n.m"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildWith(t, test.src, test.own)
			if err == nil || err.Error() != test.want {
				t.Errorf("error = %v, want %q", err, test.want)
			}
		})
	}
}
