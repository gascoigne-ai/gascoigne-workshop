package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/settingsui"
	"strings"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/fonts"
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
func monoUI(a *App) bool {
	return a.ctx.Width(a.ctx.F.UI, "iiiiii") == a.ctx.Width(a.ctx.F.UI, "WWWWWW")
}

func monoCode(a *App) bool {
	return a.ctx.Width(a.ctx.F.Mono, "iiiiii") == a.ctx.Width(a.ctx.F.Mono, "WWWWWW")
}

func TestPickingAnInterfaceFontChangesTheFacesAndIsSaved(t *testing.T) {
	a, _ := newTestApp(t)
	a.settingsPath = filepath.Join(t.TempDir(), "settings.json")
	a.prefs.Families = testFamilies(t)

	a.prefs.SetFont(a, settingsui.FontUI, "go mono")
	if !monoUI(a) || a.settings.UIFont != "Go Mono" {
		t.Fatalf("mono ui = %v, setting %q", monoUI(a), a.settings.UIFont)
	}
	if !monoCode(a) {
		t.Fatal("the code face changed with the interface font")
	}
	a.prefs.SetFont(a, settingsui.FontUI, "")
	if monoUI(a) || a.settings.UIFont != "" {
		t.Fatal("an empty name should restore the built-in font")
	}
}

func TestPickingACodeFontOnlyChangesTheCodeFace(t *testing.T) {
	a, _ := newTestApp(t)
	a.prefs.Families = testFamilies(t)
	a.prefs.SetFont(a, settingsui.FontCode, "Go")
	if monoCode(a) || a.settings.CodeFont != "Go" {
		t.Fatalf("code face mono = %v, setting %q", monoCode(a), a.settings.CodeFont)
	}
	if monoUI(a) {
		t.Fatal("the interface font changed with the code font")
	}
}

func TestSavedFontsApplyOnceTheScanFinishes(t *testing.T) {
	a, _ := newTestApp(t)
	a.settings.UIFont, a.settings.CodeFont = "Go Mono", "Go"
	a.prefs.FontsScanned(a, testFamilies(t))
	if !monoUI(a) || monoCode(a) {
		t.Fatalf("mono ui %v mono code %v after the scan", monoUI(a), monoCode(a))
	}
}

func TestASavedFontThatIsGoneFallsBackToTheBuiltInOne(t *testing.T) {
	a, _ := newTestApp(t)
	a.settings.UIFont = "Removed Font"
	a.prefs.FontsScanned(a, testFamilies(t))
	if monoUI(a) {
		t.Fatal("the interface font is not the built-in one")
	}
	if a.settings.UIFont != "Removed Font" {
		t.Fatal("the saved name was dropped; the font may come back")
	}
}

func TestTheAppearancePageHasFontControlsShowingTheCurrentFamily(t *testing.T) {
	a, _ := newTestApp(t)
	a.prefs.Families = testFamilies(t)
	a.settings.UIFont = "Go Mono"
	dump := Dump(a.prefs.FormPage(a, "Appearance"))
	for _, want := range []string{"Interface font", "Code font", `text="Go Mono"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("appearance page lacks %q", want)
		}
	}
}
