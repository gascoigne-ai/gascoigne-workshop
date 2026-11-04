package vml

import (
	"github.com/hajimehoshi/ebiten/v2"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Every built-in icon parses, draws something inside its own viewBox, and
// draws without a window at the scales the UI uses, hovered or not.
func TestBuiltInIconsParseAndStayInsideTheirViewBox(t *testing.T) {
	names := IconNames()
	if len(names) < 11 {
		t.Fatalf("only %d icons: %v", len(names), names)
	}
	for _, name := range names {
		ic, _ := lookupIcon(name)
		if ic.w <= 0 || ic.h <= 0 || len(ic.shapes) == 0 {
			t.Errorf("%s: empty icon", name)
		}
		for _, sh := range ic.shapes {
			for _, cmd := range sh.cmds {
				n := map[byte]int{'M': 1, 'L': 1, 'C': 3, 'Z': 0}[cmd.op]
				for i := 0; i < n; i++ {
					x, y := cmd.p[2*i], cmd.p[2*i+1]
					if x < -0.01 || y < -0.01 || x > ic.w+0.01 || y > ic.h+0.01 {
						t.Errorf("%s: point %.2f,%.2f outside the %gx%g viewBox", name, x, y, ic.w, ic.h)
					}
				}
			}
		}
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

func TestRegisterIconsAddsAndReplacesAndReportsBrokenFiles(t *testing.T) {
	err := RegisterIcons(fstest.MapFS{
		"star.svg":   {Data: []byte(`<svg viewBox="0 0 10 10"><polygon points="5,0 6,4 10,4 7,6 8,10 5,7.5 2,10 3,6 0,4 4,4"/></svg>`)},
		"broken.svg": {Data: []byte("<svg viewBox=\"0 0 4 4\">\n<blink/></svg>")},
		"notes.txt":  {Data: []byte("ignored")},
	})
	if err == nil || !strings.Contains(err.Error(), "broken.svg:2") {
		t.Fatalf("err = %v", err)
	}
	if !HasIcon("star") || HasIcon("broken") {
		t.Fatal("registration")
	}
	t.Cleanup(func() { icons.Lock(); delete(icons.byName, "star"); icons.Unlock() })
}

func TestSVGSubsetErrorsNameFileAndLine(t *testing.T) {
	for name, tt := range map[string]struct{ src, want string }{
		"no root":       {`<path d="M0 0"/>`, "starts with <svg>"},
		"no viewBox":    {`<svg><path d="M0 0L1 1"/></svg>`, "viewBox"},
		"bad paint":     {`<svg viewBox="0 0 4 4"><path d="M0 0L1 1" stroke="red"/></svg>`, "none, currentColor or token"},
		"unknown tag":   {"<svg viewBox=\"0 0 4 4\">\n\n<text/></svg>", "svg:3: unknown element <text>"},
		"bad path":      {`<svg viewBox="0 0 4 4"><path d="L1 1"/></svg>`, "must start with M"},
		"short path":    {`<svg viewBox="0 0 4 4"><path d="M1"/></svg>`, "number is missing"},
		"draws nothing": {`<svg viewBox="0 0 4 4"></svg>`, "draws nothing"},
		"bad width":     {`<svg viewBox="0 0 4 4" stroke-width="x"><path d="M0 0L1 1"/></svg>`, "stroke-width"},
	} {
		_, err := parseSVG("t.svg", []byte(tt.src))
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

func TestPathDataCommands(t *testing.T) {
	tests := []struct {
		name, d string
		last    [2]float64
		kinds   string
	}{
		{"absolute lines", "M1 2L3 4L5 6", [2]float64{5, 6}, "MLL"},
		{"relative", "m1 2l2 2h3v-1", [2]float64{6, 3}, "MLLL"},
		{"implicit lineto after M", "M0 0 4 4 8 0", [2]float64{8, 0}, "MLL"},
		{"packed numbers", "M1-2.5.5.5", [2]float64{0.5, 0.5}, "ML"},
		{"cubic", "M0 0C1 1 2 1 3 0", [2]float64{3, 0}, "MC"},
		{"smooth cubic", "M0 0C1 1 2 1 3 0S5 -1 6 0", [2]float64{6, 0}, "MCC"},
		{"quadratic", "M0 0Q1 2 2 0", [2]float64{2, 0}, "MC"},
		{"close returns to start", "M1 1L3 1L3 3Z", [2]float64{3, 3}, "MLLZ"},
		{"arc is cubics ending on the point", "M0 0A5 5 0 0 1 10 0", [2]float64{10, 0}, "MCC"},
		{"two subpaths", "M0 0L1 1M5 5L6 6", [2]float64{6, 6}, "MLML"},
	}
	for _, tt := range tests {
		cmds, err := parsePathData(tt.d)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		var kinds string
		for _, c := range cmds {
			kinds += string(c.op)
		}
		if kinds != tt.kinds {
			t.Errorf("%s: commands %q, want %q", tt.name, kinds, tt.kinds)
		}
		last := cmds[len(cmds)-1]
		if last.op != 'Z' {
			var got [2]float64
			switch last.op {
			case 'C':
				got = [2]float64{last.p[4], last.p[5]}
			default:
				got = [2]float64{last.p[0], last.p[1]}
			}
			if got != tt.last {
				t.Errorf("%s: ends at %v, want %v", tt.name, got, tt.last)
			}
		}
	}
	// a half circle of radius 5 passes through (5, -5) or (5, 5) at its middle
	cmds, _ := parsePathData("M0 0A5 5 0 0 1 10 0")
	mid := cmds[1].p
	if d := mid[4] - 5; d < -0.01 || d > 0.01 {
		t.Errorf("arc middle x = %v", mid[4])
	}
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

func TestIconMasksAreCachedPerSizeAndWeightAndTokenIconsAreDrawnDirectly(t *testing.T) {
	ic, _ := lookupIcon("trash")
	a := iconMask(ic, 2, 1)
	if b := iconMask(ic, 2, 1); b != a {
		t.Fatal("the same icon, scale and weight must reuse its mask")
	}
	if iconMask(ic, 3, 1) == a || iconMask(ic, 2, 1.5) == a {
		t.Fatal("another scale or weight is another mask")
	}
	if b := a.Bounds(); b.Dx() != 28+2*iconMaskMargin || b.Dy() != 32+2*iconMaskMargin {
		t.Fatalf("mask %v for a 14x16 icon at 2x", b)
	}
	if !ic.cacheable() {
		t.Fatal("a currentColor icon is cacheable")
	}
	tok, err := parseSVG("t.svg", []byte(`<svg viewBox="0 0 4 4"><path d="M0 0L4 4" stroke="token(error)" fill="none"/></svg>`))
	if err != nil || tok.cacheable() {
		t.Fatalf("an icon with a token paint is drawn directly: %v %v", err, tok.cacheable())
	}
	masks.Lock()
	for k := range masks.m {
		delete(masks.m, k)
	}
	masks.Unlock()
	c := testCtx(t)
	for range 3 { // several frames, hovered and not: one mask per size
		drawn(t, c, Row(W(30), H(30), Icon("trash")), 30, 30)
	}
	masks.Lock()
	n := len(masks.m)
	masks.Unlock()
	if n != 1 {
		t.Fatalf("%d masks after drawing one icon three times", n)
	}
}
