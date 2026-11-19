package widget

import (
	"embed"
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

//go:embed icons/*.svg
var iconFiles embed.FS

var icons = struct {
	sync.Mutex
	byName map[string]*svgIcon
}{byName: map[string]*svgIcon{}}

func init() {
	entries, err := iconFiles.ReadDir("icons")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		src, err := iconFiles.ReadFile("icons/" + e.Name())
		if err != nil {
			panic(err)
		}
		name := strings.TrimSuffix(e.Name(), ".svg")
		ic, err := parseSVG(e.Name(), src)
		if err != nil {
			panic("built-in icon " + err.Error())
		}
		icons.byName[name] = ic
	}
}

// RegisterIcons adds the *.svg files at the root of fsys as icons, named
// by file name without the extension; a file replaces a built-in icon of
// the same name. Files that do not parse are skipped and reported in the
// error, which names each file and line.
func RegisterIcons(fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		src, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ic, err := parseSVG(e.Name(), src)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		icons.Lock()
		icons.byName[strings.TrimSuffix(e.Name(), ".svg")] = ic
		icons.Unlock()
	}
	return errors.Join(errs...)
}

// IconNames lists the available icons, sorted.
func IconNames() []string {
	icons.Lock()
	defer icons.Unlock()
	names := slices.Sorted(maps.Keys(icons.byName))
	return names
}

func lookupIcon(name string) (*svgIcon, bool) {
	icons.Lock()
	defer icons.Unlock()
	ic, ok := icons.byName[name]
	return ic, ok
}

// HasIcon reports whether an icon of that name exists.
func HasIcon(name string) bool { _, ok := lookupIcon(name); return ok }

// Icon is a vector icon by name, drawn in the element's color (Color, or
// muted by default) and in the icon's own hover color. It has the icon's
// natural size, centered; W and H scale it to fit. An unknown name panics
// and lists the names: it is a mistake in the program.
func Icon(name string, opts ...Option) *El {
	if _, ok := lookupIcon(name); !ok {
		panic(fmt.Sprintf("widget: no icon %q; icons: %s", name, strings.Join(IconNames(), " ")))
	}
	return Custom(iconB{name: name}, With(nil, opts)...)
}

// iconB draws a parsed SVG icon.
type iconB struct {
	Base
	name string
}

// Describe puts the icon's name in a Dump.
func (b iconB) Describe() string { return "icon=" + b.name }

func (b iconB) Measure(*El, *core.Ctx, int) (int, int) {
	ic, _ := lookupIcon(b.name)
	return int(math.Ceil(ic.w)), int(math.Ceil(ic.h))
}

// iconToken is the color token an icon is drawn in: its color, and under
// the pointer the element's HoverColor, else the icon's own hover token.
func iconToken(element *El, ic *svgIcon, hovered bool) core.Token {
	if !hovered {
		return element.ColorToken(core.Muted)
	}
	switch hover := element.HoverColorToken(); {
	case hover != "":
		return hover
	case ic.hover != "none" && ic.hover != "":
		return ic.hover
	}
	return element.ColorToken(core.Muted)
}

func (b iconB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	ic, ok := lookupIcon(b.name)
	if !ok {
		return
	}
	r := element.Rect()
	w, h := element.FixedSize()
	k := 1.0
	switch {
	case w > 0 && h > 0:
		k = math.Min(float64(w)/ic.w, float64(h)/ic.h)
	case w > 0:
		k = float64(w) / ic.w
	case h > 0:
		k = float64(h) / ic.h
	}
	ox := float64(r.X) + (float64(r.W)-ic.w*k)/2
	oy := float64(r.Y) + (float64(r.H)-ic.h*k)/2
	paintIcon(ctx, dst, ic, ox, oy, k, iconToken(element, ic, ctx.Hovered(r)))
}

// drawIconAt draws the named icon at its natural size, centered in rect, in
// the color token: for widgets that paint icons inside their own drawing
// (the checkbox's check, a code view's fold arrows).
func drawIconAt(ctx *core.Ctx, dst *ebiten.Image, name string, rect core.Rect, token core.Token) {
	ic, ok := lookupIcon(name)
	if !ok {
		return
	}
	paintIcon(ctx, dst, ic, float64(rect.X)+(float64(rect.W)-ic.w)/2, float64(rect.Y)+(float64(rect.H)-ic.h)/2, 1, token)
}

// paintIcon draws ic with its origin at ox, oy (logical pixels), scaled by
// k, in the color token cur.
func paintIcon(ctx *core.Ctx, dst *ebiten.Image, ic *svgIcon, ox, oy, scale float64, cur core.Token) {
	s := scale * ctx.Scale
	weight := 1.0 // the theme can thicken or thin every icon
	if ctx.Th.IconStroke > 0 {
		weight = ctx.Th.IconStroke
	}
	if ic.cacheable() {
		// one white mask per icon, size and weight; tinted when drawn, so
		// every use of the icon after the first is a single image draw
		mask := iconMask(ic, s, weight)
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(math.Round(ox*ctx.Scale)-iconMaskMargin, math.Round(oy*ctx.Scale)-iconMaskMargin)
		op.ColorScale.ScaleWithColor(ctx.Color(cur))
		dst.DrawImage(mask, &op)
		return
	}
	renderShapes(dst, ic, ox*ctx.Scale, oy*ctx.Scale, s, weight, func(paint svgPaint) color.RGBA { return ctx.Color(paintToken(paint, cur)) })
}

