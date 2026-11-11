package view

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// An icon is a small SVG file: the subset below, drawn by the toolkit in
// the color of the element it sits in, so icons follow the theme.
//
//	<svg viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.5"
//	     stroke-linecap="round" data-hover="text">
//	  <path d="M1 6H11M6 1V11"/>
//	</svg>
//
// Elements: path (commands M L H V C S Q T A Z, absolute and relative),
// line, polyline, polygon, rect (with rx), circle and ellipse. Attributes
// fill and stroke (none, currentColor or token(name) for a theme color),
// stroke-width, stroke-linecap and stroke-linejoin, inherited from svg or
// a g element. data-hover names the token an icon takes under the pointer.
// Anything else is an error that names the file and line.

// svgCmd is one path command in icon units; C is the only curve (Q, T and
// A are converted).
type svgCmd struct {
	op byte // 'M', 'L', 'C' or 'Z'
	p  [6]float64
}

// svgPaint is a fill or stroke: nothing, the element's color, or a token.
type svgPaint struct {
	none  bool
	token core.Token // "" means currentColor
}

type svgShape struct {
	cmds       []svgCmd
	fill, line svgPaint
	width      float64
	cap        vector.LineCap
	join       vector.LineJoin
}

// Icon is a parsed SVG icon.
type svgIcon struct {
	name   string
	w, h   float64 // the viewBox size, which is the icon's natural size in pixels
	hover  core.Token
	shapes []svgShape
}

type svgStyle struct {
	fill, stroke svgPaint
	width        float64
	cap          vector.LineCap
	join         vector.LineJoin
}

func defaultSVGStyle() svgStyle {
	return svgStyle{fill: svgPaint{}, stroke: svgPaint{none: true}, width: 1, cap: vector.LineCapButt, join: vector.LineJoinMiter}
}

type svgError struct {
	name string
	line int
	msg  string
}

func (e *svgError) Error() string { return fmt.Sprintf("%s:%d: %s", e.name, e.line, e.msg) }

// parseSVG reads one icon file.
func parseSVG(name string, src []byte) (*svgIcon, error) {
	d := xml.NewDecoder(bytes.NewReader(src))
	line := func() int { return 1 + bytes.Count(src[:min(int(d.InputOffset()), len(src))], []byte("\n")) }
	fail := func(format string, args ...any) (*svgIcon, error) {
		return nil, &svgError{name, line(), fmt.Sprintf(format, args...)}
	}
	icon := &svgIcon{name: name, hover: core.Text}
	styles := []svgStyle{defaultSVGStyle()}
	seenRoot := false
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fail("%v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			st := styles[len(styles)-1]
			attrs := map[string]string{}
			for _, a := range t.Attr {
				attrs[a.Name.Local] = a.Value
			}
			var err error
			if st, err = applyStyle(st, attrs); err != nil {
				return fail("%v", err)
			}
			styles = append(styles, st)
			tag := t.Name.Local
			if !seenRoot {
				if tag != "svg" {
					return fail("an icon starts with <svg>, not <%s>", tag)
				}
				seenRoot = true
				vb := strings.Fields(strings.ReplaceAll(attrs["viewBox"], ",", " "))
				if len(vb) != 4 {
					return fail(`svg needs viewBox="0 0 width height"`)
				}
				w, e1 := strconv.ParseFloat(vb[2], 64)
				h, e2 := strconv.ParseFloat(vb[3], 64)
				if e1 != nil || e2 != nil || w <= 0 || h <= 0 {
					return fail("bad viewBox %q", attrs["viewBox"])
				}
				icon.w, icon.h = w, h
				if v := attrs["data-hover"]; v != "" {
					icon.hover = core.Token(v)
				}
				continue
			}
			if tag == "g" {
				continue
			}
			cmds, err := shapeCmds(tag, attrs)
			if err != nil {
				return fail("%v", err)
			}
			icon.shapes = append(icon.shapes, svgShape{cmds: cmds, fill: st.fill, line: st.stroke, width: st.width, cap: st.cap, join: st.join})
		case xml.EndElement:
			styles = styles[:len(styles)-1]
		}
	}
	if !seenRoot {
		return fail("the file is empty")
	}
	if len(icon.shapes) == 0 {
		return fail("the icon draws nothing")
	}
	return icon, nil
}

