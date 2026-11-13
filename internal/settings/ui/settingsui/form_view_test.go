package settingsui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	core "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Sizes the form is pinned at: about the width of the dialog's page, and a
// narrow one where descriptions and chips are squeezed.
const (
	formWideW, formWideH     = 560, 1400
	formNarrowW, formNarrowH = 300, 1800
)

// formCase is one state of the form: the page, what is set before it is
// drawn and the search text.
type formCase struct {
	name          string
	width, height int
	category      string
	search        string
	prepare       func(t *testing.T, dialog *SettingsDialog, host *fakeHost)
}

// changeAppearance moves several Appearance settings off their defaults:
// the interface font is installed, the code font is not.
func changeAppearance(t *testing.T, dialog *SettingsDialog, host *fakeHost) {
	host.current.Theme = "light"
	host.current.UIFont = "Go Mono"
	host.current.CodeFont = "Missing Sans"
	host.current.UIScalePct = 150
	host.current.Corners = "round"
	host.current.Flat = true
	host.current.ResponseView = "raw"
}

// editRequests fills a certificate and refuses a typed timeout.
func editRequests(t *testing.T, dialog *SettingsDialog, host *fakeHost) {
	host.current.CACert = "/etc/ssl/ca.pem"
	*dialog.Input(host, "timeoutSec") = NewTextField("0")
	dialog.Edited(host, "timeoutSec")
}

