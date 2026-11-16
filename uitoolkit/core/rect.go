// Package core is a small GPU-drawn UI toolkit on Ebitengine: theme
// tokens, a drawing context that works in logical pixels, input
// sampling and the clipboard. It knows nothing about the application;
// package view builds declarative views on top of it.
package core

// Rect is an axis-aligned pixel rectangle.
type Rect struct{ X, Y, W, H int }

// Contains reports whether the point is inside r.
func (r Rect) Contains(posX, posY int) bool {
	return posX >= r.X && posX < r.X+r.W && posY >= r.Y && posY < r.Y+r.H
}
