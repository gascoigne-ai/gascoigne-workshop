package settingsui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/platform/fonts"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/adapter/settingsfile"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fakeHost keeps the settings in memory and records what the dialog asked.
type fakeHost struct {
	current   settings.Settings
	path      string
	toasts    []string
	menus     [][]prefs.MenuItem
	menuAt    [2]int // where the last menu opened
	unsaved   bool
	pickTitle string
	pickDone  func(path string)
	saved     int
	fontsSet  int                    // how many times the dialog applied the fonts
	store     settings.SettingsStore // replaces the real files when a test sets it
	changes   int                    // how many Change messages the dialog sent
}

func newHost(t *testing.T) *fakeHost {
	return &fakeHost{current: settings.Default(), path: filepath.Join(t.TempDir(), "settings.json")}
}

func (h *fakeHost) Size() (int, int)            { return 1280, 800 }
func (h *fakeHost) Settings() settings.Settings { return h.current }
func (h *fakeHost) Change(edit func(*settings.Settings)) {
	h.changes++
	edit(&h.current)
	_ = settingsDisk.Save(h.path, h.current)
}
func (h *fakeHost) SetScaleLive(percent int)            { h.current.UIScalePct = percent }
func (h *fakeHost) Save()                               { h.saved++ }
func (h *fakeHost) Replace(current settings.Settings)   { h.current = current }
func (h *fakeHost) SettingsPath() string                { return h.path }
func (h *fakeHost) SetFonts(ui, code w.FontData) error  { h.fontsSet++; return nil }
func (h *fakeHost) Toast(text string, _ severity.Level) { h.toasts = append(h.toasts, text) }
func (h *fakeHost) SetStatus(string)                    {}
func (h *fakeHost) OpenMenu(posX, posY int, items []prefs.MenuItem) {
	h.menus = append(h.menus, items)
	h.menuAt = [2]int{posX, posY}
}
func (h *fakeHost) ExtraPage(id string) Node { return Text("page " + id) }
func (h *fakeHost) ActionIDs() []string      { return []string{"send"} }
func (h *fakeHost) Close()                   {}
func (h *fakeHost) MarkDirty()               {}
func (h *fakeHost) Later(callback func())    { callback() }

func dump(node Node) string { return Dump(node) }

func TestAPageShowsItsSettingsAndTheSearchFiltersAcrossPages(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	if text := dump(d.FormPage(testCtx(t), host, "Requests")); !strings.Contains(text, "Proxy") || strings.Contains(text, "Corners") {
		t.Fatalf("Requests page:\n%s", text)
	}
	d.Search = NewTextField("corners")
	if text := dump(d.FormPage(testCtx(t), host, "Requests")); !strings.Contains(text, "Corners") || strings.Contains(text, "Proxy") {
		t.Fatalf("search results:\n%s", text)
	}
}

func TestANumberOutsideItsRangeIsExplainedAndNotApplied(t *testing.T) {
	var d SettingsDialog
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
	var d SettingsDialog
	host := newHost(t)
	host.Change(func(current *settings.Settings) { current.Layout = "side" })
	d.Reset(host, "layout")
	if data, _ := os.ReadFile(host.path); strings.Contains(string(data), "layout") || host.current.Layout != "stacked" {
		t.Fatalf("layout %q, file %q", host.current.Layout, data)
	}
}

func TestRawTextWithProblemsChangesNothingAndGoodTextIsWrittenAsTyped(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	d.OpenRaw(host)
	d.RawText.Text = `{"layout": "diagonal"}`
	d.RawEdited(host)
	if host.current.Layout != "stacked" || len(d.Problems) != 1 {
		t.Fatalf("layout %q problems %+v", host.current.Layout, d.Problems)
	}
	d.RawText.Text = "{\n  \"layout\": \"side\"\n}"
	d.RawEdited(host)
	if host.current.Layout != "side" || len(d.Problems) != 0 {
		t.Fatalf("layout %q problems %+v", host.current.Layout, d.Problems)
	}
	if data, _ := os.ReadFile(host.path); string(data) != d.RawText.Text {
		t.Fatalf("file %q", data)
	}
}

