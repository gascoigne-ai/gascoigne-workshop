package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
)

// Kind is the type of a setting's value.
type Kind int

// The kinds of value a setting can hold.
const (
	String         Kind = iota // text
	File                       // text that is the path of a file
	Enum                       // text, one of Allowed (or any text when Open)
	Bool                       // true or false
	Int                        // a whole number within Min and Max
	Shortcuts                  // an object of action id to key string
	Colors                     // an object of theme id to an object of token to color
	CollectionList             // a list of open collections, each a folder and an environment name
)

// Field describes one key of settings.json. The form view, the raw text
// view, validation and completion all read this table, so there is no
// second list of settings to keep in step.
type Field struct {
	Key, Title, Description string
	Category                string
	Kind                    Kind
	Allowed                 []string // the values of an Enum
	Open                    bool     // an Enum that also accepts other text, such as a user theme
	Min, Max                int      // the range of an Int
	Hidden                  bool     // kept in the file but not shown in the form
}

// Categories are the pages of the form, in order.
var Categories = []string{"Appearance", "Colors", "Shortcuts", "Requests", "General", "Experimental"}

// rows are the settings, in the order of the form, the file and Schema.
// They stay in this file: the keys grpc and sse are the one allowed use of
// those protocol words.
var rows = []row{
	enumRow(Field{Key: "theme", Title: "Theme", Description: "The color theme: dark, light or the id of a theme file in the themes folder.", Category: "Appearance", Allowed: []string{"dark", "light"}, Open: true}, func(prefs *Settings) *string { return &prefs.Theme }),
	textRow(Field{Key: "uiFont", Title: "Interface font", Description: "Family name of an installed font for labels, buttons and fields; empty is the built-in font.", Category: "Appearance"}, func(prefs *Settings) *string { return &prefs.UIFont }),
	intRow(Field{Key: "fontSize", Title: "Interface font size", Description: "The size of the interface text in pixels, 10 to 24.", Category: "Appearance", Min: MinFontSize, Max: MaxFontSize}, func(prefs *Settings) *int { return &prefs.FontSize }),
	intRow(Field{Key: "smallFontSize", Title: "Small text size", Description: "The size of captions, hints and table headers in pixels, 8 to 24.", Category: "Appearance", Min: MinSmallFontSize, Max: MaxSmallFontSize}, func(prefs *Settings) *int { return &prefs.SmallFontSize }),
	textRow(Field{Key: "codeFont", Title: "Code font", Description: "Family name of an installed font for bodies and the code views; empty is the built-in font.", Category: "Appearance"}, func(prefs *Settings) *string { return &prefs.CodeFont }),
	intRow(Field{Key: "codeFontSize", Title: "Code font size", Description: "The size of bodies and code views in pixels, 8 to 32.", Category: "Appearance", Min: MinCodeFontSize, Max: MaxCodeFontSize}, func(prefs *Settings) *int { return &prefs.CodeFontSize }),
	intRow(Field{Key: "uiScale", Title: "Interface scale", Description: "Scale of the whole interface in percent, on top of the display's own scale.", Category: "Appearance", Min: MinUIScalePct, Max: MaxUIScalePct}, func(prefs *Settings) *int { return &prefs.UIScalePct }),
	enumRow(Field{Key: "corners", Title: "Corners", Description: "How round controls are.", Category: "Appearance", Allowed: []string{"square", "soft", "round"}}, func(prefs *Settings) *string { return &prefs.Corners }),
	flagRow(Field{Key: "flat", Title: "Flat colors", Description: "Turn every gradient into a flat fill.", Category: "Appearance"}, func(prefs *Settings) *bool { return &prefs.Flat }),
	enumRow(Field{Key: "contrast", Title: "Contrast", Description: "High strengthens the selection, find and accent colors.", Category: "Appearance", Allowed: []string{"normal", "high"}}, func(prefs *Settings) *string { return &prefs.Contrast }),
	enumRow(Field{Key: "layout", Title: "Request and response", Description: "Place the request and response panes stacked or side by side.", Category: "Appearance", Allowed: []string{"stacked", "side"}}, func(prefs *Settings) *string { return &prefs.Layout }),
	enumRow(Field{Key: "responseView", Title: "Response view", Description: "How a response body is shown when it arrives.", Category: "Appearance", Allowed: []string{"compact", "pretty", "raw"}}, func(prefs *Settings) *string { return &prefs.ResponseView }),
	flagRow(Field{Key: "autoSave", Title: "Save changes automatically", Description: "On: every change is written to settings.json at once. Off: changes apply now but are written only when you click Save.", Category: "General"}, func(prefs *Settings) *bool { return &prefs.AutoSave }),
	colorsRow(Field{Key: "colors", Title: "Color overrides", Description: "Your colors, by theme id and token name, as #rrggbb.", Category: "Colors"}, func(prefs *Settings) *map[string]map[string]string { return &prefs.Colors }),
	shortcutsRow(Field{Key: "shortcuts", Title: "Shortcut overrides", Description: "Your key bindings by action id, such as \"Ctrl+Shift+Tab\"; empty text unbinds.", Category: "Shortcuts"}, func(prefs *Settings) *map[string]string { return &prefs.Shortcuts }),
	intRow(Field{Key: "timeoutSec", Title: "Request timeout", Description: "How long a request may run, in seconds.", Category: "Requests", Min: 1, Max: 3600}, func(prefs *Settings) *int { return &prefs.TimeoutSec }),
	textRow(Field{Key: "proxy", Title: "Proxy", Description: "URL of an HTTP proxy for every REST request; empty uses the environment's.", Category: "Requests"}, func(prefs *Settings) *string { return &prefs.Proxy }),
	fileRow(Field{Key: "caCert", Title: "CA certificates", Description: "Path of a PEM file of extra certificate authorities to trust.", Category: "Requests"}, func(prefs *Settings) *string { return &prefs.CACert }),
	fileRow(Field{Key: "clientCert", Title: "Client certificate", Description: "Path of a client certificate for servers that ask for one.", Category: "Requests"}, func(prefs *Settings) *string { return &prefs.ClientCert }),
	fileRow(Field{Key: "clientKey", Title: "Client key", Description: "Path of the private key of the client certificate.", Category: "Requests"}, func(prefs *Settings) *string { return &prefs.ClientKey }),
	flagRow(Field{Key: "grpc", Title: "gRPC", Description: "Experimental: offer gRPC as a request protocol.", Category: "Experimental"}, func(prefs *Settings) *bool { return &prefs.GRPC }),
	flagRow(Field{Key: "sse", Title: "Server-Sent Events", Description: "Experimental: offer SSE as a request protocol.", Category: "Experimental"}, func(prefs *Settings) *bool { return &prefs.SSE }),
	collectionsRow(Field{Key: "collections", Title: "Open collections", Description: "The collections open in the window, in tree order: the folder of each and the name of its active environment; one without a folder is the built-in or configured collection.", Category: "General", Hidden: true}, func(prefs *Settings) *[]OpenCollection { return &prefs.Collections }),
	textRow(Field{Key: "workspace", Title: "Collection folder", Description: "The one open collection of an older settings file; read only when collections is absent.", Category: "General", Hidden: true}, func(prefs *Settings) *string { return &prefs.Workspace }),
	textRow(Field{Key: "environment", Title: "Active environment", Description: "The active environment of an older settings file; read only when collections is absent.", Category: "General", Hidden: true}, func(prefs *Settings) *string { return &prefs.Environment }),
}

