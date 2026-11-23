package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

// changeSettings is saveSettings for the settings dialog's own controls (see
// settingsdialog.Component.Change).
func (a *App) changeSettings(edit func(*settings.Settings)) { a.prefs.Change(edit) }

// saveSettings edits and applies the settings and, unless auto-save is off,
// writes them (see settingsdialog.Component.Edit).
func (a *App) saveSettings(edit func(*settings.Settings)) { a.prefs.Edit(edit) }

// SettingsPath is the settings file, "" when settings are not kept.
func (a *App) SettingsPath() string { return a.prefs.SettingsPath() }

// networkSettings are the settings the protocols are built from.
type networkSettings struct{ proxy, caCert, clientCert, clientKey string }

func (a *App) networkSettings() networkSettings {
	s := a.settings
	return networkSettings{s.Proxy, s.CACert, s.ClientCert, s.ClientKey}
}
