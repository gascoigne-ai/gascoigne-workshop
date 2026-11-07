package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Kind is the type of a setting's value.
type Kind int

// The kinds of value a setting can hold.
const (
	String    Kind = iota // text
	File                  // text that is the path of a file
	Enum                  // text, one of Allowed (or any text when Open)
	Bool                  // true or false
	Int                   // a whole number within Min and Max
	Shortcuts             // an object of action id to key string
	Colors                // an object of theme id to an object of token to color
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
var Categories = []string{"Appearance", "Colors", "Shortcuts", "Requests", "General"}

// Schema lists every key of settings.json.
var Schema = []Field{
	{Key: "theme", Title: "Theme", Description: "The color theme: dark, light or the id of a theme file in the themes folder.", Category: "Appearance", Kind: Enum, Allowed: []string{"dark", "light"}, Open: true},
	{Key: "textSize", Title: "Text size", Description: "The size of the interface text.", Category: "Appearance", Kind: Enum, Allowed: []string{"small", "medium", "large"}},
	{Key: "uiScale", Title: "Interface scale", Description: "Scale of the whole interface in percent, on top of the display's own scale.", Category: "Appearance", Kind: Int, Min: MinUIScalePct, Max: MaxUIScalePct},
	{Key: "uiFont", Title: "Interface font", Description: "Family name of an installed font for labels, buttons and fields; empty is the built-in font.", Category: "Appearance", Kind: String},
	{Key: "codeFont", Title: "Code font", Description: "Family name of an installed font for bodies and the code views; empty is the built-in font.", Category: "Appearance", Kind: String},
	{Key: "corners", Title: "Corners", Description: "How round controls are.", Category: "Appearance", Kind: Enum, Allowed: []string{"square", "soft", "round"}},
	{Key: "flat", Title: "Flat colors", Description: "Turn every gradient into a flat fill.", Category: "Appearance", Kind: Bool},
	{Key: "contrast", Title: "Contrast", Description: "High strengthens the selection, find and accent colors.", Category: "Appearance", Kind: Enum, Allowed: []string{"normal", "high"}},
	{Key: "layout", Title: "Request and response", Description: "Place the request and response panes side by side or stacked.", Category: "Appearance", Kind: Enum, Allowed: []string{"side", "stacked"}},
	{Key: "responseView", Title: "Response view", Description: "How a response body is shown when it arrives.", Category: "Appearance", Kind: Enum, Allowed: []string{"compact", "pretty", "raw"}},
	{Key: "autoSave", Title: "Save changes automatically", Description: "On: every change is written to settings.json at once. Off: changes apply now but are written only when you click Save.", Category: "General", Kind: Bool},
	{Key: "colors", Title: "Color overrides", Description: "Your colors, by theme id and token name, as #rrggbb.", Category: "Colors", Kind: Colors},
	{Key: "shortcuts", Title: "Shortcut overrides", Description: "Your key bindings by action id, such as \"Ctrl+Shift+Tab\"; empty text unbinds.", Category: "Shortcuts", Kind: Shortcuts},
	{Key: "timeoutSec", Title: "Request timeout", Description: "How long a request may run, in seconds.", Category: "Requests", Kind: Int, Min: 1, Max: 3600},
	{Key: "proxy", Title: "Proxy", Description: "URL of an HTTP proxy for every REST request; empty uses the environment's.", Category: "Requests", Kind: String},
	{Key: "caCert", Title: "CA certificates", Description: "Path of a PEM file of extra certificate authorities to trust.", Category: "Requests", Kind: File},
	{Key: "clientCert", Title: "Client certificate", Description: "Path of a client certificate for servers that ask for one.", Category: "Requests", Kind: File},
	{Key: "clientKey", Title: "Client key", Description: "Path of the private key of the client certificate.", Category: "Requests", Kind: File},
	{Key: "workspace", Title: "Collection folder", Description: "The folder the collection is loaded from and saved to.", Category: "General", Kind: String, Hidden: true},
	{Key: "environment", Title: "Active environment", Description: "Name of the environment in use.", Category: "General", Kind: String, Hidden: true},
}

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
	return validateKeys(data)
}

// lineColumn spells an offset as "line 2, column 5".
func lineColumn(data []byte, offset int) string {
	before := data[:min(offset, len(data))]
	line := bytes.Count(before, []byte("\n")) + 1
	column := len(before) - (bytes.LastIndexByte(before, '\n') + 1) + 1
	return fmt.Sprintf("line %d, column %d", line, column)
}

// validateKeys checks the settings object of valid JSON text: its sections
// and, for the older flat form, its keys.
func validateKeys(data []byte) []Problem { return validateObject(data, 0, "") }

// validateObject walks one object of the text, which starts at byte base:
// the top level (section "") or the body of a section. Offsets in the
// problems are in the whole text.
func validateObject(data []byte, base int, section string) []Problem {
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
			problems = append(problems, validateObject(raw, base+valueStart, key)...)
			continue
		}
		field, known := FieldFor(key)
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
	}
	return ""
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
	parsed := Default()
	if err := unmarshalSettings(data, &parsed); err != nil {
		return last, []Problem{{Message: err.Error(), End: len(data)}}
	}
	return parsed, nil
}

// asMap is settings as the map its JSON form gives.
func asMap(s Settings) map[string]any {
	data, _ := json.Marshal(s)
	out := map[string]any{}
	_ = json.Unmarshal(data, &out)
	return out
}

// Changed reports whether the value of key differs from the default.
func Changed(s Settings, key string) bool {
	return !reflect.DeepEqual(asMap(s)[key], asMap(Default())[key])
}

// Reset puts the default back as the value of key.
func Reset(s *Settings, key string) {
	current, defaults := asMap(*s), asMap(Default())
	if value, ok := defaults[key]; ok {
		current[key] = value
	} else {
		delete(current, key)
	}
	data, _ := json.Marshal(current)
	rebuilt := Default()
	_ = json.Unmarshal(data, &rebuilt)
	*s = rebuilt
}

// Minimal is the settings text with only what differs from the defaults,
// grouped by section, so the file stays short and a reset leaves no trace.
func Minimal(s Settings) ([]byte, error) {
	current, defaults := asMap(s), asMap(Default())
	kept := map[string]any{}
	for key, value := range current {
		if !reflect.DeepEqual(value, defaults[key]) {
			kept[key] = value
		}
	}
	return sectionedText(kept)
}

// Value is the value of key in s as JSON gives it back (text, a float64
// for numbers, a bool, a map), or nil when s has none.
func Value(s Settings, key string) any {
	value := asMap(s)[key]
	if number, ok := value.(float64); ok && number == float64(int(number)) {
		return int(number)
	}
	return value
}

// With is s with key set to value; an unknown key or a value of the
// wrong type leaves s as it was.
func With(s Settings, key string, value any) Settings {
	if _, known := FieldFor(key); !known {
		return s
	}
	current := asMap(s)
	current[key] = value
	data, _ := json.Marshal(current)
	changed := Default()
	if err := json.Unmarshal(data, &changed); err != nil {
		return s
	}
	return changed
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
	}
	return "text"
}
