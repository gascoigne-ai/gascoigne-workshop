package vml

import (
	"math"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

func sum(w []float64) (s float64) {
	for _, v := range w {
		s += v
	}
	return
}

func TestResizePair(t *testing.T) {
	tests := []struct {
		name                    string
		weights                 []float64
		i, leftPx, totalPx, min int
		want                    []float64
	}{
		{"split evenly stays", []float64{50, 50}, 0, 100, 200, 56, []float64{50, 50}},
		{"drag right", []float64{50, 50}, 0, 140, 200, 56, []float64{70, 30}},
		{"drag left", []float64{50, 50}, 0, 50, 200, 56, []float64{(50 + 50) * 56.0 / 200, 100 - 100*56.0/200}},
		{"clamped at the minimum on the left", []float64{50, 50}, 0, 0, 200, 56, []float64{28, 72}},
		{"clamped at the minimum on the right", []float64{50, 50}, 0, 999, 200, 56, []float64{72, 28}},
		{"the middle pair only", []float64{20, 30, 50}, 1, 100, 200, 56, []float64{20, 40, 40}},
		{"a pair too small for two minimums splits evenly", []float64{30, 70}, 0, 10, 100, 56, []float64{50, 50}},
		{"unequal pair keeps its combined weight", []float64{28, 40, 32}, 0, 200, 400, 56, []float64{34, 34, 32}},
	}
	for _, tt := range tests {
		orig := append([]float64(nil), tt.weights...)
		got := ResizePair(tt.weights, tt.i, tt.leftPx, tt.totalPx, tt.min)
		if len(got) != len(orig) {
			t.Fatalf("%s: %v", tt.name, got)
		}
		for k := range got {
			if math.Abs(got[k]-tt.want[k]) > 0.001 {
				t.Errorf("%s: %v, want %v", tt.name, got, tt.want)
				break
			}
		}
		if math.Abs(sum(got)-sum(orig)) > 0.001 {
			t.Errorf("%s: total weight changed from %v to %v", tt.name, sum(orig), sum(got))
		}
		if !reflect.DeepEqual(tt.weights, orig) {
			t.Errorf("%s: the input was changed", tt.name)
		}
	}
}

// A 400px editable two-column table: 36px action cells, then 164px columns.
// The divider between the columns is at x=200, the header row at y=0..28.
// The view is rebuilt after every change, as the app does each frame.
type tableRig struct {
	t        *testing.T
	c        *widget.Ctx
	widths   *ColumnWidths
	editable bool
	root     *El
}

func newRig(t *testing.T, widths *ColumnWidths, editable bool) *tableRig {
	r := &tableRig{t: t, c: testCtx(t), widths: widths, editable: editable}
	r.rebuild()
	return r
}

func (r *tableRig) rebuild() {
	spec := editableSpec()
	spec.Editable = r.editable
	spec.Widths = r.widths
	r.root = Col(Table(spec))
	Layout(r.c, r.root, 400, 400)
}

func (r *tableRig) input(in widget.Input) {
	r.c.In = in
	Update(r.c, r.root)
	r.rebuild()
}

// columnWidths reads the layout: the first two header column cells.
func (r *tableRig) columnWidths() (first, second int) {
	header := r.root.kids[0].kids[0].kids[0] // table, header Stack, header Row
	off := 0
	if r.editable {
		off = 1
	}
	return header.kids[off].rect.W, header.kids[off+1].rect.W
}

func TestDraggingAHeaderDividerResizesTheColumns(t *testing.T) {
	var widths ColumnWidths
	r := newRig(t, &widths, true)
	if a, b := r.columnWidths(); a != 164 || b != 164 {
		t.Fatalf("default widths %d %d", a, b)
	}
	r.input(widget.Input{X: 200, Y: 14, Click: true, Held: true})
	r.input(widget.Input{X: 260, Y: 14, Held: true})
	if a, b := r.columnWidths(); a < 222 || a > 226 || a+b != 328 {
		t.Fatalf("after dragging to x=260 the columns are %d and %d", a, b)
	}
	if len(widths.Weights) != 2 {
		t.Fatalf("widths = %v", widths.Weights)
	}
	r.input(widget.Input{X: 260, Y: 14}) // release
	r.input(widget.Input{X: 100, Y: 14, Held: true})
	if a, _ := r.columnWidths(); a < 222 {
		t.Fatalf("a released divider must not follow the pointer, first = %d", a)
	}
}

func TestDividerDragStopsAtTheMinimumWidth(t *testing.T) {
	var widths ColumnWidths
	r := newRig(t, &widths, true)
	r.input(widget.Input{X: 200, Y: 14, Click: true, Held: true})
	r.input(widget.Input{X: 36, Y: 14, Held: true}) // all the way left
	if a, b := r.columnWidths(); a != MinColumnPx || b != 328-MinColumnPx {
		t.Fatalf("columns %d and %d", a, b)
	}
}

func TestDoubleClickOnADividerResetsTheWidths(t *testing.T) {
	widths := ColumnWidths{Weights: []float64{3, 1}}
	r := newRig(t, &widths, true)
	if a, _ := r.columnWidths(); a != 246 {
		t.Fatalf("test setup: %d", a)
	}
	r.input(widget.Input{X: 36 + 246, Y: 14, Click: true, Held: true, Clicks: 2})
	if widths.Weights != nil {
		t.Fatalf("widths = %v, want nil (the defaults)", widths.Weights)
	}
	if a, _ := r.columnWidths(); a != 164 {
		t.Fatalf("default again: %d", a)
	}
}

func TestReadOnlyTablesCanBeResizedToo(t *testing.T) {
	var widths ColumnWidths
	r := newRig(t, &widths, false) // columns start at x=0: 200 each
	r.input(widget.Input{X: 200, Y: 14, Click: true, Held: true})
	r.input(widget.Input{X: 300, Y: 14, Held: true})
	if a, _ := r.columnWidths(); a < 298 || a > 302 {
		t.Fatalf("first = %d", a)
	}
}

func TestDividerAsksForTheResizeCursorAndCellsStillClick(t *testing.T) {
	var widths ColumnWidths
	var edits [][2]int
	spec := editableSpec()
	spec.Widths = &widths
	spec.OnEditCell = func(r, c int) { edits = append(edits, [2]int{r, c}) }
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)

	c.In = widget.Input{X: 200, Y: 14}
	Update(c, root)
	if c.Cursor != widget.CursorResizeH {
		t.Fatalf("cursor %v over the divider", c.Cursor)
	}
	c.Cursor = widget.CursorDefault
	c.In = widget.Input{X: 100, Y: 14}
	Update(c, root)
	if c.Cursor != widget.CursorDefault {
		t.Fatalf("cursor %v away from it", c.Cursor)
	}
	c.In = widget.Input{X: cellX0, Y: cellY(0), Click: true, Held: true}
	Update(c, root)
	if len(edits) != 1 {
		t.Fatalf("a click in a cell still edits it, edits = %v", edits)
	}
}

func TestTablesWithoutWidthsStateHaveNoDividers(t *testing.T) {
	spec := editableSpec() // Widths is nil: not resizable
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)
	c.In = widget.Input{X: 200, Y: 14, Click: true, Held: true}
	Update(c, root)
	if c.Cursor != widget.CursorDefault {
		t.Fatal("no divider without a Widths state")
	}
}
