package settingsui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fontfamily"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// testFamilies are real font files (the Go fonts) written to a folder and
// scanned: "Go" and "Go Mono".
func testFamilies(t *testing.T) []fontfamily.Family {
	t.Helper()
	dir := t.TempDir()
	for name, data := range map[string][]byte{"r.ttf": goregular.TTF, "b.ttf": gobold.TTF, "m.ttf": gomono.TTF, "mb.ttf": gomonobold.TTF} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return fonts.Scan(dir)
}

func TestTheFontControlShowsTheCurrentFamilyInItsOwnFont(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	d.Families = testFamilies(t)
	host.current.UIFont = "Go Mono"
	page := dump(d.FormPage(host, "Appearance"))
	if !strings.Contains(page, `text="Go Mono"`) || !strings.Contains(page, "font=custom") {
		t.Fatalf("the interface font control does not show its family in that font:\n%s", page)
	}
}

func TestTheFontListOpensWithEveryFamilyEachInItsOwnFont(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	d.Families = testFamilies(t)
	if d.FontList(host) != nil {
		t.Fatal("the list is shown while closed")
	}
	d.OpenFontList(host, FontUI)
	list := dump(d.FontList(host))
	for _, want := range []string{`text="Built-in"`, `text="Go"`, `text="Go Mono"`} {
		if !strings.Contains(list, want) {
			t.Errorf("the list lacks %s:\n%s", want, list)
		}
	}
	if strings.Count(list, "font=custom") != 2 {
		t.Errorf("the two families are not drawn in their own fonts:\n%s", list)
	}
}

func TestPickingAFamilyAppliesItAndClosesTheList(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	d.Families = testFamilies(t)
	d.OpenFontList(host, FontCode)
	d.PickFont(host, "Go")
	if host.current.CodeFont != "Go" || d.FontListOpen() {
		t.Fatalf("CodeFont %q, open %v", host.current.CodeFont, d.FontListOpen())
	}
	d.OpenFontList(host, FontCode)
	d.PickFont(host, "") // Built-in
	if host.current.CodeFont != "" {
		t.Fatalf("CodeFont %q after choosing Built-in", host.current.CodeFont)
	}
}

func TestTheFontListSearchNarrowsTheFamilies(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	d.Families = testFamilies(t)
	d.OpenFontList(host, FontUI)
	d.FontSearch = NewTextField("mono")
	list := dump(d.FontList(host))
	if !strings.Contains(list, `text="Go Mono"`) || strings.Contains(list, `text="Go"`) {
		t.Fatalf("search results:\n%s", list)
	}
}

// A system can have hundreds of families; reading every font file to draw
// the list would stall it, so only the rows in view load their font.
func TestOnlyTheVisibleRowsLoadTheirFonts(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	base := testFamilies(t)[0]
	for i := range 300 {
		d.Families = append(d.Families, fontfamily.Family{Name: fmt.Sprintf("Family %03d", i), Regular: fmt.Sprintf("%s#%d", base.Regular, i)})
	}
	d.OpenFontList(host, FontUI)
	_ = d.FontList(host)
	if got := d.loadedFaces(); got > 14 {
		t.Fatalf("%d fonts loaded for the first screen of a 300-family list", got)
	}
}

func TestEscapingOrClickingOutsideClosesTheList(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	d.OpenFontList(host, FontUI)
	d.CloseFontList()
	if d.FontListOpen() || d.FontList(host) != nil {
		t.Fatal("the list stayed open")
	}
}

// Opening the list scrolls to the family in use, so it is on screen without
// the user hunting for it through hundreds of names.
func TestTheFontListOpensScrolledToTheSelectedFamily(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	for i := range 300 {
		d.Families = append(d.Families, fontfamily.Family{Name: fmt.Sprintf("Family %03d", i), Regular: fmt.Sprintf("none#%d", i)})
	}
	host.current.UIFont = "Family 200"
	d.OpenFontList(host, FontUI)
	list := dump(d.FontList(host))
	if !strings.Contains(list, `text="Family 200"`) {
		t.Fatalf("the selected family is not in the rows built on open:\n%.600s", list)
	}
	if strings.Contains(list, `text="Family 000"`) {
		t.Fatal("the list opened at the top instead of at the selection")
	}
	// the selected row sits inside the visible part of the list
	row := d.selectedRow(host)
	if y := row*fontListRowH - d.list.scroll.Y; y < 0 || y+fontListRowH > fontListH {
		t.Fatalf("row %d at y=%d is outside the %dpx viewport", row, y, fontListH)
	}
}

func TestTheFontListOpensAtTheTopWhenNothingIsSelected(t *testing.T) {
	var d SettingsDialog
	host := newHost(t)
	d.Families = testFamilies(t)
	d.OpenFontList(host, FontCode) // Built-in is in use
	if d.list.scroll.Y != 0 {
		t.Fatalf("scroll = %d, want the top", d.list.scroll.Y)
	}
}