func parsePaint(v string) (svgPaint, error) {
	v = strings.TrimSpace(v)
	switch {
	case v == "none":
		return svgPaint{none: true}, nil
	case v == "currentColor":
		return svgPaint{}, nil
	case strings.HasPrefix(v, "token(") && strings.HasSuffix(v, ")"):
		return svgPaint{token: core.Token(strings.TrimSpace(v[6 : len(v)-1]))}, nil
	}
	return svgPaint{}, fmt.Errorf("a paint is none, currentColor or token(name), not %q", v)
}

func applyStyle(st svgStyle, attrs map[string]string) (svgStyle, error) {
	var err error
	if v, ok := attrs["fill"]; ok {
		if st.fill, err = parsePaint(v); err != nil {
			return st, err
		}
	}
	if v, ok := attrs["stroke"]; ok {
		if st.stroke, err = parsePaint(v); err != nil {
			return st, err
		}
	}
	if v, ok := attrs["stroke-width"]; ok {
		if st.width, err = strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil || st.width < 0 {
			return st, fmt.Errorf("bad stroke-width %q", v)
		}
	}
	switch attrs["stroke-linecap"] {
	case "", "butt":
		if _, ok := attrs["stroke-linecap"]; ok {
			st.cap = vector.LineCapButt
		}
	case "round":
		st.cap = vector.LineCapRound
	case "square":
		st.cap = vector.LineCapSquare
	default:
		return st, fmt.Errorf("bad stroke-linecap %q", attrs["stroke-linecap"])
	}
	switch attrs["stroke-linejoin"] {
	case "":
	case "miter":
		st.join = vector.LineJoinMiter
	case "round":
		st.join = vector.LineJoinRound
	case "bevel":
		st.join = vector.LineJoinBevel
	default:
		return st, fmt.Errorf("bad stroke-linejoin %q", attrs["stroke-linejoin"])
	}
	return st, nil
}

func num(attrs map[string]string, name string, def float64) (float64, error) {
	v, ok := attrs[name]
	if !ok {
		return def, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a number", name, v)
	}
	return f, nil
}

