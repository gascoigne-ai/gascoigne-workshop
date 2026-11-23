package logic

// RowRange is the half-open range of tree rows to build: 0 <= First <= Last <= Count.
type RowRange struct{ First, Last int }

// RowGeometry is what decides which rows are in view: the number of rows,
// the one height they all have and the height of the box that shows them.
type RowGeometry struct{ Count, RowH, ViewH int }

// Range is the rows to build for a scroll position: those the box shows
// plus one, for a row that shows only in part. A box taller than ViewH
// (ViewH is an upper bound, the window's height) only shows fewer rows than
// are built, never more.
//
// Every row has the one height RowH, so the first row in view is a
// division. Rejected: a prefix sum of heights, which rows of one height do
// not need; revisit when rows stop having one height. Rejected too: rows
// built around the view (overscan). The window builds the tree again
// before it paints whenever Tick reports that the range moved, so a row
// that scrolls into view is never missing from a painted frame.
// RowH <= 0 or ViewH <= 0 builds every row: nothing is known to cut by.
func (geometry RowGeometry) Range(scrollY int) RowRange {
	if geometry.RowH <= 0 || geometry.ViewH <= 0 {
		return RowRange{0, geometry.Count}
	}
	inView := (geometry.ViewH + geometry.RowH - 1) / geometry.RowH
	first := max(0, min(scrollY/geometry.RowH, geometry.Count-inView))
	return RowRange{first, min(geometry.Count, first+inView+1)}
}
