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
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	a.form.Families = testFamilies(t)

	a.form.SetFont(a, form.FontUI, "go mono")
	if !monoUI(a) || a.settings.UIFont != "Go Mono" {
		t.Fatalf("mono ui = %v, setting %q", monoUI(a), a.settings.UIFont)
	}
	if !monoCode(a) {
		t.Fatal("the code face changed with the interface font")
	}
	a.form.SetFont(a, form.FontUI, "")
	if monoUI(a) || a.settings.UIFont != "" {
		t.Fatal("an empty name should restore the built-in font")
	}
}

func TestPickingACodeFontOnlyChangesTheCodeFace(t *testing.T) {
	a := testApp(t)
	a.form.Families = testFamilies(t)
	a.form.SetFont(a, form.FontCode, "Go")
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
	a.form.FontsScanned(a, testFamilies(t))
	if !monoUI(a) || monoCode(a) {
		t.Fatalf("mono ui %v mono code %v after the scan", monoUI(a), monoCode(a))
	}
}

func TestASavedFontThatIsGoneFallsBackToTheBuiltInOne(t *testing.T) {
	a := testApp(t)
	a.settings.UIFont = "Removed Font"
	a.form.FontsScanned(a, testFamilies(t))
	if monoUI(a) {
		t.Fatal("the interface font is not the built-in one")
	}
	if a.settings.UIFont != "Removed Font" {
		t.Fatal("the saved name was dropped; the font may come back")
	}
}

func TestTheAppearancePageHasFontControlsShowingTheCurrentFamily(t *testing.T) {
	a := testApp(t)
	a.form.Families = testFamilies(t)
	a.settings.UIFont = "Go Mono"
	dump := Dump(a.form.View(a.ctx, a, "Appearance", ""))
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
	a.form.Families = testFamilies(t)
	a.form.OpenFontList(a.ctx, a, form.FontUI)
	a.step(core.Input{X: 5000, Y: 5000, Escape: true})
	if a.form.FontListOpen() || !a.menu {
		t.Fatalf("after the first Escape: list open %v, dialog open %v; want the list closed and the dialog open", a.form.FontListOpen(), a.menu)
	}
	a.step(core.Input{X: 5000, Y: 5000, Escape: true})
	if a.menu {
		t.Fatal("the second Escape should close the dialog")
	}
}

func TestReopeningTheSettingsStartsWithTheFontListClosedAndWorking(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.form.Families = testFamilies(t)
	a.form.OpenFontList(a.ctx, a, form.FontUI)
	a.closeSettings() // the dialog's own close button
	if a.form.FontListOpen() {
		t.Fatal("the list stayed open behind a closed dialog")
	}
	a.menu = true
	a.step(core.Input{X: 5000, Y: 5000})
	if a.form.FontListOpen() {
		t.Fatal("the reopened dialog shows the old list")
	}
	a.form.OpenFontList(a.ctx, a, form.FontUI)
	if !a.form.FontListOpen() {
		t.Fatal("the list cannot be opened again")
	}
}

// The list is a modal overlay: a click outside it only closes it, and the
// next click reaches the dialog.
func TestAClickOutsideTheFontListClosesOnlyTheList(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.form.OpenFontList(a.ctx, a, form.FontUI)
	click(a, menuBtnX, menuBtnY)
	if !a.menu || a.form.FontListOpen() {
		t.Fatalf("dialog open %v, list open %v; want the list closed and the dialog open", a.menu, a.form.FontListOpen())
	}
	click(a, menuBtnX, menuBtnY)
	if a.menu {
		t.Fatal("the next click on the button should close the dialog")
	}
}