// renderShapes draws the icon's shapes with the icon's origin at dx, dy in
// device pixels and s device pixels per icon unit; colorOf picks the color
// of a paint.
func renderShapes(dst *ebiten.Image, ic *svgIcon, dx, dy, scale, weight float64, colorOf func(svgPaint) color.RGBA) {
	for _, sh := range ic.shapes {
		var p vector.Path
		for _, cmd := range sh.cmds {
			pt := func(index int) (float32, float32) {
				return float32(dx + cmd.p[index]*scale), float32(dy + cmd.p[index+1]*scale)
			}
			switch cmd.op {
			case 'M':
				p.MoveTo(pt(0))
			case 'L':
				p.LineTo(pt(0))
			case 'C':
				x1, y1 := pt(0)
				x2, y2 := pt(2)
				x3, y3 := pt(4)
				p.CubicTo(x1, y1, x2, y2, x3, y3)
			case 'Z':
				p.Close()
			}
		}
		if !sh.fill.none {
			var dp vector.DrawPathOptions
			dp.AntiAlias = true
			dp.ColorScale.ScaleWithColor(colorOf(sh.fill))
			vector.FillPath(dst, &p, &vector.FillOptions{}, &dp)
		}
		if !sh.line.none && sh.width > 0 {
			var dp vector.DrawPathOptions
			dp.AntiAlias = true
			dp.ColorScale.ScaleWithColor(colorOf(sh.line))
			vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: float32(sh.width * scale * weight), LineCap: sh.cap, LineJoin: sh.join}, &dp)
		}
	}
}

// iconMaskMargin is the transparent border around a cached icon, so antialiased
// edges and round caps are not cut off.
const iconMaskMargin = 2

type maskKey struct {
	icon      *svgIcon
	scale, wt float64
}

var masks = struct {
	sync.Mutex
	m map[maskKey]*ebiten.Image
}{m: map[maskKey]*ebiten.Image{}}

// maskLimit bounds the cache; when it is full the whole cache is dropped,
// which only costs the next frame a redraw of each icon.
const maskLimit = 512

// iconMask is the icon drawn in white at s device pixels per unit, with a
// margin, cached by icon, scale and stroke weight.
func iconMask(ic *svgIcon, scale, weight float64) *ebiten.Image {
	key := maskKey{ic, scale, weight}
	masks.Lock()
	defer masks.Unlock()
	if img, ok := masks.m[key]; ok {
		return img
	}
	if len(masks.m) >= maskLimit {
		masks.m = map[maskKey]*ebiten.Image{}
	}
	w, h := int(math.Ceil(ic.w*scale))+2*iconMaskMargin, int(math.Ceil(ic.h*scale))+2*iconMaskMargin
	img := ebiten.NewImage(w, h)
	white := color.RGBA{255, 255, 255, 255}
	renderShapes(img, ic, iconMaskMargin, iconMaskMargin, scale, weight, func(svgPaint) color.RGBA { return white })
	masks.m[key] = img
	return img
}

// cacheable reports whether the icon is drawn in one color, the element's:
// icons with token paints are drawn directly.
func (ic *svgIcon) cacheable() bool {
	for _, sh := range ic.shapes {
		if sh.fill.token != "" || sh.line.token != "" {
			return false
		}
	}
	return true
}

func paintToken(paint svgPaint, current core.Token) core.Token {
	if paint.token != "" {
		return paint.token
	}
	return current
}

// The named icons below are wrappers over Icon for the icons the toolkit
// ships (widget/icons/*.svg).

// CrossIcon is a small x, for delete buttons; it turns red under the pointer.
func CrossIcon(opts ...Option) *El { return Icon("cross", opts...) }

// TriangleIcon is a disclosure arrow: down when open, right when closed.
func TriangleIcon(open bool, opts ...Option) *El {
	if open {
		return Icon("triangle-down", opts...)
	}
	return Icon("triangle-right", opts...)
}

// GripIcon is the six-dot handle that rows are dragged by.
func GripIcon(opts ...Option) *El { return Icon("grip", opts...) }

// DotsIcon is the three-dot "more" glyph.
func DotsIcon(opts ...Option) *El { return Icon("dots", opts...) }

// GearIcon is a cogwheel, the settings button.
func GearIcon(opts ...Option) *El { return Icon("gear", opts...) }

// TrashIcon is a trash can, for delete buttons; it turns red under the pointer.
func TrashIcon(opts ...Option) *El { return Icon("trash", opts...) }

// CopyIcon is two overlapping sheets.
func CopyIcon(opts ...Option) *El { return Icon("copy", opts...) }

// FolderIcon is an outlined folder, for choosing a file.
func FolderIcon(opts ...Option) *El { return Icon("folder", opts...) }

// SearchIcon is a magnifying glass.
func SearchIcon(opts ...Option) *El { return Icon("search", opts...) }

// PlusIcon is a plus sign.
func PlusIcon(opts ...Option) *El { return Icon("plus", opts...) }

// PromptIcon is a terminal prompt, ">_".
func PromptIcon(opts ...Option) *El { return Icon("prompt", opts...) }

// ChevronUpIcon is a small arrow up, for stepping back through a list.
func ChevronUpIcon(opts ...Option) *El { return Icon("chevron-up", opts...) }

// ChevronDownIcon is a small arrow down, for stepping forward through a list.
func ChevronDownIcon(opts ...Option) *El { return Icon("chevron-down", opts...) }
