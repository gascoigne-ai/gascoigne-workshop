// Package widget is a small GPU-drawn UI toolkit on Ebitengine: theme
// tokens, a drawing context that works in logical pixels, input
// sampling, and the state of the stateful widgets (text field, code
// view). It knows nothing about the application; package vml builds
// declarative views on top of it.
package widget

// Rect is an axis-aligned pixel rectangle.
type Rect struct{ X, Y, W, H int }

// Contains reports whether the point is inside r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Inset shrinks r by n on every side, never below zero size.
func (r Rect) Inset(n int) Rect {
	return Rect{r.X + n, r.Y + n, max(r.W-2*n, 0), max(r.H-2*n, 0)}
}

// SplitLeft cuts a w-wide strip off the left; w is clamped to r.
func (r Rect) SplitLeft(w int) (left, rest Rect) {
	w = min(max(w, 0), r.W)
	return Rect{r.X, r.Y, w, r.H}, Rect{r.X + w, r.Y, r.W - w, r.H}
}

// SplitRight cuts a w-wide strip off the right; w is clamped to r.
func (r Rect) SplitRight(w int) (rest, right Rect) {
	w = min(max(w, 0), r.W)
	return Rect{r.X, r.Y, r.W - w, r.H}, Rect{r.X + r.W - w, r.Y, w, r.H}
}

// SplitTop cuts an h-high strip off the top; h is clamped to r.
func (r Rect) SplitTop(h int) (top, rest Rect) {
	h = min(max(h, 0), r.H)
	return Rect{r.X, r.Y, r.W, h}, Rect{r.X, r.Y + h, r.W, r.H - h}
}

// SplitBottom cuts an h-high strip off the bottom; h is clamped to r.
func (r Rect) SplitBottom(h int) (rest, bottom Rect) {
	h = min(max(h, 0), r.H)
	return Rect{r.X, r.Y, r.W, r.H - h}, Rect{r.X, r.Y + r.H - h, r.W, h}
}
