// Package app is the composition root: the App that builds the three shared
// stores, hands the adapters the program was given to the window composite
// (ui/window), and runs the frame loop around it as an ebiten.Game. It holds
// no handler: what a click or a key does lives in the window and its screen
// areas. What stays here is what only a running program knows: the window
// system's scale, the desktop's key repeat, the window title and the cursor.
package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/window"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// App is the whole program's window. It implements ebiten.Game.
type App struct {
	window    *window.Component
	sampler   core.Sampler // the input of a frame, with the desktop's key repeat
	tps       int          // the update rate last set
	title     string       // the window title last set
	baseScale float64      // the platform's interface factor under the user's scale
	display   DisplayScale // the desktop's display scale when the window system hides it; nil has none
}

// The forwards below are what the program, the end-to-end screenshots and
// apptest reach the window through; none of them holds logic.

// Err is the error that ended the run, if any.
func (a *App) Err() error { return a.window.Err() }

// FlushWorkspace writes every open collection and waits for the writes; main
// calls it on exit.
func (a *App) FlushWorkspace() error { return a.window.FlushWorkspace() }

// WaitWrites blocks until the background writes are done, for the paths that
// must not lose data. It is exported for apptest, whose tests wait for saves
// before they read files.
func (a *App) WaitWrites() { a.window.WaitWrites() }

// RestoredWindow is the window geometry of the last run, clamped to a
// monitor of mw by mh; the zero Window when nothing was remembered.
func (a *App) RestoredWindow(mw, mh int) state.Window { return a.window.RestoredWindow(mw, mh) }

// SaveState writes the state now; main calls it on exit.
func (a *App) SaveState() error { return a.window.SaveState() }

// Step runs one frame with the given input. Tests drive the app through Step
// without a window; it is exported only for apptest, the test-support package
// that lives outside package app so whole-app tests can move to e2e.
func (a *App) Step(in core.Input) { a.window.Step(in) }

// SetSize sets the window size in logical pixels and nothing else, which is
// what a test window of fixed size wants; apptest calls it before the first
// Step.
func (a *App) SetSize(width, height int) { a.window.SetSize(width, height) }

// Tree is the laid-out view tree the next Draw paints. It is exported
// because apptest cannot build a dump or a widget query without it, and
// dump formatting belongs to the tests, not to production code.
func (a *App) Tree() view.Node { return a.window.Tree() }

// SettingsPath is the settings file the window reads and writes; "" keeps
// settings in memory.
func (a *App) SettingsPath() string { return a.window.SettingsPath() }

// Driver is the way the end-to-end screenshots put the window into a state.
func (a *App) Driver() *window.Driver { return a.window.Driver() }
