package jsonview

import (
	"reflect"
	"strings"
	"testing"
)

func kinds(l Line) []Kind {
	var out []Kind
	for _, s := range l.Spans {
		out = append(out, s.Kind)
	}
	return out
}

func text(l Line) string {
	var b strings.Builder
	for _, s := range l.Spans {
		b.WriteString(s.Text)
	}
	return b.String()
}

func TestLinesPrettyPrintsAndColors(t *testing.T) {
	lines := Lines([]byte(`{"total_count":36351,"incomplete_results":false,"owner":{"login":"a\"b","x":null}}`))
	want := []struct {
		text     string
		kinds    []Kind
		foldable bool
	}{
		{`{`, []Kind{Punct}, true},
		{`  "total_count": 36351,`, []Kind{Plain, Key, Punct, Plain, Number, Punct}, false},
		{`  "incomplete_results": false,`, []Kind{Plain, Key, Punct, Plain, Literal, Punct}, false},
		{`  "owner": {`, []Kind{Plain, Key, Punct, Plain, Punct}, true},
		{`    "login": "a\"b",`, []Kind{Plain, Key, Punct, Plain, String, Punct}, false},
		{`    "x": null`, []Kind{Plain, Key, Punct, Plain, Literal}, false},
		{`  }`, []Kind{Plain, Punct}, false},
		{`}`, []Kind{Punct}, false},
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	for i, w := range want {
		if got := text(lines[i]); got != w.text {
			t.Errorf("line %d text = %q, want %q", i, got, w.text)
		}
		if got := kinds(lines[i]); !reflect.DeepEqual(got, w.kinds) {
			t.Errorf("line %d kinds = %v, want %v", i, got, w.kinds)
		}
		if lines[i].Foldable != w.foldable {
			t.Errorf("line %d foldable = %v, want %v", i, lines[i].Foldable, w.foldable)
		}
	}
}

func TestLinesNonJSONIsPlain(t *testing.T) {
	tests := []struct {
		name, in string
		want     []string
	}{
		{"text", "hello\r\nworld", []string{"hello", "world"}},
		{"broken json", `{"a":`, []string{`{"a":`}},
		{"empty", "", []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := Lines([]byte(tt.in))
			var got []string
			for _, l := range lines {
				got = append(got, text(l))
				if l.Foldable || kinds(l)[0] != Plain {
					t.Errorf("line %q not plain", text(l))
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLinesKeepsStringsWithPunctuation(t *testing.T) {
	lines := Lines([]byte(`{"u":"https://x/{a}, [b]: c"}`))
	if got := text(lines[1]); got != `  "u": "https://x/{a}, [b]: c"` {
		t.Fatalf("got %q", got)
	}
	if got := kinds(lines[1]); !reflect.DeepEqual(got, []Kind{Plain, Key, Punct, Plain, String}) {
		t.Fatalf("kinds = %v", got)
	}
}
