// Package core is a small GPU-drawn UI toolkit on Ebitengine: theme
// tokens, a drawing context that works in logical pixels, input
// sampling, and the state of the stateful widgets (text field, code
// view). It knows nothing about the application; package view builds
// declarative views on top of it.
package core

// Rect is an axis-aligned pixel rectangle.
type Rect struct{ X, Y, W, H int }

// Contains reports whether the point is inside r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}