func TestAChangeOnDiskIsPickedUpOnceAndAnOwnWriteIsNot(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	host.Change(func(current *settings.Settings) { current.Corners = "round" })
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
	var d SettingsDialog
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
func (h *fakeHost) Pointer() (int, int) { return 400, 300 }
func (h *fakeHost) Unsaved() bool       { return h.unsaved }
func (h *fakeHost) Persist()            { _ = settingsDisk.Save(h.path, h.current); h.unsaved = false; h.saved++ }
func (h *fakeHost) SettingsStore() settings.SettingsStore {
	if h.store != nil {
		return h.store
	}
	return settingsfile.Files{}
}

func TestAFileSettingHasAFolderButtonInsideItsFieldThatFillsIt(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	page := dump(d.FormPage(testCtx(t), host, "Requests"))
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
	var d SettingsDialog
	host := newHost(t)
	if d.saveButton(testCtx(t), host) != nil {
		t.Fatal("with auto-save on and nothing pending there is nothing to save, so no button")
	}
	host.current.AutoSave = false
	if !strings.Contains(dump(d.saveButton(testCtx(t), host)), "onClick=true") {
		t.Fatalf("with auto-save off the Save button must work:\n%s", dump(d.saveButton(testCtx(t), host)))
	}
	host.unsaved = true
	host.current.AutoSave = true // changes pending even though the option is back on: Save stays available
	if !strings.Contains(dump(d.saveButton(testCtx(t), host)), "onClick=true") {
		t.Fatal("Save is hidden although changes are pending")
	}
}

func TestAnEmptyFileSettingIsOnlyAButtonAndAFilledOneShowsItsPath(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	empty := dump(d.fileControl(testCtx(t), host, mustField(t, "caCert")))
	if strings.Contains(empty, "fieldB") || !strings.Contains(empty, "icon=folder") || !strings.Contains(empty, "Choose file") {
		t.Fatalf("an empty file setting should be one button with the folder icon and a label, no field:\n%s", empty)
	}
	host.current.CACert = "/etc/ssl/ca.pem"
	d = SettingsDialog{}
	filled := dump(d.fileControl(testCtx(t), host, mustField(t, "caCert")))
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

// The row under the title bar holds Save only: it is not there when changes
// are written at once, and the search field is in the page list.
func TestToolbarHoldsSaveOnlyWhenThereIsSomethingToSave(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	if bar := d.toolbar(testCtx(t), host); bar != nil {
		t.Fatalf("auto-save is on but there is a toolbar:\n%s", Dump(bar))
	}
	host.current.AutoSave = false
	bar := d.toolbar(testCtx(t), host)
	if bar == nil {
		t.Fatal("auto-save is off but there is no toolbar")
	}
	Layout(testCtx(t), bar, 900, w.DialogToolbarH)
	dump := Dump(bar)
	if !strings.Contains(dump, `text="Save"`) || strings.Contains(dump, "fieldB") {
		t.Errorf("toolbar:\n%s", dump)
	}
	if title := Dump(d.titleBar(host)); !strings.Contains(title, "Settings") {
		t.Errorf("no title:\n%s", title)
	}
}

func TestNavItemTemplateMatchesTheGoView(t *testing.T) {
	var d SettingsDialog
	p := pages[0]
	for _, open := range []bool{true, false} {
		items := []Item{H(w.RowH), PadX(12), AlignItems(AlignCenter), Control("button"), OnClick(func() { d.Cat, d.scroll.Y = p.id, 0 })}
		goView := Row(append(items, HoverBg(w.Hover), Text(p.label, Muted(), Grow(1)))...)
		if open {
			goView = Row(append(items, Bg(w.AccentSoft), Text(p.label, Bold(), Color(w.AccentLine), Grow(1)))...)
		}
		assertSameDump(t, d.navItem(testCtx(t), p, open), goView, 170, w.RowH)
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

func (h *fakeHost) FontStore() settings.FontStore { return fonts.System{} }

// Send routes a message to the fake's own recorder of it.
func (h *fakeHost) Send(msg prefs.Msg) {
	switch msg := msg.(type) {
	case prefs.Change:
		h.Change(msg.Edit)
	case prefs.ScaleLive:
		h.SetScaleLive(msg.Percent)
	case prefs.Save:
		h.Save()
	case prefs.Replace:
		h.Replace(msg.Settings)
	case prefs.Persist:
		h.Persist()
	case prefs.FileMarked:
		h.unsaved = !msg.Saved
	case prefs.Toast:
		h.Toast(msg.Text, msg.Level)
	case prefs.SetStatus:
		h.SetStatus(msg.Text)
	case prefs.OpenMenu:
		h.OpenMenu(msg.X, msg.Y, msg.Items)
	case prefs.PickFile:
		h.PickFile(msg.Title, msg.Done)
	case prefs.Close:
		h.Close()
	case prefs.MarkDirty:
		h.MarkDirty()
	case prefs.Later:
		h.Later(msg.Func)
	}
}

// The Experimental category was in the schema but not in the dialog's page
// list, so its settings could not be reached from the form.
func TestEveryVisibleCategoryHasAPageInTheDialog(t *testing.T) {
	shown := map[string]bool{}
	for _, entry := range pages {
		shown[entry.category] = true
	}
	for _, category := range settings.Categories {
		if category == "Colors" || category == "Shortcuts" {
			continue // host pages, not forms
		}
		if !shown[category] {
			t.Errorf("category %q has no page", category)
		}
	}
}

// settingsDisk is the settings store the dialog tests write files through.
var settingsDisk settingsfile.Files
