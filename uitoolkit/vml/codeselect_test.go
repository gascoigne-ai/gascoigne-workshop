package vml

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// shiftKey sends one Shift+movement key to a view whose pointer is away.
func shiftKey(v *CodeView, c *core.Ctx, r core.Rect, edit core.Edit) {
	c.In = core.Input{X: 5000, Y: 5000, Shift: true, Edits: []core.Edit{edit}}
	v.Update(c, r)
}

// clickAt puts the caret before column col of line and releases.
func clickAt(v *CodeView, c *core.Ctx, r core.Rect, line, col int) {
	press(v, c, r, colX(c, v, r, col), rowY(c, v, r, line), 1)
	release(v, c, r, 0, 0)
}

func TestCodeViewShiftArrowsMoveTheEndOfTheSelection(t *testing.T) {
	tests := []struct {
		name      string
		line, col int
		keys      []core.Edit
		want      string
		wantAfter string // after the extra keys, "" when unused
		extraKeys []core.Edit
	}{
		{name: "right takes characters", line: 1, col: 2, keys: []core.Edit{core.EditRight, core.EditRight, core.EditRight}, want: `"a"`},
		{name: "left goes backwards", line: 1, col: 5, keys: []core.Edit{core.EditLeft, core.EditLeft, core.EditLeft}, want: `"a"`},
		{name: "right at a line end crosses to the next line", line: 0, col: 0, keys: []core.Edit{core.EditRight, core.EditRight}, want: "{\n"},
		{name: "left at a line start crosses to the previous line", line: 1, col: 0, keys: []core.Edit{core.EditLeft}, want: "\n"},
		{name: "down keeps the column", line: 1, col: 2, keys: []core.Edit{core.EditDown}, want: "\"a\": 1,\n  "},
		{name: "up keeps the column", line: 2, col: 4, keys: []core.Edit{core.EditUp}, want: "\": 1,\n  \"i"},
		{name: "end takes the rest of the line", line: 1, col: 2, keys: []core.Edit{core.EditEnd}, want: `"a": 1,`},
		{name: "home takes the start of the line", line: 1, col: 5, keys: []core.Edit{core.EditHome}, want: `  "a"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v, c, _, r := viewport(t)
			clickAt(v, c, r, test.line, test.col)
			for _, edit := range test.keys {
				shiftKey(v, c, r, edit)
			}
			if got := v.SelectedText(); got != test.want {
				t.Fatalf("selected %q, want %q", got, test.want)
			}
		})
	}
}

func TestCodeViewShiftArrowWithoutACaretStartsAtTheFirstLine(t *testing.T) {
	v, c, _, r := viewport(t)
	v.Focused = true
	shiftKey(v, c, r, core.EditRight)
	if got := v.SelectedText(); got != "{" {
		t.Fatalf("selected %q, want %q", got, "{")
	}
}

func TestCodeViewShiftDownSkipsFoldedLines(t *testing.T) {
	v, c, _, r := viewport(t)
	v.ToggleFold(2) // hides lines 3..12
	clickAt(v, c, r, 2, 0)
	shiftKey(v, c, r, core.EditDown)
	if v.selB.line != 13 {
		t.Fatalf("caret on line %d, want 13", v.selB.line)
	}
}

func TestCodeViewShiftArrowsStopAtTheEnds(t *testing.T) {
	v, c, _, r := viewport(t)
	clickAt(v, c, r, 0, 0)
	shiftKey(v, c, r, core.EditLeft)
	shiftKey(v, c, r, core.EditUp)
	if v.SelectedText() != "" {
		t.Fatalf("moved before the start: %q", v.SelectedText())
	}
}

func TestCodeViewPlainArrowsAndUnfocusedViewsDoNotSelect(t *testing.T) {
	v, c, _, r := viewport(t)
	clickAt(v, c, r, 1, 2)
	c.In = core.Input{X: 5000, Y: 5000, Edits: []core.Edit{core.EditRight}}
	v.Update(c, r)
	if v.SelectedText() != "" {
		t.Fatalf("a plain arrow selected %q", v.SelectedText())
	}
	v.Focused = false
	shiftKey(v, c, r, core.EditRight)
	if v.SelectedText() != "" {
		t.Fatalf("an unfocused view selected %q", v.SelectedText())
	}
}
