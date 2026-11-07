package settingsui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// descriptionCols is how many characters a setting's description may have
// on one line before it is wrapped.
const descriptionCols = 76

// settingControl draws the control of one setting.
type settingControl func(d *Dialog, host Host, field settings.Field) Node

// settingControls are the settings whose control is not the plain one for
// their kind. Everything else is a row of chips (an enum), On / Off (a
// boolean) or a text field (text and numbers).
var settingControls = map[string]settingControl{
	"theme": func(d *Dialog, host Host, _ settings.Field) Node { return d.themeChips(host) },
	"textSize": func(d *Dialog, host Host, field settings.Field) Node {
		return d.enumChips(host, field, labelsOf(appearance.TextSizeChoices, sizeLabel))
	},
	"corners": func(d *Dialog, host Host, field settings.Field) Node {
		return d.enumChips(host, field, labelsOf(appearance.CornerChoices, cornerLabel))
	},
	"contrast": func(d *Dialog, host Host, field settings.Field) Node {
		return d.enumChips(host, field, labelsOf(appearance.ContrastChoices, contrastLabel))
	},
	"layout": func(d *Dialog, host Host, field settings.Field) Node {
		return d.enumChips(host, field, labelsOf(appearance.LayoutChoices, layoutLabel))
	},
	"responseView": func(d *Dialog, host Host, field settings.Field) Node { return d.enumChips(host, field, nil) },
	"uiScale":      func(d *Dialog, host Host, _ settings.Field) Node { return d.scaleControl(host) },
	"uiFont":       func(d *Dialog, host Host, _ settings.Field) Node { return d.fontControl(host, FontUI) },
	"codeFont":     func(d *Dialog, host Host, _ settings.Field) Node { return d.fontControl(host, FontCode) },
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

// choice is one chip of a segmented choice: gold when selected.
func choice(host Host, label string, selected bool, edit func(*settings.Settings)) Node {
	apply := OnClick(func() { host.Change(edit) })
	if selected {
		return PrimaryButton(label, Grow(1), apply)
	}
	return Button(label, Grow(1), apply)
}

// enumChips is a row of chips, one per allowed value; the current one is lit.
func (d *Dialog) enumChips(host Host, field settings.Field, labels map[string]string) Node {
	current, _ := settings.Value(host.Settings(), field.Key).(string)
	if field.Key == "corners" {
		current = appearance.CornerIDOf(current)
	}
	return Row(H(w.RowH), Gap(6),
		For(field.Allowed, func(_ int, id string) Node {
			label := labels[id]
			if label == "" {
				label = capitalized(id)
			}
			return choice(host, label, id == current, func(s *settings.Settings) { *s = settings.With(*s, field.Key, id) })
		}),
	)
}

func (d *Dialog) themeChips(host Host) Node {
	current := host.Settings().Theme
	return Row(H(w.RowH), Gap(6),
		For(appearance.AllThemeChoices(), func(_ int, c appearance.ThemeChoice) Node {
			return choice(host, c.Label, c.ID == current, func(s *settings.Settings) { s.Theme = c.ID })
		}),
	)
}

func (d *Dialog) scaleControl(host Host) Node {
	percent := int(host.Settings().UIScale()*100 + 0.5)
	return Row(H(w.RowH), Gap(8),
		Slider(&d.scale, appearance.ScaleMin, appearance.ScaleMax, appearance.ScaleStep, percent, OnSlide(host.SetScaleLive), // live: the UI resizes while dragging
			SliderMid(settings.DefaultUIScalePct), OnSlideEnd(func(int) { host.Save() })),
		Text(fmt.Sprintf("%d%%", d.scale.Shown(percent)), W(40), Color(w.Muted)),
	)
}

// boolChips is Off and On for a true-or-false setting.
func (d *Dialog) boolChips(host Host, field settings.Field) Node {
	on, _ := settings.Value(host.Settings(), field.Key).(bool)
	set := func(value bool) func(*settings.Settings) {
		return func(s *settings.Settings) { *s = settings.With(*s, field.Key, value) }
	}
	return Row(H(w.RowH), Gap(6), choice(host, "Off", !on, set(false)), choice(host, "On", on, set(true)))
}

// Input is the text field of a text or number setting, filled from the
// setting the first time it is needed.
func (d *Dialog) Input(host Host, key string) *TextField {
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
func (d *Dialog) Edited(host Host, key string) {
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
	host.Change(func(s *settings.Settings) { *s = settings.With(*s, key, value) })
}

func (d *Dialog) setNote(key, note string) {
	if d.notes == nil {
		d.notes = map[string]string{}
	}
	d.notes[key] = note
}

// Note is why the value typed for key was refused, "" when it was not.
func (d *Dialog) Note(key string) string { return d.notes[key] }

// textControl is the field of a text or number setting.
func (d *Dialog) textControl(host Host, field settings.Field) Node {
	input := d.Input(host, field.Key)
	note := d.notes[field.Key]
	edited := func() { d.Edited(host, field.Key) }
	return Col(
		Field(input, H(w.RowH), Bg(w.Sunken), Placeholder("Default"), OnChange(edited), host.TextMenu(input, edited)),
		If(note != "", Row(H(w.NoteH), Text(note, Small(), Color(w.Error), Ellipsis(), Grow(1)))),
	)
}

// Reset puts the default back for key and drops it from the file.
func (d *Dialog) Reset(host Host, key string) {
	host.Change(func(s *settings.Settings) { settings.Reset(s, key) })
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
func (d *Dialog) settingRow(host Host, field settings.Field) Node {
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
	return Col(PadY(8), BorderB(w.BorderSoft),
		Row(H(w.RowH), Gap(6),
			Text(field.Title, Bold()),
			If(changed, Text("•", Color(w.AccentLine), Tooltip("Changed from the default"))),
			Spacer(),
			If(changed, Button("Reset", W(56), H(w.CompactH), OnClick(func() { d.Reset(host, field.Key) }), Tooltip("Remove it from settings.json so the default applies"))),
		),
		Lines(wrapWords(field.Description, descriptionCols), w.Muted),
		control,
	)
}

// openPageButton leads from a search result to the page that edits the
// setting, which is a table and not a single control.
func (d *Dialog) openPageButton(field settings.Field) Node {
	page := "colors"
	if field.Kind == settings.Shortcuts {
		page = "shortcuts"
	}
	return Row(H(w.RowH), Button("Open "+strings.ToLower(field.Category), W(120), OnClick(func() {
		d.Cat, d.Search = page, NewTextField("")
		d.scroll.Y = 0
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

// FormPage is the settings of one category, or, while a search text is
// typed, those of every category that match it.
func (d *Dialog) FormPage(host Host, category string) Node {
	searching := d.Searching()
	var rows []Item
	last := ""
	for _, field := range settings.Schema {
		if field.Hidden || !settingMatches(field, d.Search.Text) || (!searching && field.Category != category) {
			continue
		}
		if searching && field.Category != last {
			last = field.Category
			rows = append(rows, Caption(field.Category, H(w.LabelH)))
		}
		rows = append(rows, d.settingRow(host, field))
	}
	if len(rows) == 0 {
		return Text("No settings match", Muted())
	}
	return Col(rows...)
}

// fileControl is the control of a file setting. With no file it is one
// button, a folder icon and "Choose file", since an empty field has nothing
// to show; with a file it is a field with the path and a folder button inside
// it at the right, which chooses another.
func (d *Dialog) fileControl(host Host, field settings.Field) Node {
	input := d.Input(host, field.Key)
	choose := func() { d.ChooseFile(host, field.Key) }
	if strings.TrimSpace(input.Text) == "" && !input.Focused {
		return Row(H(w.RowH),
			Row(W(150), H(w.RowH), Bg(w.Sunken), Control("button"), Lighten(), PadX(10), Gap(8), AlignItems(AlignCenter),
				OnClick(choose), Tooltip("Choose the file"),
				FolderIcon(W(16)), Text("Choose file", Grow(1), Color(w.ChipText)),
			),
			Spacer(),
		)
	}
	note := d.notes[field.Key]
	edited := func() { d.Edited(host, field.Key) }
	return Col(
		Row(H(w.RowH), Bg(w.Sunken), Control("field"), AlignItems(AlignCenter),
			Field(input, Grow(1), Placeholder("No file"), OnChange(edited), host.TextMenu(input, edited)),
			IconButton(FolderIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(choose), Tooltip("Choose another file")),
		),
		If(note != "", Row(H(w.NoteH), Text(note, Small(), Color(w.Error), Ellipsis(), Grow(1)))),
	)
}

// ChooseFile asks for a file for the setting key and puts its path in the
// setting.
func (d *Dialog) ChooseFile(host Host, key string) {
	field, ok := settings.FieldFor(key)
	if !ok {
		return
	}
	host.PickFile("Choose "+field.Title, func(path string) {
		*d.Input(host, key) = NewTextField(path)
		d.Edited(host, key)
	})
}
