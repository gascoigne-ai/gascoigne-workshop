package settingsui

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/respview"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestTheRawViewColorsKeysStringsNumbersAndLiterals(t *testing.T) {
	var d Dialog
	host := newHost(t)
	d.OpenRaw(host)
	line := `  "layout": "side", "uiScale": 150, "flat": true`
	spans := d.RawText.Styler(line)
	var joined strings.Builder
	styles := map[w.Token]bool{}
	for _, span := range spans {
		joined.WriteString(span.Text)
		styles[span.Style] = true
	}
	if joined.String() != line {
		t.Fatalf("the spans do not rebuild the line: %q", joined.String())
	}
	if len(styles) < 4 {
		t.Fatalf("only %d colors on a line with keys, strings, numbers and a literal: %v", len(styles), spans)
	}
	var keyStyle, stringStyle w.Token
	for _, span := range spans {
		switch span.Text {
		case `"layout"`:
			keyStyle = span.Style
		case `"side"`:
			stringStyle = span.Style
		}
	}
	if keyStyle != respview.Tokens[jsonview.Key] || keyStyle == stringStyle {
		t.Fatalf("key style %q, string style %q", keyStyle, stringStyle)
	}
}

func TestTypingAQuoteOpensTheCompletionMenuWithoutCtrlSpace(t *testing.T) {
	tests := []struct {
		name, before, after string
		cursor              int
		wantMenu            bool
		wantFirst           string
	}{
		{"an opening quote of a key", `{ `, `{ "`, 3, true, "caCert"},
		{"an opening quote of an enum value", `{"layout": `, `{"layout": "`, 12, true, "side"},
		{"a letter is not a trigger", `{ "`, `{ "l`, 4, false, ""},
		{"a quote closing a string is not a trigger", `{"layout": "side`, `{"layout": "side"`, 16, false, ""},
		{"a pasted run of text is not a trigger", `{ `, `{ "layout"`, 11, false, ""},
		{"nothing to offer for free text", `{"proxy": `, `{"proxy": "`, 11, false, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var d Dialog
			host := newHost(t)
			d.OpenRaw(host)
			d.RawText.Focused = true
			d.RawText.Text = test.before
			d.RawText.Cursor = len([]rune(test.before))
			d.RawChanged(host) // primes what was there before
			host.menus = nil
			host.toasts = nil
			d.RawText.Text = test.after
			d.RawText.Cursor = test.cursor
			d.RawChanged(host)
			switch {
			case test.wantMenu && (len(host.menus) != 1 || host.menus[0][0].Label != test.wantFirst):
				t.Fatalf("menus = %+v, want one starting with %q", host.menus, test.wantFirst)
			case !test.wantMenu && len(host.menus) != 0:
				t.Fatalf("a menu opened: %+v", host.menus)
			}
			if len(host.toasts) != 0 {
				t.Fatalf("an automatic completion toasted: %v", host.toasts)
			}
		})
	}
}
