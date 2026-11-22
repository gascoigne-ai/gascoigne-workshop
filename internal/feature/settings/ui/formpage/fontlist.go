package formpage

import (
	"bytes"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The size of the font list: its width, the height of the scrolling part
// the chevron of the font control; a row is one Row high.
const (
	fontListW   = 320
	fontListH   = 240
	chevronW    = 12 // the chevron of the font control
	fontListGap = 8  // kept free between the list and the window edge
)

// builtIn is the name the list shows for the embedded font.
const builtIn = "Built-in"

// fontList is the state of the open font dropdown.
type fontList struct {
	open   bool
	role   form.FontRole
	x, y   int
	scroll ScrollState
	faces  map[string]*text.GoTextFaceSource // by file; nil when the file could not be read
}

// OpenFontList opens the dropdown for role below the pointer.
func (p *Page) OpenFontList(ctx *w.Ctx, host Host, role form.FontRole) {
	rowH := ctx.Sizes.Row
	px, py := host.Pointer()
	width, height := host.Size()
	p.list.open, p.list.role = true, role
	p.list.x = min(max(px-fontListW/2, fontListGap), width-fontListW-fontListGap)
	p.list.y = min(py+16, height-fontListH-rowH-fontListGap)
	p.FontSearch = NewTextField("")
	p.FontSearch.Focused = true
	// open on the family in use, centered, so it is in view at once
	rows := len(p.Families) + 1 // Built-in comes first
	centered := p.selectedRow(host)*rowH - (fontListH-rowH)/2
	p.list.scroll.Y = min(max(centered, 0), max(rows*rowH-fontListH, 0))
}

// selectedRow is the index of the family in use among the unfiltered rows:
// 0 for Built-in, or for a name that is not installed.
func (p *Page) selectedRow(host Host) int {
	current := form.SavedFont(host.Settings(), p.list.role)
	for i, family := range p.Families {
		if strings.EqualFold(family.Name, current) {
			return i + 1
		}
	}
	return 0
}

// CloseFontList closes the dropdown.
func (p *Page) CloseFontList() { p.list.open = false }

// FontListOpen reports whether the dropdown is open.
func (p *Page) FontListOpen() bool { return p.list.open }

// PickFont applies the family called name, "" for the built-in font, for
// the role the dropdown was opened for, and closes it.
func (p *Page) PickFont(host Host, name string) {
	p.CloseFontList()
	p.SetFont(host, p.list.role, name)
}

// loadedFaces is how many font files the list has tried to read.
func (p *Page) loadedFaces() int { return len(p.list.faces) }

// face is the font of family, read the first time it is asked for; nil
// when the file cannot be read, which draws the name in the interface font.
func (p *Page) face(host Host, family settings.Family) *text.GoTextFaceSource {
	if source, seen := p.list.faces[family.Regular]; seen {
		return source
	}
	if p.list.faces == nil {
		p.list.faces = map[string]*text.GoTextFaceSource{}
	}
	var source *text.GoTextFaceSource
	if regular, _, err := host.FontStore().Load(settings.Family{Name: family.Name, Regular: family.Regular}); err == nil {
		source, _ = text.NewGoTextFaceSource(bytes.NewReader(regular))
	}
	p.list.faces[family.Regular] = source
	return source
}

// fontControl is the control of a font setting: the current family, drawn
// in its font, and a chevron; clicking opens the dropdown.
func (p *Page) fontControl(ctx *w.Ctx, host Host, role form.FontRole) Node {
	name := form.SavedFont(host.Settings(), role)
	family, found := settings.Find(p.Families, name)
	current := fontChoice{Name: builtIn, Muted: true}
	switch {
	case found:
		current = fontChoice{Name: family.Name, Face: p.face(host, family), Ellipsis: true}
	case name != "":
		current = fontChoice{Name: name + " (not installed)", Muted: true, Ellipsis: true}
	}
	return fontControlTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"current":  current,
		"open":     func() { p.OpenFontList(ctx, host, role) },
		"chevronW": chevronW,
	}))
}

// matchingFamilies are the families whose name contains the search text.
func (p *Page) matchingFamilies() []settings.Family {
	return settings.Matching(p.Families, p.FontSearch.Text, len(p.Families))
}

