package widget

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image/color"
	"io"
	"maps"
	"math"
	"slices"
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
	// solid and gradient are for pictures drawn by RasterizeSVG, whose
	// colors are their own; icon files cannot use them.
	solid    bool
	rgba     color.RGBA
	gradient string // the id of a linearGradient
}

// svgGradient is a linear gradient in viewBox units, from (x1, y1) to
// (x2, y2).
type svgGradient struct {
	x1, y1, x2, y2 float64
	stops          []svgStop
}

type svgStop struct {
	offset float64
	rgba   color.RGBA
}

type svgShape struct {
	cmds       []svgCmd
	fill, line svgPaint
	width      float64
	cap        vector.LineCap
	join       vector.LineJoin
}

// svgIcon is a parsed SVG icon.
type svgIcon struct {
	name   string
	w, h   float64 // the viewBox size, which is the icon's natural size in pixels
	hover  core.Token
	shapes []svgShape
	// gradients are the picture's linearGradient elements by id.
	gradients map[string]*svgGradient
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

// paintParser says which paints a file may use: icons take the theme's
// (none, currentColor, token(name)), pictures add colors and gradients.
type paintParser struct{ colors bool }

var (
	iconPaints    = paintParser{}
	picturePaints = paintParser{colors: true}
)

// parseSVG reads one icon file.
func parseSVG(name string, src []byte) (*svgIcon, error) {
	return parseSVGAs(name, src, iconPaints)
}

// svgParser is the state of one file while its elements stream by.
type svgParser struct {
	paints   paintParser
	icon     *svgIcon
	styles   []svgStyle // one per open element, the innermost last
	gradient *svgGradient
	seenRoot bool
}

// parseSVGAs parses an icon, or with picturePaints a picture whose paints
// may be colors and gradients (RasterizeSVG).
func parseSVGAs(name string, src []byte, paints paintParser) (*svgIcon, error) {
	decoder := xml.NewDecoder(bytes.NewReader(src))
	fail := func(err error) (*svgIcon, error) {
		line := 1 + bytes.Count(src[:min(int(decoder.InputOffset()), len(src))], []byte("\n"))
		return nil, &svgError{name, line, err.Error()}
	}
	parser := &svgParser{
		paints: paints,
		icon:   &svgIcon{name: name, hover: core.Text, gradients: map[string]*svgGradient{}},
		styles: []svgStyle{defaultSVGStyle()},
	}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fail(err)
		}
		if err := parser.handle(token); err != nil {
			return fail(err)
		}
	}
	if !parser.seenRoot {
		return fail(errors.New("the file is empty"))
	}
	if len(parser.icon.shapes) == 0 {
		return fail(errors.New("the icon draws nothing"))
	}
	return parser.icon, nil
}

func (parser *svgParser) handle(token xml.Token) error {
	switch token := token.(type) {
	case xml.StartElement:
		return parser.start(token)
	case xml.EndElement:
		parser.styles = parser.styles[:len(parser.styles)-1]
	}
	return nil
}

func (parser *svgParser) start(element xml.StartElement) error {
	attrs := map[string]string{}
	for _, attr := range element.Attr {
		attrs[attr.Name.Local] = attr.Value
	}
	style, err := applyStyle(parser.styles[len(parser.styles)-1], attrs, parser.paints)
	if err != nil {
		return err
	}
	parser.styles = append(parser.styles, style)
	tag := element.Name.Local
	if !parser.seenRoot {
		parser.seenRoot = true
		return parseRoot(parser.icon, tag, attrs)
	}
	if tag == "g" {
		return nil
	}
	handled, err := parser.pictureElement(tag, attrs)
	if handled || err != nil {
		return err
	}
	cmds, err := shapeCmds(tag, attrs)
	if err != nil {
		return err
	}
	parser.icon.shapes = append(parser.icon.shapes, svgShape{cmds: cmds, fill: style.fill, line: style.stroke, width: style.width, cap: style.cap, join: style.join})
	return nil
}

// parseRoot reads the svg element: the viewBox size and data-hover.
func parseRoot(icon *svgIcon, tag string, attrs map[string]string) error {
	if tag != "svg" {
		return fmt.Errorf("an icon starts with <svg>, not <%s>", tag)
	}
	box := strings.Fields(strings.ReplaceAll(attrs["viewBox"], ",", " "))
	if len(box) != 4 {
		return errors.New(`svg needs viewBox="0 0 width height"`)
	}
	width, widthErr := strconv.ParseFloat(box[2], 64)
	height, heightErr := strconv.ParseFloat(box[3], 64)
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return fmt.Errorf("bad viewBox %q", attrs["viewBox"])
	}
	icon.w, icon.h = width, height
	if hover := attrs["data-hover"]; hover != "" {
		icon.hover = core.Token(hover)
	}
	return nil
}

