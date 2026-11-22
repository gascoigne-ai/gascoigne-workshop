// Package settingsui is the settings dialog: a title bar, a list of pages
// with the search field on top, a title row over every page, the pages
// generated from the settings schema (a row per setting: title, description
// and a control beside or under them, a reset icon when it differs from the
// default; a bool is a switch; a font and its size share one row), the font pickers, and the text of
// settings.json with live validation and completion. The application
// supplies the pages that are tables (colors, shortcuts) and does what a
// change of settings implies; the dialog reaches it through Host.
package settingsui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"path/filepath"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The size of the dialog: a title bar, a list of pages on the left and the
// page on the right.
const (
	DialogWidth  = 760
	DialogMaxH   = w.DialogMaxH
	ScreenMargin = w.DialogMargin // kept free around the dialog in a small window
	MinWidth     = 520            // the smallest the user can make it
	MinHeight    = 360
	DialogNavW   = 170
)

// toolbarBtnW is the width of Save in the dialog's toolbar; editBtnW is that
// of the button in the page title row that opens the text of the file.
const (
	toolbarBtnW = 60
	editBtnW    = 150
)

// navGap is the gap between two pages in the list beside the form.
const navGap = w.StepHair

// scope is the sizes of the faces the context draws with plus the view's own
// entries, so a template holds no size as a number.
func scope(sizes w.Sizes, own markup.Scope) markup.Scope {
	merged := sizes.Scope()
	for name, value := range own {
		merged[name] = value
	}
	return merged
}

// Host is what the dialog needs from the application: reads, and Send for
// everything it asks to be done.
type Host interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Settings are the settings in effect.
	Settings() settings.Settings
	// Send asks the application to do something (see package prefs).
	Send(msg prefs.Msg)
	// SettingsPath is the settings file, "" when settings are not kept.
	SettingsPath() string
	// SettingsStore reads, stats and writes the settings file.
	SettingsStore() settings.SettingsStore
	// FontStore finds installed fonts and reads their files.
	FontStore() settings.FontStore
	// SetFonts switches the interface and code fonts; the error says why
	// they were refused, which is why this is not a message.
	SetFonts(ui, code w.FontData) error
	// ExtraPage is the content of a page the application draws itself, nil
	// for an id it does not know.
	ExtraPage(id string) Node
	// ActionIDs are the ids of the shortcut actions, offered when completing.
	ActionIDs() []string
	// Unsaved reports whether changes were applied but not written to the
	// file (auto-save is off).
	Unsaved() bool
	// Pointer is the pointer position in the window, where a dropdown opens.
	Pointer() (x, y int)
}

// page is one entry of the dialog's list: a form of the settings of a
// category, or a page the host draws.
type page struct {
	id, label string
	category  string // the schema category a form page shows; "" for a host page
}

var pages = []page{
	{"appearance", "Appearance", "Appearance"},
	{"colors", "Colors", ""},
	{"shortcuts", "Shortcuts", ""},
	{"requests", "Requests", "Requests"},
	{"general", "General", "General"},
	{"experimental", "Experimental", "Experimental"},
	{"system", "System", ""},
}

// SettingsDialog is the state of the settings dialog. The zero value is ready.
type SettingsDialog struct {
	Size   ResizeState // the size the user gave the dialog; 0 is the default
	Move   MoveState   // where the user dragged it; 0,0 is the center
	Cat    string      // the page shown
	Search TextField   // filters the settings across pages
	scroll ScrollState

	fields NotedFields[string] // fields of the text and number settings, and why a typed value was refused
	scale  SliderState

	Families   []settings.Family // installed font families, once scanned
	FontSearch TextField         // narrows the font dropdown
	list       fontList

	Raw        bool               // the dialog shows settings.json as text
	RawText    TextArea           // that text
	Problems   []settings.Problem // what is wrong with it
	rawBefore  string             // the text before the last change, to tell a typed quote from a paste
	rawPending bool               // the raw text was applied but not written (auto-save is off)
	stamp      settings.FileStamp // settings.json as last written or read
}

// pageID is the page shown: the chosen one, else the first.
func (d *SettingsDialog) pageID() string {
	for _, p := range pages {
		if p.id == d.Cat {
			return p.id
		}
	}
	return pages[0].id
}

// Fresh forgets everything the user did in the dialog, as closing it does:
// the page, the search, the raw view, what was typed and any open list. What
// the program learned (installed fonts, the file's stamp) stays.
func (d *SettingsDialog) Fresh() {
	d.Cat, d.Search = "", NewTextField("")
	d.scroll = ScrollState{} // the size the user chose stays
	d.fields.Forget()
	d.Raw, d.RawText, d.Problems, d.rawBefore = false, TextArea{}, nil, ""
	d.FontSearch = NewTextField("")
	d.list = fontList{faces: d.list.faces}
}

// ResetScroll puts the page back at its top.
func (d *SettingsDialog) ResetScroll() { d.scroll.Y = 0 }

