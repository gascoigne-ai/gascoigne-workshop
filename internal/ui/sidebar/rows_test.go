package sidebar

import (
	"strings"
	"testing"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// treeRows are the elements of the tree's scroll box that are rows of the tree.
func treeRows(root Node) []Node {
	scroll := find(root, "scrollB")
	var rows []Node
	for _, child := range scroll.Children() {
		if strings.Contains(Dump(child), "control=row") && strings.Contains(strings.SplitN(Dump(child), "\n", 2)[0], "onClick=true") {
			rows = append(rows, child)
		}
	}
	return rows
}

// The windowing cuts rows by one height times an index, so the tree must
// keep one height and no gap between rows.
func TestTreeRowsHaveOneHeightAndStack(t *testing.T) {
	ctx, side := benchSidebar(t, 50)
	rows := treeRows(laidOut(ctx, side))
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	for i, row := range rows {
		rect := row.Rect()
		if rect.H != ctx.Sizes.Row || rect.Y != rows[0].Rect().Y+i*ctx.Sizes.Row {
			t.Fatalf("row %d is at y=%d h=%d, want y=%d h=%d", i, rect.Y, rect.H, rows[0].Rect().Y+i*ctx.Sizes.Row, ctx.Sizes.Row)
		}
	}
}

func TestSidebarBuildsOnlyRowsInView(t *testing.T) {
	ctx, side := benchSidebar(t, 5000)
	rowH := ctx.Sizes.Row
	side.scroll.Y = 2500 * rowH
	root := side.View(ctx, sidebarH)
	Layout(ctx, root, sidebarW, sidebarH)

	rows := treeRows(root)
	if want := (sidebarH+rowH-1)/rowH + 1; len(rows) != want {
		t.Errorf("built %d rows, want %d", len(rows), want)
	}
	if !strings.Contains(Dump(root), `text="r2499"`) {
		t.Errorf("the row at the scroll position is missing")
	}
	scroll := find(root, "scrollB")
	var spacers int
	for _, child := range scroll.Children() {
		if !strings.Contains(strings.SplitN(Dump(child), "\n", 2)[0], "control=row") {
			spacers++
		}
	}
	if spacers != 2 {
		t.Errorf("%d spacers, want 2:\n%s", spacers, Dump(scroll)[:600])
	}
	last := scroll.Children()[len(scroll.Children())-1].Rect()
	first := scroll.Children()[0].Rect()
	if height := last.Y + last.H - first.Y; height != 5000*rowH {
		t.Errorf("content is %d high, want %d", height, 5000*rowH)
	}
}

func TestSidebarSmallTreeHasNoSpacers(t *testing.T) {
	ctx, side, _, _ := withShop(t)
	scroll := find(laidOut(ctx, side), "scrollB")
	for _, child := range scroll.Children() {
		if !strings.Contains(strings.SplitN(Dump(child), "\n", 2)[0], "control=row") {
			t.Errorf("a tree that fits has a spacer:\n%s", Dump(scroll))
		}
	}
	if len(scroll.Children()) != 4 {
		t.Errorf("%d rows, want 4", len(scroll.Children()))
	}
}

func TestSidebarTickAsksRebuildWhenRangeMoves(t *testing.T) {
	ctx, side := benchSidebar(t, 5000)
	rowH := ctx.Sizes.Row
	side.scroll.Y = 100 * rowH
	side.View(ctx, sidebarH)

	tests := []struct {
		name string
		do   func()
		want bool
	}{
		{"same position", func() {}, false},
		{"inside the same row", func() { side.scroll.Y = 100*rowH + rowH - 1 }, false},
		{"past a row boundary", func() { side.scroll.Y = 101 * rowH }, true},
		{"back again after a rebuild", func() { side.View(ctx, sidebarH) }, false},
		{"a thumb drag far down", func() { side.scroll.Y = 4000 * rowH }, true},
		{"sent mode builds no rows", func() { side.showSent = true; side.View(ctx, sidebarH); side.scroll.Y = 5 * rowH }, false},
	}
	for _, test := range tests {
		test.do()
		if got := side.Tick(); got != test.want {
			t.Errorf("%s: Tick = %v, want %v", test.name, got, test.want)
		}
	}
}
