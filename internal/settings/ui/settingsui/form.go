package settingsui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/severity"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// settingControl draws the control of one setting.
type settingControl func(ctx *w.Ctx, dialog *SettingsDialog, host Host, field settings.Field) Node

// placement is where a row puts its control: beside the title block, or
// under it when the control needs the width of the row.
type placement int

const (
	placeByKind placement = iota // from the Kind of the setting (see placementOf)
	placeRight
	placeBelow
)

// customPlacement is the placement of a setting whose control is in
// settingControls and is not chosen by Kind alone: a stepper is short, the
// rest of them need the width. A key that is not here goes by Kind.
var customPlacement = map[string]placement{
	"fontSize":      placeRight,
	"smallFontSize": placeRight,
	"codeFontSize":  placeRight,
	"theme":         placeBelow, // one chip per installed theme
	"uiScale":       placeBelow, // a slider wants a long track
	"uiFont":        placeBelow, // a dropdown field with a family name
	"codeFont":      placeBelow,
}

// shortChoices is how many values an enum may have for its chips to sit
// beside the title; more of them go under it, as a long row of chips would
// squeeze the description.
const shortChoices = 3

// placementOf is where the control of field goes. Why by Kind and not a
// per-key table: a new boolean or short enum lands right without a row here
// (revisit when a kind needs both places).
func placementOf(field settings.Field) placement {
	if place, ok := customPlacement[field.Key]; ok {
		return place
	}
	switch {
	case field.Kind == settings.Bool:
		return placeRight
	case field.Kind == settings.Enum && !field.Open && len(field.Allowed) <= shortChoices:
		return placeRight
	}
	return placeBelow
}

// settingControls are the settings whose control is not the plain one for
// their kind. Everything else is a row of chips (an enum), On / Off (a
// boolean) or a text field (text and numbers).
var settingControls = map[string]settingControl{
	"fontSize":      stepControlOf,
	"smallFontSize": stepControlOf,
	"codeFontSize":  stepControlOf,
	"theme": func(ctx *w.Ctx, dialog *SettingsDialog, host Host, _ settings.Field) Node {
		return dialog.themeChips(ctx, host)
	},
	"corners": func(ctx *w.Ctx, dialog *SettingsDialog, host Host, field settings.Field) Node {
		return dialog.enumChips(ctx, host, field, labelsOf(appearance.CornerChoices, cornerLabel))
	},
	"contrast": func(ctx *w.Ctx, dialog *SettingsDialog, host Host, field settings.Field) Node {
		return dialog.enumChips(ctx, host, field, labelsOf(appearance.ContrastChoices, contrastLabel))
	},
	"layout": func(ctx *w.Ctx, dialog *SettingsDialog, host Host, field settings.Field) Node {
		return dialog.enumChips(ctx, host, field, labelsOf(appearance.LayoutChoices, layoutLabel))
	},
	"responseView": func(ctx *w.Ctx, dialog *SettingsDialog, host Host, field settings.Field) Node {
		return dialog.enumChips(ctx, host, field, nil)
	},
	"uiScale": func(ctx *w.Ctx, dialog *SettingsDialog, host Host, _ settings.Field) Node {
		return dialog.scaleControl(ctx, host)
	},
	"uiFont": func(ctx *w.Ctx, dialog *SettingsDialog, host Host, _ settings.Field) Node {
		return dialog.fontControl(ctx, host, FontUI)
	},
	"codeFont": func(ctx *w.Ctx, dialog *SettingsDialog, host Host, _ settings.Field) Node {
		return dialog.fontControl(ctx, host, FontCode)
	},
}

// stepControlOf is the control of a size: a number field with a stepper.
func stepControlOf(ctx *w.Ctx, dialog *SettingsDialog, host Host, field settings.Field) Node {
	return dialog.stepControl(ctx, host, field)
}

func cornerLabel(cornerChoice appearance.CornerChoice) (string, string) {
	return cornerChoice.ID, cornerChoice.Label
}
func contrastLabel(contrastChoice appearance.ContrastChoice) (string, string) {
	return contrastChoice.ID, contrastChoice.Label
}
func layoutLabel(layoutChoice appearance.LayoutChoice) (string, string) {
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

// chipRow is a row of chips sharing its width.
func chipRow(ctx *w.Ctx, chips []chip) Node {
	return chipsTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{"chips": chips}))
}

