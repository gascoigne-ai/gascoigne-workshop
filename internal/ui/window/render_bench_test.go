package window

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The benchmarks time the parts of a frame in the scene that costs the most:
// the settings dialog open over the panes, as while it is dragged. They
// measure the CPU side only; the GPU cost is read with the ebitenginedebug
// build (see .harness/tasks/133-render-speed.md).

func newBenchApp(b *testing.B, dialog bool) *Component {
	b.Helper()
	prefs := settings.Default()
	prefs.Layout = "side"
	path := filepath.Join(b.TempDir(), "settings.json")
	if err := settingsDisk.Save(path, prefs); err != nil {
		b.Fatal(err)
	}
	a, err := newWindow(withAdapters(startOptions{SettingsPath: path}))
	if err != nil {
		b.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.Step(core.Input{})
	if dialog {
		a.over.Settings.Show()
	}
	a.Step(core.Input{})
	b.ReportAllocs()
	b.ResetTimer()
	return a
}

func BenchmarkBuildTree(b *testing.B) {
	a := newBenchApp(b, true)
	for i := 0; i < b.N; i++ {
		a.view()
	}
}

func BenchmarkLayout(b *testing.B) {
	a := newBenchApp(b, true)
	root := a.view()
	for i := 0; i < b.N; i++ {
		view.Layout(a.ctx, root, a.w, a.h)
	}
}

func BenchmarkDrawTree(b *testing.B) {
	a := newBenchApp(b, true)
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	screen := ebiten.NewImage(winW, winH)
	for i := 0; i < b.N; i++ {
		view.Draw(a.ctx, root, screen)
	}
}

// BenchmarkDragFrame is one frame of dragging the dialog: the pointer moves,
// Update builds and lays out the tree, Draw builds, lays out and paints it.
func BenchmarkDragFrame(b *testing.B) {
	a := newBenchApp(b, true)
	screen := ebiten.NewImage(winW, winH)
	for i := 0; i < b.N; i++ {
		a.Step(core.Input{X: 600 + i%40, Y: 300 + i%20})
		a.dirty = true
		a.Draw(screen)
	}
}

func BenchmarkDragFrameNoDialog(b *testing.B) {
	a := newBenchApp(b, false)
	screen := ebiten.NewImage(winW, winH)
	for i := 0; i < b.N; i++ {
		a.Step(core.Input{X: 600 + i%40, Y: 300 + i%20})
		a.dirty = true
		a.Draw(screen)
	}
}

// BenchmarkDrawStats reports what one drag frame asks the GPU for, as
// metrics per frame: the counts the batching items bring down.
func BenchmarkDrawStats(b *testing.B) {
	a := newBenchApp(b, true)
	screen := ebiten.NewImage(winW, winH)
	a.ctx.Stats = &core.DrawStats{}
	for i := 0; i < b.N; i++ {
		a.Step(core.Input{X: 600 + i%40, Y: 300 + i%20})
		a.dirty = true
		a.ctx.Stats = &core.DrawStats{}
		a.Draw(screen)
	}
	b.ReportMetric(float64(a.ctx.Stats.MaskFills), "maskfills/frame")
	b.ReportMetric(float64(a.ctx.Stats.RingFills), "ringfills/frame")
	b.ReportMetric(float64(a.ctx.Stats.Widths), "widths/frame")
}

// BenchmarkLongList draws a scrolled list of 2000 rows in a box that shows
// about 30 of them: the frame pays only for the rows in view once children
// outside the clip are skipped. It reports the elements drawn per frame.
func BenchmarkLongList(b *testing.B) {
	a := newBenchApp(b, false)
	rows := make([]view.Item, 2000)
	for i := range rows {
		rows[i] = view.Row(view.H(24), view.Bg(core.Panel), view.Border(core.Rule), view.Text("row "+strconv.Itoa(i)))
	}
	root := view.Row(view.AlignItems(view.AlignStart), view.Col(view.W(600), view.H(700), widget.Scroll(&widget.ScrollState{Y: 24 * 1000}, rows...)))
	view.Layout(a.ctx, root, a.w, a.h)
	screen := ebiten.NewImage(winW, winH)
	a.ctx.Stats = &core.DrawStats{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.ctx.Stats = &core.DrawStats{}
		view.Draw(a.ctx, root, screen)
	}
	b.ReportMetric(float64(a.ctx.Stats.Elements), "elements/frame")
}
