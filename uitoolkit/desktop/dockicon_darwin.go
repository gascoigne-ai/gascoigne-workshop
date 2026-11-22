//go:build darwin && cgo

package desktop

/*
#cgo darwin LDFLAGS: -framework Cocoa
void setDockIcon(const void *bytes, int n);
*/
import "C"

import "unsafe"

// SetDockIcon shows the PNG as the Dock icon. It runs on the main thread, once the
// event loop is going, so it may be called before the window opens.
func SetDockIcon(png []byte) {
	if len(png) == 0 {
		return
	}
	C.setDockIcon(unsafe.Pointer(&png[0]), C.int(len(png)))
}
