package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// settingsPollTicks is how often (in frames) the settings file is checked
// for a change made outside the program.
const settingsPollTicks = 30

// fileStamp is what identifies a version of settings.json cheaply.
type fileStamp struct {
	modified time.Time
	size     int64
	exists   bool
}

func statFile(path string) fileStamp {
	info, err := os.Stat(path)
	if err != nil {
		return fileStamp{}
	}
	return fileStamp{modified: info.ModTime(), size: info.Size(), exists: true}
}

// stampSettings notes the file as it is now, so our own write is not taken
// for an outside change.
func (a *App) stampSettings() {
	if a.settingsPath != "" {
		a.settingsStamp = statFile(a.settingsPath)
	}
}

// pollSettingsFile applies a change made to settings.json outside the
// program, to the form and to the open raw view alike. A file with
// problems changes nothing and is reported.
func (a *App) pollSettingsFile() {
	if a.settingsPath == "" {
		return
	}
	now := statFile(a.settingsPath)
	if now == a.settingsStamp || !now.exists {
		return
	}
	a.settingsStamp = now
	data, err := os.ReadFile(a.settingsPath)
	if err != nil {
		a.toast("Cannot read settings.json: "+err.Error(), true)
		return
	}
	parsed, problems := settings.Parse(data, a.settings)
	if a.rawMode {
		a.rawText = NewTextArea(string(data))
		a.rawText.Styler = a.rawStyler
		a.rawProblems = problems
	}
	if len(problems) > 0 {
		a.toast("settings.json has problems and was not applied: "+problems[0].Message, true)
		return
	}
	a.applySettings(parsed)
}

// applySettings makes s the settings in effect, with everything that
// follows from them.
func (a *App) applySettings(s settings.Settings) {
	a.settings = s
	if err := a.ctx.SetTheme(appearance.ThemeWith(s)); err != nil {
		a.status = err.Error()
	}
	for _, t := range a.tabs {
		t.raw, t.compact = viewFlags(s.ResponseView)
		a.refreshView(t)
	}
	a.colorInputs, a.colorNotes = nil, nil
	a.settingInputs, a.settingNotes = nil, nil
	a.uiFont, a.codeFont = NewTextField(s.UIFont), NewTextField(s.CodeFont)
	a.applyFonts()
	a.dirty = true
}

// openRawSettings switches the dialog to the text of settings.json.
func (a *App) openRawSettings() {
	text := ""
	if data, err := os.ReadFile(a.settingsPath); a.settingsPath != "" && err == nil {
		text = string(data)
	} else if minimal, err := settings.Minimal(a.settings); err == nil {
		text = string(minimal) + "\n"
	}
	a.rawMode = true
	a.rawText = NewTextArea(text)
	a.rawText.Styler = a.rawStyler
	a.rawProblems = settings.Validate([]byte(text))
}

func (a *App) closeRawSettings() { a.rawMode = false }

// rawEdited runs after every change of the raw text. Text without problems
// becomes the settings and is written to the file as typed; text with
// problems changes nothing, and the problems are listed under the editor.
func (a *App) rawEdited() {
	text := a.rawText.Text
	a.rawProblems = settings.Validate([]byte(text))
	if len(a.rawProblems) > 0 {
		return
	}
	parsed, _ := settings.Parse([]byte(text), a.settings)
	a.applySettings(parsed)
	if a.settingsPath == "" {
		return
	}
	if err := settings.WriteText(a.settingsPath, []byte(text)); err != nil {
		a.toast("Could not save settings: "+err.Error(), true)
		return
	}
	a.stampSettings()
}

// rawStyler colors the keys that have a problem in the error color.
func (a *App) rawStyler(line string) []Span {
	var spans []Span
	rest := line
	for rest != "" {
		start, end := nextBadKey(rest, a.rawProblems)
		if start < 0 {
			break
		}
		if start > 0 {
			spans = append(spans, Span{Text: rest[:start], Style: w.Text})
		}
		spans = append(spans, Span{Text: rest[start:end], Style: w.Error})
		rest = rest[end:]
	}
	if rest != "" {
		spans = append(spans, Span{Text: rest, Style: w.Text})
	}
	return spans
}

// nextBadKey finds the first quoted key of a problem in text.
func nextBadKey(text string, problems []settings.Problem) (start, end int) {
	start = -1
	for _, problem := range problems {
		if problem.Key == "" {
			continue
		}
		quoted := `"` + problem.Key + `"`
		if i := strings.Index(text, quoted); i >= 0 && (start < 0 || i < start) {
			start, end = i, i+len(quoted)
		}
	}
	return start, end
}

// completeRaw opens a menu of what fits at the caret of the raw text; the
// choice replaces the part of the word that was typed.
func (a *App) completeRaw() {
	text := a.rawText.Text
	runes := []rune(text)
	caret := min(a.rawText.Cursor, len(runes))
	suggestions := settings.Complete(text, len(string(runes[:caret])), a.actionIDs)
	if len(suggestions) == 0 {
		a.toast("Nothing to complete here", false)
		return
	}
	items := make([]menuItem, len(suggestions))
	for i, suggestion := range suggestions {
		items[i] = menuItem{label: suggestion.Label, run: func() {
			from := len([]rune(text[:suggestion.From]))
			a.rawText.Text = string(runes[:from]) + suggestion.Insert + string(runes[caret:])
			a.rawText.Cursor = from + len([]rune(suggestion.Insert))
			a.rawText.Focused = true
			a.rawEdited()
		}}
	}
	a.openMenu(a.w/2-settingsW/2+settingsNavW+40, a.h/2-settingsMaxH/4, items)
}

// rawView is the dialog's page while it shows settings.json as text.
func (a *App) rawView() Node {
	status, color := "Saved as you type", w.Muted
	if len(a.rawProblems) > 0 {
		status, color = fmt.Sprintf("%d problem(s): the last good settings stay in effect", len(a.rawProblems)), w.Error
	}
	return Col(Grow(1), Gap(6),
		Row(H(w.RowH), Gap(6),
			Text("settings.json", Bold()),
			Spacer(),
			Text(status, Small(), Color(color)),
		),
		Area(&a.rawText, Grow(1), Bg(w.Sunken), OnChange(a.rawEdited), a.textMenuOpt(&a.rawText.TextField, a.rawEdited)),
		For(a.rawProblems, func(_ int, problem settings.Problem) Node {
			prefix := ""
			if problem.Key != "" {
				prefix = problem.Key + ": "
			}
			return Row(H(w.NoteH), Text(prefix+problem.Message, Small(), Color(w.Error), Ellipsis(), Grow(1)))
		}),
		Text(w.PlatformKeyText("linux", "Ctrl+Space")+" completes keys and values", Small(), Muted()),
	)
}
