package core

import (
	"embed"
	"errors"
	"image/color"
	"io/fs"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	names := make([]string, 0, len(icons.byName))
	for n := range icons.byName {
		names = append(names, n)
	}
	sort.Strings(names)
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

// IconSize is the natural size of an icon in logical pixels, and whether
// there is an icon of that name.
func IconSize(name string) (w, h float64, ok bool) {
	ic, ok := lookupIcon(name)
	if !ok {
		return 0, 0, false
	}
	return ic.w, ic.h, true
}

// IconHover is the color token an icon turns to under the pointer, as its
// file says, or "" for an icon that keeps its color.
func IconHover(name string) Token {
	ic, ok := lookupIcon(name)
	if !ok || ic.hover == "none" {
		return ""
	}
	return ic.hover
}

// DrawIcon draws the icon called name with its origin at x, y (logical
// pixels), scaled by k, in the token current for its one-color parts.
// The theme's IconStroke thickens or thins its lines. An unknown name
// draws nothing.
func (c *Ctx) DrawIcon(dst *ebiten.Image, name string, x, y, k float64, current Token) {
	ic, ok := lookupIcon(name)
	if !ok {
		return
	}
	s := k * c.Scale
	weight := 1.0 // the theme can thicken or thin every icon
	if c.Th.IconStroke > 0 {
		weight = c.Th.IconStroke
	}
	if ic.cacheable() {
		// one white mask per icon, size and weight; tinted when drawn, so
		// every use of the icon after the first is a single image draw
		mask := iconMask(ic, s, weight)
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(math.Round(x*c.Scale)-iconMaskMargin, math.Round(y*c.Scale)-iconMaskMargin)
		op.ColorScale.ScaleWithColor(c.Color(current))
		dst.DrawImage(mask, &op)
		return
	}
	renderShapes(dst, ic, x*c.Scale, y*c.Scale, s, weight, func(p svgPaint) color.RGBA { return c.Color(paintToken(p, current)) })
}

// renderShapes draws the icon's shapes with the icon's origin at dx, dy in
// device pixels and s device pixels per icon unit; colorOf picks the color
// of a paint.
func renderShapes(dst *ebiten.Image, ic *svgIcon, dx, dy, s, weight float64, colorOf func(svgPaint) color.RGBA) {
	for _, sh := range ic.shapes {
		var p vector.Path
		for _, cmd := range sh.cmds {
			pt := func(i int) (float32, float32) {
				return float32(dx + cmd.p[i]*s), float32(dy + cmd.p[i+1]*s)
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
			vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: float32(sh.width * s * weight), LineCap: sh.cap, LineJoin: sh.join}, &dp)
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
func iconMask(ic *svgIcon, s, weight float64) *ebiten.Image {
	key := maskKey{ic, s, weight}
	masks.Lock()
	defer masks.Unlock()
	if img, ok := masks.m[key]; ok {
		return img
	}
	if len(masks.m) >= maskLimit {
		masks.m = map[maskKey]*ebiten.Image{}
	}
	w, h := int(math.Ceil(ic.w*s))+2*iconMaskMargin, int(math.Ceil(ic.h*s))+2*iconMaskMargin
	img := ebiten.NewImage(w, h)
	white := color.RGBA{255, 255, 255, 255}
	renderShapes(img, ic, iconMaskMargin, iconMaskMargin, s, weight, func(svgPaint) color.RGBA { return white })
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

func paintToken(p svgPaint, current Token) Token {
	if p.token != "" {
		return p.token
	}
	return current
}
