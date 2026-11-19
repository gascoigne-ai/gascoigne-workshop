package widget

import (
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

// svgWant is one expected path command: its letter and the points that
// matter (the first two for M and L, all six for C).
type svgWant struct {
	op byte
	p  []float64
}

func sameCommands(t *testing.T, got []svgCmd, want []svgWant) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%d commands %v, want %d", len(got), got, len(want))
	}
	for i, cmd := range got {
		if cmd.op != want[i].op {
			t.Fatalf("command %d is %q, want %q (all: %v)", i, cmd.op, want[i].op, got)
		}
		for j, value := range want[i].p {
			if math.Abs(cmd.p[j]-value) > 1e-9 {
				t.Fatalf("command %d (%q) point %d is %v, want %v (all: %v)", i, cmd.op, j, cmd.p[j], value, got)
			}
		}
	}
}

func TestPathDataExactCommands(t *testing.T) {
	tests := []struct {
		name, d string
		want    []svgWant
	}{
		{"smooth cubic mirrors the last control point", "M0 0C1 1 2 1 3 0S5 -1 6 0",
			[]svgWant{{'M', []float64{0, 0}}, {'C', []float64{1, 1, 2, 1, 3, 0}}, {'C', []float64{4, -1, 5, -1, 6, 0}}}},
		{"smooth cubic without a cubic before starts at the point", "M1 1S3 3 4 4",
			[]svgWant{{'M', []float64{1, 1}}, {'C', []float64{1, 1, 3, 3, 4, 4}}}},
		{"relative cubic", "M1 1c1 1 2 2 3 3",
			[]svgWant{{'M', []float64{1, 1}}, {'C', []float64{2, 2, 3, 3, 4, 4}}}},
		{"relative smooth cubic", "M0 0c1 1 2 1 3 0s2 -1 3 0",
			[]svgWant{{'M', []float64{0, 0}}, {'C', []float64{1, 1, 2, 1, 3, 0}}, {'C', []float64{4, -1, 5, -1, 6, 0}}}},
		{"quadratic becomes a cubic", "M0 0Q3 3 6 0",
			[]svgWant{{'M', []float64{0, 0}}, {'C', []float64{2, 2, 4, 2, 6, 0}}}},
		{"smooth quadratic mirrors the control point", "M0 0Q3 3 6 0T12 0",
			[]svgWant{{'M', []float64{0, 0}}, {'C', []float64{2, 2, 4, 2, 6, 0}}, {'C', []float64{8, -2, 10, -2, 12, 0}}}},
		{"smooth quadratic without a quadratic before is a line-like curve", "M0 0T6 0",
			[]svgWant{{'M', []float64{0, 0}}, {'C', []float64{0, 0, 2, 0, 6, 0}}}},
		{"horizontal and vertical, absolute and relative", "M1 1H5V4h-2v-1",
			[]svgWant{{'M', []float64{1, 1}}, {'L', []float64{5, 1}}, {'L', []float64{5, 4}}, {'L', []float64{3, 4}}, {'L', []float64{3, 3}}}},
		{"relative moveto continues with relative lineto", "m1 1 2 2",
			[]svgWant{{'M', []float64{1, 1}}, {'L', []float64{3, 3}}}},
		{"repeated lineto arguments", "M0 0L1 1 2 2",
			[]svgWant{{'M', []float64{0, 0}}, {'L', []float64{1, 1}}, {'L', []float64{2, 2}}}},
		{"repeated horizontal arguments", "M0 0H1 2",
			[]svgWant{{'M', []float64{0, 0}}, {'L', []float64{1, 0}}, {'L', []float64{2, 0}}}},
		{"repeated cubic arguments", "M0 0C1 1 2 2 3 3 4 4 5 5 6 6",
			[]svgWant{{'M', []float64{0, 0}}, {'C', []float64{1, 1, 2, 2, 3, 3}}, {'C', []float64{4, 4, 5, 5, 6, 6}}}},
		{"separators", "M 0 0 L 1 1 , 2 2",
			[]svgWant{{'M', []float64{0, 0}}, {'L', []float64{1, 1}}, {'L', []float64{2, 2}}}},
		{"close goes back to the subpath start", "M1 1L3 1Z l1 1",
			[]svgWant{{'M', []float64{1, 1}}, {'L', []float64{3, 1}}, {'Z', nil}, {'L', []float64{2, 2}}}},
		{"close goes back to the start of the latest subpath", "M1 1L3 1M5 5L6 5Z h1",
			[]svgWant{{'M', []float64{1, 1}}, {'L', []float64{3, 1}}, {'M', []float64{5, 5}}, {'L', []float64{6, 5}}, {'Z', nil}, {'L', []float64{6, 5}}}},
		{"an implicit lineto does not move the subpath start", "M1 1 4 4Z l1 0",
			[]svgWant{{'M', []float64{1, 1}}, {'L', []float64{4, 4}}, {'Z', nil}, {'L', []float64{2, 1}}}},
		{"relative arc ends at the offset point", "M1 0a5 5 0 0 1 10 0",
			[]svgWant{{'M', []float64{1, 0}}, {'C', nil}, {'C', nil}}},
		{"an arc to its own start draws nothing", "M1 1A5 5 0 0 1 1 1L4 4",
			[]svgWant{{'M', []float64{1, 1}}, {'L', []float64{4, 4}}}},
		{"arc flags without separators", "M0 0A5 5 0 1110 0",
			[]svgWant{{'M', []float64{0, 0}}, {'C', nil}, {'C', nil}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds, err := parsePathData(tt.d)
			if err != nil {
				t.Fatal(err)
			}
			sameCommands(t, cmds, tt.want)
		})
	}
}

