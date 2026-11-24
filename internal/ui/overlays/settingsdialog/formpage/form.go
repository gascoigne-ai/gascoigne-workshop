package formpage

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// settingControl draws the control of one setting.
type settingControl func(ctx *w.Ctx, page *Page, host Host, field settings.Field) Node

// settingControls are the settings whose control is not the plain one for
// their kind. Everything else is a row of chips (an enum), On / Off (a
// boolean) or a text field (text and numbers).
var settingControls = map[string]settingControl{
	"smallFontSize": stepControlOf,
	"theme": func(ctx *w.Ctx, page *Page, host Host, _ settings.Field) Node {
		return page.themeChips(ctx, host)
	},
	"corners": func(ctx *w.Ctx, page *Page, host Host, field settings.Field) Node {
		return page.enumChips(ctx, host, field, labelsOf(theme.CornerChoices, cornerLabel))
	},
	"contrast": func(ctx *w.Ctx, page *Page, host Host, field settings.Field) Node {
		return page.enumChips(ctx, host, field, labelsOf(theme.ContrastChoices, contrastLabel))
	},
	"layout": func(ctx *w.Ctx, page *Page, host Host, field settings.Field) Node {
		return page.enumChips(ctx, host, field, labelsOf(settings.LayoutChoices, layoutLabel))
	},
	"responseView": func(ctx *w.Ctx, page *Page, host Host, field settings.Field) Node {
		return page.enumChips(ctx, host, field, nil)
	},
	"uiScale": func(ctx *w.Ctx, page *Page, host Host, _ settings.Field) Node {
		return page.scaleControl(ctx, host)
	},
}

// stepControlOf is the control of a size: a number field with a stepper.
func stepControlOf(ctx *w.Ctx, page *Page, host Host, field settings.Field) Node {
	return page.stepControl(ctx, host, field)
}

func cornerLabel(cornerChoice theme.CornerChoice) (string, string) {
	return cornerChoice.ID, cornerChoice.Label
}
func contrastLabel(contrastChoice theme.ContrastChoice) (string, string) {
	return contrastChoice.ID, contrastChoice.Label
}
func layoutLabel(layoutChoice settings.LayoutChoice) (string, string) {
	return layoutChoice.ID, layoutChoice.Label
}

// labelsOf maps ids to the labels a table of choices gives them.
func labelsOf[T any](choices []T, pick func(T) (id, label string)) map[string]string {
	labels := map[string]string{}
	for _, choice := range choices {
		id, label := pick(choice)
		labels[id] = label
	}
	return labels
}

// capitalized is id with a capital first letter, the label of a value
// that has no better one.
func capitalized(id string) string {
	if id == "" {
		return id
	}
	return strings.ToUpper(id[:1]) + id[1:]
}

var (
	chipsTemplate       = markup.Must("chips.vml", chipsSource)
	scaleTemplate       = markup.Must("scale.vml", scaleSource)
	stepTemplate        = markup.Must("step.vml", stepSource)
	textControlTemplate = markup.Must("textcontrol.vml", textcontrolSource)
	openPageTemplate    = markup.Must("openpage.vml", openpageSource)
	// with no file there is no field, only a button (see fileControl)
	emptyFileTemplate  = markup.Must("file-empty.vml", fileEmptySource)
	chosenFileTemplate = markup.Must("file-chosen.vml", fileChosenSource)
	settingRowTemplate = markup.Must("settingrow.vml", settingrowSource)
	// a search lists captions between the rows, so the page walks built nodes
	formPageTemplate    = markup.Must("formpage.vml", formpageSource)
	formCaptionTemplate = markup.Must("formcaption.vml", formcaptionSource)
	formEmptyTemplate   = markup.Must("formempty.vml", formemptySource)
)

