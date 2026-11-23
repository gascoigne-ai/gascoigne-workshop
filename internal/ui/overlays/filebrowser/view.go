package filebrowser

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	browsing "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The size of the browser.
const (
	browserW      = 640
	browserMaxH   = 520
	browserMargin = 40
	browserIconW  = 16
)

// browserFootH is the footer's buttons and its padding.
func browserFootH(faces w.Sizes) int { return faces.Row + 2*w.Gutter }

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

// View is the browser while it is open, nothing otherwise.
func (c *Component) View(ctx *w.Ctx) Node {
	open := c.session
	if open == nil {
		return nil
	}
	if !c.deps.Dialogs.Asking() { // the field keeps the keyboard, unless a question is asked over it
		open.path.Focused = true
	}
	model := open.model
	_, windowH := c.deps.Window.Size()
	height := min(browserMaxH, windowH-2*browserMargin)
	faces := ctx.Sizes
	listH := height - faces.DialogTitle - faces.DialogToolbar - browserFootH(faces)
	visible := model.Visible()
	first, last := shownRows(open.scroll.Y, listH, faces.Row, len(visible))
	rows := make([]Node, 0, last-first)
	for i := first; i < last; i++ {
		rows = append(rows, c.row(ctx, i, visible[i], i == model.Selected))
	}
	primary := "Open"
	if model.Mode == browsing.Save {
		primary = "Save"
	}
	return sizes.Build(ctx, browserTemplate, markup.Scope{
		"width":       browserW,
		"height":      height,
		"move":        &open.move,
		"close":       c.Close,
		"title":       open.title,
		"subtitle":    DialogSubtitle(model.Count()),
		"path":        &open.path,
		"typed":       c.typed,
		"up":          func() { c.update(browsing.WentUp{}) },
		"hidden":      model.ShowHidden,
		"toggle":      func() { c.update(browsing.HiddenToggled{}) },
		"scroll":      &open.scroll,
		"above":       first * faces.Row,
		"rows":        rows,
		"below":       (len(visible) - last) * faces.Row,
		"empty":       len(visible) == 0,
		"problem":     model.Problem,
		"hint":        c.hint(),
		"primary":     primary,
		"enter":       func() { c.update(browsing.Entered{}) },
		"browserFoot": browserFootH(faces),
	})
}

// hint names the program that would give native dialogs, so the user learns
// that the browser is a fallback; "" when there is none to name.
func (c *Component) hint() string {
	program := c.deps.Desktop.Suggested()
	if program == "" {
		return ""
	}
	return "For native dialogs install " + program
}

// typed follows an edit of the path field; the list goes back to its top,
// where the new first match is.
func (c *Component) typed() {
	c.session.scroll.Y = 0
	c.update(browsing.Typed{Text: c.session.path.Text})
}

// openRow selects a row and takes it, as Enter would.
func (c *Component) openRow(index int) {
	c.update(browsing.Picked{Index: index})
	c.update(browsing.Entered{})
}

// shownRows are the rows built for a list scrolled to scrollY: those in
// view and one more at each end, so Up and Down can bring the next row into
// view. Only these are built, since a folder may hold thousands of names.
func shownRows(scrollY, listH, rowH, count int) (first, last int) {
	first = min(max(scrollY/rowH-1, 0), count)
	last = min(first+listH/rowH+3, count)
	return first, last
}

// row is one entry of the list: a click selects it, a double click opens it.
func (c *Component) row(ctx *w.Ctx, index int, entry workspace.FolderEntry, selected bool) Node {
	return sizes.Build(ctx, browserRowTemplate, markup.Scope{
		"row": browserRow{
			Name:     entry.Name,
			Dir:      entry.Dir,
			Selected: selected,
			Pick:     func() { c.update(browsing.Picked{Index: index}) },
			Open:     func() { c.openRow(index) },
		},
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
  <row h="{{browserFoot}}" pad-x="{{dialogPad}}" gap="{{looseGap}}" align-items="center" border-t="border-soft">
    <text if="problem" small color="error" ellipsis grow="1" shrink>{{problem}}</text>
    <text if="!problem" small muted ellipsis grow="1" shrink>{{hint}}</text>
    <segment-chip selected="false" min-w="{{dialogButtonMinW}}" h="{{rowH}}" on-click="close">Cancel</segment-chip>
    <segment-chip selected="true" min-w="{{dialogButtonMinW}}" h="{{rowH}}" on-click="enter">{{primary}}</segment-chip>
  </row>
</dialog>
`

	filebrowserrowSource = `<list-row selected="{{row.Selected}}" h="{{rowH}}" pad-x="{{gutter}}" gap="{{looseGap}}" align-items="center" keep-in-view="{{row.Selected}}" on-click="row.Pick" on-double-click="row.Open">
  <icon if="row.Dir" name="folder" w="{{iconW}}" muted/>
  <spacer if="!row.Dir" w="{{iconW}}"/>
  <text ellipsis grow="1">{{row.Name}}</text>
</list-row>
`
)
