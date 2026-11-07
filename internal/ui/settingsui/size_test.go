package settingsui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

var dialogRect = regexp.MustCompile(`resizableB w=(\d+) h=(\d+)`)

// dialogSize is the size the laid out dialog has.
func dialogSize(t *testing.T, d *Dialog, host Host) (int, int) {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	root := d.View(host)
	Layout(ctx, root, 1280, 800)
	for _, line := range strings.Split(Dump(root), "\n") {
		if m := dialogRect.FindStringSubmatch(line); m != nil {
			var w, h int
			fmt.Sscan(m[1], &w)
			fmt.Sscan(m[2], &h)
			return w, h
		}
	}
	t.Fatalf("no dialog in:\n%s", Dump(root))
	return 0, 0
}

func TestTheDialogHasItsDefaultSizeUntilTheUserResizesIt(t *testing.T) {
	var d Dialog
	host := newHost(t)
	if w, h := dialogSize(t, &d, host); w != DialogWidth || h != DialogMaxH {
		t.Fatalf("default size %dx%d, want %dx%d", w, h, DialogWidth, DialogMaxH)
	}
	d.Size.W, d.Size.H = 900, 480
	if w, h := dialogSize(t, &d, host); w != 900 || h != 480 {
		t.Fatalf("resized to %dx%d, want 900x480", w, h)
	}
}

func TestTheDialogSizeStaysInsideTheWindowAndAboveItsMinimum(t *testing.T) {
	var d Dialog
	host := newHost(t) // a 1280x800 window
	d.Size.W, d.Size.H = 5000, 5000
	if w, h := dialogSize(t, &d, host); w != 1280-2*ScreenMargin || h != 800-2*ScreenMargin {
		t.Fatalf("too big: %dx%d, want the window less its margins", w, h)
	}
	d.Size.W, d.Size.H = 10, 10
	if w, h := dialogSize(t, &d, host); w != MinWidth || h != MinHeight {
		t.Fatalf("too small: %dx%d, want %dx%d", w, h, MinWidth, MinHeight)
	}
}

func TestAFreshDialogKeepsTheSizeTheUserChose(t *testing.T) {
	var d Dialog
	d.Size.W, d.Size.H = 900, 480
	d.Fresh()
	if d.Size.W != 900 || d.Size.H != 480 {
		t.Fatalf("closing the dialog forgot its size: %dx%d", d.Size.W, d.Size.H)
	}
}
