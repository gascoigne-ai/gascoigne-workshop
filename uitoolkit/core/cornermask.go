package core

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// maskLimit bounds the cached corner masks: a UI uses a handful of radii, so
// the limit is reached only by a runaway (a radius animated every frame).
const maskLimit = 64

// cornerKey names one cached mask.
type cornerKey struct{ radius int }

// cornerMasks caches white corner masks by radius, in device pixels. It is
// owned by one Ctx and used from the draw goroutine only. The maker is a
// field so a test can count builds without a GPU.
type cornerMasks struct {
	images map[cornerKey]*ebiten.Image
	make   func(radius int) *ebiten.Image
}

// get returns the mask for radius, building it on a miss. When the cache is
// full it drops every mask first: a simple bound beats tracking use.
func (masks *cornerMasks) get(radius int) *ebiten.Image {
	key := cornerKey{radius}
	if img, ok := masks.images[key]; ok {
		return img
	}
	if masks.images == nil || len(masks.images) >= maskLimit {
		masks.images = make(map[cornerKey]*ebiten.Image)
	}
	img := masks.make(radius)
	masks.images[key] = img
	return img
}

// reset drops every mask, as a change of scale or theme does.
func (masks *cornerMasks) reset() { masks.images = nil }

// buildCornerMask draws a white square of side 2*radius with all corners
// rounded by radius, anti-aliased as a fill of that shape is, and one solid
// column to its right (the solid texel of sliceQuads). Quarters of it are the corners of a
// fill of any size.
func buildCornerMask(radius int) *ebiten.Image {
	size := float32(2 * radius)
	img := ebiten.NewImage(2*radius+1, 2*radius)
	verts, idx := cornersPath(0, 0, size, size, float32(radius), CornersAll).AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range verts {
		verts[i].ColorR, verts[i].ColorG, verts[i].ColorB, verts[i].ColorA = 1, 1, 1, 1
		verts[i].SrcX, verts[i].SrcY = 0.5, 0.5
	}
	img.DrawTriangles(verts, idx, gradientSrc, &ebiten.DrawTrianglesOptions{AntiAlias: true})
	vector.FillRect(img, size, 0, 1, size, color.White, false)
	return img
}

// frect is a rectangle in float device pixels: left, top, right, bottom.
type frect struct{ x0, y0, x1, y1 float32 }

func (rect frect) area() float32 { return (rect.x1 - rect.x0) * (rect.y1 - rect.y0) }

// quad is one piece of a rounded fill: where it lands and the part of the
// mask it samples.
type quad struct{ dst, src frect }

// sliceQuads cuts the rectangle x0,y0 to x1,y1 into the pieces of a rounded
// fill: four squares of side r at the corners (a quarter of the mask disc
// for a rounded corner, a solid texel for the others), four edge strips and
// the centre. r is clamped to half the shorter side; empty pieces are left out.
func sliceQuads(x0, y0, x1, y1, r float32, corners Corners) []quad {
	r = min(r, (x1-x0)/2, (y1-y0)/2)
	// The mask is the disc, 2r square, and a solid column right of it.
	solid := frect{2*r + 0.5, 0.5, 2*r + 0.5, 0.5}
	piece := func(corner Corners, disc frect) frect {
		if corners&corner != 0 {
			return disc
		}
		return solid
	}
	xs := [4]float32{x0, x0 + r, x1 - r, x1}
	ys := [4]float32{y0, y0 + r, y1 - r, y1}
	cell := func(col, row int) frect { return frect{xs[col], ys[row], xs[col+1], ys[row+1]} }
	sources := [3][3]frect{
		{piece(CornerTL, frect{0, 0, r, r}), solid, piece(CornerTR, frect{r, 0, 2 * r, r})},
		{solid, solid, solid},
		{piece(CornerBL, frect{0, r, r, 2 * r}), solid, piece(CornerBR, frect{r, r, 2 * r, 2 * r})},
	}
	// Corners first (TL, TR, BR, BL), then the edges and the centre.
	order := [9][2]int{{0, 0}, {2, 0}, {2, 2}, {0, 2}, {1, 0}, {1, 2}, {0, 1}, {2, 1}, {1, 1}}
	quads := make([]quad, 0, len(order))
	for _, at := range order {
		dst := cell(at[0], at[1])
		if dst.x1 > dst.x0 && dst.y1 > dst.y0 {
			quads = append(quads, quad{dst, sources[at[1]][at[0]]})
		}
	}
	return quads
}

// gradientAt is the color of a vertical gradient from top (at y0) to bottom
// (at y1) at height y, as the color scales a vertex carries.
func gradientAt(top, bottom color.RGBA, y0, y1, y float32) (red, green, blue, alpha float32) {
	t := min(max((y-y0)/max(y1-y0, 1), 0), 1)
	mix := func(value, target uint8) float32 {
		return (float32(value) + (float32(target)-float32(value))*t) / 255
	}
	return mix(top.R, bottom.R), mix(top.G, bottom.G), mix(top.B, bottom.B), mix(top.A, bottom.A)
}