// Widths of the controls of a setting row, by role.
const (
	scaleShownW = 40  // the percent beside the interface scale slider
	openPageW   = 120 // the button from a search result to its page
	chooseFileW = 150 // the Choose file button of an empty file setting
	folderIconW = 16  // the folder icon in that button
	stepFieldW  = 56  // the number between the minus and plus buttons of a stepped setting
)

// chip is what the chips template shows of one value of a segmented
// choice: gold when selected.
type chip struct {
	Label    string
	Selected bool
	Pick     func()
}

// choice is one chip of a segmented choice.
func choice(host Host, label string, selected bool, edit func(*settings.Settings)) chip {
	pick := func() { host.Send(prefs.Change{Edit: edit}) }
	return chip{Label: label, Selected: selected, Pick: pick}
}

// setKey is the edit that sets key to value; a refusal is shown as a toast,
// not dropped.
func setKey(host Host, key string, value settings.Value) func(*settings.Settings) {
	return func(edited *settings.Settings) {
		if err := settings.Set(edited, key, value); err != nil {
			host.Send(prefs.Toast{Text: "Could not change the setting: " + err.Error(), Level: severity.Error})
		}
	}
}

// resetKey puts the default back for key; a refusal is shown as a toast.
func resetKey(host Host, edited *settings.Settings, key string) {
	if err := settings.Reset(edited, key); err != nil {
		host.Send(prefs.Toast{Text: "Could not reset the setting: " + err.Error(), Level: severity.Error})
	}
}

// chipRow is a row of chips. Under the title the chips share the row's width;
// beside it each is as wide as its label, since an equal share cut the longer
// labels ("Side by...") while the shorter ones had room to spare.
func chipRow(ctx *w.Ctx, chips []chip, place form.Place) Node {
	share := 1
	if place == form.PlaceRight {
		share = 0
	}
	return chipsTemplate.MustBuild(markup.Scope{"chips": chips, "share": share}, ctx.SizesScope())
}

// enumChips is a row of chips, one per allowed value; the current one is lit.
func (p *Page) enumChips(ctx *w.Ctx, host Host, field settings.Field, labels map[string]string) Node {
	value, _ := settings.Get(host.Settings(), field.Key)
	text, _ := value.(settings.Text)
	current := string(text)
	if field.Key == "corners" {
		current = theme.CornerIDOf(current)
	}
	chips := make([]chip, 0, len(field.Allowed))
	for _, id := range field.Allowed {
		label := labels[id]
		if label == "" {
			label = capitalized(id)
		}
		edit := setKey(host, field.Key, settings.Text(id))
		chips = append(chips, choice(host, label, id == current, edit))
	}
	return chipRow(ctx, chips, form.PlaceOf(field))
}

func (p *Page) themeChips(ctx *w.Ctx, host Host) Node {
	current := host.Settings().Theme
	themes := theme.AllThemeChoices()
	chips := make([]chip, 0, len(themes))
	for _, theme := range themes {
		edit := func(edited *settings.Settings) { edited.Theme = theme.ID }
		chips = append(chips, choice(host, theme.Label, theme.ID == current, edit))
	}
	return chipRow(ctx, chips, form.PlaceBelow)
}

func (p *Page) scaleControl(ctx *w.Ctx, host Host) Node {
	percent := int(host.Settings().UIScale()*100 + 0.5)
	return scaleTemplate.MustBuild(markup.Scope{
		"scale":    &p.scale,
		"min":      settings.ScaleSliderMin,
		"max":      settings.ScaleSliderMax,
		"step":     settings.ScaleSliderStep,
		"mid":      settings.DefaultUIScalePct,
		"value":    percent,
		"slide":    func(percent int) { host.Send(prefs.ScaleLive{Percent: percent}) }, // live: the UI resizes while dragging
		"slideEnd": func(int) { host.Send(prefs.Save{}) },
		"shown":    fmt.Sprintf("%d%%", p.scale.Shown(percent)),
		"shownW":   scaleShownW,
	}, ctx.SizesScope())
}

