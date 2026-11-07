package settingsui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// fakeHost keeps the settings in memory and records what the dialog asked.
type fakeHost struct {
	current settings.Settings
	path    string
	toasts  []string
	menus   [][]MenuItem
	saved   int
}

func newHost(t *testing.T) *fakeHost {
	return &fakeHost{current: settings.Default(), path: filepath.Join(t.TempDir(), "settings.json")}
}

func (h *fakeHost) Size() (int, int)            { return 1280, 800 }
func (h *fakeHost) Settings() settings.Settings { return h.current }
func (h *fakeHost) Change(edit func(*settings.Settings)) {
	edit(&h.current)
	_ = h.current.Save(h.path)
}
func (h *fakeHost) SetScaleLive(percent int)            { h.current.UIScalePct = percent }
func (h *fakeHost) Save()                               { h.saved++ }
func (h *fakeHost) Replace(s settings.Settings)         { h.current = s }
func (h *fakeHost) SettingsPath() string                { return h.path }
func (h *fakeHost) SetFonts(ui, code w.FontData) error  { return nil }
func (h *fakeHost) Toast(text string, _ bool)           { h.toasts = append(h.toasts, text) }
func (h *fakeHost) SetStatus(string)                    {}
func (h *fakeHost) TextMenu(*TextField, func()) Option  { return func(*El) {} }
func (h *fakeHost) OpenMenu(_, _ int, items []MenuItem) { h.menus = append(h.menus, items) }
func (h *fakeHost) ExtraPage(id string) Node            { return Text("page " + id) }
func (h *fakeHost) ActionIDs() []string                 { return []string{"send"} }
func (h *fakeHost) Close()                              {}
func (h *fakeHost) MarkDirty()                          {}
func (h *fakeHost) Later(f func())                      { f() }

func dump(node Node) string { return Dump(node) }

func TestAPageShowsItsSettingsAndTheSearchFiltersAcrossPages(t *testing.T) {
	var d Dialog
	host := newHost(t)
	if text := dump(d.FormPage(host, "Requests")); !strings.Contains(text, "Proxy") || strings.Contains(text, "Corners") {
		t.Fatalf("Requests page:\n%s", text)
	}
	d.Search = NewTextField("corners")
	if text := dump(d.FormPage(host, "Requests")); !strings.Contains(text, "Corners") || strings.Contains(text, "Proxy") {
		t.Fatalf("search results:\n%s", text)
	}
}

func TestANumberOutsideItsRangeIsExplainedAndNotApplied(t *testing.T) {
	var d Dialog
	host := newHost(t)
	*d.Input(host, "timeoutSec") = NewTextField("0")
	d.Edited(host, "timeoutSec")
	if host.current.TimeoutSec != 60 || d.Note("timeoutSec") == "" {
		t.Fatalf("timeout %d note %q", host.current.TimeoutSec, d.Note("timeoutSec"))
	}
	*d.Input(host, "timeoutSec") = NewTextField("20")
	d.Edited(host, "timeoutSec")
	if host.current.TimeoutSec != 20 || d.Note("timeoutSec") != "" {
		t.Fatalf("timeout %d note %q", host.current.TimeoutSec, d.Note("timeoutSec"))
	}
}

func TestResetRemovesTheKeyFromTheFile(t *testing.T) {
	var d Dialog
	host := newHost(t)
	host.Change(func(s *settings.Settings) { s.Layout = "stacked" })
	d.Reset(host, "layout")
	if data, _ := os.ReadFile(host.path); strings.Contains(string(data), "layout") || host.current.Layout != "side" {
		t.Fatalf("layout %q, file %q", host.current.Layout, data)
	}
}

func TestRawTextWithProblemsChangesNothingAndGoodTextIsWrittenAsTyped(t *testing.T) {
	var d Dialog
	host := newHost(t)
	d.OpenRaw(host)
	d.RawText.Text = `{"layout": "diagonal"}`
	d.RawEdited(host)
	if host.current.Layout != "side" || len(d.Problems) != 1 {
		t.Fatalf("layout %q problems %+v", host.current.Layout, d.Problems)
	}
	d.RawText.Text = "{\n  \"layout\": \"stacked\"\n}"
	d.RawEdited(host)
	if host.current.Layout != "stacked" || len(d.Problems) != 0 {
		t.Fatalf("layout %q problems %+v", host.current.Layout, d.Problems)
	}
	if data, _ := os.ReadFile(host.path); string(data) != d.RawText.Text {
		t.Fatalf("file %q", data)
	}
}

func TestAChangeOnDiskIsPickedUpOnceAndAnOwnWriteIsNot(t *testing.T) {
	var d Dialog
	host := newHost(t)
	host.Change(func(s *settings.Settings) { s.Corners = "round" })
	d.Stamp(host.path)
	host.current.Corners = "square" // unsaved: an own write must not reload over it
	d.Poll(host)
	if host.current.Corners != "square" {
		t.Fatal("the dialog reloaded the file it had just written")
	}
	if err := os.WriteFile(host.path, []byte(`{"corners": "soft"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	d.Poll(host)
	if host.current.Corners != "soft" {
		t.Fatalf("Corners = %q after an outside change", host.current.Corners)
	}
}

func TestCompletionOffersWhatSettingsCompleteGivesAtTheCaret(t *testing.T) {
	var d Dialog
	host := newHost(t)
	d.OpenRaw(host)
	d.RawText = NewTextArea(`{"la`)
	d.RawText.Cursor = 4
	d.CompleteRaw(host)
	if len(host.menus) != 1 || len(host.menus[0]) != 1 || host.menus[0][0].Label != "layout" {
		t.Fatalf("menus = %+v", host.menus)
	}
	host.menus[0][0].Run()
	if d.RawText.Text != `{"layout"` {
		t.Fatalf("text = %q", d.RawText.Text)
	}
}
func (h *fakeHost) Pointer() (int, int) { return 400, 300 }
