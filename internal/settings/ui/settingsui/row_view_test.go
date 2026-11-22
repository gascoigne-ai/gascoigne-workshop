package settingsui

import (
	"path/filepath"
	"strings"
	"testing"

	core "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The widths a row is laid out at: the dialog's page at its default and at
// its smallest size.
var rowWidths = []int{560, 320}

// rowPage is the form page of category laid out at width.
func rowPage(t *testing.T, driver *formDriver, width int) *El {
	t.Helper()
	root := driver.dialog.FormPage(driver.ctx, driver.host, driver.category)
	Layout(driver.ctx, root, width, 2400)
	return root
}

// titleOf is the title text of the setting named title.
func titleOf(t *testing.T, root *El, title string) *El {
	t.Helper()
	found := findAll(root, `text="`+title+`"`, "bold=true")
	if len(found) != 1 {
		t.Fatalf("%d titles %q:\n%s", len(found), title, Dump(root))
	}
	return found[0]
}

// firstOf is the first element matching parts.
func firstOf(t *testing.T, root *El, parts ...string) *El {
	t.Helper()
	found := findAll(root, parts...)
	if len(found) == 0 {
		t.Fatalf("nothing matches %q:\n%s", parts, Dump(root))
	}
	return found[0]
}

func within(inner, outer core.Rect) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y && inner.X+inner.W <= outer.X+outer.W && inner.Y+inner.H <= outer.Y+outer.H
}

// A switch setting has its title on the left and the switch on the right of
// the same row, centered on the title block; nothing overlaps or is clipped
// at either width.
func TestTheSwitchOfABooleanSettingSitsRightOfItsTitle(t *testing.T) {
	for _, width := range rowWidths {
		driver := newFormDriver(t, "General")
		root := rowPage(t, driver, width)
		title := titleOf(t, root, "Save changes automatically").Rect()
		switches := findAll(root, "switchB")
		if len(switches) != 1 {
			t.Fatalf("width %d: %d switches:\n%s", width, len(switches), Dump(root))
		}
		toggle := switches[0].Rect()
		if toggle.X < title.X+title.W {
			t.Errorf("width %d: switch at x=%d overlaps the title ending at %d", width, toggle.X, title.X+title.W)
		}
		if toggle.X+toggle.W > width {
			t.Errorf("width %d: switch ends at %d, clipped", width, toggle.X+toggle.W)
		}
		paragraph := firstOf(t, root, "paragraphB").Rect()
		if toggle.X < paragraph.X+paragraph.W {
			t.Errorf("width %d: switch at x=%d overlaps the description ending at %d", width, toggle.X, paragraph.X+paragraph.W)
		}
		block := core.Rect{X: 0, Y: title.Y, W: width, H: paragraph.Y + paragraph.H - title.Y}
		if mid := toggle.Y + toggle.H/2; mid < block.Y+block.H/2-core.RowH/2 || mid > block.Y+block.H/2+core.RowH/2 {
			t.Errorf("width %d: switch centered at y=%d, the title block spans %d..%d", width, mid, block.Y, block.Y+block.H)
		}
	}
}

// A long description wraps under the title in the room the switch leaves.
func TestALongDescriptionWrapsUnderTheTitle(t *testing.T) {
	driver := newFormDriver(t, "General")
	wide := firstOf(t, rowPage(t, driver, 560), "paragraphB").Rect()
	narrowRoot := rowPage(t, driver, 320)
	narrow := firstOf(t, narrowRoot, "paragraphB").Rect()
	if narrow.H <= wide.H {
		t.Fatalf("description is %d high at 320 and %d at 560: it does not wrap", narrow.H, wide.H)
	}
	title := titleOf(t, narrowRoot, "Save changes automatically").Rect()
	if narrow.Y < title.Y+title.H || narrow.X != title.X {
		t.Fatalf("description at %d,%d is not under the title at %d,%d", narrow.X, narrow.Y, title.X, title.Y+title.H)
	}
}

func TestABooleanSettingIsASwitchAndNotChips(t *testing.T) {
	driver := newFormDriver(t, "General")
	page := driver.view()
	if !strings.Contains(page, "switchB") || strings.Contains(page, `text="On"`) || strings.Contains(page, `text="Off"`) {
		t.Fatalf("a boolean is not a switch:\n%s", page)
	}
}

// The switch runs the change of its own setting.
func TestClickingASwitchFlipsItsSetting(t *testing.T) {
	driver := newFormDriver(t, "General")
	driver.clickOn(0, "switchB")
	if driver.host.current.AutoSave {
		t.Fatal("autoSave is still on")
	}
	driver.clickOn(0, "switchB")
	if !driver.host.current.AutoSave {
		t.Fatal("autoSave is still off")
	}
}

func TestClickingTheSwitchOfAnOffSettingTurnsItOn(t *testing.T) {
	driver := newFormDriver(t, "Experimental")
	driver.host.current.GRPC = false
	driver.clickOn(0, "switchB")
	if !driver.host.current.GRPC {
		t.Fatal("grpc is still off")
	}
}

