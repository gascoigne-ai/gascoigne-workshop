package formpage

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	core "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Window sizes the open list is pinned at: the fake host's own size, and a
// small window.
const (
	listWideW, listWideH     = 1280, 800
	listNarrowW, listNarrowH = 640, 400
)

// manyFamilies are 300 families whose files cannot be read, so every row
// shows its name in the interface font.
func manyFamilies() []settings.Family {
	var families []settings.Family
	for i := range 300 {
		families = append(families, settings.Family{Name: fmt.Sprintf("Family %03d", i), Regular: fmt.Sprintf("none#%d", i)})
	}
	return families
}

// The laid-out open list is pinned as text (testdata/fontlist-*.dump,
// captured from the Go-built view), so a change in how it is built cannot
// move a pixel unnoticed.
func TestTheFontListLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		role          form.FontRole
		current       string // the family in use for role
		search        string
		families      func(t *testing.T) []settings.Family
	}{
		{"builtin-selected-wide", listWideW, listWideH, form.FontCode, "", "", testFamilies},
		{"family-selected-narrow", listNarrowW, listNarrowH, form.FontUI, "Go Mono", "", testFamilies},
		{"searched-wide", listWideW, listWideH, form.FontUI, "Go", "mono", testFamilies},
		{"no-match-wide", listWideW, listWideH, form.FontUI, "", "nothing like it", testFamilies},
		{"no-families-narrow", listNarrowW, listNarrowH, form.FontUI, "", "", func(*testing.T) []settings.Family { return nil }},
		{"many-scrolled-wide", listWideW, listWideH, form.FontUI, "Family 150", "", func(*testing.T) []settings.Family { return manyFamilies() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var d Page
			host := newHost(t)
			d.Families = tc.families(t)
			if tc.role == form.FontUI {
				host.current.UIFont = tc.current
			} else {
				host.current.CodeFont = tc.current
			}
			d.OpenFontList(testCtx(t), host, tc.role)
			d.FontSearch = NewTextField(tc.search)
			root := inWindow(d.FontList(testCtx(t), host))
			ctx, err := core.NewCtx(testTheme())
			if err != nil {
				t.Fatal(err)
			}
			Layout(ctx, root, tc.width, tc.height)
			want, err := os.ReadFile(filepath.Join("testdata", "fontlist-"+tc.name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got := Dump(root); got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// inWindow puts the list in a window-sized parent, as the dialog does, so
// it floats at its own place instead of filling the window.
func inWindow(list Node) *El { return Stack(list) }

// listDriver runs the open list frame by frame as the application does:
// rebuild the view, lay it out, deliver the input.
type listDriver struct {
	t    *testing.T
	ctx  *core.Ctx
	page Page
	host *fakeHost
}

func newListDriver(t *testing.T, families []settings.Family) *listDriver {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	driver := &listDriver{t: t, ctx: ctx, host: newHost(t)}
	driver.page.Families = families
	return driver
}

// step runs one frame; input is built from the frame's laid-out view.
func (driver *listDriver) step(input func(root *El) core.Input) {
	list := driver.page.FontList(driver.ctx, driver.host)
	if list == nil {
		driver.t.Fatal("the list is closed")
	}
	root := inWindow(list)
	Layout(driver.ctx, root, listWideW, listWideH)
	driver.ctx.In = input(root)
	Update(driver.ctx, root)
}

// listRows are the list's rows on screen, in order.
func listRows(root *El) []*El { return findAll(root, "control=row") }

// clickRow clicks the middle of the index-th row on screen.
func (driver *listDriver) clickRow(index int) {
	driver.t.Helper()
	driver.step(func(root *El) core.Input {
		found := listRows(root)
		if len(found) <= index {
			driver.t.Fatalf("%d rows, want index %d:\n%s", len(found), index, Dump(root))
		}
		rect := found[index].Rect()
		return core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	})
}

func TestClickingAFamilyRowAppliesItAndClosesTheList(t *testing.T) {
	driver := newListDriver(t, testFamilies(t))
	driver.page.OpenFontList(driver.ctx, driver.host, form.FontUI)
	driver.clickRow(2) // Built-in, Go, Go Mono
	if driver.host.current.UIFont != "Go Mono" || driver.page.FontListOpen() {
		t.Fatalf("UIFont %q, open %v; want Go Mono and the list closed", driver.host.current.UIFont, driver.page.FontListOpen())
	}
}

func TestClickingTheBuiltInRowGoesBackToTheEmbeddedFont(t *testing.T) {
	driver := newListDriver(t, testFamilies(t))
	driver.host.current.CodeFont = "Go"
	driver.page.OpenFontList(driver.ctx, driver.host, form.FontCode)
	driver.clickRow(0)
	if driver.host.current.CodeFont != "" || driver.page.FontListOpen() {
		t.Fatalf("CodeFont %q, open %v; want Built-in and the list closed", driver.host.current.CodeFont, driver.page.FontListOpen())
	}
}

// Typing goes into the search field, focused on open, narrows the rows and
// starts the list again from the top.
func TestTypingASearchNarrowsTheRowsAndScrollsToTheTop(t *testing.T) {
	driver := newListDriver(t, manyFamilies())
	driver.host.current.UIFont = "Family 150"
	driver.page.OpenFontList(driver.ctx, driver.host, form.FontUI)
	if driver.page.list.scroll.Y == 0 {
		t.Fatal("the list did not open scrolled to the family in use")
	}
	driver.step(func(*El) core.Input { return core.Input{Typed: []rune("Family 29")} })
	if got := driver.page.FontSearch.Text; got != "Family 29" {
		t.Fatalf("search text %q, want Family 29", got)
	}
	if driver.page.list.scroll.Y != 0 {
		t.Fatalf("scroll = %d after typing, want the top", driver.page.list.scroll.Y)
	}
	first := driver.page.matchingFamilies()[0].Name
	driver.clickRow(0)
	if driver.host.current.UIFont != first {
		t.Fatalf("UIFont %q, want the first match, %s", driver.host.current.UIFont, first)
	}
}

func TestAClickOutsideTheListDismissesIt(t *testing.T) {
	driver := newListDriver(t, testFamilies(t))
	driver.page.OpenFontList(driver.ctx, driver.host, form.FontUI)
	driver.step(func(*El) core.Input { return core.Input{X: 5, Y: 5, Click: true} })
	if driver.page.FontListOpen() || driver.host.current.UIFont != "" {
		t.Fatalf("open %v, UIFont %q; want the list closed and nothing picked", driver.page.FontListOpen(), driver.host.current.UIFont)
	}
}

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
