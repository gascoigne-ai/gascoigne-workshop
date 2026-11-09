package ui

import (
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// pageHost is the App as the Colors and Shortcuts pages of the settings
// dialog see it; the rest of their hosts (Settings) is the App's own.
type pageHost struct{ *App }

func (host pageHost) Edit(edit func(*settings.Settings)) { host.saveSettings(edit) }

func (host pageHost) Color(tok w.Token) color.RGBA { return host.ctx.Th.Color(tok) }

func (host pageHost) TextMenu(field *TextField, changed func()) Option {
	return host.textMenuOpt(field, changed)
}

func (host pageHost) Actions() []shortcutpage.Action {
	actions := make([]shortcutpage.Action, len(shortcuts))
	for i, s := range shortcuts {
		actions[i] = s.action()
	}
	return actions
}