// Schema lists every key of settings.json, in the order of its rows.
var Schema = fieldsOfRows(rows)

// legacyKeys are keys that files of version 0 and 1 may still carry and that
// the readers translate; they are not in Schema, so the form, completion and
// reset never see them.
var legacyKeys = map[string]legacyKey{
	legacySizeKey: {
		Field:       Field{Key: legacySizeKey, Category: "Appearance", Kind: Enum, Allowed: []string{"small", "medium", "large"}},
		replacement: "fontSize",
	},
}

// legacyKey is how an old key is checked and which setting replaced it.
type legacyKey struct {
	Field
	replacement string
}

// firstVersionWithoutLegacyKeys is the file version that replaced them.
const firstVersionWithoutLegacyKeys = 2

// FieldFor is the schema entry of key.
func FieldFor(key string) (Field, bool) {
	for _, field := range Schema {
		if field.Key == key {
			return field, true
		}
	}
	return Field{}, false
}

// Problem is something wrong in settings text: a syntax error, an unknown
// key or a value that does not fit its key. Start and End are byte offsets
// of the text it points at; Key is "" for a syntax error.
type Problem struct {
	Key        string
	Message    string
	Start, End int
}

// Validate checks settings text against the schema.
func Validate(data []byte) []Problem {
	var syntax any
	if err := json.Unmarshal(data, &syntax); err != nil {
		var bad *json.SyntaxError
		if errors.As(err, &bad) {
			end := min(int(bad.Offset), len(data))
			return []Problem{{Message: lineColumn(data, end) + ": " + bad.Error(), Start: max(end-1, 0), End: end}}
		}
		return []Problem{{Message: err.Error(), End: len(data)}}
	}
	if _, ok := syntax.(map[string]any); !ok {
		return []Problem{{Message: "settings must be a JSON object", End: len(data)}}
	}
	version, _ := fileversion.Of(data) // a bad version is reported by checkVersion
	return validateObject(data, 0, "", version)
}

// lineColumn spells an offset as "line 2, column 5".
func lineColumn(data []byte, offset int) string {
	before := data[:min(offset, len(data))]
	line := bytes.Count(before, []byte("\n")) + 1
	column := len(before) - (bytes.LastIndexByte(before, '\n') + 1) + 1
	return fmt.Sprintf("line %d, column %d", line, column)
}

