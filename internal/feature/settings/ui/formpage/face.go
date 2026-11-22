package formpage

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fontMinW is the least width the font drop-down keeps beside the stepper:
// it elides its name before it gives up more room (the toolkit has no
// wrapping, and a font that shrank to nothing could not be read).
const fontMinW = 120

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
func (p *Page) faceRow(ctx *w.Ctx, host Host, face form.Face) Node {
	font, size := form.SchemaRow(face.Font), form.SchemaRow(face.Size)
	line := faceView{
		Font:    p.fontControl(ctx, host, face.Role),
		Size:    p.stepControl(ctx, host, size),
		Changed: settings.Changed(host.Settings(), size.Key),
		Reset:   func() { p.Reset(host, size.Key) },
	}
	note := p.model.Note(size.Key)
	if note == "" {
		note = p.model.Note(font.Key)
	}
	view := settingView{
		Title:       font.Title,
		Changed:     settings.Changed(host.Settings(), font.Key),
		Reset:       func() { p.Reset(host, font.Key) },
		Description: Paragraph(form.FaceDescription(font, size)),
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
