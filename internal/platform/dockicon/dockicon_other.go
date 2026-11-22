//go:build !(darwin && cgo)

package dockicon

// Set does nothing: the window icon is enough here.
func Set(png []byte) {}