// validateObject walks one object of the text, which starts at byte base:
// the top level (section "") or the body of a section. Offsets in the
// problems are in the whole text. version is the version of the file, which
// decides whether a legacy key is still a setting.
func validateObject(data []byte, base int, section string, version int) []Problem {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if _, err := decoder.Token(); err != nil { // the opening brace
		return nil
	}
	var problems []Problem
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			break
		}
		key, _ := keyToken.(string)
		keyEnd := int(decoder.InputOffset())
		keyStart := bytes.LastIndexByte(data[:keyEnd-1], '"')
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			break
		}
		at := func(message string) Problem {
			return Problem{Key: key, Message: message, Start: base + keyStart, End: base + keyEnd}
		}
		if section == "" && objectSection(key) {
			if len(raw) == 0 || raw[0] != '{' {
				problems = append(problems, at("must be an object of settings"))
				continue
			}
			valueStart := int(decoder.InputOffset()) - len(raw)
			problems = append(problems, validateObject(raw, base+valueStart, key, version)...)
			continue
		}
		if key == fileversion.Key {
			if message := checkVersion(raw, section); message != "" {
				problems = append(problems, at(message))
			}
			continue
		}
		field, known := FieldFor(key)
		if legacy, isLegacy := legacyKeys[key]; isLegacy {
			if version >= firstVersionWithoutLegacyKeys {
				problems = append(problems, at("replaced by "+legacy.replacement))
				continue
			}
			field, known = legacy.Field, true
		}
		switch {
		case !known:
			problems = append(problems, at("unknown setting"))
		case section != "" && SectionOf(field) != section:
			problems = append(problems, at(wrongSection(field)))
		default:
			if message := checkValue(field, raw); message != "" {
				problems = append(problems, at(message))
			}
		}
	}
	return problems
}

// checkVersion is what is wrong with raw as the version of the file, or "".
// The version is a property of the file, not a setting: it is not in the
// schema, and only the top level holds it.
func checkVersion(raw json.RawMessage, section string) string {
	if section != "" {
		return "belongs at the top level"
	}
	version, err := fileversion.Parse(raw)
	if err != nil {
		return fmt.Sprintf("must be a whole number from 0 to %d", FileVersion)
	}
	if version > FileVersion {
		return (&fileversion.NewerError{Found: version, Newest: FileVersion}).Error()
	}
	return ""
}

// wrongSection says where a setting written in the wrong section belongs.
func wrongSection(field Field) string {
	if section := SectionOf(field); section != "" {
		return "belongs in " + section
	}
	return "belongs at the top level"
}

// checkValue is what is wrong with raw as the value of field, or "".
func checkValue(field Field, raw json.RawMessage) string {
	switch field.Kind {
	case String, File:
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return "must be text"
		}
	case Enum:
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return "must be text, one of " + strings.Join(field.Allowed, ", ")
		}
		if !field.Open && !contains(field.Allowed, text) {
			return "must be one of " + strings.Join(field.Allowed, ", ")
		}
	case Bool:
		var flag bool
		if json.Unmarshal(raw, &flag) != nil {
			return "must be true or false"
		}
	case Int:
		var number float64
		if json.Unmarshal(raw, &number) != nil {
			return "must be a number"
		}
		if number != float64(int(number)) || int(number) < field.Min || int(number) > field.Max {
			return fmt.Sprintf("must be a whole number between %d and %d", field.Min, field.Max)
		}
	case Shortcuts:
		var table map[string]string
		if json.Unmarshal(raw, &table) != nil {
			return "must be an object of text values, action id to key"
		}
	case Colors:
		var table map[string]map[string]string
		if json.Unmarshal(raw, &table) != nil {
			return "must be an object of themes, each an object of token to #rrggbb text"
		}
	case CollectionList:
		if !decodesStrictly(raw, &[]OpenCollection{}) {
			return "must be a list of objects with folder and environment text"
		}
	}
	return ""
}

// decodesStrictly reports whether raw fits into, with no key the type does
// not have, so a misspelled key is shown rather than silently dropped.
func decodesStrictly(raw json.RawMessage, into any) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(into) == nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// Parse reads settings text over the defaults. Text with problems changes
// nothing: last, the settings in effect, is returned with the problems.
func Parse(data []byte, last Settings) (Settings, []Problem) {
	if problems := Validate(data); len(problems) > 0 {
		return last, problems
	}
	parsed, _, err := fileversion.Read(data, readers)
	if err != nil {
		return last, []Problem{{Message: err.Error(), End: len(data)}}
	}
	return parsed, nil
}

// Expectation says in words what value a setting takes: "one of side,
// stacked", "true or false", "a whole number from 50 to 300".
func Expectation(field Field) string {
	switch field.Kind {
	case Enum:
		if field.Open {
			return strings.Join(field.Allowed, ", ") + ", or the id of a theme file"
		}
		return "one of " + strings.Join(field.Allowed, ", ")
	case Bool:
		return "true or false"
	case Int:
		return fmt.Sprintf("a whole number from %d to %d", field.Min, field.Max)
	case File:
		return "the path of a file"
	case Shortcuts:
		return "an object of action id to key text"
	case Colors:
		return "an object of theme id to token to #rrggbb"
	case CollectionList:
		return "a list of objects with folder and environment text"
	}
	return "text"
}