// boolSwitch is the toggle of a true-or-false setting.
func (p *Page) boolSwitch(host Host, field settings.Field) Node {
	value, _ := settings.Get(host.Settings(), field.Key)
	on, _ := value.(settings.Flag)
	flip := func() { host.Send(prefs.Change{Edit: setKey(host, field.Key, settings.Flag(!on))}) }
	return Switch(bool(on), OnChange(flip))
}

// Input is the text field of a text or number setting, filled from the
// setting the first time it is needed.
func (p *Page) Input(host Host, key string) *TextField {
	if input, ok := p.inputs[key]; ok {
		return input
	}
	if p.inputs == nil {
		p.inputs = map[string]*TextField{}
	}
	input := new(TextField)
	*input = NewTextField(form.TextOf(host.Settings(), key))
	p.inputs[key] = input
	return input
}

// Edited applies the text typed into the field of key. A number that is not
// whole or outside its range is refused and explained under the row.
func (p *Page) Edited(host Host, key string) {
	p.update(host, form.Typed{Key: key, Text: p.Input(host, key).Text})
}

// Note is why the value typed for key was refused, "" when it was not.
func (p *Page) Note(key string) string { return p.model.Note(key) }

// update runs msg through the form model and carries out what it asks.
func (p *Page) update(host Host, msg form.Msg) {
	model, effects := form.Update(p.model, host.Settings(), msg)
	p.model = model
	for _, effect := range effects {
		p.carryOut(host, effect)
	}
}

func (p *Page) carryOut(host Host, effect form.Effect) {
	switch effect := effect.(type) {
	case form.Apply:
		host.Send(prefs.Change{Edit: setKey(host, effect.Key, effect.Value)})
	case form.Restore:
		host.Send(prefs.Change{Edit: func(current *settings.Settings) { resetKey(host, current, effect.Key) }})
	case form.Fill:
		*p.Input(host, effect.Key) = NewTextField(effect.Text)
	case form.ForgetTexts:
		p.inputs = nil
	case form.ApplyFonts:
		p.Fonts.Apply(host)
	case form.AskFile:
		host.Send(prefs.PickFile{Title: effect.Title, Done: func(path string) { p.update(host, form.Picked{Key: effect.Key, Path: path}) }})
	}
}

// textControl is the field of a text or number setting.
func (p *Page) textControl(ctx *w.Ctx, host Host, field settings.Field) Node {
	return textControlTemplate.MustBuild(markup.Scope{
		"input":  p.Input(host, field.Key),
		"edited": func() { p.Edited(host, field.Key) },
	}, ctx.SizesScope())
}

// stepControl is the control of a whole-number setting that is changed in
// small steps: a minus button, the number as a field and a plus button. A
// step starts from the setting, not from the text, so it recovers from an
// entry that was refused; at the end of the range it does nothing.
func (p *Page) stepControl(ctx *w.Ctx, host Host, field settings.Field) Node {
	return stepTemplate.MustBuild(markup.Scope{
		"input":  p.Input(host, field.Key),
		"edited": func() { p.Edited(host, field.Key) },
		"less":   func() { p.Step(host, field, -1) },
		"more":   func() { p.Step(host, field, 1) },
		"fieldW": stepFieldW,
	}, ctx.SizesScope())
}

// Step moves the number setting of field by delta, within its range, and
// refreshes its field; at an end of the range nothing happens.
func (p *Page) Step(host Host, field settings.Field, delta int) {
	p.update(host, form.Stepped{Key: field.Key, Delta: delta})
}

// Reset puts the default back for key and drops it from the file.
func (p *Page) Reset(host Host, key string) { p.update(host, form.Reset{Key: key}) }

// Forget drops the typed text and the notes, so the fields are rebuilt from
// the settings in effect.
func (p *Page) Forget() { p.model, p.inputs = form.Model{}, nil }

