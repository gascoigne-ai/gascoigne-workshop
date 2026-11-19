package widget

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/vector"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// RasterizeSVG draws a small SVG picture into a size by size image without a
// window or a GPU, for the program's own icon, which is needed before any
// window exists. It reads the icon subset (see parseSVG) with two additions:
// a fill is a color (#rgb or #rrggbb) or a linearGradient from <defs>
// (userSpaceOnUse, with stops), and a shape is only filled: stroke must be
// none. A fill uses the non-zero rule, so a hole is a subpath wound the
// other way. Shapes are painted in order, over each other.
func RasterizeSVG(name string, src []byte, size int) (*image.RGBA, error) {
	picture, err := parseSVGAs(name, src, picturePaints)
	if err != nil {
		return nil, err
	}
	out := image.NewRGBA(image.Rect(0, 0, size, size))
	scale := float64(size) / math.Max(picture.w, picture.h)
	for _, shape := range picture.shapes {
		if !shape.line.none {
			return nil, fmt.Errorf("%s: a picture is filled, not stroked: set stroke to none", name)
		}
		source, err := fillSource(picture, shape.fill, scale)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		var raster vector.Rasterizer
		raster.Reset(size, size)
		raster.DrawOp = draw.Over
		tracePath(&raster, shape.cmds, scale)
		raster.Draw(out, out.Bounds(), source, image.Point{})
	}
	return out, nil
}

// tracePath adds the path commands, scaled, to the rasterizer.
func tracePath(raster *vector.Rasterizer, cmds []svgCmd, scale float64) {
	point := func(cmd svgCmd, index int) (float32, float32) {
		return float32(cmd.p[index] * scale), float32(cmd.p[index+1] * scale)
	}
	for _, cmd := range cmds {
		switch cmd.op {
		case 'M':
			raster.MoveTo(point(cmd, 0))
		case 'L':
			raster.LineTo(point(cmd, 0))
		case 'C':
			x1, y1 := point(cmd, 0)
			x2, y2 := point(cmd, 2)
			x3, y3 := point(cmd, 4)
			raster.CubeTo(x1, y1, x2, y2, x3, y3)
		case 'Z':
			raster.ClosePath()
		}
	}
}

// fillSource is what a fill paints: one color or a gradient.
func fillSource(picture *svgIcon, paint svgPaint, scale float64) (image.Image, error) {
	switch {
	case paint.solid:
		return image.NewUniform(paint.rgba), nil
	case paint.gradient != "":
		gradient, ok := picture.gradients[paint.gradient]
		if !ok || len(gradient.stops) == 0 {
			return nil, fmt.Errorf("no gradient %q with stops", paint.gradient)
		}
		return gradientImage{gradient: gradient, scale: scale}, nil
	}
	return nil, errors.New("a fill is a color (#rrggbb) or a gradient (url(#id))")
}

// gradientImage is an endless image whose color follows a linear gradient,
// in pixels: the gradient's own units times scale.
type gradientImage struct {
	gradient *svgGradient
	scale    float64
}

func (gradientImage) ColorModel() color.Model { return color.RGBAModel }
func (gradientImage) Bounds() image.Rectangle {
	return image.Rect(math.MinInt32, math.MinInt32, math.MaxInt32, math.MaxInt32)
}

func (g gradientImage) At(posX, posY int) color.Color {
	grad := g.gradient
	dx, dy := grad.x2-grad.x1, grad.y2-grad.y1
	px, py := (float64(posX)+0.5)/g.scale-grad.x1, (float64(posY)+0.5)/g.scale-grad.y1
	along := 0.0
	if length := dx*dx + dy*dy; length > 0 {
		along = (px*dx + py*dy) / length
	}
	return stopColor(grad.stops, math.Min(math.Max(along, 0), 1))
}

// stopColor is the color at position along the stops, mixed between the two
// stops around it.
func stopColor(stops []svgStop, along float64) color.RGBA {
	if along <= stops[0].offset {
		return stops[0].rgba
	}
	for i := 1; i < len(stops); i++ {
		if along > stops[i].offset {
			continue
		}
		from, to := stops[i-1], stops[i]
		ratio := (along - from.offset) / math.Max(to.offset-from.offset, 1e-9)
		mixed := core.Mix(from.rgba, to.rgba, ratio)
		mixed.A = 0xff
		return mixed
	}
	return stops[len(stops)-1].rgba
}

func parseGradient(attrs map[string]string) (*svgGradient, error) {
	if attrs["id"] == "" {
		return nil, errors.New("a linearGradient needs an id")
	}
	if units := attrs["gradientUnits"]; units != "userSpaceOnUse" {
		return nil, fmt.Errorf(`a linearGradient needs gradientUnits="userSpaceOnUse", not %q`, units)
	}
	x1, e1 := num(attrs, "x1", 0)
	y1, e2 := num(attrs, "y1", 0)
	x2, e3 := num(attrs, "x2", 0)
	y2, e4 := num(attrs, "y2", 0)
	if err := errors.Join(e1, e2, e3, e4); err != nil {
		return nil, err
	}
	return &svgGradient{x1: x1, y1: y1, x2: x2, y2: y2}, nil
}

func parseStop(attrs map[string]string) (svgStop, error) {
	offset, err := num(attrs, "offset", 0)
	if err != nil {
		return svgStop{}, err
	}
	rgba, err := core.ParseHexStrict(attrs["stop-color"])
	if err != nil {
		return svgStop{}, fmt.Errorf("stop-color: %w", err)
	}
	return svgStop{offset: offset, rgba: rgba}, nil
}
