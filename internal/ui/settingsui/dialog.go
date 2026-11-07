// Package settingsui is the settings dialog: a title bar with a search
// field, a list of pages, the pages generated from the settings schema
// (title, description and a control per setting, a mark and Reset when it
// differs from the default), the font pickers, and the text of
// settings.json with live validation and completion. The application
// supplies the pages that are tables (colors, shortcuts) and does what a
// change of settings implies; the dialog reaches it through Host.
package settingsui

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The size of the dialog: a title bar, a list of pages on the left and the
// page on the right.
const (
	DialogWidth  = 760
	DialogMaxH   = 560
	ScreenMargin = 40 // kept free around the dialog in a small window
	DialogNavW   = 170
	DialogTitleH = 44
)

// MenuItem is one row of the completion menu the dialog asks the host to open.
type MenuItem struct {
	Label string
	Run   func()
}

// Host is what the dialog needs from the application.
type Host interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Settings are the settings in effect.
	Settings() settings.Settings
	// Change edits the settings, applies them and saves the file.
	Change(edit func(*settings.Settings))
	// SetScaleLive applies an interface scale while its slider is dragged,
	// without saving; Save writes the file when the drag ends.
	SetScaleLive(percent int)
	Save()
	// Replace makes s the settings in effect with everything that follows
	// from them (theme, views, fields), without writing the file.
	Replace(s settings.Settings)
	// SettingsPath is the settings file, "" when settings are not kept.
	SettingsPath() string
	// SetFonts switches the interface and code fonts.
	SetFonts(ui, code w.FontData) error
	// Toast shows a message over the window; SetStatus replaces the status line.
	Toast(text string, isErr bool)
	SetStatus(text string)
	// TextMenu is the option that gives a field its right-click menu;
	// changed runs after the menu changed the text.
	TextMenu(field *TextField, changed func()) Option
	// OpenMenu opens a menu at a point of the window.
	OpenMenu(x, y int, items []MenuItem)
	// ExtraPage is the content of a page the application draws itself, nil
	// for an id it does not know.
	ExtraPage(id string) Node
	// ActionIDs are the ids of the shortcut actions, offered when completing.
	ActionIDs() []string
	// Pointer is the pointer position in the window, where a dropdown opens.
	Pointer() (x, y int)
	// Close closes the dialog; MarkDirty asks for a redraw; Later runs f on
	// the UI goroutine.
	Close()
	MarkDirty()
	Later(f func())
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
}

// Dialog is the state of the settings dialog. The zero value is ready.
type Dialog struct {
	Cat    string    // the page shown
	Search TextField // filters the settings across pages
	scroll ScrollState

	inputs map[string]*TextField // fields of the text and number settings
	notes  map[string]string     // why a typed value was refused
	scale  SliderState

	Families   []fonts.Family // installed font families, once scanned
	FontSearch TextField      // narrows the font dropdown
	list       fontList

	Raw       bool               // the dialog shows settings.json as text
	RawText   TextArea           // that text
	Problems  []settings.Problem // what is wrong with it
	rawBefore string             // the text before the last change, to tell a typed quote from a paste
	stamp     fileStamp          // settings.json as last written or read
}

// pageID is the page shown: the chosen one, else the first.
func (d *Dialog) pageID() string {
	for _, p := range pages {
		if p.id == d.Cat {
			return p.id
		}
	}
	return pages[0].id
}

// ResetScroll puts the page back at its top.
func (d *Dialog) ResetScroll() { d.scroll.Y = 0 }

// View is the dialog, centered over a dimmed window.
func (d *Dialog) View(host Host) Node {
	_, height := host.Size()
	h := min(DialogMaxH, height-2*ScreenMargin)
	cur := d.pageID()
	return Frag(
		Row(Bg(w.Shadow)), // dims the window
		Col(W(DialogWidth), H(h), Bg(w.Bg), Control("menu"), Shadow(), FloatCenter(), FocusScope(), Modal(), OnDismiss(host.Close),
			Row(H(DialogTitleH), PadX(16), Gap(8), AlignItems(AlignCenter), BorderB(w.BorderSoft),
				Text("Settings", Bold(), W(70)),
				If(!d.Raw, Field(&d.Search, H(28), Grow(1), Bg(w.Sunken), Placeholder("Search settings"), OnChange(d.ResetScroll), host.TextMenu(&d.Search, nil))),
				If(d.Raw, Spacer(Grow(1))),
				d.modeSwitch(host),
				IconButton(CrossIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(host.Close), Tooltip("Close (Esc)")),
			),
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
func (d *Dialog) pageContent(host Host, id string) Node {
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

// navItem is one page in the list: the open one is tinted and bold.
func (d *Dialog) navItem(p page, open bool) Node {
	items := []Item{
		H(w.RowH), PadX(12), AlignItems(AlignCenter), Control("button"),
		OnClick(func() { d.Cat, d.scroll.Y = p.id, 0 }),
	}
	if open {
		return Row(append(items, Bg(w.AccentSoft), Text(p.label, Bold(), Color(w.AccentLine), Grow(1)))...)
	}
	return Row(append(items, HoverBg(w.Hover), Text(p.label, Muted(), Grow(1)))...)
}

// modeSwitch changes the dialog between the form and the text of settings.json.
func (d *Dialog) modeSwitch(host Host) Node {
	if d.Raw {
		return Button("Form", W(60), H(w.CompactH), OnClick(d.CloseRaw), Tooltip("Back to the settings form"))
	}
	return Button("JSON", W(60), H(w.CompactH), OnClick(func() { d.OpenRaw(host) }), Tooltip("Edit settings.json as text"))
}

// Reload resets what depends on the settings in effect: typed text and
// notes, and the fonts. The host calls it after it replaced the settings.
func (d *Dialog) Reload(host Host) {
	d.inputs, d.notes = nil, nil
	d.ApplyFonts(host)
}

// Searching reports whether a search text is typed.
func (d *Dialog) Searching() bool { return strings.TrimSpace(d.Search.Text) != "" }
