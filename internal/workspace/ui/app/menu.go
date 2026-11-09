package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/controller/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
)

// changeSettings edits the settings, applies them and saves them. The
// color fields are rebuilt, since the colors in effect may have changed.
func (a *App) changeSettings(edit func(*settings.Settings)) {
	was := a.settings.ResponseView
	a.saveSettings(edit)
	if a.settings.ResponseView != was { // the choice applies to the tabs already open
		for _, t := range a.tabs {
			t.raw, t.compact = viewer.Flags(a.settings.ResponseView)
			a.refreshView(t)
		}
	}
	a.colors.Forget()
}

// saveSettings edits and applies the settings and, unless auto-save is off,
// writes them. Turning auto-save off writes that choice once; turning it on
// writes whatever was waiting.
func (a *App) saveSettings(edit func(*settings.Settings)) {
	wasAuto, wasNetwork := a.settings.AutoSave, a.networkSettings()
	edit(&a.settings)
	if a.networkSettings() != wasNetwork {
		a.reg = a.newRegistry() // protocols hold the proxy and certificates they were built with
	}
	if err := a.ctx.SetTheme(appearance.ThemeWith(a.settings)); err != nil {
		a.status = err.Error()
		return
	}
	if a.settingsPath == "" {
		return
	}
	if !a.settings.AutoSave && wasAuto == false {
		a.settingsUnsaved = true // applied now, written when the user saves
		return
	}
	// auto-save is on, was just turned on (which writes what was pending) or
	// was just turned off (which writes that choice, and nothing else changed)
	a.Persist()
}

// networkSettings are the settings the protocols are built from.
type networkSettings struct{ proxy, caCert, clientCert, clientKey string }

func (a *App) networkSettings() networkSettings {
	s := a.settings
	return networkSettings{s.Proxy, s.CACert, s.ClientCert, s.ClientKey}
}
