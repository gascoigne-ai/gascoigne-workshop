package settingsui

import (
	"path/filepath"
	"strings"
	"testing"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The page title row names the page and the file; the file shows by its
// base name, the full path is a tooltip only.
func TestThePageTitleRowShowsThePageAndTheFileName(t *testing.T) {
	driver := newFrameDriver(t)
	driver.dialog.Cat = "appearance"
	dump := Dump(driver.root())
	path := driver.host.path
	if !strings.Contains(dump, `text="settings.json"`) {
		t.Fatalf("no file name in the title row:\n%s", dump)
	}
	if strings.Contains(dump, `text="`+path+`"`) || strings.Contains(dump, `text="`+filepath.Dir(path)) {
		t.Fatalf("the full path is shown as text:\n%s", dump)
	}
	if !strings.Contains(dump, `text="Edit in settings.json"`) {
		t.Fatalf("no Edit in settings.json button:\n%s", dump)
	}
}

// Every page gets the title row, tables included.
func TestEveryPageShowsTheTitleRowWithTheEditButton(t *testing.T) {
	for _, id := range []string{"appearance", "colors", "shortcuts", "requests", "general", "experimental", "system"} {
		driver := newFrameDriver(t)
		driver.dialog.Cat = id
		if !strings.Contains(Dump(driver.root()), `text="Edit in settings.json"`) {
			t.Errorf("page %s has no Edit in settings.json button", id)
		}
	}
}

func TestEditInSettingsJSONOpensTheRawViewAndBackReturns(t *testing.T) {
	driver := newFrameDriver(t)
	driver.clickOn(`text="Edit in settings.json"`)
	if !driver.dialog.Raw {
		t.Fatal("Edit in settings.json did not open the raw view")
	}
	if !strings.Contains(Dump(driver.root()), `text="Back to the form"`) {
		t.Fatal("the raw view has no Back to the form button")
	}
	driver.clickOn(`text="Back to the form"`)
	if driver.dialog.Raw {
		t.Fatal("Back to the form left the raw view open")
	}
}

// The search field is the first thing in the navigation column.
func TestTheSearchFieldSitsAboveThePageList(t *testing.T) {
	driver := newFrameDriver(t)
	root := driver.root()
	field := findAll(root, "fieldB")
	first := findAll(root, `text="Colors"`)
	if len(field) == 0 || len(first) == 0 {
		t.Fatalf("no search field or no first page:\n%s", Dump(root))
	}
	search, page := field[0].Rect(), first[0].Rect()
	if search.Y+search.H > page.Y {
		t.Errorf("search field %v is not above the first page %v", search, page)
	}
	if search.X+search.W > page.X+DialogNavW {
		t.Errorf("search field %v is not in the navigation column of the first page %v", search, page)
	}
}

// Opening a page from a form row shows it at its top with the search cleared.
func TestOpenPageShowsThePageAtItsTopAndClearsTheSearch(t *testing.T) {
	dialog := SettingsDialog{Cat: "appearance", Search: NewTextField("overrides")}
	dialog.scroll.Y = 300
	dialog.OpenPage("colors")
	if dialog.Cat != "colors" || dialog.Searching() || dialog.scroll.Y != 0 {
		t.Fatalf("page %q search %q scroll %d", dialog.Cat, dialog.Search.Text, dialog.scroll.Y)
	}
}

// The dialog asks its host for the form of a category, with the search text.
func TestThePageOfAFormCategoryIsTheHostsForm(t *testing.T) {
	driver := newFrameDriver(t)
	driver.dialog.Cat = "requests"
	driver.dialog.Search = NewTextField("proxy")
	if dump := Dump(driver.root()); !strings.Contains(dump, `text="form Requests proxy"`) {
		t.Fatalf("the form of Requests with the search text is not the page:\n%s", dump)
	}
}