// enumChips is a row of chips, one per allowed value; the current one is lit.
func (d *SettingsDialog) enumChips(ctx *w.Ctx, host Host, field settings.Field, labels map[string]string) Node {
	value, _ := settings.Get(host.Settings(), field.Key)
	text, _ := value.(settings.Text)
	current := string(text)
	if field.Key == "corners" {
		current = appearance.CornerIDOf(current)
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
	return chipRow(ctx, chips)
}

func (d *SettingsDialog) themeChips(ctx *w.Ctx, host Host) Node {
	current := host.Settings().Theme
	themes := appearance.AllThemeChoices()
	chips := make([]chip, 0, len(themes))
	for _, theme := range themes {
		edit := func(edited *settings.Settings) { edited.Theme = theme.ID }
		chips = append(chips, choice(host, theme.Label, theme.ID == current, edit))
	}
	return chipRow(ctx, chips)
}

func (d *SettingsDialog) scaleControl(ctx *w.Ctx, host Host) Node {
	percent := int(host.Settings().UIScale()*100 + 0.5)
	return scaleTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"scale":    &d.scale,
		"min":      appearance.ScaleMin,
		"max":      appearance.ScaleMax,
		"step":     appearance.ScaleStep,
		"mid":      settings.DefaultUIScalePct,
		"value":    percent,
		"slide":    func(percent int) { host.Send(prefs.ScaleLive{Percent: percent}) }, // live: the UI resizes while dragging
		"slideEnd": func(int) { host.Send(prefs.Save{}) },
		"shown":    fmt.Sprintf("%d%%", d.scale.Shown(percent)),
		"shownW":   scaleShownW,
	}))
}

// boolSwitch is the toggle of a true-or-false setting.
func (d *SettingsDialog) boolSwitch(host Host, field settings.Field) Node {
	value, _ := settings.Get(host.Settings(), field.Key)
	on, _ := value.(settings.Flag)
	flip := func() { host.Send(prefs.Change{Edit: setKey(host, field.Key, settings.Flag(!on))}) }
	return Switch(bool(on), OnChange(flip))
}

// Input is the text field of a text or number setting, filled from the
// setting the first time it is needed.
func (d *SettingsDialog) Input(host Host, key string) *TextField {
	return d.fields.Input(key, func() string {
		value, _ := settings.Get(host.Settings(), key)
		switch value := value.(type) {
		case settings.Text:
			return string(value)
		case settings.Number:
			return strconv.Itoa(int(value))
		}
		return ""
	})
}

// Edited applies the text typed into the field of key. A number that is not
// a whole number in range is explained under the field, and the setting
// keeps its last good value.
func (d *SettingsDialog) Edited(host Host, key string) {
	field, ok := settings.FieldFor(key)
	if !ok {
		return
	}
	text := strings.TrimSpace(d.Input(host, key).Text)
	var value settings.Value = settings.Text(text)
	if field.Kind == settings.Int {
		number, err := strconv.Atoi(text)
		if err != nil || number < field.Min || number > field.Max {
			d.fields.SetNote(key, fmt.Sprintf("Enter a whole number between %d and %d", field.Min, field.Max))
			return
		}
		value = settings.Number(number)
	}
	d.fields.SetNote(key, "")
	host.Send(prefs.Change{Edit: setKey(host, key, value)})
}

// Note is why the value typed for key was refused, "" when it was not.
func (d *SettingsDialog) Note(key string) string { return d.fields.Note(key) }

// textControl is the field of a text or number setting.
func (d *SettingsDialog) textControl(ctx *w.Ctx, host Host, field settings.Field) Node {
	return textControlTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"input":  d.Input(host, field.Key),
		"edited": func() { d.Edited(host, field.Key) },
	}))
}

// stepControl is the control of a whole-number setting that is changed in
// small steps: a minus button, the number as a field and a plus button. A
// step starts from the setting, not from the text, so it recovers from an
// entry that was refused; at the end of the range it does nothing.
func (d *SettingsDialog) stepControl(ctx *w.Ctx, host Host, field settings.Field) Node {
	return stepTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"input":  d.Input(host, field.Key),
		"edited": func() { d.Edited(host, field.Key) },
		"less":   func() { d.Step(host, field, -1) },
		"more":   func() { d.Step(host, field, 1) },
		"fieldW": stepFieldW,
	}))
}

// Step moves the number setting of field by delta, within its range, and
// shows the new value in its field; at an end of the range nothing changes.
func (d *SettingsDialog) Step(host Host, field settings.Field, delta int) {
	value, _ := settings.Get(host.Settings(), field.Key)
	number, _ := value.(settings.Number)
	stepped := int(number) + delta
	if stepped < field.Min || stepped > field.Max {
		return
	}
	*d.Input(host, field.Key) = NewTextField(strconv.Itoa(stepped))
	d.Edited(host, field.Key)
}