// shapeCmds turns one shape element into path commands.
func shapeCmds(tag string, a map[string]string) ([]svgCmd, error) {
	switch tag {
	case "path":
		return parsePathData(a["d"])
	case "line":
		x1, e1 := num(a, "x1", 0)
		y1, e2 := num(a, "y1", 0)
		x2, e3 := num(a, "x2", 0)
		y2, e4 := num(a, "y2", 0)
		if err := errors.Join(e1, e2, e3, e4); err != nil {
			return nil, err
		}
		return []svgCmd{{'M', [6]float64{x1, y1}}, {'L', [6]float64{x2, y2}}}, nil
	case "polyline", "polygon":
		nums, err := parseNumbers(a["points"])
		if err != nil || len(nums) < 4 || len(nums)%2 != 0 {
			return nil, fmt.Errorf("%s needs pairs of numbers in points", tag)
		}
		var cmds []svgCmd
		for i := 0; i < len(nums); i += 2 {
			op := byte('L')
			if i == 0 {
				op = 'M'
			}
			cmds = append(cmds, svgCmd{op, [6]float64{nums[i], nums[i+1]}})
		}
		if tag == "polygon" {
			cmds = append(cmds, svgCmd{op: 'Z'})
		}
		return cmds, nil
	case "rect":
		x, e1 := num(a, "x", 0)
		y, e2 := num(a, "y", 0)
		w, e3 := num(a, "width", 0)
		h, e4 := num(a, "height", 0)
		rx, e5 := num(a, "rx", 0)
		if err := errors.Join(e1, e2, e3, e4, e5); err != nil {
			return nil, err
		}
		if w <= 0 || h <= 0 {
			return nil, errors.New("rect needs width and height")
		}
		rx = min(rx, w/2, h/2)
		if rx <= 0 {
			return []svgCmd{{'M', [6]float64{x, y}}, {'L', [6]float64{x + w, y}}, {'L', [6]float64{x + w, y + h}}, {'L', [6]float64{x, y + h}}, {op: 'Z'}}, nil
		}
		d := fmt.Sprintf("M%g %gH%gA%g %g 0 0 1 %g %gV%gA%g %g 0 0 1 %g %gH%gA%g %g 0 0 1 %g %gV%gA%g %g 0 0 1 %g %gZ",
			x+rx, y, x+w-rx, rx, rx, x+w, y+rx, y+h-rx, rx, rx, x+w-rx, y+h, x+rx, rx, rx, x, y+h-rx, y+rx, rx, rx, x+rx, y)
		return parsePathData(d)
	case "circle", "ellipse":
		cx, e1 := num(a, "cx", 0)
		cy, e2 := num(a, "cy", 0)
		rx, e3 := num(a, "rx", 0)
		ry, e4 := num(a, "ry", 0)
		if tag == "circle" {
			r, e := num(a, "r", 0)
			rx, ry, e3, e4 = r, r, e, nil
		}
		if err := errors.Join(e1, e2, e3, e4); err != nil {
			return nil, err
		}
		if rx <= 0 || ry <= 0 {
			return nil, fmt.Errorf("%s needs a radius", tag)
		}
		d := fmt.Sprintf("M%g %gA%g %g 0 1 0 %g %gA%g %g 0 1 0 %g %gZ", cx+rx, cy, rx, ry, cx-rx, cy, rx, ry, cx+rx, cy)
		return parsePathData(d)
	}
	return nil, fmt.Errorf("unknown element <%s>; icons use path, line, polyline, polygon, rect, circle, ellipse, g", tag)
}

func parseNumbers(s string) ([]float64, error) {
	var out []float64
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ',' || r == '\n' || r == '\t' }) {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// pathScanner reads path data: commands and numbers, with or without
// separators ("M1-2.5.5" is 1, -2.5, .5).
type pathScanner struct {
	s string
	i int
}

func (p *pathScanner) skip() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == ',' || p.s[p.i] == '\n' || p.s[p.i] == '\t' || p.s[p.i] == '\r') {
		p.i++
	}
}

func (p *pathScanner) command() (byte, bool) {
	p.skip()
	if p.i < len(p.s) && strings.ContainsRune("MmLlHhVvCcSsQqTtAaZz", rune(p.s[p.i])) {
		c := p.s[p.i]
		p.i++
		return c, true
	}
	return 0, false
}

func (p *pathScanner) more() bool {
	p.skip()
	return p.i < len(p.s) && !strings.ContainsRune("MmLlHhVvCcSsQqTtAaZz", rune(p.s[p.i]))
}

func (p *pathScanner) number() (float64, error) {
	p.skip()
	start := p.i
	if p.i < len(p.s) && (p.s[p.i] == '-' || p.s[p.i] == '+') {
		p.i++
	}
	dot := false
	for p.i < len(p.s) {
		c := p.s[p.i]
		if c >= '0' && c <= '9' {
			p.i++
		} else if c == '.' && !dot {
			dot = true
			p.i++
		} else {
			break
		}
	}
	if p.i < len(p.s) && (p.s[p.i] == 'e' || p.s[p.i] == 'E') {
		p.i++
		if p.i < len(p.s) && (p.s[p.i] == '-' || p.s[p.i] == '+') {
			p.i++
		}
		for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
		}
	}
	return strconv.ParseFloat(p.s[start:p.i], 64)
}

