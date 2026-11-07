package settingsui

import (
	"image/color"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
		{"an opening quote of a key", `{ `, `{ "`, 3, true, "appearance"},
		{"an opening quote of an enum value", `{"appearance": {"layout": `, `{"appearance": {"layout": "`, 0, true, "side"},
		{"a letter is not a trigger", `{ "`, `{ "l`, 4, false, ""},
		{"a quote closing a string is not a trigger", `{"appearance": {"layout": "side`, `{"appearance": {"layout": "side"`, 0, false, ""},
		{"a pasted run of text is not a trigger", `{ `, `{ "layout"`, 0, false, ""},
		{"nothing to offer for free text", `{"requests": {"proxy": `, `{"requests": {"proxy": "`, 0, false, ""},
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
			d.RawText.Cursor = len([]rune(test.after))
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

func TestTheCompletionMenuOpensAtTheCaret(t *testing.T) {
	var d Dialog
	host := newHost(t)
	d.OpenRaw(host)
	d.RawText.Focused = true
	d.RawText.Text = "{\n  \"appearance\": {\n    \"la"
	d.RawText.Cursor = len([]rune(d.RawText.Text))
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.In = core.Input{X: 5000, Y: 5000}
	d.RawText.Update(ctx, core.Rect{X: 300, Y: 100, W: 500, H: 300})
	x, y, ok := d.RawText.CaretAt()
	if !ok {
		t.Fatal("the editor does not know where its caret is")
	}
	d.CompleteRaw(host)
	if len(host.menus) != 1 || host.menuAt != [2]int{x, y} {
		t.Fatalf("menu at %v, want the caret at %d,%d (menus %d)", host.menuAt, x, y, len(host.menus))
	}
}

func testTheme() core.Theme {
	return core.Theme{
		Colors:   map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff), core.Bg: core.RGB(0x101010)},
		FontSize: 14, SmallSize: 12, MonoSize: 13,
	}
}
