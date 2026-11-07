package settingsui

import (
	"bytes"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/fonts"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The size of the font list: its width, the height of the scrolling part
// and of a row.
const (
	fontListW    = 320
	fontListH    = 240
	fontListRowH = 28
	fontListGap  = 8 // kept free between the list and the window edge
)

// builtIn is the name the list shows for the embedded font.
const builtIn = "Built-in"

// fontList is the state of the open font dropdown.
type fontList struct {
	open   bool
	role   FontRole
	x, y   int
	scroll ScrollState
	faces  map[string]*text.GoTextFaceSource // by file; nil when the file could not be read
}

// OpenFontList opens the dropdown for role below the pointer.
func (d *Dialog) OpenFontList(host Host, role FontRole) {
	px, py := host.Pointer()
	width, height := host.Size()
	d.list.open, d.list.role = true, role
	d.list.x = min(max(px-fontListW/2, fontListGap), width-fontListW-fontListGap)
	d.list.y = min(py+16, height-fontListH-fontListRowH-fontListGap)
	d.list.scroll.Y = 0
	d.FontSearch = NewTextField("")
	d.FontSearch.Focused = true
}

// CloseFontList closes the dropdown.
func (d *Dialog) CloseFontList() { d.list.open = false }

// FontListOpen reports whether the dropdown is open.
func (d *Dialog) FontListOpen() bool { return d.list.open }

// PickFont applies the family called name, "" for the built-in font, for
// the role the dropdown was opened for, and closes it.
func (d *Dialog) PickFont(host Host, name string) {
	d.CloseFontList()
	d.SetFont(host, d.list.role, name)
}

// loadedFaces is how many font files the list has tried to read.
func (d *Dialog) loadedFaces() int { return len(d.list.faces) }

// face is the font of family, read the first time it is asked for; nil
// when the file cannot be read, which draws the name in the interface font.
func (d *Dialog) face(family fonts.Family) *text.GoTextFaceSource {
	if source, seen := d.list.faces[family.Regular]; seen {
		return source
	}
	if d.list.faces == nil {
		d.list.faces = map[string]*text.GoTextFaceSource{}
	}
	var source *text.GoTextFaceSource
	if regular, _, err := fonts.Load(fonts.Family{Name: family.Name, Regular: family.Regular}); err == nil {
		source, _ = text.NewGoTextFaceSource(bytes.NewReader(regular))
	}
	d.list.faces[family.Regular] = source
	return source
}

// fontLabel is a name shown in its own font: Text with the family's font
// when it could be loaded.
func (d *Dialog) fontLabel(family fonts.Family, opts ...Option) Node {
	options := append([]Option{Ellipsis(), Grow(1)}, opts...)
	if source := d.face(family); source != nil {
		options = append(options, FontSource(source))
	}
	return Text(family.Name, options...)
}

// fontControl is the control of a font setting: the current family, drawn
// in its font, and a chevron; clicking opens the dropdown.
func (d *Dialog) fontControl(host Host, role FontRole) Node {
	name := savedFont(host.Settings(), role)
	family, found := fonts.Find(d.Families, name)
	label := Text(builtIn, Grow(1), Muted())
	switch {
	case found:
		label = d.fontLabel(family)
	case name != "":
		label = Text(name+" (not installed)", Ellipsis(), Grow(1), Muted())
	}
	return Row(H(w.RowH), Bg(w.Sunken), Control("field"), PadX(8), Gap(6), AlignItems(AlignCenter),
		OnClick(func() { d.OpenFontList(host, role) }), Tooltip("Pick an installed font"),
		label, ChevronDownIcon(W(12)),
	)
}

// matchingFamilies are the families whose name contains the search text.
func (d *Dialog) matchingFamilies() []fonts.Family {
	return fonts.Matching(d.Families, d.FontSearch.Text, len(d.Families))
}

// FontList is the dropdown while it is open: a search field and the
// families, each in its own font. Only the rows in view are built, so only
// their fonts are read.
func (d *Dialog) FontList(host Host) Node {
	if !d.list.open {
		return nil
	}
	current := savedFont(host.Settings(), d.list.role)
	families := d.matchingFamilies()
	showBuiltIn := strings.TrimSpace(d.FontSearch.Text) == ""
	count := len(families)
	if showBuiltIn {
		count++
	}
	first := min(max(d.list.scroll.Y/fontListRowH, 0), max(count-1, 0))
	last := min(first+fontListH/fontListRowH+2, count)
	var rows []Item
	rows = append(rows, Spacer(H(first*fontListRowH)))
	for i := first; i < last; i++ {
		if showBuiltIn && i == 0 {
			rows = append(rows, d.fontRow(host, Text(builtIn, Grow(1), Muted()), current == "", ""))
			continue
		}
		index := i
		if showBuiltIn {
			index = i - 1 // the first row is Built-in
		}
		family := families[index]
		rows = append(rows, d.fontRow(host, d.fontLabel(family), strings.EqualFold(family.Name, current), family.Name))
	}
	rows = append(rows, Spacer(H((count-last)*fontListRowH)))
	var note Node
	if len(d.Families) == 0 {
		note = Row(H(fontListRowH), PadX(8), Text("Looking for installed fonts...", Small(), Muted()))
	} else if count == 0 {
		note = Row(H(fontListRowH), PadX(8), Text("No font matches", Small(), Muted()))
	}
	return Col(W(fontListW), Pad(6), Gap(6), Bg(w.Bg), Border(w.Border), Control("menu"), Shadow(),
		FloatAt(d.list.x, d.list.y), Modal(), FocusScope(), OnDismiss(d.CloseFontList),
		Field(&d.FontSearch, H(28), Bg(w.Sunken), Placeholder("Search fonts"), OnChange(func() { d.list.scroll.Y = 0 }), host.TextMenu(&d.FontSearch, nil)),
		note,
		Scroll(&d.list.scroll, H(fontListH), Col(rows...)),
	)
}

// fontRow is one row of the list; picking it applies the family.
func (d *Dialog) fontRow(host Host, label Node, selected bool, name string) Node {
	look := HoverBg(w.Hover)
	if selected {
		look = Bg(w.Selected)
	}
	return Row(H(fontListRowH), PadX(8), AlignItems(AlignCenter), Control("row"), look, OnClick(func() { d.PickFont(host, name) }), label)
}