// flag reads an arc flag, which may be written without a separator ("11").
func (p *pathScanner) flag() (bool, error) {
	p.skip()
	if p.i < len(p.s) && (p.s[p.i] == '0' || p.s[p.i] == '1') {
		v := p.s[p.i] == '1'
		p.i++
		return v, nil
	}
	return false, errors.New("an arc flag is 0 or 1")
}

func (p *pathScanner) numbers(n int) ([]float64, error) {
	out := make([]float64, n)
	for i := range out {
		v, err := p.number()
		if err != nil {
			return nil, fmt.Errorf("path data: a number is missing near %q", p.s[min(p.i, len(p.s)):])
		}
		out[i] = v
	}
	return out, nil
}

// parsePathData turns path data into absolute M, L, C and Z commands.
func parsePathData(d string) ([]svgCmd, error) {
	p := &pathScanner{s: d}
	var cmds []svgCmd
	var cx, cy, sx, sy float64 // current point and subpath start
	var lastCx, lastCy float64 // last control point, for S and T
	var prev byte              // previous command letter, upper case
	cubic := func(x1, y1, x2, y2, x, y float64) {
		cmds = append(cmds, svgCmd{'C', [6]float64{x1, y1, x2, y2, x, y}})
		lastCx, lastCy = x2, y2
		cx, cy = x, y
	}
	for {
		c, ok := p.command()
		if !ok {
			p.skip()
			if p.i < len(p.s) {
				return nil, fmt.Errorf("path data: unexpected %q", p.s[p.i:min(p.i+6, len(p.s))])
			}
			break
		}
		rel := c >= 'a'
		up := c &^ 0x20
		off := func(x, y float64) (float64, float64) {
			if rel {
				return cx + x, cy + y
			}
			return x, y
		}
		if up == 'Z' {
			cmds = append(cmds, svgCmd{op: 'Z'})
			cx, cy = sx, sy
			prev = 'Z'
			continue
		}
		first := true
		for first || p.more() {
			first = false
			switch up {
			case 'M', 'L':
				n, err := p.numbers(2)
				if err != nil {
					return nil, err
				}
				x, y := off(n[0], n[1])
				op := up
				cmds = append(cmds, svgCmd{op, [6]float64{x, y}})
				cx, cy = x, y
				if up == 'M' && op == 'M' {
					sx, sy = x, y
				}
				if up == 'M' { // further pairs after M are implicit L
					up = 'L'
				}
				lastCx, lastCy = x, y
			case 'H', 'V':
				n, err := p.numbers(1)
				if err != nil {
					return nil, err
				}
				x, y := cx, cy
				switch {
				case up == 'H' && rel:
					x += n[0]
				case up == 'H':
					x = n[0]
				case rel:
					y += n[0]
				default:
					y = n[0]
				}
				cmds = append(cmds, svgCmd{'L', [6]float64{x, y}})
				cx, cy = x, y
				lastCx, lastCy = x, y
			case 'C':
				n, err := p.numbers(6)
				if err != nil {
					return nil, err
				}
				x1, y1 := off(n[0], n[1])
				x2, y2 := off(n[2], n[3])
				x, y := off(n[4], n[5])
				cubic(x1, y1, x2, y2, x, y)
			case 'S':
				n, err := p.numbers(4)
				if err != nil {
					return nil, err
				}
				x1, y1 := cx, cy
				if prev == 'C' || prev == 'S' {
					x1, y1 = 2*cx-lastCx, 2*cy-lastCy
				}
				x2, y2 := off(n[0], n[1])
				x, y := off(n[2], n[3])
				cubic(x1, y1, x2, y2, x, y)
			case 'Q', 'T':
				var qx, qy, x, y float64
				if up == 'Q' {
					n, err := p.numbers(4)
					if err != nil {
						return nil, err
					}
					qx, qy = off(n[0], n[1])
					x, y = off(n[2], n[3])
				} else {
					n, err := p.numbers(2)
					if err != nil {
						return nil, err
					}
					qx, qy = cx, cy
					if prev == 'Q' || prev == 'T' {
						qx, qy = 2*cx-lastCx, 2*cy-lastCy
					}
					x, y = off(n[0], n[1])
				}
				x0, y0 := cx, cy
				cubic(x0+2.0/3*(qx-x0), y0+2.0/3*(qy-y0), x+2.0/3*(qx-x), y+2.0/3*(qy-y), x, y)
				lastCx, lastCy = qx, qy
			case 'A':
				n, err := p.numbers(3)
				if err != nil {
					return nil, err
				}
				large, err := p.flag()
				if err != nil {
					return nil, err
				}
				sweep, err := p.flag()
				if err != nil {
					return nil, err
				}
				e, err := p.numbers(2)
				if err != nil {
					return nil, err
				}
				x, y := off(e[0], e[1])
				for _, seg := range arcToCubics(cx, cy, n[0], n[1], n[2], large, sweep, x, y) {
					cmds = append(cmds, svgCmd{'C', seg})
				}
				cx, cy = x, y
				lastCx, lastCy = x, y
			}
			prev = up
		}
		prev = up
	}
	if len(cmds) == 0 || cmds[0].op != 'M' {
		return nil, errors.New("path data must start with M")
	}
	return cmds, nil
}

