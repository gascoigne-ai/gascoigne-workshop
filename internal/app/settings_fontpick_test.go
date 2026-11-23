package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"strings"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
)

// testFamilies are real font files (the Go fonts) written to a folder
// and scanned: "Go" is proportional, "Go Mono" is monospace.
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

// monospace reports whether the interface (or code) face has equal widths.
func monoUI(app *App) bool {
	return app.ctx.Width(app.ctx.F.UI, "iiiiii") == app.ctx.Width(app.ctx.F.UI, "WWWWWW")
}

func monoCode(app *App) bool {
	return app.ctx.Width(app.ctx.F.Mono, "iiiiii") == app.ctx.Width(app.ctx.F.Mono, "WWWWWW")
}

func TestPickingAnInterfaceFontChangesTheFacesAndIsSaved(t *testing.T) {
	a := testApp(t)
	a.over.Settings.Form.Families = testFamilies(t)

	a.over.Settings.Form.SetFont(a.over.Settings, form.FontUI, "go mono")
	if !monoUI(a) || a.settings.UIFont != "Go Mono" {
		t.Fatalf("mono ui = %v, setting %q", monoUI(a), a.settings.UIFont)
	}
	if !monoCode(a) {
		t.Fatal("the code face changed with the interface font")
	}
	a.over.Settings.Form.SetFont(a.over.Settings, form.FontUI, "")
	if monoUI(a) || a.settings.UIFont != "" {
		t.Fatal("an empty name should restore the built-in font")
	}
}

func TestPickingACodeFontOnlyChangesTheCodeFace(t *testing.T) {
	a := testApp(t)
	a.over.Settings.Form.Families = testFamilies(t)
	a.over.Settings.Form.SetFont(a.over.Settings, form.FontCode, "Go")
	if monoCode(a) || a.settings.CodeFont != "Go" {
		t.Fatalf("code face mono = %v, setting %q", monoCode(a), a.settings.CodeFont)
	}
	if monoUI(a) {
		t.Fatal("the interface font changed with the code font")
	}
}

func TestSavedFontsApplyOnceTheScanFinishes(t *testing.T) {
	a := testApp(t)
	a.settings.UIFont, a.settings.CodeFont = "Go Mono", "Go"
	a.over.Settings.Form.FontsScanned(a.over.Settings, testFamilies(t))
	if !monoUI(a) || monoCode(a) {
		t.Fatalf("mono ui %v mono code %v after the scan", monoUI(a), monoCode(a))
	}
}

func TestASavedFontThatIsGoneFallsBackToTheBuiltInOne(t *testing.T) {
	a := testApp(t)
	a.settings.UIFont = "Removed Font"
	a.over.Settings.Form.FontsScanned(a.over.Settings, testFamilies(t))
	if monoUI(a) {
		t.Fatal("the interface font is not the built-in one")
	}
	if a.settings.UIFont != "Removed Font" {
		t.Fatal("the saved name was dropped; the font may come back")
	}
}

func TestTheAppearancePageHasFontControlsShowingTheCurrentFamily(t *testing.T) {
	a := testApp(t)
	a.over.Settings.Form.Families = testFamilies(t)
	a.settings.UIFont = "Go Mono"
	dump := Dump(a.over.Settings.Form.View(a.ctx, a.over.Settings, "Appearance", ""))
	for _, want := range []string{"Interface font", "Code font", `text="Go Mono"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("appearance page lacks %q", want)
		}
	}
}

// A font list left open when the settings dialog closed came back, stale and
// unclickable, when it was opened again. The list belongs to the dialog's
// open state; Escape closes the list first, then the dialog.
func TestEscapeClosesTheFontListBeforeTheDialog(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.over.Settings.Form.Families = testFamilies(t)
	a.over.Settings.Form.OpenFontList(a.ctx, a.over.Settings, form.FontUI)
	a.Step(core.Input{X: 5000, Y: 5000, Escape: true})
	if a.over.Settings.Form.FontListOpen() || !a.over.Settings.Open() {
		t.Fatalf("after the first Escape: list open %v, dialog open %v; want the list closed and the dialog open", a.over.Settings.Form.FontListOpen(), a.over.Settings.Open())
	}
	a.Step(core.Input{X: 5000, Y: 5000, Escape: true})
	if a.over.Settings.Open() {
		t.Fatal("the second Escape should close the dialog")
	}
}

func TestReopeningTheSettingsStartsWithTheFontListClosedAndWorking(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.over.Settings.Form.Families = testFamilies(t)
	a.over.Settings.Form.OpenFontList(a.ctx, a.over.Settings, form.FontUI)
	a.over.Settings.Close() // the dialog's own close button
	if a.over.Settings.Form.FontListOpen() {
		t.Fatal("the list stayed open behind a closed dialog")
	}
	a.over.Settings.Show()
	a.Step(core.Input{X: 5000, Y: 5000})
	if a.over.Settings.Form.FontListOpen() {
		t.Fatal("the reopened dialog shows the old list")
	}
	a.over.Settings.Form.OpenFontList(a.ctx, a.over.Settings, form.FontUI)
	if !a.over.Settings.Form.FontListOpen() {
		t.Fatal("the list cannot be opened again")
	}
}

// The list is a modal overlay: a click outside it only closes it, and the
// next click reaches the dialog.
func TestAClickOutsideTheFontListClosesOnlyTheList(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.over.Settings.Form.OpenFontList(a.ctx, a.over.Settings, form.FontUI)
	click(a, menuBtnX, menuBtnY)
	if !a.over.Settings.Open() || a.over.Settings.Form.FontListOpen() {
		t.Fatalf("dialog open %v, list open %v; want the list closed and the dialog open", a.over.Settings.Open(), a.over.Settings.Form.FontListOpen())
	}
	click(a, menuBtnX, menuBtnY)
	if a.over.Settings.Open() {
		t.Fatal("the next click on the button should close the dialog")
	}
}
