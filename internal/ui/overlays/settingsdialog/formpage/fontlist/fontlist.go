// Package fontlist is the font drop-down of the settings form: the control
// that shows the family in use, and the list that opens below the pointer
// with a search field and every installed family drawn in its own font.
//
// The list is a child of formpage, which draws the form rows and owns a List.
// It was part of formpage until the form row and the drop-down could be told
// apart: the list shares no state with the rows but the settings, which it
// reads through Host and changes through prefs messages. A choice goes out as
// the same Change message every other setting uses; nothing is stored beside
// the settings, so the saved family and the one shown cannot disagree.
//
// Why a component here and not a toolkit widget: the list is specific to
// fonts (it reads font files and applies the family to the interface and the
// code face), and no other view lists long rows with a filter. Revisit when a
// second view needs a filtered list of many rows: then the scrolling and
// filtering part moves to uitoolkit/widget and this package keeps the font
// knowledge.
package fontlist

import (
	"bytes"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist/logic"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// builtIn is the name the list shows for the embedded font.
const builtIn = "Built-in"

// List is the state of the font drop-down: the installed families and, while
// it is open, the role it picks for, its place, the search and the scroll.
// The zero value is ready; the host is passed to each call, not kept, so the
// value can be a field of a page that has no host yet.
//
// Families is a slice because it is read in order (the rows) and by name
// (fonts.Find, a scan over some hundreds of names, at most once per frame);
// a map by name would lose the sorted order. faces is a map by file because it
// is looked up per visible row; a nil value records a file that could not be
// read, so it is tried once. Revisit the slice when a lookup by name shows in
// a profile. The visible rows are computed from the scroll, not stored.
type List struct {
	Families []fonts.Family // installed font families, once scanned
	Search   TextField      // narrows the list

	open   *logic.Opened // nil while closed
	scroll ScrollState
	faces  map[string]*text.GoTextFaceSource // by file; nil when the file could not be read
}

// Open opens the list for role below the pointer, on the family in use.
func (l *List) Open(ctx *w.Ctx, host Host, role form.FontRole) {
	rowH := ctx.Sizes.Row
	pointerX, pointerY := host.Pointer()
	width, height := host.Size()
	x, y := logic.Place(pointerX, pointerY, width, height, rowH)
	l.open = &logic.Opened{Role: role, X: x, Y: y}
	l.Search = NewTextField("")
	l.Search.Focused = true
	selected := logic.Selected(l.names(), form.SavedFont(host.Settings(), role))
	rows := len(l.Families) + 1 // Built-in comes first
	l.scroll.Y = logic.OpenScroll(selected, rows, rowH, logic.ListH)
}

// Close closes the list.
func (l *List) Close() { l.open = nil }

// IsOpen reports whether the list is open.
func (l *List) IsOpen() bool { return l.open != nil }

// Role is the setting the open list picks a family for; false while closed.
func (l *List) Role() (role form.FontRole, open bool) {
	if l.open == nil {
		return 0, false
	}
	return l.open.Role, true
}

// Fresh closes the list and clears the search, as closing the dialog does;
// the fonts already read stay.
func (l *List) Fresh() {
	l.Search = NewTextField("")
	l.open = nil
	l.scroll = ScrollState{}
}

// Pick applies the family called name, "" for the built-in font, for the role
// the list was opened for, and closes it.
func (l *List) Pick(host Host, name string) {
	if l.open == nil {
		return
	}
	role := l.open.Role
	l.Close()
	l.Set(host, role, name)
}

// FacesLoaded is how many font files the list has tried to read.
func (l *List) FacesLoaded() int { return len(l.faces) }

// names are the family names in list order.
func (l *List) names() []string {
	names := make([]string, len(l.Families))
	for i, family := range l.Families {
		names[i] = family.Name
	}
	return names
}

// face is the font of family, read the first time it is asked for; nil
// when the file cannot be read, which draws the name in the interface font.
func (l *List) face(host Host, family fonts.Family) *text.GoTextFaceSource {
	if source, seen := l.faces[family.Regular]; seen {
		return source
	}
	if l.faces == nil {
		l.faces = map[string]*text.GoTextFaceSource{}
	}
	var source *text.GoTextFaceSource
	if regular, _, err := host.FontStore().Load(fonts.Family{Name: family.Name, Regular: family.Regular}); err == nil {
		source, _ = text.NewGoTextFaceSource(bytes.NewReader(regular))
	}
	l.faces[family.Regular] = source
	return source
}

// matching are the families whose name contains the search text.
func (l *List) matching() []fonts.Family {
	return fonts.Matching(l.Families, l.Search.Text, len(l.Families))
}
