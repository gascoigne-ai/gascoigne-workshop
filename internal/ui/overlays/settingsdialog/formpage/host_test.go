package formpage

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fakeHost keeps the settings in memory and records what the form asked;
// written is the settings file the form would have saved.
type fakeHost struct {
	current   settings.Settings
	written   []byte
	toasts    []string
	pickTitle string
	pickDone  func(path string)
	opened    string // the page the form asked the dialog to show
	saved     int
	fontsSet  int // how many times the form applied the fonts
	changes   int // how many Change messages the form sent
}

func newHost(t *testing.T) *fakeHost {
	return &fakeHost{current: settings.Default()}
}

func (h *fakeHost) Size() (int, int)                   { return 1280, 800 }
func (h *fakeHost) Settings() settings.Settings        { return h.current }
func (h *fakeHost) Pointer() (int, int)                { return 400, 300 }
func (h *fakeHost) OpenPage(id string)                 { h.opened = id }
func (h *fakeHost) FontStore() fontlist.FontStore      { return fonts.System{} }
func (h *fakeHost) SetFonts(ui, code w.FontData) error { h.fontsSet++; return nil }

// Send routes a message to the fake's own recorder of it.
func (h *fakeHost) Send(msg prefs.Msg) {
	switch msg := msg.(type) {
	case prefs.Change:
		h.changes++
		msg.Edit(&h.current)
		h.written, _ = settings.Encode(h.current)
	case prefs.ScaleLive:
		h.current.UIScalePct = msg.Percent
	case prefs.Save:
		h.saved++
	case prefs.Toast:
		h.toasts = append(h.toasts, msg.Text)
	case prefs.PickFile:
		h.pickTitle, h.pickDone = msg.Title, msg.Done
	case prefs.Later:
		msg.Func()
	}
}

// testFamilies are real font files (the Go fonts) written to a folder and
// scanned: "Go" and "Go Mono".
func testFamilies(t *testing.T) []fonts.Family {
	t.Helper()
	dir := t.TempDir()
	for name, data := range map[string][]byte{"r.ttf": goregular.TTF, "b.ttf": gobold.TTF, "m.ttf": gomono.TTF, "mb.ttf": gomonobold.TTF} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return fonts.Scan(dir)
}

func dump(node Node) string { return Dump(node) }

// testTheme is the theme the views are built with in tests.
func testTheme() core.Theme {
	return core.Theme{
		Colors:   map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff), core.Bg: core.RGB(0x101010)},
		FontSize: 14, SmallSize: 12, MonoSize: 13,
	}
}

// testCtx is the context the views are built with in tests: the default
// faces, so the sizes are the default ones.
func testCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func mustField(t *testing.T, key string) settings.Field {
	t.Helper()
	field, ok := settings.FieldFor(key)
	if !ok {
		t.Fatalf("no setting %s", key)
	}
	return field
}

func TestAPageShowsItsSettingsAndTheSearchFiltersAcrossPages(t *testing.T) {
	var d Page
	host := newHost(t)
	if text := dump(d.View(testCtx(t), host, "Requests", "")); !strings.Contains(text, "Proxy") || strings.Contains(text, "Corners") {
		t.Fatalf("Requests page:\n%s", text)
	}
	if text := dump(d.View(testCtx(t), host, "Requests", "corners")); !strings.Contains(text, "Corners") || strings.Contains(text, "Proxy") {
		t.Fatalf("search results:\n%s", text)
	}
}

func TestANumberOutsideItsRangeIsExplainedAndNotApplied(t *testing.T) {
	var d Page
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

func TestANumberOutsideItsRangeSendsNoChange(t *testing.T) {
	var d Page
	host := newHost(t)
	*d.Input(host, "timeoutSec") = NewTextField("0")
	d.Edited(host, "timeoutSec")
	if host.changes != 0 || d.Note("timeoutSec") == "" {
		t.Fatalf("changes %d note %q", host.changes, d.Note("timeoutSec"))
	}
	*d.Input(host, "timeoutSec") = NewTextField("20")
	d.Edited(host, "timeoutSec")
	if host.changes != 1 || host.current.TimeoutSec != 20 {
		t.Fatalf("changes %d timeout %d", host.changes, host.current.TimeoutSec)
	}
}

func TestResetRemovesTheKeyFromTheFile(t *testing.T) {
	var d Page
	host := newHost(t)
	host.Send(prefs.Change{Edit: func(current *settings.Settings) { current.Layout = "side" }})
	d.Reset(host, "layout")
	if strings.Contains(string(host.written), "layout") || host.current.Layout != "stacked" {
		t.Fatalf("layout %q, file %q", host.current.Layout, host.written)
	}
}

func TestAFileSettingHasAFolderButtonInsideItsFieldThatFillsIt(t *testing.T) {
	var d Page
	host := newHost(t)
	page := dump(d.View(testCtx(t), host, "Requests", ""))
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

func TestAnEmptyFileSettingIsOnlyAButtonAndAFilledOneShowsItsPath(t *testing.T) {
	var d Page
	host := newHost(t)
	empty := dump(d.fileControl(testCtx(t), host, mustField(t, "caCert")))
	if strings.Contains(empty, "fieldB") || !strings.Contains(empty, "icon=folder") || !strings.Contains(empty, "Choose file") {
		t.Fatalf("an empty file setting should be one button with the folder icon and a label, no field:\n%s", empty)
	}
	host.current.CACert = "/etc/ssl/ca.pem"
	d = Page{}
	filled := dump(d.fileControl(testCtx(t), host, mustField(t, "caCert")))
	if !strings.Contains(filled, "fieldB") || !strings.Contains(filled, "icon=folder") {
		t.Fatalf("a set file shows its path with a folder button:\n%s", filled)
	}
}

// dialogPageW is the width of the page in the dialog at its default size: its
// width of 760 less the list of pages of 170 (package settingsdialog).
const dialogPageW = 760 - 170

// findAll is every element under root, in tree order, whose own line of
// Dump has all the parts.
func findAll(root *El, parts ...string) []*El {
	var found []*El
	line, _, _ := strings.Cut(Dump(root), "\n")
	matches := true
	for _, part := range parts {
		matches = matches && strings.Contains(line, part)
	}
	if matches {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findAll(child, parts...)...)
	}
	return found
}