// pictureElement handles the elements only a picture has (defs and
// gradients); handled is false for any other element.
func (parser *svgParser) pictureElement(tag string, attrs map[string]string) (handled bool, err error) {
	if !parser.paints.colors {
		return false, nil
	}
	switch {
	case tag == "defs":
		return true, nil
	case tag == "linearGradient":
		parser.gradient, err = parseGradient(attrs)
		if err != nil {
			return true, err
		}
		parser.icon.gradients[attrs["id"]] = parser.gradient
		return true, nil
	case tag == "stop" && parser.gradient != nil:
		stop, err := parseStop(attrs)
		if err != nil {
			return true, err
		}
		parser.gradient.stops = append(parser.gradient.stops, stop)
		return true, nil
	}
	return false, nil
}

func (paints paintParser) parse(value string) (svgPaint, error) {
	value = strings.TrimSpace(value)
	switch {
	case value == "none":
		return svgPaint{none: true}, nil
	case value == "currentColor":
		return svgPaint{}, nil
	case strings.HasPrefix(value, "token(") && strings.HasSuffix(value, ")"):
		return svgPaint{token: core.Token(strings.TrimSpace(value[6 : len(value)-1]))}, nil
	}
	if paints.colors {
		if strings.HasPrefix(value, "url(#") && strings.HasSuffix(value, ")") {
			return svgPaint{gradient: value[5 : len(value)-1]}, nil
		}
		if rgba, err := core.ParseHexStrict(value); err == nil {
			return svgPaint{solid: true, rgba: rgba}, nil
		}
	}
	return svgPaint{}, fmt.Errorf("a paint is none, currentColor or token(name), not %q", value)
}

// lineCaps has an empty row: a present but empty stroke-linecap is butt.
var lineCaps = map[string]vector.LineCap{
	"":       vector.LineCapButt,
	"butt":   vector.LineCapButt,
	"round":  vector.LineCapRound,
	"square": vector.LineCapSquare,
}

var lineJoins = map[string]vector.LineJoin{
	"miter": vector.LineJoinMiter,
	"round": vector.LineJoinRound,
	"bevel": vector.LineJoinBevel,
}

// namedAttr is the table's value for attribute name, or current when the
// attribute is absent.
func namedAttr[T any](attrs map[string]string, name string, table map[string]T, current T) (T, error) {
	value, ok := attrs[name]
	if !ok {
		return current, nil
	}
	named, ok := table[value]
	if !ok {
		return current, fmt.Errorf("bad %s %q", name, value)
	}
	return named, nil
}

