package dockicon

import "testing"

func TestSetAcceptsEmptyInput(t *testing.T) {
	Set(nil) // must not panic or touch the Dock
}
