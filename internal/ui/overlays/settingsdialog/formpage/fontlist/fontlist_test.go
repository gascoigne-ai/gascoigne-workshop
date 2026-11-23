package fontlist

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist/logic"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

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

func TestTheFontListOpensWithEveryFamilyEachInItsOwnFont(t *testing.T) {
	var d List
	host := newHost(t)
	d.Families = testFamilies(t)
	if d.View(testCtx(t), host) != nil {
		t.Fatal("the list is shown while closed")
	}
	d.Open(testCtx(t), host, form.FontUI)
	list := dump(d.View(testCtx(t), host))
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
	var d List
	host := newHost(t)
	d.Families = testFamilies(t)
	d.Open(testCtx(t), host, form.FontCode)
	d.Pick(host, "Go")
	if host.current.CodeFont != "Go" || d.IsOpen() {
		t.Fatalf("CodeFont %q, open %v", host.current.CodeFont, d.IsOpen())
	}
	d.Open(testCtx(t), host, form.FontCode)
	d.Pick(host, "") // Built-in
	if host.current.CodeFont != "" {
		t.Fatalf("CodeFont %q after choosing Built-in", host.current.CodeFont)
	}
}

func TestTheFontListSearchNarrowsTheFamilies(t *testing.T) {
	var d List
	host := newHost(t)
	d.Families = testFamilies(t)
	d.Open(testCtx(t), host, form.FontUI)
	d.Search = NewTextField("mono")
	list := dump(d.View(testCtx(t), host))
	if !strings.Contains(list, `text="Go Mono"`) || strings.Contains(list, `text="Go"`) {
		t.Fatalf("search results:\n%s", list)
	}
}

// A system can have hundreds of families; reading every font file to draw
// the list would stall it, so only the rows in view load their font.
func TestOnlyTheVisibleRowsLoadTheirFonts(t *testing.T) {
	var d List
	host := newHost(t)
	base := testFamilies(t)[0]
	for i := range 300 {
		d.Families = append(d.Families, fonts.Family{Name: fmt.Sprintf("Family %03d", i), Regular: fmt.Sprintf("%s#%d", base.Regular, i)})
	}
	d.Open(testCtx(t), host, form.FontUI)
	_ = d.View(testCtx(t), host)
	if got := d.FacesLoaded(); got > 14 {
		t.Fatalf("%d fonts loaded for the first screen of a 300-family list", got)
	}
}

func TestEscapingOrClickingOutsideClosesTheList(t *testing.T) {
	var d List
	host := newHost(t)
	d.Open(testCtx(t), host, form.FontUI)
	d.Close()
	if d.IsOpen() || d.View(testCtx(t), host) != nil {
		t.Fatal("the list stayed open")
	}
}

// Opening the list scrolls to the family in use, so it is on screen without
// the user hunting for it through hundreds of names.
func TestTheFontListOpensScrolledToTheSelectedFamily(t *testing.T) {
	var d List
	host := newHost(t)
	for i := range 300 {
		d.Families = append(d.Families, fonts.Family{Name: fmt.Sprintf("Family %03d", i), Regular: fmt.Sprintf("none#%d", i)})
	}
	host.current.UIFont = "Family 200"
	d.Open(testCtx(t), host, form.FontUI)
	list := dump(d.View(testCtx(t), host))
	if !strings.Contains(list, `text="Family 200"`) {
		t.Fatalf("the selected family is not in the rows built on open:\n%.600s", list)
	}
	if strings.Contains(list, `text="Family 000"`) {
		t.Fatal("the list opened at the top instead of at the selection")
	}
	// the selected row sits inside the visible part of the list
	row, rowH := logic.Selected(d.names(), "Family 200"), testCtx(t).Sizes.Row
	if y := row*rowH - d.scroll.Y; y < 0 || y+rowH > logic.ListH {
		t.Fatalf("row %d at y=%d is outside the %dpx viewport", row, y, logic.ListH)
	}
}

func TestTheFontListOpensAtTheTopWhenNothingIsSelected(t *testing.T) {
	var d List
	host := newHost(t)
	d.Families = testFamilies(t)
	d.Open(testCtx(t), host, form.FontCode) // Built-in is in use
	if d.scroll.Y != 0 {
		t.Fatalf("scroll = %d, want the top", d.scroll.Y)
	}
}
