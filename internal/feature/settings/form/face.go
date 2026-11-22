package form

import (
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

// FontRole is which text a font is for.
type FontRole int

// The two roles a font can be picked for.
const (
	FontUI   FontRole = iota // labels, buttons, fields
	FontCode                 // bodies, headers, the code views
)

// SavedFont is the family name stored for role.
func SavedFont(current settings.Settings, role FontRole) string {
	if role == FontCode {
		return current.CodeFont
	}
	return current.UIFont
}

// Face pairs a font setting with the setting of its size; the form shows
// the pair as one row, so the size is not a row of its own. Small text size
// has no font and stays a plain stepper row. The pairing is a UI fact, not a
// field of settings.Field: the model and the file stay as they are. A slice
// of two structs because it is read per schema row per frame; revisit at more
// than about six faces or when something outside the form needs the pairing.
type Face struct {
	Font, Size string // keys of settings.Schema
	Role       FontRole
}

// Faces are the fonts of the form with the size beside each.
var Faces = []Face{
	{Font: "uiFont", Size: "fontSize", Role: FontUI},
	{Font: "codeFont", Size: "codeFontSize", Role: FontCode},
}

// FaceOfFont is the face whose font setting is key.
func FaceOfFont(key string) (Face, bool) {
	for _, face := range Faces {
		if face.Font == key {
			return face, true
		}
	}
	return Face{}, false
}

// IsFaceSize reports whether key is the size of a face, shown in its row.
func IsFaceSize(key string) bool {
	for _, face := range Faces {
		if face.Size == key {
			return true
		}
	}
	return false
}

// faceMatches reports whether the search text fits the font of face or its
// size, so the line is listed once, whichever of them the text names.
func faceMatches(face Face, query string) bool {
	return Matches(SchemaRow(face.Font), query) || Matches(SchemaRow(face.Size), query)
}

// SchemaRow is the row of key in settings.Schema; the keys of Faces are
// pinned to exist by a test.
func SchemaRow(key string) settings.Field {
	for _, field := range settings.Schema {
		if field.Key == key {
			return field
		}
	}
	return settings.Field{}
}

// FaceDescription is the description of the font followed by a clause for the
// size, its range read from the size row so it is stated once.
func FaceDescription(font, size settings.Field) string {
	return fmt.Sprintf("%s Size in pixels, %d to %d.", font.Description, size.Min, size.Max)
}

// Matches reports whether field fits the search text: its title,
// description, key or one of its allowed values contains it.
func Matches(field settings.Field, query string) bool {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return true
	}
	haystack := strings.ToLower(field.Title + " " + field.Description + " " + field.Key + " " + strings.Join(field.Allowed, " "))
	return strings.Contains(haystack, needle)
}

// Lists reports whether the search text lists field: a font through
// its face, so the line shows whichever of its two settings the text names.
func Lists(field settings.Field, face Face, isFont bool, query string) bool {
	if isFont {
		return faceMatches(face, query)
	}
	return Matches(field, query)
}
