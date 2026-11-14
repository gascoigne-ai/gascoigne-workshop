package ui

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Widths the status bar is pinned at: the test window and a narrow one
// where a long status line runs out of room.
const (
	statusWideW   = winW
	statusNarrowW = 280
)

// The status bar with a status line too long for the narrow width is
// pinned at both widths.
func TestTheStatusBarLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", statusWideW},
		{"narrow", statusNarrowW},
	}
	for _, size := range sizes {
		name := "statusbar-long-" + size.name
		t.Run(name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			a.status = "Sent 200 OK in 120ms to https://api.github.com/repos/golang/go/issues"
			root := a.statusBar()
			Layout(a.ctx, root, size.width, statusH)
			want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got := Dump(root); got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// Clicking Log in the status bar opens the log of calls.
func TestClickingLogInTheStatusBarOpensTheCallLog(t *testing.T) {
	a, _ := newTestApp(t)
	if a.calls.Shown() {
		t.Fatal("the call log is open before the click")
	}
	clickText(t, a, "Log", 0)
	if !a.calls.Shown() {
		t.Fatal("clicking Log left the call log closed")
	}
}
