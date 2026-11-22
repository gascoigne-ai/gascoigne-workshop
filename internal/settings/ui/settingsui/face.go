package settingsui

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fontFace pairs a font setting with the setting of its size; the form shows
// the pair as one row, so the size is not a row of its own. Small text size
// has no font and stays a plain stepper row. The pairing is a UI fact, not a
// field of settings.Field: the model and the file stay as they are. A slice
// of two structs because it is read per schema row per frame; revisit at more
// than about six faces or when something outside the form needs the pairing.
type fontFace struct {
	Font, Size string // keys of settings.Schema
	Role       FontRole
}

var fontFaces = []fontFace{
	{Font: "uiFont", Size: "fontSize", Role: FontUI},
	{Font: "codeFont", Size: "codeFontSize", Role: FontCode},
}

// fontMinW is the least width the font drop-down keeps beside the stepper:
// it elides its name before it gives up more room (the toolkit has no
// wrapping, and a font that shrank to nothing could not be read).
const fontMinW = 120

// faceOfFont is the face whose font setting is key.
func faceOfFont(key string) (fontFace, bool) {
	for _, face := range fontFaces {
		if face.Font == key {
			return face, true
		}
	}
	return fontFace{}, false
}

// isFaceSize reports whether key is the size of a face, shown in its row.
func isFaceSize(key string) bool {
	for _, face := range fontFaces {
		if face.Size == key {
			return true
		}
	}
	return false
}

// faceMatches reports whether the search text fits the font of face or its
// size, so the line is listed once, whichever of them the text names.
func faceMatches(face fontFace, query string) bool {
	return settingMatches(schemaRow(face.Font), query) || settingMatches(schemaRow(face.Size), query)
}

// schemaRow is the row of key in settings.Schema; the keys of fontFaces are
// pinned to exist by a test.
func schemaRow(key string) settings.Field {
	for _, field := range settings.Schema {
		if field.Key == key {
			return field
		}
	}
	return settings.Field{}
}

// faceDescription is the description of the font followed by a clause for the
// size, its range read from the size row so it is stated once.
func faceDescription(font, size settings.Field) string {
	return fmt.Sprintf("%s Size in pixels, %d to %d.", font.Description, size.Min, size.Max)
}

// faceView is what the control line of a face shows.
type faceView struct {
	Font    Node
	Size    Node
	Changed bool // the size differs from the default: its Reset icon
	Reset   func()
}

var faceLineTemplate = markup.Must("faceline.vml", faceLineSource)

// faceRow is one row for a font and its size: the title block of the font
// (its Reset icon, the description with the size's range), then the font
// drop-down, which grows, and the stepper with its own Reset icon. The icon's
// slot is reserved while the size is unchanged so the steppers of the faces
// line up and the font does not change width. A note of either setting shows
// under the line, the size's first.
func (d *SettingsDialog) faceRow(ctx *w.Ctx, host Host, face fontFace) Node {
	font, size := schemaRow(face.Font), schemaRow(face.Size)
	line := faceView{
		Font:    d.fontControl(ctx, host, face.Role),
		Size:    d.stepControl(ctx, host, size),
		Changed: settings.Changed(host.Settings(), size.Key),
		Reset:   func() { d.Reset(host, size.Key) },
	}
	note := d.fields.Note(size.Key)
	if note == "" {
		note = d.fields.Note(font.Key)
	}
	view := settingView{
		Title:       font.Title,
		Changed:     settings.Changed(host.Settings(), font.Key),
		Reset:       func() { d.Reset(host, font.Key) },
		Description: Paragraph(faceDescription(font, size)),
		Below:       faceLineTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{"face": line, "fontMinW": fontMinW})),
		Note:        note,
	}
	return settingRowTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{"setting": view}))
}

const faceLineSource = `<row gap="{{looseGap}}" align-items="center">
  <col grow="1" min-w="{{fontMinW}}">
    <slot name="face.Font"/>
  </col>
  <slot name="face.Size"/>
  <row w="{{iconBtnW}}" h="{{compactH}}" align-items="center">
    <icon-button if="face.Changed" icon="undo" w="{{iconBtnW}}" h="{{compactH}}" on-click="face.Reset"
                 tooltip="Changed from the default. Remove it from settings.json so the default applies"/>
  </row>
</row>
`
