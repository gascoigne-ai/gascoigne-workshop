package response

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// toastCall is one message the component showed.
type toastCall struct {
	text  string
	level severity.Level
}

// saveCall is one file the component asked to be written.
type saveCall struct {
	path string
	body []byte
}

// fake is every port of the component in one test double: the open tabs, the
// settings it reads and what it was asked to do.
type fake struct {
	set     *tabs.Set
	mode    string // the response view setting
	now     time.Time
	home    string
	widths  widget.TableState
	curl    string
	preview error

	committed int         // CommitEdit calls
	shown     []*tabs.Tab // ShowResponse calls
	menus     [][]widget.MenuItem
	toasts    []toastCall
	picked    []string // "title|suggested" of each PickSave
	pickedTo  func(path string)
	saved     []saveCall
	writeErr  error
	previewed []string // content type of each Preview
}

func (fake *fake) Current() *tabs.Tab { return fake.set.Current() }

func (fake *fake) IsOpen(tab *tabs.Tab) bool { return slices.Contains(fake.set.Tabs(), tab) }

func (fake *fake) Results(*tabs.Tab, *widget.ScrollState) view.Node {
	return view.Col(view.Text("results"))
}

func (fake *fake) Widths() *widget.TableState { return &fake.widths }

func (fake *fake) CellStyler(*request.Request) func(string) []widget.Span { return nil }

func (fake *fake) PickSave(title, suggested string, answer func(path string)) {
	fake.picked = append(fake.picked, title+"|"+suggested)
	fake.pickedTo = answer
}

func (fake *fake) Write(path string, body []byte, done func(error)) {
	fake.saved = append(fake.saved, saveCall{path, body})
	done(fake.writeErr)
}

func (fake *fake) Home() (string, error) { return fake.home, nil }

func (fake *fake) Curl(*tabs.Tab) string { return fake.curl }

func (fake *fake) Preview(_ context.Context, _ []byte, contentType string) error {
	fake.previewed = append(fake.previewed, contentType)
	return fake.preview
}

func (fake *fake) CommitEdit() { fake.committed++ }

func (fake *fake) ShowResponse(tab *tabs.Tab) { fake.shown = append(fake.shown, tab) }

func (fake *fake) Open(_, _ int, items []widget.MenuItem) { fake.menus = append(fake.menus, items) }

func (fake *fake) Toast(text string, level severity.Level) {
	fake.toasts = append(fake.toasts, toastCall{text, level})
}

func (fake *fake) Hint(label, action string) string {
	if action == "find" {
		return label + " (Ctrl+F)"
	}
	return label
}

func (fake *fake) Shortcut(action string) string {
	if action == "find" {
		return "Ctrl+F"
	}
	return ""
}

func (fake *fake) ResponseView() string { return fake.mode }

func (fake *fake) Now() time.Time { return fake.now }

// world is the component over a fake, with one open request.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	pane *Component
	tab  *tabs.Tab
}

func newWorld(t *testing.T) *world {
	t.Helper()
	prefs := settings.Default()
	ctx, err := core.NewCtx(theme.ThemeWith(prefs))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet(), mode: "pretty", now: time.Unix(1_700_000_000, 0)}
	fake.set.Open(&request.Request{Name: "R", URL: "https://api.example.test/users"})
	pane := New(Deps{Tabs: fake, Slots: fake, Headers: fake, Files: fake, Exports: fake, Window: fake, Hints: fake, Defaults: fake})
	return &world{t: t, ctx: ctx, fake: fake, pane: pane, tab: fake.set.Current()}
}

// clipboard keeps the last text written to it.
type clipboard struct{ text string }

func (clip *clipboard) Read() (string, error) { return clip.text, nil }

func (clip *clipboard) Write(text string) error {
	clip.text = text
	return nil
}
