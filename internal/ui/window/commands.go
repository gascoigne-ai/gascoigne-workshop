package window

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

func (a *Component) newRequest() { a.newRequestIn(a.selectedFolder()) }

// commands are the shortcut actions in the order the Shortcuts page lists
// them: each component offers its own, and the window offers the ones that
// reach across components (they read a table cell edit, the text focus or the
// focused collection). A command that moves to a component keeps its place.
func (a *Component) commands() []keymap.Command {
	ctrl := func(key ebiten.Key) keymap.Chord { return keymap.Chord{Key: int(key), Ctrl: true} }
	return slices.Concat(
		[]keymap.Command{
			{ID: "send", Name: "Send request", Default: ctrl(ebiten.KeyEnter), Run: a.sendCurrent},
			{ID: "focus-url", Name: "Focus URL", Default: ctrl(ebiten.KeyL), Run: a.focusURL},
		},
		a.strip.Commands(),
		a.editor.Commands(),
		[]keymap.Command{
			{ID: "delete-row", Name: "Delete row", Default: keymap.Chord{Key: int(ebiten.KeyDelete)}, Run: a.deleteRow},
			{ID: "complete-variable", Name: "Complete variable", Default: ctrl(ebiten.KeySpace), Run: a.completeVariable},
			{ID: "save", Name: "Save collection", Default: ctrl(ebiten.KeyS), Run: a.saveWorkspace},
		},
		a.side.Commands(),
		a.resp.Commands(),
		a.over.Settings.Commands(),
	)
}

// keymap is the binding of every command under the settings in effect. It is
// derived where it is read, never kept, so a changed setting needs no
// invalidation.
func (a *Component) keymap() keymap.Keymap {
	return keymap.Resolve(a.settings.Shortcuts, a.commands(), shortcutpage.ParseChord)
}

// runKeymap runs the commands whose chord went down this frame.
func (a *Component) runKeymap(in core.Input) {
	keys := a.keymap()
	for _, key := range in.Keys {
		binding := engine.Binding{Key: key, Ctrl: in.Ctrl, Shift: in.Shift, Alt: in.Alt, Bound: true}
		chord, _ := shortcutpage.ChordOf(binding)
		keys.Run(keymap.Press{Chord: chord, TypesText: binding.TypesText()}, a.typing())
	}
}

// sendCurrent sends the request; while a table cell is edited, Ctrl+Enter
// adds a row below instead.
func (a *Component) sendCurrent() {
	if a.editor.Tables.Editing() {
		a.editor.Tables.InsertRowBelow()
		return
	}
	a.editor.URLBar.Send()
}

// focusURL puts the keyboard in the URL field with its text selected.
func (a *Component) focusURL() {
	a.commitEdit()
	a.editor.URLBar.FocusURL()
}
