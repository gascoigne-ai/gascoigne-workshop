package settingsui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// fakeHost keeps the settings in memory and records what the dialog asked.
type fakeHost struct {
	current   settings.Settings
	path      string
	toasts    []string
	menus     [][]MenuItem
	menuAt    [2]int // where the last menu opened
	unsaved   bool
	pickTitle string
	pickDone  func(path string)
	saved     int
}

func newHost(t *testing.T) *fakeHost {
	return &fakeHost{current: settings.Default(), path: filepath.Join(t.TempDir(), "settings.json")}
}

func (h *fakeHost) Size() (int, int)            { return 1280, 800 }
func (h *fakeHost) Settings() settings.Settings { return h.current }
func (h *fakeHost) Change(edit func(*settings.Settings)) {
	edit(&h.current)
	_ = settingsfile.Save(h.path, h.current)
}
func (h *fakeHost) SetScaleLive(percent int)           { h.current.UIScalePct = percent }
func (h *fakeHost) Save()                              { h.saved++ }
func (h *fakeHost) Replace(s settings.Settings)        { h.current = s }
func (h *fakeHost) SettingsPath() string               { return h.path }
func (h *fakeHost) SetFonts(ui, code w.FontData) error { return nil }
func (h *fakeHost) Toast(text string, _ bool)          { h.toasts = append(h.toasts, text) }
func (h *fakeHost) SetStatus(string)                   {}
func (h *fakeHost) TextMenu(*TextField, func()) Option { return func(*El) {} }
func (h *fakeHost) OpenMenu(x, y int, items []MenuItem) {
	h.menus = append(h.menus, items)
	h.menuAt = [2]int{x, y}
}
func (h *fakeHost) ExtraPage(id string) Node { return Text("page " + id) }
func (h *fakeHost) ActionIDs() []string      { return []string{"send"} }
func (h *fakeHost) Close()                   {}
func (h *fakeHost) MarkDirty()               {}
func (h *fakeHost) Later(f func())           { f() }

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
	d.Stamp(host, host.path)
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
	d.RawText = NewTextArea(`{"appearance": {"la`)
	d.RawText.Cursor = len([]rune(d.RawText.Text))
	d.CompleteRaw(host)
	if len(host.menus) != 1 || len(host.menus[0]) != 1 || host.menus[0][0].Label != "layout" {
		t.Fatalf("menus = %+v", host.menus)
	}
	host.menus[0][0].Run()
	if d.RawText.Text != `{"appearance": {"layout"` {
		t.Fatalf("text = %q", d.RawText.Text)
	}
}
func (h *fakeHost) Pointer() (int, int)               { return 400, 300 }
func (h *fakeHost) Unsaved() bool                     { return h.unsaved }
func (h *fakeHost) Persist()                          { _ = settingsfile.Save(h.path, h.current); h.unsaved = false; h.saved++ }
func (h *fakeHost) SettingsStore() port.SettingsStore { return settingsfile.Files{} }
func (h *fakeHost) MarkUnsaved()                      { h.unsaved = true }
func (h *fakeHost) MarkSaved()                        { h.unsaved = false }

func TestAFileSettingHasAFolderButtonInsideItsFieldThatFillsIt(t *testing.T) {
	var d Dialog
	host := newHost(t)
	page := dump(d.FormPage(host, "Requests"))
	if strings.Count(page, "icon=folder") != 3 {
		t.Fatalf("each of the three certificate settings needs a folder button:\n%s", page)
	}
	d.ChooseFile(host, "caCert")
	if host.pickTitle == "" || !strings.Contains(host.pickTitle, "CA certificates") {
		t.Fatalf("the picker title is %q", host.pickTitle)
	}
	host.pickDone("/etc/ssl/ca.pem")
	if host.current.CACert != "/etc/ssl/ca.pem" || d.Input(host, "caCert").Text != "/etc/ssl/ca.pem" {
		t.Fatalf("CACert %q, field %q", host.current.CACert, d.Input(host, "caCert").Text)
	}
}
func (h *fakeHost) PickFile(title string, done func(path string)) {
	h.pickTitle, h.pickDone = title, done
}

