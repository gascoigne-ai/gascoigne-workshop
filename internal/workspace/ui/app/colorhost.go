package ui

import (
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// colorHost is the App as the Colors page sees it; the rest of colorpage.Host
// (Settings) is the App's own.
type colorHost struct{ *App }

func (host colorHost) Edit(edit func(*settings.Settings)) { host.saveSettings(edit) }

func (host colorHost) Color(tok w.Token) color.RGBA { return host.ctx.Th.Color(tok) }

func (host colorHost) TextMenu(field *TextField, changed func()) Option {
	return host.textMenuOpt(field, changed)
}
