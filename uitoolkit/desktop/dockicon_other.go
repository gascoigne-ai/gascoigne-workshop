//go:build !(darwin && cgo)

package desktop

// SetDockIcon does nothing: the window icon is enough here.
func SetDockIcon(png []byte) {}
