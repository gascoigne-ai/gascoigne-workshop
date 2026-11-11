package settingsui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// PollTicks is how often (in frames) the settings file is checked for a
// change made outside the program.
const PollTicks = 30

// Stamp notes the file as it is now, so the program's own write is not
// taken for an outside change.
func (d *SettingsDialog) Stamp(host Host, path string) {
	if path != "" {
		d.stamp = host.SettingsStore().Stat(path)
	}
}

// Poll applies a change made to settings.json outside the program, to the
// form and to the open raw view alike. A file with problems changes nothing
// and is reported.
func (d *SettingsDialog) Poll(host Host) {
	path := host.SettingsPath()
	if path == "" || host.Unsaved() { // an outside change must not overwrite what was not saved
		return
	}
	now := host.SettingsStore().Stat(path)
	if now == d.stamp || !now.Exists {
		return
	}
	d.stamp = now
	data, err := host.SettingsStore().ReadText(path)
	if err != nil {
		host.Send(prefs.Toast{Text: "Cannot read settings.json: " + err.Error(), IsErr: true})
		return
	}
	parsed, problems := settings.Parse(data, host.Settings())
	if d.Raw {
		d.RawText = NewTextArea(string(data))
		d.RawText.Styler = d.rawStyler
		d.Problems = problems
	}
	if len(problems) > 0 {
		host.Send(prefs.Toast{Text: "settings.json has problems and was not applied: " + problems[0].Message, IsErr: true})
		return
	}
	host.Send(prefs.Replace{Settings: parsed})
}

// OpenRaw switches the dialog to the text of settings.json.
func (d *SettingsDialog) OpenRaw(host Host) {
	text := ""
	if data, err := host.SettingsStore().ReadText(host.SettingsPath()); host.SettingsPath() != "" && err == nil {
		text = string(data)
	} else if minimal, err := settings.Minimal(host.Settings()); err == nil {
		text = string(minimal) + "\n"
	}
	d.Raw = true
	d.RawText = NewTextArea(text)
	d.RawText.Styler = d.rawStyler
	d.Problems = settings.Validate([]byte(text))
	d.rawBefore = text
}

// CloseRaw switches back to the form.
func (d *SettingsDialog) CloseRaw() { d.Raw = false }

// RawChanged runs after every change of the raw text: the edit is checked
// and applied (RawEdited), and typing the opening quote of a key or a value
// offers what fits there without Ctrl+Space.
func (d *SettingsDialog) RawChanged(host Host) {
	before := d.rawBefore
	d.rawBefore = d.RawText.Text
	d.RawEdited(host)
	runes := []rune(d.RawText.Text)
	caret := d.RawText.Cursor
	typedQuote := len(runes) == len([]rune(before))+1 && caret > 0 && caret <= len(runes) && runes[caret-1] == '"'
	if typedQuote && d.RawText.Focused {
		d.complete(host, true)
	}
}

// RawEdited runs after every change of the raw text. Text without problems
// becomes the settings and is written to the file as typed; text with
// problems changes nothing, and the problems are listed under the editor.
func (d *SettingsDialog) RawEdited(host Host) {
	text := d.RawText.Text
	d.Problems = settings.Validate([]byte(text))
	if len(d.Problems) > 0 {
		return
	}
	parsed, _ := settings.Parse([]byte(text), host.Settings())
	host.Send(prefs.Replace{Settings: parsed})
	path := host.SettingsPath()
	if path == "" {
		return
	}
	if !parsed.AutoSave { // held back until the user saves
		d.rawPending = true
		host.Send(prefs.MarkUnsaved{})
		return
	}
	d.rawPending = false
	if err := host.SettingsStore().WriteText(path, []byte(text)); err != nil {
		host.Send(prefs.Toast{Text: "Could not save settings: " + err.Error(), IsErr: true})
		return
	}
	d.Stamp(host, path)
}

// rawStyler colors one line of the text as JSON, and the keys that have a
// problem in the error color.
func (d *SettingsDialog) rawStyler(line string) []Span {
	bad := map[string]bool{}
	for _, problem := range d.Problems {
		if problem.Key != "" {
			bad[`"`+problem.Key+`"`] = true
		}
	}
	tokens := jsonview.Tokenize(line)
	spans := make([]Span, len(tokens))
	for i, token := range tokens {
		style := w.Token(token.Kind.Style())
		if token.Kind == jsonview.Key && bad[token.Text] {
			style = w.Error
		}
		spans[i] = Span{Text: token.Text, Style: style}
	}
	return spans
}

// RawStyle is how the raw view colors one line of its text.
func (d *SettingsDialog) RawStyle(line string) []Span { return d.rawStyler(line) }

// CompleteRaw opens a menu of what fits at the caret of the raw text; the
// choice replaces the part of the word that was typed.
func (d *SettingsDialog) CompleteRaw(host Host) { d.complete(host, false) }

// complete opens the completion menu; quiet leaves the user alone when
// there is nothing to offer, as when it opened by itself.
func (d *SettingsDialog) complete(host Host, quiet bool) {
	text := d.RawText.Text
	runes := []rune(text)
	caret := min(d.RawText.Cursor, len(runes))
	suggestions := settings.Complete(text, len(string(runes[:caret])), host.ActionIDs())
	if len(suggestions) == 0 {
		if !quiet {
			host.Send(prefs.Toast{Text: "Nothing to complete here", IsErr: false})
		}
		return
	}
	items := make([]prefs.MenuItem, len(suggestions))
	for i, suggestion := range suggestions {
		items[i] = prefs.MenuItem{Label: suggestion.Label, Hint: suggestion.Detail, Run: func() {
			from := len([]rune(text[:suggestion.From]))
			d.RawText.Text = string(runes[:from]) + suggestion.Insert + string(runes[caret:])
			d.RawText.Cursor = from + len([]rune(suggestion.Insert))
			d.RawText.Focused = true
			d.RawEdited(host)
		}}
	}
	if x, y, ok := d.RawText.CaretAt(); ok {
		host.Send(prefs.OpenMenu{X: x, Y: y, Items: items}) // just below the caret
		return
	}
	width, height := host.Size()
	host.Send(prefs.OpenMenu{X: width/2 - DialogWidth/2 + DialogNavW + 40, Y: height/2 - DialogMaxH/4, Items: items})
}

// RawFocused reports whether the raw text has the keyboard focus.
func (d *SettingsDialog) RawFocused() bool { return d.Raw && d.RawText.Focused }

// rawView is the dialog's page while it shows settings.json as text.
func (d *SettingsDialog) rawView(host Host) Node {
	status, color := "Saved as you type", w.Muted
	if len(d.Problems) > 0 {
		status, color = fmt.Sprintf("%d problem(s): the last good settings stay in effect", len(d.Problems)), w.Error
	}
	edited := func() { d.RawChanged(host) }
	return Col(Grow(1), Gap(6),
		Row(H(w.RowH), Gap(6),
			Text("settings.json", Bold()),
			Spacer(),
			Text(status, Small(), Color(color)),
		),
		Area(&d.RawText, Grow(1), Bg(w.Sunken), OnChange(edited), TextMenu(&d.RawText.TextField)),
		For(d.Problems, func(_ int, problem settings.Problem) Node {
			prefix := ""
			if problem.Key != "" {
				prefix = problem.Key + ": "
			}
			return Note(prefix+problem.Message, w.Error)
		}),
		Text(w.PlatformKeyText("linux", "Ctrl+Space")+" completes keys and values", Small(), Muted()),
	)
}