// The laid-out form is pinned as text (testdata/form-*.dump, captured from
// the Go-built view), so a change in how it is built cannot move a pixel
// unnoticed.
func TestTheSettingsFormLaysOutAsPinned(t *testing.T) {
	cases := []formCase{
		{"appearance-default-wide", formWideW, formWideH, "Appearance", "", nil},
		{"appearance-default-narrow", formNarrowW, formNarrowH, "Appearance", "", nil},
		{"appearance-changed-wide", formWideW, formWideH, "Appearance", "", changeAppearance},
		{"appearance-changed-narrow", formNarrowW, formNarrowH, "Appearance", "", changeAppearance},
		{"requests-default-wide", formWideW, formWideH, "Requests", "", nil},
		{"requests-edited-narrow", formNarrowW, formNarrowH, "Requests", "", editRequests},
		{"general-default-wide", formWideW, formWideH, "General", "", nil},
		{"general-changed-narrow", formNarrowW, formNarrowH, "General", "", func(_ *testing.T, _ *SettingsDialog, host *fakeHost) { host.current.AutoSave = false }},
		{"search-overrides-wide", formWideW, formWideH, "Appearance", "overrides", nil},
		{"search-across-narrow", formNarrowW, formNarrowH, "Appearance", "file", nil},
		{"search-none-wide", formWideW, formWideH, "Appearance", "nothing like it", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var dialog SettingsDialog
			host := newHost(t)
			dialog.Families = testFamilies(t)
			dialog.Search = NewTextField(tc.search)
			if tc.prepare != nil {
				tc.prepare(t, &dialog, host)
			}
			root := dialog.FormPage(host, tc.category)
			ctx, err := core.NewCtx(testTheme())
			if err != nil {
				t.Fatal(err)
			}
			Layout(ctx, root, tc.width, tc.height)
			want, err := os.ReadFile(filepath.Join("testdata", "form-"+tc.name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got := Dump(root); got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// formDriver runs a page of the form frame by frame as the application
// does: rebuild the view, lay it out, deliver the input.
type formDriver struct {
	t        *testing.T
	ctx      *core.Ctx
	dialog   SettingsDialog
	host     *fakeHost
	category string
}

func newFormDriver(t *testing.T, category string) *formDriver {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	driver := &formDriver{t: t, ctx: ctx, host: newHost(t), category: category}
	driver.dialog.Families = testFamilies(t)
	return driver
}

// step runs one frame; input is built from the frame's laid-out view.
func (driver *formDriver) step(input func(root *El) core.Input) {
	root := driver.dialog.FormPage(driver.host, driver.category)
	Layout(driver.ctx, root, formWideW, formWideH)
	driver.ctx.In = input(root)
	Update(driver.ctx, root)
}

// find is the index-th element matching parts in this frame's view.
func (driver *formDriver) find(root *El, index int, parts ...string) *El {
	driver.t.Helper()
	found := findAll(root, parts...)
	if len(found) <= index {
		driver.t.Fatalf("%d elements match %q, want index %d:\n%s", len(found), parts, index, Dump(root))
	}
	return found[index]
}

// clickOn clicks the middle of the index-th element matching parts.
func (driver *formDriver) clickOn(index int, parts ...string) {
	driver.t.Helper()
	driver.step(func(root *El) core.Input {
		rect := driver.find(root, index, parts...).Rect()
		return core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	})
}

// pressAt presses on the first element matching parts at a fraction of
// its width and holds the button down.
func (driver *formDriver) pressAt(fraction float64, parts ...string) {
	driver.t.Helper()
	driver.step(func(root *El) core.Input {
		rect := driver.find(root, 0, parts...).Rect()
		return core.Input{X: rect.X + int(fraction*float64(rect.W)), Y: rect.Y + rect.H/2, Click: true, Held: true}
	})
}

// release lets the mouse button go away from everything.
func (driver *formDriver) release() {
	driver.step(func(*El) core.Input { return core.Input{X: 5000, Y: 5000} })
}

// retype replaces the text of the focused field the way a user does: go
// to the end, backspace over it, then type.
func (driver *formDriver) retype(text string) {
	edits := []core.Edit{core.EditEnd}
	for range 32 {
		edits = append(edits, core.EditBackspace)
	}
	driver.step(func(*El) core.Input { return core.Input{X: 5000, Y: 5000, Edits: edits} })
	driver.step(func(*El) core.Input { return core.Input{X: 5000, Y: 5000, Typed: []rune(text)} })
}

// view is the page as the next frame draws it, laid out.
func (driver *formDriver) view() string {
	root := driver.dialog.FormPage(driver.host, driver.category)
	Layout(driver.ctx, root, formWideW, formWideH)
	return Dump(root)
}

func TestClickingAChipAppliesItsValue(t *testing.T) {
	cases := []struct {
		name  string
		label string
		index int // which chip of that label, in page order
		got   func(current settings.Settings) any
		want  any
	}{
		{"theme", `text="Light"`, 0, func(current settings.Settings) any { return current.Theme }, "light"},
		{"text size with its own label", `text="Large"`, 0, func(current settings.Settings) any { return current.TextSize }, "large"},
		{"corners", `text="Round"`, 0, func(current settings.Settings) any { return current.Corners }, "round"},
		{"a value without a label is capitalized", `text="Pretty"`, 0, func(current settings.Settings) any { return current.ResponseView }, "pretty"},
		{"layout", `text="Side by side"`, 0, func(current settings.Settings) any { return current.Layout }, "side"},
		{"boolean on", `text="On"`, 0, func(current settings.Settings) any { return current.Flat }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			driver.clickOn(tc.index, tc.label)
			if got := tc.got(driver.host.current); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestClickingOffTurnsABooleanSettingOff(t *testing.T) {
	driver := newFormDriver(t, "General")
	driver.clickOn(0, `text="Off"`)
	if driver.host.current.AutoSave {
		t.Fatal("autoSave is still on")
	}
}

// The selected chip is the primary button; clicking it again keeps the value.
func TestTheChipOfTheCurrentValueIsLit(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	driver.host.current.Corners = "round"
	page := driver.view()
	if !strings.Contains(page, `text="Round" grow=1 fg=accent-text`) {
		t.Fatalf("the Round chip is not lit:\n%s", page)
	}
	driver.clickOn(0, `text="Round"`)
	if driver.host.current.Corners != "round" {
		t.Fatalf("corners %q after clicking the lit chip", driver.host.current.Corners)
	}
}

func TestResetPutsTheDefaultBack(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	driver.host.current.Corners = "round"
	driver.clickOn(0, `text="Reset"`)
	if driver.host.current.Corners != settings.Default().Corners {
		t.Fatalf("corners %q after Reset", driver.host.current.Corners)
	}
	if strings.Contains(driver.view(), `text="Reset"`) {
		t.Fatal("Reset is still shown with every setting at its default")
	}
}

// The scale slider runs from 50 to 200 with 100 in the middle of its track;
// it applies live while dragged, shows the value it is at and saves once
// it is let go.
func TestTheScaleSliderAppliesLiveAndSavesOnRelease(t *testing.T) {
	cases := []struct {
		name     string
		fraction float64
		want     int
	}{
		{"the left end is the smallest", 0, 50},
		{"the middle is the default", 0.5, 100},
		{"the right end is the largest", 0.99, 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			driver.host.current.UIScalePct = 150
			driver.pressAt(tc.fraction, "sliderB")
			if driver.host.current.UIScalePct != tc.want || driver.host.saved != 0 {
				t.Fatalf("scale %d saved %d while dragging, want %d and not saved", driver.host.current.UIScalePct, driver.host.saved, tc.want)
			}
			driver.release()
			if driver.host.saved != 1 {
				t.Fatalf("saved %d times after the release, want once", driver.host.saved)
			}
		})
	}
}

// The percent next to the slider follows the thumb while it is dragged.
func TestTheScaleLabelShowsTheDraggedValue(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	driver.host.current.UIScalePct = 150
	driver.pressAt(0, "sliderB")
	driver.host.current.UIScalePct = 150 // the label must show the drag, not the setting
	if page := driver.view(); !strings.Contains(page, `text="50%"`) {
		t.Fatalf("the label does not show the dragged value:\n%s", page)
	}
}

func TestTypingANumberAppliesItOrExplainsWhyNot(t *testing.T) {
	cases := []struct {
		name    string
		typed   string
		timeout int
		note    string
	}{
		{"in range", "20", 20, ""},
		{"out of range", "0", 60, "Enter a whole number between 1 and 3600"},
		{"not a number", "soon", 60, "Enter a whole number between 1 and 3600"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFormDriver(t, "Requests")
			driver.clickOn(0, "fieldB") // the timeout comes first
			driver.retype(tc.typed)
			if driver.host.current.TimeoutSec != tc.timeout || driver.dialog.Note("timeoutSec") != tc.note {
				t.Fatalf("timeout %d note %q", driver.host.current.TimeoutSec, driver.dialog.Note("timeoutSec"))
			}
			if tc.note != "" && !strings.Contains(driver.view(), `text="`+tc.note+`"`) {
				t.Fatalf("the note is not under the field:\n%s", driver.view())
			}
		})
	}
}

// The text field of a setting starts with its value and edits it as text.
func TestATextFieldStartsWithTheSettingAndAppliesWhatIsTyped(t *testing.T) {
	driver := newFormDriver(t, "Requests")
	driver.host.current.Proxy = "http://proxy:3128"
	driver.step(func(*El) core.Input { return core.Input{} })
	if got := driver.dialog.Input(driver.host, "proxy").Text; got != "http://proxy:3128" {
		t.Fatalf("proxy field %q", got)
	}
	driver.clickOn(1, "fieldB")
	driver.retype("http://other:8080")
	if driver.host.current.Proxy != "http://other:8080" {
		t.Fatalf("proxy %q", driver.host.current.Proxy)
	}
}

func TestRightClickingANumberFieldOpensItsTextMenu(t *testing.T) {
	driver := newFormDriver(t, "Requests")
	driver.step(func(root *El) core.Input {
		rect := driver.find(root, 0, "fieldB").Rect()
		return core.Input{X: rect.X + 2, Y: rect.Y + 2, RightClick: true}
	})
	if !MenuOpen(driver.ctx) {
		t.Fatal("a right click on the field did not open its text menu")
	}
}

// A search hit on a setting edited on its own page leads to that page.
func TestOpenGoesToThePageOfATableSetting(t *testing.T) {
	cases := []struct{ label, page string }{
		{`text="Open colors"`, "colors"},
		{`text="Open shortcuts"`, "shortcuts"},
	}
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			driver.dialog.Search = NewTextField("overrides")
			driver.dialog.scroll.Y = 300
			driver.clickOn(0, tc.label)
			if driver.dialog.Cat != tc.page || driver.dialog.Searching() || driver.dialog.scroll.Y != 0 {
				t.Fatalf("page %q search %q scroll %d", driver.dialog.Cat, driver.dialog.Search.Text, driver.dialog.scroll.Y)
			}
		})
	}
}

func TestChooseFileAsksForTheFileOfItsSetting(t *testing.T) {
	cases := []struct {
		name  string
		click func(driver *formDriver)
		title string
	}{
		{"empty", func(driver *formDriver) { driver.clickOn(1, `text="Choose file"`) }, "Choose Client certificate"},
		{"chosen", func(driver *formDriver) {
			driver.host.current.CACert = "/etc/ssl/ca.pem"
			driver.clickOn(0, "control=icon-button")
		}, "Choose CA certificates"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFormDriver(t, "Requests")
			tc.click(driver)
			if driver.host.pickTitle != tc.title {
				t.Fatalf("picker title %q, want %q", driver.host.pickTitle, tc.title)
			}
		})
	}
}

func TestClickingAFontControlOpensTheListForItsRole(t *testing.T) {
	cases := []struct {
		name  string
		index int
		role  FontRole
	}{
		{"interface", 0, FontUI},
		{"code", 1, FontCode},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFormDriver(t, "Appearance")
			driver.clickOn(tc.index, "control=field", "onClick=true")
			if !driver.dialog.FontListOpen() || driver.dialog.list.role != tc.role {
				t.Fatalf("open %v role %v, want the list open for %v", driver.dialog.FontListOpen(), driver.dialog.list.role, tc.role)
			}
		})
	}
}

// The font control draws an installed family in its own font and reads
// the file once; Built-in and a missing family use the interface font.
func TestTheFontControlReadsAnInstalledFamilyOnce(t *testing.T) {
	driver := newFormDriver(t, "Appearance")
	driver.host.current.UIFont = "Go"
	driver.host.current.CodeFont = "Missing Sans"
	driver.view()
	driver.view()
	if driver.dialog.loadedFaces() != 1 {
		t.Fatalf("%d font files read, want the interface font's only", driver.dialog.loadedFaces())
	}
}
