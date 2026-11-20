package core

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// gradientAt keeps what fillGradient computed per vertex before it was
// extracted: a clamped linear mix of the two ends.
func TestGradientAt(t *testing.T) {
	top, bottom := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 51}
	tests := []struct {
		name string
		y    float32
		want [4]float32
	}{
		{"top", 10, [4]float32{1, 0, 0, 1}},
		{"bottom", 30, [4]float32{0, 1, 0, 0.2}},
		{"middle", 20, [4]float32{0.5, 0.5, 0, 0.6}},
		{"above clamps to top", 0, [4]float32{1, 0, 0, 1}},
		{"below clamps to bottom", 99, [4]float32{0, 1, 0, 0.2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, g, b, a := gradientAt(top, bottom, 10, 30, tc.y)
			got := [4]float32{r, g, b, a}
			for i := range got {
				if d := got[i] - tc.want[i]; d > 0.002 || d < -0.002 {
					t.Fatalf("gradientAt = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestGradientAtFlatSpanDoesNotDivideByZero(t *testing.T) {
	r, _, _, _ := gradientAt(color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 0, 255}, 5, 5, 5)
	if r != 1 {
		t.Fatalf("red = %v, want 1", r)
	}
}

func TestSliceQuads(t *testing.T) {
	tests := []struct {
		name           string
		x0, y0, x1, y1 float32
		r              float32
		corners        Corners
		count          int
		discCorners    int // quads that sample the disc, not the solid texel
	}{
		{"all corners", 0, 0, 20, 12, 4, CornersAll, 9, 4},
		{"top corners only", 0, 0, 20, 12, 4, CornersTop, 9, 2},
		{"bottom-left only", 0, 0, 20, 12, 4, CornerBL, 9, 1},
		{"no corners", 0, 0, 20, 12, 4, 0, 9, 0},
		{"width below 2r clamps the radius", 0, 0, 6, 12, 4, CornersAll, 6, 4},
		{"square clamps to a disc", 0, 0, 8, 8, 4, CornersAll, 4, 4},
		{"fractional edges", 2.5, 1.25, 22.5, 13.75, 4, CornersAll, 9, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			quads := sliceQuads(tc.x0, tc.y0, tc.x1, tc.y1, tc.r, tc.corners)
			if len(quads) != tc.count {
				t.Fatalf("got %d quads, want %d: %v", len(quads), tc.count, quads)
			}
			assertCoversExactly(t, quads, frect{tc.x0, tc.y0, tc.x1, tc.y1})
			discs := 0
			for _, q := range quads {
				if q.src.x1 > q.src.x0 {
					discs++
				}
			}
			if discs != tc.discCorners {
				t.Fatalf("%d quads sample the disc, want %d", discs, tc.discCorners)
			}
		})
	}
}

// assertCoversExactly checks the quads tile rect: their areas add up to its
// area and no two overlap, so there is no gap and no double blend.
func assertCoversExactly(t *testing.T, quads []quad, rect frect) {
	t.Helper()
	var sum float32
	for i, q := range quads {
		sum += q.dst.area()
		if q.dst.x0 < rect.x0 || q.dst.y0 < rect.y0 || q.dst.x1 > rect.x1 || q.dst.y1 > rect.y1 {
			t.Fatalf("quad %v leaves %v", q.dst, rect)
		}
		for _, other := range quads[i+1:] {
			if q.dst.x0 < other.dst.x1 && other.dst.x0 < q.dst.x1 && q.dst.y0 < other.dst.y1 && other.dst.y0 < q.dst.y1 {
				t.Fatalf("quads overlap: %v and %v", q.dst, other.dst)
			}
		}
	}
	if d := sum - rect.area(); d > 0.001 || d < -0.001 {
		t.Fatalf("quad areas add up to %v, want %v", sum, rect.area())
	}
}

func TestSliceQuadsDiscQuarters(t *testing.T) {
	quads := sliceQuads(0, 0, 20, 12, 4, CornersAll)
	// The corner squares come first, in the order TL, TR, BR, BL; each samples
	// its own quarter of the 8x8 disc.
	want := []frect{{0, 0, 4, 4}, {4, 0, 8, 4}, {4, 4, 8, 8}, {0, 4, 4, 8}}
	for i, src := range want {
		if len(quads) < 4 || quads[i].src != src {
			t.Fatalf("corner %d samples %v, want %v", i, quads, src)
		}
	}
}

func TestCornerMasksBuildOncePerRadius(t *testing.T) {
	built := 0
	masks := &cornerMasks{make: func(int) *ebiten.Image { built++; return nil }}
	for _, radius := range []int{4, 4, 6, 4, 6} {
		masks.get(radius)
	}
	if built != 2 {
		t.Fatalf("built %d masks, want 2", built)
	}
}

func TestCornerMasksAreBounded(t *testing.T) {
	built := 0
	masks := &cornerMasks{make: func(int) *ebiten.Image { built++; return nil }}
	for radius := 1; radius <= maskLimit; radius++ {
		masks.get(radius)
	}
	if len(masks.images) != maskLimit {
		t.Fatalf("holds %d masks, want %d", len(masks.images), maskLimit)
	}
	masks.get(maskLimit + 1)
	if len(masks.images) != 1 {
		t.Fatalf("full cache holds %d masks after one more, want 1 (dropped all)", len(masks.images))
	}
	masks.get(1) // dropped, so built again
	if built != maskLimit+2 {
		t.Fatalf("built %d masks, want %d", built, maskLimit+2)
	}
}

func TestCornerMasksResetDropsEverything(t *testing.T) {
	built := 0
	masks := &cornerMasks{make: func(int) *ebiten.Image { built++; return nil }}
	masks.get(4)
	masks.reset()
	masks.get(4)
	if built != 2 {
		t.Fatalf("built %d masks, want 2", built)
	}
}

func TestScaleAndThemeChangesDropTheMasks(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	built := 0
	ctx.maskMaker = func(int) *ebiten.Image { built++; return nil }
	ctx.cornerMask(4)
	if err := ctx.SetScale(1.5); err != nil {
		t.Fatal(err)
	}
	ctx.cornerMask(4)
	if err := ctx.SetTheme(testTheme()); err != nil {
		t.Fatal(err)
	}
	ctx.cornerMask(4)
	if built != 3 {
		t.Fatalf("built %d masks, want 3 (one per scale or theme)", built)
	}
}

func TestRingQuads(t *testing.T) {
	tests := []struct {
		name           string
		x0, y0, x1, y1 float32
		r, width       float32
		count          int
	}{
		{"corners and four edges", 0, 0, 20, 12, 4, 1, 8},
		{"wider line", 0, 0, 20, 12, 4, 2, 8},
		{"line as wide as the radius", 0, 0, 20, 12, 4, 4, 8},
		{"width below 2r drops the empty edge", 0, 0, 8, 12, 4, 1, 6},
		{"square of 2r is four corners", 0, 0, 8, 8, 4, 1, 4},
		{"fractional origin", 3, 5, 23, 17, 4, 1, 8},
		{"one pixel radius", 0, 0, 2, 2, 1, 1, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			quads := ringQuads(tc.x0, tc.y0, tc.x1, tc.y1, tc.r, tc.width)
			if len(quads) != tc.count {
				t.Fatalf("got %d quads, want %d: %v", len(quads), tc.count, quads)
			}
			assertRingCoverage(t, quads, frect{tc.x0, tc.y0, tc.x1, tc.y1}, tc.r, tc.width)
		})
	}
}

// assertRingCoverage checks that the quads never overlap, stay inside rect and
// leave the inner rect (rect minus the line width on each side) uncovered
// except for the corner squares, which are covered whole (the mask is clear
// there).
func assertRingCoverage(t *testing.T, quads []quad, rect frect, r, width float32) {
	t.Helper()
	inner := frect{rect.x0 + width, rect.y0 + width, rect.x1 - width, rect.y1 - width}
	r = min(r, (rect.x1-rect.x0)/2, (rect.y1-rect.y0)/2)
	for y := rect.y0; y < rect.y1; y++ {
		for x := rect.x0; x < rect.x1; x++ {
			covers := 0
			for _, q := range quads {
				if x >= q.dst.x0 && x < q.dst.x1 && y >= q.dst.y0 && y < q.dst.y1 {
					covers++
				}
			}
			inInner := x >= inner.x0 && x < inner.x1 && y >= inner.y0 && y < inner.y1
			inCorner := (x < rect.x0+r || x >= rect.x1-r) && (y < rect.y0+r || y >= rect.y1-r)
			want := 1
			if inInner && !inCorner {
				want = 0
			}
			if covers != want {
				t.Fatalf("pixel %v,%v covered %d times, want %d (quads %v)", x, y, covers, want, quads)
			}
		}
	}
}

func TestRingQuadsSamplesTheMask(t *testing.T) {
	quads := ringQuads(0, 0, 20, 12, 4, 2)
	// Corners first (TL, TR, BR, BL), each its own quarter of the 8x8 ring.
	want := []frect{{0, 0, 4, 4}, {4, 0, 8, 4}, {4, 4, 8, 8}, {0, 4, 4, 8}}
	for i, src := range want {
		if len(quads) < 8 || quads[i].src != src {
			t.Fatalf("corner %d samples %v, want %v", i, quads, src)
		}
	}
	// The edges are the line width thick and sample the solid texel.
	for _, q := range quads[4:] {
		if q.src.x1 != q.src.x0 || q.src.y1 != q.src.y0 {
			t.Fatalf("edge %v samples more than the solid texel", q)
		}
	}
}

func TestRingMasksAreKeyedByWidth(t *testing.T) {
	built := 0
	masks := &cornerMasks{makeRing: func(int, int) *ebiten.Image { built++; return nil }, make: func(int) *ebiten.Image { built++; return nil }}
	masks.getRing(4, 1)
	masks.getRing(4, 1)
	masks.getRing(4, 2)
	masks.get(4)
	if built != 3 {
		t.Fatalf("built %d masks, want 3 (ring w1, ring w2, fill)", built)
	}
}

func TestRoundedOutlineIsOneRingDraw(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Stats = &DrawStats{}
	img := ebiten.NewImage(20, 20)
	ctx.OutlineStyled(img, Rect{X: 2, Y: 2, W: 16, H: 16}, color.RGBA{255, 0, 0, 255}, 4)
	if ctx.Stats.RingFills != 1 {
		t.Fatalf("RingFills = %d, want 1", ctx.Stats.RingFills)
	}
}

// A radius that rounds below one device pixel is a square outline, as before.
func TestOutlineStyledWithTinyRadiusIsTheSquareOutline(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Stats = &DrawStats{}
	img := ebiten.NewImage(20, 20)
	ctx.OutlineStyled(img, Rect{X: 2, Y: 2, W: 16, H: 16}, color.RGBA{255, 0, 0, 255}, 0.5)
	if ctx.Stats.RingFills != 0 {
		t.Fatalf("RingFills = %d, want 0", ctx.Stats.RingFills)
	}
}