func applyStyle(style svgStyle, attrs map[string]string, paints paintParser) (svgStyle, error) {
	var err error
	if value, ok := attrs["fill"]; ok {
		if style.fill, err = paints.parse(value); err != nil {
			return style, err
		}
	}
	if value, ok := attrs["stroke"]; ok {
		if style.stroke, err = paints.parse(value); err != nil {
			return style, err
		}
	}
	if value, ok := attrs["stroke-width"]; ok {
		if style.width, err = strconv.ParseFloat(strings.TrimSpace(value), 64); err != nil || style.width < 0 {
			return style, fmt.Errorf("bad stroke-width %q", value)
		}
	}
	if style.cap, err = namedAttr(attrs, "stroke-linecap", lineCaps, style.cap); err != nil {
		return style, err
	}
	if attrs["stroke-linejoin"] == "" { // present but empty keeps the inherited join
		return style, nil
	}
	style.join, err = namedAttr(attrs, "stroke-linejoin", lineJoins, style.join)
	return style, err
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

// shapeBuilder turns one shape element into path commands; tag tells
// elements that share a builder apart.
type shapeBuilder func(tag string, attrs map[string]string) ([]svgCmd, error)

// shapes are the drawable elements; the "unknown element" error lists them.
var shapes = map[string]shapeBuilder{
	"path":     pathShape,
	"line":     lineShape,
	"polyline": polyShape,
	"polygon":  polyShape,
	"rect":     rectShape,
	"circle":   ellipseShape,
	"ellipse":  ellipseShape,
}

// shapeCmds turns one shape element into path commands.
func shapeCmds(tag string, attrs map[string]string) ([]svgCmd, error) {
	build, ok := shapes[tag]
	if !ok {
		known := append(slices.Sorted(maps.Keys(shapes)), "g")
		return nil, fmt.Errorf("unknown element <%s>; icons use %s", tag, strings.Join(known, ", "))
	}
	return build(tag, attrs)
}

func pathShape(_ string, attrs map[string]string) ([]svgCmd, error) {
	return parsePathData(attrs["d"])
}

func lineShape(_ string, attrs map[string]string) ([]svgCmd, error) {
	x1, e1 := num(attrs, "x1", 0)
	y1, e2 := num(attrs, "y1", 0)
	x2, e3 := num(attrs, "x2", 0)
	y2, e4 := num(attrs, "y2", 0)
	if err := errors.Join(e1, e2, e3, e4); err != nil {
		return nil, err
	}
	return []svgCmd{{'M', [6]float64{x1, y1}}, {'L', [6]float64{x2, y2}}}, nil
}

func polyShape(tag string, attrs map[string]string) ([]svgCmd, error) {
	nums, err := parseNumbers(attrs["points"])
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
}

func rectShape(_ string, attrs map[string]string) ([]svgCmd, error) {
	x, e1 := num(attrs, "x", 0)
	y, e2 := num(attrs, "y", 0)
	w, e3 := num(attrs, "width", 0)
	h, e4 := num(attrs, "height", 0)
	rx, e5 := num(attrs, "rx", 0)
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
}

// ellipseShape builds a circle (r) or an ellipse (rx and ry) from four arcs'
// worth of two half turns.
func ellipseShape(tag string, attrs map[string]string) ([]svgCmd, error) {
	cx, e1 := num(attrs, "cx", 0)
	cy, e2 := num(attrs, "cy", 0)
	var rx, ry float64
	var e3, e4 error
	if tag == "circle" {
		rx, e3 = num(attrs, "r", 0)
		ry = rx
	} else {
		rx, e3 = num(attrs, "rx", 0)
		ry, e4 = num(attrs, "ry", 0)
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

func parseNumbers(text string) ([]float64, error) {
	var out []float64
	for _, f := range strings.FieldsFunc(text, func(char rune) bool { return char == ' ' || char == ',' || char == '\n' || char == '\t' }) {
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
func (p *pathScanner) flag() (float64, error) {
	p.skip()
	if p.i < len(p.s) && (p.s[p.i] == '0' || p.s[p.i] == '1') {
		value := float64(p.s[p.i] - '0')
		p.i++
		return value, nil
	}
	return 0, errors.New("an arc flag is 0 or 1")
}

// arcArguments reads rx ry rotation large-flag sweep-flag x y, the flags as
// 0 or 1.
func (p *pathScanner) arcArguments() ([]float64, error) {
	radii, err := p.numbers(3)
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
	end, err := p.numbers(2)
	if err != nil {
		return nil, err
	}
	return append(radii, large, sweep, end[0], end[1]), nil
}

func (p *pathScanner) numbers(count int) ([]float64, error) {
	out := make([]float64, count)
	for i := range out {
		v, err := p.number()
		if err != nil {
			return nil, fmt.Errorf("path data: a number is missing near %q", p.s[min(p.i, len(p.s)):])
		}
		out[i] = v
	}
	return out, nil
}

// pathState is what path data carries from one command to the next.
type pathState struct {
	cmds           []svgCmd
	cx, cy         float64 // current point
	sx, sy         float64 // start of the subpath, where Z returns to
	lastCx, lastCy float64 // last control point, for S and T
	prev           byte    // previous command letter, upper case
	rel            bool    // the running command is lower case
}

// point is an argument pair as an absolute point.
func (state *pathState) point(x, y float64) (float64, float64) {
	if state.rel {
		return state.cx + x, state.cy + y
	}
	return x, y
}

// moveTo appends an M or L command and makes its point current.
func (state *pathState) moveTo(op byte, x, y float64) {
	state.cmds = append(state.cmds, svgCmd{op, [6]float64{x, y}})
	state.cx, state.cy = x, y
	state.lastCx, state.lastCy = x, y
}

func (state *pathState) cubic(x1, y1, x2, y2, x, y float64) {
	state.cmds = append(state.cmds, svgCmd{'C', [6]float64{x1, y1, x2, y2, x, y}})
	state.lastCx, state.lastCy = x2, y2
	state.cx, state.cy = x, y
}

// quad appends the cubic that equals the quadratic through (qx, qy).
func (state *pathState) quad(qx, qy, x, y float64) {
	x0, y0 := state.cx, state.cy
	state.cubic(x0+2.0/3*(qx-x0), y0+2.0/3*(qy-y0), x+2.0/3*(qx-x), y+2.0/3*(qy-y), x, y)
	state.lastCx, state.lastCy = qx, qy
}

// reflected is the last control point mirrored through the current point
// when the previous command is one of kinds, else the current point.
func (state *pathState) reflected(kinds string) (float64, float64) {
	if strings.IndexByte(kinds, state.prev) < 0 {
		return state.cx, state.cy
	}
	return 2*state.cx - state.lastCx, 2*state.cy - state.lastCy
}

// pathCommand is one command letter: how many numbers it takes (read
// replaces the plain number reader where flags are mixed in) and what it
// does with them.
type pathCommand struct {
	args int
	read func(scanner *pathScanner) ([]float64, error)
	run  func(state *pathState, args []float64)
}

// pathCommands is keyed by the upper case letter.
var pathCommands = map[byte]pathCommand{
	'M': {2, nil, func(state *pathState, args []float64) {
		x, y := state.point(args[0], args[1])
		state.moveTo('M', x, y)
		state.sx, state.sy = x, y
	}},
	'L': {2, nil, func(state *pathState, args []float64) {
		x, y := state.point(args[0], args[1])
		state.moveTo('L', x, y)
	}},
	'H': {1, nil, func(state *pathState, args []float64) {
		x, _ := state.point(args[0], 0)
		state.moveTo('L', x, state.cy)
	}},
	'V': {1, nil, func(state *pathState, args []float64) {
		_, y := state.point(0, args[0])
		state.moveTo('L', state.cx, y)
	}},
	'C': {6, nil, func(state *pathState, args []float64) {
		x1, y1 := state.point(args[0], args[1])
		x2, y2 := state.point(args[2], args[3])
		x, y := state.point(args[4], args[5])
		state.cubic(x1, y1, x2, y2, x, y)
	}},
	'S': {4, nil, func(state *pathState, args []float64) {
		x1, y1 := state.reflected("CS")
		x2, y2 := state.point(args[0], args[1])
		x, y := state.point(args[2], args[3])
		state.cubic(x1, y1, x2, y2, x, y)
	}},
	'Q': {4, nil, func(state *pathState, args []float64) {
		qx, qy := state.point(args[0], args[1])
		x, y := state.point(args[2], args[3])
		state.quad(qx, qy, x, y)
	}},
	'T': {2, nil, func(state *pathState, args []float64) {
		qx, qy := state.reflected("QT")
		x, y := state.point(args[0], args[1])
		state.quad(qx, qy, x, y)
	}},
	'A': {7, (*pathScanner).arcArguments, func(state *pathState, args []float64) {
		x, y := state.point(args[5], args[6])
		for _, segment := range arcToCubics(state.cx, state.cy, args[0], args[1], args[2], args[3] == 1, args[4] == 1, x, y) {
			state.cmds = append(state.cmds, svgCmd{'C', segment})
		}
		state.cx, state.cy = x, y
		state.lastCx, state.lastCy = x, y
	}},
	'Z': {0, nil, func(state *pathState, _ []float64) {
		state.cmds = append(state.cmds, svgCmd{op: 'Z'})
		state.cx, state.cy = state.sx, state.sy
	}},
}

// run reads and runs one command letter and its repeated argument groups.
func (state *pathState) run(scanner *pathScanner, letter byte) error {
	state.rel = letter >= 'a'
	name := letter &^ 0x20
	for {
		command := pathCommands[name]
		read := command.read
		if read == nil {
			read = func(scanner *pathScanner) ([]float64, error) { return scanner.numbers(command.args) }
		}
		args, err := read(scanner)
		if err != nil {
			return err
		}
		command.run(state, args)
		if name == 'M' { // further pairs after M are implicit L
			name = 'L'
		}
		state.prev = name
		if command.args == 0 || !scanner.more() {
			return nil
		}
	}
}

// parsePathData turns path data into absolute M, L, C and Z commands.
func parsePathData(text string) ([]svgCmd, error) {
	scanner := &pathScanner{s: text}
	state := &pathState{}
	for {
		letter, ok := scanner.command()
		if !ok {
			break
		}
		if err := state.run(scanner, letter); err != nil {
			return nil, err
		}
	}
	scanner.skip()
	if scanner.i < len(scanner.s) {
		return nil, fmt.Errorf("path data: unexpected %q", scanner.s[scanner.i:min(scanner.i+6, len(scanner.s))])
	}
	if len(state.cmds) == 0 || state.cmds[0].op != 'M' {
		return nil, errors.New("path data must start with M")
	}
	return state.cmds, nil
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
	angle := func(ux, uy, vx, vy float64) float64 { return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy) }
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
	pt := func(value float64) (float64, float64, float64, float64) {
		cosA, sinA := math.Cos(value), math.Sin(value)
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
