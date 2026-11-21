// Package settings holds the user's preferences and the rules of their JSON
// text. Unknown values are kept as written; the UI decides how to fall back.
// Reading and writing the file is the settingsfile adapter's job.
package settings

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
)

// Settings are the user preferences.
type Settings struct {
	Theme string `json:"theme"` // "dark" or "light"

	// FontSize is the size of the interface text in pixels, MinFontSize to
	// MaxFontSize; the small and mono faces derive from it (uitoolkit/theming).
	FontSize int `json:"fontSize"`

	// Layout places the request and response panes: "stacked" (the default) or "side" by side.
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

	// Collections are the collections the window had open, in tree order.
	Collections []OpenCollection `json:"collections,omitempty"`

	// Workspace and Environment are what a file written when one collection
	// was open at a time remembers: its folder and the name of its active
	// environment. They are read only when Collections is empty
	// (RememberedCollections) and cleared when it is written.
	Workspace   string `json:"workspace,omitempty"`
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

	// GRPC and SSE turn on the experimental gRPC and Server-Sent Events
	// protocols; both are off until the user opts in.
	GRPC bool `json:"grpc,omitempty"`
	SSE  bool `json:"sse,omitempty"`
}

// DefaultFontSize is the interface text size in pixels a first run uses;
// MinFontSize and MaxFontSize bound it. They are stated again in
// uitoolkit/theming, which this package may not import;
// appearance's TestFontSizeRangeMatchesToolkit pins the two together.
const (
	DefaultFontSize = 14
	MinFontSize     = 10
	MaxFontSize     = 24
)

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

// baseScales is the platform's own factor under the user's scale: Linux
// desktops draw text and controls small, the others are right as they are.
var baseScales = map[string]float64{"linux": 1.1}

// BaseScale is the interface factor the platform needs before the user's
// UIScale; 1 for a platform not in the table.
func BaseScale(goos string) float64 {
	if factor, ok := baseScales[goos]; ok {
		return factor
	}
	return 1
}

// Default is what a first run uses.
func Default() Settings {
	return Settings{AutoSave: true, Theme: "dark", FontSize: DefaultFontSize, Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: DefaultUIScalePct, Corners: "soft", ResponseView: "compact"}
}

// Decode reads settings text over the defaults, so fields missing from
// the text keep their defaults. Text that does not parse yields Default
// and an error.
func Decode(data []byte) (Settings, error) {
	decoded, _, err := fileversion.Read(data, readers)
	if err != nil {
		return Default(), err
	}
	return decoded, nil
}

// readers read settings.json by version (the fileversion package doc).
// Versions 0 and 1 share a layout, sectioned, the older flat form or a mix,
// and both carry the text size as "textSize": small, medium or large, so
// both read it through decodeLegacyTextSize into FontSize. Version 2 writes
// the pixel size itself. The legacy table is looked up once per load and is
// closed: no later build writes another word, so it needs no extension point.
var readers = []func([]byte) (Settings, error){decodeLegacyTextSize, decodeLegacyTextSize, decodeOverDefaults}

// FileVersion is the version of settings.json this build writes.
var FileVersion = len(readers) - 1

// legacyFontSizes are the pixel sizes of the words "textSize" held before
// version 2.
var legacyFontSizes = map[string]int{"small": 13, "medium": 14, "large": 16}

// decodeOverDefaults reads settings text over the defaults.
func decodeOverDefaults(data []byte) (Settings, error) {
	flat, err := flatten(data)
	if err != nil {
		return Default(), err
	}
	return unmarshalFlat(flat)
}

// decodeLegacyTextSize reads a version 0 or 1 file: "textSize" becomes
// FontSize unless the file also states fontSize, which wins. A word the table
// does not know, a value that is not text, or no key at all means the default.
func decodeLegacyTextSize(data []byte) (Settings, error) {
	flat, err := flatten(data)
	if err != nil {
		return Default(), err
	}
	legacy, hasLegacy := flat[legacySizeKey]
	delete(flat, legacySizeKey)
	if _, hasFontSize := flat["fontSize"]; hasLegacy && !hasFontSize {
		flat["fontSize"] = json.RawMessage(strconv.Itoa(legacyFontSize(legacy)))
	}
	return unmarshalFlat(flat)
}

const legacySizeKey = "textSize"

func legacyFontSize(raw json.RawMessage) int {
	var word string
	if json.Unmarshal(raw, &word) != nil {
		return DefaultFontSize
	}
	if size, ok := legacyFontSizes[word]; ok {
		return size
	}
	return DefaultFontSize
}

// Encode writes only what differs from the defaults, so the text shows
// what the user changed and a reset leaves no trace.
func Encode(prefs Settings) ([]byte, error) {
	data, err := Minimal(prefs)
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
