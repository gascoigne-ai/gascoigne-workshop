package ui

import (
	"strings"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
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
	a.families = testFamilies(t)

	a.setFont(fontUI, "go mono")
	if !monoUI(a) || a.settings.UIFont != "Go Mono" {
		t.Fatalf("mono ui = %v, setting %q", monoUI(a), a.settings.UIFont)
	}
	if !monoCode(a) {
		t.Fatal("the code face changed with the interface font")
	}
	a.setFont(fontUI, "")
	if monoUI(a) || a.settings.UIFont != "" {
		t.Fatal("an empty name should restore the built-in font")
	}
}

func TestPickingACodeFontOnlyChangesTheCodeFace(t *testing.T) {
	a, _ := newTestApp(t)
	a.families = testFamilies(t)
	a.setFont(fontCode, "Go")
	if monoCode(a) || a.settings.CodeFont != "Go" {
		t.Fatalf("code face mono = %v, setting %q", monoCode(a), a.settings.CodeFont)
	}
	if monoUI(a) {
		t.Fatal("the interface font changed with the code font")
	}
}

func TestAnUnknownFontNameIsReportedAndChangesNothing(t *testing.T) {
	a, _ := newTestApp(t)
	a.families = testFamilies(t)
	a.setFont(fontUI, "Comic Nope")
	if a.settings.UIFont != "" || a.fontNotes[fontUI] == "" {
		t.Fatalf("setting %q, note %q", a.settings.UIFont, a.fontNotes[fontUI])
	}
}

func TestSavedFontsApplyOnceTheScanFinishes(t *testing.T) {
	a, _ := newTestApp(t)
	a.settings.UIFont, a.settings.CodeFont = "Go Mono", "Go"
	a.fontsScanned(testFamilies(t))
	if !monoUI(a) || monoCode(a) {
		t.Fatalf("mono ui %v mono code %v after the scan", monoUI(a), monoCode(a))
	}
}

func TestASavedFontThatIsGoneFallsBackToTheBuiltInOne(t *testing.T) {
	a, _ := newTestApp(t)
	a.settings.UIFont = "Removed Font"
	a.fontsScanned(testFamilies(t))
	if monoUI(a) {
		t.Fatal("the interface font is not the built-in one")
	}
	if a.settings.UIFont != "Removed Font" {
		t.Fatal("the saved name was dropped; the font may come back")
	}
}

func TestTheAppearancePageHasFontFieldsWithSuggestions(t *testing.T) {
	a, _ := newTestApp(t)
	a.families = testFamilies(t)
	a.uiFont = NewTextField("mono")
	dump := Dump(a.formPage("Appearance"))
	for _, want := range []string{"Interface font", "Code font", `text="Go Mono"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("appearance page lacks %q", want)
		}
	}
}

// The suggestions are chips of the width of their names; a long list of
// them ran out of the settings pane, so they appear only once the user has
// typed something to narrow them, and there are few.
func TestFontSuggestionsNeedTypedTextAndAreFew(t *testing.T) {
	a, _ := newTestApp(t)
	a.families = testFamilies(t)
	a.uiFont = NewTextField("")
	if dump := Dump(a.fontPicker(fontUI, &a.uiFont)); strings.Contains(dump, `text="Go Mono"`) || strings.Contains(dump, `text="Go"`) {
		t.Fatalf("suggestions shown for an empty field:\n%s", dump)
	}
	if fontSuggestions > 3 {
		t.Fatalf("fontSuggestions = %d, want at most 3 so the chips fit the pane", fontSuggestions)
	}
}
