// Package filebrowser is the way the window asks the user for a file: the
// desktop's own dialog when it has a program for one, else the in-app file
// browser it draws itself. The rules of the browser (which folder, which
// row, what Enter does) are in feature/workspace/filebrowser; this package
// holds its widget state, lists the folders it asks for and draws it.
//
// The feature package of the same name is imported as browsing.
//
// The browser is one slot, not a stack: asking for another file while one
// question is open replaces it, and the replaced question is dropped
// unanswered. The open browser covers the window, so no click can ask a
// second question under it. Revisit when a caller asks while another
// question must stay.
package filebrowser

import (
	"context"
	"path/filepath"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	browsing "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component asks for files and shows the in-app browser while it is open.
type Component struct {
	deps    Deps
	session *session
}

// session is the browser while it is open.
type session struct {
	model  browsing.Model
	path   TextField   // the path field, mirroring model.Text
	scroll ScrollState // the list's scroll position
	move   MoveState   // where the user dragged the dialog
	title  string
	done   func(path string)
}

// New is the component with no question asked.
func New(deps Deps) *Component { return &Component{deps: deps} }

// chooser is one way to ask the user for a file. ask returns a function that
// takes the question back.
type chooser struct {
	usable func(c *Component) bool
	ask    func(c *Component, title string, mode filedialog.Mode, name string, done func(path string)) (cancel func())
}

// choosers are the ways to ask for a file, best first: the desktop's own
// dialog when it has a program for one, else the in-app file browser, which
// always works and keeps the FileDialog contract (why not a D-Bus portal: the
// filedialog package doc).
var choosers = []chooser{
	{func(c *Component) bool { return c.deps.Desktop.Available() }, (*Component).askDesktop},
	{func(*Component) bool { return true }, (*Component).browseHome},
}

// Pick asks for a file with the first chooser that can be used and calls
// done with the path. It returns a function that closes the question without
// an answer (it kills the native dialog or closes the browser), for a caller
// that has to take it back, as the settings dialog does when it is closed.
func (c *Component) Pick(title string, mode filedialog.Mode, name string, done func(path string)) (cancel func()) {
	for _, chooser := range choosers {
		if chooser.usable(c) {
			return chooser.ask(c, title, mode, name, done)
		}
	}
	return func() {}
}

// askDesktop runs the desktop's dialog off the UI goroutine; its answer
// comes back through the bus.
func (c *Component) askDesktop(title string, mode filedialog.Mode, name string, done func(path string)) (cancel func()) {
	ctx, stop := context.WithCancel(context.Background())
	c.deps.Bus.Run(ctx, func(ctx context.Context) bus.Msg {
		path, err := c.deps.Desktop.Choose(ctx, title, mode, name)
		switch {
		case err == filedialog.ErrCancelled || ctx.Err() != nil: // closed by the user, or taken back
			return nil
		case err != nil:
			return func() { c.deps.Window.Toast("File dialog failed: "+err.Error(), severity.Error) }
		default:
			return func() { done(path) }
		}
	})
	return stop
}

// browserModes maps what a caller asks for to what the in-app browser shows.
var browserModes = map[filedialog.Mode]browsing.Mode{
	filedialog.OpenFile: browsing.Open,
	filedialog.SaveFile: browsing.Save,
}

// browseHome opens the in-app file browser in the home folder, every time: a
// browser that remembered the last folder would open somewhere the user no
// longer expects after a while.
func (c *Component) browseHome(title string, mode filedialog.Mode, name string, done func(path string)) (cancel func()) {
	return c.Browse("", title, mode, name, done)
}

// Browse opens the in-app file browser in start, or in the home folder when
// start is "" (the root when that is unknown too). A screenshot scene opens
// it on a folder of its own.
func (c *Component) Browse(start, title string, mode filedialog.Mode, name string, done func(path string)) (cancel func()) {
	home, err := c.deps.Folders.Home()
	if err != nil {
		home = "" // ~ is then typed as a name, and the root is the last start
	}
	model, effect := browsing.New(browserModes[mode], start, home, name)
	opened := &session{model: model, title: title, done: done}
	c.session = opened
	c.follow(effect)
	return func() {
		if c.session == opened {
			c.Close()
		}
	}
}

// Shown reports whether the in-app browser is open.
func (c *Component) Shown() bool { return c.session != nil }

// Path is the text of the path field, "" while the browser is closed.
func (c *Component) Path() string {
	if c.session == nil {
		return ""
	}
	return c.session.path.Text
}

// Close closes the browser without an answer.
func (c *Component) Close() { c.session = nil }

// update applies msg to the open browser and does what it asks.
func (c *Component) update(msg browsing.Msg) {
	if c.session == nil {
		return
	}
	model, effect := browsing.Update(c.session.model, msg)
	c.session.model = model
	c.follow(effect)
}

// follow lists the folder the browser asked for, or takes the chosen path,
// asking first when it replaces a file. A listing is read on the UI
// goroutine: a folder reads in far less than a frame, and the next frame
// must show it.
func (c *Component) follow(effect browsing.Effect) {
	open := c.session
	if open.path.Text != open.model.Text {
		open.path = NewTextField(open.model.Text)
	}
	switch {
	case effect.List != "":
		entries, err := c.deps.Folders.List(effect.List)
		open.scroll = ScrollState{}
		c.update(browsing.Listed{Dir: effect.List, Entries: entries, Err: err})
	case effect.Chosen != "" && effect.Replace:
		chosen := effect.Chosen
		c.deps.Dialogs.Confirm("Replace file", filepath.Base(chosen)+" already exists.\nReplace it?", "Replace", func() { c.take(open, chosen) })
	case effect.Chosen != "":
		c.take(open, effect.Chosen)
	}
}

// take closes the browser and hands the chosen path to its caller.
func (c *Component) take(taken *session, path string) {
	if c.session == taken {
		c.Close()
	}
	taken.done(path)
}
