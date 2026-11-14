package settingsui

import (
	"embed"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// descriptionCols is how many characters a setting's description may have
// on one line before it is wrapped.
const descriptionCols = 76

// settingControl draws the control of one setting.
type settingControl func(d *SettingsDialog, host Host, field settings.Field) Node

// settingControls are the settings whose control is not the plain one for
// their kind. Everything else is a row of chips (an enum), On / Off (a
// boolean) or a text field (text and numbers).
var settingControls = map[string]settingControl{
	"theme": func(d *SettingsDialog, host Host, _ settings.Field) Node { return d.themeChips(host) },
	"textSize": func(d *SettingsDialog, host Host, field settings.Field) Node {
		return d.enumChips(host, field, labelsOf(appearance.TextSizeChoices, sizeLabel))
	},
	"corners": func(d *SettingsDialog, host Host, field settings.Field) Node {
		return d.enumChips(host, field, labelsOf(appearance.CornerChoices, cornerLabel))
	},
	"contrast": func(d *SettingsDialog, host Host, field settings.Field) Node {
		return d.enumChips(host, field, labelsOf(appearance.ContrastChoices, contrastLabel))
	},
	"layout": func(d *SettingsDialog, host Host, field settings.Field) Node {
		return d.enumChips(host, field, labelsOf(appearance.LayoutChoices, layoutLabel))
	},
	"responseView": func(d *SettingsDialog, host Host, field settings.Field) Node { return d.enumChips(host, field, nil) },
	"uiScale":      func(d *SettingsDialog, host Host, _ settings.Field) Node { return d.scaleControl(host) },
	"uiFont":       func(d *SettingsDialog, host Host, _ settings.Field) Node { return d.fontControl(host, FontUI) },
	"codeFont":     func(d *SettingsDialog, host Host, _ settings.Field) Node { return d.fontControl(host, FontCode) },
}

func sizeLabel(c appearance.SizeChoice) (string, string)         { return c.ID, c.Label }
func cornerLabel(c appearance.CornerChoice) (string, string)     { return c.ID, c.Label }
func contrastLabel(c appearance.ContrastChoice) (string, string) { return c.ID, c.Label }
func layoutLabel(c appearance.LayoutChoice) (string, string)     { return c.ID, c.Label }

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

//go:embed chips.vml scale.vml textcontrol.vml openpage.vml
//go:embed file-empty.vml file-chosen.vml
//go:embed settingrow.vml formpage.vml formcaption.vml formempty.vml
var formTemplates embed.FS

var (
	chipsTemplate       = markup.MustLoad(formTemplates, "chips.vml")
	scaleTemplate       = markup.MustLoad(formTemplates, "scale.vml")
	textControlTemplate = markup.MustLoad(formTemplates, "textcontrol.vml")
	openPageTemplate    = markup.MustLoad(formTemplates, "openpage.vml")
	// with no file there is no field, only a button (see fileControl)
	emptyFileTemplate  = markup.MustLoad(formTemplates, "file-empty.vml")
	chosenFileTemplate = markup.MustLoad(formTemplates, "file-chosen.vml")
	settingRowTemplate = markup.MustLoad(formTemplates, "settingrow.vml")
	// a search lists captions between the rows, so the page walks built nodes
	formPageTemplate    = markup.MustLoad(formTemplates, "formpage.vml")
	formCaptionTemplate = markup.MustLoad(formTemplates, "formcaption.vml")
	formEmptyTemplate   = markup.MustLoad(formTemplates, "formempty.vml")
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

// chipRow is a row of chips sharing its width.
func chipRow(chips []chip) Node {
	return chipsTemplate.MustBuild(markup.Scope{"chips": chips, "rowH": w.RowH})
}

// enumChips is a row of chips, one per allowed value; the current one is lit.
func (d *SettingsDialog) enumChips(host Host, field settings.Field, labels map[string]string) Node {
	current, _ := settings.Value(host.Settings(), field.Key).(string)
	if field.Key == "corners" {
		current = appearance.CornerIDOf(current)
	}
	chips := make([]chip, 0, len(field.Allowed))
	for _, id := range field.Allowed {
		label := labels[id]
		if label == "" {
			label = capitalized(id)
		}
		edit := func(edited *settings.Settings) { *edited = settings.With(*edited, field.Key, id) }
		chips = append(chips, choice(host, label, id == current, edit))
	}
	return chipRow(chips)
}

func (d *SettingsDialog) themeChips(host Host) Node {
	current := host.Settings().Theme
	themes := appearance.AllThemeChoices()
	chips := make([]chip, 0, len(themes))
	for _, theme := range themes {
		edit := func(edited *settings.Settings) { edited.Theme = theme.ID }
		chips = append(chips, choice(host, theme.Label, theme.ID == current, edit))
	}
	return chipRow(chips)
}

func (d *SettingsDialog) scaleControl(host Host) Node {
	percent := int(host.Settings().UIScale()*100 + 0.5)
	return scaleTemplate.MustBuild(markup.Scope{
		"rowH":     w.RowH,
		"scale":    &d.scale,
		"min":      appearance.ScaleMin,
		"max":      appearance.ScaleMax,
		"step":     appearance.ScaleStep,
		"mid":      settings.DefaultUIScalePct,
		"value":    percent,
		"slide":    func(percent int) { host.Send(prefs.ScaleLive{Percent: percent}) }, // live: the UI resizes while dragging
		"slideEnd": func(int) { host.Send(prefs.Save{}) },
		"shown":    fmt.Sprintf("%d%%", d.scale.Shown(percent)),
	})
}

// boolChips is Off and On for a true-or-false setting.
func (d *SettingsDialog) boolChips(host Host, field settings.Field) Node {
	on, _ := settings.Value(host.Settings(), field.Key).(bool)
	set := func(value bool) func(*settings.Settings) {
		return func(edited *settings.Settings) { *edited = settings.With(*edited, field.Key, value) }
	}
	return chipRow([]chip{choice(host, "Off", !on, set(false)), choice(host, "On", on, set(true))})
}

// Input is the text field of a text or number setting, filled from the
// setting the first time it is needed.
func (d *SettingsDialog) Input(host Host, key string) *TextField {
	if d.inputs == nil {
		d.inputs = map[string]*TextField{}
	}
	if input, ok := d.inputs[key]; ok {
		return input
	}
	text := ""
	switch value := settings.Value(host.Settings(), key).(type) {
	case string:
		text = value
	case int:
		text = strconv.Itoa(value)
	}
	input := NewTextField(text)
	d.inputs[key] = &input
	return &input
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
	var value any = text
	if field.Kind == settings.Int {
		number, err := strconv.Atoi(text)
		if err != nil || number < field.Min || number > field.Max {
			d.setNote(key, fmt.Sprintf("Enter a whole number between %d and %d", field.Min, field.Max))
			return
		}
		value = number
	}
	d.setNote(key, "")
	host.Send(prefs.Change{Edit: func(s *settings.Settings) { *s = settings.With(*s, key, value) }})
}

func (d *SettingsDialog) setNote(key, note string) {
	if d.notes == nil {
		d.notes = map[string]string{}
	}
	d.notes[key] = note
}

// Note is why the value typed for key was refused, "" when it was not.
func (d *SettingsDialog) Note(key string) string { return d.notes[key] }

// textControl is the field of a text or number setting.
func (d *SettingsDialog) textControl(host Host, field settings.Field) Node {
	return textControlTemplate.MustBuild(markup.Scope{
		"rowH":   w.RowH,
		"input":  d.Input(host, field.Key),
		"note":   d.notes[field.Key],
		"edited": func() { d.Edited(host, field.Key) },
	})
}

// Reset puts the default back for key and drops it from the file.
func (d *SettingsDialog) Reset(host Host, key string) {
	host.Send(prefs.Change{Edit: func(s *settings.Settings) { settings.Reset(s, key) }})
	d.inputs, d.notes = nil, nil
	if key == "uiFont" || key == "codeFont" {
		d.ApplyFonts(host)
	}
}

// wrapWords breaks text into lines of at most cols characters.
func wrapWords(text string, cols int) string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		if line != "" && len(line)+1+len(word) > cols {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	return strings.Join(append(lines, line), "\n")
}

// settingRow is one setting: its title with a mark and a Reset button when
// it differs from the default, its description and its control.
func (d *SettingsDialog) settingRow(host Host, field settings.Field) Node {
	changed := settings.Changed(host.Settings(), field.Key)
	var control Node
	switch custom, ok := settingControls[field.Key]; {
	case ok:
		control = custom(d, host, field)
	case field.Kind == settings.Enum:
		control = d.enumChips(host, field, nil)
	case field.Kind == settings.Bool:
		control = d.boolChips(host, field)
	case field.Kind == settings.Shortcuts || field.Kind == settings.Colors:
		control = d.openPageButton(field)
	case field.Kind == settings.File:
		control = d.fileControl(host, field)
	default:
		control = d.textControl(host, field)
	}
	return settingRowTemplate.MustBuild(markup.Scope{
		"rowH":     w.RowH,
		"compactH": w.CompactH,
		"setting": settingView{
			Title:       field.Title,
			Changed:     changed,
			Reset:       func() { d.Reset(host, field.Key) },
			Description: wrapWords(field.Description, descriptionCols),
			Control:     control,
		},
	})
}

// settingView is what the row template shows of one setting.
type settingView struct {
	Title       string
	Changed     bool // differs from the default: a mark and Reset
	Reset       func()
	Description string // wrapped, one line per "\n"
	Control     Node   // built by the kind's own template
}

// openPageButton leads from a search result to the page that edits the
// setting, which is a table and not a single control.
func (d *SettingsDialog) openPageButton(field settings.Field) Node {
	page := "colors"
	if field.Kind == settings.Shortcuts {
		page = "shortcuts"
	}
	open := func() {
		d.Cat, d.Search = page, NewTextField("")
		d.scroll.Y = 0
	}
	return openPageTemplate.MustBuild(markup.Scope{
		"rowH":  w.RowH,
		"label": "Open " + strings.ToLower(field.Category),
		"open":  open,
	})
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
func (d *SettingsDialog) FormPage(host Host, category string) Node {
	searching := d.Searching()
	var rows []Node
	last := ""
	for _, field := range settings.Schema {
		if field.Hidden || !settingMatches(field, d.Search.Text) || (!searching && field.Category != category) {
			continue
		}
		if searching && field.Category != last {
			last = field.Category
			rows = append(rows, formCaptionTemplate.MustBuild(markup.Scope{"category": field.Category, "labelH": w.LabelH}))
		}
		rows = append(rows, d.settingRow(host, field))
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
func (d *SettingsDialog) fileControl(host Host, field settings.Field) Node {
	input := d.Input(host, field.Key)
	choose := func() { d.ChooseFile(host, field.Key) }
	if strings.TrimSpace(input.Text) == "" && !input.Focused {
		return emptyFileTemplate.MustBuild(markup.Scope{"rowH": w.RowH, "choose": choose})
	}
	return chosenFileTemplate.MustBuild(markup.Scope{
		"rowH":     w.RowH,
		"compactH": w.CompactH,
		"iconBtnW": w.IconBtnW,
		"input":    input,
		"note":     d.notes[field.Key],
		"edited":   func() { d.Edited(host, field.Key) },
		"choose":   choose,
	})
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
