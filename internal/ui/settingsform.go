package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// descriptionCols is how many characters a setting's description may have
// on one line before it is wrapped.
const descriptionCols = 76

// settingControl draws the control of one setting.
type settingControl func(a *App, field settings.Field) Node

// settingControls are the settings whose control is not the plain one for
// their kind. Everything else is a row of chips (an enum), On / Off (a
// boolean) or a text field (text and numbers).
var settingControls = map[string]settingControl{
	"theme": func(a *App, _ settings.Field) Node { return a.themeChips() },
	"textSize": func(a *App, field settings.Field) Node {
		return a.enumChips(field, labelsOf(appearance.TextSizeChoices, sizeLabel))
	},
	"corners": func(a *App, field settings.Field) Node {
		return a.enumChips(field, labelsOf(appearance.CornerChoices, cornerLabel))
	},
	"contrast": func(a *App, field settings.Field) Node {
		return a.enumChips(field, labelsOf(appearance.ContrastChoices, contrastLabel))
	},
	"layout": func(a *App, field settings.Field) Node {
		return a.enumChips(field, labelsOf(appearance.LayoutChoices, layoutLabel))
	},
	"responseView": func(a *App, field settings.Field) Node { return a.enumChips(field, nil) },
	"uiScale":      func(a *App, _ settings.Field) Node { return a.scaleControl() },
	"uiFont":       func(a *App, _ settings.Field) Node { return a.fontPicker(fontUI, &a.uiFont) },
	"codeFont":     func(a *App, _ settings.Field) Node { return a.fontPicker(fontCode, &a.codeFont) },
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

// enumChips is a row of chips, one per allowed value; the current one is lit.
func (a *App) enumChips(field settings.Field, labels map[string]string) Node {
	current, _ := settings.Value(a.settings, field.Key).(string)
	if field.Key == "corners" {
		current = appearance.CornerIDOf(current)
	}
	return Row(H(w.RowH), Gap(6),
		For(field.Allowed, func(_ int, id string) Node {
			label := labels[id]
			if label == "" {
				label = capitalized(id)
			}
			return a.choice(label, id == current, func(s *settings.Settings) { *s = settings.With(*s, field.Key, id) })
		}),
	)
}

func (a *App) themeChips() Node {
	return Row(H(w.RowH), Gap(6),
		For(appearance.AllThemeChoices(), func(_ int, c appearance.ThemeChoice) Node {
			return a.choice(c.Label, c.ID == a.settings.Theme, func(s *settings.Settings) { s.Theme = c.ID })
		}),
	)
}

func (a *App) scaleControl() Node {
	return Row(H(w.RowH), Gap(8),
		Slider(&a.scaleSlider, appearance.ScaleMin, appearance.ScaleMax, appearance.ScaleStep, a.scalePct(), OnSlide(func(v int) { a.settings.UIScalePct, a.dirty = v, true }), // live: the UI resizes while dragging
			SliderMid(settings.DefaultUIScalePct), OnSlideEnd(func(int) { a.saveSettings(func(*settings.Settings) {}) })),
		Text(fmt.Sprintf("%d%%", a.scaleSlider.Shown(a.scalePct())), W(40), Color(w.Muted)),
	)
}

// boolChips is Off and On for a true-or-false setting.
func (a *App) boolChips(field settings.Field) Node {
	on, _ := settings.Value(a.settings, field.Key).(bool)
	set := func(value bool) func(*settings.Settings) {
		return func(s *settings.Settings) { *s = settings.With(*s, field.Key, value) }
	}
	return Row(H(w.RowH), Gap(6), a.choice("Off", !on, set(false)), a.choice("On", on, set(true)))
}

// settingInput is the text field of a text or number setting, filled from
// the setting the first time it is needed.
func (a *App) settingInput(key string) *TextField {
	if a.settingInputs == nil {
		a.settingInputs = map[string]*TextField{}
	}
	if input, ok := a.settingInputs[key]; ok {
		return input
	}
	text := ""
	switch value := settings.Value(a.settings, key).(type) {
	case string:
		text = value
	case int:
		text = strconv.Itoa(value)
	}
	input := NewTextField(text)
	a.settingInputs[key] = &input
	return &input
}

// settingEdited applies the text typed into the field of key. A number
// that is not a whole number in range is explained under the field and the
// setting keeps its last good value.
func (a *App) settingEdited(key string) {
	field, ok := settings.FieldFor(key)
	if !ok {
		return
	}
	text := strings.TrimSpace(a.settingInput(key).Text)
	var value any = text
	if field.Kind == settings.Int {
		number, err := strconv.Atoi(text)
		if err != nil || number < field.Min || number > field.Max {
			a.settingNote(key, fmt.Sprintf("Enter a whole number between %d and %d", field.Min, field.Max))
			return
		}
		value = number
	}
	a.settingNote(key, "")
	a.changeSettings(func(s *settings.Settings) { *s = settings.With(*s, key, value) })
	if key == "uiFont" || key == "codeFont" {
		a.applyFonts()
	}
}

func (a *App) settingNote(key, note string) {
	if a.settingNotes == nil {
		a.settingNotes = map[string]string{}
	}
	a.settingNotes[key] = note
}

// textControl is the field of a text or number setting.
func (a *App) textControl(field settings.Field) Node {
	input := a.settingInput(field.Key)
	note := a.settingNotes[field.Key]
	return Col(
		Field(input, H(w.RowH), Bg(w.Sunken), Placeholder("Default"), OnChange(func() { a.settingEdited(field.Key) }), a.textMenuOpt(input, func() { a.settingEdited(field.Key) })),
		If(note != "", Row(H(w.NoteH), Text(note, Small(), Color(w.Error), Ellipsis(), Grow(1)))),
	)
}

// resetSetting puts the default back for key and drops it from the file.
func (a *App) resetSetting(key string) {
	a.changeSettings(func(s *settings.Settings) { settings.Reset(s, key) })
	a.settingInputs, a.settingNotes = nil, nil
	switch key {
	case "uiFont", "codeFont":
		a.uiFont, a.codeFont = NewTextField(a.settings.UIFont), NewTextField(a.settings.CodeFont)
		a.applyFonts()
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
func (a *App) settingRow(field settings.Field) Node {
	changed := settings.Changed(a.settings, field.Key)
	var control Node
	switch custom, ok := settingControls[field.Key]; {
	case ok:
		control = custom(a, field)
	case field.Kind == settings.Enum:
		control = a.enumChips(field, nil)
	case field.Kind == settings.Bool:
		control = a.boolChips(field)
	case field.Kind == settings.Shortcuts || field.Kind == settings.Colors:
		control = a.openPageButton(field)
	default:
		control = a.textControl(field)
	}
	return Col(PadY(8), BorderB(w.BorderSoft),
		Row(H(w.RowH), Gap(6),
			Text(field.Title, Bold()),
			If(changed, Text("•", Color(w.AccentLine), Tooltip("Changed from the default"))),
			Spacer(),
			If(changed, Button("Reset", W(56), H(w.CompactH), OnClick(func() { a.resetSetting(field.Key) }), Tooltip("Remove it from settings.json so the default applies"))),
		),
		Lines(wrapWords(field.Description, descriptionCols), w.Muted),
		control,
	)
}

// openPageButton leads from a search result to the page that edits the
// setting, which is a table and not a single control.
func (a *App) openPageButton(field settings.Field) Node {
	page := "colors"
	if field.Kind == settings.Shortcuts {
		page = "shortcuts"
	}
	return Row(H(w.RowH), Button("Open "+strings.ToLower(field.Category), W(120), OnClick(func() {
		a.settingsCat, a.settingsSearch = page, NewTextField("")
		a.menuScroll.Y = 0
	})))
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

// formPage is the settings of one category, or, while a search text is
// typed, those of every category that match it.
func (a *App) formPage(category string) Node {
	searching := strings.TrimSpace(a.settingsSearch.Text) != ""
	var rows []Item
	last := ""
	for _, field := range settings.Schema {
		if field.Hidden || !settingMatches(field, a.settingsSearch.Text) || (!searching && field.Category != category) {
			continue
		}
		if searching && field.Category != last {
			last = field.Category
			rows = append(rows, Caption(field.Category, H(w.LabelH)))
		}
		rows = append(rows, a.settingRow(field))
	}
	if len(rows) == 0 {
		return Text("No settings match", Muted())
	}
	return Col(rows...)
}
