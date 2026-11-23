package filebrowser

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	browsing "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/filebrowser"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// browserKey is a key of the browser and what it does; when says whether the key
// is the browser's this frame (Backspace is the field's while a name is
// typed).
type browserKey struct {
	key  ebiten.Key
	edit w.Edit
	when func(in w.Input, model browsing.Model) bool
	msg  browsing.Msg
}

var keys = []browserKey{
	{ebiten.KeyArrowUp, w.EditUp, func(in w.Input, _ browsing.Model) bool { return in.Alt }, browsing.WentUp{}},
	{ebiten.KeyArrowUp, w.EditUp, func(in w.Input, _ browsing.Model) bool { return !in.Alt }, browsing.Moved{By: -1}},
	{ebiten.KeyArrowDown, w.EditDown, func(w.Input, browsing.Model) bool { return true }, browsing.Moved{By: 1}},
	{ebiten.KeyBackspace, w.EditBackspace, func(_ w.Input, model browsing.Model) bool { return model.AtFolder() }, browsing.WentUp{}},
}

// isBrowsers reports whether the browser takes key this frame: a field with
// selected text keeps its Backspace, which deletes the selection.
func (open *session) isBrowsers(binding browserKey, in w.Input) bool {
	if binding.key == ebiten.KeyBackspace && open.path.HasSel {
		return false
	}
	return slices.Contains(in.Edits, binding.edit) && binding.when(in, open.model)
}

// Keys handles the browser's keys and takes them from the frame's input, so
// the path field under them does not act on them too. It does nothing while
// a question is asked over the browser, which has the keys then. Escape is
// the window's, which closes the topmost layer.
func (c *Component) Keys(ctx *w.Ctx, in w.Input) {
	if c.session == nil || c.deps.Dialogs.Asking() {
		return
	}
	if in.Enter {
		ctx.In.Consume(ebiten.KeyEnter)
		c.update(browsing.Entered{})
		return
	}
	for _, binding := range keys {
		if c.session.isBrowsers(binding, in) {
			ctx.In.Consume(binding.key)
			c.update(binding.msg)
			return
		}
	}
}
