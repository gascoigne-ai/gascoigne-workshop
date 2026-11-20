// Package destpicker is the dialog that asks where a request or folder goes
// when it is moved or copied: every folder of every open collection in a
// list, some of them refused with a reason. The rules (which rows are
// refused) come from controller/transfer; this package holds the selection
// and draws the list.
package destpicker

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The size of the dialog and what its footer is made of.
const (
	dialogW    = 440
	dialogMaxH = 480
	margin     = w.DialogMargin
	footPad    = w.Gutter
	footH      = w.RowH + 2*footPad // the footer's buttons and its padding
	stripeW    = 3
	stripeH    = 14
)

// Host is what the dialog needs from the application that shows it.
type Host interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Pick is the destination the user chose; the host closes the dialog.
	Pick(destination transfer.Destination)
	// Dismiss closes the dialog without a choice.
	Dismiss()
}

// Entry is one row of the list: a destination and what the list says of it.
type Entry struct {
	transfer.Destination
	Name   string
	Stripe w.Token // the color of the collection, on its root row only
	Reason string  // why the row is refused, "" when it can be picked
}

// modeText is what the dialog says for each mode.
type modeText struct{ title, primary string }

var modeTexts = map[transfer.Mode]modeText{
	transfer.Move: {"Move to", "Move here"},
	transfer.Copy: {"Copy to", "Copy here"},
}

// reasons is the small note on a refused row.
var reasons = map[transfer.Why]string{
	transfer.WhyItself:     "itself",
	transfer.WhyInside:     "inside itself",
	transfer.WhySameParent: "current folder",
}

// Picker is the open dialog: the rows, which one is selected and where the
// user dragged the dialog.
type Picker struct {
	entries  []Entry
	selected int // index of the selected row, -1 for none
	node     string
	mode     modeText
	scroll   ScrollState
	move     MoveState
}

// New is the dialog for moving or copying node to one of destinations.
// stripe gives the color of a collection, shown on its root row.
func New(node model.Node, mode transfer.Mode, destinations []transfer.Destination, stripe func(*model.Collection) w.Token) *Picker {
	picker := &Picker{selected: -1, node: node.Name(), mode: modeTexts[mode]}
	for _, destination := range destinations {
		entry := Entry{Destination: destination, Name: destination.Folder.Name, Reason: reasons[destination.Why]}
		if destination.Depth == 0 {
			entry.Name, entry.Stripe = destination.Collection.Name, stripe(destination.Collection)
		}
		picker.entries = append(picker.entries, entry)
	}
	return picker
}

// Rows are the destinations in the order they are listed.
func (picker *Picker) Rows() []Entry { return picker.entries }

// Selected is the index of the selected row, -1 for none.
func (picker *Picker) Selected() int { return picker.selected }

// Select selects row index and reports whether it could be: a refused row
// cannot.
func (picker *Picker) Select(index int) bool {
	if index < 0 || index >= len(picker.entries) || picker.entries[index].Reason != "" {
		return false
	}
	picker.selected = index
	return true
}

// Step moves the selection by one pickable row in direction delta (-1 up,
// 1 down), skipping refused rows. With nothing selected, down goes to the
// first row and up to the last; at the end the selection stays.
func (picker *Picker) Step(delta int) {
	start := picker.selected
	if start < 0 && delta < 0 {
		start = len(picker.entries)
	}
	for index := start + delta; index >= 0 && index < len(picker.entries); index += delta {
		if picker.Select(index) {
			return
		}
	}
}

// choose tells host the selected row, if there is one.
func (picker *Picker) choose(host Host) {
	if picker.selected >= 0 {
		host.Pick(picker.entries[picker.selected].Destination)
	}
}

// Keys does what the frame's keys ask: Up and Down move the selection,
// Enter picks. Escape is the overlay's, not the dialog's.
func (picker *Picker) Keys(in w.Input, host Host) {
	for _, edit := range in.Edits {
		switch edit {
		case w.EditUp:
			picker.Step(-1)
		case w.EditDown:
			picker.Step(1)
		}
	}
	if in.Enter {
		picker.choose(host)
	}
}

var (
	dialogTemplate = markup.Must("destpicker.vml", dialogSource)
	rowTemplate    = markup.Must("destpickerrow.vml", rowSource)
)

