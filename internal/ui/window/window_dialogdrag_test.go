package window

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// settingsRect is the rect of the settings dialog in the laid out window.
func settingsRect(t *testing.T, app *Component) core.Rect {
	t.Helper()
	for _, line := range strings.Split(layoutDump(app), "\n") {
		if !strings.Contains(line, "resizableB") {
			continue
		}
		var r core.Rect
		m := rectPattern.FindStringSubmatch(line)
		fmt.Sscan(strings.Join(m[1:], " "), &r.X, &r.Y, &r.W, &r.H)
		return r
	}
	t.Fatalf("no settings dialog in:\n%s", layoutDump(app))
	return core.Rect{}
}

// Pressing on an edge or a corner of the settings dialog, anywhere along it
// (the title bar included), and dragging resizes it: the dialog stays open
// and inside the window.
func TestResizingTheSettingsDialogKeepsItOpenAndVisible(t *testing.T) {
	type grab struct {
		name   string
		x, y   func(rect core.Rect) int
		dx, dy int
	}
	at := func(callback func(rect core.Rect) int) func(core.Rect) int { return callback }
	left, right := at(func(rect core.Rect) int { return rect.X + 2 }), at(func(rect core.Rect) int { return rect.X + rect.W - 3 })
	top, bottom := at(func(rect core.Rect) int { return rect.Y + 2 }), at(func(rect core.Rect) int { return rect.Y + rect.H - 3 })
	midX, midY := at(func(rect core.Rect) int { return rect.X + rect.W/2 }), at(func(rect core.Rect) int { return rect.Y + rect.H/2 })
	titleY := at(func(rect core.Rect) int { return rect.Y + 20 })
	grabs := []grab{
		{"right edge", right, midY, 80, 0},
		{"just outside the right edge", at(func(rect core.Rect) int { return rect.X + rect.W + 2 }), midY, 60, 0},
		{"just outside the top left corner", at(func(rect core.Rect) int { return rect.X - 2 }), at(func(rect core.Rect) int { return rect.Y - 2 }), -40, -30},
		{"right edge in the title bar", right, titleY, 80, 0},
		{"left edge", left, midY, -80, 0},
		{"left edge in the title bar", left, titleY, -80, 0},
		{"left edge inward", left, midY, 150, 0},
		{"top edge", midX, top, 0, -60},
		{"top edge inward", midX, top, 0, 60},
		{"bottom edge", midX, bottom, 0, 60},
		{"bottom edge inward", midX, bottom, 0, -60},
		{"top left corner", left, top, -60, -40},
		{"bottom right corner", right, bottom, 60, 40},
	}
	for _, g := range grabs {
		t.Run(g.name, func(t *testing.T) {
			a := testApp(t)
			a.over.Settings.Show()
			a.Step(core.Input{X: 5000, Y: 5000})
			r := settingsRect(t, a)
			x, y := g.x(r), g.y(r)
			a.Step(core.Input{X: x, Y: y, Click: true, Held: true})
			for i := 1; i <= 8; i++ {
				a.Step(core.Input{X: x + g.dx*i/8, Y: y + g.dy*i/8, Held: true})
			}
			a.Step(core.Input{X: x + g.dx, Y: y + g.dy})
			if !a.over.Settings.Open() {
				t.Fatal("the settings dialog closed")
			}
			after := settingsRect(t, a)
			if after.W < 100 || after.H < 100 || after.X < 0 || after.Y < 0 || after.X+after.W > a.w || after.Y+after.H > a.h {
				t.Fatalf("dialog %+v in a %dx%d window, was %+v", after, a.w, a.h, r)
			}
			t.Logf("%+v -> %+v", r, after)
		})
	}
}