// arcToCubics converts an SVG elliptical arc to cubic segments, by the
// endpoint-to-center conversion of the SVG specification.
func arcToCubics(x1, y1, rx, ry, rotDeg float64, large, sweep bool, x2, y2 float64) [][6]float64 {
	if x1 == x2 && y1 == y2 {
		return nil
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		return [][6]float64{{x1, y1, x2, y2, x2, y2}} // a straight line
	}
	phi := rotDeg * math.Pi / 180
	cosP, sinP := math.Cos(phi), math.Sin(phi)
	dx, dy := (x1-x2)/2, (y1-y2)/2
	x1p, y1p := cosP*dx+sinP*dy, -sinP*dx+cosP*dy
	if lam := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry); lam > 1 {
		s := math.Sqrt(lam)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	coef := 0.0
	if den != 0 {
		coef = math.Sqrt(math.Max(num/den, 0))
	}
	if large == sweep {
		coef = -coef
	}
	cxp, cyp := coef*rx*y1p/ry, -coef*ry*x1p/rx
	cx, cy := cosP*cxp-sinP*cyp+(x1+x2)/2, sinP*cxp+cosP*cyp+(y1+y2)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		a := math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
		return a
	}
	th1 := angle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	dth := angle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if !sweep && dth > 0 {
		dth -= 2 * math.Pi
	} else if sweep && dth < 0 {
		dth += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dth) / (math.Pi / 2)))
	if n < 1 {
		n = 1
	}
	step := dth / float64(n)
	t := 4.0 / 3 * math.Tan(step/4)
	var out [][6]float64
	pt := func(a float64) (float64, float64, float64, float64) {
		cosA, sinA := math.Cos(a), math.Sin(a)
		x := cosP*rx*cosA - sinP*ry*sinA + cx
		y := sinP*rx*cosA + cosP*ry*sinA + cy
		dxA := -cosP*rx*sinA - sinP*ry*cosA
		dyA := -sinP*rx*sinA + cosP*ry*cosA
		return x, y, dxA, dyA
	}
	a := th1
	for i := 0; i < n; i++ {
		xa, ya, dxa, dya := pt(a)
		xb, yb, dxb, dyb := pt(a + step)
		out = append(out, [6]float64{xa + t*dxa, ya + t*dya, xb - t*dxb, yb - t*dyb, xb, yb})
		a += step
	}
	// land exactly on the end point
	out[len(out)-1][4], out[len(out)-1][5] = x2, y2
	return out
}