// Reset puts the default back for key and drops it from the file.
func (d *SettingsDialog) Reset(host Host, key string) {
	host.Send(prefs.Change{Edit: func(current *settings.Settings) { resetKey(host, current, key) }})
	d.fields.Forget()
	if key == "uiFont" || key == "codeFont" {
		d.ApplyFonts(host)
	}
}

// settingRow is one setting: the title block (the title with a reset icon
// when it differs from the default, the description under it) and its
// control, beside the block or under it as placementOf says. A refused
// value is explained under the whole row.
func (d *SettingsDialog) settingRow(ctx *w.Ctx, host Host, field settings.Field) Node {
	var control Node
	switch custom, ok := settingControls[field.Key]; {
	case ok:
		control = custom(ctx, d, host, field)
	case field.Kind == settings.Enum:
		control = d.enumChips(ctx, host, field, nil)
	case field.Kind == settings.Bool:
		control = d.boolSwitch(host, field)
	case field.Kind == settings.Shortcuts || field.Kind == settings.Colors:
		control = d.openPageButton(ctx, field)
	case field.Kind == settings.File:
		control = d.fileControl(ctx, host, field)
	default:
		control = d.textControl(ctx, host, field)
	}
	view := settingView{
		Title:       field.Title,
		Changed:     settings.Changed(host.Settings(), field.Key),
		Reset:       func() { d.Reset(host, field.Key) },
		Description: Paragraph(field.Description),
		Note:        d.fields.Note(field.Key),
	}
	if placementOf(field) == placeRight {
		view.Side = control
	} else {
		view.Below = control
	}
	return settingRowTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{"setting": view}))
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
func (d *SettingsDialog) openPageButton(ctx *w.Ctx, field settings.Field) Node {
	page := "colors"
	if field.Kind == settings.Shortcuts {
		page = "shortcuts"
	}
	open := func() {
		d.Cat, d.Search = page, NewTextField("")
		d.scroll.Y = 0
	}
	return openPageTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"label":     "Open " + strings.ToLower(field.Category),
		"open":      open,
		"openPageW": openPageW,
	}))
}

// settingMatches reports whether field fits the search text: its title,
// description, key or one of its allowed values contains it.
func settingMatches(field settings.Field, query string) bool {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return true
	}
	haystack := strings.ToLower(field.Title + " " + field.Description + " " + field.Key + " " + strings.Join(field.Allowed, " "))
	return strings.Contains(haystack, needle)
}

// FormPage is the settings of one category, or, while a search text is
// typed, those of every category that match it.
func (d *SettingsDialog) FormPage(ctx *w.Ctx, host Host, category string) Node {
	searching := d.Searching()
	var rows []Node
	last := ""
	for _, field := range settings.Schema {
		if field.Hidden || !settingMatches(field, d.Search.Text) || (!searching && field.Category != category) {
			continue
		}
		if searching && field.Category != last {
			last = field.Category
			rows = append(rows, formCaptionTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{"category": field.Category})))
		}
		rows = append(rows, d.settingRow(ctx, host, field))
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
func (d *SettingsDialog) fileControl(ctx *w.Ctx, host Host, field settings.Field) Node {
	input := d.Input(host, field.Key)
	choose := func() { d.ChooseFile(host, field.Key) }
	if strings.TrimSpace(input.Text) == "" && !input.Focused {
		return emptyFileTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{"choose": choose, "chooseW": chooseFileW, "folderIconW": folderIconW}))
	}
	return chosenFileTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"input":  input,
		"edited": func() { d.Edited(host, field.Key) },
		"choose": choose,
	}))
}

// ChooseFile asks for a file for the setting key and puts its path in the
// setting.
func (d *SettingsDialog) ChooseFile(host Host, key string) {
	field, ok := settings.FieldFor(key)
	if !ok {
		return
	}
	host.Send(prefs.PickFile{Title: "Choose " + field.Title, Done: func(path string) {
		*d.Input(host, key) = NewTextField(path)
		d.Edited(host, key)
	}})
}

// Templates, parsed once above.
const (
	chipsSource = `<row h="{{rowH}}" gap="{{tightGap}}">
  <segment-chip for="chip in chips" selected="{{chip.Selected}}" grow="1" on-click="chip.Pick">{{chip.Label}}</segment-chip>
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
