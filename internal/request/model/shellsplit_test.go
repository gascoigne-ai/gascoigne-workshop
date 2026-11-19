package model

import (
	"reflect"
	"strings"
	"testing"
)

// Pins ShellSplit for each quote kind before it is split per kind.
func TestShellSplitHandlesEveryQuoteKind(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"plain words", "a  b\tc\nd", []string{"a", "b", "c", "d"}},
		{"single quotes keep everything", `a 'b "c" \n'`, []string{"a", `b "c" \n`}},
		{"double quotes unescape only specials", `"a \" \\ \$ \` + "`" + ` \n"`, []string{"a \" \\ $ ` \\n"}},
		{"ansi c quotes", `$'a\nb\tc\rd\'e\\'`, []string{"a\nb\tc\rd'e\\"}},
		{"backslash escapes", `a\ b \"c`, []string{"a b", `"c`}},
		{"line continuation", "a \\\nb \\\r\nc", []string{"a", "b", "c"}},
		{"adjacent quotes join", `a'b'"c"d`, []string{"abcd"}},
		{"empty quotes make an empty word", `'' ""`, []string{"", ""}},
		{"trailing backslash is dropped", `a\`, []string{"a"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ShellSplit(tc.in)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ShellSplit(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
			}
		})
	}
	for _, in := range []string{`'a`, `"a`, `$'a`, `"a\"`} {
		if _, err := ShellSplit(in); err == nil || !strings.Contains(err.Error(), "unterminated") {
			t.Errorf("ShellSplit(%q) err = %v, want unterminated", in, err)
		}
	}
}
