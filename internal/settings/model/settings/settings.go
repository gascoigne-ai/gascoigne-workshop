// Package settings holds the user's preferences and the rules of their JSON
// text. Unknown values are kept as written; the UI decides how to fall back.
// Reading and writing the file is the settingsfile adapter's job.
package settings

import "time"

// Settings are the user preferences.
type Settings struct {
	Theme    string `json:"theme"`    // "dark" or "light"
	TextSize string `json:"textSize"` // "small", "medium" or "large"

	// Layout places the request and response panes: "side" by side or "stacked".
	Layout string `json:"layout"`

	// Contrast is "normal" or "high"; high strengthens the selection,
	// find and accent colors.
	Contrast string `json:"contrast"`

	// Colors are the user's color overrides: theme id, then token name,
	// then "#rrggbb".
	Colors map[string]map[string]string `json:"colors,omitempty"`

	// Shortcuts are the user's key bindings: action id to a key string such
	// as "Ctrl+Shift+Tab"; "" means unbound. Missing actions keep their default.
	Shortcuts map[string]string `json:"shortcuts,omitempty"`

	// Corners is how round controls are: "square", "soft" (the theme's own
	// radii) or "round".
	Corners string `json:"corners"`

	// ResponseView is how a response body is shown: "compact" (the
	// default), "pretty" or "raw".
	ResponseView string `json:"responseView"`

	// Flat turns every gradient into a flat fill.
	Flat bool `json:"flat"`

	// UIScalePct is the interface scale in percent, on top of the
	// display's own scale factor; 0 means the default.
	UIScalePct int `json:"uiScale"`

	// UIFont and CodeFont are the family names of the interface and code
	// fonts, found among the installed fonts; "" is the built-in Go font.
	UIFont   string `json:"uiFont,omitempty"`
	CodeFont string `json:"codeFont,omitempty"`

	// AutoSave writes every change to the file at once. When it is off a
	// change applies but is written only when the user saves.
	AutoSave bool `json:"autoSave"`

	// Workspace is the folder the collection is loaded from and saved to;
	// "" keeps the built-in sample collection in memory.
	Workspace string `json:"workspace,omitempty"`

	// Environment is the name of the active environment, "" for none.
	Environment string `json:"environment,omitempty"`

	// Proxy is the URL of an HTTP proxy for every REST request, such as
	// "http://localhost:8080"; "" uses the environment's.
	Proxy string `json:"proxy,omitempty"`

	// CACert is a PEM file of extra certificate authorities to trust;
	// ClientCert and ClientKey are a client certificate for servers that
	// ask for one.
	CACert     string `json:"caCert,omitempty"`
	ClientCert string `json:"clientCert,omitempty"`
	ClientKey  string `json:"clientKey,omitempty"`

	// TimeoutSec is the default request timeout in seconds.
	TimeoutSec int `json:"timeoutSec"`
}

// DefaultUIScalePct is the display's own scale; MinUIScalePct and
// MaxUIScalePct bound what is accepted.
const (
	DefaultUIScalePct = 100
	MinUIScalePct     = 50
	MaxUIScalePct     = 300
)

// UIScale is the interface scale as a factor, always within bounds.
func (s Settings) UIScale() float64 {
	pct := s.UIScalePct
	if pct == 0 {
		pct = DefaultUIScalePct
	}
	return float64(max(MinUIScalePct, min(pct, MaxUIScalePct))) / 100
}

// Default is what a first run uses.
func Default() Settings {
	return Settings{AutoSave: true, Theme: "dark", TextSize: "medium", Layout: "side", Contrast: "normal", TimeoutSec: 60, UIScalePct: DefaultUIScalePct, Corners: "soft", ResponseView: "compact"}
}

// Decode reads settings text over the defaults, so fields missing from
// the text keep their defaults. Text that does not parse yields Default
// and an error.
func Decode(data []byte) (Settings, error) {
	s := Default()
	if err := unmarshalSettings(data, &s); err != nil {
		return Default(), err
	}
	return s, nil
}

// Encode writes only what differs from the defaults, so the text shows
// what the user changed and a reset leaves no trace.
func Encode(s Settings) ([]byte, error) {
	data, err := Minimal(s)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// FileStamp identifies a version of the settings file cheaply: a changed
// stamp means the file was written. The zero value is a missing file.
type FileStamp struct {
	Modified time.Time
	Size     int64
	Exists   bool
}