func TestPathDataErrors(t *testing.T) {
	tests := []struct{ name, d, want string }{
		{"empty", "", "must start with M"},
		{"starts with a line", "L1 1", "must start with M"},
		{"short lineto", "M0 0L1", "number is missing"},
		{"short cubic", "M0 0C1 1 2 2 3", "number is missing"},
		{"arguments after close", "M0 0L1 1Z 5", "unexpected"},
		{"unknown command", "M0 0X1", "number is missing"},
		{"bad arc flag", "M0 0A5 5 0 2 1 10 0", "arc flag"},
		{"short arc", "M0 0A5 5 0 0 1 10", "number is missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePathData(tt.d)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func svgShapeOps(t *testing.T, src string) (string, []svgCmd) {
	t.Helper()
	icon, err := parseSVG("t.svg", []byte(`<svg viewBox="0 0 20 20">`+src+`</svg>`))
	if err != nil {
		t.Fatal(err)
	}
	cmds := icon.shapes[0].cmds
	var ops string
	for _, cmd := range cmds {
		ops += string(cmd.op)
	}
	return ops, cmds
}

func TestSVGShapeElementsBecomePathCommands(t *testing.T) {
	tests := []struct {
		name, element, ops string
	}{
		{"path", `<path d="M1 1L2 2"/>`, "ML"},
		{"line", `<line x1="1" y1="2" x2="3" y2="4"/>`, "ML"},
		{"polyline", `<polyline points="0,0 5,5 10,0"/>`, "MLL"},
		{"polygon closes", `<polygon points="0,0 5,5 10,0"/>`, "MLLZ"},
		{"plain rect", `<rect x="1" y="2" width="4" height="3"/>`, "MLLLZ"},
		{"rounded rect", `<rect width="10" height="10" rx="2"/>`, "MLCLCLCLCZ"},
		{"circle", `<circle cx="5" cy="5" r="3"/>`, "MCCCCZ"},
		{"ellipse", `<ellipse cx="5" cy="5" rx="3" ry="2"/>`, "MCCCCZ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops, _ := svgShapeOps(t, tt.element)
			if ops != tt.ops {
				t.Fatalf("commands %q, want %q", ops, tt.ops)
			}
		})
	}
}

func TestSVGShapeGeometry(t *testing.T) {
	_, line := svgShapeOps(t, `<line x1="1" y1="2" x2="3" y2="4"/>`)
	sameCommands(t, line, []svgWant{{'M', []float64{1, 2}}, {'L', []float64{3, 4}}})
	_, rect := svgShapeOps(t, `<rect x="1" y="2" width="4" height="3"/>`)
	sameCommands(t, rect, []svgWant{{'M', []float64{1, 2}}, {'L', []float64{5, 2}}, {'L', []float64{5, 5}}, {'L', []float64{1, 5}}, {'Z', nil}})
	_, clamped := svgShapeOps(t, `<rect width="4" height="2" rx="9"/>`) // rx is at most half the shorter side
	if first := clamped[0]; first.p[0] != 1 || first.p[1] != 0 {
		t.Fatalf("clamped rect starts at %v, want 1 0", first.p[:2])
	}
	_, circle := svgShapeOps(t, `<circle cx="5" cy="6" r="3"/>`)
	if first := circle[0]; first.p[0] != 8 || first.p[1] != 6 {
		t.Fatalf("circle starts at %v, want 8 6", first.p[:2])
	}
}

func TestSVGShapeErrors(t *testing.T) {
	tests := []struct{ name, element, want string }{
		{"line with a bad number", `<line x1="a"/>`, "not a number"},
		{"polyline with one point", `<polyline points="1,2"/>`, "polyline needs pairs of numbers"},
		{"polygon with an odd count", `<polygon points="1,2 3,4 5"/>`, "polygon needs pairs of numbers"},
		{"rect without size", `<rect width="4"/>`, "rect needs width and height"},
		{"rect with a bad rx", `<rect width="4" height="4" rx="x"/>`, "not a number"},
		{"circle without radius", `<circle r="0"/>`, "circle needs a radius"},
		{"ellipse without a radius", `<ellipse rx="3"/>`, "ellipse needs a radius"},
		{"circle with a bad number", `<circle r="x"/>`, "not a number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSVG("t.svg", []byte(`<svg viewBox="0 0 20 20">`+tt.element+`</svg>`))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSVGUnknownElementListsTheSupportedOnes(t *testing.T) {
	_, err := parseSVG("t.svg", []byte(`<svg viewBox="0 0 20 20"><text/></svg>`))
	if err == nil || !strings.Contains(err.Error(), "unknown element <text>") {
		t.Fatalf("err = %v", err)
	}
	for _, name := range []string{"path", "line", "polyline", "polygon", "rect", "circle", "ellipse", "g"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not list %s", err, name)
		}
	}
}

func TestSVGStyleInheritsFromSvgAndGroups(t *testing.T) {
	src := `<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="bevel">
	<path d="M0 0L1 1"/>
	<g stroke="token(error)" stroke-width="3" stroke-linecap="square" stroke-linejoin="miter"><path d="M0 0L1 1"/></g>
	<g stroke-linecap="butt"><path d="M0 0L1 1" fill="token(accent)"/></g>
	<path d="M0 0L1 1"/>
	</svg>`
	icon, err := parseSVG("t.svg", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(icon.shapes) != 4 {
		t.Fatalf("%d shapes, want 4", len(icon.shapes))
	}
	first, grouped, butt, last := icon.shapes[0], icon.shapes[1], icon.shapes[2], icon.shapes[3]
	if !first.fill.none || first.line.none || first.width != 2 || first.cap != vector.LineCapRound || first.join != vector.LineJoinBevel {
		t.Errorf("first shape style %+v", first)
	}
	if grouped.line.token != "error" || grouped.width != 3 || grouped.cap != vector.LineCapSquare || grouped.join != vector.LineJoinMiter {
		t.Errorf("grouped shape style %+v", grouped)
	}
	if butt.cap != vector.LineCapButt || butt.join != vector.LineJoinBevel || butt.fill.token != "accent" || butt.width != 2 {
		t.Errorf("butt shape style %+v", butt)
	}
	if last.cap != vector.LineCapRound || last.line.token != "" {
		t.Errorf("a group's style must end with it: %+v", last)
	}
}

func TestSVGStyleDefaults(t *testing.T) {
	icon, err := parseSVG("t.svg", []byte(`<svg viewBox="0 0 20 20"><path d="M0 0L1 1"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	shape := icon.shapes[0]
	if shape.fill.none || !shape.line.none || shape.width != 1 || shape.cap != vector.LineCapButt || shape.join != vector.LineJoinMiter {
		t.Errorf("default style %+v", shape)
	}
}

func TestSVGStyleErrors(t *testing.T) {
	tests := []struct{ name, attrs, want string }{
		{"cap", `stroke-linecap="triangle"`, `bad stroke-linecap "triangle"`},
		{"join", `stroke-linejoin="arcs"`, `bad stroke-linejoin "arcs"`},
		{"negative width", `stroke-width="-1"`, "bad stroke-width"},
		{"bad fill", `fill="red"`, "none, currentColor or token"},
		{"bad stroke", `stroke="red"`, "none, currentColor or token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSVG("t.svg", []byte(`<svg viewBox="0 0 20 20" `+tt.attrs+`><path d="M0 0L1 1"/></svg>`))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSVGRootErrorsAndHover(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{"not svg", `<g/>`, "starts with <svg>, not <g>"},
		{"empty file", ``, "the file is empty"},
		{"three numbers", `<svg viewBox="0 0 4"><path d="M0 0L1 1"/></svg>`, "viewBox"},
		{"not numbers", `<svg viewBox="0 0 a b"><path d="M0 0L1 1"/></svg>`, `bad viewBox "0 0 a b"`},
		{"zero size", `<svg viewBox="0 0 0 4"><path d="M0 0L1 1"/></svg>`, "bad viewBox"},
		{"malformed xml", `<svg viewBox="0 0 4 4"><path`, "XML syntax error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSVG("t.svg", []byte(tt.src))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
	icon, err := parseSVG("t.svg", []byte(`<svg viewBox="0,0,12 9" data-hover="accent"><path d="M0 0L1 1"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if icon.w != 12 || icon.h != 9 || icon.hover != "accent" {
		t.Errorf("icon %v x %v hover %q, want 12 x 9 accent", icon.w, icon.h, icon.hover)
	}
	plain, _ := parseSVG("t.svg", []byte(`<svg viewBox="0 0 4 4"><path d="M0 0L1 1"/></svg>`))
	if plain.hover != "text" {
		t.Errorf("default hover %q, want text", plain.hover)
	}
}

func TestPicturesReadColorsAndGradientsIconsDoNot(t *testing.T) {
	src := `<svg viewBox="0 0 10 10"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="10" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#000000"/><stop offset="1" stop-color="#ffffff"/></linearGradient></defs><path fill="url(#g)" d="M0 0H10V10Z"/><path fill="#0f0" stroke="none" d="M0 0H5V5Z"/></svg>`
	picture, err := parseSVGAs("t.svg", []byte(src), picturePaints)
	if err != nil {
		t.Fatal(err)
	}
	gradient := picture.gradients["g"]
	if gradient == nil || len(gradient.stops) != 2 || gradient.y2 != 10 {
		t.Fatalf("gradient %+v", gradient)
	}
	if picture.shapes[0].fill.gradient != "g" || !picture.shapes[1].fill.solid || picture.shapes[1].fill.rgba.G != 0xff {
		t.Errorf("fills %+v %+v", picture.shapes[0].fill, picture.shapes[1].fill)
	}
	if _, err := parseSVG("t.svg", []byte(src)); err == nil {
		t.Error("an icon must refuse a picture's defs and colors")
	}
	_, err = parseSVGAs("t.svg", []byte(`<svg viewBox="0 0 4 4"><stop offset="0" stop-color="#000"/></svg>`), picturePaints)
	if err == nil || !strings.Contains(err.Error(), "unknown element <stop>") {
		t.Errorf("a stop outside a gradient: err = %v", err)
	}
	_, err = parseSVGAs("t.svg", []byte(`<svg viewBox="0 0 4 4"><linearGradient id="g"/></svg>`), picturePaints)
	if err == nil || !strings.Contains(err.Error(), "gradientUnits") {
		t.Errorf("a gradient without units: err = %v", err)
	}
}

// FuzzParseSVG: any bytes read as an icon or a picture return an icon or an
// error, never a panic or a hang; what parses has shapes and a size.
func FuzzParseSVG(fuzz *testing.F) {
	entries, err := iconFiles.ReadDir("icons")
	if err != nil {
		fuzz.Fatal(err)
	}
	for _, entry := range entries {
		src, err := iconFiles.ReadFile("icons/" + entry.Name())
		if err != nil {
			fuzz.Fatal(err)
		}
		fuzz.Add(src)
	}
	fuzz.Add([]byte(`<svg viewBox="0 0 10 10"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="10" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#000000"/><stop offset="1" stop-color="#ffffff"/></linearGradient></defs><rect width="10" height="10" rx="4" fill="url(#g)"/></svg>`))
	fuzz.Add([]byte(`<svg viewBox="0 0 10 10"><path d="M0 0A5 5 0 1110 0q1 2 3 4t5 5s1 1 2 2Z"/></svg>`))
	fuzz.Fuzz(func(t *testing.T, src []byte) {
		for _, paints := range []paintParser{iconPaints, picturePaints} {
			icon, err := parseSVGAs("fuzz.svg", src, paints)
			if err != nil {
				continue
			}
			if len(icon.shapes) == 0 || icon.w <= 0 || icon.h <= 0 {
				t.Fatalf("parsed without shapes or size: %+v", icon)
			}
		}
	})
}
