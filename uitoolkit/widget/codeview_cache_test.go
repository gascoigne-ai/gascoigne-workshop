package widget

import (
	"github.com/hajimehoshi/ebiten/v2"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestCodeViewLayoutIsBuiltOnceWhileNothingChanges(t *testing.T) {
	v, c, _, r := viewport(t)
	v.SetLines(sampleLines())
	g := v.geometry(c, r)
	v.layout(g, r)
	before := v.layoutBuilds
	for range 3 {
		v.layout(g, r)
	}
	if v.layoutBuilds != before {
		t.Fatalf("layout rebuilt %d times for the same content, width and font", v.layoutBuilds-before)
	}
}

// A cached layout must never outlive what it was built from: each change
// below alters the rows or the texts, so the next layout is a new build.
func TestCodeViewLayoutIsRebuiltWhenItsInputsChange(t *testing.T) {
	tests := []struct {
		name   string
		wrap   bool // the view wraps before the change; only then does its width shape the rows
		change func(v *CodeView, c *core.Ctx, r *core.Rect)
	}{
		{"new lines", false, func(v *CodeView, c *core.Ctx, r *core.Rect) { v.SetLines(sampleLines()) }},
		{"a fold toggled", false, func(v *CodeView, c *core.Ctx, r *core.Rect) { v.ToggleFold(2) }},
		{"a fold written to the map", false, func(v *CodeView, c *core.Ctx, r *core.Rect) { v.Folded[2] = true }},
		{"lines assigned directly", false, func(v *CodeView, c *core.Ctx, r *core.Rect) { v.Lines = sampleLines() }},
		{"wrap switched on", false, func(v *CodeView, c *core.Ctx, r *core.Rect) { v.Wrap = true }},
		{"the view narrowed", true, func(v *CodeView, c *core.Ctx, r *core.Rect) { r.W -= 100 }},
		{"the font scale changed", false, func(v *CodeView, c *core.Ctx, r *core.Rect) {
			if err := c.SetScale(2); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, c, _, r := viewport(t)
			v.SetLines(sampleLines())
			v.Wrap = tt.wrap
			v.layout(v.geometry(c, r), r)
			before := v.layoutBuilds
			tt.change(v, c, &r)
			v.layout(v.geometry(c, r), r)
			if v.layoutBuilds == before {
				t.Fatal("the layout was served from the cache after its inputs changed")
			}
		})
	}
}

func TestCodeViewNumberDigitsFollowsWidestNumber(t *testing.T) {
	tests := []struct {
		name  string
		lines func() []Line
		want  int
	}{
		{"by position", func() []Line { return make([]Line, 1204) }, 4},
		{"by Num", func() []Line { return []Line{{Num: 7}, {Num: 99999}, {}} }, 5},
		{"empty", func() []Line { return nil }, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &CodeView{}
			v.SetLines(tt.lines())
			if got := v.numberDigits(); got != tt.want {
				t.Fatalf("digits = %d, want %d", got, tt.want)
			}
		})
	}
	t.Run("lines assigned without SetLines", func(t *testing.T) {
		v := &CodeView{}
		v.SetLines(make([]Line, 5))
		_ = v.numberDigits()
		v.Lines = make([]Line, 12000)
		if got := v.numberDigits(); got != 5 {
			t.Fatalf("digits = %d, want 5", got)
		}
	})
}

// Draw built the geometry four times, and each build scanned every line
// for the widest number.
func TestCodeViewScansLineNumbersOncePerContent(t *testing.T) {
	v, c, _, r := viewport(t)
	screen := ebiten.NewImage(r.W, r.H)
	for range 3 {
		v.Update(c, r)
		v.Draw(c, screen, r)
	}
	if v.digitScans != 1 {
		t.Fatalf("scans = %d after three frames, want 1", v.digitScans)
	}
	v.SetLines(sampleLines())
	for range 3 {
		v.Update(c, r)
		v.Draw(c, screen, r)
	}
	if v.digitScans != 2 {
		t.Fatalf("scans = %d after SetLines, want 2", v.digitScans)
	}
}
