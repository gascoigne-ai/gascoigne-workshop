package fontlist

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// chevronW is the width of the chevron of the font control.
const chevronW = 12

// choice is what a template shows of one family: a row of the list, or the
// family in use in the font control.
type choice struct {
	Name     string
	Face     *text.GoTextFaceSource // nil draws the name in the interface font
	Muted    bool                   // Built-in, which has no font of its own, or a family not installed
	Ellipsis bool                   // a family name, which can be long
	Pick     func()                 // the list's rows only
}

var (
	controlTemplate = markup.Must("fontcontrol.vml", controlSource)
	listTemplate    = markup.Must("fontlist.vml", listSource)
	rowTemplate     = markup.Must("fontrow.vml", rowSource)
)

// Control is the control of a font setting: the current family, drawn in its
// font, and a chevron; clicking opens the list.
func (l *List) Control(ctx *w.Ctx, host Host, role form.FontRole) Node {
	name := form.SavedFont(host.Settings(), role)
	family, found := fonts.Find(l.Families, name)
	current := choice{Name: builtIn, Muted: true}
	switch {
	case found:
		current = choice{Name: family.Name, Face: l.face(host, family), Ellipsis: true}
	case name != "":
		current = choice{Name: name + " (not installed)", Muted: true, Ellipsis: true}
	}
	return sizes.Build(ctx, controlTemplate, markup.Scope{
		"current":  current,
		"open":     func() { l.Open(ctx, host, role) },
		"chevronW": chevronW,
	})
}

// View is the list while it is open: a search field and the families, each in
// its own font; nil while closed. Only the rows in view are built, so only
// their fonts are read.
func (l *List) View(ctx *w.Ctx, host Host) Node {
	if l.open == nil {
		return nil
	}
	rowH := ctx.Sizes.Row
	current := form.SavedFont(host.Settings(), l.open.Role)
	families := l.matching()
	showBuiltIn := logic.ShowsBuiltIn(l.Search.Text)
	count := len(families)
	if showBuiltIn {
		count++
	}
	first, last := logic.Visible(l.scroll.Y, rowH, count, logic.ListH)
	rows := make([]Node, 0, last-first)
	for i := first; i < last; i++ {
		if showBuiltIn && i == 0 {
			rows = append(rows, l.row(ctx, host, choice{Name: builtIn, Muted: true}, current == "", ""))
			continue
		}
		index := i
		if showBuiltIn {
			index = i - 1 // the first row is Built-in
		}
		family := families[index]
		shown := choice{Name: family.Name, Face: l.face(host, family), Ellipsis: true}
		rows = append(rows, l.row(ctx, host, shown, strings.EqualFold(family.Name, current), family.Name))
	}
	return sizes.Build(ctx, listTemplate, markup.Scope{
		"width":         logic.ListW,
		"height":        logic.ListH,
		"x":             l.open.X,
		"y":             l.open.Y,
		"close":         l.Close,
		"search":        &l.Search,
		"searchChanged": func() { l.scroll.Y = 0 },
		"note":          logic.Note(len(l.Families), count),
		"scroll":        &l.scroll,
		"above":         first * rowH,
		"rows":          rows,
		"below":         (count - last) * rowH,
	})
}

// row is one row of the list; picking it applies the family called name, ""
// for Built-in.
func (l *List) row(ctx *w.Ctx, host Host, shown choice, selected bool, name string) Node {
	shown.Pick = func() { l.Pick(host, name) }
	return sizes.Build(ctx, rowTemplate, markup.Scope{"row": shown, "selected": selected})
}

// Templates, parsed once above.
const (
	controlSource = `<row h="{{rowH}}" bg="sunken" control="field" pad-x="{{looseGap}}" gap="{{tightGap}}" align-items="center"
     on-click="open" tooltip="Pick an installed font">
  <text grow="1" muted="{{current.Muted}}" ellipsis="{{current.Ellipsis}}" font-source="current.Face">{{current.Name}}</text>
  <icon name="chevron-down" w="{{chevronW}}"/>
</row>
`

	listSource = `<col w="{{width}}" pad="{{tightGap}}" gap="{{tightGap}}" bg="bg" border="border" control="menu" shadow float-at="{{x}} {{y}}" modal focus-scope on-dismiss="close">
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

	rowSource = `<list-row selected="{{selected}}" h="{{rowH}}" pad-x="{{looseGap}}" align-items="center" on-click="row.Pick">
  <text grow="1" muted="{{row.Muted}}" ellipsis="{{row.Ellipsis}}" font-source="row.Face">{{row.Name}}</text>
</list-row>
`
)