// rowItem is what the row template shows of one entry.
type rowItem struct {
	Name     string
	Indent   int
	Root     bool
	Stripe   w.Token
	Reason   string
	Selected bool
	Pick     func()
	Open     func()
}

// View is the dialog.
func (picker *Picker) View(host Host) Node {
	_, windowH := host.Size()
	height := min(dialogMaxH, windowH-2*margin)
	listH := height - w.DialogTitleH - footH
	first, last := shownRows(picker.scroll.Y, listH, len(picker.entries))
	rows := make([]Node, 0, last-first)
	for index := first; index < last; index++ {
		rows = append(rows, picker.row(index, host))
	}
	return dialogTemplate.MustBuild(markup.Scope{
		"width":    dialogW,
		"height":   height,
		"move":     &picker.move,
		"dismiss":  host.Dismiss,
		"title":    picker.mode.title,
		"subtitle": DialogSubtitle(picker.node),
		"scroll":   &picker.scroll,
		"above":    first * w.RowH,
		"rows":     rows,
		"below":    (len(picker.entries) - last) * w.RowH,
		"foot":     footH,
		"looseGap": w.LooseGap,
		"rowH":     w.RowH,
		"canPick":  picker.selected >= 0,
		"primary":  picker.mode.primary,
		"pick":     func() { picker.choose(host) },
	})
}

func (picker *Picker) row(index int, host Host) Node {
	entry := picker.entries[index]
	item := rowItem{
		Name:     entry.Name,
		Indent:   w.Gutter * (1 + entry.Depth),
		Root:     entry.Depth == 0,
		Stripe:   entry.Stripe,
		Reason:   entry.Reason,
		Selected: index == picker.selected,
		Pick:     func() { picker.Select(index) },
		Open: func() {
			if picker.Select(index) {
				picker.choose(host)
			}
		},
	}
	return rowTemplate.MustBuild(markup.Scope{"row": item, "rowH": w.RowH, "gutter": w.Gutter, "tightGap": w.TightGap, "stripeW": stripeW, "stripeH": stripeH})
}

// shownRows are the rows built for a list scrolled to scrollY: those in
// view and one more at each end, so Up and Down can bring the next row into
// view.
func shownRows(scrollY, listH, count int) (first, last int) {
	first = min(max(scrollY/w.RowH-1, 0), count)
	last = min(first+listH/w.RowH+3, count)
	return first, last
}

// Templates, parsed once above.
const (
	dialogSource = `<!-- The destination picker: the folders of every open collection in a list, the
     footer says what is not changed and holds the buttons. -->
<dialog w="{{width}}" h="{{height}}" moved="move" on-dismiss="dismiss">
  <dialog-title title="{{title}}" on-dismiss="dismiss">
    <slot name="subtitle"/>
  </dialog-title>
  <scroll state="scroll" grow="1" bg="sunken">
    <col>
      <spacer h="{{above}}"/>
      <slot for="row in rows" name="row"/>
      <spacer h="{{below}}"/>
    </col>
  </scroll>
  <row h="{{foot}}" pad-x="16" gap="{{looseGap}}" align-items="center" border-t="border-soft">
    <text small muted ellipsis grow="1" shrink>Variables are not changed</text>
    <segment-chip selected="false" min-w="80" h="{{rowH}}" on-click="dismiss">Cancel</segment-chip>
    <segment-chip if="canPick" selected="true" min-w="80" h="{{rowH}}" on-click="pick">{{primary}}</segment-chip>
    <segment-chip if="!canPick" selected="false" min-w="80" h="{{rowH}}" muted>{{primary}}</segment-chip>
  </row>
</dialog>
`

	rowSource = `<list-row selected="{{row.Selected}}" h="{{rowH}}" pad-l="{{row.Indent}}" pad-r="{{gutter}}" gap="{{tightGap}}" align-items="center" keep-in-view="{{row.Selected}}" on-click="row.Pick" on-double-click="row.Open">
  <row if="row.Stripe" w="{{stripeW}}" h="{{stripeH}}" bg="{{row.Stripe}}"/>
  <text if="!row.Reason" bold="{{row.Root}}" ellipsis grow="1">{{row.Name}}</text>
  <text if="row.Reason" bold="{{row.Root}}" muted ellipsis grow="1">{{row.Name}}</text>
  <text if="row.Reason" small muted>{{row.Reason}}</text>
</list-row>
`
)
