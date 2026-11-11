package widget

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// TextMenu gives a Field or Area the right-click menu of a text field:
// Undo, Redo, Cut, Copy, Paste and Select all, through the context's
// clipboard. After an edit the element's own OnChange runs, so a model
// that follows the text stays in step. Secrets never reach the clipboard:
// Cut and Copy are disabled on a masked field. Pass the field the element
// edits (for an Area, its TextField).
func TextMenu(field *TextField) Option {
	return func(e *El) {
		OnRightClickIn(func(c *core.Ctx, x, y int) {
			OpenMenu(c, x, y, textMenuItems(c.Clipboard, field, e.ChangeHandler()))
		})(e)
	}
}

func textMenuItems(clipboard core.Clipboard, field *TextField, changed func()) []MenuItem {
	done := func() {
		if changed != nil {
			changed()
		}
	}
	hasSelection := field.SelectedText() != "" && !field.Masked
	return []MenuItem{
		{Label: "Undo", Shortcut: "Ctrl+Z", Icon: "undo", Disabled: !field.CanUndo(), Run: func() {
			if field.Undo() {
				done()
			}
		}},
		{Label: "Redo", Shortcut: "Ctrl+Y", Icon: "redo", Disabled: !field.CanRedo(), Run: func() {
			if field.Redo() {
				done()
			}
		}},
		{Separator: true},
		{Label: "Cut", Shortcut: "Ctrl+X", Icon: "scissors", Disabled: !hasSelection, Run: func() {
			_ = clipboard.Write(field.SelectedText())
			if field.DeleteSelection() {
				done()
			}
		}},
		{Label: "Copy", Shortcut: "Ctrl+C", Icon: "copy", Disabled: !hasSelection, Run: func() { _ = clipboard.Write(field.SelectedText()) }},
		{Label: "Paste", Shortcut: "Ctrl+V", Icon: "clipboard", Run: func() {
			if text, err := clipboard.Read(); err == nil && field.Insert(text) {
				done()
			}
		}},
		{Separator: true},
		{Label: "Select all", Shortcut: "Ctrl+A", Disabled: field.Text == "", Run: field.SelectAll},
	}
}