// View is the dialog, centered over a dimmed window.
func (d *SettingsDialog) View(ctx *w.Ctx, host Host) Node {
	width, height := host.Size()
	maxW, maxH := max(width-2*ScreenMargin, 1), max(height-2*ScreenMargin, 1)
	dw, dh := d.Size.W, d.Size.H
	if dw == 0 {
		dw = DialogWidth
	}
	if dh == 0 {
		dh = DialogMaxH
	}
	dw, dh = min(max(dw, min(MinWidth, maxW)), maxW), min(max(dh, min(MinHeight, maxH)), maxH)
	cur := d.pageID()
	nav := make([]Node, len(pages))
	for i, p := range pages {
		nav[i] = d.navItem(ctx, p, p.id == cur)
	}
	var raw Node
	if d.Raw {
		raw = d.rawView(ctx, host)
	}
	dialog := dialogTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"size":     &d.Size,
		"move":     &d.Move,
		"minW":     min(MinWidth, maxW),
		"minH":     min(MinHeight, maxH),
		"maxW":     maxW,
		"maxH":     maxH,
		"width":    dw,
		"height":   dh,
		"close":    func() { host.Send(prefs.Close{}) },
		"titleBar": d.titleBar(host),
		"toolbar":  d.toolbar(ctx, host),
		"titleRow": d.pageTitleRow(ctx, host, cur),
		"search":   &d.Search,
		"searched": d.ResetScroll,
		"raw":      d.Raw,
		"rawView":  raw,
		"navW":     DialogNavW,
		"navGap":   navGap,
		"nav":      nav,
		"scroll":   &d.scroll,
		"page":     d.pageContent(ctx, host, cur),
	}))
	return Frag(dialog, d.FontList(ctx, host)) // the font list floats over the dialog; a template has one root
}

var (
	dialogTemplate      = markup.Must("dialog.vml", dialogSource)
	titleBarTemplate    = markup.Must("titlebar.vml", titlebarSource)
	toolbarTemplate     = markup.Must("toolbar.vml", toolbarSource)
	titleRowTemplate    = markup.Must("titlerow.vml", titlerowSource)
	savePendingTemplate = markup.Must("save-pending.vml", savePendingSource)
	saveIdleTemplate    = markup.Must("save-idle.vml", saveIdleSource)
)

// pageContent is the content of page id.
func (d *SettingsDialog) pageContent(ctx *w.Ctx, host Host, id string) Node {
	for _, p := range pages {
		if p.id != id {
			continue
		}
		if p.category != "" {
			return d.FormPage(ctx, host, p.category)
		}
		return host.ExtraPage(id)
	}
	return nil
}

// titleBar is the dialog's title bar: the name and the close button.
func (d *SettingsDialog) titleBar(host Host) Node {
	return titleBarTemplate.MustBuild(markup.Scope{
		"close":    func() { host.Send(prefs.Close{}) },
		"subtitle": DialogSubtitle(d.subtitle()),
	})
}

// subtitle is what the title bar says besides the name: the page shown, or
// the file in the raw view.
func (d *SettingsDialog) subtitle() string {
	if d.Raw {
		return "settings.json"
	}
	cur := d.pageID()
	for _, p := range pages {
		if p.id == cur {
			return p.label
		}
	}
	return ""
}

// toolbar is the row under the title bar: Save, when there is something to
// save; nil when there is not, so no empty row is left.
func (d *SettingsDialog) toolbar(ctx *w.Ctx, host Host) Node {
	save := d.saveButton(ctx, host)
	if save == nil {
		return nil
	}
	return toolbarTemplate.MustBuild(markup.Scope{"save": save})
}

// settingsFile is the name of the file the settings live in, as the title
// row shows it; the full path is its tooltip.
const settingsFile = "settings.json"

// pageTitleRow is the first row of the page: a pill with the page shown,
// the file the settings live in, and the button that switches between the
// form and the text of that file.
func (d *SettingsDialog) pageTitleRow(ctx *w.Ctx, host Host, id string) Node {
	label := ""
	for _, entry := range pages {
		if entry.id == id {
			label = entry.label
		}
	}
	file, path := settingsFile, host.SettingsPath()
	if path != "" {
		file = filepath.Base(path)
	}
	return titleRowTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"label":    label,
		"file":     file,
		"path":     path,
		"raw":      d.Raw,
		"openRaw":  func() { d.OpenRaw(host) },
		"closeRaw": d.CloseRaw,
		"editW":    editBtnW,
	}))
}

var (
	navOpenTemplate   = markup.Must("navitem-open.vml", navitemOpenSource)
	navClosedTemplate = markup.Must("navitem-closed.vml", navitemClosedSource)
)

// navItem is one page in the list: the open one is tinted and bold.
func (d *SettingsDialog) navItem(ctx *w.Ctx, entry page, open bool) Node {
	template := navClosedTemplate
	if open {
		template = navOpenTemplate
	}
	return template.MustBuild(scope(ctx.Sizes, markup.Scope{
		"label": entry.label,
		"pick":  func() { d.Cat, d.scroll.Y = entry.id, 0 },
	}))
}