// settingRow is one setting: the title block (the title with a reset icon
// when it differs from the default, the description under it) and its
// control, beside the block or under it as form.PlaceOf says. A refused
// value is explained under the whole row.
func (p *Page) settingRow(ctx *w.Ctx, host Host, field settings.Field) Node {
	var control Node
	switch custom, ok := settingControls[field.Key]; {
	case ok:
		control = custom(ctx, p, host, field)
	case field.Kind == settings.Enum:
		control = p.enumChips(ctx, host, field, nil)
	case field.Kind == settings.Bool:
		control = p.boolSwitch(host, field)
	case field.Kind == settings.Shortcuts || field.Kind == settings.Colors:
		control = p.openPageButton(ctx, host, field)
	case field.Kind == settings.File:
		control = p.fileControl(ctx, host, field)
	default:
		control = p.textControl(ctx, host, field)
	}
	view := settingView{
		Title:       field.Title,
		Changed:     settings.Changed(host.Settings(), field.Key),
		Reset:       func() { p.Reset(host, field.Key) },
		Description: Paragraph(field.Description),
		Note:        p.model.Note(field.Key),
	}
	if form.PlaceOf(field) == form.PlaceRight {
		view.Side = control
	} else {
		view.Below = control
	}
	return settingRowTemplate.MustBuild(markup.Scope{"setting": view}, ctx.SizesScope())
}

// settingView is what the row template shows of one setting.
type settingView struct {
	Title       string
	Changed     bool // differs from the default: the reset icon
	Reset       func()
	Description Node   // wraps to the room the control leaves
	Side        Node   // the control beside the title block, nil when it is below
	Below       Node   // the control under the title block, nil when it is beside
	Note        string // why the typed value was refused, "" when it was not
}

// openPageButton leads from a search result to the page that edits the
// setting, which is a table and not a single control.
func (p *Page) openPageButton(ctx *w.Ctx, host Host, field settings.Field) Node {
	page := "colors"
	if field.Kind == settings.Shortcuts {
		page = "shortcuts"
	}
	open := func() { host.OpenPage(page) }
	return openPageTemplate.MustBuild(markup.Scope{
		"label":     "Open " + strings.ToLower(field.Category),
		"open":      open,
		"openPageW": openPageW,
	}, ctx.SizesScope())
}

// View is the settings of one category, or, while a search text is typed,
// those of every category that match it.
func (p *Page) View(ctx *w.Ctx, host Host, category, query string) Node {
	searching := strings.TrimSpace(query) != ""
	var rows []Node
	last := ""
	for _, field := range settings.Schema {
		face, isFont := form.FaceOfFont(field.Key)
		if field.Hidden || form.IsFaceSize(field.Key) || !form.Lists(field, face, isFont, query) || (!searching && field.Category != category) {
			continue
		}
		if searching && field.Category != last {
			last = field.Category
			rows = append(rows, formCaptionTemplate.MustBuild(markup.Scope{"category": field.Category}, ctx.SizesScope()))
		}
		if isFont {
			rows = append(rows, p.faceRow(ctx, host, face))
			continue
		}
		rows = append(rows, p.settingRow(ctx, host, field))
	}
	if len(rows) == 0 {
		return formEmptyTemplate.MustBuild(nil)
	}
	return formPageTemplate.MustBuild(markup.Scope{"rows": rows})
}

// fileControl is the control of a file setting. With no file it is one
// button, a folder icon and "Choose file", since an empty field has nothing
// to show; with a file it is a field with the path and a folder button inside
// it at the right, which chooses another.
func (p *Page) fileControl(ctx *w.Ctx, host Host, field settings.Field) Node {
	input := p.Input(host, field.Key)
	choose := func() { p.ChooseFile(host, field.Key) }
	if strings.TrimSpace(input.Text) == "" && !input.Focused {
		return emptyFileTemplate.MustBuild(markup.Scope{"choose": choose, "chooseW": chooseFileW, "folderIconW": folderIconW}, ctx.SizesScope())
	}
	return chosenFileTemplate.MustBuild(markup.Scope{
		"input":  input,
		"edited": func() { p.Edited(host, field.Key) },
		"choose": choose,
	}, ctx.SizesScope())
}

