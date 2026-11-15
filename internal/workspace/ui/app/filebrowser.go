package app

import (
	"context"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fileChooser is one way to ask the user for a file. ask returns a
// function that takes the question back.
type fileChooser struct {
	usable func(a *App) bool
	ask    func(a *App, title string, save bool, name string, done func(path string)) (cancel func())
}

// fileChoosers are the ways to ask for a file, best first: the desktop's
// own dialog when it has a program for one, else the in-app file browser
// (decision 18), which always works.
var fileChoosers = []fileChooser{
	{func(a *App) bool { return a.files.Available() }, (*App).askDesktop},
	{func(*App) bool { return true }, (*App).browseFiles},
}

// pickFile asks for a file with the first chooser that can be used and
// calls done with the path. It returns a function that closes the question
// without an answer (it kills the native dialog or closes the browser), for
// a caller that has to take it back, as the settings dialog does when it is
// closed.
func (a *App) pickFile(title string, save bool, name string, done func(path string)) (cancel func()) {
	for _, chooser := range fileChoosers {
		if chooser.usable(a) {
			return chooser.ask(a, title, save, name, done)
		}
	}
	return func() {}
}

// askDesktop runs the desktop's dialog off the UI goroutine; its answer
// comes back through the bus.
func (a *App) askDesktop(title string, save bool, name string, done func(path string)) (cancel func()) {
	ctx, stop := context.WithCancel(context.Background())
	go func() {
		path, err := a.files.Choose(ctx, title, save, name)
		switch {
		case err == port.ErrCancelled || ctx.Err() != nil: // closed by the user, or taken back
		case err != nil:
			a.bus.Dispatch(func() { a.toast("File dialog failed: "+err.Error(), true) })
		default:
			a.bus.Dispatch(func() { done(path) })
		}
	}()
	return stop
}

// fileBrowser is the in-app file browser while it is open.
type fileBrowser struct {
	model  filebrowser.Model
	path   TextField   // the path field, mirroring model.Text
	scroll ScrollState // the list's scroll position
	move   MoveState   // where the user dragged the dialog
	title  string
	done   func(path string)
}

// browseFiles opens the in-app file browser in the home folder, every
// time: a browser that remembered the last folder would open somewhere
// the user no longer expects after a while.
func (a *App) browseFiles(title string, save bool, name string, done func(path string)) (cancel func()) {
	return a.browseIn("", title, save, name, done)
}

// browseIn opens the in-app file browser in start, or in the home folder
// when start is "" (the root when that is unknown too).
func (a *App) browseIn(start, title string, save bool, name string, done func(path string)) (cancel func()) {
	home, err := a.folders.Home()
	if err != nil {
		home = "" // ~ is then typed as a name, and the root is the last start
	}
	mode := filebrowser.Open
	if save {
		mode = filebrowser.Save
	}
	model, effect := filebrowser.New(mode, start, home, name)
	browser := &fileBrowser{model: model, title: title, done: done}
	a.browser = browser
	a.followBrowser(effect)
	return func() {
		if a.browser == browser {
			a.closeBrowser()
		}
	}
}

func (a *App) closeBrowser() { a.browser = nil }

// browse applies msg to the open browser and does what it asks.
func (a *App) browse(msg filebrowser.Msg) {
	if a.browser == nil {
		return
	}
	model, effect := filebrowser.Update(a.browser.model, msg)
	a.browser.model = model
	a.followBrowser(effect)
}

// followBrowser lists the folder the browser asked for, or takes the chosen
// path, asking first when it replaces a file. A listing is read on the UI
// goroutine: a folder reads in far less than a frame, and the next frame
// must show it.
func (a *App) followBrowser(effect filebrowser.Effect) {
	browser := a.browser
	if browser.path.Text != browser.model.Text {
		browser.path = NewTextField(browser.model.Text)
	}
	switch {
	case effect.List != "":
		entries, err := a.folders.List(effect.List)
		browser.scroll = ScrollState{}
		a.browse(filebrowser.Listed{Dir: effect.List, Entries: entries, Err: err})
	case effect.Chosen != "" && effect.Replace:
		chosen := effect.Chosen
		a.confirm("Replace file", filepath.Base(chosen)+" already exists.\nReplace it?", "Replace", func() { a.takeBrowsed(browser, chosen) })
	case effect.Chosen != "":
		a.takeBrowsed(browser, effect.Chosen)
	}
}

// takeBrowsed closes the browser and hands the chosen path to its caller.
func (a *App) takeBrowsed(browser *fileBrowser, path string) {
	if a.browser == browser {
		a.closeBrowser()
	}
	browser.done(path)
}

// browserKey is a key of the browser and what it does; when says whether
// the key is the browser's this frame (Backspace is the field's while a
// name is typed).
type browserKey struct {
	key  ebiten.Key
	edit w.Edit
	when func(in w.Input, model filebrowser.Model) bool
	msg  filebrowser.Msg
}

var browserKeys = []browserKey{
	{ebiten.KeyArrowUp, w.EditUp, func(in w.Input, _ filebrowser.Model) bool { return in.Alt }, filebrowser.WentUp{}},
	{ebiten.KeyArrowUp, w.EditUp, func(in w.Input, _ filebrowser.Model) bool { return !in.Alt }, filebrowser.Moved{By: -1}},
	{ebiten.KeyArrowDown, w.EditDown, func(w.Input, filebrowser.Model) bool { return true }, filebrowser.Moved{By: 1}},
	{ebiten.KeyBackspace, w.EditBackspace, func(_ w.Input, model filebrowser.Model) bool { return model.AtFolder() }, filebrowser.WentUp{}},
}

// keyIsBrowsers reports whether the browser takes key this frame: a field
// with selected text keeps its Backspace, which deletes the selection.
func (browser *fileBrowser) keyIsBrowsers(key browserKey, in w.Input) bool {
	if key.key == ebiten.KeyBackspace && browser.path.HasSel {
		return false
	}
	return hasEdit(in.Edits, key.edit) && key.when(in, browser.model)
}

// fileBrowserKeys handles the browser's keys and takes them from the
// frame's input, so the path field under them does not act on them too.
// Escape is handled with the other layers (escape).
func (a *App) fileBrowserKeys(in w.Input) {
	if in.Enter {
		a.ctx.In.Consume(ebiten.KeyEnter)
		a.browse(filebrowser.Entered{})
		return
	}
	for _, key := range browserKeys {
		if a.browser.keyIsBrowsers(key, in) {
			a.ctx.In.Consume(key.key)
			a.browse(key.msg)
			return
		}
	}
}

func hasEdit(edits []w.Edit, edit w.Edit) bool {
	for _, each := range edits {
		if each == edit {
			return true
		}
	}
	return false
}

// The size of the browser.
const (
	browserW      = 640
	browserMaxH   = 520
	browserMargin = 40
	browserIconW  = 16
	browserFootH  = w.RowH + 2*12 // the footer's buttons and its padding
)

var (
	browserTemplate    = markup.Must("filebrowser.vml", filebrowserSource)
	browserRowTemplate = markup.Must("filebrowserrow.vml", filebrowserrowSource)
)

// browserRow is what a row of the list shows of one entry.
type browserRow struct {
	Name     string
	Dir      bool
	Selected bool
	Pick     func()
	Open     func()
}

// fileBrowserView is the browser while it is open, nothing otherwise.
func (a *App) fileBrowserView() Node {
	browser := a.browser
	if browser == nil {
		return nil
	}
	if a.dialog == nil { // the field keeps the keyboard, unless a question is asked over it
		browser.path.Focused = true
	}
	model := browser.model
	height := min(browserMaxH, a.h-2*browserMargin)
	listH := height - w.DialogTitleH - w.DialogToolbarH - browserFootH
	visible := model.Visible()
	first, last := shownRows(browser.scroll.Y, listH, len(visible))
	rows := make([]Node, 0, last-first)
	for i := first; i < last; i++ {
		rows = append(rows, a.fileBrowserRow(i, visible[i], i == model.Selected))
	}
	primary := "Open"
	if model.Mode == filebrowser.Save {
		primary = "Save"
	}
	return browserTemplate.MustBuild(markup.Scope{
		"width":       browserW,
		"height":      height,
		"move":        &browser.move,
		"close":       a.closeBrowser,
		"title":       browser.title,
		"subtitle":    DialogSubtitle(model.Count()),
		"path":        &browser.path,
		"typed":       a.browserTyped,
		"up":          func() { a.browse(filebrowser.WentUp{}) },
		"hidden":      model.ShowHidden,
		"toggle":      func() { a.browse(filebrowser.HiddenToggled{}) },
		"scroll":      &browser.scroll,
		"above":       first * w.RowH,
		"rows":        rows,
		"below":       (len(visible) - last) * w.RowH,
		"empty":       len(visible) == 0,
		"problem":     model.Problem,
		"primary":     primary,
		"enter":       func() { a.browse(filebrowser.Entered{}) },
		"compactH":    w.CompactH,
		"rowH":        w.RowH,
		"iconBtnW":    w.IconBtnW,
		"gutter":      w.Gutter,
		"browserFoot": browserFootH,
	})
}

// browserTyped follows an edit of the path field; the list goes back to its
// top, where the new first match is.
func (a *App) browserTyped() {
	a.browser.scroll.Y = 0
	a.browse(filebrowser.Typed{Text: a.browser.path.Text})
}

// openBrowserRow selects a row and takes it, as Enter would.
func (a *App) openBrowserRow(index int) {
	a.browse(filebrowser.Picked{Index: index})
	a.browse(filebrowser.Entered{})
}

// shownRows are the rows built for a list scrolled to scrollY: those in
// view and one more at each end, so Up and Down can bring the next row into
// view. Only these are built, since a folder may hold thousands of names.
func shownRows(scrollY, listH, count int) (first, last int) {
	first = min(max(scrollY/w.RowH-1, 0), count)
	last = min(first+listH/w.RowH+3, count)
	return first, last
}

// fileBrowserRow is one entry of the list: a click selects it, a double
// click opens it.
func (a *App) fileBrowserRow(index int, entry port.FolderEntry, selected bool) Node {
	return browserRowTemplate.MustBuild(markup.Scope{
		"row": browserRow{
			Name:     entry.Name,
			Dir:      entry.Dir,
			Selected: selected,
			Pick:     func() { a.browse(filebrowser.Picked{Index: index}) },
			Open:     func() { a.openBrowserRow(index) },
		},
		"rowH":  w.RowH,
		"iconW": browserIconW,
	})
}

// Templates, parsed once above.
const (
	filebrowserSource = `<!-- The in-app file browser: the path field filters the list or takes a path,
     the list shows the folder, the footer says what went wrong and holds the buttons. -->
<dialog w="{{width}}" h="{{height}}" moved="move" on-dismiss="close">
  <dialog-title title="{{title}}" on-dismiss="close">
    <slot name="subtitle"/>
  </dialog-title>
  <dialog-toolbar>
    <icon-button icon="chevron-up" w="{{iconBtnW}}" h="{{compactH}}" on-click="up" tooltip="Parent folder (Alt+Up)"/>
    <field state="path" h="{{rowH}}" grow="1" bg="sunken" placeholder="Folder, file name or pattern such as *.json" on-change="typed" text-menu="path"/>
    <check-row on="{{hidden}}" on-click="toggle">
      <text>Hidden files</text>
    </check-row>
  </dialog-toolbar>
  <scroll state="scroll" grow="1" bg="sunken">
    <col>
      <spacer h="{{above}}"/>
      <slot for="row in rows" name="row"/>
      <spacer h="{{below}}"/>
      <row if="empty" h="{{rowH}}" pad-x="{{gutter}}">
        <text muted>Nothing to show</text>
      </row>
    </col>
  </scroll>
  <row h="{{browserFoot}}" pad-x="16" gap="8" align-items="center" border-t="border-soft">
    <text small color="error" ellipsis grow="1" shrink>{{problem}}</text>
    <choice selected="false" min-w="80" h="{{rowH}}" on-click="close">Cancel</choice>
    <choice selected="true" min-w="80" h="{{rowH}}" on-click="enter">{{primary}}</choice>
  </row>
</dialog>
`

	filebrowserrowSource = `<list-row selected="{{row.Selected}}" h="{{rowH}}" pad-x="12" gap="8" align-items="center" keep-in-view="{{row.Selected}}" on-click="row.Pick" on-double-click="row.Open">
  <icon if="row.Dir" name="folder" w="{{iconW}}" muted/>
  <spacer if="!row.Dir" w="{{iconW}}"/>
  <text ellipsis grow="1">{{row.Name}}</text>
</list-row>
`
)