// FontList is the dropdown while it is open: a search field and the
// families, each in its own font. Only the rows in view are built, so only
// their fonts are read.
func (p *Page) FontList(ctx *w.Ctx, host Host) Node {
	if !p.list.open {
		return nil
	}
	rowH := ctx.Sizes.Row
	current := form.SavedFont(host.Settings(), p.list.role)
	families := p.matchingFamilies()
	showBuiltIn := strings.TrimSpace(p.FontSearch.Text) == ""
	count := len(families)
	if showBuiltIn {
		count++
	}
	first := min(max(p.list.scroll.Y/rowH, 0), max(count-1, 0))
	last := min(first+fontListH/rowH+2, count)
	rows := make([]Node, 0, last-first)
	for i := first; i < last; i++ {
		if showBuiltIn && i == 0 {
			rows = append(rows, p.fontRow(ctx, host, fontChoice{Name: builtIn, Muted: true}, current == "", ""))
			continue
		}
		index := i
		if showBuiltIn {
			index = i - 1 // the first row is Built-in
		}
		family := families[index]
		choice := fontChoice{Name: family.Name, Face: p.face(host, family), Ellipsis: true}
		rows = append(rows, p.fontRow(ctx, host, choice, strings.EqualFold(family.Name, current), family.Name))
	}
	note := ""
	if len(p.Families) == 0 {
		note = "Looking for installed fonts..."
	} else if count == 0 {
		note = "No font matches"
	}
	return fontListTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"width":         fontListW,
		"height":        fontListH,
		"x":             p.list.x,
		"y":             p.list.y,
		"close":         p.CloseFontList,
		"search":        &p.FontSearch,
		"searchChanged": func() { p.list.scroll.Y = 0 },
		"note":          note,
		"scroll":        &p.list.scroll,
		"above":         first * rowH,
		"rows":          rows,
		"below":         (count - last) * rowH,
	}))
}

// fontChoice is what a template shows of one family: a row of the list, or
// the family in use in the font control.
type fontChoice struct {
	Name     string
	Face     *text.GoTextFaceSource // nil draws the name in the interface font
	Muted    bool                   // Built-in, which has no font of its own, or a family not installed
	Ellipsis bool                   // a family name, which can be long
	Pick     func()                 // the list's rows only
}

var (
	fontControlTemplate = markup.Must("fontcontrol.vml", fontcontrolSource)
	fontListTemplate    = markup.Must("fontlist.vml", fontlistSource)
	fontRowTemplate     = markup.Must("fontrow.vml", fontrowSource)
)

// fontRow is one row of the list; picking it applies the family called
// name, "" for Built-in.
func (p *Page) fontRow(ctx *w.Ctx, host Host, choice fontChoice, selected bool, name string) Node {
	choice.Pick = func() { p.PickFont(host, name) }
	return fontRowTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{"row": choice, "selected": selected}))
}

// Templates, parsed once above.
const (
	fontcontrolSource = `<row h="{{rowH}}" bg="sunken" control="field" pad-x="{{looseGap}}" gap="{{tightGap}}" align-items="center"
     on-click="open" tooltip="Pick an installed font">
  <text grow="1" muted="{{current.Muted}}" ellipsis="{{current.Ellipsis}}" font-source="current.Face">{{current.Name}}</text>
  <icon name="chevron-down" w="{{chevronW}}"/>
</row>
`

	fontlistSource = `<col w="{{width}}" pad="{{tightGap}}" gap="{{tightGap}}" bg="bg" border="border" control="menu" shadow float-at="{{x}} {{y}}" modal focus-scope on-dismiss="close">
  <field state="search" h="{{rowH}}" bg="sunken" placeholder="Search fonts" on-change="searchChanged" text-menu="search"/>
  <row if="note" h="{{rowH}}" pad-x="{{looseGap}}">
    <text small muted>{{note}}</text>
  </row>
  <scroll state="scroll" h="{{height}}">
    <col>
      <spacer h="{{above}}"/>
      <slot for="row in rows" name="row"/>
      <spacer h="{{below}}"/>
    </col>
  </scroll>
</col>
`

	fontrowSource = `<list-row selected="{{selected}}" h="{{rowH}}" pad-x="{{looseGap}}" align-items="center" on-click="row.Pick">
  <text grow="1" muted="{{row.Muted}}" ellipsis="{{row.Ellipsis}}" font-source="row.Face">{{row.Name}}</text>
</list-row>
`
)