// ChooseFile asks for a file for the setting key and puts its path in the
// setting.
func (p *Page) ChooseFile(host Host, key string) { p.update(host, form.Choose{Key: key}) }

// Templates, parsed once above.
const (
	chipsSource = `<row h="{{rowH}}" gap="{{tightGap}}">
  <segment-chip for="chip in chips" selected="{{chip.Selected}}" grow="{{share}}" on-click="chip.Pick">{{chip.Label}}</segment-chip>
</row>
`

	scaleSource = `<row h="{{rowH}}" gap="{{looseGap}}">
  <slider state="scale" min="{{min}}" max="{{max}}" step="{{step}}" value="{{value}}" slider-mid="{{mid}}"
          on-slide="slide" on-slide-end="slideEnd"/>
  <text w="{{shownW}}" color="muted">{{shown}}</text>
</row>
`

	stepSource = `<row h="{{rowH}}" gap="{{tightGap}}" align-items="center">
  <button w="{{iconBtnW}}" h="{{compactH}}" on-click="less" tooltip="Smaller">-</button>
  <field state="input" w="{{fieldW}}" h="{{rowH}}" bg="sunken" on-change="edited" text-menu="input"/>
  <button w="{{iconBtnW}}" h="{{compactH}}" on-click="more" tooltip="Larger">+</button>
</row>
`

	textcontrolSource = `<field state="input" h="{{rowH}}" bg="sunken" placeholder="Default" on-change="edited" text-menu="input"/>
`

	openpageSource = `<row h="{{rowH}}">
  <button w="{{openPageW}}" on-click="open">{{label}}</button>
</row>
`

	fileEmptySource = `<row h="{{rowH}}">
  <row w="{{chooseW}}" h="{{rowH}}" bg="sunken" control="button" lighten pad-x="{{labelPad}}" gap="{{looseGap}}" align-items="center"
       on-click="choose" tooltip="Choose the file">
    <icon name="folder" w="{{folderIconW}}"/>
    <text grow="1" color="chip-text">Choose file</text>
  </row>
  <spacer/>
</row>
`

	fileChosenSource = `<row h="{{rowH}}" bg="sunken" control="field" align-items="center">
  <field state="input" h="{{rowH}}" grow="1" placeholder="No file" on-change="edited" text-menu="input"/>
  <icon-button icon="folder" w="{{iconBtnW}}" h="{{compactH}}" on-click="choose" tooltip="Choose another file"/>
</row>
`

	settingrowSource = `<col pad-y="{{looseGap}}" gap="{{tightGap}}" border-b="border-soft">
  <row gap="{{looseGap}}" align-items="center">
    <col grow="1">
      <row h="{{rowH}}" gap="{{tightGap}}" align-items="center">
        <text bold>{{setting.Title}}</text>
        <icon-button if="setting.Changed" icon="undo" w="{{iconBtnW}}" h="{{compactH}}" on-click="setting.Reset"
                     tooltip="Changed from the default. Remove it from settings.json so the default applies"/>
      </row>
      <slot name="setting.Description"/>
    </col>
    <slot if="setting.Side" name="setting.Side"/>
  </row>
  <slot if="setting.Below" name="setting.Below"/>
  <note if="setting.Note" tone="error">{{setting.Note}}</note>
</col>
`

	formpageSource = `<col>
  <slot for="row in rows" name="row"/>
</col>
`

	formcaptionSource = `<col pad-t="{{looseGap}}">
  <row h="{{labelH}}" border-b="border-soft" align-items="center">
    <caption>{{category}}</caption>
  </row>
</col>
`

	formemptySource = `<text muted>No settings match</text>
`
)
