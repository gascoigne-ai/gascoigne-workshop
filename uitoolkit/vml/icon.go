package vml

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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

// Icon is a vector icon by name, drawn in the element's color (Color, or
// muted by default) and in the icon's own hover color. It has the icon's
// natural size, centered; W and H scale it to fit. An unknown name panics
// and lists the names: it is a mistake in the program.
func Icon(name string, opts ...Option) *El {
	ic, ok := lookupIcon(name)
	if !ok {
		panic(fmt.Sprintf("vml: no icon %q; icons: %s", name, strings.Join(IconNames(), " ")))
	}
	e := newEl(iconB{}, with(nil, opts))
	e.iconName = name
	_ = ic
	return e
}

// iconB draws a parsed SVG icon.
type iconB struct{ plain }

func (iconB) measure(e *El, _ *core.Ctx, _ int) (int, int) {
	ic, _ := lookupIcon(e.iconName)
	return int(math.Ceil(ic.w)), int(math.Ceil(ic.h))
}

func (iconB) draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	ic, ok := lookupIcon(e.iconName)
	if !ok {
		return
	}
	r := e.rect
	k := 1.0
	switch {
	case e.w > 0 && e.h > 0:
		k = math.Min(float64(e.w)/ic.w, float64(e.h)/ic.h)
	case e.w > 0:
		k = float64(e.w) / ic.w
	case e.h > 0:
		k = float64(e.h) / ic.h
	}
	ox := float64(r.X) + (float64(r.W)-ic.w*k)/2
	oy := float64(r.Y) + (float64(r.H)-ic.h*k)/2
	cur := e.color(c, core.Muted)
	if c.Hovered(r) && ic.hover != "none" && ic.hover != "" {
		cur = ic.hover
	}
	s := k * c.Scale
	weight := 1.0 // the theme can thicken or thin every icon
	if c.Th.IconStroke > 0 {
		weight = c.Th.IconStroke
	}
	dx, dy := ox*c.Scale, oy*c.Scale
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
			dp.ColorScale.ScaleWithColor(c.Color(paintToken(sh.fill, cur)))
			vector.FillPath(dst, &p, &vector.FillOptions{}, &dp)
		}
		if !sh.line.none && sh.width > 0 {
			var dp vector.DrawPathOptions
			dp.AntiAlias = true
			dp.ColorScale.ScaleWithColor(c.Color(paintToken(sh.line, cur)))
			vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: float32(sh.width * s * weight), LineCap: sh.cap, LineJoin: sh.join}, &dp)
		}
	}
}

func paintToken(p svgPaint, current core.Token) core.Token {
	if p.token != "" {
		return p.token
	}
	return current
}
