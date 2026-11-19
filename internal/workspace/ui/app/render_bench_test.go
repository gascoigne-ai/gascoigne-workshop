package app

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The benchmarks time the parts of a frame in the scene that costs the most:
// the settings dialog open over the panes, as while it is dragged. They
// measure the CPU side only; the GPU cost is read with the ebitenginedebug
// build (see .harness/tasks/133-render-speed.md).

func newBenchApp(b *testing.B, dialog bool) *App {
	b.Helper()
	prefs := settings.Default()
	prefs.Layout = "side"
	path := filepath.Join(b.TempDir(), "settings.json")
	if err := settingsfile.Save(path, prefs); err != nil {
		b.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		b.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.step(core.Input{})
	a.menu = dialog
	a.step(core.Input{})
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
		a.step(core.Input{X: 600 + i%40, Y: 300 + i%20})
		a.dirty = true
		a.Draw(screen)
	}
}

func BenchmarkDragFrameNoDialog(b *testing.B) {
	a := newBenchApp(b, false)
	screen := ebiten.NewImage(winW, winH)
	for i := 0; i < b.N; i++ {
		a.step(core.Input{X: 600 + i%40, Y: 300 + i%20})
		a.dirty = true
		a.Draw(screen)
	}
}
