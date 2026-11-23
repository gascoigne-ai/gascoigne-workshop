package overlays

import (
	"cmp"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/menus"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

var dialogTemplate = markup.Must("dialog.vml", dialogSource)

// Views are the overlays of the window, one node per slot of its template.
// A slot of a closed overlay is nil, which the template leaves out.
type Views struct {
	Settings, Picker, Browser, Popup, Dialog, CallLog, Toasts Node
}

// Views builds the overlays, in the order the window stacks them. Each
// slot is its own node, not one fragment: the window keeps its slot names.
func (c *Component) Views(ctx *w.Ctx) Views {
	return Views{
		Settings: c.Settings.View(ctx),
		Picker:   c.pickerView(ctx),
		Browser:  c.Browser.View(ctx),
		Popup:    menus.View(ctx),
		Dialog:   c.dialogView(ctx),
		CallLog:  c.calls.View(ctx, windowSize{c}),
		Toasts:   c.Toasts.View(),
	}
}

// windowSize is the window as the call trace asks for it.
type windowSize struct{ *Component }

// Size is the window's size in logical pixels.
func (host windowSize) Size() (int, int) { return host.deps.Window.Size() }

// pickerView is the destination picker while it is open, nothing otherwise.
func (c *Component) pickerView(ctx *w.Ctx) Node {
	if c.picker == nil {
		return nil
	}
	return c.picker.View(ctx, c.pickerHost)
}

// dialogView is the open dialog, nothing when none is open: a template
// always builds its element, so Go decides whether there is one.
func (c *Component) dialogView(ctx *w.Ctx) Node {
	open := c.dialog
	if open == nil {
		return nil
	}
	return dialogTemplate.MustBuild(markup.Scope{
		"width":   cmp.Or(open.width, w.DialogW),
		"state":   &open.DialogState,
		"move":    &open.move,
		"dismiss": c.CloseDialog,
		"form":    dialogForm(ctx, open),
	})
}

// dialogForm is the content of a dialog, nothing for the others.
func dialogForm(ctx *w.Ctx, open *dialog) Node {
	if open.content == nil {
		return nil
	}
	return open.content(ctx)
}

// Template, parsed once above.
const dialogSource = `<!-- A dialog's own content, an MCP tool's typed form for one, sits between its body and its input. -->
<message-dialog w="{{width}}" state="state" moved="move" on-dismiss="dismiss">
  <slot if="form" name="form"/>
</message-dialog>
`
