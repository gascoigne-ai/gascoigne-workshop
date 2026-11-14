// Package settingsui is the settings dialog: a title bar with a search
// field, a list of pages, the pages generated from the settings schema
// (title, description and a control per setting, a mark and Reset when it
// differs from the default), the font pickers, and the text of
// settings.json with live validation and completion. The application
// supplies the pages that are tables (colors, shortcuts) and does what a
// change of settings implies; the dialog reaches it through Host.
package settingsui

import (
	"embed"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fontfamily"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The size of the dialog: a title bar, a list of pages on the left and the
// page on the right.
const (
	DialogWidth  = 760
	DialogMaxH   = 560
	ScreenMargin = 40  // kept free around the dialog in a small window
	MinWidth     = 520 // the smallest the user can make it
	MinHeight    = 360
	DialogNavW   = 170
	DialogTitleH = 44
)

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
	SettingsStore() port.SettingsStore
	// FontStore finds installed fonts and reads their files.
	FontStore() port.FontStore
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
}

// SettingsDialog is the state of the settings dialog. The zero value is ready.
type SettingsDialog struct {
	Size   ResizeState // the size the user gave the dialog; 0 is the default
	Cat    string      // the page shown
	Search TextField   // filters the settings across pages
	scroll ScrollState

	inputs map[string]*TextField // fields of the text and number settings
	notes  map[string]string     // why a typed value was refused
	scale  SliderState

	Families   []fontfamily.Family // installed font families, once scanned
	FontSearch TextField           // narrows the font dropdown
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
	d.inputs, d.notes = nil, nil
	d.Raw, d.RawText, d.Problems, d.rawBefore = false, TextArea{}, nil, ""
	d.FontSearch = NewTextField("")
	d.list = fontList{faces: d.list.faces}
}

// ResetScroll puts the page back at its top.
func (d *SettingsDialog) ResetScroll() { d.scroll.Y = 0 }

// View is the dialog, centered over a dimmed window.
func (d *SettingsDialog) View(host Host) Node {
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
	return Frag(
		ResizableDialog(&d.Size, min(MinWidth, maxW), min(MinHeight, maxH), maxW, maxH, W(dw), H(dh), OnDismiss(func() { host.Send(prefs.Close{}) }),
			d.titleBar(host),
			If(d.Raw, Col(Grow(1), Pad(16), d.rawView(host))),
			If(!d.Raw, Row(Grow(1),
				Col(W(DialogNavW), Pad(8), Gap(2), Bg(w.Frame),
					For(pages, func(_ int, p page) Node { return d.navItem(p, p.id == cur) }),
				),
				Scroll(&d.scroll, Grow(1), Pad(16), BorderL(w.BorderSoft), d.pageContent(host, cur)),
			)),
		),
		d.FontList(host),
	)
}

// pageContent is the content of page id.
func (d *SettingsDialog) pageContent(host Host, id string) Node {
	for _, p := range pages {
		if p.id != id {
			continue
		}
		if p.category != "" {
			return d.FormPage(host, p.category)
		}
		return host.ExtraPage(id)
	}
	return nil
}

// titleBar is the dialog's header. The search field (or a spacer in the raw
// view), Save and the mode switch are slots: they hold state or logic.
func (d *SettingsDialog) titleBar(host Host) Node {
	search := If(!d.Raw, Field(&d.Search, H(28), Grow(1), Bg(w.Sunken), Placeholder("Search settings"), OnChange(d.ResetScroll), TextMenu(&d.Search)))
	if d.Raw {
		search = Spacer(Grow(1))
	}
	return DialogTitle("Settings", OnDismiss(func() { host.Send(prefs.Close{}) }), search, d.saveButton(host), d.modeSwitch(host))
}

//go:embed navitem-open.vml navitem-closed.vml
var navItemTemplates embed.FS

var (
	navOpenTemplate   = markup.MustLoad(navItemTemplates, "navitem-open.vml")
	navClosedTemplate = markup.MustLoad(navItemTemplates, "navitem-closed.vml")
)

// navItem is one page in the list: the open one is tinted and bold.
func (d *SettingsDialog) navItem(p page, open bool) Node {
	template := navClosedTemplate
	if open {
		template = navOpenTemplate
	}
	return template.MustBuild(markup.Scope{
		"rowH":   w.RowH,
		"gutter": w.Gutter,
		"label":  p.label,
		"pick":   func() { d.Cat, d.scroll.Y = p.id, 0 },
	})
}

// modeSwitch changes the dialog between the form and the text of settings.json.
func (d *SettingsDialog) modeSwitch(host Host) Node {
	if d.Raw {
		return Button("Form", W(60), H(w.CompactH), OnClick(d.CloseRaw), Tooltip("Back to the settings form"))
	}
	return Button("JSON", W(60), H(w.CompactH), OnClick(func() { d.OpenRaw(host) }), Tooltip("Edit settings.json as text"))
}

// Reload resets what depends on the settings in effect: typed text and
// notes, and the fonts. The host calls it after it replaced the settings.
func (d *SettingsDialog) Reload(host Host) {
	d.inputs, d.notes = nil, nil
	d.ApplyFonts(host)
}

// Searching reports whether a search text is typed.
func (d *SettingsDialog) Searching() bool { return strings.TrimSpace(d.Search.Text) != "" }

// saveButton writes the changes that were applied but not stored; it is lit
// while there are some. With auto-save on and nothing pending there is
// nothing to save and there is no button.
func (d *SettingsDialog) saveButton(host Host) Node {
	if host.Unsaved() {
		return PrimaryButton("Save", W(60), H(w.CompactH), OnClick(func() { d.Save(host) }), Tooltip("Write the changes to settings.json"))
	}
	if host.Settings().AutoSave {
		return nil
	}
	return Button("Save", W(60), H(w.CompactH), OnClick(func() { d.Save(host) }), Tooltip("Nothing to save yet; changes are written when you click Save"))
}

// Save writes what is applied but not stored: the raw text as typed when
// that is what was held back, else the settings in effect.
func (d *SettingsDialog) Save(host Host) {
	if d.rawPending && d.Raw && len(d.Problems) == 0 && host.SettingsPath() != "" {
		if err := host.SettingsStore().WriteText(host.SettingsPath(), []byte(d.RawText.Text)); err != nil {
			host.Send(prefs.Toast{Text: "Could not save settings: " + err.Error(), IsErr: true})
			return
		}
		d.Stamp(host, host.SettingsPath())
		d.rawPending = false
		host.Send(prefs.MarkSaved{})
		return
	}
	d.rawPending = false
	host.Send(prefs.Persist{})
}