// Placement follows the kind of the control: a short choice, a switch and a
// stepper go to the right of the title, a wide control goes under it.
func TestThePlacementOfAControlFollowsItsKind(t *testing.T) {
	cases := []struct {
		name, page, title, control string
		right                      bool
	}{
		{"enum of three", "Appearance", "Response view", `text="Compact"`, true},
		{"enum of two", "Appearance", "Contrast", `text="Normal"`, true},
		{"stepped number", "Appearance", "Interface font size", `text="+"`, true},
		{"theme chips", "Appearance", "Theme", `text="Light"`, false},
		{"scale slider", "Appearance", "Interface scale", "sliderB", false},
		{"font dropdown", "Appearance", "Interface font", "fieldB", false},
		{"text field", "Requests", "Proxy", "fieldB", false},
		{"file field", "Requests", "CA certificates", `text="Choose file"`, false},
	}
	for _, tc := range cases {
		for _, width := range rowWidths {
			driver := newFormDriver(t, tc.page)
			root := rowPage(t, driver, width)
			title := titleOf(t, root, tc.title).Rect()
			found := findAll(root, tc.control)
			if len(found) == 0 {
				t.Fatalf("%s: no %s:\n%s", tc.name, tc.control, Dump(root))
			}
			var control core.Rect
			for _, candidate := range found { // the first one at or below the title
				if candidate.Rect().Y+candidate.Rect().H > title.Y {
					control = candidate.Rect()
					break
				}
			}
			switch {
			case tc.right && control.X < title.X+title.W:
				t.Errorf("%s at %d: control at x=%d is not right of the title ending at %d", tc.name, width, control.X, title.X+title.W)
			case !tc.right && control.Y < title.Y+title.H:
				t.Errorf("%s at %d: control at y=%d is not under the title ending at %d", tc.name, width, control.Y, title.Y+title.H)
			case control.X+control.W > width:
				t.Errorf("%s at %d: control ends at %d, clipped", tc.name, width, control.X+control.W)
			}
		}
	}
}

// Every row keeps all of its content inside the row.
func TestRowsDoNotClipTheirContent(t *testing.T) {
	for _, category := range []string{"Appearance", "Requests", "General", "Experimental"} {
		for _, width := range rowWidths {
			driver := newFormDriver(t, category)
			root := rowPage(t, driver, width)
			for _, row := range root.Children() {
				for _, kid := range row.Children() {
					if !within(kid.Rect(), row.Rect()) {
						t.Errorf("%s at %d: %v leaves its row %v", category, width, kid.Rect(), row.Rect())
					}
				}
			}
		}
	}
}

// The reset icon shows beside the title of a changed setting only, and puts
// the default back.
func TestTheResetIconShowsForAChangedSettingAndRestoresTheDefault(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	if strings.Contains(driver.view(), "icon=undo") {
		t.Fatal("a reset icon shows with every setting at its default")
	}
	driver.host.current.Corners = "round"
	root := rowPage(t, driver, 560)
	icon := findAll(root, "icon=undo")
	if len(icon) != 1 {
		t.Fatalf("%d reset icons, want 1:\n%s", len(icon), Dump(root))
	}
	title := titleOf(t, root, "Corners").Rect()
	if icon[0].Rect().X < title.X+title.W || icon[0].Rect().Y >= title.Y+title.H {
		t.Errorf("reset icon at %v is not beside the title at %v", icon[0].Rect(), title)
	}
	driver.clickOn(0, "icon=undo")
	if driver.host.current.Corners == "round" {
		t.Fatal("the reset icon did not restore the default")
	}
	if strings.Contains(driver.view(), "icon=undo") {
		t.Fatal("the reset icon is still shown")
	}
}

// A category heading with a thin rule shows in search results only, never on
// a single page (user decision: the page title already names it).
func TestACategoryHeadingShowsInSearchResultsOnly(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	if strings.Contains(driver.view(), `text="APPEARANCE"`) {
		t.Fatal("a single page shows a category heading")
	}
	driver.dialog.Search = NewTextField("file")
	root := rowPage(t, driver, 560)
	heading := findAll(root, `text="REQUESTS"`)
	if len(heading) != 1 {
		t.Fatalf("%d headings for Requests in search results:\n%s", len(heading), Dump(root))
	}
	if !strings.Contains(Dump(root), "borderB=border-soft") || !strings.Contains(firstLineAround(root, heading[0]), "borderB=border-soft") {
		t.Fatalf("the heading has no thin rule:\n%s", Dump(root))
	}
}

// firstLineAround is the properties of the parent of child in root.
func firstLineAround(root, child *El) string {
	for _, kid := range root.Children() {
		for _, grand := range kid.Children() {
			if grand == child {
				line, _, _ := strings.Cut(Dump(kid), "\n")
				return line
			}
		}
		if grand := firstLineAround(kid, child); grand != "" {
			return grand
		}
	}
	return ""
}

// A refused number is explained under its row; the setting keeps its value.
func TestARefusalNoteShowsUnderTheRow(t *testing.T) {
	driver := newFormDriver(t, "Requests")
	*driver.dialog.Input(driver.host, "timeoutSec") = NewTextField("0")
	driver.dialog.Edited(driver.host, "timeoutSec")
	root := rowPage(t, driver, 560)
	note := findAll(root, "Enter a whole number")
	if len(note) != 1 {
		t.Fatalf("%d notes:\n%s", len(note), Dump(root))
	}
	title := titleOf(t, root, "Request timeout").Rect()
	if note[0].Rect().Y < title.Y+title.H {
		t.Fatalf("the note at y=%d is not under the title row", note[0].Rect().Y)
	}
}

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
