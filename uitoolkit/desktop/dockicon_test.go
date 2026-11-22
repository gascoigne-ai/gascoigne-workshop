package desktop

import "testing"

func TestSetDockIconAcceptsEmptyInput(t *testing.T) {
	SetDockIcon(nil) // must not panic or touch the Dock
}
