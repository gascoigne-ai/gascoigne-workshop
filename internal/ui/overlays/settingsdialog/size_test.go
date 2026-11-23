package settingsdialog

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
func dialogSize(t *testing.T, dialog *SettingsDialog, host Host) (int, int) {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	root := dialog.View(testCtx(t), host)
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
	var d SettingsDialog
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
	var d SettingsDialog
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
	var d SettingsDialog
	d.Size.W, d.Size.H = 900, 480
	d.Fresh()
	if d.Size.W != 900 || d.Size.H != 480 {
		t.Fatalf("closing the dialog forgot its size: %dx%d", d.Size.W, d.Size.H)
	}
}

// A whole drag of the dialog's edge, built, laid out and updated frame by
// frame as the app does, never closes the dialog.
func TestResizingTheDialogByAnEdgeKeepsItOpenAndInsideTheWindow(t *testing.T) {
	presses := []struct {
		name string
		at   func(rect core.Rect) (int, int)
		to   func(posX, posY int) (int, int)
	}{
		{"right edge", func(rect core.Rect) (int, int) { return rect.X + rect.W - 2, rect.Y + rect.H/2 }, func(posX, posY int) (int, int) { return posX + 120, posY }},
		{"left edge", func(rect core.Rect) (int, int) { return rect.X + 2, rect.Y + rect.H/2 }, func(posX, posY int) (int, int) { return posX - 120, posY }},
		{"bottom edge", func(rect core.Rect) (int, int) { return rect.X + rect.W/2, rect.Y + rect.H - 2 }, func(posX, posY int) (int, int) { return posX, posY + 80 }},
		{"top edge", func(rect core.Rect) (int, int) { return rect.X + rect.W/2, rect.Y + 2 }, func(posX, posY int) (int, int) { return posX, posY - 80 }},
		{"bottom right corner", func(rect core.Rect) (int, int) { return rect.X + rect.W - 2, rect.Y + rect.H - 2 }, func(posX, posY int) (int, int) { return posX + 90, posY + 60 }},
	}
	for _, p := range presses {
		t.Run(p.name, func(t *testing.T) {
			var d SettingsDialog
			host := newHost(t)
			ctx, err := core.NewCtx(testTheme())
			if err != nil {
				t.Fatal(err)
			}
			var root Node
			frame := func(in core.Input) {
				root = Stack(Row(), d.View(testCtx(t), host)) // the window's Stack, which places the floating dialog
				Layout(ctx, root, 1280, 800)
				ctx.In = in
				Update(ctx, root)
				t.Logf("in %+v size %+v rect %+v", in, d.Size, dialogRectOf(root))
			}
			frame(core.Input{})
			rect := dialogRectOf(root)
			x, y := p.at(rect)
			frame(core.Input{X: x, Y: y, Click: true, Held: true})
			tx, ty := p.to(x, y)
			for i := 1; i <= 10; i++ {
				frame(core.Input{X: x + (tx-x)*i/10, Y: y + (ty-y)*i/10, Held: true})
			}
			frame(core.Input{X: tx, Y: ty})
			after := dialogRectOf(root)
			if after.W == 0 || after.H == 0 {
				t.Fatalf("the dialog vanished: %+v", after)
			}
			if after.X < 0 || after.Y < 0 || after.X+after.W > 1280 || after.Y+after.H > 800 {
				t.Fatalf("the dialog left the window: %+v", after)
			}
			t.Logf("before %+v after %+v move %+v size %+v", rect, after, d.Move, d.Size)
		})
	}
}

// dialogRectOf is the rect of the resizable dialog in a laid out view.
func dialogRectOf(root Node) core.Rect {
	var find func(element *El) (core.Rect, bool)
	find = func(element *El) (core.Rect, bool) {
		if strings.Contains(strings.SplitN(Dump(element), "\n", 2)[0], "resizableB") {
			return element.Rect(), true
		}
		for _, k := range element.Children() {
			if r, ok := find(k); ok {
				return r, true
			}
		}
		return core.Rect{}, false
	}
	r, _ := find(root)
	return r
}