// Reload resets what depends on the settings in effect: typed text and
// notes, and the fonts. The host calls it after it replaced the settings.
func (d *SettingsDialog) Reload(host Host) {
	d.fields.Forget()
	d.ApplyFonts(host)
}

// Searching reports whether a search text is typed.
func (d *SettingsDialog) Searching() bool { return strings.TrimSpace(d.Search.Text) != "" }

// saveButton writes the changes that were applied but not stored; it is lit
// while there are some. With auto-save on and nothing pending there is
// nothing to save and there is no button.
func (d *SettingsDialog) saveButton(ctx *w.Ctx, host Host) Node {
	template := saveIdleTemplate
	switch {
	case host.Unsaved():
		template = savePendingTemplate
	case host.Settings().AutoSave:
		return nil
	}
	return template.MustBuild(scope(ctx.Sizes, markup.Scope{
		"save": func() { d.Save(host) },
		"btnW": toolbarBtnW,
	}))
}

// Save writes what is applied but not stored: the raw text as typed when
// that is what was held back, else the settings in effect.
func (d *SettingsDialog) Save(host Host) {
	if d.rawPending && d.Raw && len(d.Problems) == 0 && host.SettingsPath() != "" {
		if !d.writeText(host, host.SettingsPath(), d.RawText.Text) {
			return
		}
		d.rawPending = false
		host.Send(prefs.FileMarked{Saved: true})
		return
	}
	d.rawPending = false
	host.Send(prefs.Persist{})
}

// Templates, parsed once above.
const (
	dialogSource = `<!-- The settings dialog: the title bar, then the raw text or the list of
     pages beside the page shown. Go sizes it to the window. -->
<resizable-dialog state="size" min-width="{{minW}}" min-height="{{minH}}" max-width="{{maxW}}" max-height="{{maxH}}"
                  w="{{width}}" h="{{height}}" moved="move" on-dismiss="close">
  <slot name="titleBar"/>
  <slot if="toolbar" name="toolbar"/>
  <col if="raw" grow="1" pad="{{dialogPad}}" gap="{{looseGap}}">
    <slot name="titleRow"/>
    <slot name="rawView"/>
  </col>
  <row if="!raw" grow="1">
    <col w="{{navW}}" pad="{{looseGap}}" gap="{{navGap}}" bg="frame" round-bl="menu">
      <field state="search" h="{{rowH}}" bg="sunken" placeholder="Search settings" on-change="searched" text-menu="search"/>
      <slot for="item in nav" name="item"/>
    </col>
    <col grow="1" border-l="border-soft">
      <col pad-x="{{dialogPad}}" pad-t="{{dialogPad}}">
        <slot name="titleRow"/>
      </col>
      <scroll state="scroll" grow="1" pad="{{dialogPad}}">
        <slot if="page" name="page"/>
      </scroll>
    </col>
  </row>
</resizable-dialog>
`

	titlebarSource = `<!-- The dialog's title bar: the name, the page shown and the close button, the
     rest is where the user drags the dialog. The controls are in toolbar.vml. -->
<dialog-title title="Settings" on-dismiss="close">
  <slot name="subtitle"/>
</dialog-title>
`

	toolbarSource = `<!-- The row under the title bar: Save, when there is something to save. -->
<dialog-toolbar>
  <spacer grow="1"/>
  <slot name="save"/>
</dialog-toolbar>
`

	titlerowSource = `<!-- The first row of every page: which page and which file, and the switch
     between the form and the text of the file. -->
<row gap="{{looseGap}}" align-items="center">
  <row h="{{rowH}}" pad-x="{{gutter}}" align-items="center" bg="accent-soft">
    <text bold color="accent-line">{{label}}</text>
  </row>
  <text muted tooltip="{{path}}">{{file}}</text>
  <spacer/>
  <button if="!raw" w="{{editW}}" h="{{compactH}}" on-click="openRaw" tooltip="Edit settings.json as text">Edit in settings.json</button>
  <button if="raw" w="{{editW}}" h="{{compactH}}" on-click="closeRaw" tooltip="Back to the settings form">Back to the form</button>
</row>
`

	savePendingSource = `<primary-button w="{{btnW}}" h="{{compactH}}" on-click="save" tooltip="Write the changes to settings.json">Save</primary-button>
`

	saveIdleSource = `<button w="{{btnW}}" h="{{compactH}}" on-click="save" tooltip="Nothing to save yet; changes are written when you click Save">Save</button>
`

	navitemOpenSource = `<row h="{{rowH}}" pad-x="{{gutter}}" align-items="center" control="button" on-click="pick" bg="accent-soft">
  <text bold color="accent-line" grow="1">{{label}}</text>
</row>
`

	navitemClosedSource = `<row h="{{rowH}}" pad-x="{{gutter}}" align-items="center" control="button" on-click="pick" hover-bg="hover">
  <text muted grow="1">{{label}}</text>
</row>
`
)
