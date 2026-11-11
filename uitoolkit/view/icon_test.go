package view

import (
	"github.com/hajimehoshi/ebiten/v2"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestIconsHaveTheirNaturalSizeAndScaleToFit(t *testing.T) {
	c := testCtx(t)
	nat := Icon("trash")
	Layout(c, Row(AlignItems(AlignCenter), nat), 100, 100)
	if nat.rect.W != 14 || nat.rect.H != 16 {
		t.Fatalf("natural size %dx%d", nat.rect.W, nat.rect.H)
	}
	if TriangleIcon(true).iconName != "triangle-down" || TriangleIcon(false).iconName != "triangle-right" {
		t.Fatal("the disclosure triangle picks its icon by state")
	}
}

func TestUnknownIconPanicsAndListsTheNames(t *testing.T) {
	defer func() {
		msg, _ := recover().(string)
		if !strings.Contains(msg, `no icon "nope"`) || !strings.Contains(msg, "trash") {
			t.Fatalf("panic %q", msg)
		}
	}()
	Icon("nope")
}

func TestThemeIconStrokeScalesTheLines(t *testing.T) {
	c := testCtx(t)
	c.Th.IconStroke = 1.6
	drawn(t, c, Row(W(30), H(30), Icon("trash")), 30, 30) // draws with the thicker stroke without a window
}

func TestImageMeasuresItsPictureAndDrawsScaledToFit(t *testing.T) {
	c := testCtx(t)
	pic := ebiten.NewImage(80, 40)
	im := Image(pic)
	Layout(c, Row(AlignItems(AlignCenter), im), 200, 100)
	if im.rect.W != 80 || im.rect.H != 40 {
		t.Fatalf("natural size %dx%d", im.rect.W, im.rect.H)
	}
	for _, scale := range []float64{1, 1.5, 2} {
		if err := c.SetScale(scale); err != nil {
			t.Fatal(err)
		}
		drawn(t, c, Row(W(30), H(30), Image(pic, Grow(1))), 30, 30)     // smaller box: scaled down
		drawn(t, c, Row(W(300), H(300), Image(pic, Grow(1))), 300, 300) // larger box: not scaled up
	}
	drawn(t, c, Row(W(30), H(30), Image(nil)), 30, 30) // no picture, no crash
}

func TestTableCellStylerSplitsACellIntoColoredParts(t *testing.T) {
	spec := TableSpec{
		Columns: []string{"Name", "Value"}, Weights: []float64{1, 1},
		Rows: []TableRow{{Cells: []string{"k", "a{{v}}b"}}},
		CellStyler: func(text string) []Span {
			if text != "a{{v}}b" {
				return nil
			}
			return []Span{{Text: "a"}, {Text: "{{v}}", Style: core.OK}, {Text: "b"}}
		},
	}
	d := Dump(Table(spec))
	for _, want := range []string{`text="a"`, `text="{{v}}"`, `fg=ok`, `text="b"`} {
		if !strings.Contains(d, want) {
			t.Errorf("missing %s in\n%s", want, d)
		}
	}
	if !strings.Contains(d, `text="k"`) {
		t.Errorf("a cell the styler declines stays plain:\n%s", d)
	}
}

// Every built-in icon draws without a window at the scales the UI uses,
// hovered or not, at its natural size and scaled to fit.
func TestEveryBuiltInIconDrawsAsAnElement(t *testing.T) {
	for _, name := range core.IconNames() {
		for _, scale := range []float64{1, 1.5, 2} {
			c := testCtx(t)
			if err := c.SetScale(scale); err != nil {
				t.Fatal(err)
			}
			c.In = core.Input{X: 5, Y: 5} // hovered
			drawn(t, c, Row(W(30), H(30), Icon(name)), 30, 30)
			c.In = core.Input{X: 500, Y: 500}
			drawn(t, c, Row(W(30), H(30), Icon(name, W(20), H(20))), 30, 30)
		}
	}
}
