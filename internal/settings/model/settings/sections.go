package settings

import (
	"bytes"
	"encoding/json"
)

// Sections are the top-level objects of settings.json, in the order they are
// written. appearance, requests and general hold the settings of that page
// of the dialog; colors and shortcuts are tables and are the section
// themselves.
var Sections = []string{"appearance", "colors", "shortcuts", "requests", "general"}

// sectionDescriptions say what each section holds, for completion.
var sectionDescriptions = map[string]string{
	"appearance": "How the window looks: theme, fonts, scale, layout.",
	"colors":     "Your color overrides, by theme.",
	"shortcuts":  "Your key bindings, by action.",
	"requests":   "How requests are sent: timeout, proxy, certificates.",
	"general":    "Saving, and what the program remembers.",
}

// objectSection reports whether name is a section that holds settings as an
// object (colors and shortcuts hold a table and are settings themselves).
func objectSection(name string) bool {
	return name == "appearance" || name == "requests" || name == "general"
}

// SectionOf is the section a setting is written in, "" for the two that are
// a section themselves (colors and shortcuts).
func SectionOf(field Field) string {
	switch field.Category {
	case "Appearance":
		return "appearance"
	case "Requests":
		return "requests"
	case "General":
		return "general"
	}
	return ""
}

// fieldsOf are the settings of an object section, in schema order.
func fieldsOf(section string) []Field {
	var fields []Field
	for _, field := range Schema {
		if SectionOf(field) == section {
			fields = append(fields, field)
		}
	}
	return fields
}

// unmarshalSettings reads settings text, sectioned or in the older flat
// form (or a mix), over *into.
func unmarshalSettings(data []byte, into *Settings) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return err
	}
	flat := map[string]json.RawMessage{}
	for key, value := range top {
		var inner map[string]json.RawMessage
		if objectSection(key) && json.Unmarshal(value, &inner) == nil {
			for innerKey, innerValue := range inner {
				flat[innerKey] = innerValue
			}
			continue
		}
		flat[key] = value
	}
	merged, err := json.Marshal(flat)
	if err != nil {
		return err
	}
	return json.Unmarshal(merged, into)
}

// sectionedText is the settings (as the flat map Minimal builds) written as
// indented JSON grouped by section, the sections in a fixed order and the
// keys of each in schema order.
func sectionedText(flat map[string]any) ([]byte, error) {
	var compact bytes.Buffer
	compact.WriteByte('{')
	wroteSection := false
	for _, section := range Sections {
		var body bytes.Buffer
		if objectSection(section) {
			body.WriteByte('{')
			wrote := false
			for _, field := range fieldsOf(section) {
				value, ok := flat[field.Key]
				if !ok {
					continue
				}
				if err := writeMember(&body, wrote, field.Key, value); err != nil {
					return nil, err
				}
				wrote = true
			}
			body.WriteByte('}')
			if !wrote {
				continue
			}
		} else {
			value, ok := flat[section]
			if !ok {
				continue
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			body.Write(encoded)
		}
		if wroteSection {
			compact.WriteByte(',')
		}
		encodedName, _ := json.Marshal(section)
		compact.Write(encodedName)
		compact.WriteByte(':')
		compact.Write(body.Bytes())
		wroteSection = true
	}
	compact.WriteByte('}')
	var indented bytes.Buffer
	if err := json.Indent(&indented, compact.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	return indented.Bytes(), nil
}

func writeMember(buffer *bytes.Buffer, comma bool, key string, value any) error {
	if comma {
		buffer.WriteByte(',')
	}
	name, err := json.Marshal(key)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	buffer.Write(name)
	buffer.WriteByte(':')
	buffer.Write(encoded)
	return nil
}
