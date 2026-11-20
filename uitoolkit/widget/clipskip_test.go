package widget

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// drawCounter is a leaf that counts the frames it is drawn in.
type drawCounter struct {
	Base
	draws int
}

func (c *drawCounter) Measure(*El, *core.Ctx, int) (int, int) { return 50, 20 }
func (c *drawCounter) Draw(*El, *core.Ctx, *ebiten.Image)     { c.draws++ }

func countedRows(count int) ([]*drawCounter, []Item) {
	counters := make([]*drawCounter, count)
	items := make([]Item, count)
	for i := range counters {
		counters[i] = &drawCounter{}
		items[i] = Custom(counters[i])
	}
	return counters, items
}

// boxed puts a scroll in a 100x100 box inside a bigger window, so the clip
// is smaller than the screen.
func boxed(scroll *El) *El {
	return Row(AlignItems(AlignStart), Col(W(100), H(100), scroll))
}

func totalDraws(counters []*drawCounter) (sum int) {
	for _, counter := range counters {
		sum += counter.draws
	}
	return sum
}

func TestAChildOutsideAClipIsNotDrawn(t *testing.T) {
	counters, rows := countedRows(50)
	root := boxed(Scroll(&ScrollState{}, rows...))
	drawn(t, testCtx(t), root, 200, 200)
	got := totalDraws(counters)
	if got < 5 || got > 6 {
		t.Errorf("%d of 50 rows drawn in a 100px box of 20px rows, want the 5 that show (6 at most)", got)
	}
	if counters[49].draws != 0 {
		t.Error("a row far below the box was drawn")
	}
}

func TestAScrolledListDrawsTheRowsInView(t *testing.T) {
	counters, rows := countedRows(50)
	root := boxed(Scroll(&ScrollState{Y: 400}, rows...))
	drawn(t, testCtx(t), root, 200, 200)
	for i, counter := range counters {
		visible := i >= 20 && i < 25 // rows 20..24 span y 400..500
		if visible && counter.draws != 1 {
			t.Errorf("row %d is in view and was drawn %d times", i, counter.draws)
		}
		if !visible && i != 19 && i != 25 && counter.draws != 0 {
			t.Errorf("row %d is out of view and was drawn", i)
		}
	}
}

// A shadow is shifted down and right, so a child just above the clip can
// still show its shadow inside: scrolled so that the first row ends 2px above
// the box, the shadow (4px down) reaches in.
func TestShadowStillDrawsWhenItsRectReachesTheClip(t *testing.T) {
	tests := []struct {
		name    string
		scrollY int
		want    int
	}{
		{"row ends 2px above the box, its shadow reaches in", 22, 1},
		{"row ends 60px above the box", 80, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := &drawCounter{}
			_, others := countedRows(20)
			rows := append([]Item{Custom(first, Shadow())}, others...)
			drawn(t, testCtx(t), boxed(Scroll(&ScrollState{Y: test.scrollY}, rows...)), 200, 200)
			if first.draws != test.want {
				t.Errorf("first row drawn %d times, want %d", first.draws, test.want)
			}
		})
	}
}

// Without a clip the box is the screen and nothing is skipped: every child
// is drawn, whatever its place.
func TestNothingIsSkippedWithoutAClip(t *testing.T) {
	counters, rows := countedRows(8)
	drawn(t, testCtx(t), Col(rows...), 200, 400)
	if got := totalDraws(counters); got != 8 {
		t.Errorf("%d of 8 children drawn in a box with no clip, want all", got)
	}
}
