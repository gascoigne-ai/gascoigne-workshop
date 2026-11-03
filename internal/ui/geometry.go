package ui

import "image"

func imageRect(r Rect) image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}