func TestTheSaveButtonIsHiddenWhileAutoSaveIsOnAndNothingIsPending(t *testing.T) {
	var d Dialog
	host := newHost(t)
	if d.saveButton(host) != nil {
		t.Fatal("with auto-save on and nothing pending there is nothing to save, so no button")
	}
	host.current.AutoSave = false
	if !strings.Contains(dump(d.saveButton(host)), "onClick=true") {
		t.Fatalf("with auto-save off the Save button must work:\n%s", dump(d.saveButton(host)))
	}
	host.unsaved = true
	host.current.AutoSave = true // changes pending even though the option is back on: Save stays available
	if !strings.Contains(dump(d.saveButton(host)), "onClick=true") {
		t.Fatal("Save is hidden although changes are pending")
	}
}

func TestAnEmptyFileSettingIsOnlyAButtonAndAFilledOneShowsItsPath(t *testing.T) {
	var d Dialog
	host := newHost(t)
	empty := dump(d.fileControl(host, mustField(t, "caCert")))
	if strings.Contains(empty, "fieldB") || !strings.Contains(empty, "icon=folder") || !strings.Contains(empty, "Choose file") {
		t.Fatalf("an empty file setting should be one button with the folder icon and a label, no field:\n%s", empty)
	}
	host.current.CACert = "/etc/ssl/ca.pem"
	d = Dialog{}
	filled := dump(d.fileControl(host, mustField(t, "caCert")))
	if !strings.Contains(filled, "fieldB") || !strings.Contains(filled, "icon=folder") {
		t.Fatalf("a set file shows its path with a folder button:\n%s", filled)
	}
}

func mustField(t *testing.T, key string) settings.Field {
	t.Helper()
	field, ok := settings.FieldFor(key)
	if !ok {
		t.Fatalf("no setting %s", key)
	}
	return field
}

// The title bar and the navigation item templates render exactly what the Go
// views they replaced did.
func TestTitleBarTemplateMatchesTheGoView(t *testing.T) {
	for _, raw := range []bool{false, true} {
		d := Dialog{Raw: raw}
		host := newHost(t)
		goView := Row(H(DialogTitleH), PadX(16), Gap(8), AlignItems(AlignCenter), BorderB(w.BorderSoft),
			Text("Settings", Bold(), W(70)),
			If(!d.Raw, Field(&d.Search, H(28), Grow(1), Bg(w.Sunken), Placeholder("Search settings"), OnChange(d.ResetScroll), host.TextMenu(&d.Search, nil))),
			If(d.Raw, Spacer(Grow(1))),
			d.saveButton(host),
			d.modeSwitch(host),
			IconButton(CrossIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(host.Close), Tooltip("Close (Esc)")),
		)
		assertSameDump(t, d.titleBar(host), goView, 900, DialogTitleH)
	}
}

func TestNavItemTemplateMatchesTheGoView(t *testing.T) {
	var d Dialog
	p := pages[0]
	for _, open := range []bool{true, false} {
		items := []Item{H(w.RowH), PadX(12), AlignItems(AlignCenter), Control("button"), OnClick(func() { d.Cat, d.scroll.Y = p.id, 0 })}
		goView := Row(append(items, HoverBg(w.Hover), Text(p.label, Muted(), Grow(1)))...)
		if open {
			goView = Row(append(items, Bg(w.AccentSoft), Text(p.label, Bold(), Color(w.AccentLine), Grow(1)))...)
		}
		assertSameDump(t, d.navItem(p, open), goView, 170, w.RowH)
	}
}

func assertSameDump(t *testing.T, tplView, goView Node, width, height int) {
	t.Helper()
	ctx, err := w.NewCtx(w.Theme{Colors: map[w.Token]color.RGBA{w.Text: w.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	Layout(ctx, tplView, width, height)
	Layout(ctx, goView, width, height)
	if got, want := Dump(tplView), Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

func (h *fakeHost) FontStore() port.FontStore { return fonts.System{} }
